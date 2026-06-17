package grpc

import (
	"context"
	"errors"
	"math/big"
	"sync/atomic"
	"testing"
	"time"

	"cosmossdk.io/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	proto "github.com/0xPolygon/polyproto/bor"
	protoutil "github.com/0xPolygon/polyproto/utils"

	"github.com/0xPolygon/heimdall-v2/x/bor/failover"
)

// newTestMulti builds a MultiBorGRPCClient over mock API clients without
// starting the prober, so tests drive failover deterministically.
func newTestMulti(clients ...*MockBorApiClient) *MultiBorGRPCClient {
	bcs := make([]*BorGRPCClient, len(clients))
	for i, c := range clients {
		bcs[i] = &BorGRPCClient{client: c}
	}
	mc := &MultiBorGRPCClient{clients: bcs, attemptTimeout: time.Second}
	mc.health = failover.New(len(bcs), mc.probe, failover.Metrics{}, log.NewNopLogger())
	mc.health.SetTuning(5*time.Millisecond, 1, 0, 50*time.Millisecond)
	return mc
}

func genesisResp(extra string) *proto.GetHeaderByNumberResponse {
	return &proto.GetHeaderByNumberResponse{Header: &proto.Header{Number: 0, ExtraData: []byte(extra)}}
}

func TestMultiGRPC_CascadesOnUnavailable(t *testing.T) {
	m0 := new(MockBorApiClient)
	m0.On("GetRootHash", mock.Anything, mock.Anything).
		Return((*proto.GetRootHashResponse)(nil), status.Error(codes.Unavailable, "down"))
	m1 := new(MockBorApiClient)
	m1.On("GetRootHash", mock.Anything, mock.Anything).
		Return(&proto.GetRootHashResponse{RootHash: "0xabc"}, nil)

	mc := newTestMulti(m0, m1)
	mc.health.MarkSuccess(1) // secondary is a validated candidate

	got, err := mc.GetRootHash(context.Background(), 1, 2)
	require.NoError(t, err)
	require.Equal(t, "0xabc", got)
	require.Equal(t, 1, mc.health.Active())
}

func TestNewMultiBorGRPCClient_RejectsZeroClients(t *testing.T) {
	require.PanicsWithValue(t, "bor failover: endpoint count must be positive", func() {
		NewMultiBorGRPCClient(nil, log.NewNopLogger(), failover.Metrics{}, time.Second)
	})
}

func TestMultiGRPC_NoCascadeOnLogicalError(t *testing.T) {
	m0 := new(MockBorApiClient)
	m0.On("GetRootHash", mock.Anything, mock.Anything).
		Return((*proto.GetRootHashResponse)(nil), status.Error(codes.InvalidArgument, "bad range"))
	m1 := new(MockBorApiClient)

	mc := newTestMulti(m0, m1)
	mc.health.MarkSuccess(1)

	_, err := mc.GetRootHash(context.Background(), 1, 2)
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	require.Equal(t, 0, mc.health.Active())
	m1.AssertNotCalled(t, "GetRootHash", mock.Anything, mock.Anything)
}

func TestMultiGRPC_NoCascadeOnCallerCancel(t *testing.T) {
	m0 := new(MockBorApiClient)
	m0.On("GetRootHash", mock.Anything, mock.Anything).
		Return((*proto.GetRootHashResponse)(nil), status.Error(codes.Unavailable, "down"))
	m1 := new(MockBorApiClient)

	mc := newTestMulti(m0, m1)
	mc.health.MarkSuccess(1) // a candidate exists, yet cancellation must still stop the cascade

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := mc.GetRootHash(ctx, 1, 2)
	require.Error(t, err)
	require.Equal(t, 0, mc.health.Active())
	m1.AssertNotCalled(t, "GetRootHash", mock.Anything, mock.Anything)
}

