package keeper

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"cosmossdk.io/collections"

	addresscodec "cosmossdk.io/core/address"
	codecTypes "github.com/cosmos/cosmos-sdk/codec/types"
	cosmosTypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/0xPolygon/heimdall-v2/x/stake/types"
)

// AddValidator adds validator indexed with address
func (k *Keeper) AddValidator(ctx context.Context, validator types.Validator) error {

	// store validator with address prefixed with validator key as index
	err := k.validators.Set(ctx, validator.Signer, validator)
	if err != nil {
		k.Logger(ctx).Error("error while setting the validator in store", "err", err)
		return err
	}

	k.Logger(ctx).Debug("Validator stored", "key", validator.Signer, "validator", validator.String())

	// add validator to validator ID => SignerAddress map
	k.SetValidatorIDToSignerAddr(ctx, validator.ValId, validator.Signer)

	return nil
}

// IsCurrentValidatorByAddress check if validator is in current validator set by signer address
func (k *Keeper) IsCurrentValidatorByAddress(ctx context.Context, address string) bool {
	// get ack count
	ackCount := k.checkpointKeeper.GetACKCount(ctx)

	// get validator info
	validator, err := k.GetValidatorInfo(ctx, address)
	if err != nil {
		return false
	}

	// check if validator is current validator
	return validator.IsCurrentValidator(ackCount)
}

// GetValidatorInfo returns the validator info given its address
func (k *Keeper) GetValidatorInfo(ctx context.Context, address string) (validator types.Validator, err error) {
	validator, err = k.validators.Get(ctx, strings.ToLower(address))

	if err != nil {
		return validator, errors.New(fmt.Sprintf("error while fetching the validator from the store %v", err))
	}

	return validator, nil
}

// GetActiveValidatorInfo returns active validator
func (k *Keeper) GetActiveValidatorInfo(ctx context.Context, address string) (validator types.Validator, err error) {
	validator, err = k.GetValidatorInfo(ctx, address)
	if err != nil {
		return validator, err
	}

	// get ack count
	ackCount := k.checkpointKeeper.GetACKCount(ctx)
	if !validator.IsCurrentValidator(ackCount) {
		return validator, errors.New("validator is not active")
	}

	return validator, nil
}

// GetCurrentValidators returns all validators who are in validator set
func (k *Keeper) GetCurrentValidators(ctx context.Context) (validators []types.Validator) {
	// get ack count
	ackCount := k.checkpointKeeper.GetACKCount(ctx)

	// Get validators
	// iterate through validator list
	k.IterateValidatorsAndApplyFn(ctx, func(validator types.Validator) error {
		// check if validator is valid for current epoch
		if validator.IsCurrentValidator(ackCount) {
			// append if validator is current validator
			validators = append(validators, validator)
		}
		return nil
	})

	return
}

func (k *Keeper) GetTotalPower(ctx context.Context) (totalPower int64, err error) {
	err = k.IterateCurrentValidatorsAndApplyFn(ctx, func(validator cosmosTypes.ValidatorI) bool {
		// TODO HV2: @Vaibhav will this result in inconsistent behaviour?
		//  Given that we have our own validator definition, I think we ought to be using that and not what's defined in cosmos.
		totalPower += validator.GetBondedTokens().Int64()
		return true
	})
	if err != nil {
		return 0, err
	}

	return
}

// GetSpanEligibleValidators returns current validators who are not getting deactivated in between next span
func (k *Keeper) GetSpanEligibleValidators(ctx context.Context) (validators []types.Validator) {
	// get ack count
	ackCount := k.checkpointKeeper.GetACKCount(ctx)

	// Get validators and iterate through validator list
	k.IterateValidatorsAndApplyFn(ctx, func(validator types.Validator) error {
		// check if validator is valid for current epoch and endEpoch is not set.
		if validator.EndEpoch == 0 && validator.IsCurrentValidator(ackCount) {
			// append if validator is current validator
			validators = append(validators, validator)
		}
		return nil
	})

	return
}

