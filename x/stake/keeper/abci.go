package keeper

import (
	"context"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/0xPolygon/heimdall-v2/x/stake/types"
	"github.com/cosmos/cosmos-sdk/telemetry"
)

// BeginBlocker will send the current time value to telemetry.
func (k *Keeper) BeginBlocker(ctx context.Context) error {
	defer telemetry.ModuleMeasureSince(types.ModuleName, time.Now(), telemetry.MetricKeyBeginBlocker)
	return nil
}

// EndBlocker called at the end of every block, and returns validator updates
func (k *Keeper) EndBlocker(ctx context.Context) ([]abci.ValidatorUpdate, error) {
	defer telemetry.ModuleMeasureSince(types.ModuleName, time.Now(), telemetry.MetricKeyEndBlocker)

	var tmValUpdates []abci.ValidatorUpdate

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
			k.Logger(ctx).Error("unable to update current validator set", "error", err)
			return tmValUpdates, err
		}

		// save set in store
		if err := k.UpdateValidatorSetInStore(ctx, currentValidatorSet); err != nil {
			// return with nothing
			k.Logger(ctx).Error("unable to update current validator set in state", "error", err)
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
				Power:  v.VotingPower,
				PubKey: tmProtoPk,
			})
		}
	}

	return tmValUpdates, nil
}
