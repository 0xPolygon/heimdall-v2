package keeper

import (
	"context"
	"errors"
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	util "github.com/0xPolygon/heimdall-v2/common/hex"
	"github.com/0xPolygon/heimdall-v2/x/stake/types"
)

// InitGenesis sets validator information for genesis in x/stake module
func (k Keeper) InitGenesis(ctx context.Context, data *types.GenesisState) []abci.ValidatorUpdate {
	k.PanicIfSetupIsIncomplete()

	// get the current validators' set
	var vals []*types.Validator
	if len(data.CurrentValidatorSet.Validators) == 0 {
		vals = data.Validators
	} else {
		vals = data.CurrentValidatorSet.Validators
	}

	if len(vals) != 0 {
		resultValSet := types.NewValidatorSet(vals)

		// at genesis, the previous validator set will be equal to the current validator set
		err := k.UpdatePreviousBlockValidatorSetInStore(ctx, *resultValSet)
		if err != nil {
			panic(fmt.Errorf("error updating previous validator set in store while initializing stake genesis: %w", err))
		}

		// Restore the H-2 set so a re-bootstrap doesn't start empty (the VE path reads it).
		if err := k.UpdatePenultimateBlockValidatorSetInStore(ctx, penultimateValidatorSetFromGenesis(data, *resultValSet)); err != nil {
			panic(fmt.Errorf("error updating penultimate validator set in store while initializing stake genesis: %w", err))
		}

		// add validators in store
		for _, validator := range resultValSet.Validators {
			// Add individual validator to the state
			if err := k.AddValidator(ctx, *validator); err != nil {
				panic(fmt.Errorf("error adding the validator while initializing stake genesis: %w", err))
			}

			// update validator set in store
			if err := k.UpdateValidatorSetInStore(ctx, *resultValSet); err != nil {
				panic(err)
			}

			// increment accum if initializing the validator set
			if len(data.CurrentValidatorSet.Validators) == 0 {
				err := k.IncrementAccum(ctx, 1)
				if err != nil {
					panic(fmt.Errorf("error incrementing the validators set accum while initializing stake genesis: %w", err))
				}
			}
		}
	}

	k.importValidatorStore(ctx, data)

	for _, sequence := range data.StakingSequences {
		err := k.SetStakingSequence(ctx, sequence)
		if err != nil {
			panic(fmt.Errorf("error in setting staking sequence while initializing stake genesis: %w", err))
		}
	}

	// set the last block txs
	if len(data.LastBlockTxs.Txs) > 0 {
		err := k.SetLastBlockTxs(ctx, data.LastBlockTxs.Txs)
		if err != nil {
			panic(fmt.Errorf("error in getting last block txs while initializing stake genesis: %w", err))
		}
	} else {
		// if no last block txs are provided, set it to empty
		err := k.SetLastBlockTxs(ctx, [][]byte{})
		if err != nil {
			panic(fmt.Errorf("error in setting last block txs while initializing stake genesis: %w", err))
		}
	}

	validators := k.GetAllValidators(ctx)
	cometVals := make([]abci.ValidatorUpdate, 0, len(validators))
	for _, validator := range validators {
		cmtPk, err := validator.CmtConsPublicKey()
		if err != nil {
			panic(err)
		}
		cometVals = append(cometVals, abci.ValidatorUpdate{
			PubKey: cmtPk,
			Power:  validator.GetVotingPower(),
		})
	}

	return cometVals
}

// penultimateValidatorSetFromGenesis returns the H-2 validator set to restore on import:
// the exported penultimate set, or the fallback (current set) when the export predates
// that field — mirroring the genesis previous==current convention.
func penultimateValidatorSetFromGenesis(data *types.GenesisState, fallback types.ValidatorSet) types.ValidatorSet {
	if len(data.PenultimateBlockValidatorSet.Validators) == 0 {
		return fallback
	}
	return data.PenultimateBlockValidatorSet
}