func TestMultiGRPC_PerAttemptTimeoutCascades(t *testing.T) {
	m0 := new(MockBorApiClient)
	m0.On("GetRootHash", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			<-args.Get(0).(context.Context).Done() // primary hangs until the attempt times out
		}).
		Return((*proto.GetRootHashResponse)(nil), status.Error(codes.DeadlineExceeded, "timeout"))
	m1 := new(MockBorApiClient)
	m1.On("GetRootHash", mock.Anything, mock.Anything).
		Return(&proto.GetRootHashResponse{RootHash: "0xabc"}, nil)

	mc := newTestMulti(m0, m1)
	mc.attemptTimeout = 30 * time.Millisecond
	mc.health.MarkSuccess(1)

	got, err := mc.GetRootHash(context.Background(), 1, 2)
	require.NoError(t, err)
	require.Equal(t, "0xabc", got)
	require.Equal(t, 1, mc.health.Active())
}

func TestMultiGRPC_BatchCascades(t *testing.T) {
	m0 := new(MockBorApiClient)
	m0.On("GetBlockInfoInBatch", mock.Anything, mock.Anything).
		Return((*proto.GetBlockInfoInBatchResponse)(nil), status.Error(codes.Unavailable, "down"))
	m1 := new(MockBorApiClient)
	m1.On("GetBlockInfoInBatch", mock.Anything, mock.Anything).
		Return(&proto.GetBlockInfoInBatchResponse{Blocks: nil}, nil)

	mc := newTestMulti(m0, m1)
	mc.health.MarkSuccess(1)

	headers, tds, authors, err := mc.GetBlockInfoInBatch(context.Background(), 0, 0)
	require.NoError(t, err)
	require.Empty(t, headers)
	require.Empty(t, tds)
	require.Empty(t, authors)
	require.Equal(t, 1, mc.health.Active())
}

