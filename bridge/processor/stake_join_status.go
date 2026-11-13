package processor

import (
	"context"
	"fmt"
	"math/big"

	"github.com/RichardKnop/machinery/v1/tasks"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	"github.com/0xPolygon/heimdall-v2/bridge/util"
	"github.com/0xPolygon/heimdall-v2/contracts/stakinginfo"
)

const infoMsgDroppingExitedJoin = "StakingProcessor: dropping validator-join task as the validator has already started exiting on L1"

// signerAccountReady reports whether a validator-join can be broadcast. Until the
// signer has a heimdall account the task is retried, unless the validator has
// already started exiting on L1: heimdall votes against such a join, so the task
// is dropped instead of retrying for as long as the account stays missing.
func (sp *StakingProcessor) signerAccountReady(eventName string, event *stakinginfo.StakinginfoStaked, txHash string) (bool, error) {
	if _, err := util.GetAccount(context.Background(), sp.cliCtx, sdk.MustAccAddressFromHex(event.Signer.Hex()).String()); err == nil {
		return true, nil
	}

	exited, err := sp.hasExitedOnMainChain(event.ValidatorId)
	if err != nil {
		sp.Logger.Error("StakingProcessor: failed to check the validator status on L1", "validatorID", event.ValidatorId, "error", err)
	}
	if exited {
		sp.Logger.Info(infoMsgDroppingExitedJoin, "validatorID", event.ValidatorId, "signer", event.Signer, "txHash", txHash)
		return false, nil
	}

	sp.Logger.Info(
		infoMsgAccountDoesNotExist,
		"event", eventName,
		"signer", event.Signer,
	)
	return false, tasks.NewErrRetryTaskLater(msgAccountDoesNotExist, util.RetryTaskDelay)
}

func (sp *StakingProcessor) hasExitedOnMainChain(validatorID *big.Int) (bool, error) {
	params, err := util.GetChainmanagerParams(sp.cliCtx.Codec)
	if err != nil {
		return false, fmt.Errorf("failed to fetch chainmanager params: %w", err)
	}

	stakingInfo, err := stakinginfo.NewStakinginfoCaller(common.HexToAddress(params.ChainParams.StakingInfoAddress), sp.contractCaller.MainChainClient)
	if err != nil {
		return false, fmt.Errorf("failed to bind staking info: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), sp.contractCaller.MainChainTimeout)
	defer cancel()

	details, err := stakingInfo.GetStakerDetails(&bind.CallOpts{Context: ctx}, validatorID)
	if err != nil {
		return false, fmt.Errorf("failed to fetch staker details: %w", err)
	}

	return details.DeactivationEpoch.Sign() != 0, nil
}
