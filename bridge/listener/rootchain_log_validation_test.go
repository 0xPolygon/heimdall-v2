package listener

import (
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"cosmossdk.io/log"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/metrics"
	chainmanagerTypes "github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
)

func TestRootChainListener_ValidateLogAgainstQuery(t *testing.T) {
	t.Parallel()

	// Must be a name in rootChainEvents (a real queried event), not an
	// arbitrary string — validateLogAgainstQuery checks membership there.
	// Belongs to the RootChain contract.
	rootChainTopic := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	rootChainEvent := &abi.Event{Name: helper.NewHeaderBlockEvent}

	// A second real, queried event, belonging to a different contract
	// (StakingInfo) — used to test cross-contract topic/address mismatches.
	stakingInfoTopic := common.HexToHash("0x2222222222222222222222222222222222222222222222222222222222222222")
	stakingInfoEvent := &abi.Event{Name: helper.StakedEvent}

	// In eventMap (so the topic resolves) but its event name isn't in
	// rootChainEvents — the gap the query-scope check exists to close.
	unqueriedTopic := common.HexToHash("0x4444444444444444444444444444444444444444444444444444444444444444")
	unqueriedEvent := &abi.Event{Name: "OwnershipTransferred"}

	// In eventMap and rootChainEvents (so it clears both earlier checks) but
	// deliberately absent from eventContract — the gap the explicit ok-check
	// on that lookup exists to close. Production listeners always populate
	// both maps together; this simulates them falling out of sync.
	unboundContractTopic := common.HexToHash("0x5555555555555555555555555555555555555555555555555555555555555555")
	unboundContractEvent := &abi.Event{Name: helper.StakeUpdateEvent}

	rootChainAddress := common.HexToAddress("0xaaaa000000000000000000000000000000aaaa")
	stakingInfoAddress := common.HexToAddress("0xbbbb000000000000000000000000000000bbbb")
	stateSenderAddress := common.HexToAddress("0xcccc000000000000000000000000000000cccc")

	contractAddresses := map[rootChainContract]common.Address{
		rootChainContractRootChain:   rootChainAddress,
		rootChainContractStakingInfo: stakingInfoAddress,
		rootChainContractStateSender: stateSenderAddress,
	}

	fromBlock := big.NewInt(100)
	toBlock := big.NewInt(200)

	newListener := func() *RootChainListener {
		rl := &RootChainListener{
			eventMap: map[common.Hash]*abi.Event{
				rootChainTopic:       rootChainEvent,
				stakingInfoTopic:     stakingInfoEvent,
				unqueriedTopic:       unqueriedEvent,
				unboundContractTopic: unboundContractEvent,
			},
			eventContract: map[common.Hash]rootChainContract{
				rootChainTopic:   rootChainContractRootChain,
				stakingInfoTopic: rootChainContractStakingInfo,
				unqueriedTopic:   rootChainContractRootChain,
				// unboundContractTopic intentionally has no eventContract entry.
			},
		}
		rl.BaseListener.Logger = log.NewNopLogger()
		return rl
	}

	t.Run("accepts a log matching its event's contract, range, and topic", func(t *testing.T) {
		t.Parallel()

		rl := newListener()
		vLog := types.Log{
			Address:     rootChainAddress,
			Topics:      []common.Hash{rootChainTopic},
			BlockNumber: 150,
		}

		event, ok := rl.validateLogAgainstQuery(vLog, contractAddresses, fromBlock, toBlock, map[string]struct{}{})
		require.True(t, ok)
		require.Same(t, rootChainEvent, event)
	})

	t.Run("accepts a log from a different watched contract when its topic matches", func(t *testing.T) {
		t.Parallel()

		rl := newListener()
		vLog := types.Log{
			Address:     stakingInfoAddress,
			Topics:      []common.Hash{stakingInfoTopic},
			BlockNumber: 150,
		}

		event, ok := rl.validateLogAgainstQuery(vLog, contractAddresses, fromBlock, toBlock, map[string]struct{}{})
		require.True(t, ok)
		require.Same(t, stakingInfoEvent, event)
	})

	t.Run("accepts a log at the exact range boundaries", func(t *testing.T) {
		t.Parallel()

		rl := newListener()

		lowerBound := types.Log{Address: rootChainAddress, Topics: []common.Hash{rootChainTopic}, BlockNumber: fromBlock.Uint64()}
		_, ok := rl.validateLogAgainstQuery(lowerBound, contractAddresses, fromBlock, toBlock, map[string]struct{}{})
		require.True(t, ok)

		upperBound := types.Log{Address: rootChainAddress, Topics: []common.Hash{rootChainTopic}, BlockNumber: toBlock.Uint64()}
		_, ok = rl.validateLogAgainstQuery(upperBound, contractAddresses, fromBlock, toBlock, map[string]struct{}{})
		require.True(t, ok)
	})

	t.Run("rejects a log from an address outside all watched contracts", func(t *testing.T) {
		t.Parallel()

		rl := newListener()
		vLog := types.Log{
			Address:     common.HexToAddress("0xdddd000000000000000000000000000000dddd"),
			Topics:      []common.Hash{rootChainTopic},
			BlockNumber: 150,
		}

		_, ok := rl.validateLogAgainstQuery(vLog, contractAddresses, fromBlock, toBlock, map[string]struct{}{})
		require.False(t, ok)
	})

	t.Run("rejects a log whose topic belongs to a different contract than its address", func(t *testing.T) {
		t.Parallel()

		rl := newListener()
		// A real, watched contract address (StakingInfo) paired with a topic
		// that belongs to RootChain — right address, wrong topic-family.
		vLog := types.Log{
			Address:     stakingInfoAddress,
			Topics:      []common.Hash{rootChainTopic},
			BlockNumber: 150,
		}

		_, ok := rl.validateLogAgainstQuery(vLog, contractAddresses, fromBlock, toBlock, map[string]struct{}{})
		require.False(t, ok)
	})

	t.Run("rejects a log whose address belongs to a different contract than its topic", func(t *testing.T) {
		t.Parallel()

		rl := newListener()
		// The topic resolves to a StakingInfo event, but the log claims to
		// come from RootChainAddress — right topic, wrong address.
		vLog := types.Log{
			Address:     rootChainAddress,
			Topics:      []common.Hash{stakingInfoTopic},
			BlockNumber: 150,
		}

		_, ok := rl.validateLogAgainstQuery(vLog, contractAddresses, fromBlock, toBlock, map[string]struct{}{})
		require.False(t, ok)
	})

	t.Run("rejects a log below the queried block range", func(t *testing.T) {
		t.Parallel()

		rl := newListener()
		vLog := types.Log{
			Address:     rootChainAddress,
			Topics:      []common.Hash{rootChainTopic},
			BlockNumber: fromBlock.Uint64() - 1,
		}

		_, ok := rl.validateLogAgainstQuery(vLog, contractAddresses, fromBlock, toBlock, map[string]struct{}{})
		require.False(t, ok)
	})

	t.Run("rejects a log above the queried block range", func(t *testing.T) {
		t.Parallel()

		rl := newListener()
		vLog := types.Log{
			Address:     rootChainAddress,
			Topics:      []common.Hash{rootChainTopic},
			BlockNumber: toBlock.Uint64() + 1,
		}

		_, ok := rl.validateLogAgainstQuery(vLog, contractAddresses, fromBlock, toBlock, map[string]struct{}{})
		require.False(t, ok)
	})

	t.Run("rejects a log with no topics", func(t *testing.T) {
		t.Parallel()

		rl := newListener()
		vLog := types.Log{
			Address:     rootChainAddress,
			Topics:      []common.Hash{},
			BlockNumber: 150,
		}

		_, ok := rl.validateLogAgainstQuery(vLog, contractAddresses, fromBlock, toBlock, map[string]struct{}{})
		require.False(t, ok)
	})

	t.Run("rejects a log with a topic outside the configured event set", func(t *testing.T) {
		t.Parallel()

		rl := newListener()
		vLog := types.Log{
			Address:     rootChainAddress,
			Topics:      []common.Hash{common.HexToHash("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")},
			BlockNumber: 150,
		}

		_, ok := rl.validateLogAgainstQuery(vLog, contractAddresses, fromBlock, toBlock, map[string]struct{}{})
		require.False(t, ok)
	})

	t.Run("rejects a log whose event resolves in eventMap but isn't in the query's topic set", func(t *testing.T) {
		t.Parallel()

		rl := newListener()
		vLog := types.Log{
			Address:     rootChainAddress,
			Topics:      []common.Hash{unqueriedTopic},
			BlockNumber: 150,
		}

		_, ok := rl.validateLogAgainstQuery(vLog, contractAddresses, fromBlock, toBlock, map[string]struct{}{})
		require.False(t, ok)
	})

	t.Run("rejects a log whose topic resolves in eventMap but has no eventContract entry", func(t *testing.T) {
		t.Parallel()

		rl := newListener()
		vLog := types.Log{
			Address:     rootChainAddress,
			Topics:      []common.Hash{unboundContractTopic},
			BlockNumber: 150,
		}

		_, ok := rl.validateLogAgainstQuery(vLog, contractAddresses, fromBlock, toBlock, map[string]struct{}{})
		require.False(t, ok)
	})

	t.Run("rejects a removed (reorg'd) log", func(t *testing.T) {
		t.Parallel()

		rl := newListener()
		vLog := types.Log{
			Address:     rootChainAddress,
			Topics:      []common.Hash{rootChainTopic},
			BlockNumber: 150,
			Removed:     true,
		}

		_, ok := rl.validateLogAgainstQuery(vLog, contractAddresses, fromBlock, toBlock, map[string]struct{}{})
		require.False(t, ok)
	})
}