func TestMultiGRPC_WrapperMethodsCascade(t *testing.T) {
	t.Run("GetVoteOnHash", func(t *testing.T) {
		m0 := new(MockBorApiClient)
		m0.On("GetVoteOnHash", mock.Anything, mock.Anything).
			Return((*proto.GetVoteOnHashResponse)(nil), status.Error(codes.Unavailable, "down"))
		m1 := new(MockBorApiClient)
		m1.On("GetVoteOnHash", mock.Anything, mock.Anything).
			Return(&proto.GetVoteOnHashResponse{Response: true}, nil)

		mc := newTestMulti(m0, m1)
		mc.health.MarkSuccess(1)

		got, err := mc.GetVoteOnHash(context.Background(), 1, 2, "0xabc", "mid")
		require.NoError(t, err)
		require.True(t, got)
		require.Equal(t, 1, mc.health.Active())
	})

	t.Run("HeaderByNumber", func(t *testing.T) {
		m0 := new(MockBorApiClient)
		m0.On("HeaderByNumber", mock.Anything, mock.Anything).
			Return((*proto.GetHeaderByNumberResponse)(nil), status.Error(codes.Unavailable, "down"))
		m1 := new(MockBorApiClient)
		m1.On("HeaderByNumber", mock.Anything, mock.Anything).
			Return(&proto.GetHeaderByNumberResponse{Header: &proto.Header{Number: 11}}, nil)

		mc := newTestMulti(m0, m1)
		mc.health.MarkSuccess(1)

		got, err := mc.HeaderByNumber(context.Background(), 11)
		require.NoError(t, err)
		require.Equal(t, int64(11), got.Number.Int64())
		require.Equal(t, 1, mc.health.Active())
	})

	t.Run("BlockByNumber", func(t *testing.T) {
		m0 := new(MockBorApiClient)
		m0.On("BlockByNumber", mock.Anything, mock.Anything).
			Return((*proto.GetBlockByNumberResponse)(nil), status.Error(codes.Unavailable, "down"))
		m1 := new(MockBorApiClient)
		m1.On("BlockByNumber", mock.Anything, mock.Anything).
			Return(&proto.GetBlockByNumberResponse{Block: &proto.Block{Header: &proto.Header{Number: 12}}}, nil)

		mc := newTestMulti(m0, m1)
		mc.health.MarkSuccess(1)

		got, err := mc.BlockByNumber(context.Background(), 12)
		require.NoError(t, err)
		require.Equal(t, uint64(12), got.NumberU64())
		require.Equal(t, 1, mc.health.Active())
	})

	t.Run("TransactionReceipt", func(t *testing.T) {
		txHash := common.HexToHash("0x1234")
		m0 := new(MockBorApiClient)
		m0.On("TransactionReceipt", mock.Anything, mock.Anything).
			Return((*proto.ReceiptResponse)(nil), status.Error(codes.Unavailable, "down"))
		m1 := new(MockBorApiClient)
		m1.On("TransactionReceipt", mock.Anything, mock.Anything).
			Return(&proto.ReceiptResponse{Receipt: &proto.Receipt{TxHash: protoutil.ConvertHashToH256(txHash)}}, nil)

		mc := newTestMulti(m0, m1)
		mc.health.MarkSuccess(1)

		got, err := mc.TransactionReceipt(context.Background(), txHash)
		require.NoError(t, err)
		require.Equal(t, txHash, got.TxHash)
		require.Equal(t, 1, mc.health.Active())
	})

	t.Run("BorBlockReceipt", func(t *testing.T) {
		txHash := common.HexToHash("0x5678")
		m0 := new(MockBorApiClient)
		m0.On("BorBlockReceipt", mock.Anything, mock.Anything).
			Return((*proto.ReceiptResponse)(nil), status.Error(codes.Unavailable, "down"))
		m1 := new(MockBorApiClient)
		m1.On("BorBlockReceipt", mock.Anything, mock.Anything).
			Return(&proto.ReceiptResponse{Receipt: &proto.Receipt{TxHash: protoutil.ConvertHashToH256(txHash)}}, nil)

		mc := newTestMulti(m0, m1)
		mc.health.MarkSuccess(1)

		got, err := mc.BorBlockReceipt(context.Background(), txHash)
		require.NoError(t, err)
		require.Equal(t, txHash, got.TxHash)
		require.Equal(t, 1, mc.health.Active())
	})

	t.Run("GetAuthor", func(t *testing.T) {
		author := common.HexToAddress("0x1111111111111111111111111111111111111111")
		m0 := new(MockBorApiClient)
		m0.On("GetAuthor", mock.Anything, mock.Anything).
			Return((*proto.GetAuthorResponse)(nil), status.Error(codes.Unavailable, "down"))
		m1 := new(MockBorApiClient)
		m1.On("GetAuthor", mock.Anything, mock.Anything).
			Return(&proto.GetAuthorResponse{Author: protoutil.ConvertAddressToH160(author)}, nil)

		mc := newTestMulti(m0, m1)
		mc.health.MarkSuccess(1)

		got, err := mc.GetAuthor(context.Background(), big.NewInt(13))
		require.NoError(t, err)
		require.Equal(t, author, *got)
		require.Equal(t, 1, mc.health.Active())
	})

	t.Run("GetTdByHash", func(t *testing.T) {
		hash := common.HexToHash("0xabcd")
		m0 := new(MockBorApiClient)
		m0.On("GetTdByHash", mock.Anything, mock.Anything).
			Return((*proto.GetTdResponse)(nil), status.Error(codes.Unavailable, "down"))
		m1 := new(MockBorApiClient)
		m1.On("GetTdByHash", mock.Anything, mock.Anything).
			Return(&proto.GetTdResponse{TotalDifficulty: 33}, nil)

		mc := newTestMulti(m0, m1)
		mc.health.MarkSuccess(1)

		got, err := mc.GetTdByHash(context.Background(), hash)
		require.NoError(t, err)
		require.Equal(t, uint64(33), got)
		require.Equal(t, 1, mc.health.Active())
	})

	t.Run("GetTdByNumber", func(t *testing.T) {
		m0 := new(MockBorApiClient)
		m0.On("GetTdByNumber", mock.Anything, mock.Anything).
			Return((*proto.GetTdResponse)(nil), status.Error(codes.Unavailable, "down"))
		m1 := new(MockBorApiClient)
		m1.On("GetTdByNumber", mock.Anything, mock.Anything).
			Return(&proto.GetTdResponse{TotalDifficulty: 44}, nil)

		mc := newTestMulti(m0, m1)
		mc.health.MarkSuccess(1)

		got, err := mc.GetTdByNumber(context.Background(), big.NewInt(14))
		require.NoError(t, err)
		require.Equal(t, uint64(44), got)
		require.Equal(t, 1, mc.health.Active())
	})
}

