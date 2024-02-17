package keeper

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	addresscodec "cosmossdk.io/core/address"
	storetypes "cosmossdk.io/store/types"
	hmTypes "github.com/0xPolygon/heimdall-v2/x/types"
	codecTypes "github.com/cosmos/cosmos-sdk/codec/types"
	cosmosTypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/0xPolygon/heimdall-v2/x/stake/types"
)

// AddValidator adds validator indexed with address
func (k *Keeper) AddValidator(ctx context.Context, validator hmTypes.Validator) error {
	store := k.storeService.OpenKVStore(ctx)

	bz, err := hmTypes.MarshallValidator(k.cdc, validator)
	if err != nil {
		return err
	}

	valAddrBytes, err := k.validatorAddressCodec.StringToBytes(validator.Signer)
	if err != nil {
		return err
	}

	// store validator with address prefixed with validator key as index
	store.Set(types.GetValidatorKey(valAddrBytes), bz)

	k.Logger(ctx).Debug("Validator stored", "key", hex.EncodeToString(types.GetValidatorKey(valAddrBytes)), "validator", validator.String())

	// add validator to validator ID => SignerAddress map
	k.SetValidatorIDToSignerAddr(ctx, validator.ValId, validator.Signer)

	return nil
}

// IsCurrentValidatorByAddress check if validator is in current validator set by signer address
func (k *Keeper) IsCurrentValidatorByAddress(ctx context.Context, address string) bool {
	// get ack count
	ackCount := k.moduleCommunicator.GetACKCount(ctx)

	// get validator info
	validator, err := k.GetValidatorInfo(ctx, address)
	if err != nil {
		return false
	}

	// check if validator is current validator
	return validator.IsCurrentValidator(ackCount)
}

// GetValidatorInfo returns validator
func (k *Keeper) GetValidatorInfo(ctx context.Context, address string) (validator hmTypes.Validator, err error) {
	store := k.storeService.OpenKVStore(ctx)
	address = strings.ToLower(address)
	valAddr, err := k.validatorAddressCodec.StringToBytes(address)
	if err != nil {
		return validator, err
	}

	// check if validator exists
	key := types.GetValidatorKey(valAddr)

	valBytes, err := store.Get(key)

	if err != nil {
		return validator, errors.New("error while fetchig the validator from the store")
	}

	if valBytes == nil {
		return validator, errors.New("Validator not found")
	}

	// unmarshall validator and return
	validator, err = hmTypes.UnmarshallValidator(k.cdc, valBytes)
	if err != nil {
		return validator, err
	}

	// return true if validator
	return validator, nil
}

// GetActiveValidatorInfo returns active validator
func (k *Keeper) GetActiveValidatorInfo(ctx context.Context, address string) (validator hmTypes.Validator, err error) {
	validator, err = k.GetValidatorInfo(ctx, address)
	if err != nil {
		return validator, err
	}

	// get ack count
	ackCount := k.moduleCommunicator.GetACKCount(ctx)
	if !validator.IsCurrentValidator(ackCount) {
		return validator, errors.New("Validator is not active")
	}

	// return true if validator
	return validator, nil
}

// GetCurrentValidators returns all validators who are in validator set
func (k *Keeper) GetCurrentValidators(ctx context.Context) (validators []hmTypes.Validator) {
	// get ack count
	ackCount := k.moduleCommunicator.GetACKCount(ctx)

	// Get validators
	// iterate through validator list
	k.IterateValidatorsAndApplyFn(ctx, func(validator hmTypes.Validator) error {
		// check if validator is valid for current epoch
		if validator.IsCurrentValidator(ackCount) {
			// append if validator is current valdiator
			validators = append(validators, validator)
		}
		return nil
	})

	return
}

func (k *Keeper) GetTotalPower(ctx context.Context) (totalPower int64) {
	k.IterateCurrentValidatorsAndApplyFn(ctx, func(validator cosmosTypes.ValidatorI) bool {
		totalPower += validator.GetBondedTokens().Int64()
		return true
	})

	return
}

