package processor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"cosmossdk.io/log"
	"github.com/RichardKnop/machinery/v1/tasks"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	cmtjson "github.com/cometbft/cometbft/libs/json"
	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/golang/mock/gomock"
	promtestutil "github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/bridge/util"
	"github.com/0xPolygon/heimdall-v2/contracts/rootchain"
	"github.com/0xPolygon/heimdall-v2/contracts/stakinginfo"
	"github.com/0xPolygon/heimdall-v2/contracts/statesender"
	"github.com/0xPolygon/heimdall-v2/helper"
	helperMocks "github.com/0xPolygon/heimdall-v2/helper/mocks"
	"github.com/0xPolygon/heimdall-v2/metrics"
	chaintypes "github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
	checkpointtypes "github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
	clerktypes "github.com/0xPolygon/heimdall-v2/x/clerk/types"
	staketypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

type eventTaskFixture struct {
	sp        *StakingProcessor
	cp        *ClerkProcessor
	fp        *FeeProcessor
	chp       *CheckpointProcessor
	applied   atomic.Bool
	broadcast *broadcastResult
	url       string
	// status forces an HTTP status on every API path with the given prefix.
	status   map[string]int
	ackCount uint64
}

func newEventTaskFixture(t *testing.T) *eventTaskFixture {
	t.Helper()
	registry := codectypes.NewInterfaceRegistry()
	sdk.RegisterInterfaces(registry)
	cryptocodec.RegisterInterfaces(registry)
	authtypes.RegisterInterfaces(registry)
	staketypes.RegisterInterfaces(registry)
	clerktypes.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)
	fixture := &eventTaskFixture{broadcast: &broadcastResult{response: &sdk.TxResponse{}}, status: map[string]int{}}
	params, err := cdc.MarshalJSON(&chaintypes.QueryParamsResponse{Params: chaintypes.DefaultParams()})
	require.NoError(t, err)
	checkpointParams, err := cdc.MarshalJSON(&checkpointtypes.QueryParamsResponse{Params: checkpointtypes.DefaultParams()})
	require.NoError(t, err)
	account, err := codectypes.NewAnyWithValue(&authtypes.BaseAccount{Address: "0x0000000000000000000000000000000000000001"})
	require.NoError(t, err)
	encoded, err := cdc.Marshal(&authtypes.QueryAccountResponse{Account: account})
	require.NoError(t, err)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			var req struct {
				ID     json.RawMessage `json:"id"`
				Method string          `json:"method"`
			}
			if json.NewDecoder(r.Body).Decode(&req) != nil {
				http.Error(w, "invalid RPC", 400)
				return
			}
			if req.Method != "abci_query" {
				http.Error(w, "mempool unavailable", 503)
				return
			}
			result, err := cmtjson.Marshal(&coretypes.ResultABCIQuery{Response: abci.ResponseQuery{Value: encoded}})
			if err != nil {
				t.Error(err)
				return
			}
			_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":%s}`, req.ID, result)
			return
		}
		for prefix, code := range fixture.status {
			if strings.HasPrefix(r.URL.Path, prefix) {
				w.WriteHeader(code)
				return
			}
		}
		switch {
		case strings.HasPrefix(r.URL.Path, "/clerk/is-old-tx") && !fixture.applied.Load():
			w.WriteHeader(http.StatusNotFound)
		case strings.HasPrefix(r.URL.Path, "/stake/is-old-tx"), strings.HasPrefix(r.URL.Path, "/topup/is-old-tx"),
			strings.HasPrefix(r.URL.Path, "/clerk/is-old-tx"):
			_, _ = fmt.Fprintf(w, `{"is_old":%t}`, fixture.applied.Load())
		case strings.HasPrefix(r.URL.Path, "/checkpoints/params"):
			_, _ = w.Write(checkpointParams)
		case strings.HasPrefix(r.URL.Path, "/checkpoints/count"):
			_, _ = fmt.Fprintf(w, `{"ack_count":"%d"}`, fixture.ackCount)
		case strings.HasPrefix(r.URL.Path, "/checkpoints/latest"):
			end := 0
			if fixture.applied.Load() {
				end = ackedEnd
			}
			_, _ = fmt.Fprintf(w, `{"checkpoint":{"end_block":"%d"}}`, end)
		case strings.HasPrefix(r.URL.Path, "/stake/validator/"):
			_, _ = w.Write([]byte(`{"validator":{"val_id":"10","nonce":"4"}}`))
		case strings.HasPrefix(r.URL.Path, "/chainmanager/"):
			_, _ = w.Write(params)
		case fixture.applied.Load():
			id := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
			_, _ = fmt.Fprintf(w, `{"record":{"id":%q}}`, id)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	previous := helper.Client
	helper.Client = server.Client()
	t.Cleanup(func() { helper.Client = previous })
	key := helper.GetPrivKey()
	helper.SetTestPrivPubKey(secp256k1.GenPrivKey())
	if len(key) > 0 {
		t.Cleanup(func() { helper.SetTestPrivPubKey(key) })
	}
	cfg := helper.CustomAppConfig{Config: *serverconfig.DefaultConfig(), Custom: helper.GetDefaultHeimdallConfig()}
	cfg.Config.API.Address, cfg.Custom.CometBFTRPCUrl = server.URL, server.URL
	helper.SetTestConfig(cfg)
	stakingABI, err := stakinginfo.StakinginfoMetaData.GetAbi()
	require.NoError(t, err)
	stateABI, err := statesender.StatesenderMetaData.GetAbi()
	require.NoError(t, err)
	rootChainABI, err := rootchain.RootchainMetaData.GetAbi()
	require.NoError(t, err)
	fixture.sp, fixture.cp = NewStakingProcessor(stakingABI), NewClerkProcessor(stateABI)
	fixture.fp, fixture.chp = NewFeeProcessor(stakingABI), NewCheckpointProcessor(rootChainABI)
	for _, processor := range []*BaseProcessor{&fixture.sp.BaseProcessor, &fixture.cp.BaseProcessor, &fixture.fp.BaseProcessor, &fixture.chp.BaseProcessor} {
		processor.Logger = log.NewNopLogger()
		processor.cliCtx = client.Context{}.WithCodec(cdc).WithInterfaceRegistry(registry)
		processor.txBroadcaster = fixture.broadcast
	}
	fixture.url = server.URL
	return fixture
}

