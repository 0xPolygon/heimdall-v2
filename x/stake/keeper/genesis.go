package keeper

import (
	"context"

	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/0xPolygon/heimdall-v2/x/stake/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// InitGenesis sets validator information for genesis.
func (k Keeper) InitGenesis(ctx context.Context, data *types.GenesisState) (res []abci.ValidatorUpdate) {

	// We need to pretend to be "n blocks before genesis", where "n" is the
	// validator update delay, so that e.g. slashing periods are correctly
	// initialized for the validator set e.g. with a one-block offset - the
	// first TM block is at height 1, so state updates applied from
	// genesis.json are in block 0.
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	ctx = sdkCtx

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
				k.Logger(ctx).Error("Error InitGenesis", "error", err)
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
			panic(err)
		}
	}
	return res
}

// ExportGenesis returns a GenesisState for a given context and keeper. The
// GenesisState will contain the pool, params, validators, and bonds found in
// the keeper.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	return &types.GenesisState{
		k.GetAllValidators(ctx),
		k.GetValidatorSet(ctx),
		k.GetStakingSequences(ctx),
	}
}