// GetSpanEligibleValidators returns current validators who are not getting deactivated in between next span
func (k *Keeper) GetSpanEligibleValidators(ctx context.Context) (validators []hmTypes.Validator) {
	// get ack count
	ackCount := k.moduleCommunicator.GetACKCount(ctx)

	// Get validators and iterate through validator list
	k.IterateValidatorsAndApplyFn(ctx, func(validator hmTypes.Validator) error {
		// check if validator is valid for current epoch and endEpoch is not set.
		if validator.EndEpoch == 0 && validator.IsCurrentValidator(ackCount) {
			// append if validator is current valdiator
			validators = append(validators, validator)
		}
		return nil
	})

	return
}

// GetAllValidators returns all validators
func (k *Keeper) GetAllValidators(ctx context.Context) (validators []*hmTypes.Validator) {
	// iterate through validators and create validator update array
	k.IterateValidatorsAndApplyFn(ctx, func(validator hmTypes.Validator) error {
		// append to list of validatorUpdates
		validators = append(validators, &validator)
		return nil
	})

	return
}

// IterateValidatorsAndApplyFn iterate validators and apply the given function.
func (k *Keeper) IterateValidatorsAndApplyFn(ctx context.Context, f func(validator hmTypes.Validator) error) {
	store := k.storeService.OpenKVStore(ctx)

	// get validator iterator
	iterator, err := store.Iterator(types.ValidatorsKey, storetypes.PrefixEndBytes(types.ValidatorsKey))
	defer iterator.Close()

	if err != nil {
		k.Logger(ctx).Error("Error in getting iterator for validators")
		return
	}

	// loop through validators to get valid validators
	for ; iterator.Valid(); iterator.Next() {
		// unmarshall validator
		validator, _ := hmTypes.UnmarshallValidator(k.cdc, iterator.Value())
		// call function and return if required
		if err := f(validator); err != nil {
			return
		}
	}
}

// UpdateSigner updates validator with signer and pubkey + validator => signer map
func (k *Keeper) UpdateSigner(ctx context.Context, newSigner string, newPubkey *codecTypes.Any, prevSigner string) error {
	// get old validator from state and make power 0
	validator, err := k.GetValidatorInfo(ctx, prevSigner)
	if err != nil {
		k.Logger(ctx).Error("Unable to fetch validator from store")
		return err
	}

	// copy power to reassign below
	validatorPower := validator.VotingPower
	validator.VotingPower = 0

	// update validator
	if err := k.AddValidator(ctx, validator); err != nil {
		k.Logger(ctx).Error("UpdateSigner | AddValidator", "error", err)
	}

	//update signer in prev Validator
	validator.Signer = newSigner
	validator.PubKey = newPubkey
	validator.VotingPower = validatorPower

	// add updated validator to store with new key
	if err = k.AddValidator(ctx, validator); err != nil {
		k.Logger(ctx).Error("UpdateSigner | AddValidator", "error", err)
	}

	return nil
}

// UpdateValidatorSetInStore adds validator set to store
func (k *Keeper) UpdateValidatorSetInStore(ctx context.Context, newValidatorSet hmTypes.ValidatorSet) error {
	// TODO check if we may have to delay this by 1 height to sync with tendermint validator updates
	store := k.storeService.OpenKVStore(ctx)

	// marshall validator set
	bz, err := k.cdc.Marshal(&newValidatorSet)
	if err != nil {
		return err
	}

	// set validator set with CurrentValidatorSetKey as key in store
	store.Set(types.CurrentValidatorSetKey, bz)

	//When there is any update in checkpoint validator set, we assign it to milestone validator set too.
	store.Set(types.CurrentMilestoneValidatorSetKey, bz)

	return nil
}

