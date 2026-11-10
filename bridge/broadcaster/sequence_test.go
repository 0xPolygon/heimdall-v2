package broadcaster

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"cosmossdk.io/log"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtjson "github.com/cometbft/cometbft/libs/json"
	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	ctypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/helper"
	clerktypes "github.com/0xPolygon/heimdall-v2/x/clerk/types"
)

func TestExpectedSequence(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		rawLog string
		want   uint64
		wantOK bool
	}{
		{name: "ante handler log", rawLog: "account sequence mismatch, expected 309494, got 309482: incorrect account sequence", want: 309494, wantOK: true},
		{name: "wrapped log", rawLog: "foo: account sequence mismatch, expected 7, got 3", want: 7, wantOK: true},
		{name: "unrelated log", rawLog: "insufficient fees", wantOK: false},
		{name: "empty log", rawLog: "", wantOK: false},
		{name: "overflowing sequence", rawLog: "account sequence mismatch, expected 99999999999999999999, got 1", wantOK: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, ok := expectedSequence(tc.rawLog)
			require.Equal(t, tc.wantOK, ok)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestHandleNonOKResponseKeepsPendingSequence(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		response    *sdk.TxResponse
		wantSeq     uint64
		wantMempool bool
	}{
		{
			name:        "mempool full keeps the unused sequence",
			response:    &sdk.TxResponse{Code: sdkerrors.ErrMempoolIsFull.ABCICode(), Codespace: sdkerrors.RootCodespace},
			wantSeq:     120,
			wantMempool: true,
		},
		{
			name: "sequence mismatch resyncs to the expected sequence",
			response: &sdk.TxResponse{
				Code:      sdkerrors.ErrWrongSequence.ABCICode(),
				Codespace: sdkerrors.RootCodespace,
				RawLog:    "account sequence mismatch, expected 127, got 120: incorrect account sequence",
			},
			wantSeq: 127,
		},
		{
			name: "a module's own code 32 isn't a sequence mismatch",
			response: &sdk.TxResponse{
				Code:      sdkerrors.ErrWrongSequence.ABCICode(),
				Codespace: "clerk",
				RawLog:    "account sequence mismatch, expected 127, got 120: incorrect account sequence",
			},
			wantSeq: 120,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tb := &TxBroadcaster{logger: log.NewNopLogger(), lastSeqNo: 120}

			err := tb.handleNonOKResponse(context.Background(), tc.response)

			require.ErrorContains(t, err, "broadcast succeeded but received non-ok response code")
			require.Equal(t, tc.wantMempool, errors.Is(err, ErrMempoolFull))
			require.Equal(t, tc.wantSeq, tb.lastSeqNo)
		})
	}
}

// cometAccountServer answers CometBFT abci_query calls for the auth Account
// query with seq, or with a failing ABCI code when seq is nil.
func cometAccountServer(t *testing.T, cdc codec.Codec, seq *uint64) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID json.RawMessage `json:"id"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))

		query := abci.ResponseQuery{Code: 1, Log: "account query failed"}
		if seq != nil {
			acc := authtypes.NewBaseAccount(heimdallAddressBytes, cosmosPrivKey.PubKey(), 1, *seq)
			anyAcc, err := codectypes.NewAnyWithValue(acc)
			require.NoError(t, err)
			value, err := cdc.Marshal(&authtypes.QueryAccountResponse{Account: anyAcc})
			require.NoError(t, err)
			query = abci.ResponseQuery{Value: value, Height: 1}
		}

		result, err := cmtjson.Marshal(ctypes.ResultABCIQuery{Response: query})
		require.NoError(t, err)
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":%s}`, req.ID, result)
	}))
}