// GetAllValidators returns all validators
func (k *Keeper) GetAllValidators(ctx context.Context) (validators []*types.Validator) {
	// iterate through validators and create validator update array
	k.IterateValidatorsAndApplyFn(ctx, func(validator types.Validator) error {
		// append to list of validatorUpdates
		validators = append(validators, &validator)
		return nil
	})

	return
}

// IterateValidatorsAndApplyFn iterate validators and apply the given function.
func (k *Keeper) IterateValidatorsAndApplyFn(ctx context.Context, f func(validator types.Validator) error) {
	// get validator iterator
	iterator, err := k.validators.Iterate(ctx, nil)

	// get validator iterator
	defer func() {
		err := iterator.Close()
		if err != nil {
			k.Logger(ctx).Error("error in closing the iterator", "error", err)
		}
	}()

	if err != nil {
		k.Logger(ctx).Error("error in getting iterator for validators")
		return
	}

	// loop through all the validators
	for ; iterator.Valid(); iterator.Next() {
		// unmarshall validator
		validator, err := iterator.Value()
		if err != nil {
			k.Logger(ctx).Error("error in getting validator from iterator", "err", err)
			return
		}

		// call function and return if required
		if err = f(validator); err != nil {
			return
		}
	}
}

// UpdateSigner updates validator fields in store
func (k *Keeper) UpdateSigner(ctx context.Context, newSigner string, newPubKey *codecTypes.Any, prevSigner string) error {
	// get old validator from state and make power 0
	validator, err := k.GetValidatorInfo(ctx, prevSigner)
	if err != nil {
		k.Logger(ctx).Error("unable to fetch validator from store")
		return err
	}

	// copy power to reassign below
	validatorPower := validator.VotingPower
	validator.VotingPower = 0

	// update validator
	if err := k.AddValidator(ctx, validator); err != nil {
		k.Logger(ctx).Error("error in adding validator", "error", err)
		return err
	}

	//update signer in prev validator
	validator.Signer = newSigner
	validator.PubKey = newPubKey
	validator.VotingPower = validatorPower

	// add updated validator to store with new key
	if err = k.AddValidator(ctx, validator); err != nil {
		k.Logger(ctx).Error("error in adding validator", "error", err)
		return err
	}

	return nil
}

// UpdateValidatorSetInStore adds validator set to store
func (k *Keeper) UpdateValidatorSetInStore(ctx context.Context, newValidatorSet types.ValidatorSet) error {
	// set validator set with CurrentValidatorSetKey as key in store
	err := k.validatorSet.Set(ctx, types.CurrentValidatorSetKey, newValidatorSet)
	if err != nil {
		k.Logger(ctx).Error("error in setting the current validator set in store", "err", err)
		return err
	}

	// When there is any update in checkpoint validator set, we assign it to milestone validator set too.
	err = k.validatorSet.Set(ctx, types.CurrentMilestoneValidatorSetKey, newValidatorSet)
	if err != nil {
		k.Logger(ctx).Error("error in setting the current milestone validator set in store", "err", err)
		return err
	}

	return nil
}

// GetValidatorSet returns current Validator Set from store
func (k *Keeper) GetValidatorSet(ctx context.Context) (validatorSet types.ValidatorSet, err error) {
	// get current validator set from store
	validatorSet, err = k.validatorSet.Get(ctx, types.CurrentValidatorSetKey)
	if err != nil {
		k.Logger(ctx).Error("error in fetching current validator set from store", "error", err)
		return validatorSet, err
	}

	// return validator set
	return validatorSet, nil
}

// IncrementAccum increments accum for validator set by n times and replace validator set in store
func (k *Keeper) IncrementAccum(ctx context.Context, times int) error {
	// get validator set
	validatorSet, err := k.GetValidatorSet(ctx)
	if err != nil {
		k.Logger(ctx).Error("error in fetching validator set from store", "error", err)
		return err

	}
	// increment accum
	validatorSet.IncrementProposerPriority(times)

	if err = k.UpdateValidatorSetInStore(ctx, validatorSet); err != nil {
		k.Logger(ctx).Error("error in updating validator set in store", "error", err)
		return err
	}

	return nil
}

