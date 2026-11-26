package processor

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/bridge/util"
	"github.com/0xPolygon/heimdall-v2/contracts/stakinginfo"
	"github.com/0xPolygon/heimdall-v2/helper"
	chainmanagertypes "github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
	staketypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

func TestCheckValidNonce_ValidatorUnknownToHeimdall(t *testing.T) {
	// Not parallel: mutates global helper config.
	registry := codectypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(registry)
	staketypes.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)

	validatorStatus := http.StatusNotFound
	validatorBody := ""
	paramsUnavailable := false
	heimdall := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasPrefix(r.URL.Path, "/stake/validator/"):
			w.WriteHeader(validatorStatus)
			if validatorBody != "" {
				_, _ = w.Write([]byte(validatorBody))
				return
			}
			if validatorStatus == http.StatusOK {
				body, err := cdc.MarshalJSON(&staketypes.QueryValidatorResponse{Validator: staketypes.Validator{ValId: 202, Nonce: 4}})
				require.NoError(t, err)
				_, _ = w.Write(body)
				return
			}
			_, _ = w.Write([]byte(`{"code":5,"message":"not found"}`))
		case strings.HasPrefix(r.URL.Path, util.ChainManagerParamsURL) && paramsUnavailable:
			w.WriteHeader(http.StatusInternalServerError)
		case strings.HasPrefix(r.URL.Path, util.ChainManagerParamsURL):
			params := chainmanagertypes.DefaultParams()
			params.ChainParams.StakingManagerAddress = testStakeManagerAddress
			body, err := cdc.MarshalJSON(&chainmanagertypes.QueryParamsResponse{Params: params})
			require.NoError(t, err)
			_, _ = w.Write(body)
		default:
			t.Errorf("unexpected heimdall path: %s", r.URL.Path)
		}
	}))
	defer heimdall.Close()

	prevClient := helper.Client
	helper.Client = heimdall.Client()
	t.Cleanup(func() { helper.Client = prevClient })

	cfg := helper.CustomAppConfig{Config: *serverconfig.DefaultConfig(), Custom: helper.GetDefaultHeimdallConfig()}
	cfg.Config.API.Address = heimdall.URL
	helper.SetTestConfig(cfg)

	stakingABI, err := stakinginfo.StakinginfoMetaData.GetAbi()
	require.NoError(t, err)

	tests := []struct {
		name              string
		validatorStatus   int
		validatorBody     string
		deactivationEpoch int64
		rpcErr            bool
		paramsUnavailable bool
		wantValid         bool
		wantErr           bool
		wantLogs          []string
		noLogs            []string
	}{
		{name: "unknown and exiting on L1 is dropped", validatorStatus: http.StatusNotFound, deactivationEpoch: 7, wantLogs: []string{infoMsgDroppingExitedValidator}, noLogs: []string{errMsgCheckExitOnL1}},
		{name: "unknown and still staked on L1 is retried", validatorStatus: http.StatusNotFound, wantErr: true, noLogs: []string{infoMsgDroppingExitedValidator, errMsgCheckExitOnL1}},
		{name: "route not found is retried without checking L1", validatorStatus: http.StatusNotFound, validatorBody: "404 page not found", deactivationEpoch: 7, wantErr: true, noLogs: []string{infoMsgDroppingExitedValidator, errMsgCheckExitOnL1}},
		{name: "unknown and L1 unavailable is retried", validatorStatus: http.StatusNotFound, rpcErr: true, wantErr: true, wantLogs: []string{errMsgCheckExitOnL1}, noLogs: []string{infoMsgDroppingExitedValidator}},
		{name: "unknown and chainmanager params unavailable is retried", validatorStatus: http.StatusNotFound, deactivationEpoch: 7, paramsUnavailable: true, wantErr: true, wantLogs: []string{errMsgCheckExitOnL1}, noLogs: []string{infoMsgDroppingExitedValidator}},
		{name: "server error is retried without checking L1", validatorStatus: http.StatusInternalServerError, deactivationEpoch: 7, wantErr: true, noLogs: []string{infoMsgDroppingExitedValidator, errMsgCheckExitOnL1}},
		{name: "known validator keeps the nonce check", validatorStatus: http.StatusOK, deactivationEpoch: 7, wantValid: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			validatorStatus = tc.validatorStatus
			validatorBody = tc.validatorBody
			paramsUnavailable = tc.paramsUnavailable
			mainChain := mainChainStub(t, tc.deactivationEpoch, tc.rpcErr)
			defer mainChain.Close()

			mainClient, err := ethclient.Dial(mainChain.URL)
			require.NoError(t, err)
			defer mainClient.Close()

			sp := NewStakingProcessor(stakingABI)
			var logs bytes.Buffer
			sp.Logger = log.NewLogger(&logs)
			sp.cliCtx = client.Context{}.WithCodec(cdc).WithInterfaceRegistry(registry)
			sp.contractCaller = &helper.ContractCaller{MainChainClient: mainClient, MainChainTimeout: 5 * time.Second}

			valid, gap, err := sp.checkValidNonce(202, 5)
			for _, msg := range tc.wantLogs {
				require.Contains(t, logs.String(), msg)
			}
			for _, msg := range tc.noLogs {
				require.NotContains(t, logs.String(), msg)
			}
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantValid, valid)
			require.Zero(t, gap)
		})
	}
}
