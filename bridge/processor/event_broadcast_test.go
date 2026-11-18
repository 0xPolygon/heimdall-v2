package processor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"github.com/RichardKnop/machinery/v1/tasks"
	cmtjson "github.com/cometbft/cometbft/libs/json"
	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	cmttypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/contracts/statesender"
	"github.com/0xPolygon/heimdall-v2/helper"
	chaintypes "github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
	checkpointtypes "github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
	clerktypes "github.com/0xPolygon/heimdall-v2/x/clerk/types"
	staketypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
	topuptypes "github.com/0xPolygon/heimdall-v2/x/topup/types"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
)

type broadcastResult struct {
	response *sdk.TxResponse
	err      error
	calls    int
}

func (b *broadcastResult) BroadcastToHeimdall(context.Context, sdk.Msg, interface{}) (*sdk.TxResponse, error) {
	b.calls++
	return b.response, b.err
}

func mempoolProcessor(t *testing.T, cdc codec.Codec, txs ...cmttypes.Tx) BaseProcessor {
	t.Helper()
	result, err := cmtjson.Marshal(&coretypes.ResultUnconfirmedTxs{Count: len(txs), Total: len(txs), Txs: txs})
	require.NoError(t, err)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			ID json.RawMessage `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(struct {
			Version string          `json:"jsonrpc"`
			ID      json.RawMessage `json:"id"`
			Result  json.RawMessage `json:"result"`
		}{"2.0", request.ID, result}); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	rpc, err := rpchttp.New(server.URL, "/websocket")
	require.NoError(t, err)
	return BaseProcessor{Logger: log.NewNopLogger(), cliCtx: client.Context{}.WithCodec(cdc), httpClient: rpc}
}

func eventTestCodec() codec.Codec {
	registry := codectypes.NewInterfaceRegistry()
	sdk.RegisterInterfaces(registry)
	clerktypes.RegisterInterfaces(registry)
	staketypes.RegisterInterfaces(registry)
	topuptypes.RegisterInterfaces(registry)
	checkpointtypes.RegisterInterfaces(registry)
	banktypes.RegisterInterfaces(registry)
	return codec.NewProtoCodec(registry)
}

func encodeEventTx(t *testing.T, cdc codec.Codec, msgs ...sdk.Msg) cmttypes.Tx {
	t.Helper()
	config := authtx.NewTxConfig(cdc, authtx.DefaultSignModes)
	builder := config.NewTxBuilder()
	require.NoError(t, builder.SetMsgs(msgs...))
	encoded, err := config.TxEncoder()(builder.GetTx())
	require.NoError(t, err)
	return encoded
}

func TestMempoolRetainsPendingEvent(t *testing.T) {
	cdc := eventTestCodec()
	tests := []struct {
		relayed sdk.Msg
		pending sdk.Msg
		spoofed sdk.Msg
	}{
		{
			&clerktypes.MsgEventRecord{From: "relayer-a", TxHash: "0x01", LogIndex: 7, Data: []byte{1}},
			&clerktypes.MsgEventRecord{From: "relayer-b", TxHash: "0x01", LogIndex: 7, Data: []byte{1}},
			&clerktypes.MsgEventRecord{From: "relayer-b", TxHash: "0x01", LogIndex: 7, Data: []byte{2}},
		},
		{
			&staketypes.MsgValidatorJoin{From: "relayer-a", ValId: 1, Amount: math.NewInt(10), TxHash: []byte{1}, LogIndex: 7},
			&staketypes.MsgValidatorJoin{From: "relayer-b", ValId: 1, Amount: math.NewInt(10), TxHash: []byte{1}, LogIndex: 7},
			&staketypes.MsgValidatorJoin{From: "relayer-b", ValId: 1, Amount: math.NewInt(20), TxHash: []byte{1}, LogIndex: 7},
		},
		{
			&staketypes.MsgStakeUpdate{From: "relayer-a", ValId: 1, NewAmount: math.NewInt(10), TxHash: []byte{1}, LogIndex: 7},
			&staketypes.MsgStakeUpdate{From: "relayer-b", ValId: 1, NewAmount: math.NewInt(10), TxHash: []byte{1}, LogIndex: 7},
			&staketypes.MsgStakeUpdate{From: "relayer-b", ValId: 1, NewAmount: math.NewInt(20), TxHash: []byte{1}, LogIndex: 7},
		},
		{
			&staketypes.MsgSignerUpdate{From: "relayer-a", ValId: 1, NewSignerPubKey: []byte{4, 1}, TxHash: []byte{1}, LogIndex: 7},
			&staketypes.MsgSignerUpdate{From: "relayer-b", ValId: 1, NewSignerPubKey: []byte{4, 1}, TxHash: []byte{1}, LogIndex: 7},
			&staketypes.MsgSignerUpdate{From: "relayer-b", ValId: 1, NewSignerPubKey: []byte{4, 2}, TxHash: []byte{1}, LogIndex: 7},
		},
		{
			&staketypes.MsgValidatorExit{From: "relayer-a", ValId: 1, DeactivationEpoch: 3, TxHash: []byte{1}, LogIndex: 7},
			&staketypes.MsgValidatorExit{From: "relayer-b", ValId: 1, DeactivationEpoch: 3, TxHash: []byte{1}, LogIndex: 7},
			&staketypes.MsgValidatorExit{From: "relayer-b", ValId: 1, DeactivationEpoch: 4, TxHash: []byte{1}, LogIndex: 7},
		},
		{
			&topuptypes.MsgTopupTx{Proposer: "relayer-a", User: "user-1", Fee: math.NewInt(10), TxHash: []byte{1}, LogIndex: 7},
			&topuptypes.MsgTopupTx{Proposer: "relayer-b", User: "user-1", Fee: math.NewInt(10), TxHash: []byte{1}, LogIndex: 7},
			&topuptypes.MsgTopupTx{Proposer: "relayer-b", User: "user-1", Fee: math.NewInt(20), TxHash: []byte{1}, LogIndex: 7},
		},
		{
			&checkpointtypes.MsgCpAck{From: "relayer-a", Number: 7, EndBlock: 256, RootHash: []byte{1}},
			&checkpointtypes.MsgCpAck{From: "relayer-b", Number: 7, EndBlock: 256, RootHash: []byte{1}},
			&checkpointtypes.MsgCpAck{From: "relayer-b", Number: 7, EndBlock: 256, RootHash: []byte{2}},
		},
	}
	for _, tc := range tests {
		t.Run(sdk.MsgTypeURL(tc.relayed), func(t *testing.T) {
			pending := encodeEventTx(t, cdc, &banktypes.MsgSend{}, tc.pending)
			bp := mempoolProcessor(t, cdc, []byte{0xff}, encodeEventTx(t, cdc, tc.spoofed), pending)
			found, err := bp.checkTxAgainstMempool(context.Background(), tc.relayed, nil)
			require.NoError(t, err)
			require.True(t, found)
			bp = mempoolProcessor(t, cdc, encodeEventTx(t, cdc, &banktypes.MsgSend{}, tc.spoofed))
			found, err = bp.checkTxAgainstMempool(context.Background(), tc.relayed, nil)
			require.NoError(t, err)
			require.False(t, found)
			var foreign sdk.Msg = &clerktypes.MsgEventRecord{TxHash: "0x01", LogIndex: 7, Data: []byte{1}}
			if _, ok := tc.relayed.(*clerktypes.MsgEventRecord); ok {
				foreign = &staketypes.MsgStakeUpdate{ValId: 1, NewAmount: math.NewInt(10), TxHash: []byte{1}, LogIndex: 7}
			}
			bp = mempoolProcessor(t, cdc, encodeEventTx(t, cdc, foreign))
			found, err = bp.checkTxAgainstMempool(context.Background(), tc.relayed, nil)
			require.NoError(t, err)
			require.False(t, found)
		})
	}
}

func TestStakingBroadcastAwaitsApplication(t *testing.T) {
	underlying := errors.New("temporary broadcast failure")
	tests := []struct {
		name     string
		response *sdk.TxResponse
		err      error
		pending  bool
	}{
		{"accepted", &sdk.TxResponse{}, nil, true},
		{"nil response", nil, nil, false},
		{"checktx rejection", &sdk.TxResponse{Code: 5}, nil, false},
		{"broadcast failure", nil, underlying, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			broadcaster := &broadcastResult{response: tc.response, err: tc.err}
			sp := &StakingProcessor{BaseProcessor: BaseProcessor{Logger: log.NewNopLogger(), txBroadcaster: broadcaster}}
			err := sp.broadcastEvent(context.Background(), &staketypes.MsgStakeUpdate{TxHash: []byte{1}, LogIndex: 7}, nil, "waiting")
			require.Error(t, err)
			require.Equal(t, 1, broadcaster.calls)
			var later tasks.ErrRetryTaskLater
			require.Equal(t, tc.pending, errors.As(err, &later))
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			}
		})
	}
}

func TestStakingBroadcastDoesNotDuplicateMempoolEvent(t *testing.T) {
	cdc := eventTestCodec()
	msg := &staketypes.MsgStakeUpdate{TxHash: []byte{1}, LogIndex: 7}
	bp := mempoolProcessor(t, cdc, encodeEventTx(t, cdc, msg))
	broadcaster := new(broadcastResult)
	bp.txBroadcaster = broadcaster
	sp := &StakingProcessor{BaseProcessor: bp}
	var later tasks.ErrRetryTaskLater
	require.ErrorAs(t, sp.broadcastEvent(context.Background(), msg, nil, "waiting"), &later)
	require.Zero(t, broadcaster.calls)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := sp.broadcastEvent(ctx, msg, nil, "waiting")
	require.Error(t, err)
	require.NotErrorAs(t, err, &later)
	require.Zero(t, broadcaster.calls)
}

func TestClerkBroadcastRetriesUntilRecordExists(t *testing.T) {
	useLiveEvents(t)
	cdc := eventTestCodec()
	params, err := cdc.MarshalJSON(&chaintypes.QueryParamsResponse{Params: chaintypes.DefaultParams()})
	require.NoError(t, err)
	var applied atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasPrefix(r.URL.Path, "/chainmanager/") {
			_, _ = w.Write(params)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/clerk/is-old-tx") {
			_, _ = fmt.Fprintf(w, `{"is_old":%t}`, applied.Load())
			return
		}
		if !applied.Load() {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		id := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
		_, _ = fmt.Fprintf(w, `{"record":{"id":%q}}`, id)
	}))
	t.Cleanup(server.Close)
	oldClient := helper.Client
	helper.Client = server.Client()
	t.Cleanup(func() { helper.Client = oldClient })
	previousKey := helper.GetPrivKey()
	helper.SetTestPrivPubKey(secp256k1.GenPrivKey())
	if len(previousKey) > 0 {
		t.Cleanup(func() { helper.SetTestPrivPubKey(previousKey) })
	}
	cfg := helper.CustomAppConfig{Config: *serverconfig.DefaultConfig(), Custom: helper.GetDefaultHeimdallConfig()}
	cfg.Config.API.Address = server.URL
	helper.SetTestConfig(cfg)
	stateABI, err := statesender.StatesenderMetaData.GetAbi()
	require.NoError(t, err)
	cp := NewClerkProcessor(stateABI)
	cp.Logger = log.NewNopLogger()
	cp.cliCtx = client.Context{}.WithCodec(cdc)
	broadcaster := &broadcastResult{response: &sdk.TxResponse{}}
	cp.txBroadcaster = broadcaster
	payload, err := prepareDummyLogBytes()
	require.NoError(t, err)
	signature := &tasks.Signature{RetryCount: 0}
	task, err := tasks.NewWithSignature(cp.sendStateSyncedToHeimdall, signature)
	require.NoError(t, err)
	var later tasks.ErrRetryTaskLater
	for range 2 {
		require.ErrorAs(t, cp.sendStateSyncedToHeimdall(task.Context, "StateSynced", payload.String()), &later)
	}
	require.Equal(t, 1, broadcaster.calls, "resent only after rebroadcastDelay")
	applied.Store(true)
	require.NoError(t, cp.sendStateSyncedToHeimdall(task.Context, "StateSynced", payload.String()))
	require.Equal(t, 1, broadcaster.calls)
}