// GetNextProposer returns next proposer
func (k *Keeper) GetNextProposer(ctx context.Context) *types.Validator {
	// get validator set
	validatorSet, err := k.GetValidatorSet(ctx)
	if err != nil {
		k.Logger(ctx).Error("error in fetching the validator set from database", "error", err)
		return nil
	}

	// Increment accum in copy
	copiedValidatorSet := validatorSet.CopyIncrementProposerPriority(1)

	// get signer address for next signer
	return copiedValidatorSet.GetProposer()
}

// GetCurrentProposer returns the current proposer from the validator set
func (k *Keeper) GetCurrentProposer(ctx context.Context) *types.Validator {
	// get validator set
	validatorSet, err := k.GetValidatorSet(ctx)
	if err != nil {
		k.Logger(ctx).Error("error in fetching the validator set from database", "error", err)
		return nil
	}

	return validatorSet.GetProposer()
}

// SetValidatorIDToSignerAddr sets mapping for validator ID to signer address
func (k *Keeper) SetValidatorIDToSignerAddr(ctx context.Context, valID uint64, signerAddr string) {
	err := k.signer.Set(ctx, valID, signerAddr)
	if err != nil {
		k.Logger(ctx).Error("key or value is nil", "error", err)
	}
}

// GetSignerFromValidatorID gets the signer address from the validator id
func (k *Keeper) GetSignerFromValidatorID(ctx context.Context, valID uint64) (common.Address, error) {
	signer, err := k.signer.Get(ctx, valID)
	if err != nil {
		k.Logger(ctx).Error("error while getting fetching signer address", "error", err)
		return common.Address{}, err
	}

	// return address from bytes
	return common.Address(common.FromHex(signer)), nil
}

// GetValidatorFromValID returns signer from validator ID
func (k *Keeper) GetValidatorFromValID(ctx context.Context, valID uint64) (validator types.Validator, err error) {
	signerAddr, err := k.GetSignerFromValidatorID(ctx, valID)
	if err != nil {
		return validator, err
	}

	// query for validator signer address
	validator, err = k.GetValidatorInfo(ctx, signerAddr.String())
	if err != nil {
		return validator, err
	}

	return validator, nil
}

// GetLastUpdated get last updated at for validator
func (k *Keeper) GetLastUpdated(ctx context.Context, valID uint64) (updatedAt string, err error) {
	// get validator
	validator, err := k.GetValidatorFromValID(ctx, valID)
	if err != nil {
		return "", err
	}

	return validator.LastUpdated, nil
}

// SetStakingSequence sets staking sequence
func (k *Keeper) SetStakingSequence(ctx context.Context, sequence string) error {
	return k.sequences.Set(ctx, sequence, true)
}

// HasStakingSequence checks if staking sequence already exists
func (k *Keeper) HasStakingSequence(ctx context.Context, sequence string) bool {
	res, err := k.sequences.Has(ctx, sequence)
	if err != nil {
		k.Logger(ctx).Error("error while checking for the existence of staking key in store", "error", err)
		return false
	}

	return res
}