// GetValidatorSet returns current Validator Set from store
func (k *Keeper) GetValidatorSet(ctx context.Context) (validatorSet hmTypes.ValidatorSet) {
	store := k.storeService.OpenKVStore(ctx)
	// get current validator set from store
	bz, err := store.Get(types.CurrentValidatorSetKey)

	if err != nil {
		k.Logger(ctx).Error("GetValidatorSet | CurrentValidatorSetKeyDoesNotExist ", "error", err)
	}

	// unmarhsall
	if err := k.cdc.Unmarshal(bz, &validatorSet); err != nil {
		k.Logger(ctx).Error("GetValidatorSet | UnmarshalBinaryBare", "error", err)
	}

	// return validator set
	return validatorSet
}

// IncrementAccum increments accum for validator set by n times and replace validator set in store
func (k *Keeper) IncrementAccum(ctx context.Context, times int) {
	// get validator set
	validatorSet := k.GetValidatorSet(ctx)

	// increment accum
	validatorSet.IncrementProposerPriority(times)

	// replace

	if err := k.UpdateValidatorSetInStore(ctx, validatorSet); err != nil {
		k.Logger(ctx).Error("IncrementAccum | UpdateValidatorSetInStore", "error", err)
	}
}

// GetNextProposer returns next proposer
func (k *Keeper) GetNextProposer(ctx context.Context) *hmTypes.Validator {
	// get validator set
	validatorSet := k.GetValidatorSet(ctx)

	// Increment accum in copy
	copiedValidatorSet := validatorSet.CopyIncrementProposerPriority(1)

	// get signer address for next signer
	return copiedValidatorSet.GetProposer()
}

// GetCurrentProposer returns current proposer
func (k *Keeper) GetCurrentProposer(ctx context.Context) *hmTypes.Validator {
	// get validator set
	validatorSet := k.GetValidatorSet(ctx)

	// return get proposer
	return validatorSet.GetProposer()
}

// SetValidatorIDToSignerAddr sets mapping for validator ID to signer address
func (k *Keeper) SetValidatorIDToSignerAddr(ctx context.Context, valID uint64, signerAddr string) {
	store := k.storeService.OpenKVStore(ctx)
	signerAddrBytes, err := k.validatorAddressCodec.StringToBytes(signerAddr)
	if err != nil {
		k.Logger(ctx).Error("SetValidatorIDToSignerAddr | Error while converting addr to bytes", "error", err)
	}

	err = store.Set(types.GetValidatorMapKey(hmTypes.ValIDToBytes(valID)), signerAddrBytes)
	if err != nil {
		k.Logger(ctx).Error("SetValidatorIDToSignerAddr | Key or value is nil", "error", err)
	}
}

// GetSignerFromValidatorID get signer address from validator ID
func (k *Keeper) GetSignerFromValidatorID(ctx context.Context, valID uint64) (common.Address, bool) {
	store := k.storeService.OpenKVStore(ctx)
	key := types.GetValidatorMapKey(hmTypes.ValIDToBytes(valID))
	// check if validator address has been mapped

	bz, err := store.Get(key)
	if err != nil || bz == nil {
		k.Logger(ctx).Error("GetSignerFromValidatorID | ValidatorIDKeyDoesNotExist ", "error", err)
		return common.Address{}, false
	}

	// return address from bytes
	return common.BytesToAddress(bz), true
}

// GetValidatorFromValID returns signer from validator ID
func (k *Keeper) GetValidatorFromValID(ctx context.Context, valID uint64) (validator hmTypes.Validator, ok bool) {
	signerAddr, ok := k.GetSignerFromValidatorID(ctx, valID)
	if !ok {
		return validator, ok
	}

	fmt.Println(signerAddr.Bytes())
	// query for validator signer address
	validator, err := k.GetValidatorInfo(ctx, signerAddr.String())
	if err != nil {
		return validator, false
	}

	return validator, true
}

// GetLastUpdated get last updated at for validator
func (k *Keeper) GetLastUpdated(ctx context.Context, valID uint64) (updatedAt string, found bool) {
	// get validator
	validator, ok := k.GetValidatorFromValID(ctx, valID)
	if !ok {
		return "", false
	}

	return validator.LastUpdated, true
}

