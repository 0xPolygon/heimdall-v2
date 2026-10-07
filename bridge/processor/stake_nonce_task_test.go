package processor

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"cosmossdk.io/log"
	"github.com/RichardKnop/machinery/v1/tasks"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	"github.com/cosmos/gogoproto/proto"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/bridge/util"
	"github.com/0xPolygon/heimdall-v2/contracts/stakinginfo"
	"github.com/0xPolygon/heimdall-v2/helper"
	staketypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

// stakeEventLog encodes eventName from the stakinginfo ABI with validatorId and
// nonce set, and placeholder values for every other input.
func stakeEventLog(t *testing.T, stakingABI *abi.ABI, eventName string, validatorID, nonce int64) string {
	t.Helper()

	event := stakingABI.Events[eventName]
	value := func(arg abi.Argument) any {
		switch {
		case arg.Name == "validatorId":
			return big.NewInt(validatorID)
		case arg.Name == "nonce":
			return big.NewInt(nonce)
		case arg.Type.T == abi.AddressTy:
			return common.HexToAddress("0x0000000000000000000000000000000000000001")
		case arg.Type.T == abi.BytesTy:
			return make([]byte, 64)
		default:
			return big.NewInt(1)
		}
	}

	topics := []common.Hash{event.ID}
	var nonIndexed []any
	for _, arg := range event.Inputs {
		if !arg.Indexed {
			nonIndexed = append(nonIndexed, value(arg))
			continue
		}
		switch v := value(arg).(type) {
		case *big.Int:
			topics = append(topics, common.BigToHash(v))
		case common.Address:
			topics = append(topics, common.BytesToHash(v.Bytes()))
		default:
			t.Fatalf("unsupported indexed input %s", arg.Name)
		}
	}

	data, err := event.Inputs.NonIndexed().Pack(nonIndexed...)
	require.NoError(t, err)

	logBytes, err := json.Marshal(types.Log{
		Address: common.HexToAddress("0x0000000000000000000000000000000000000002"),
		Topics:  topics,
		Data:    data,
		TxHash:  common.HexToHash("0xaa"),
	})
	require.NoError(t, err)
	return string(logBytes)
}

func TestStakeTasks_OutOfOrderNonceRetriesLater(t *testing.T) {
	// Not parallel: mutates global helper config.
	const (
		validatorID   = 10
		heimdallNonce = 256
		eventNonce    = 261
	)

	registry := codectypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(registry)
	staketypes.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)

	heimdall := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var resp proto.Message
		switch {
		case strings.HasPrefix(r.URL.Path, "/stake/is-old-tx"):
			resp = &staketypes.QueryStakeIsOldTxResponse{IsOld: false}
		case strings.HasPrefix(r.URL.Path, "/stake/validator/"):
			resp = &staketypes.QueryValidatorResponse{Validator: staketypes.Validator{
				ValId: validatorID, Nonce: heimdallNonce, VotingPower: 100,
				Signer: "0x0000000000000000000000000000000000000001",
			}}
		default:
			t.Errorf("unexpected heimdall path: %s", r.URL.Path)
			return
		}
		body, err := cdc.MarshalJSON(resp)
		require.NoError(t, err)
		_, _ = w.Write(body)
	}))
	defer heimdall.Close()

	// Other tests in this package swap helper.Client for gomock mocks without
	// restoring it, so install a real client for this test's REST calls.
	prevClient := helper.Client
	helper.Client = heimdall.Client()
	t.Cleanup(func() { helper.Client = prevClient })

	cfg := helper.CustomAppConfig{Config: *serverconfig.DefaultConfig(), Custom: helper.GetDefaultHeimdallConfig()}
	cfg.Config.API.Address = heimdall.URL
	helper.SetTestConfig(cfg)

	stakingABI, err := stakinginfo.StakinginfoMetaData.GetAbi()
	require.NoError(t, err)

	sp := NewStakingProcessor(stakingABI)
	sp.Logger = log.NewNopLogger()
	sp.cliCtx = client.Context{}.WithCodec(cdc)

	tests := []struct {
		event string
		task  func(context.Context, string, string) error
	}{
		{event: "StakeUpdate", task: sp.sendStakeUpdateToHeimdall},
		{event: "UnstakeInit", task: sp.sendUnstakeInitToHeimdall},
		{event: "SignerChange", task: sp.sendSignerChangeToHeimdall},
	}

	for _, tc := range tests {
		t.Run(tc.event, func(t *testing.T) {
			signature := &tasks.Signature{}
			task, err := tasks.NewWithSignature(tc.task, signature)
			require.NoError(t, err)

			err = tc.task(task.Context, tc.event, stakeEventLog(t, stakingABI, tc.event, validatorID, eventNonce))

			var retryLater tasks.ErrRetryTaskLater
			require.True(t, errors.As(err, &retryLater), "got %v", err)
			require.Equal(t, (eventNonce-heimdallNonce)*util.StakeNonceRetryDelay, retryLater.RetryIn())
			_, stamped := nonceRetryStart(signature.Headers)
			require.True(t, stamped)
		})
	}
}