func TestMultiGRPC_ProbeValidatesGenesis(t *testing.T) {
	primary := new(MockBorApiClient)
	primary.On("HeaderByNumber", mock.Anything, mock.Anything).Return(genesisResp("genesis-a"), nil)
	match := new(MockBorApiClient)
	match.On("HeaderByNumber", mock.Anything, mock.Anything).Return(genesisResp("genesis-a"), nil)
	wrong := new(MockBorApiClient)
	wrong.On("HeaderByNumber", mock.Anything, mock.Anything).Return(genesisResp("genesis-b"), nil)

	mc := newTestMulti(primary, match, wrong)
	require.NoError(t, mc.probe(0)) // primary establishes the expected genesis
	require.NoError(t, mc.probe(1)) // same genesis → ok
	require.Error(t, mc.probe(2))   // different genesis → rejected
}

func TestMultiGRPC_ProbeRunsEndpointValidators(t *testing.T) {
	primary := new(MockBorApiClient)
	primary.On("HeaderByNumber", mock.Anything, mock.Anything).Return(genesisResp("genesis-a"), nil)
	fallback := new(MockBorApiClient)
	fallback.On("HeaderByNumber", mock.Anything, mock.Anything).Return(genesisResp("genesis-a"), nil)

	errParity := errors.New("parity mismatch")
	mc := newTestMulti(primary, fallback)
	mc.validators = []EndpointValidator{
		func(_ context.Context, i int, _ EndpointHeaderFetcher) error {
			if i == 1 {
				return errParity
			}
			return nil
		},
	}

	require.NoError(t, mc.probe(0))
	require.ErrorIs(t, mc.probe(1), errParity)
	require.Empty(t, mc.health.Candidates(0))
}

func TestCheckGenesis(t *testing.T) {
	mc := &MultiBorGRPCClient{}
	a := common.HexToHash("0x0a")
	b := common.HexToHash("0x0b")

	require.Error(t, mc.checkGenesis(1, a))   // expected unknown + fallback → rejected
	require.NoError(t, mc.checkGenesis(0, a)) // primary establishes the expectation
	require.NoError(t, mc.checkGenesis(1, a)) // matches
	require.Error(t, mc.checkGenesis(2, b))   // mismatch
}

func TestCheckGenesis_FallbackAnchorsWhenPrimaryUnreachable(t *testing.T) {
	mc := &MultiBorGRPCClient{}
	mc.primaryProbeFailures.Store(primaryAnchorFailureThreshold)
	a := common.HexToHash("0x0a")
	b := common.HexToHash("0x0b")

	require.NoError(t, mc.checkGenesis(1, a)) // fallback provisionally establishes the expectation
	require.NoError(t, mc.checkGenesis(2, a)) // another fallback on the same chain matches
	require.Error(t, mc.checkGenesis(2, b))   // a mismatched endpoint is still rejected
}