// A staking task stays alive after broadcasting until heimdall reports the event applied, polling at
// the stake interval however long it has already waited.
func TestStakeTasksWaitAfterBroadcast(t *testing.T) {
	for _, event := range []string{"Staked", "StakeUpdate", "SignerChange", "UnstakeInit"} {
		t.Run(event, func(t *testing.T) {
			useLiveEvents(t)
			f := newEventTaskFixture(t)
			run := map[string]func(context.Context, string, string) error{
				"Staked": f.sp.sendValidatorJoinToHeimdall, "StakeUpdate": f.sp.sendStakeUpdateToHeimdall,
				"SignerChange": f.sp.sendSignerChangeToHeimdall, "UnstakeInit": f.sp.sendUnstakeInitToHeimdall,
			}[event]
			payload := stakeEventLog(t, f.sp.stakingInfoAbi, event, 10, 5)
			signature := &tasks.Signature{Headers: tasks.Headers{eventRetryStartHeader: strconv.FormatInt(time.Now().Add(-time.Hour).Unix(), 10)}}
			task, err := tasks.NewWithSignature(run, signature)
			require.NoError(t, err)
			var later tasks.ErrRetryTaskLater
			require.ErrorAs(t, run(task.Context, event, payload), &later)
			require.Equal(t, util.StakeNonceRetryDelay, later.RetryIn())
			require.Equal(t, 1, f.broadcast.calls)
			f.applied.Store(true)
			require.NoError(t, run(context.Background(), event, payload))
			require.Equal(t, 1, f.broadcast.calls)
		})
	}
}

// A topup or checkpoint ACK task stays alive after broadcasting until heimdall shows it applied.
func TestTopupAndCheckpointAckTasksWaitAfterBroadcast(t *testing.T) {
	for _, event := range []string{"TopUpFee", "NewHeaderBlock"} {
		t.Run(event, func(t *testing.T) {
			f := newEventTaskFixture(t)
			run, payload := f.fp.sendTopUpFeeToHeimdall, stakeEventLog(t, f.fp.stakingInfoAbi, event, 10, 5)
			if event == "NewHeaderBlock" {
				run, payload = f.chp.sendCheckpointAckToHeimdall, checkpointAckLog(t, f.chp.rootChainAbi)
			}
			var later tasks.ErrRetryTaskLater
			require.ErrorAs(t, run(context.Background(), event, payload), &later)
			require.Equal(t, 1, f.broadcast.calls)
			f.applied.Store(true)
			require.NoError(t, run(context.Background(), event, payload))
			require.Equal(t, 1, f.broadcast.calls)
		})
	}
}

// A checkpoint ACK task retries rather than failing while heimdall's API is unreachable.
func TestCheckpointAckRetriesWhileAPIUnavailable(t *testing.T) {
	f := newEventTaskFixture(t)
	client := helperMocks.NewMockHTTPClient(gomock.NewController(t))
	client.EXPECT().Get(gomock.Any()).AnyTimes().Return(nil, errors.New("api unavailable"))
	helper.Client = client
	var later tasks.ErrRetryTaskLater
	require.ErrorAs(t, f.chp.sendCheckpointAckToHeimdall(context.Background(), "NewHeaderBlock", checkpointAckLog(t, f.chp.rootChainAbi)), &later)
	require.Zero(t, f.broadcast.calls)
}

// ackedEnd is the end block of the checkpoint checkpointAckLog acknowledges.
const ackedEnd = 255

