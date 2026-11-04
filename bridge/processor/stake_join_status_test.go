package processor

import (
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"cosmossdk.io/log"
	"github.com/RichardKnop/machinery/v1/tasks"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/bridge/util"
	"github.com/0xPolygon/heimdall-v2/contracts/stakinginfo"
	"github.com/0xPolygon/heimdall-v2/helper"
	chainmanagertypes "github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
	staketypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

const testStakingInfoAddress = "0x00000000000000000000000000000000000000a1"

// mainChainStub answers eth_call for getStakerDetails with the given
// deactivation epoch, or with an RPC error when rpcErr is set.
func mainChainStub(t *testing.T, deactivationEpoch int64, rpcErr bool) *httptest.Server {
	t.Helper()

	stakingABI, err := stakinginfo.StakinginfoMetaData.GetAbi()
	require.NoError(t, err)

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     json.RawMessage   `json:"id"`
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))

		resp := map[string]any{"jsonrpc": "2.0", "id": req.ID}
		switch {
		case rpcErr || req.Method != "eth_call":
			resp["error"] = map[string]any{"code": -32000, "message": "unavailable"}
		default:
			var call struct {
				To   common.Address `json:"to"`
				Data hexutil.Bytes  `json:"input"`
			}
			require.NoError(t, json.Unmarshal(req.Params[0], &call))
			require.Equal(t, common.HexToAddress(testStakingInfoAddress), call.To)
			require.Equal(t, stakingABI.Methods["getStakerDetails"].ID, []byte(call.Data[:4]))

			out, err := stakingABI.Methods["getStakerDetails"].Outputs.Pack(
				big.NewInt(0), big.NewInt(0), big.NewInt(1), big.NewInt(deactivationEpoch),
				common.Address{}, big.NewInt(3),
			)
			require.NoError(t, err)
			resp["result"] = hexutil.Encode(out)
		}
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}))
}

func TestSendValidatorJoin_SignerAccountMissing(t *testing.T) {
	// Not parallel: mutates global helper config.
	registry := codectypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(registry)
	staketypes.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)

	paramsUnavailable := false
	heimdall := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var resp proto.Message
		switch {
		case strings.HasPrefix(r.URL.Path, "/stake/is-old-tx"):
			resp = &staketypes.QueryStakeIsOldTxResponse{IsOld: false}
		case strings.HasPrefix(r.URL.Path, util.ChainManagerParamsURL) && paramsUnavailable:
			w.WriteHeader(http.StatusInternalServerError)
			return
		case strings.HasPrefix(r.URL.Path, util.ChainManagerParamsURL):
			params := chainmanagertypes.DefaultParams()
			params.ChainParams.StakingInfoAddress = testStakingInfoAddress
			resp = &chainmanagertypes.QueryParamsResponse{Params: params}
		default:
			t.Errorf("unexpected heimdall path: %s", r.URL.Path)
			return
		}
		body, err := cdc.MarshalJSON(resp)
		require.NoError(t, err)
		_, _ = w.Write(body)
	}))
	defer heimdall.Close()

	prevClient := helper.Client
	helper.Client = heimdall.Client()
	t.Cleanup(func() { helper.Client = prevClient })

	// Nothing listens on the CometBFT URL, so the signer account lookup fails.
	cfg := helper.CustomAppConfig{Config: *serverconfig.DefaultConfig(), Custom: helper.GetDefaultHeimdallConfig()}
	cfg.Config.API.Address = heimdall.URL
	cfg.Custom.CometBFTRPCUrl = "http://127.0.0.1:1"
	helper.SetTestConfig(cfg)

	stakingABI, err := stakinginfo.StakinginfoMetaData.GetAbi()
	require.NoError(t, err)

	tests := []struct {
		name              string
		deactivationEpoch int64
		rpcErr            bool
		paramsUnavailable bool
		retry             bool
	}{
		{name: "validator still staked", retry: true},
		{name: "validator exited", deactivationEpoch: 7},
		{name: "main chain unavailable", rpcErr: true, retry: true},
		{name: "chainmanager params unavailable", deactivationEpoch: 7, paramsUnavailable: true, retry: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			paramsUnavailable = tc.paramsUnavailable
			mainChain := mainChainStub(t, tc.deactivationEpoch, tc.rpcErr)
			defer mainChain.Close()

			mainClient, err := ethclient.Dial(mainChain.URL)
			require.NoError(t, err)
			defer mainClient.Close()

			sp := NewStakingProcessor(stakingABI)
			sp.Logger = log.NewNopLogger()
			sp.cliCtx = client.Context{}.WithCodec(cdc).WithInterfaceRegistry(registry)
			sp.contractCaller = helper.ContractCaller{MainChainClient: mainClient, MainChainTimeout: 5 * time.Second}

			err = sp.sendValidatorJoinToHeimdall("Staked", stakeEventLog(t, stakingABI, "Staked", 202, 1))

			if !tc.retry {
				require.NoError(t, err)
				return
			}
			var retryLater tasks.ErrRetryTaskLater
			require.True(t, errors.As(err, &retryLater), "got %v", err)
			require.Equal(t, util.RetryTaskDelay, retryLater.RetryIn())
		})
	}
}

func TestSignerAccountReady_AccountExists(t *testing.T) {
	// Not parallel: mutates global helper config.
	registry := codectypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(registry)
	authtypes.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)

	signer := common.HexToAddress("0x0000000000000000000000000000000000000001")
	account, err := codectypes.NewAnyWithValue(&authtypes.BaseAccount{Address: sdk.AccAddress(signer.Bytes()).String()})
	require.NoError(t, err)
	value, err := cdc.Marshal(&authtypes.QueryAccountResponse{Account: account})
	require.NoError(t, err)

	cometBFT := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID json.RawMessage `json:"id"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result":  map[string]any{"response": map[string]any{"code": 0, "value": value, "height": "1"}},
		}))
	}))
	defer cometBFT.Close()

	cfg := helper.CustomAppConfig{Config: *serverconfig.DefaultConfig(), Custom: helper.GetDefaultHeimdallConfig()}
	cfg.Custom.CometBFTRPCUrl = cometBFT.URL
	helper.SetTestConfig(cfg)

	stakingABI, err := stakinginfo.StakinginfoMetaData.GetAbi()
	require.NoError(t, err)

	sp := NewStakingProcessor(stakingABI)
	sp.Logger = log.NewNopLogger()
	sp.cliCtx = client.Context{}.WithCodec(cdc).WithInterfaceRegistry(registry)

	ready, err := sp.signerAccountReady("Staked", &stakinginfo.StakinginfoStaked{Signer: signer, ValidatorId: big.NewInt(202)}, "0xaa")
	require.NoError(t, err)
	require.True(t, ready)
}