func TestMultiGRPC_PrimaryReclaimAdoptsActive(t *testing.T) {
	m0 := new(MockBorApiClient)
	m0.On("HeaderByNumber", mock.Anything, mock.Anything).Return(genesisResp("primary"), nil)
	m1 := new(MockBorApiClient)
	m1.On("HeaderByNumber", mock.Anything, mock.Anything).Return(genesisResp("fallback"), nil)

	mc := newTestMulti(m0, m1)
	// Boot window: the primary was unreachable, so the fallback provisionally
	// anchored its own genesis and became the active endpoint.
	mc.primaryProbeFailures.Store(primaryAnchorFailureThreshold)
	require.NoError(t, mc.probe(1))
	mc.health.SetActive(1)
	require.Equal(t, 1, mc.health.Active())

	// The primary recovers: its probe reclaims identity and moves traffic back at once.
	require.NoError(t, mc.probe(0))
	require.Equal(t, 0, mc.health.Active())
}

func TestCheckGenesis_PrimaryReclaimsProvisionalAnchor(t *testing.T) {
	mc := &MultiBorGRPCClient{}
	mc.primaryProbeFailures.Store(primaryAnchorFailureThreshold)
	a := common.HexToHash("0x0a")
	b := common.HexToHash("0x0b")

	require.NoError(t, mc.checkGenesis(1, a)) // fallback provisionally anchors a
	require.NoError(t, mc.checkGenesis(0, b)) // primary reclaims with its own genesis, never rejected
	require.Equal(t, b, *mc.expectedGenesis.Load())
	require.Error(t, mc.checkGenesis(1, a)) // the stale provisional fallback now mismatches the primary
}

func TestCaptureExpectedGenesis(t *testing.T) {
	up := new(MockBorApiClient)
	up.On("HeaderByNumber", mock.Anything, mock.Anything).Return(genesisResp("g"), nil)
	mc := &MultiBorGRPCClient{clients: []*BorGRPCClient{{client: up}}, attemptTimeout: time.Second}
	mc.captureExpectedGenesis()
	require.NotNil(t, mc.expectedGenesis.Load())

	down := new(MockBorApiClient)
	down.On("HeaderByNumber", mock.Anything, mock.Anything).
		Return((*proto.GetHeaderByNumberResponse)(nil), status.Error(codes.Unavailable, "down"))
	mc2 := &MultiBorGRPCClient{clients: []*BorGRPCClient{{client: down}}, attemptTimeout: time.Second}
	mc2.captureExpectedGenesis()
	require.Nil(t, mc2.expectedGenesis.Load()) // unreachable primary → nothing captured
}

func TestMultiGRPC_ProbeRejectsUnreachable(t *testing.T) {
	down := new(MockBorApiClient)
	down.On("HeaderByNumber", mock.Anything, mock.Anything).
		Return((*proto.GetHeaderByNumberResponse)(nil), status.Error(codes.Unavailable, "down"))

	mc := newTestMulti(down)
	require.Error(t, mc.probe(0))
}

func TestNewMultiBorGRPCClient_StartsProberAndCloses(t *testing.T) {
	var probed atomic.Int64
	m0 := new(MockBorApiClient)
	m0.On("HeaderByNumber", mock.Anything, mock.Anything).
		Run(func(mock.Arguments) { probed.Add(1) }).Return(genesisResp("g"), nil)
	m1 := new(MockBorApiClient)
	m1.On("HeaderByNumber", mock.Anything, mock.Anything).
		Run(func(mock.Arguments) { probed.Add(1) }).Return(genesisResp("g"), nil)

	mc := NewMultiBorGRPCClient([]*BorGRPCClient{{client: m0}, {client: m1}}, log.NewNopLogger(), failover.Metrics{}, time.Second)
	require.NotNil(t, mc)

	require.Eventually(t, func() bool { return probed.Load() > 0 }, time.Second, 5*time.Millisecond)
	require.NotPanics(t, func() { mc.Close(log.NewNopLogger()) })
}
