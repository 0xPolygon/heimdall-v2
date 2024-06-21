package keeper

import (
	"context"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/telemetry"

	"github.com/0xPolygon/heimdall-v2/x/stake/types"
)

// EndBlocker called at the end of every block, and returns validator updates
func (k *Keeper) EndBlocker(ctx context.Context) ([]abci.ValidatorUpdate, error) {
	defer telemetry.ModuleMeasureSince(types.ModuleName, time.Now(), telemetry.MetricKeyEndBlocker)

	var cmtValUpdates []abci.ValidatorUpdate

	currentValidatorSet, err := k.GetValidatorSet(ctx)
	if err != nil {
		k.Logger(ctx).Error("error while calling the GetValidatorSet fn", "err", err)
		return cmtValUpdates, err
	}

	allValidators := k.GetAllValidators(ctx)
	ackCount := k.checkpointKeeper.GetACKCount(ctx)

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
			return cmtValUpdates, err
		}

		// save set in store
		if err := k.UpdateValidatorSetInStore(ctx, currentValidatorSet); err != nil {
			// return with nothing
			k.Logger(ctx).Error("unable to update current validator set in state", "error", err)
			return cmtValUpdates, err
		}

		// convert updates from map to array
		for _, v := range setUpdates {
			cmtProtoPk, err := v.CmtConsPublicKey()
			if err != nil {
				// TODO HV2 Should we panic at this condition?
				panic(err)
			}

			cmtValUpdates = append(cmtValUpdates, abci.ValidatorUpdate{
				Power:  v.VotingPower,
				PubKey: cmtProtoPk,
			})
		}
	}

	return cmtValUpdates, nil
}