func TestHandleNonOKResponseFallsBackToCommittedSequence(t *testing.T) {
	// Not parallel: mutates global helper config and keys, which later tests in
	// the package must not rely on.
	registry := codectypes.NewInterfaceRegistry()
	authtypes.RegisterInterfaces(registry)
	cryptocodec.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)

	committed := uint64(95)
	unparsable := &sdk.TxResponse{Code: sdkerrors.ErrWrongSequence.ABCICode(), Codespace: sdkerrors.RootCodespace, RawLog: "incorrect account sequence"}
	insufficientFee := &sdk.TxResponse{Code: sdkerrors.ErrInsufficientFee.ABCICode(), Codespace: sdkerrors.RootCodespace}
	otherCodespace := &sdk.TxResponse{Code: sdkerrors.ErrMempoolIsFull.ABCICode(), Codespace: "clerk"}

	tests := []struct {
		name        string
		response    *sdk.TxResponse
		accountSeq  *uint64
		wantSeq     uint64
		wantNonOKRe bool
	}{
		{name: "unparsable mismatch resets to committed", response: unparsable, accountSeq: &committed, wantSeq: committed, wantNonOKRe: true},
		{name: "other rejection keeps the unused sequence", response: insufficientFee, accountSeq: &committed, wantSeq: 120, wantNonOKRe: true},
		{name: "module codespace is not mistaken for mempool full", response: otherCodespace, accountSeq: &committed, wantSeq: 120, wantNonOKRe: true},
		{name: "failed account query keeps sequence and reports it", response: unparsable, accountSeq: nil, wantSeq: 120},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			comet := cometAccountServer(t, cdc, tc.accountSeq)
			defer comet.Close()

			conf := helper.GetDefaultHeimdallConfig()
			conf.CometBFTRPCUrl = comet.URL
			helper.SetTestConfig(helper.CustomAppConfig{Config: *serverconfig.DefaultConfig(), Custom: conf})
			helper.SetTestPrivPubKey(privKey)

			tb := &TxBroadcaster{
				CliCtx:    client.Context{}.WithCodec(cdc).WithInterfaceRegistry(registry),
				logger:    log.NewNopLogger(),
				lastSeqNo: 120,
			}

			err := tb.handleNonOKResponse(context.Background(), tc.response)

			require.Error(t, err)
			require.False(t, errors.Is(err, ErrMempoolFull))
			require.Equal(t, tc.wantNonOKRe, strings.Contains(err.Error(), "broadcast succeeded but received non-ok response code"), "got %v", err)
			require.Equal(t, tc.wantSeq, tb.lastSeqNo)
		})
	}
}

// rejectingNode stubs the heimdall REST auth params and a CometBFT RPC that
// reports committed sequence 40 and answers broadcast_tx_sync with broadcast.
func rejectingNode(t *testing.T, cdc codec.Codec, broadcast func(id json.RawMessage) string) (*httptest.Server, *httptest.Server) {
	t.Helper()

	rest := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/cosmos/auth/v1beta1/params", r.URL.Path)
		body, err := cdc.MarshalJSON(&authtypes.QueryParamsResponse{Params: authtypes.DefaultParams()})
		require.NoError(t, err)
		_, _ = w.Write(body)
	}))
	accounts := cometAccountServer(t, cdc, ptr(uint64(40)))
	comet := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var req struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
		}
		require.NoError(t, json.Unmarshal(raw, &req))
		if req.Method == "broadcast_tx_sync" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, broadcast(req.ID))
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(raw))
		accounts.Config.Handler.ServeHTTP(w, r)
	}))
	t.Cleanup(func() { rest.Close(); accounts.Close(); comet.Close() })
	return rest, comet
}

func ptr[T any](v T) *T { return &v }