func TestRootChainListener_ValidateAndHandleLogs(t *testing.T) {
	t.Parallel()

	knownTopic := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	knownEvent := &abi.Event{Name: helper.StakedEvent}
	rootChainAddress := common.HexToAddress("0xaaaa000000000000000000000000000000aaaa")
	contractAddresses := map[rootChainContract]common.Address{
		rootChainContractRootChain: rootChainAddress,
	}
	fromBlock := big.NewInt(100)
	toBlock := big.NewInt(200)

	newListener := func() *RootChainListener {
		rl := &RootChainListener{
			eventMap:      map[common.Hash]*abi.Event{knownTopic: knownEvent},
			eventContract: map[common.Hash]rootChainContract{knownTopic: rootChainContractRootChain},
		}
		rl.BaseListener.Logger = log.NewNopLogger()
		return rl
	}

	t.Run("a bad log anywhere in the batch fails the whole batch and dispatches nothing", func(t *testing.T) {
		t.Parallel()

		rl := newListener()

		// A valid log ahead of an invalid one: if dispatch ran before the whole
		// batch was validated, this would call handleLog for the valid entry
		// (which needs ABI/codec plumbing this test deliberately doesn't set up,
		// so it would panic) before returning the error for the invalid one.
		logs := []types.Log{
			{Address: rootChainAddress, Topics: []common.Hash{knownTopic}, BlockNumber: 150},
			{Address: common.HexToAddress("0xcccc000000000000000000000000000000cccc"), Topics: []common.Hash{knownTopic}, BlockNumber: 150},
		}

		var err error
		require.NotPanics(t, func() {
			err = rl.validateAndHandleLogs(logs, contractAddresses, fromBlock, toBlock, map[string]struct{}{})
		})
		require.ErrorIs(t, err, errUnexpectedRootChainLog)
	})

	t.Run("an empty batch succeeds with no dispatch", func(t *testing.T) {
		t.Parallel()

		rl := newListener()

		err := rl.validateAndHandleLogs(nil, contractAddresses, fromBlock, toBlock, map[string]struct{}{})
		require.NoError(t, err)
	})
}