// // IterateCurrentValidatorsAndApplyFn iterate through current validators
// func (k *Keeper) IterateCurrentValidatorsAndApplyFn(ctx context.Context, f func(validator *hmTypes.Validator) bool) {
// 	currentValidatorSet := k.GetValidatorSet(ctx)
// 	for _, v := range currentValidatorSet.Validators {
// 		if stop := f(v); stop {
// 			return
// 		}
// 	}
// }

//
// Staking sequence
//

// SetStakingSequence sets staking sequence
func (k *Keeper) SetStakingSequence(ctx context.Context, sequence string) {
	store := k.storeService.OpenKVStore(ctx)

	store.Set(types.GetStakingSequenceKey(sequence), types.DefaultValue)
}

// HasStakingSequence checks if staking sequence already exists
func (k *Keeper) HasStakingSequence(ctx context.Context, sequence string) bool {
	store := k.storeService.OpenKVStore(ctx)
	key := types.GetStakingSequenceKey(sequence)

	bz, err := store.Get(key)
	if bz == nil || err != nil {
		return false
	}

	return true
}

// GetStakingSequences checks if Staking already exists
func (k *Keeper) GetStakingSequences(ctx context.Context) (sequences []string) {
	k.IterateStakingSequencesAndApplyFn(ctx, func(sequence string) error {
		sequences = append(sequences, sequence)
		return nil
	})

	return
}

// IterateStakingSequencesAndApplyFn iterate validators and apply the given function.
func (k *Keeper) IterateStakingSequencesAndApplyFn(ctx context.Context, f func(sequence string) error) {
	store := k.storeService.OpenKVStore(ctx)

	// get validator iterator
	iterator, err := store.Iterator(types.ValidatorsKey, storetypes.PrefixEndBytes(types.ValidatorsKey))
	defer iterator.Close()

	if err != nil {
		k.Logger(ctx).Error("Error in getting iterator for validators")
		return
	}

	// loop through validators to get valid validators
	for ; iterator.Valid(); iterator.Next() {
		sequence := string(iterator.Key()[len(types.StakingSequenceKey):])

		// call function and return if required
		if err := f(sequence); err != nil {
			return
		}
	}
}

// GetValIdFromAddress returns a validator's id given its address string
func (k *Keeper) GetValIdFromAddress(ctx context.Context, address string) (uint64, error) {
	// get ack count
	ackCount := k.moduleCommunicator.GetACKCount(ctx)

	address = strings.ToLower(address)

	// get validator info
	validator, err := k.GetValidatorInfo(ctx, address)
	if err != nil {
		return 0, err
	}

	// check if validator is current validator
	if validator.IsCurrentValidator(ackCount) {
		return validator.ValId, nil
	}

	return 0, errors.New("Address not found in current validator set")
}

// TODO H2 Please how to use the stop parameter here
// IterateCurrentValidatorsAndApplyFn iterate through current validators
func (k Keeper) IterateCurrentValidatorsAndApplyFn(ctx context.Context, f func(validator cosmosTypes.ValidatorI) bool) error {
	currentValidatorSet := k.GetValidatorSet(ctx)
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
	validatorSet := k.GetMilestoneValidatorSet(ctx)

	// increment accum
	validatorSet.IncrementProposerPriority(times)

	// replace

	if err := k.UpdateMilestoneValidatorSetInStore(ctx, validatorSet); err != nil {
		k.Logger(ctx).Error("IncrementAccum | UpdateValidatorSetInStore", "error", err)
	}
}

// GetMilestoneValidatorSet returns current milestone Validator Set from store
func (k *Keeper) GetMilestoneValidatorSet(ctx context.Context) (validatorSet hmTypes.ValidatorSet) {
	store := k.storeService.OpenKVStore(ctx)

	var bz []byte

	bz, err := store.Get(types.CurrentMilestoneValidatorSetKey)
	if bz == nil {
		bz, err = store.Get(types.CurrentValidatorSetKey)
	}

	if err != nil {
		k.Logger(ctx).Error("GetMilestoneValidatorSet | UnmarshalBinaryBare", "error", err)
		return validatorSet
	}

	// unmarhsall
	if err := k.cdc.Unmarshal(bz, &validatorSet); err != nil {
		k.Logger(ctx).Error("GetMilestoneValidatorSet | UnmarshalBinaryBare", "error", err)
	}

	// return validator set
	return validatorSet
}