func TestBroadcastToHeimdallRejections(t *testing.T) {
	// Not parallel: mutates global helper config.
	registry := codectypes.NewInterfaceRegistry()
	authtypes.RegisterInterfaces(registry)
	cryptocodec.RegisterInterfaces(registry)
	clerktypes.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)

	tests := []struct {
		name        string
		broadcast   func(id json.RawMessage) string
		wantMempool bool
		wantOK      bool
		wantSeq     uint64
		wantLog     string
	}{
		{
			name: "identical tx in the mempool is an accepted broadcast",
			broadcast: func(id json.RawMessage) string {
				return fmt.Sprintf(`{"jsonrpc":"2.0","id":%s,"error":{"code":-32603,"message":"Internal error","data":"tx already exists in cache"}}`, id)
			},
			wantOK:  true,
			wantSeq: 42,
			wantLog: "Tx already in the mempool",
		},
		{
			name: "mempool full keeps the pending sequence",
			broadcast: func(id json.RawMessage) string {
				return fmt.Sprintf(`{"jsonrpc":"2.0","id":%s,"error":{"code":-32603,"message":"Internal error","data":"mempool is full: number of txs 5000 (max: 5000)"}}`, id)
			},
			wantMempool: true,
			wantSeq:     41,
		},
		{
			name: "sequence mismatch resyncs to the expected sequence",
			broadcast: func(id json.RawMessage) string {
				return fmt.Sprintf(`{"jsonrpc":"2.0","id":%s,"result":{"code":32,"data":"","log":"account sequence mismatch, expected 45, got 41: incorrect account sequence","codespace":"sdk","hash":"00"}}`, id)
			},
			wantSeq: 45,
			wantLog: "Resyncing account sequence from CheckTx",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rest, comet := rejectingNode(t, cdc, tc.broadcast)

			conf := helper.GetDefaultHeimdallConfig()
			conf.CometBFTRPCUrl = comet.URL
			srvConf := serverconfig.DefaultConfig()
			srvConf.API.Address = rest.URL
			helper.SetTestConfig(helper.CustomAppConfig{Config: *srvConf, Custom: conf})
			helper.SetTestPrivPubKey(privKey)
			prevClient := helper.Client
			helper.Client = &http.Client{}
			t.Cleanup(func() { helper.Client = prevClient })

			cometClient, err := rpchttp.New(comet.URL, "/websocket")
			require.NoError(t, err)
			cliCtx := client.Context{}.
				WithCodec(cdc).
				WithInterfaceRegistry(registry).
				WithTxConfig(authtx.NewTxConfig(cdc, authtx.DefaultSignModes)).
				WithChainID("heimdall-test").
				WithClient(cometClient).
				WithAccountRetriever(authtypes.AccountRetriever{})

			tb := NewTxBroadcaster(cdc, context.Background(), cliCtx, func(string) sdk.AccountI {
				return authtypes.NewBaseAccount(helper.GetAddress(), cosmosPrivKey.PubKey(), 7, 41)
			})
			logs := &bytes.Buffer{}
			tb.logger = log.NewLogger(logs, log.ColorOption(false))

			msg := clerktypes.NewMsgEventRecord(sdk.MustHexifyAddressBytes(helper.GetAddress()), "0x01", 0, 100, 49992, []byte{0x01}, []byte{0xde, 0xad}, "80002")
			txRes, err := tb.BroadcastToHeimdall(context.Background(), &msg, nil)
			require.Equal(t, tc.wantSeq, tb.lastSeqNo)
			require.Contains(t, logs.String(), tc.wantLog)
			if tc.wantOK {
				require.NoError(t, err)
				require.Equal(t, abci.CodeTypeOK, txRes.Code)
				return
			}

			require.ErrorContains(t, err, "broadcast succeeded but received non-ok response code")
			require.Equal(t, tc.wantMempool, errors.Is(err, ErrMempoolFull))
			require.Equal(t, tc.wantSeq, tb.lastSeqNo)
			require.Contains(t, logs.String(), tc.wantLog)
		})
	}
}

func TestInMempoolCache(t *testing.T) {
	require.True(t, inMempoolCache(&sdk.TxResponse{Code: sdkerrors.ErrTxInMempoolCache.ABCICode(), Codespace: sdkerrors.RootCodespace}))
	require.False(t, inMempoolCache(&sdk.TxResponse{Code: sdkerrors.ErrTxInMempoolCache.ABCICode(), Codespace: "clerk"}), "a module code 19 is something else")
	require.False(t, inMempoolCache(&sdk.TxResponse{Code: sdkerrors.ErrMempoolIsFull.ABCICode(), Codespace: sdkerrors.RootCodespace}))
}
