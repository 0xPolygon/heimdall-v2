package processor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"cosmossdk.io/log"
	"github.com/RichardKnop/machinery/v1/tasks"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtsecp256k1 "github.com/cometbft/cometbft/crypto/secp256k1"
	cmtjson "github.com/cometbft/cometbft/libs/json"
	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	ctypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	cosmossecp256k1 "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/bridge/broadcaster"
	"github.com/0xPolygon/heimdall-v2/bridge/util"
	"github.com/0xPolygon/heimdall-v2/contracts/statesender"
	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/metrics"
	chainmanagertypes "github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
	clerktypes "github.com/0xPolygon/heimdall-v2/x/clerk/types"
)

const (
	burstAccountNumber = 7
	burstCommittedSeq  = 40
	burstBorChainID    = "80002"
)

// burstNode stubs the heimdall REST API and CometBFT RPC a validator's bridge
// talks to during a state sync burst, and records every broadcast it accepts
// or rejects.
type burstNode struct {
	t   *testing.T
	cdc codec.Codec
	txc client.TxConfig

	mu sync.Mutex
	// mempoolTotal is what num_unconfirmed_txs reports.
	mempoolTotal int
	// stored holds the state sync IDs heimdall already recorded.
	stored map[uint64]bool
	// broadcastResults answers successive broadcast_tx_sync calls; nil is CheckTx OK.
	broadcastResults []func() (*ctypes.ResultBroadcastTx, error)
	// sequences lists the signer sequence of every broadcast tx, in order.
	sequences []uint64
	// broadcastData lists the record data of every broadcast tx, in order.
	broadcastData [][]byte
	// pending holds the clerk records unconfirmed_txs reports.
	pending []*clerktypes.MsgEventRecord
	// restCalls counts heimdall REST requests.
	restCalls int
	// countUnavailable makes num_unconfirmed_txs fail.
	countUnavailable bool
	// paramsUnavailable makes the chain manager params query fail.
	paramsUnavailable bool
	// paramsCalls counts chain manager params queries.
	paramsCalls int
	// paramsBlock, when set, holds chain manager params queries until closed.
	paramsBlock chan struct{}
	// recordLookups counts stored-record queries per state sync ID.
	recordLookups map[uint64]int
	// lookupDelay slows every stored-record and record-list query.
	lookupDelay time.Duration

	// lookupStatus, when set, is the answer to every stored-record query.
	lookupStatus int
	// listPages lists the record-list pages queried, in order.
	listPages []uint64
	// listUnavailable makes record-list queries fail.
	listUnavailable bool
	// latestOnL1 is the StateSender counter the latest-id query reports.
	latestOnL1 uint64
	// latestUnavailable makes the latest-id query fail.
	latestUnavailable bool
	// recordCount is how many records the record-count query reports.
	recordCount uint64
	// countCalls counts record-count queries.
	countCalls int
	// recordCountUnavailable makes the record-count query fail.
	recordCountUnavailable bool
	// borLast is bor's last state sync ID; borUnavailable makes reading it fail.
	borLast        uint64
	borUnavailable bool
}

func newBurstNode(t *testing.T) (*burstNode, *httptest.Server, *httptest.Server) {
	t.Helper()

	registry := codectypes.NewInterfaceRegistry()
	authtypes.RegisterInterfaces(registry)
	cryptocodec.RegisterInterfaces(registry)
	clerktypes.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)

	n := &burstNode{t: t, cdc: cdc, txc: authtx.NewTxConfig(cdc, authtx.DefaultSignModes), stored: map[uint64]bool{}, latestOnL1: 1 << 40}
	rest := httptest.NewServer(http.HandlerFunc(n.serveREST))
	comet := httptest.NewServer(http.HandlerFunc(n.serveComet))
	t.Cleanup(rest.Close)
	t.Cleanup(comet.Close)
	return n, rest, comet
}

