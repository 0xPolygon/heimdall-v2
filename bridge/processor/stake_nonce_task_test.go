package processor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

// stakeNonceProcessor returns a StakingProcessor whose heimdall reports every event as new and the
// validator at heimdallNonce.
func stakeNonceProcessor(t *testing.T, validatorID, heimdallNonce uint64) (*StakingProcessor, *abi.ABI, *bytes.Buffer) {
	t.Helper()

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
	t.Cleanup(heimdall.Close)

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
	logs := new(bytes.Buffer)
	sp.Logger = log.NewLogger(logs, log.ColorOption(false))
	sp.cliCtx = client.Context{}.WithCodec(cdc)
	return sp, stakingABI, logs
}

func stakeNonceTasks(sp *StakingProcessor) map[string]func(context.Context, string, string) error {
	return map[string]func(context.Context, string, string) error{
		"StakeUpdate":  sp.sendStakeUpdateToHeimdall,
		"UnstakeInit":  sp.sendUnstakeInitToHeimdall,
		"SignerChange": sp.sendSignerChangeToHeimdall,
	}
}

func TestStakeTasks_OutOfOrderNonceRetriesLater(t *testing.T) {
	// Not parallel: mutates global helper config.
	const (
		validatorID   = 10
		heimdallNonce = 256
	)
	sp, stakingABI, logs := stakeNonceProcessor(t, validatorID, heimdallNonce)

	for event, task := range stakeNonceTasks(sp) {
		for _, eventNonce := range []int64{heimdallNonce + 2, heimdallNonce + 5, heimdallNonce + 10, heimdallNonce + 50} {
			t.Run(fmt.Sprintf("%s/%d", event, eventNonce), func(t *testing.T) {
				logs.Reset()
				signature := &tasks.Signature{}
				taskCtx, err := tasks.NewWithSignature(task, signature)
				require.NoError(t, err)

				err = task(taskCtx.Context, event, stakeEventLog(t, stakingABI, event, validatorID, eventNonce))

				var retryLater tasks.ErrRetryTaskLater
				require.True(t, errors.As(err, &retryLater), "got %v", err)
				delay := time.Duration(min(eventNonce-heimdallNonce, 10)) * util.StakeNonceRetryDelay
				require.Equal(t, delay, retryLater.RetryIn())
				require.Contains(t, logs.String(), errMsgNonceNotInOrder)
				require.Contains(t, logs.String(), fmt.Sprintf("delay=%d", int64(delay)))
				require.NotContains(t, logs.String(), infoMsgNonceAlreadyUsed)
				_, stamped := signature.Headers[eventRetryStartHeader]
				require.True(t, stamped)
			})
		}
	}
}

// A nonce heimdall already used can never apply, so the task ends instead of polling forever.
func TestStakeTasks_UsedNonceEndsTask(t *testing.T) {
	// Not parallel: mutates global helper config.
	const (
		validatorID   = 10
		heimdallNonce = 256
	)
	sp, stakingABI, logs := stakeNonceProcessor(t, validatorID, heimdallNonce)

	for event, task := range stakeNonceTasks(sp) {
		for _, eventNonce := range []int64{0, 1, heimdallNonce - 1, heimdallNonce} {
			t.Run(fmt.Sprintf("%s/%d", event, eventNonce), func(t *testing.T) {
				logs.Reset()
				signature := &tasks.Signature{}
				taskCtx, err := tasks.NewWithSignature(task, signature)
				require.NoError(t, err)

				err = task(taskCtx.Context, event, stakeEventLog(t, stakingABI, event, validatorID, eventNonce))
				require.NoError(t, err)
				require.NotContains(t, signature.Headers, eventRetryStartHeader)
				require.Contains(t, logs.String(), infoMsgNonceAlreadyUsed)
				require.NotContains(t, logs.String(), errMsgNonceNotInOrder)
			})
		}
	}
}