// UpdateMilestoneValidatorSetInStore adds milestone validator set to store
func (k *Keeper) UpdateMilestoneValidatorSetInStore(ctx context.Context, newValidatorSet hmTypes.ValidatorSet) error {
	// TODO check if we may have to delay this by 1 height to sync with tendermint validator updates
	store := k.storeService.OpenKVStore(ctx)

	// marshall validator set
	bz, err := k.cdc.Marshal(&newValidatorSet)
	if err != nil {
		return err
	}

	// set validator set with CurrentMilestoneValidatorSetKey as key in store
	store.Set(types.CurrentMilestoneValidatorSetKey, bz)

	return nil
}

// GetMilestoneCurrentProposer returns current proposer
func (k *Keeper) GetMilestoneCurrentProposer(ctx context.Context) *hmTypes.Validator {
	// get validator set
	validatorSet := k.GetMilestoneValidatorSet(ctx)

	// return get proposer
	return validatorSet.GetProposer()
}

// ValidatorAddressCodec return the validator address codec
func (k *Keeper) ValidatorAddressCodec() addresscodec.Codec {
	return k.validatorAddressCodec
}

////////////////////////    Slashing Code //////////////////////////////
// // Slashing api's
// // AddValidatorSigningInfo creates a signing info for validator
// func (k *Keeper) AddValidatorSigningInfo(ctx context.Context, valID hmTypes.ValidatorID, valSigningInfo hmTypes.ValidatorSigningInfo) error {
// 	k.moduleCommunicator.CreateValidatorSigningInfo(ctx, valID, valSigningInfo)
// 	return nil
// }

// // UpdatePower updates validator with signer and pubkey + validator => signer map
// func (k *Keeper) Slash(ctx context.Context, valSlashingInfo hmTypes.ValidatorSlashingInfo) error {
// 	// get validator from state
// 	validator, found := k.GetValidatorFromValID(ctx, valSlashingInfo.ID)
// 	if !found {
// 		k.Logger(ctx).Error("Unable to fetch validator from store")
// 		return errors.New("validator not found")
// 	}

// 	k.Logger(ctx).Debug("validator fetched", "validator", validator)

// 	updatedPower := int64(0)
// 	// calculate power after slash
// 	if validator.VotingPower >= int64(valSlashingInfo.SlashedAmount) {
// 		updatedPower = validator.VotingPower - int64(valSlashingInfo.SlashedAmount)
// 	}

// 	k.Logger(ctx).Info("slashAmount", valSlashingInfo.SlashedAmount, "prevPower", validator.VotingPower, "updatedPower", updatedPower)

// 	// update power and jail status.
// 	validator.VotingPower = updatedPower
// 	validator.Jailed = valSlashingInfo.IsJailed

// 	// add updated validator to store with new key
// 	if err := k.AddValidator(ctx, validator); err != nil {
// 		k.Logger(ctx).Error("Failed to add validator", "error", err)
// 	}

// 	k.Logger(ctx).Debug("updated validator with slashed voting power and jail status", "validator", validator)

// 	return nil
// }

// // Unjail a validator
// func (k *Keeper) Unjail(ctx context.Context, valID hmTypes.ValidatorID) {
// 	// get validator from state and make jailed = false
// 	validator, found := k.GetValidatorFromValID(ctx, valID)
// 	if !found {
// 		k.Logger(ctx).Error("Unable to fetch validator from store")
// 		return
// 	}

// 	if !validator.Jailed {
// 		k.Logger(ctx).Info("Already unjailed.")
// 		return
// 	}
// 	// unjail validator
// 	validator.Jailed = false

// 	// add updated validator to store with new key
// 	if err := k.AddValidator(ctx, validator); err != nil {
// 		k.Logger(ctx).Error("Failed to add validator", "Error", err)
// 	}
// }