// GetStakingSequences returns all the sequences appended together
func (k *Keeper) GetStakingSequences(ctx context.Context) (sequences []string, err error) {
	err = k.IterateStakingSequencesAndApplyFn(ctx, func(sequence string) error {
		sequences = append(sequences, sequence)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return
}

// IterateStakingSequencesAndApplyFn iterates staking sequences and applies the given function.
func (k *Keeper) IterateStakingSequencesAndApplyFn(ctx context.Context, f func(sequence string) error) (e error) {
	// get staking sequence iterator
	iterator, err := k.sequences.Iterate(ctx, nil)
	defer func(iterator collections.Iterator[string, bool]) {
		err := iterator.Close()
		if err != nil {
			k.Logger(ctx).Error("error in closing the iterator", "error", err)
			e = err
		}
	}(iterator)

	if err != nil {
		k.Logger(ctx).Error("error in getting iterator for validators")
		return
	}

	// loop through validators to get valid value
	for ; iterator.Valid(); iterator.Next() {
		sequence, err := iterator.Key()
		if err != nil {
			k.Logger(ctx).Error("error in getting key value", "err", err)
		}

		// call function and return if required
		if err := f(sequence); err != nil {
			return
		}
	}

	return
}

// GetValIdFromAddress returns a validator's id given its address string
func (k *Keeper) GetValIdFromAddress(ctx context.Context, address string) (uint64, error) {
	// get ack count
	ackCount := k.checkpointKeeper.GetACKCount(ctx)

	validator, err := k.GetValidatorInfo(ctx, strings.ToLower(address))
	if err != nil {
		return 0, err
	}

	// check if validator is current validator
	if !validator.IsCurrentValidator(ackCount) {
		return 0, errors.New("address not found in current validator set")
	}

	return validator.ValId, nil
}

// IterateCurrentValidatorsAndApplyFn iterate through current validators
func (k Keeper) IterateCurrentValidatorsAndApplyFn(ctx context.Context, f func(validator cosmosTypes.ValidatorI) bool) error {
	// TODO HV2: @Vaibhav check how to use the stop parameter here and eventually update/delete this comment
	currentValidatorSet, err := k.GetValidatorSet(ctx)
	if err != nil {
		k.Logger(ctx).Error("error in fetching the validator set from database", "error", err)
		return err
	}

	for _, v := range currentValidatorSet.Validators {
		if stop := f(v); !stop {
			return nil
		}
	}
	return nil
}

// MilestoneIncrementAccum increments accum for milestone validator set by n times and replace validator set in store
func (k *Keeper) MilestoneIncrementAccum(ctx context.Context, times int) {
	// get milestone validator set
	validatorSet, err := k.GetMilestoneValidatorSet(ctx)
	if err != nil {
		k.Logger(ctx).Error("error in fetching the milestone validator set from the db", "error", err)
		return
	}

	// increment accum
	validatorSet.IncrementProposerPriority(times)

	if err := k.UpdateMilestoneValidatorSetInStore(ctx, validatorSet); err != nil {
		k.Logger(ctx).Error("error in setting the milestone validator set in the db", "error", err)
	}
}

// GetMilestoneValidatorSet returns current milestone Validator Set from store
func (k *Keeper) GetMilestoneValidatorSet(ctx context.Context) (validatorSet types.ValidatorSet, err error) {
	// get the current milestone validator set
	validatorSet, err = k.validatorSet.Get(ctx, types.CurrentMilestoneValidatorSetKey)
	if err != nil {
		k.Logger(ctx).Error("error while getting milestone validator set from store", "error", err)
		return validatorSet, err
	}

	// return validator set
	return validatorSet, nil
}

// UpdateMilestoneValidatorSetInStore adds milestone validator set to store
func (k *Keeper) UpdateMilestoneValidatorSetInStore(ctx context.Context, newValidatorSet types.ValidatorSet) error {
	// set validator set with CurrentMilestoneValidatorSetKey as key in store
	return k.validatorSet.Set(ctx, types.CurrentMilestoneValidatorSetKey, newValidatorSet)
}

// GetMilestoneCurrentProposer returns current proposer
func (k *Keeper) GetMilestoneCurrentProposer(ctx context.Context) *types.Validator {
	// get validator set
	validatorSet, err := k.GetMilestoneValidatorSet(ctx)
	if err != nil {
		return nil
	}

	// return get proposer
	return validatorSet.GetProposer()
}

// ValidatorAddressCodec return the validator address codec
func (k *Keeper) ValidatorAddressCodec() addresscodec.Codec {
	return k.validatorAddressCodec
}