func (n *burstNode) serveREST(w http.ResponseWriter, r *http.Request) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.restCalls++

	var resp interface{ ProtoMessage() }
	switch {
	case r.URL.Path == util.ChainManagerParamsURL && n.paramsUnavailable:
		n.paramsCalls++
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	case r.URL.Path == util.ChainManagerParamsURL:
		n.paramsCalls++
		if block := n.paramsBlock; block != nil {
			n.mu.Unlock()
			<-block
			n.mu.Lock()
		}
		params := chainmanagertypes.DefaultParams()
		params.ChainParams.BorChainId = burstBorChainID
		resp = &chainmanagertypes.QueryParamsResponse{Params: params}
	case r.URL.Path == util.AccountParamsURL:
		resp = &authtypes.QueryParamsResponse{Params: authtypes.DefaultParams()}
	case r.URL.Path == util.ClerkLatestRecordIDURL && n.latestUnavailable:
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	case r.URL.Path == util.ClerkRecordCountURL && n.recordCountUnavailable:
		n.countCalls++
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	case r.URL.Path == util.ClerkRecordCountURL:
		n.countCalls++
		resp = &clerktypes.RecordCountResponse{Count: n.recordCount}
	case r.URL.Path == util.ClerkLatestRecordIDURL:
		resp = &clerktypes.LatestRecordIdResponse{LatestRecordId: n.latestOnL1}
	case r.URL.Path == "/clerk/event-records/list":
		var page, limit uint64
		_, err := fmt.Sscanf(r.URL.RawQuery, "page=%d&limit=%d", &page, &limit)
		require.NoError(n.t, err)
		n.listPages = append(n.listPages, page)
		if n.lookupDelay > 0 {
			n.mu.Unlock()
			select {
			case <-time.After(n.lookupDelay):
			case <-r.Context().Done():
			}
			n.mu.Lock()
		}
		if n.listUnavailable {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		list := &clerktypes.RecordListResponse{}
		for id := (page-1)*limit + 1; id <= page*limit; id++ {
			if n.stored[id] {
				list.EventRecords = append(list.EventRecords, clerktypes.EventRecord{Id: id})
			}
		}
		if len(list.EventRecords) == 0 && page > 1 {
			// What heimdall's keeper answers for an empty page.
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = fmt.Fprintf(w, `{"code":13,"message":"page %d does not exist","details":[]}`, page)
			return
		}
		resp = list
	case strings.HasPrefix(r.URL.Path, "/clerk/event-records/"):
		var id uint64
		_, err := fmt.Sscanf(r.URL.Path, util.ClerkEventRecordURL, &id)
		require.NoError(n.t, err)
		if n.recordLookups == nil {
			n.recordLookups = map[uint64]int{}
		}
		n.recordLookups[id]++
		if n.lookupDelay > 0 {
			n.mu.Unlock()
			select {
			case <-time.After(n.lookupDelay):
			case <-r.Context().Done():
			}
			n.mu.Lock()
		}
		if n.lookupStatus != 0 {
			w.WriteHeader(n.lookupStatus)
			return
		}
		if !n.stored[id] {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		resp = &clerktypes.RecordResponse{Record: clerktypes.EventRecord{Id: id}}
	default:
		n.t.Errorf("unexpected heimdall REST path: %s", r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	body, err := n.cdc.MarshalJSON(resp.(codec.ProtoMarshaler))
	require.NoError(n.t, err)
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(body)
}

func (n *burstNode) serveComet(w http.ResponseWriter, r *http.Request) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if r.Method == http.MethodGet && r.URL.Path == util.CometBFTUnconfirmedTxsCountURL {
		if n.countUnavailable {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":-1,"result":{"n_txs":"%d","total":"%d","total_bytes":"0"}}`, n.mempoolTotal, n.mempoolTotal)
		return
	}

	raw, err := io.ReadAll(r.Body)
	require.NoError(n.t, err)
	var req struct {
		ID     json.RawMessage `json:"id"`
		Method string          `json:"method"`
		Params struct {
			Tx []byte `json:"tx"`
		} `json:"params"`
	}
	require.NoError(n.t, json.Unmarshal(raw, &req))

	var result any
	var rpcErr error
	switch req.Method {
	case "abci_query":
		result = n.accountQuery()
	case "broadcast_tx_sync":
		result, rpcErr = n.broadcast(req.Params.Tx)
	case "unconfirmed_txs":
		result = n.unconfirmedTxs()
	default:
		n.t.Errorf("unexpected comet RPC method: %s", req.Method)
	}

	w.Header().Set("Content-Type", "application/json")
	if rpcErr != nil {
		_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32603,"message":"Internal error","data":%q}}`, req.ID, rpcErr.Error())
		return
	}
	body, err := cmtjson.Marshal(result)
	require.NoError(n.t, err)
	_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":%s}`, req.ID, body)
}

func (n *burstNode) accountQuery() *ctypes.ResultABCIQuery {
	acc := authtypes.NewBaseAccount(helper.GetAddress(), burstPubKey(), burstAccountNumber, burstCommittedSeq)
	anyAcc, err := codectypes.NewAnyWithValue(acc)
	require.NoError(n.t, err)
	value, err := n.cdc.Marshal(&authtypes.QueryAccountResponse{Account: anyAcc})
	require.NoError(n.t, err)
	return &ctypes.ResultABCIQuery{Response: abci.ResponseQuery{Value: value, Height: 1}}
}

func (n *burstNode) broadcast(txBytes []byte) (*ctypes.ResultBroadcastTx, error) {
	tx, err := n.txc.TxDecoder()(txBytes)
	require.NoError(n.t, err)
	sigs, err := tx.(authsigning.SigVerifiableTx).GetSignaturesV2()
	require.NoError(n.t, err)
	require.Len(n.t, sigs, 1)
	n.sequences = append(n.sequences, sigs[0].Sequence)
	record, ok := tx.GetMsgs()[0].(*clerktypes.MsgEventRecord)
	require.True(n.t, ok)
	n.broadcastData = append(n.broadcastData, record.Data)

	var next func() (*ctypes.ResultBroadcastTx, error)
	if len(n.broadcastResults) > 0 {
		next = n.broadcastResults[0]
		n.broadcastResults = n.broadcastResults[1:]
	}
	if next == nil {
		return &ctypes.ResultBroadcastTx{Code: abci.CodeTypeOK, Hash: []byte{0x01}}, nil
	}
	return next()
}

func (n *burstNode) unconfirmedTxs() *ctypes.ResultUnconfirmedTxs {
	res := &ctypes.ResultUnconfirmedTxs{}
	for _, msg := range n.pending {
		builder := n.txc.NewTxBuilder()
		require.NoError(n.t, builder.SetMsgs(msg))
		txBytes, err := n.txc.TxEncoder()(builder.GetTx())
		require.NoError(n.t, err)
		res.Txs = append(res.Txs, txBytes)
	}
	res.Count, res.Total = len(res.Txs), len(res.Txs)
	return res
}

// burstRecord is the MsgEventRecord a bridge builds for stateSyncedLog(id).
func burstRecord(id uint64) *clerktypes.MsgEventRecord {
	stateSenderABI, _ := statesender.StatesenderMetaData.GetAbi()
	data, _ := stateSenderABI.Events["StateSynced"].Inputs.NonIndexed().Pack([]byte{0xde, 0xad})
	var event statesender.StatesenderStateSynced
	_ = stateSenderABI.UnpackIntoInterface(&event, "StateSynced", data)
	msg := clerktypes.NewMsgEventRecord("", common.BigToHash(new(big.Int).SetUint64(id)).String(), 0, 100, id,
		common.HexToAddress("0x0000000000000000000000000000000000000abc").Bytes(), event.Data, burstBorChainID)
	return &msg
}

func insufficientFeeCheckTx() (*ctypes.ResultBroadcastTx, error) {
	return &ctypes.ResultBroadcastTx{Code: 13, Codespace: "sdk", Log: "insufficient fee"}, nil
}

func txInCacheRPC() (*ctypes.ResultBroadcastTx, error) {
	return nil, errors.New("tx already exists in cache")
}

func mempoolFullRPC() (*ctypes.ResultBroadcastTx, error) {
	return nil, errors.New("mempool is full: number of txs 5000 (max: 5000), total txs bytes 9740231 (max: 1073741824)")
}

func wrongSequenceCheckTx(expected uint64) func() (*ctypes.ResultBroadcastTx, error) {
	return func() (*ctypes.ResultBroadcastTx, error) {
		return &ctypes.ResultBroadcastTx{
			Code:      32,
			Codespace: "sdk",
			Log:       fmt.Sprintf("account sequence mismatch, expected %d, got %d: incorrect account sequence", expected, burstCommittedSeq),
		}, nil
	}
}

var burstPrivKey = cmtsecp256k1.GenPrivKey()

func unixAgo(d time.Duration) string { return strconv.FormatInt(time.Now().Add(-d).Unix(), 10) }

func burstPubKey() *cosmossecp256k1.PubKey {
	return &cosmossecp256k1.PubKey{Key: burstPrivKey.PubKey().Bytes()}
}

// newBurstClerkProcessor wires a clerk processor and a real TxBroadcaster to
// the stub node, as the bridge on one validator would be.
func newBurstClerkProcessor(t *testing.T, n *burstNode, rest, comet *httptest.Server) (*ClerkProcessor, *bytes.Buffer) {
	t.Helper()

	conf := helper.GetDefaultHeimdallConfig()
	conf.CometBFTRPCUrl = comet.URL
	srvConf := serverconfig.DefaultConfig()
	srvConf.API.Address = rest.URL
	helper.SetTestConfig(helper.CustomAppConfig{Config: *srvConf, Custom: conf})
	helper.SetTestPrivPubKey(burstPrivKey)

	prevClient := helper.Client
	helper.Client = &http.Client{}
	t.Cleanup(func() { helper.Client = prevClient })

	cometClient, err := rpchttp.New(comet.URL, "/websocket")
	require.NoError(t, err)

	cliCtx := client.Context{}.
		WithCodec(n.cdc).
		WithInterfaceRegistry(n.cdc.InterfaceRegistry()).
		WithTxConfig(n.txc).
		WithChainID("heimdall-test").
		WithClient(cometClient).
		WithAccountRetriever(authtypes.AccountRetriever{})

	tb := broadcaster.NewTxBroadcaster(n.cdc, context.Background(), cliCtx, func(string) sdk.AccountI {
		return authtypes.NewBaseAccount(helper.GetAddress(), burstPubKey(), burstAccountNumber, burstCommittedSeq)
	})

	stateSenderABI, err := statesender.StatesenderMetaData.GetAbi()
	require.NoError(t, err)

	logs := &bytes.Buffer{}
	cp := NewClerkProcessor(stateSenderABI)
	// Past its warm-up and seeded, as a bridge that has been running for a while.
	cp.startedAt = time.Now().Add(-clerkWarmup)
	cp.seeded.Store(true)
	n.borUnavailable = true
	cp.stateReceiver = func(string) (lastStateIDReader, error) {
		if n.borUnavailable {
			return nil, errors.New("bor unreachable")
		}
		return fakeStateReceiver{last: new(big.Int).SetUint64(n.borLast)}, nil
	}
	cp.Logger = log.NewLogger(logs, log.ColorOption(false), log.LevelOption(zerolog.DebugLevel))
	cp.cliCtx = cliCtx
	cp.txBroadcaster = tb
	cp.httpClient = cometClient
	return cp, logs
}

// burstOrigin is the L1 log origin of stateSyncedLog(t, id).
func burstOrigin(id uint64) string {
	return fmt.Sprintf("%s:%d", common.BigToHash(new(big.Int).SetUint64(id)).Hex(), 0)
}

func stateSyncedLog(t *testing.T, id uint64) string {
	t.Helper()
	return stateSyncedLogWithData(t, id, []byte{0xde, 0xad})
}

func stateSyncedLogWithData(t *testing.T, id uint64, payload []byte) string {
	t.Helper()

	stateSenderABI, err := statesender.StatesenderMetaData.GetAbi()
	require.NoError(t, err)
	event := stateSenderABI.Events["StateSynced"]
	data, err := event.Inputs.NonIndexed().Pack(payload)
	require.NoError(t, err)

	logBytes, err := json.Marshal(types.Log{
		Address: common.HexToAddress("0x0000000000000000000000000000000000001001"),
		Topics: []common.Hash{
			event.ID,
			common.BigToHash(new(big.Int).SetUint64(id)),
			common.BytesToHash(common.HexToAddress("0x0000000000000000000000000000000000000abc").Bytes()),
		},
		Data:        data,
		TxHash:      common.BigToHash(new(big.Int).SetUint64(id)),
		BlockNumber: 100,
	})
	require.NoError(t, err)
	return string(logBytes)
}

// runClerkTask runs one attempt of the clerk task under a machinery signature,
// so retry headers persist across attempts the way they do in the bridge.
func runClerkTask(t *testing.T, cp *ClerkProcessor, signature *tasks.Signature, id uint64) error {
	t.Helper()

	return runClerkTaskWithLog(t, cp, signature, stateSyncedLog(t, id))
}

func runClerkTaskWithLog(t *testing.T, cp *ClerkProcessor, signature *tasks.Signature, logBytes string) error {
	t.Helper()

	task, err := tasks.NewWithSignature(cp.sendStateSyncedToHeimdall, signature)
	require.NoError(t, err)
	return cp.sendStateSyncedToHeimdall(task.Context, "StateSynced", logBytes)
}

// requireAwaitingStore asserts a broadcast was accepted and the task now waits
// for heimdall to store the record.
func requireAwaitingStore(t *testing.T, err error) {
	t.Helper()

	var retryLater tasks.ErrRetryTaskLater
	require.True(t, errors.As(err, &retryLater), "got %v", err)
	require.Contains(t, retryLater.Error(), msgAwaitingStore)
}

// releasedSignature returns the signature of a task whose record entered the
// mempool clerkMaxStoreWait ago, its store check due.
func releasedSignature() *tasks.Signature {
	return &tasks.Signature{Headers: tasks.Headers{
		acceptedAtHeader:     unixAgo(clerkMaxStoreWait),
		windowReleasedHeader: headerTrue,
	}}
}

// requireWarmingUp asserts a warm-up deferral that runs again in about delay,
// the rest of the warm-up measured from the task's own clock.
func requireWarmingUp(t *testing.T, err error, delay time.Duration) {
	t.Helper()

	var retryLater tasks.ErrRetryTaskLater
	require.ErrorAs(t, err, &retryLater)
	require.Contains(t, retryLater.Error(), msgWarmingUp)
	// Plus the per-ID spread, below clerkDeferRetryDelay.
	require.InDelta(t, delay.Seconds(), retryLater.RetryIn().Seconds(), 0.5+clerkDeferRetryDelay.Seconds())
	require.GreaterOrEqual(t, retryLater.RetryIn(), delay-time.Second)
}

// fakeStateReceiver answers bor's last state sync ID.
type fakeStateReceiver struct {
	last     *big.Int
	err      error
	deadline *time.Duration
}

func (f fakeStateReceiver) LastStateId(opts *bind.CallOpts) (*big.Int, error) {
	if f.deadline != nil {
		if d, ok := opts.Context.Deadline(); ok {
			*f.deadline = time.Until(d)
		}
	}
	return f.last, f.err
}

// blockingStateReceiver answers only when its call's context ends.
type blockingStateReceiver struct{}

func (blockingStateReceiver) LastStateId(opts *bind.CallOpts) (*big.Int, error) {
	select {
	case <-opts.Context.Done():
		return nil, opts.Context.Err()
	case <-time.After(time.Minute):
		return big.NewInt(1), nil
	}
}

// orderDelay is the order-gate retry delay of a task for id whose base delay
// is base.
func orderDelay(id uint64, base time.Duration) time.Duration {
	return wholeSeconds(base + spread(id, base/5))
}

func requireRetryLater(t *testing.T, err error, msg string, delay time.Duration) {
	t.Helper()

	var retryLater tasks.ErrRetryTaskLater
	require.True(t, errors.As(err, &retryLater), "got %v", err)
	require.Contains(t, retryLater.Error(), msg)
	require.Equal(t, delay, retryLater.RetryIn())
}

// These run the real clerk task and broadcaster through the rejections a bridge
// meets during a state sync burst.
func TestClerkTaskUnderStateSyncBurst(t *testing.T) {
	// Not parallel: it swaps global helper config, keys and client, and asserts
	// deltas on global metric counters. Go runs parallel tests only after the
	// serial ones in a package finish, so those can't interleave with it.

	t.Run("busy mempool defers without broadcasting", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.mempoolTotal = clerkMempoolBusyTxs
		cp, logs := newBurstClerkProcessor(t, n, rest, comet)
		cp.pending.touch(49992-clerkFrontierIDs, "", time.Now())
		before := testutil.ToFloat64(metrics.ClerkMempoolBusyDeferrals)

		requireRetryLater(t, runClerkTask(t, cp, &tasks.Signature{}, 49992), msgMempoolBusy, clerkDeferRetryDelay)
		require.Equal(t, before+1, testutil.ToFloat64(metrics.ClerkMempoolBusyDeferrals))
		require.Empty(t, n.sequences)
		require.Contains(t, logs.String(), infoMsgClerkMempoolBusy)
	})

	t.Run("mempool below the busy mark broadcasts", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.mempoolTotal = clerkMempoolBusyTxs - 1
		cp, logs := newBurstClerkProcessor(t, n, rest, comet)
		cp.pending.touch(49992-clerkFrontierIDs, "", time.Now())

		requireAwaitingStore(t, runClerkTask(t, cp, &tasks.Signature{}, 49992))
		require.Contains(t, logs.String(), debugMsgClerkNewEventFound)
		require.Equal(t, []uint64{burstCommittedSeq}, n.sequences)
	})

	t.Run("already stored record is not broadcast", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.stored[49992] = true
		cp, logs := newBurstClerkProcessor(t, n, rest, comet)

		require.NoError(t, runClerkTask(t, cp, &tasks.Signature{}, 49992))
		require.Empty(t, n.sequences)
		require.Contains(t, logs.String(), infoMsgClerkIgnoringAlreadyProcessed)
		require.NotContains(t, logs.String(), debugMsgClerkNewEventFound)
	})

	t.Run("mempool full keeps the sequence for the retry", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		// The first tx (40) is accepted and stays pending, so the committed
		// account keeps reporting 40 while the node's next sequence is 41.
		n.broadcastResults = append(n.broadcastResults, nil, mempoolFullRPC)
		cp, logs := newBurstClerkProcessor(t, n, rest, comet)

		requireAwaitingStore(t, runClerkTask(t, cp, &tasks.Signature{}, 49991))
		signature := &tasks.Signature{}
		requireRetryLater(t, runClerkTask(t, cp, signature, 49992), broadcaster.ErrMempoolFull.Error(), util.RetryStateSyncTaskDelay)
		require.Contains(t, logs.String(), infoMsgClerkMempoolFull)
		requireAwaitingStore(t, runClerkTask(t, cp, signature, 49992))

		// Resetting to the committed sequence here would sign the retry with 40
		// and fail CheckTx on every following broadcast until 40 commits.
		require.Equal(t, []uint64{burstCommittedSeq, burstCommittedSeq + 1, burstCommittedSeq + 1}, n.sequences)
	})

	t.Run("own txs pending: sequence mismatch resyncs instead of looping", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		// Five of this validator's txs (40..44) sit in the mempool, so CheckTx expects 45
		// while the committed account still reports 40.
		n.broadcastResults = append(n.broadcastResults, wrongSequenceCheckTx(burstCommittedSeq+5))
		cp, logs := newBurstClerkProcessor(t, n, rest, comet)

		err := runClerkTask(t, cp, &tasks.Signature{}, 49992)
		require.ErrorContains(t, err, "non-ok response code: 32")
		require.Contains(t, logs.String(), errMsgClerkBroadcasting)
		requireAwaitingStore(t, runClerkTask(t, cp, &tasks.Signature{}, 49992))

		require.Equal(t, []uint64{burstCommittedSeq, burstCommittedSeq + 5}, n.sequences)
	})

	t.Run("a busy mempool keeps deferring however long the task has waited", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.mempoolTotal = 4 * clerkMempoolBusyTxs
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.pending.touch(49000, "", time.Now())

		requireRetryLater(t, runClerkTask(t, cp, &tasks.Signature{RetryCount: 0}, 49092), msgMempoolBusy, clerkDeferRetryDelay)
		require.Empty(t, n.sequences)
	})

	t.Run("record already pending in the mempool is not broadcast again", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.pending = []*clerktypes.MsgEventRecord{burstRecord(49992)}
		cp, logs := newBurstClerkProcessor(t, n, rest, comet)

		waiting := &tasks.Signature{}
		requireRetryLater(t, runClerkTask(t, cp, waiting, 49992), msgAwaitingStore, clerkMaxStoreWait)
		require.Empty(t, n.sequences)
		require.Contains(t, logs.String(), infoMsgClerkTxInMempool)
		require.Contains(t, logs.String(), "Duplicate clerk tx found in mempool")

		requireAwaitingStore(t, runClerkTask(t, cp, &tasks.Signature{}, 49993))
		require.Equal(t, []uint64{burstCommittedSeq}, n.sequences)
	})

	t.Run("data above the max state sync size is dropped", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		cp, logs := newBurstClerkProcessor(t, n, rest, comet)

		atLimit := make([]byte, helper.MaxStateSyncSize)
		requireAwaitingStore(t, runClerkTaskWithLog(t, cp, &tasks.Signature{}, stateSyncedLogWithData(t, 49992, atLimit)))
		require.NotContains(t, logs.String(), "data is too large")
		requireAwaitingStore(t, runClerkTaskWithLog(t, cp, &tasks.Signature{}, stateSyncedLogWithData(t, 49993, make([]byte, helper.MaxStateSyncSize+1))))
		require.Contains(t, logs.String(), "data is too large")

		require.Len(t, n.broadcastData, 2)
		require.Len(t, n.broadcastData[0], helper.MaxStateSyncSize)
		require.Empty(t, n.broadcastData[1])
	})

	t.Run("lowest pending records broadcast into a busy mempool up to the frontier mark", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.mempoolTotal = clerkFrontierBusyTxs - 1
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.pending.touch(49992-clerkFrontierIDs+1, "", time.Now())

		requireAwaitingStore(t, runClerkTask(t, cp, &tasks.Signature{}, 49992))
		require.Equal(t, []uint64{burstCommittedSeq}, n.sequences)

		n.mempoolTotal = clerkFrontierBusyTxs
		cp.mempool.forget()
		requireRetryLater(t, runClerkTask(t, cp, &tasks.Signature{}, 49991), msgMempoolBusy, clerkDeferRetryDelay)
		require.Len(t, n.sequences, 1)
	})

	t.Run("records far above the lowest pending one wait for it", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		cp, logs := newBurstClerkProcessor(t, n, rest, comet)
		cp.pending.touch(49992-clerkOrderWindow, "", time.Now())
		before := testutil.ToFloat64(metrics.ClerkOrderDeferrals)

		requireRetryLater(t, runClerkTask(t, cp, &tasks.Signature{}, 49992), msgAheadOfPending, orderDelay(49992, clerkDeferRetryDelay))
		require.Equal(t, before+1, testutil.ToFloat64(metrics.ClerkOrderDeferrals))
		require.Zero(t, n.restCalls, "an out-of-order task must not query heimdall")
		require.Contains(t, logs.String(), infoMsgClerkAheadOfPending)
		requireAwaitingStore(t, runClerkTask(t, cp, &tasks.Signature{}, 49992-clerkOrderWindow+1))
		require.Equal(t, []uint64{burstCommittedSeq}, n.sequences)

		// The lowest one is stored by another validator: its task ends and frees the window.
		n.stored[49992-clerkOrderWindow] = true
		require.NoError(t, runClerkTask(t, cp, &tasks.Signature{}, 49992-clerkOrderWindow))
		requireAwaitingStore(t, runClerkTask(t, cp, &tasks.Signature{}, 49992))
		require.Equal(t, []uint64{burstCommittedSeq, burstCommittedSeq + 1}, n.sequences)
	})

	t.Run("a full mempool past the age limit still retries and holds the window", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.broadcastResults = append(n.broadcastResults, mempoolFullRPC)
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)

		requireRetryLater(t, runClerkTask(t, cp, &tasks.Signature{RetryCount: 0}, 49992-clerkOrderWindow), broadcaster.ErrMempoolFull.Error(), util.RetryStateSyncTaskDelay)
		requireRetryLater(t, runClerkTask(t, cp, &tasks.Signature{}, 49992), msgAheadOfPending, orderDelay(49992, clerkDeferRetryDelay))
		require.Equal(t, []uint64{burstCommittedSeq}, n.sequences)
	})

	t.Run("a task failing its last attempt frees the window", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.broadcastResults = append(n.broadcastResults, insufficientFeeCheckTx, insufficientFeeCheckTx)
		cp, logs := newBurstClerkProcessor(t, n, rest, comet)
		before := testutil.ToFloat64(metrics.ClerkTasksExhausted)

		withRetries := &tasks.Signature{RetryCount: 1}
		require.ErrorContains(t, runClerkTask(t, cp, withRetries, 49992-clerkOrderWindow), "non-ok response code: 13")
		require.Contains(t, logs.String(), errMsgClerkBroadcasting)
		require.NotContains(t, logs.String(), errMsgClerkTaskExhausted)
		require.Equal(t, before, testutil.ToFloat64(metrics.ClerkTasksExhausted))
		requireRetryLater(t, runClerkTask(t, cp, &tasks.Signature{}, 49992), msgAheadOfPending, orderDelay(49992, clerkDeferRetryDelay))

		lastAttempt := &tasks.Signature{RetryCount: 0}
		require.ErrorContains(t, runClerkTask(t, cp, lastAttempt, 49992-clerkOrderWindow), "non-ok response code: 13")
		require.Contains(t, logs.String(), "ERR "+errMsgClerkTaskExhausted)
		require.Equal(t, before+1, testutil.ToFloat64(metrics.ClerkTasksExhausted))
		requireAwaitingStore(t, runClerkTask(t, cp, &tasks.Signature{}, 49992))
	})

	t.Run("the lowest pending record waits only at its own mark", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.mempoolTotal = clerkLowestBusyTxs - 1
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)

		requireAwaitingStore(t, runClerkTask(t, cp, &tasks.Signature{}, 49992))
		require.Equal(t, []uint64{burstCommittedSeq}, n.sequences)

		n.mempoolTotal = clerkLowestBusyTxs
		cp.mempool.forget()
		require.True(t, cp.lowestBusy.due(time.Now()))
		requireRetryLater(t, runClerkTask(t, cp, &tasks.Signature{}, 49993), msgMempoolBusy, clerkDeferRetryDelay)
		require.Len(t, n.sequences, 1)
	})

	t.Run("a lower released record doesn't keep the pending frontier from its busy-mempool pass", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.mempoolTotal = 4999
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.lowestBusy = rateGate{every: clerkLowestBusyEvery}
		cp.released.touch(49992, burstOrigin(49992), time.Now())

		requireAwaitingStore(t, runClerkTask(t, cp, &tasks.Signature{}, 49993))
		require.Equal(t, []uint64{burstCommittedSeq}, n.sequences, "the pending frontier ranks against pending IDs only")
		require.True(t, cp.released.has(49992, burstOrigin(49992), time.Now()))
	})

	t.Run("the lowest pending record still gets through a busy mempool once per interval", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.mempoolTotal = 4999
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.lowestBusy = rateGate{every: clerkLowestBusyEvery}

		requireAwaitingStore(t, runClerkTask(t, cp, &tasks.Signature{}, 49992))
		require.Equal(t, []uint64{burstCommittedSeq}, n.sequences, "other load alone doesn't stop state sync")

		requireRetryLater(t, runClerkTask(t, cp, &tasks.Signature{}, 49993), msgMempoolBusy, clerkDeferRetryDelay)
		require.Len(t, n.sequences, 1, "the next lowest waits for the next interval")

		cp.pending.touch(49991, "lower", time.Now())
		cp.lowestBusy.holdUntil(time.Time{})
		requireRetryLater(t, runClerkTask(t, cp, &tasks.Signature{}, 49994), msgMempoolBusy, clerkDeferRetryDelay)
		require.Len(t, n.sequences, 1, "only the lowest ID gets the pass")
	})

	t.Run("an unreadable mempool count holds back all but the frontier", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.countUnavailable = true
		cp, logs := newBurstClerkProcessor(t, n, rest, comet)
		cp.pending.touch(49992-clerkFrontierIDs, "", time.Now())
		before := testutil.ToFloat64(metrics.ClerkMempoolBusyDeferrals)

		requireRetryLater(t, runClerkTask(t, cp, &tasks.Signature{}, 49992), msgMempoolBusy, clerkDeferRetryDelay)
		require.Equal(t, before+1, testutil.ToFloat64(metrics.ClerkMempoolBusyDeferrals))
		require.Equal(t, 1, strings.Count(logs.String(), warnMsgClerkMempoolCount))
		requireRetryLater(t, runClerkTask(t, cp, &tasks.Signature{}, 49993), msgMempoolBusy, clerkDeferRetryDelay)
		require.Equal(t, 1, strings.Count(logs.String(), warnMsgClerkMempoolCount), "the cached failure is not logged again")
		require.Equal(t, 2, strings.Count(logs.String(), debugMsgClerkNoMempoolSize))
		require.Empty(t, n.sequences)

		requireAwaitingStore(t, runClerkTask(t, cp, &tasks.Signature{}, 49992-1))
		require.Equal(t, []uint64{burstCommittedSeq}, n.sequences)
	})

	t.Run("a copy with altered fields in the mempool does not block the real record", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.pending = []*clerktypes.MsgEventRecord{{
			TxHash:   common.BigToHash(big.NewInt(49992)).String(),
			LogIndex: 0,
			Id:       49992,
			Data:     []byte("not the event data"),
		}}
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)

		requireAwaitingStore(t, runClerkTask(t, cp, &tasks.Signature{}, 49992))
		require.Equal(t, []uint64{burstCommittedSeq}, n.sequences)
	})

	t.Run("an identical copy in the mempool settles the task like its own broadcast", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.pending = []*clerktypes.MsgEventRecord{burstRecord(49992)}
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)

		signature := &tasks.Signature{RetryCount: 0}
		requireRetryLater(t, runClerkTask(t, cp, signature, 49992), msgAwaitingStore, clerkMaxStoreWait)
		require.Empty(t, n.sequences)
		require.True(t, windowReleased(signature))
		require.Equal(t, clerkTaskRetries, signature.RetryCount)
		require.Equal(t, uint64(50000), cp.pending.touch(50000, "", time.Now()), "the copy's ID no longer holds the window")
	})

	t.Run("a task dying on a heimdall query on its last attempt frees the window", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.paramsUnavailable = true
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)

		require.Error(t, runClerkTask(t, cp, &tasks.Signature{RetryCount: 1}, 49992))
		require.Equal(t, uint64(49992), cp.pending.touch(50000, "", time.Now()))

		require.Error(t, runClerkTask(t, cp, &tasks.Signature{RetryCount: 0}, 49992))
		require.Equal(t, uint64(50000), cp.pending.touch(50000, "", time.Now()))
	})

	t.Run("a broadcast makes the next task read the mempool size again", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.mempoolTotal = clerkMempoolBusyTxs - 1
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.pending.touch(49992-clerkFrontierIDs, "", time.Now())

		requireAwaitingStore(t, runClerkTask(t, cp, &tasks.Signature{}, 49992))
		n.mempoolTotal = clerkMempoolBusyTxs
		requireRetryLater(t, runClerkTask(t, cp, &tasks.Signature{}, 49993), msgMempoolBusy, clerkDeferRetryDelay)
		require.Len(t, n.sequences, 1)
	})

	t.Run("an accepted broadcast moves the window on before heimdall stores it", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)

		lowest := &tasks.Signature{UUID: "lowest"}
		cp.pending.touch(49992-clerkOrderWindow, "lowest", time.Now())
		requireRetryLater(t, runClerkTask(t, cp, &tasks.Signature{}, 49992), msgAheadOfPending, orderDelay(49992, clerkDeferRetryDelay))

		requireAwaitingStore(t, runClerkTask(t, cp, lowest, 49992-clerkOrderWindow))
		requireAwaitingStore(t, runClerkTask(t, cp, &tasks.Signature{}, 49992))
		require.Equal(t, []uint64{burstCommittedSeq, burstCommittedSeq + 1}, n.sequences)
	})

	t.Run("a broadcast record is not resent while heimdall may still store it", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		cp, logs := newBurstClerkProcessor(t, n, rest, comet)

		signature := &tasks.Signature{}
		requireAwaitingStore(t, runClerkTask(t, cp, signature, 49992))
		acceptedAt, ok := retryStart(signature.Headers, acceptedAtHeader)
		require.True(t, ok)
		require.WithinDuration(t, time.Now(), acceptedAt, 5*time.Second)

		require.Equal(t, 1, strings.Count(logs.String(), debugMsgClerkBroadcast))
		signature.Headers[acceptedAtHeader] = unixAgo(clerkMaxStoreWait - time.Minute)
		requireAwaitingStore(t, runClerkTask(t, cp, signature, 49992))
		require.Equal(t, []uint64{burstCommittedSeq}, n.sequences, "no second broadcast before the check is due")
		require.Equal(t, 1, strings.Count(logs.String(), debugMsgClerkBroadcast))
		require.Equal(t, 1, strings.Count(logs.String(), debugMsgClerkAwaitingStore))

		n.stored[49992] = true
		require.NoError(t, runClerkTask(t, cp, signature, 49992))
		require.False(t, cp.released.has(49992, burstOrigin(49992), time.Now()), "a stored record stops being tracked")
	})

	t.Run("a record still unstored when its check is due is sent again", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		cp, logs := newBurstClerkProcessor(t, n, rest, comet)
		before := testutil.ToFloat64(metrics.ClerkRecordsUnstored)

		signature := &tasks.Signature{}
		requireAwaitingStore(t, runClerkTask(t, cp, signature, 49992))
		signature.Headers[acceptedAtHeader] = unixAgo(clerkMaxStoreWait)
		requireRetryLater(t, runClerkTask(t, cp, signature, 49992), msgAwaitingStore, clerkMaxStoreWait)
		require.Len(t, n.sequences, 2)
		require.Equal(t, before+1, testutil.ToFloat64(metrics.ClerkRecordsUnstored))
		require.Contains(t, logs.String(), "ERR "+errMsgClerkUnstored)
		require.Equal(t, uint64(50000), cp.pending.touch(50000, "", time.Now()), "a rebroadcast never rejoins the pending set")

		n.stored[49992] = true
		signature.Headers[acceptedAtHeader] = unixAgo(clerkMaxStoreWait)
		require.NoError(t, runClerkTask(t, cp, signature, 49992))
		require.Len(t, n.sequences, 2)
		require.Equal(t, before+1, testutil.ToFloat64(metrics.ClerkRecordsUnstored))
	})

	t.Run("unparsable log is dropped without querying heimdall", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		cp, logs := newBurstClerkProcessor(t, n, rest, comet)

		logBytes, err := json.Marshal(types.Log{Topics: []common.Hash{{0x01}}, TxHash: common.HexToHash("0x01")})
		require.NoError(t, err)
		require.NoError(t, runClerkTaskWithLog(t, cp, &tasks.Signature{}, string(logBytes)))
		require.Contains(t, logs.String(), errMsgClerkParsingEvent)
		require.Zero(t, n.restCalls)
		require.Empty(t, n.sequences)
	})

	t.Run("ordering holds on the last attempt too", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.pending.touch(49992-clerkOrderWindow, "", time.Now())

		requireRetryLater(t, runClerkTask(t, cp, &tasks.Signature{RetryCount: 0}, 49992), msgAheadOfPending, orderDelay(49992, clerkDeferRetryDelay))
		require.Empty(t, n.sequences)
		require.Equal(t, uint64(49992-clerkOrderWindow), cp.pending.touch(50000, "", time.Now()), "a deferral is not a last-attempt failure")
	})

	t.Run("an unreadable log fails the task", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		cp, logs := newBurstClerkProcessor(t, n, rest, comet)

		require.Error(t, runClerkTaskWithLog(t, cp, &tasks.Signature{}, "{not json"))
		require.Contains(t, logs.String(), errMsgClerkUnmarshallingEvent)
		require.Zero(t, n.restCalls)
	})

	t.Run("the lowest IDs, about a block of records, share the lowest mark", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.mempoolTotal = (clerkFrontierBusyTxs + clerkLowestBusyTxs) / 2
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.pending.touch(49992-clerkLowestIDs+1, "", time.Now())

		requireAwaitingStore(t, runClerkTask(t, cp, &tasks.Signature{}, 49992))
		cp.mempool.forget()
		requireRetryLater(t, runClerkTask(t, cp, &tasks.Signature{}, 49992-clerkLowestIDs+1+clerkLowestIDs), msgMempoolBusy, clerkDeferRetryDelay)
		require.Len(t, n.sequences, 1)
	})

	t.Run("a state sync with ID 0 is dropped and holds nothing back", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		cp, logs := newBurstClerkProcessor(t, n, rest, comet)

		require.NoError(t, runClerkTask(t, cp, &tasks.Signature{}, 0))
		require.Contains(t, logs.String(), errMsgClerkInvalidID)
		require.Zero(t, n.restCalls)
		requireAwaitingStore(t, runClerkTask(t, cp, &tasks.Signature{}, 150))
		require.Equal(t, []uint64{burstCommittedSeq}, n.sequences)
	})

	t.Run("an accepted broadcast checks storage once, clerkMaxStoreWait later", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)

		requireRetryLater(t, runClerkTask(t, cp, &tasks.Signature{}, 49992), msgAwaitingStore, clerkMaxStoreWait)
		requireRetryLater(t, runClerkTask(t, cp, &tasks.Signature{}, 49992+clerkFrontierIDs), msgAwaitingStore, clerkMaxStoreWait)
	})

	t.Run("chain manager params are read once per cache period", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		clock := time.Unix(1_793_846_312, 0)
		cp.chainParams.now = func() time.Time { return clock }

		paramsReads := func() int { return n.paramsCalls }
		_, err := cp.getClerkContext()
		require.NoError(t, err)
		clock = clock.Add(chainParamsCacheTTL - time.Nanosecond)
		_, err = cp.getClerkContext()
		require.NoError(t, err)
		require.Equal(t, 1, paramsReads())

		clock = clock.Add(time.Nanosecond)
		_, err = cp.getClerkContext()
		require.NoError(t, err)
		require.Equal(t, 2, paramsReads())
	})

	t.Run("chain params read by one task while the others get the last ones", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		clock := time.Unix(1_793_846_312, 0)
		cp.chainParams.now = func() time.Time { return clock }
		_, err := cp.getClerkContext()
		require.NoError(t, err)

		n.mu.Lock()
		n.paramsBlock = make(chan struct{})
		n.mu.Unlock()
		clock = clock.Add(chainParamsCacheTTL)
		done := make(chan struct{})
		go func() {
			defer close(done)
			_, _ = cp.getClerkContext()
		}()
		require.Eventually(t, func() bool {
			n.mu.Lock()
			defer n.mu.Unlock()
			return n.paramsCalls == 2
		}, 5*time.Second, time.Millisecond)

		_, err = cp.getClerkContext()
		require.NoError(t, err, "the last params while another task reads")
		n.mu.Lock()
		require.Equal(t, 2, n.paramsCalls)
		close(n.paramsBlock)
		n.mu.Unlock()
		<-done
	})

	t.Run("a task arriving during the first chain params read retries later without spending a retry", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.chainParams.reading = true

		requireRetryLater(t, runClerkTask(t, cp, &tasks.Signature{RetryCount: 0}, 49992), errChainParamsReadInProgress.Error(), clerkDeferRetryDelay)
	})

	t.Run("a failed chain params refresh keeps serving the last params", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		clock := time.Unix(1_793_846_312, 0)
		cp.chainParams.now = func() time.Time { return clock }

		first, err := cp.getClerkContext()
		require.NoError(t, err)
		n.paramsUnavailable = true
		clock = clock.Add(chainParamsCacheTTL)
		stale, err := cp.getClerkContext()
		require.NoError(t, err)
		require.Equal(t, first.ChainmanagerParams, stale.ChainmanagerParams)
		require.Equal(t, 2, n.paramsCalls)

		clock = clock.Add(chainParamsFailTTL - time.Nanosecond)
		_, err = cp.getClerkContext()
		require.NoError(t, err)
		require.Equal(t, 2, n.paramsCalls, "a failed refresh is shared too")
		clock = clock.Add(time.Nanosecond)
		_, err = cp.getClerkContext()
		require.NoError(t, err)
		require.Equal(t, 3, n.paramsCalls)
	})

	t.Run("an unreadable chain params read is shared for chainParamsFailTTL", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.paramsUnavailable = true
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		clock := time.Unix(1_793_846_312, 0)
		cp.chainParams.now = func() time.Time { return clock }

		_, err := cp.getClerkContext()
		require.Error(t, err)
		clock = clock.Add(chainParamsFailTTL - time.Nanosecond)
		_, err = cp.getClerkContext()
		require.Error(t, err)
		require.Equal(t, 1, n.paramsCalls)

		clock = clock.Add(time.Nanosecond)
		n.paramsUnavailable = false
		_, err = cp.getClerkContext()
		require.NoError(t, err)
		require.Equal(t, 2, n.paramsCalls)
	})

	t.Run("a state sync ID beyond int64 is dropped", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		cp, logs := newBurstClerkProcessor(t, n, rest, comet)

		require.NoError(t, runClerkTask(t, cp, &tasks.Signature{}, 1<<63+5))
		require.Contains(t, logs.String(), errMsgClerkInvalidID)
		require.Zero(t, n.restCalls)
		require.Equal(t, uint64(150), cp.pending.touch(150, "", time.Now()), "the dropped ID never became pending")
	})

	t.Run("a stored record ends the task without querying chain params", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.stored[49992] = true
		n.paramsUnavailable = true
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)

		require.NoError(t, runClerkTask(t, cp, &tasks.Signature{RetryCount: 0}, 49992))
		require.Zero(t, n.paramsCalls)
		require.Equal(t, uint64(50000), cp.pending.touch(50000, "", time.Now()))
	})

	t.Run("the lowest pending record is held only at the lowest mark", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.mempoolTotal = clerkLowestBusyTxs
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)

		require.True(t, cp.lowestBusy.due(time.Now()))
		requireRetryLater(t, runClerkTask(t, cp, &tasks.Signature{}, 49992), msgMempoolBusy, clerkDeferRetryDelay)
		n.mempoolTotal = clerkLowestBusyTxs - 1
		cp.mempool.forget()
		requireAwaitingStore(t, runClerkTask(t, cp, &tasks.Signature{}, 49992))
		require.Equal(t, []uint64{burstCommittedSeq}, n.sequences)

		n.broadcastResults = append(n.broadcastResults, mempoolFullRPC)
		requireRetryLater(t, runClerkTask(t, cp, &tasks.Signature{}, 49991), broadcaster.ErrMempoolFull.Error(), util.RetryStateSyncTaskDelay)
	})

	t.Run("an accepted broadcast restores the task's retry budget", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)

		spent := &tasks.Signature{RetryCount: 0, RetryTimeout: 610}
		requireAwaitingStore(t, runClerkTask(t, cp, spent, 49992))
		require.Equal(t, clerkTaskRetries, spent.RetryCount)
		require.Zero(t, spent.RetryTimeout, "machinery's backoff restarts too")

		larger := &tasks.Signature{RetryCount: clerkTaskRetries + 3}
		requireAwaitingStore(t, runClerkTask(t, cp, larger, 49993))
		require.Equal(t, clerkTaskRetries+3, larger.RetryCount)

		n.broadcastResults = append(n.broadcastResults, insufficientFeeCheckTx)
		failed := &tasks.Signature{RetryCount: 1}
		require.Error(t, runClerkTask(t, cp, failed, 49994))
		require.Equal(t, 1, failed.RetryCount)
	})

	t.Run("a released record with an ID above the latest on L1 holds its rebroadcasts", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.latestOnL1 = 49991
		cp, logs := newBurstClerkProcessor(t, n, rest, comet)
		before := testutil.ToFloat64(metrics.ClerkRecordsNotOnL1)

		signature := releasedSignature()
		requireRetryLater(t, runClerkTask(t, cp, signature, 49992), msgNotOnL1, clerkMaxStoreWait)
		require.Empty(t, n.sequences, "no broadcast while above the L1 counter")
		require.Equal(t, before+1, testutil.ToFloat64(metrics.ClerkRecordsNotOnL1))
		require.Contains(t, logs.String(), "WRN "+warnMsgClerkNotOnL1)
		require.Equal(t, uint64(50000), cp.pending.touch(50000, "", time.Now()), "it doesn't hold the window")

		n.latestOnL1 = 49992
		cp.latestOnL1.at = time.Time{}
		requireAwaitingStore(t, runClerkTask(t, cp, signature, 49992))
		require.Len(t, n.sequences, 1, "a lagging counter only delays the record")
	})

	t.Run("a record heimdall keeps refusing is given up once it is too old", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.latestOnL1 = 49991
		cp, logs := newBurstClerkProcessor(t, n, rest, comet)
		before := testutil.ToFloat64(metrics.ClerkTasksExhausted)
		cp.released.touch(49992, burstOrigin(49992), time.Now())

		signature := releasedSignature()
		signature.Headers[firstAcceptedAtHeader] = unixAgo(clerkMaxReleasedAge)
		require.NoError(t, runClerkTask(t, cp, signature, 49992), "the task ends")
		require.Empty(t, n.sequences, "no broadcast, not even while above the L1 counter")
		require.Equal(t, before+1, testutil.ToFloat64(metrics.ClerkTasksExhausted))
		require.Contains(t, logs.String(), "ERR "+errMsgClerkAbandoned)
		require.False(t, cp.released.has(49992, burstOrigin(49992), time.Now()), "a later task for the ID isn't turned away")
	})

	t.Run("an unreadable lookup at the age cap doesn't give up on the record", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.lookupStatus = http.StatusServiceUnavailable
		cp, logs := newBurstClerkProcessor(t, n, rest, comet)
		before := testutil.ToFloat64(metrics.ClerkTasksExhausted)
		cp.released.touch(49992, burstOrigin(49992), time.Now())

		signature := releasedSignature()
		signature.Headers[firstAcceptedAtHeader] = unixAgo(clerkMaxReleasedAge)
		requireRetryLater(t, runClerkTask(t, cp, signature, 49992), msgRecordUnreadable, abandonRetryDelay(49992))
		require.Equal(t, before, testutil.ToFloat64(metrics.ClerkTasksExhausted))
		require.NotContains(t, logs.String(), errMsgClerkAbandoned)
		require.True(t, cp.released.has(49992, burstOrigin(49992), time.Now()), "the task keeps its lease")
		require.Empty(t, n.sequences)

		n.lookupStatus = 0
		require.NoError(t, runClerkTask(t, cp, signature, 49992), "it gives up once heimdall answers that the record is missing")
		require.Equal(t, before+1, testutil.ToFloat64(metrics.ClerkTasksExhausted))
	})

	t.Run("a standby that still sees the lease waits even after the give-up is recorded", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.released.touch(49992, burstOrigin(49992), time.Now())
		cp.released.abandoned.add(49992, burstOrigin(49992), time.Now())

		standby := &tasks.Signature{UUID: "standby"}
		requireRetryLater(t, runClerkTask(t, cp, standby, 49992), msgReleasedOwned, clerkMaxStoreWait)

		cp.released.doneOrigin(49992, burstOrigin(49992))
		require.NoError(t, runClerkTask(t, cp, standby, 49992), "once the lease is gone it ends")
		require.Empty(t, n.sequences)
	})

	t.Run("a standby queued before the original gives up doesn't restart the clock", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		cp, logs := newBurstClerkProcessor(t, n, rest, comet)

		original := &tasks.Signature{UUID: "original"}
		requireAwaitingStore(t, runClerkTask(t, cp, original, 49992))
		standby := &tasks.Signature{UUID: "standby"}
		requireRetryLater(t, runClerkTask(t, cp, standby, 49992), msgReleasedOwned, clerkMaxStoreWait)
		require.Equal(t, headerTrue, headerString(standby.Headers, standbyHeader))

		original.Headers[acceptedAtHeader] = unixAgo(clerkMaxStoreWait)
		original.Headers[firstAcceptedAtHeader] = unixAgo(clerkMaxReleasedAge)
		require.NoError(t, runClerkTask(t, cp, original, 49992), "the original gives up")
		require.Contains(t, logs.String(), "ERR "+errMsgClerkAbandoned)

		require.NoError(t, runClerkTask(t, cp, standby, 49992), "the standby ends instead of taking over")
		require.NoError(t, runClerkTask(t, cp, &tasks.Signature{UUID: "replay"}, 49992), "so does a replay within the cap")
		require.Len(t, n.sequences, 1, "nothing broadcast after the give-up")
		require.Empty(t, headerString(standby.Headers, firstAcceptedAtHeader))
	})

	t.Run("giving up on one origin keeps another origin's lease for the same ID", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		const other = "0xother:7"
		cp.released.touch(49992, burstOrigin(49992), time.Now())
		cp.released.touch(49992, other, time.Now())

		signature := releasedSignature()
		signature.Headers[firstAcceptedAtHeader] = unixAgo(clerkMaxReleasedAge)
		require.NoError(t, runClerkTask(t, cp, signature, 49992))
		require.False(t, cp.released.has(49992, burstOrigin(49992), time.Now()))
		require.True(t, cp.released.has(49992, other, time.Now()), "the other log's task still owns its lease")
		require.Empty(t, n.sequences)
	})

	t.Run("a rebroadcast keeps the time the record first entered the mempool", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		before := testutil.ToFloat64(metrics.ClerkRecordsUnstored)

		signature := &tasks.Signature{}
		requireAwaitingStore(t, runClerkTask(t, cp, signature, 49992))
		first := signature.Headers[firstAcceptedAtHeader]
		require.Equal(t, signature.Headers[acceptedAtHeader], first)

		first = unixAgo(time.Hour - time.Minute)
		signature.Headers[firstAcceptedAtHeader] = first
		signature.Headers[acceptedAtHeader] = unixAgo(clerkMaxStoreWait)
		requireAwaitingStore(t, runClerkTask(t, cp, signature, 49992))
		require.Len(t, n.sequences, 2)
		require.Equal(t, first, signature.Headers[firstAcceptedAtHeader])
		require.NotEqual(t, first, signature.Headers[acceptedAtHeader])
		require.Equal(t, before+1, testutil.ToFloat64(metrics.ClerkRecordsUnstored))
	})

	t.Run("a rebroadcast held back by the busy gate is counted once", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.mempoolTotal = clerkLowestBusyTxs
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		before := testutil.ToFloat64(metrics.ClerkRecordsUnstored)
		require.True(t, cp.lowestBusy.due(time.Now()))

		signature := releasedSignature()
		for range 3 {
			requireRetryLater(t, runClerkTask(t, cp, signature, 49992), msgMempoolBusy, clerkDeferRetryDelay)
		}
		require.Empty(t, n.sequences)
		require.Equal(t, before+1, testutil.ToFloat64(metrics.ClerkRecordsUnstored))
	})

	t.Run("an unreadable latest ID on L1 doesn't hold a rebroadcast", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.latestUnavailable = true
		cp, logs := newBurstClerkProcessor(t, n, rest, comet)

		requireAwaitingStore(t, runClerkTask(t, cp, releasedSignature(), 49992))
		require.Len(t, n.sequences, 1)
		require.NotContains(t, logs.String(), warnMsgClerkNotOnL1)
	})

	t.Run("a released task broadcasts into a busy mempool and a second task for its ID waits", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		cp, logs := newBurstClerkProcessor(t, n, rest, comet)

		signature := &tasks.Signature{}
		requireAwaitingStore(t, runClerkTask(t, cp, signature, 49992))
		require.True(t, windowReleased(signature))
		cp.pending.touch(49000, "", time.Now())

		requireRetryLater(t, runClerkTask(t, cp, &tasks.Signature{}, 49992), msgReleasedOwned, clerkMaxStoreWait)
		require.Contains(t, logs.String(), debugMsgClerkDuplicateTask)
		require.Len(t, n.sequences, 1)
		require.Equal(t, uint64(49000), cp.pending.touch(49001, "", time.Now()), "the duplicate never rejoins the pending set")

		n.mempoolTotal = clerkLowestBusyTxs - 1
		cp.mempool.forget()
		signature.Headers[acceptedAtHeader] = unixAgo(clerkMaxStoreWait)
		requireRetryLater(t, runClerkTask(t, cp, signature, 49992), msgMempoolBusy, deferDelay(49992, 49000, true))
		require.Len(t, n.sequences, 1, "a lower pending ID keeps a released one behind the busy mark")
		require.Greater(t, deferDelay(49992, 49000, true), releasedRetryDelay-time.Second, "on the slow cadence")

		cp.pending.done(49000)
		cp.pending.done(49001)
		requireAwaitingStore(t, runClerkTask(t, cp, signature, 49992))
		require.Len(t, n.sequences, 2, "the lowest unstored ID is held only at the lowest mark")
		require.True(t, cp.released.has(49992, burstOrigin(49992), time.Now()), "each run of the released task keeps its entry alive")

		n.mempoolTotal = clerkLowestBusyTxs
		cp.mempool.forget()
		signature.Headers[acceptedAtHeader] = unixAgo(clerkMaxStoreWait)
		require.True(t, cp.lowestBusy.due(time.Now()))
		requireRetryLater(t, runClerkTask(t, cp, signature, 49992), msgMempoolBusy, clerkDeferRetryDelay)

		n.stored[49992] = true
		require.NoError(t, runClerkTask(t, cp, signature, 49992))
		// Storing clears the released ID: a later task for it runs again, not ended.
		n.stored[49992] = false
		n.mempoolTotal = 0
		cp.mempool.forget()
		requireAwaitingStore(t, runClerkTask(t, cp, &tasks.Signature{}, 49992))
	})

	t.Run("a released task that runs out of retries frees its ID for another task", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.broadcastResults = append(n.broadcastResults, insufficientFeeCheckTx)
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)

		lastAttempt := releasedSignature()
		lastAttempt.RetryCount = 0
		require.Error(t, runClerkTask(t, cp, lastAttempt, 49992))
		require.False(t, cp.released.has(49992, burstOrigin(49992), time.Now()))
		requireAwaitingStore(t, runClerkTask(t, cp, &tasks.Signature{}, 49992))
	})

	t.Run("after a restart no task broadcasts until the order window refilled", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		n.recordCount = 1 << 40
		cp.seeded.Store(false)
		cp.startedAt = time.Now().Add(-clerkWarmup + clerkDeferRetryDelay + time.Second)

		requireWarmingUp(t, runClerkTask(t, cp, &tasks.Signature{}, 49992+clerkOrderWindow), clerkDeferRetryDelay+time.Second)
		requireWarmingUp(t, runClerkTask(t, cp, releasedSignature(), 49800), clerkDeferRetryDelay+time.Second)
		cp.startedAt = time.Now().Add(-clerkWarmup + time.Second)
		requireWarmingUp(t, runClerkTask(t, cp, &tasks.Signature{}, 49992), clerkDeferRetryDelay)
		require.Empty(t, n.sequences)
		require.Equal(t, 1, n.countCalls, "warm-up reads the record count once")
		require.Equal(t, 1, n.paramsCalls, "and the chain params for bor's StateReceiver")
		require.Equal(t, 2, n.restCalls, "and nothing else")

		cp.startedAt = time.Now().Add(-clerkWarmup)
		requireRetryLater(t, runClerkTask(t, cp, &tasks.Signature{}, 49992+clerkOrderWindow), msgAheadOfPending, orderDelay(49992+clerkOrderWindow, clerkDeferRetryDelay))
		requireAwaitingStore(t, runClerkTask(t, cp, &tasks.Signature{}, 49992))
		require.True(t, cp.released.has(49800, burstOrigin(49800), time.Now()), "a released task keeps its entry during warm-up")
	})

	t.Run("a task during warm-up runs again when it ends", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.seeded.Store(false)
		cp.startedAt = time.Now()

		var retryLater tasks.ErrRetryTaskLater
		require.ErrorAs(t, runClerkTask(t, cp, &tasks.Signature{}, 49992), &retryLater)
		require.InDelta(t, wholeSeconds(clerkWarmup+spread(49992, clerkDeferRetryDelay)).Seconds(), retryLater.RetryIn().Seconds(), 1)
	})

	t.Run("after a restart a task delayed past the warm-up isn't overtaken", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.recordCount = 49991
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.seeded.Store(false)
		cp.startedAt = time.Now()

		// Only a far-ahead task runs during the warm-up; 49992's task is delayed.
		requireWarmingUp(t, runClerkTask(t, cp, &tasks.Signature{}, 49992+clerkOrderWindow), clerkWarmup)
		cp.startedAt = time.Now().Add(-clerkWarmup)
		requireRetryLater(t, runClerkTask(t, cp, &tasks.Signature{}, 49992+clerkOrderWindow), msgAheadOfPending, orderDelay(49992+clerkOrderWindow, clerkDeferRetryDelay))
		require.Empty(t, n.sequences)

		requireAwaitingStore(t, runClerkTask(t, cp, &tasks.Signature{}, 49992))
	})

	t.Run("warm-up retries are spread per ID", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		now := time.Now()
		cp.startedAt = now
		cp.seeded.Store(true)

		var first, second tasks.ErrRetryTaskLater
		require.ErrorAs(t, cp.warmUp(context.Background(), 49991, "a", false, now), &first)
		require.ErrorAs(t, cp.warmUp(context.Background(), 49992, "a", false, now), &second)
		require.Equal(t, 31*time.Second, first.RetryIn(), "30.3s up to whole seconds")
		require.Equal(t, 31*time.Second, second.RetryIn())

		var later tasks.ErrRetryTaskLater
		require.ErrorAs(t, cp.warmUp(context.Background(), 49990, "a", false, now.Add(1500*time.Millisecond)), &later)
		require.Equal(t, clerkWarmup-time.Second, later.RetryIn(), "28.5s up to whole seconds, so a restart declares few delay queues")
	})

	t.Run("the seed is the ID bor waits on when heimdall has gaps", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.recordCount = 49995
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.seeded.Store(false)
		n.borUnavailable = false
		n.borLast = 49991

		cp.seedFrontier(context.Background(), time.Now())
		lowest, _, ok := cp.pending.lowest(time.Now())
		require.True(t, ok)
		require.Equal(t, uint64(49992), lowest)
	})

	t.Run("a bor behind heimdall doesn't pull the seed onto stored records", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.recordCount = 52000
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.seeded.Store(false)
		n.borUnavailable = false
		n.borLast = 49991
		for id := uint64(49992); id < 49992+clerkOrderWindow; id++ {
			n.stored[id] = true
		}

		cp.seedFrontier(context.Background(), time.Now())
		lowest, _, _ := cp.pending.lowest(time.Now())
		require.Equal(t, uint64(49992+clerkOrderWindow), lowest, "no gap within a window of bor: the first ID the scan didn't read")
		require.Equal(t, []uint64{1000, 1001, 1002, 1002, 1003, 1004}, n.listPages, "the clean window after bor, then the next one")
		require.True(t, cp.seeded.Load(), "a clean scan backs the seed")
	})

	t.Run("a stored seed short of heimdall's bound moves on to the next gap", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.recordCount = 52000
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.seeded.Store(false)
		n.borUnavailable = false
		n.borLast = 49991
		for id := uint64(49992); id < 49992+clerkOrderWindow; id++ {
			n.stored[id] = true
		}

		now := time.Now()
		cp.seedFrontier(context.Background(), now)
		seed := uint64(49992 + clerkOrderWindow)
		lowest, _, _ := cp.pending.lowest(now)
		require.Equal(t, seed, lowest)

		n.stored[seed] = true
		later := now.Add(2 * clerkLowestIdle)
		require.Equal(t, seed+1, cp.dropStoredLowest(context.Background(), seed, later), "the next ID heimdall lacks holds the window")
		require.True(t, cp.pending.seedOnly(seed+1))

		for id := seed + 1; id <= seed+1+clerkOrderWindow; id++ {
			n.stored[id] = true
		}
		later = later.Add(2 * clerkLowestIdle)
		require.Equal(t, seed+2+clerkOrderWindow, cp.dropStoredLowest(context.Background(), seed+1, later), "a clean window moves the seed past it")

		n.recordCount = seed + 2 + clerkOrderWindow
		n.stored[seed+2+clerkOrderWindow] = true
		later = later.Add(2 * clerkLowestIdle)
		next := cp.dropStoredLowest(context.Background(), seed+2+clerkOrderWindow, later)
		_, _, ok := cp.pending.lowest(later)
		require.False(t, ok, "past heimdall's bound nothing is left to seed")
		require.Zero(t, next)
	})

	t.Run("a stored seed stays when the next scan can't run", func(t *testing.T) {
		for _, tc := range []struct {
			name   string
			break_ func(n *burstNode, cp *ClerkProcessor)
		}{
			{"record count unreadable", func(n *burstNode, _ *ClerkProcessor) { n.recordCountUnavailable = true }},
			{"record list unreadable", func(n *burstNode, _ *ClerkProcessor) { n.listUnavailable = true }},
			{"another seeding running", func(_ *burstNode, cp *ClerkProcessor) { cp.seeding.Store(true) }},
		} {
			t.Run(tc.name, func(t *testing.T) {
				n, rest, comet := newBurstNode(t)
				n.recordCount = 52000
				cp, _ := newBurstClerkProcessor(t, n, rest, comet)
				now := time.Now()
				cp.pending.seed(50092, now)
				n.stored[50092] = true
				tc.break_(n, cp)

				later := now.Add(2 * clerkLowestIdle)
				require.Equal(t, uint64(50092), cp.dropStoredLowest(context.Background(), 50092, later))
				require.True(t, cp.pending.seedOnly(50092), "kept for a later check")
			})
		}
	})

	t.Run("one seed scan walks several windows to the first gap", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.recordCount = 52000
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.seeded.Store(false)
		n.borUnavailable = false
		n.borLast = 49991
		gap := uint64(49992 + 3*clerkOrderWindow + 7)
		for id := uint64(49992); id < gap; id++ {
			n.stored[id] = true
		}

		cp.seedFrontier(context.Background(), time.Now())
		lowest, _, _ := cp.pending.lowest(time.Now())
		require.Equal(t, gap, lowest)
	})

	t.Run("a seed scan stops after its windows on the first ID it didn't read", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.recordCount = 52000
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.seeded.Store(false)
		n.borUnavailable = false
		n.borLast = 49991
		end := uint64(49992 + seedScanWindows*clerkOrderWindow)
		for id := uint64(49992); id <= end; id++ {
			n.stored[id] = true
		}

		cp.seedFrontier(context.Background(), time.Now())
		lowest, _, _ := cp.pending.lowest(time.Now())
		require.Equal(t, end, lowest)
	})

	t.Run("a seed scan reaching heimdall's bound seeds past it", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.recordCount = 50100
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.seeded.Store(false)
		n.borUnavailable = false
		n.borLast = 49991
		for id := uint64(49992); id <= 50100; id++ {
			n.stored[id] = true
		}

		cp.seedFrontier(context.Background(), time.Now())
		lowest, _, _ := cp.pending.lowest(time.Now())
		require.Equal(t, uint64(50101), lowest)
	})

	t.Run("a stored seed's own task moves the seed on", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.recordCount = 52000
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		now := time.Now()
		cp.pending.seed(50092, now)
		n.stored[50092] = true

		cp.finishStored(context.Background(), 50092, now)
		lowest, _, _ := cp.pending.lowest(now)
		require.Equal(t, uint64(50093), lowest, "the next ID heimdall lacks")
		require.True(t, cp.pending.seeded(50093))

		n.stored[50093] = true
		n.listUnavailable = true
		cp.finishStored(context.Background(), 50093, now)
		require.True(t, cp.pending.seedOnly(50093), "kept for a later check when the scan can't run")

		cp.pending.touch(49000, "task", now)
		cp.finishStored(context.Background(), 49000, now)
		lowest, _, _ = cp.pending.lowest(now)
		require.Equal(t, uint64(50093), lowest, "a task that isn't the seed just ends")
	})

	t.Run("a record count beyond int64 seeds nothing", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.recordCount = math.MaxInt64
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.seeded.Store(false)
		n.borUnavailable = false
		n.borLast = 49991

		cp.seedFrontier(context.Background(), time.Now())
		_, _, ok := cp.pending.lowest(time.Now())
		require.False(t, ok)
		require.False(t, cp.seeded.Load())
	})

	t.Run("a seed out of time before the record count seeds nothing", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.recordCount = 52000
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.seeded.Store(false)
		n.borUnavailable = false
		n.borLast = 49991
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		cp.seedFrontier(ctx, time.Now())
		_, _, ok := cp.pending.lowest(time.Now())
		require.False(t, ok)
		require.False(t, cp.seeded.Load(), "a later task tries again")
		require.Empty(t, n.listPages)
	})

	t.Run("a slow heimdall can't make the seed scan outlast seedTimeout", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.recordCount = 52000
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.seeded.Store(false)
		n.borUnavailable = false
		n.borLast = 49991
		for id := uint64(49992); id < 49991+clerkOrderWindow; id++ {
			n.stored[id] = true
		}
		n.lookupDelay = seedTimeout * 2 / 5

		started := time.Now()
		cp.seedFrontier(context.Background(), started)
		require.Less(t, time.Since(started), seedTimeout+time.Second)
		lowest, _, _ := cp.pending.lowest(time.Now())
		require.Equal(t, uint64(52001), lowest, "out of time before the gap at the window's end")
	})

	t.Run("a bridge quiet through the warm-up seeds on its first task", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.recordCount = 49991
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.seeded.Store(false)
		n.borUnavailable = false
		n.borLast = n.recordCount

		requireRetryLater(t, runClerkTask(t, cp, &tasks.Signature{}, 49992+clerkOrderWindow), msgAheadOfPending, orderDelay(49992+clerkOrderWindow, clerkDeferRetryDelay))
		require.True(t, cp.seeded.Load())
		require.Empty(t, n.sequences)
	})

	t.Run("a record count unreadable through the warm-up is seeded once readable", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.recordCountUnavailable = true
		n.recordCount = 49991
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.seeded.Store(false)
		n.borUnavailable = false
		n.borLast = n.recordCount

		requireAwaitingStore(t, runClerkTask(t, cp, &tasks.Signature{}, 49992+clerkOrderWindow))
		require.False(t, cp.seeded.Load(), "an unreadable heimdall doesn't stall state sync")

		n.recordCountUnavailable = false
		cp.seedCheck = rateGate{}
		requireRetryLater(t, runClerkTask(t, cp, &tasks.Signature{}, 49992+2*clerkOrderWindow), msgAheadOfPending, orderDelay(49992+2*clerkOrderWindow, 2*clerkDeferRetryDelay))
		require.True(t, cp.seeded.Load())
	})

	t.Run("the seed deadline cuts a record lookup short", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.recordCount = 49992
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.seeded.Store(false)
		n.borUnavailable = false
		n.borLast = 49991
		n.lookupDelay = time.Minute

		started := time.Now()
		cp.seedFrontier(context.Background(), started)
		require.Less(t, time.Since(started), seedTimeout+time.Second)
		lowest, _, _ := cp.pending.lowest(time.Now())
		require.Equal(t, uint64(49993), lowest)
	})

	t.Run("a seed waits for chain params another task is reading", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.recordCount = 52000
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.seeded.Store(false)
		cp.chainParams.reading = true

		cp.seedFrontier(context.Background(), time.Now())
		require.False(t, cp.seeded.Load(), "a later task seeds with the params")
		_, _, ok := cp.pending.lowest(time.Now())
		require.False(t, ok)
	})

	t.Run("a seed taken without bor is provisional until bor answers", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.recordCount = 52000
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.seeded.Store(false)
		start := time.Now()
		cp.startedAt = start.Add(-clerkWarmup)
		n.stored[49992] = true

		cp.seedFrontier(context.Background(), start)
		lowest, _, _ := cp.pending.lowest(start)
		require.Equal(t, uint64(52001), lowest, "heimdall's bound meanwhile")
		require.False(t, cp.seeded.Load(), "bor is asked again")

		n.recordCount = 52010
		cp.seedFrontier(context.Background(), start.Add(clerkSeedRetryBackoff))
		lowest, _, _ = cp.pending.lowest(start.Add(clerkSeedRetryBackoff))
		require.Equal(t, uint64(52001), lowest, "the provisional seed is placed once")
		require.Len(t, cp.pending.leases.entries, 1)

		n.borUnavailable = false
		n.borLast = 49991
		cp.seedFrontier(context.Background(), start.Add(2*clerkSeedRetryBackoff))
		lowest, _, _ = cp.pending.lowest(start.Add(2 * clerkSeedRetryBackoff))
		require.Equal(t, uint64(49993), lowest, "the gap below the bound, once bor answers")
		require.Len(t, cp.pending.leases.entries, 1)
		require.True(t, cp.seeded.Load())
	})

	t.Run("a bor answering during the warm-up replaces the provisional seed before it ends", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.recordCount = 52000
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.seeded.Store(false)
		start := time.Now()
		cp.startedAt = start
		n.stored[49992] = true

		cp.seedFrontier(context.Background(), start)
		lowest, _, _ := cp.pending.lowest(start)
		require.Equal(t, uint64(52001), lowest)

		n.borUnavailable = false
		n.borLast = 49991
		cp.seedFrontier(context.Background(), start.Add(clerkStoreCheckDelay))
		lowest, _, _ = cp.pending.lowest(start.Add(clerkStoreCheckDelay))
		require.Equal(t, uint64(49993), lowest)
		require.True(t, cp.seeded.Load())
	})

	t.Run("a node without bor stops asking it after clerkSeedRetryWindow", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.recordCount = 52000
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.seeded.Store(false)
		start := time.Now()
		cp.startedAt = start

		cp.seedFrontier(context.Background(), start.Add(clerkSeedRetryWindow-clerkSeedRetryBackoff))
		require.False(t, cp.seeded.Load())
		cp.seedFrontier(context.Background(), start.Add(clerkSeedRetryWindow))
		require.True(t, cp.seeded.Load())
	})

	t.Run("only one task seeds at a time", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.recordCount = 49991
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.seeded.Store(false)
		cp.seeding.Store(true)

		cp.seedFrontier(context.Background(), time.Now())
		require.Zero(t, n.countCalls)
		require.False(t, cp.seeded.Load())

		cp.seeding.Store(false)
		cp.seedFrontier(context.Background(), time.Now())
		require.Equal(t, 1, n.countCalls)
		require.False(t, cp.seeding.Load(), "released when done")
	})

	t.Run("the gap scan stops at heimdall's highest stored ID", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.recordCount = 49993
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.seeded.Store(false)
		n.borUnavailable = false
		n.borLast = 49991
		n.stored[49992] = true
		n.stored[49993] = true

		cp.seedFrontier(context.Background(), time.Now())
		lowest, _, _ := cp.pending.lowest(time.Now())
		require.Equal(t, uint64(49994), lowest)
		require.Equal(t, []uint64{1000}, n.listPages, "nothing above the highest stored ID is read")
		require.True(t, cp.seeded.Load(), "bor backs a seed with no gap below heimdall's bound")
	})

	t.Run("the seed is the first gap after bor's next ID", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.recordCount = 52000
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.seeded.Store(false)
		n.borUnavailable = false
		n.borLast = 49991
		n.stored[49992] = true
		n.stored[49994] = true

		cp.seedFrontier(context.Background(), time.Now())
		lowest, _, _ := cp.pending.lowest(time.Now())
		require.Equal(t, uint64(49993), lowest)
	})

	t.Run("the gap scan reads the page holding bor's next ID on a page boundary", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.recordCount = 52000
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.seeded.Store(false)
		n.borUnavailable = false
		n.borLast = 49999
		for id := uint64(50000); id < 50000+clerkOrderWindow; id++ {
			n.stored[id] = true
		}

		cp.seedFrontier(context.Background(), time.Now())
		lowest, _, _ := cp.pending.lowest(time.Now())
		require.Equal(t, uint64(50000+clerkOrderWindow), lowest)
		require.Equal(t, []uint64{1000, 1001, 1002, 1002, 1003, 1004}, n.listPages)
	})

	t.Run("the gap scan reads a last ID that opens its page", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.recordCount = 49951
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.seeded.Store(false)
		n.borUnavailable = false
		n.borLast = 49949
		n.stored[49950] = true
		n.stored[49951] = true

		cp.seedFrontier(context.Background(), time.Now())
		lowest, _, _ := cp.pending.lowest(time.Now())
		require.Equal(t, uint64(49952), lowest)
		require.Equal(t, []uint64{999, 1000}, n.listPages)
	})

	t.Run("a whole page heimdall lacks is a gap", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.recordCount = 52000
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.seeded.Store(false)
		n.borUnavailable = false
		n.borLast = 50000
		for id := uint64(50051); id <= 50100; id++ {
			n.stored[id] = true
		}

		cp.seedFrontier(context.Background(), time.Now())
		lowest, _, _ := cp.pending.lowest(time.Now())
		require.Equal(t, uint64(50001), lowest)
	})

	t.Run("a fresh chain seeds the first state sync, finally only when bor confirms it", func(t *testing.T) {
		for _, tc := range []struct {
			name           string
			borUnavailable bool
			borLast        uint64
			wantSeed       bool
			wantFinal      bool
		}{
			{name: "bor at 0", wantSeed: true, wantFinal: true},
			{name: "bor unreachable seeds provisionally", borUnavailable: true, wantSeed: true},
			{name: "bor ahead of an unreadable store", borLast: 5},
			{name: "bor at the first ID", borLast: 1},
		} {
			t.Run(tc.name, func(t *testing.T) {
				n, rest, comet := newBurstNode(t)
				cp, _ := newBurstClerkProcessor(t, n, rest, comet)
				cp.seeded.Store(false)
				n.borUnavailable = tc.borUnavailable
				n.borLast = tc.borLast

				cp.seedFrontier(context.Background(), cp.startedAt.Add(clerkWarmup))
				require.Equal(t, tc.wantFinal, cp.seeded.Load())
				lowest, _, ok := cp.pending.lowest(time.Now())
				require.Equal(t, tc.wantSeed, ok)
				if tc.wantSeed {
					require.Equal(t, uint64(1), lowest)
				}
			})
		}
	})

	t.Run("a provisional seed of 1 is placed once", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.seeded.Store(false)
		start := cp.startedAt.Add(clerkWarmup)

		cp.seedFrontier(context.Background(), start)
		n.recordCount = 1
		cp.seedFrontier(context.Background(), start.Add(clerkSeedRetryBackoff))
		lowest, _, _ := cp.pending.lowest(start.Add(clerkSeedRetryBackoff))
		require.Equal(t, uint64(2), lowest, "the first stored record moves it")
	})

	t.Run("a repeated empty store read doesn't refresh a provisional seed of 1", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.seeded.Store(false)
		start := cp.startedAt.Add(clerkWarmup)

		cp.seedFrontier(context.Background(), start)
		later := start.Add(seedLive + time.Second)
		cp.seedFrontier(context.Background(), later)
		_, _, ok := cp.pending.lowest(later)
		require.False(t, ok)
	})

	t.Run("a provisional seed from an empty store read gives way to the store's bound", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.seeded.Store(false)
		start := cp.startedAt.Add(clerkWarmup)

		cp.seedFrontier(context.Background(), start)
		lowest, _, _ := cp.pending.lowest(start)
		require.Equal(t, uint64(1), lowest)

		n.recordCount = 49991
		cp.seedFrontier(context.Background(), start.Add(clerkSeedRetryBackoff))
		lowest, _, _ = cp.pending.lowest(start.Add(clerkSeedRetryBackoff))
		require.Equal(t, uint64(49992), lowest)
		require.Len(t, cp.pending.leases.entries, 1)
		require.False(t, cp.seeded.Load(), "still provisional without bor")
	})

	t.Run("a failed record list read isn't taken for a gap", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.recordCount = 52000
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.seeded.Store(false)
		n.borUnavailable = false
		n.borLast = 49991
		n.listUnavailable = true
		n.stored[49993] = true
		start := cp.startedAt.Add(clerkWarmup)

		cp.seedFrontier(context.Background(), start)
		lowest, _, _ := cp.pending.lowest(start)
		require.Equal(t, uint64(52001), lowest)
		require.False(t, cp.seeded.Load(), "a failed scan proves nothing; a later task scans again")

		n.listUnavailable = false
		cp.seedFrontier(context.Background(), start.Add(clerkSeedRetryBackoff-time.Second))
		require.False(t, cp.seeded.Load(), "a failed attempt backs off for clerkSeedRetryBackoff")
		cp.seedFrontier(context.Background(), start.Add(clerkSeedRetryBackoff))
		lowest, _, _ = cp.pending.lowest(start.Add(clerkSeedRetryBackoff))
		require.Equal(t, uint64(49992), lowest, "the gap the failed scan missed")
		require.True(t, cp.seeded.Load())
	})

	t.Run("a bor answering the largest ID can't wrap the seed", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.recordCount = 49991
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.seeded.Store(false)
		n.borUnavailable = false
		n.borLast = math.MaxUint64

		cp.seedFrontier(context.Background(), time.Now())
		lowest, _, _ := cp.pending.lowest(time.Now())
		require.Equal(t, uint64(49992), lowest)
	})

	t.Run("bor exactly at heimdall's highest stored ID seeds the next one", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.recordCount = 49991
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.seeded.Store(false)
		n.borUnavailable = false
		n.borLast = 49991

		cp.seedFrontier(context.Background(), time.Now())
		lowest, _, _ := cp.pending.lowest(time.Now())
		require.Equal(t, uint64(49992), lowest)
		require.Empty(t, n.listPages, "no list read needed")
	})

	t.Run("the seed never goes above heimdall's highest stored ID + 1", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.recordCount = 49991
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.seeded.Store(false)
		n.borUnavailable = false
		n.borLast = 49995

		cp.seedFrontier(context.Background(), time.Now())
		lowest, _, _ := cp.pending.lowest(time.Now())
		require.Equal(t, uint64(49992), lowest)
	})

	t.Run("an unreadable record count isn't taken for an empty chain", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.recordCountUnavailable = true
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.seeded.Store(false)
		n.borUnavailable = false
		n.borLast = 0

		cp.seedFrontier(context.Background(), time.Now())
		_, _, ok := cp.pending.lowest(time.Now())
		require.False(t, ok)
		require.False(t, cp.seeded.Load())
	})

	t.Run("an unreadable record count is read again at most once per clerkStoreCheckDelay", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.recordCountUnavailable = true
		n.recordCount = 49991
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.seeded.Store(false)
		start := time.Now()

		cp.seedFrontier(context.Background(), start)
		cp.seedFrontier(context.Background(), start.Add(clerkStoreCheckDelay-time.Second))
		require.Equal(t, 1, n.countCalls)
		require.False(t, cp.seeded.Load())

		n.recordCountUnavailable = false
		cp.seedFrontier(context.Background(), start.Add(clerkStoreCheckDelay))
		require.Equal(t, 2, n.countCalls)
		lowest, _, ok := cp.pending.lowest(start.Add(clerkStoreCheckDelay))
		require.True(t, ok)
		require.Equal(t, uint64(49992), lowest)
	})

	t.Run("bor's last state sync ID is read with a timeout and checked", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)

		var deadline time.Duration
		cp.stateReceiver = func(string) (lastStateIDReader, error) {
			return fakeStateReceiver{last: big.NewInt(49991), deadline: &deadline}, nil
		}
		last, err := cp.borLastStateID(context.Background())
		require.NoError(t, err)
		require.Equal(t, uint64(49991), last)
		require.InDelta(t, 2, deadline.Seconds(), 0.5)

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		cp.stateReceiver = func(string) (lastStateIDReader, error) {
			return blockingStateReceiver{}, nil
		}
		started := time.Now()
		_, err = cp.borLastStateID(ctx)
		require.ErrorIs(t, err, context.DeadlineExceeded)
		require.Less(t, time.Since(started), borStateTimeout/2, "the bor read is bounded by the caller's ctx")

		cp.stateReceiver = func(string) (lastStateIDReader, error) {
			return fakeStateReceiver{err: errors.New("bor rpc down")}, nil
		}
		_, err = cp.borLastStateID(context.Background())
		require.ErrorContains(t, err, "bor rpc down")

		cp.stateReceiver = func(string) (lastStateIDReader, error) {
			return fakeStateReceiver{last: new(big.Int).Lsh(big.NewInt(1), 64)}, nil
		}
		_, err = cp.borLastStateID(context.Background())
		require.Error(t, err)

		cancelled, cancelNow := context.WithCancel(context.Background())
		cancelNow()
		cp.stateReceiver = func(string) (lastStateIDReader, error) {
			t.Fatal("no bor read once the seed's ctx is done")
			return nil, nil
		}
		_, err = cp.borLastStateID(cancelled)
		require.ErrorIs(t, err, context.Canceled)

		n.paramsUnavailable = true
		cp.chainParams = cachedChainParams{}
		cp.stateReceiver = func(string) (lastStateIDReader, error) {
			t.Fatal("no state receiver without chain params")
			return nil, nil
		}
		_, err = cp.borLastStateID(context.Background())
		require.Error(t, err)
	})

	t.Run("a released task clears its ID from the pending set", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.recordCount = 49999
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.seeded.Store(false)
		cp.pending.seed(49992, time.Now())

		requireAwaitingStore(t, runClerkTask(t, cp, releasedSignature(), 49992))
		require.Equal(t, uint64(50000), cp.pending.touch(50000, "", time.Now()))

		cp.startedAt = time.Now()
		cp.pending.seed(49993, time.Now())
		requireWarmingUp(t, runClerkTask(t, cp, releasedSignature(), 49993), wholeSeconds(clerkWarmup+spread(49993, clerkDeferRetryDelay)))
		require.Equal(t, uint64(50000), cp.pending.touch(50000, "", time.Now()), "also during the warm-up")
	})

	t.Run("the frontier is seeded once, right after the last stored ID", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.recordCount = 49991
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.seeded.Store(false)
		n.borUnavailable = false
		n.borLast = n.recordCount
		start := time.Now()

		cp.seedFrontier(context.Background(), start)
		lowest, _, ok := cp.pending.lowest(start)
		require.True(t, ok)
		require.Equal(t, uint64(49992), lowest)

		cp.seedFrontier(context.Background(), start.Add(clerkStoreCheckDelay))
		require.Equal(t, 1, n.countCalls)
	})

	t.Run("an unreadable record count leaves the frontier unseeded and is retried", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.seeded.Store(false)
		cp.startedAt = time.Now()
		rest.Close()

		requireWarmingUp(t, runClerkTask(t, cp, &tasks.Signature{}, 49992), clerkWarmup)
		require.False(t, cp.seeded.Load())
		_, _, ok := cp.pending.lowest(time.Now())
		require.True(t, ok)
		require.Equal(t, uint64(49992), cp.pending.touch(49992, "", time.Now()))
	})

	t.Run("a duplicate of a released task leaves the pending set", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.pending.touch(49992, "", time.Now())
		cp.released.touch(49992, burstOrigin(49992), time.Now())

		requireRetryLater(t, runClerkTask(t, cp, &tasks.Signature{}, 49992), msgReleasedOwned, clerkMaxStoreWait)
		require.Equal(t, uint64(50000), cp.pending.touch(50000, "", time.Now()))
	})

	t.Run("a copy in the mempool doesn't stand in for the rebroadcast after a failed store check", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		n.pending = []*clerktypes.MsgEventRecord{burstRecord(49992)}

		signature := releasedSignature()
		requireAwaitingStore(t, runClerkTask(t, cp, signature, 49992))
		require.Len(t, n.sequences, 1, "this bridge's own tx goes out")
		acceptedAt, ok := retryStart(signature.Headers, acceptedAtHeader)
		require.True(t, ok)
		require.WithinDuration(t, time.Now(), acceptedAt, 5*time.Second)
	})

	t.Run("only this bridge's own accepted tx starts the give-up clock", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		n.pending = []*clerktypes.MsgEventRecord{burstRecord(49992)}

		signature := &tasks.Signature{}
		requireAwaitingStore(t, runClerkTask(t, cp, signature, 49992))
		require.Empty(t, n.sequences, "a first copy in the mempool is still the same event")
		require.Empty(t, headerString(signature.Headers, firstAcceptedAtHeader))

		n.pending = nil
		signature.Headers[acceptedAtHeader] = unixAgo(clerkMaxStoreWait)
		requireAwaitingStore(t, runClerkTask(t, cp, signature, 49992))
		require.Len(t, n.sequences, 1)
		require.Equal(t, signature.Headers[acceptedAtHeader], signature.Headers[firstAcceptedAtHeader])
	})

	t.Run("a duplicate delivery giving up doesn't free an ID another delivery holds", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)

		cp.pending.touch(49992, "task-b", time.Now())
		n.broadcastResults = append(n.broadcastResults, insufficientFeeCheckTx)
		lastAttempt := &tasks.Signature{UUID: "task-a", RetryCount: 0}
		require.Error(t, runClerkTask(t, cp, lastAttempt, 49992))

		requireRetryLater(t, runClerkTask(t, cp, &tasks.Signature{UUID: "task-c"}, 49992+clerkOrderWindow), msgAheadOfPending, orderDelay(49992+clerkOrderWindow, clerkDeferRetryDelay))
	})

	t.Run("a seeded ID's task giving up also drops the seed", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.pending.seed(49992, time.Now())

		n.broadcastResults = append(n.broadcastResults, insufficientFeeCheckTx)
		require.Error(t, runClerkTask(t, cp, &tasks.Signature{UUID: "task-a", RetryCount: 0}, 49992))
		_, _, ok := cp.pending.lowest(time.Now())
		require.False(t, ok)
	})

	t.Run("a task's warm-up lease is its own", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.startedAt = time.Now()
		requireWarmingUp(t, runClerkTask(t, cp, &tasks.Signature{UUID: "task-a"}, 49992), wholeSeconds(clerkWarmup+spread(49992, clerkDeferRetryDelay)))

		cp.startedAt = time.Now().Add(-clerkWarmup)
		n.broadcastResults = append(n.broadcastResults, insufficientFeeCheckTx)
		require.Error(t, runClerkTask(t, cp, &tasks.Signature{UUID: "task-a", RetryCount: 0}, 49992))
		_, _, ok := cp.pending.lowest(time.Now())
		require.False(t, ok, "the task that gave up held the only lease")
	})

	t.Run("a duplicate of a released task takes over once that task gives up", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.released.touch(49992, burstOrigin(49992), time.Now())

		duplicate := &tasks.Signature{UUID: "replay"}
		requireRetryLater(t, runClerkTask(t, cp, duplicate, 49992), msgReleasedOwned, clerkMaxStoreWait)

		n.broadcastResults = append(n.broadcastResults, insufficientFeeCheckTx)
		require.Error(t, runClerkTask(t, cp, releasedSignature(), 49992), "the released task's last attempt")
		require.False(t, cp.released.has(49992, burstOrigin(49992), time.Now()))

		cp.pending.touch(49000, "", time.Now())
		requireAwaitingStore(t, runClerkTask(t, cp, duplicate, 49992))
		require.True(t, windowReleased(duplicate), "the takeover continues as released")
		require.Equal(t, uint64(49000), cp.pending.touch(49001, "", time.Now()), "it doesn't hold the window again")
	})

	t.Run("a record the node already holds is awaited, not retried as a failure", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		n.broadcastResults = append(n.broadcastResults, txInCacheRPC)

		signature := &tasks.Signature{RetryCount: 0}
		requireAwaitingStore(t, runClerkTask(t, cp, signature, 49992))
		require.True(t, windowReleased(signature))
		require.Equal(t, clerkTaskRetries, signature.RetryCount)

		requireAwaitingStore(t, runClerkTask(t, cp, &tasks.Signature{UUID: "next"}, 49993))
		require.Equal(t, uint64(burstCommittedSeq+1), n.sequences[len(n.sequences)-1], "the next tx signs the following sequence")
	})

	t.Run("record lookups beyond the bound wait without holding a worker", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		releases := make([]func(), 0, clerkRecordLookups)
		for range clerkRecordLookups {
			release := cp.recordLookups.tryAcquire(false)
			require.NotNil(t, release)
			releases = append(releases, release)
		}

		cp.pending.touch(49991, "lower", time.Now())
		requireRetryLater(t, runClerkTask(t, cp, &tasks.Signature{}, 49992), msgLookupsBusy, wholeSeconds(clerkDeferRetryDelay+spread(49992, clerkDeferRetryDelay)))
		require.Zero(t, n.recordLookups[49992])
		require.Empty(t, n.sequences)

		requireRetryLater(t, runClerkTask(t, cp, releasedSignature(), 49993), msgLookupsBusy, wholeSeconds(releasedRetryDelay+spread(49993, releasedRetryDelay)))
		require.Zero(t, n.recordLookups[49993], "a released store check waits on the slow cadence")

		requireAwaitingStore(t, runClerkTask(t, cp, &tasks.Signature{}, 49991))
		require.NotZero(t, n.recordLookups[49991], "the lowest ID has a slot of its own")

		releases[0]()
		requireAwaitingStore(t, runClerkTask(t, cp, &tasks.Signature{}, 49992))
		require.NotNil(t, cp.recordLookups.tryAcquire(false), "the task gave its slot back")
	})

	t.Run("a second lowest-ID lookup in flight waits instead of taking another worker", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		releasePriority := cp.recordLookups.tryAcquire(true)
		require.NotNil(t, releasePriority)

		requireRetryLater(t, runClerkTask(t, cp, &tasks.Signature{}, 49991), msgLookupsBusy, clerkDeferRetryDelay)
		require.Zero(t, n.recordLookups[49991])

		for range clerkRecordLookups {
			require.NotNil(t, cp.recordLookups.tryAcquire(false), "the shared slots stay free for the other tasks")
		}

		releasePriority()
		requireAwaitingStore(t, runClerkTask(t, cp, &tasks.Signature{}, 49991))
		require.NotZero(t, n.recordLookups[49991])
	})

	t.Run("a lookup that panics gives its slot back", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.pending.touch(49991, "lower", time.Now())
		var event statesender.StatesenderStateSynced
		event.Id = big.NewInt(49992)
		n.stored[49992] = true
		cp.cliCtx.Codec = nil

		require.Panics(t, func() { _, _, _ = cp.checkStored(context.Background(), "StateSynced", &event, nil, 49992, 49991) })
		for range clerkRecordLookups {
			require.NotNil(t, cp.recordLookups.tryAcquire(false))
		}
		require.NotNil(t, cp.recordLookups.tryAcquire(true))
	})

	t.Run("a last attempt that panics still lets go of the order window", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.recordCount = 49991
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.seeded.Store(false)
		cp.startedAt = time.Now()
		cp.stateReceiver = func(string) (lastStateIDReader, error) { panic("bor client") }

		require.Panics(t, func() { _ = runClerkTask(t, cp, &tasks.Signature{UUID: "task-a"}, 49992) })
		_, _, ok := cp.pending.lowest(time.Now())
		require.False(t, ok, "neither the task's lease nor the seed is left behind")
	})

	t.Run("a task releasing the window keeps another task's lease on its ID", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)

		signature := &tasks.Signature{UUID: "task-a"}
		cp.pending.touch(49992, "task-b", time.Now())
		requireAwaitingStore(t, runClerkTask(t, cp, signature, 49992))
		require.True(t, windowReleased(signature))

		lowest, _, ok := cp.pending.lowest(time.Now())
		require.True(t, ok, "task-b, from another publish of the ID, still holds it")
		require.Equal(t, uint64(49992), lowest)
	})

	t.Run("only one duplicate of a released task waits to take over", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.released.touch(49992, burstOrigin(49992), time.Now())

		first := &tasks.Signature{UUID: "replay-1"}
		requireRetryLater(t, runClerkTask(t, cp, first, 49992), msgReleasedOwned, clerkMaxStoreWait)
		require.NoError(t, runClerkTask(t, cp, &tasks.Signature{UUID: "replay-2"}, 49992), "a second waiter ends")
		requireRetryLater(t, runClerkTask(t, cp, first, 49992), msgReleasedOwned, clerkMaxStoreWait)
	})

	t.Run("a released task missing two slow periods no longer ends a redelivery", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		cp, logs := newBurstClerkProcessor(t, n, rest, comet)
		cp.released.touch(49992, burstOrigin(49992), time.Now().Add(-releasedLive+time.Minute))
		requireRetryLater(t, runClerkTask(t, cp, &tasks.Signature{}, 49992), msgReleasedOwned, clerkMaxStoreWait)
		require.Contains(t, logs.String(), debugMsgClerkDuplicateTask)

		cp.released.touch(49993, burstOrigin(49993), time.Now().Add(-releasedLive-time.Second))
		requireAwaitingStore(t, runClerkTask(t, cp, &tasks.Signature{}, 49993))
	})

	t.Run("an idle lowest pending ID heimdall already stored stops holding the window", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		before := testutil.ToFloat64(metrics.ClerkPendingReconciled)

		cp.pending.touch(49000, "", time.Now().Add(-clerkLowestIdle+time.Second))
		n.stored[49000] = true
		calls := n.restCalls
		requireRetryLater(t, runClerkTask(t, cp, &tasks.Signature{}, 49992), msgAheadOfPending, orderDelay(49992, 9*clerkDeferRetryDelay))
		require.Equal(t, calls, n.restCalls, "a lowest ID whose task still runs isn't looked up")

		cp.pending.touch(49000, "", time.Now().Add(-clerkLowestIdle))
		requireAwaitingStore(t, runClerkTask(t, cp, &tasks.Signature{}, 49000+clerkOrderWindow))
		require.Equal(t, 1, n.recordLookups[49000])
		require.Equal(t, before+1, testutil.ToFloat64(metrics.ClerkPendingReconciled))
		require.Equal(t, uint64(49992), cp.pending.touch(49992, "", time.Now()), "the broadcast ID left the window too")
	})

	t.Run("a released task ranks against a lower released ID before the pending ones", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.mempoolTotal = clerkMempoolBusyTxs + 200
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.released.touch(49900, burstOrigin(49900), time.Now())
		cp.pending.touch(49992+100, "", time.Now())

		requireRetryLater(t, runClerkTask(t, cp, releasedSignature(), 49992), msgMempoolBusy, deferDelay(49992, 49900, true))
		require.Empty(t, n.sequences)
	})

	t.Run("a task ending without an error on its last attempt keeps its released entry", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		cp, logs := newBurstClerkProcessor(t, n, rest, comet)
		before := testutil.ToFloat64(metrics.ClerkTasksExhausted)
		cp.released.touch(49992, burstOrigin(49992), time.Now())

		requireRetryLater(t, runClerkTask(t, cp, &tasks.Signature{RetryCount: 0}, 49992), msgReleasedOwned, clerkMaxStoreWait)
		require.True(t, cp.released.has(49992, burstOrigin(49992), time.Now()))

		n.stored[49993] = true
		cp.pending.touch(49993, "", time.Now())
		require.NoError(t, runClerkTask(t, cp, &tasks.Signature{RetryCount: 0}, 49993))
		require.Equal(t, before, testutil.ToFloat64(metrics.ClerkTasksExhausted))
		require.NotContains(t, logs.String(), errMsgClerkTaskExhausted)
	})

	t.Run("a stored idle lowest inside the window stops skewing the busy gate", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		n.mempoolTotal = clerkMempoolBusyTxs
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.pending.touch(49950, "", time.Now().Add(-clerkLowestIdle))
		n.stored[49950] = true

		requireAwaitingStore(t, runClerkTask(t, cp, &tasks.Signature{}, 49992))
		require.Equal(t, 1, n.recordLookups[49950])
	})

	t.Run("the lowest pending task itself doesn't look anything up", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		cp.pending.touch(49992, "", time.Now().Add(-clerkLowestIdle))

		requireAwaitingStore(t, runClerkTask(t, cp, &tasks.Signature{}, 49992))
		require.Equal(t, 1, n.recordLookups[49992], "only its own stored check")
	})

	t.Run("the idle check and the lookup are on the same ID when the lowest moved", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		start := time.Now()
		n.stored[49000] = true

		cp.pending.touch(49000, "", start)
		cp.pending.touch(49992, "", start.Add(clerkLowestIdle))
		// 48000 was the lowest when the caller looked; another task removed it since.
		require.Equal(t, uint64(49992), cp.dropStoredLowest(context.Background(), 48000, start.Add(clerkLowestIdle)))
		require.Equal(t, 1, n.recordLookups[49000])
		require.Zero(t, n.recordLookups[48000])
	})

	t.Run("an idle unstored lowest is returned even if the caller saw an older one", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		start := time.Now()

		cp.pending.touch(49000, "", start)
		require.Equal(t, uint64(49000), cp.dropStoredLowest(context.Background(), 48000, start.Add(clerkLowestIdle)))
	})

	t.Run("the lowest pending ID counts as idle from exactly clerkLowestIdle", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)
		start := time.Now()
		n.stored[49000] = true

		cp.pending.touch(49000, "", start)
		cp.pending.touch(49992, "", start.Add(clerkLowestIdle))
		require.Equal(t, uint64(49000), cp.dropStoredLowest(context.Background(), 49000, start.Add(clerkLowestIdle-time.Nanosecond)))
		require.Equal(t, uint64(49992), cp.dropStoredLowest(context.Background(), 49000, start.Add(clerkLowestIdle)))
	})

	t.Run("a task for the same ID from another log of the same L1 tx is not a released duplicate", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		cp, logs := newBurstClerkProcessor(t, n, rest, comet)
		cp.released.touch(49992, burstOrigin(49992), time.Now())

		var vLog types.Log
		require.NoError(t, json.Unmarshal([]byte(stateSyncedLog(t, 49992)), &vLog))
		vLog.Index = 3
		other, err := json.Marshal(vLog)
		require.NoError(t, err)

		requireAwaitingStore(t, runClerkTaskWithLog(t, cp, &tasks.Signature{}, string(other)))
		require.NotContains(t, logs.String(), debugMsgClerkDuplicateTask)
		requireRetryLater(t, runClerkTask(t, cp, &tasks.Signature{}, 49992), msgReleasedOwned, clerkMaxStoreWait)
	})

	t.Run("an idle lowest pending ID still unstored keeps holding the window", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)

		cp.pending.touch(49000, "", time.Now().Add(-clerkLowestIdle))
		requireRetryLater(t, runClerkTask(t, cp, &tasks.Signature{}, 49992), msgAheadOfPending, orderDelay(49992, 9*clerkDeferRetryDelay))
		calls := n.restCalls
		requireRetryLater(t, runClerkTask(t, cp, &tasks.Signature{}, 49993), msgAheadOfPending, orderDelay(49993, 9*clerkDeferRetryDelay))
		require.Equal(t, calls, n.restCalls, "the lookup runs at most once per clerkStoreCheckDelay")
	})

	t.Run("a task for the same ID from another L1 log is not a released duplicate", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		cp, logs := newBurstClerkProcessor(t, n, rest, comet)
		cp.released.touch(49992, "0xother:7", time.Now())

		requireAwaitingStore(t, runClerkTask(t, cp, &tasks.Signature{}, 49992))
		require.NotContains(t, logs.String(), debugMsgClerkDuplicateTask)
	})

	t.Run("mempool txs that aren't a clerk record are skipped", func(t *testing.T) {
		n, rest, comet := newBurstNode(t)
		cp, _ := newBurstClerkProcessor(t, n, rest, comet)

		empty, err := n.txc.TxEncoder()(n.txc.NewTxBuilder().GetTx())
		require.NoError(t, err)
		builder := n.txc.NewTxBuilder()
		require.NoError(t, builder.SetMsgs(burstRecord(49992)))
		record, err := n.txc.TxEncoder()(builder.GetTx())
		require.NoError(t, err)

		require.Nil(t, cp.decodeClerkMsg([]byte("not a tx")))
		require.Nil(t, cp.decodeClerkMsg(empty))
		got := cp.decodeClerkMsg(record)
		require.NotNil(t, got)
		require.True(t, sameEventRecord(got, burstRecord(49992)))
	})
}
