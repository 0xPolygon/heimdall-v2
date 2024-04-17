package keeper

import (
	"context"
	"time"

	"github.com/0xPolygon/heimdall-v2/helper"
	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/0xPolygon/heimdall-v2/x/stake/types"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BeginBlocker will persist the current header and validator set as a historical entry
// and prune the oldest entry based on the HistoricalEntries parameter
func (k *Keeper) BeginBlocker(ctx context.Context) error {
	defer telemetry.ModuleMeasureSince(types.ModuleName, time.Now(), telemetry.MetricKeyBeginBlocker)
	return nil
}

// EndBlocker called at every block, update validator set
func (k *Keeper) EndBlocker(ctx context.Context) ([]abci.ValidatorUpdate, error) {
	defer telemetry.ModuleMeasureSince(types.ModuleName, time.Now(), telemetry.MetricKeyEndBlocker)

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	var tmValUpdates []abci.ValidatorUpdate

	// --- Start update to new validators
	currentValidatorSet := k.GetValidatorSet(ctx)
	allValidators := k.GetAllValidators(ctx)
	ackCount := k.moduleCommunicator.GetACKCount(ctx)

	// get validator updates
	setUpdates := types.GetUpdatedValidators(
		&currentValidatorSet, // pointer to current validator set -- UpdateValidators will modify it
		allValidators,        // All validators
		ackCount,             // ack count
	)

	if len(setUpdates) > 0 {
		// create new validator set
		if err := currentValidatorSet.UpdateWithChangeSet(setUpdates); err != nil {
			// return error
			k.Logger(ctx).Error("Unable to update current validator set", "Error", err)
			return tmValUpdates, err
		}

		//Hardfork to remove the rotation of validator list on stake update
		if sdkCtx.BlockHeight() < helper.GetAalborgHardForkHeight() {
			// increment proposer priority
			currentValidatorSet.IncrementProposerPriority(1)
		}

		// save set in store
		if err := k.UpdateValidatorSetInStore(ctx, currentValidatorSet); err != nil {
			// return with nothing
			k.Logger(ctx).Error("Unable to update current validator set in state", "Error", err)
			return tmValUpdates, err
		}

		// convert updates from map to array
		for _, v := range setUpdates {
			tmProtoPk, err := v.CmtConsPublicKey()
			if err != nil {
				// TODO HV2 Should we panic at this condition?
				panic(err)
			}

			tmValUpdates = append(tmValUpdates, abci.ValidatorUpdate{
				Power: v.VotingPower,

				PubKey: tmProtoPk,
			})
		}
	}

	return tmValUpdates, nil
}
