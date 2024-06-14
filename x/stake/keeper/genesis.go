package keeper

import (
	"context"

	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/0xPolygon/heimdall-v2/x/stake/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// InitGenesis sets validator information for genesis.
func (k Keeper) InitGenesis(ctx context.Context, data *types.GenesisState) (res []abci.ValidatorUpdate) {
	ctx = sdk.UnwrapSDKContext(ctx)

	// get current val set
	var vals []*types.Validator
	if len(data.CurrentValidatorSet.Validators) == 0 {
		vals = data.Validators
	} else {
		vals = data.CurrentValidatorSet.Validators
	}

	if len(vals) != 0 {
		resultValSet := types.NewValidatorSet(vals)

		// add validators in store
		for _, validator := range resultValSet.Validators {
			// Add individual validator to state
			if err := k.AddValidator(ctx, *validator); err != nil {
				k.Logger(ctx).Error("error caused inside InitGenesis fn", "error", err)
				panic(err)
			}

			// update validator set in store
			if err := k.UpdateValidatorSetInStore(ctx, *resultValSet); err != nil {
				panic(err)
			}

			// increment accum if init validator set
			if len(data.CurrentValidatorSet.Validators) == 0 {
				k.IncrementAccum(ctx, 1)
			}
		}
	}

	for _, sequence := range data.StakingSequences {
		err := k.SetStakingSequence(ctx, sequence)
		if err != nil {
			k.Logger(ctx).Error("error in setting staking sequence", "error", err)
			panic(err)
		}
	}
	return res
}

// ExportGenesis returns a GenesisState for a given context and keeper. The
// GenesisState will contain the validators and the staking sequences
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	return &types.GenesisState{
		k.GetAllValidators(ctx),
		k.GetValidatorSet(ctx),
		k.GetStakingSequences(ctx),
	}
}