// mockEthGetLogs starts a JSON-RPC HTTP server that answers eth_getLogs with
// the given logs JSON for every request, regardless of the requested block
// range — simulating an endpoint that doesn't honor FilterLogs' filter, the
// exact scenario validateLogAgainstQuery exists to catch. requestCount is
// incremented on every call, so a caller can confirm bisection actually
// re-queried the endpoint more than once.
func mockEthGetLogs(t *testing.T, logsJSON string, requestCount *int) string {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var req struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
		}
		require.NoError(t, json.Unmarshal(body, &req))
		require.Equal(t, "eth_getLogs", req.Method)
		*requestCount++

		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write([]byte(`{"jsonrpc":"2.0","id":` + string(req.ID) + `,"result":` + logsJSON + `}`))
		require.NoError(t, err)
	}))
	t.Cleanup(server.Close)
	return server.URL
}

// TestProcessRootChainBlockRange_DedupesRejectionMetricAcrossBisection proves
// the fix for the metric-inflation bug: a persistently-bad log returned by
// every sub-query during range bisection must only increment
// rootchain_listener_log_rejected_total once per top-level call, not once
// per bisection level that re-encounters it.
func TestProcessRootChainBlockRange_DedupesRejectionMetricAcrossBisection(t *testing.T) {
	knownTopic := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	knownEvent := &abi.Event{Name: helper.NewHeaderBlockEvent}
	// Not in eventMap at all: every sub-range query re-encounters this same
	// log and fails the same "unrecognized topic" check, regardless of which
	// bisected fromBlock/toBlock it's queried against.
	badLogTopic := common.HexToHash("0x9999999999999999999999999999999999999999999999999999999999999999")

	badLog := &types.Log{
		Address: common.HexToAddress("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"),
		Topics:  []common.Hash{badLogTopic},
		TxHash:  common.HexToHash("0xbad"),
		Index:   0,
	}
	logsJSON, err := json.Marshal([]*types.Log{badLog})
	require.NoError(t, err)

	rl := &RootChainListener{
		eventMap:      map[common.Hash]*abi.Event{knownTopic: knownEvent},
		eventContract: map[common.Hash]rootChainContract{knownTopic: rootChainContractRootChain},
	}
	rl.BaseListener.Logger = log.NewNopLogger()

	var requestCount int
	rpcURL := mockEthGetLogs(t, string(logsJSON), &requestCount)
	client, err := ethclient.Dial(rpcURL)
	require.NoError(t, err)
	rl.contractCaller.MainChainClient = client
	rl.contractCaller.MainChainTimeout = 5 * time.Second

	rootChainContext := &RootChainListenerContext{
		ChainmanagerParams: &chainmanagerTypes.Params{
			ChainParams: chainmanagerTypes.ChainParams{
				RootChainAddress:   "0x1111111111111111111111111111111111111111",
				StakingInfoAddress: "0x2222222222222222222222222222222222222222",
				StateSenderAddress: "0x3333333333333333333333333333333333333333",
			},
		},
	}

	before := testutil.ToFloat64(metrics.RootChainListenerLogRejected)

	rejectedLogs := make(map[string]struct{})
	// A 4-block range bisects into several sub-queries (range, then halves,
	// down to single blocks) before processRootChainBlockRange gives up —
	// every one of them re-fetches and re-rejects the same bad log.
	err = rl.processRootChainBlockRange(rootChainContext, big.NewInt(100), big.NewInt(103), rejectedLogs)
	require.Error(t, err)

	require.Greater(t, requestCount, 1, "the range must actually have been bisected into more than one sub-query")

	after := testutil.ToFloat64(metrics.RootChainListenerLogRejected)
	require.Equal(t, float64(1), after-before, "one bad log across a bisected range must increment the rejection counter exactly once")
}