func checkpointAckLog(t *testing.T, rootChainABI *abi.ABI) string {
	t.Helper()
	event := rootChainABI.Events["NewHeaderBlock"]
	data, err := event.Inputs.NonIndexed().Pack(big.NewInt(0), big.NewInt(ackedEnd), [32]byte{1})
	require.NoError(t, err)
	encoded, err := json.Marshal(ethtypes.Log{
		Topics: []common.Hash{event.ID, common.BytesToHash([]byte{1}), common.BigToHash(big.NewInt(10000)), common.BigToHash(big.NewInt(1))},
		Data:   data,
	})
	require.NoError(t, err)
	return string(encoded)
}

// A log that does not decode fails the task outright instead of retrying.
func TestEventTasksRejectInvalidABI(t *testing.T) {
	f := newEventTaskFixture(t)
	state, err := prepareDummyLogBytes()
	require.NoError(t, err)
	runs := map[string]func(context.Context, string, string) error{
		"Staked": f.sp.sendValidatorJoinToHeimdall, "StakeUpdate": f.sp.sendStakeUpdateToHeimdall,
		"SignerChange": f.sp.sendSignerChangeToHeimdall, "UnstakeInit": f.sp.sendUnstakeInitToHeimdall,
		"StateSynced": f.cp.sendStateSyncedToHeimdall,
	}
	for event, run := range runs {
		t.Run(event, func(t *testing.T) {
			payload := state.String()
			if event != "StateSynced" {
				payload = stakeEventLog(t, f.sp.stakingInfoAbi, event, 10, 5)
			}
			var eventLog ethtypes.Log
			require.NoError(t, json.Unmarshal([]byte(payload), &eventLog))
			eventLog.Topics = eventLog.Topics[:1]
			encoded, err := json.Marshal(eventLog)
			require.NoError(t, err)
			err = run(context.Background(), event, string(encoded))
			require.Error(t, err)
			var later tasks.ErrRetryTaskLater
			require.NotErrorAs(t, err, &later)
			require.Zero(t, f.broadcast.calls)
		})
	}
}

// An unreachable mempool retries the task rather than broadcasting or completing it.
func TestClerkMempoolOutageDoesNotCompleteTask(t *testing.T) {
	f := newEventTaskFixture(t)
	rpc, err := rpchttp.New(f.url, "/websocket")
	require.NoError(t, err)
	f.cp.httpClient = rpc
	payload, err := prepareDummyLogBytes()
	require.NoError(t, err)
	var later tasks.ErrRetryTaskLater
	require.ErrorAs(t, f.cp.sendStateSyncedToHeimdall(context.Background(), "StateSynced", payload.String()), &later)
	require.Zero(t, f.broadcast.calls)
}

// Messages without an L1 event identity never match a mempool entry.
func TestMempoolIgnoresUnsupportedMessages(t *testing.T) {
	cdc := eventTestCodec()
	msg := &banktypes.MsgSend{}
	bp := mempoolProcessor(t, cdc, encodeEventTx(t, cdc, msg))
	found, err := bp.checkTxAgainstMempool(context.Background(), msg, nil)
	require.NoError(t, err)
	require.False(t, found)
}

// Backoff grows with the time waited, survives a malformed start header and stays capped.
func TestEventRetryKeepsLongLivedBackoff(t *testing.T) {
	for _, start := range []string{"0", strconv.FormatInt(time.Now().Add(-time.Hour).Unix(), 10)} {
		signature := &tasks.Signature{Headers: tasks.Headers{eventRetryStartHeader: start}}
		task, err := tasks.NewWithSignature(func(context.Context) error { return nil }, signature)
		require.NoError(t, err)
		var later tasks.ErrRetryTaskLater
		require.ErrorAs(t, retryEvent(task.Context, "pending", util.RetryStateSyncTaskDelay), &later)
		if start == "0" {
			require.NotEqual(t, "0", signature.Headers[eventRetryStartHeader])
			require.Equal(t, util.RetryStateSyncTaskDelay, later.RetryIn())
		} else {
			require.Equal(t, 16*util.RetryStateSyncTaskDelay, later.RetryIn())
		}
	}
}

// Only retries for events still unapplied after the self-heal depth count as stuck.
func TestEventRetryCountsStuckEvents(t *testing.T) {
	for _, waited := range []time.Duration{time.Hour, helper.DefaultSHMaxDepthDuration + time.Hour} {
		signature := &tasks.Signature{Headers: tasks.Headers{eventRetryStartHeader: strconv.FormatInt(time.Now().Add(-waited).Unix(), 10)}}
		task, err := tasks.NewWithSignature(func(context.Context) error { return nil }, signature)
		require.NoError(t, err)
		before := promtestutil.ToFloat64(metrics.EventRetriesPastSelfHealDepth)
		require.Error(t, retryEvent(task.Context, "pending", util.RetryStateSyncTaskDelay))
		stuck := waited > helper.DefaultSHMaxDepthDuration
		require.Equal(t, before+map[bool]float64{true: 1}[stuck], promtestutil.ToFloat64(metrics.EventRetriesPastSelfHealDepth))
	}
}