// importValidatorStore restores an exported stake store exactly. The set copies written
// above only refresh on voting power changes, so their nonce and last_updated can lag the
// store, they omit validators outside the current set, and rebuilding the set re-increments
// proposer priority. Genesis files without ValidatorSigners skip this to keep their import
// unchanged.
func (k Keeper) importValidatorStore(ctx context.Context, data *types.GenesisState) {
	if len(data.ValidatorSigners) == 0 {
		return
	}

	if err := data.ValidateValidatorSigners(); err != nil {
		panic(fmt.Errorf("invalid validator signers in stake genesis: %w", err))
	}

	var errs []error
	for _, validator := range data.Validators {
		errs = append(errs, k.AddValidator(ctx, *validator))
	}

	// AddValidator maps each ID to the last record written, which is wrong for an ID
	// whose old signer record is still in the store after a signer change.
	for _, vs := range data.ValidatorSigners {
		errs = append(errs, k.signer.Set(ctx, vs.ValId, util.FormatAddress(vs.Signer)))
	}

	// CometBFT runs the genesis validators, derived from the current set, for the first
	// two blocks, and the previous set becomes the penultimate set that verifies the
	// second block's vote extensions. So previous must be the current set, not the
	// exported previous block's set.
	current := data.CurrentValidatorSet
	errs = append(errs,
		k.UpdateValidatorSetInStore(ctx, current),
		k.UpdatePreviousBlockValidatorSetInStore(ctx, current),
		k.UpdatePenultimateBlockValidatorSetInStore(ctx, penultimateValidatorSetFromGenesis(data, current)),
	)
	if err := errors.Join(errs...); err != nil {
		panic(fmt.Errorf("error importing the validator store while initializing stake genesis: %w", err))
	}
}

func (k Keeper) exportValidatorSigners(ctx context.Context) ([]types.ValidatorSigner, error) {
	var signers []types.ValidatorSigner
	err := k.signer.Walk(ctx, nil, func(valID uint64, signer string) (bool, error) {
		signers = append(signers, types.ValidatorSigner{ValId: valID, Signer: signer})
		return false, nil
	})

	return signers, err
}

// ExportGenesis returns a GenesisState for the given stake context and keeper.
// The GenesisState will contain the validators and the staking sequences
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	k.PanicIfSetupIsIncomplete()

	validatorSet, err := k.GetValidatorSet(ctx)
	if err != nil {
		k.Logger(ctx).Error("Error in fetching validator set from store", "err", err)
		return nil
	}

	sequences, err := k.GetStakingSequences(ctx)
	if err != nil {
		k.Logger(ctx).Error("Error in fetching staking sequences from store", "err", err)
		return nil
	}

	previousValidatorSet, err := k.GetPreviousBlockValidatorSet(ctx)
	if err != nil {
		k.Logger(ctx).Error("Error in fetching previous validator set from store", "err", err)
		return nil
	}

	lastBlockTxs, err := k.GetLastBlockTxs(ctx)
	if err != nil {
		k.Logger(ctx).Error("Error in fetching last block txs from store", "err", err)
		return nil
	}

	penultimateValidatorSet, err := k.GetPenultimateBlockValidatorSet(ctx)
	if err != nil {
		k.Logger(ctx).Error("Error in fetching penultimate validator set from store", "err", err)
		return nil
	}

	validatorSigners, err := k.exportValidatorSigners(ctx)
	if err != nil {
		k.Logger(ctx).Error("Error in fetching validator signers from store", "err", err)
		return nil
	}

	return &types.GenesisState{
		Validators:                   k.GetAllValidators(ctx),
		CurrentValidatorSet:          validatorSet,
		StakingSequences:             sequences,
		PreviousBlockValidatorSet:    previousValidatorSet,
		LastBlockTxs:                 lastBlockTxs,
		PenultimateBlockValidatorSet: penultimateValidatorSet,
		ValidatorSigners:             validatorSigners,
	}
}
