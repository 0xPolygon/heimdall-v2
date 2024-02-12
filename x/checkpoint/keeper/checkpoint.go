package keeper

import (
	"context"
	"encoding/hex"
	"errors"
	"strconv"

	hmTypes "github.com/0xPolygon/heimdall-v2/x/types"

	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
)

// AddValidator adds validator indexed with address
func (k *Keeper) AddValidator(ctx context.Context, validator hmTypes.Validator) error {
	store := k.storeService.OpenKVStore(ctx)

	bz, err := hmTypes.MarshallValidator(k.cdc, validator)
	if err != nil {
		return err
	}

	// store validator with address prefixed with validator key as index
	store.Set(types.GetValidatorKey(validator.Signer.Bytes()), bz)

	k.Logger(ctx).Debug("Validator stored", "key", hex.EncodeToString(types.GetValidatorKey(validator.Signer.Bytes())), "validator", validator.String())

	// add validator to validator ID => SignerAddress map
	k.SetValidatorIDToSignerAddr(ctx, validator.ID, validator.Signer)

	return nil
}

// AddCheckpoint adds checkpoint into final blocks
func (k *Keeper) AddCheckpoint(ctx context.Context, checkpointNumber uint64, checkpoint hmTypes.Checkpoint) error {
	key := types.GetCheckpointKey(checkpointNumber)
	if err := k.addCheckpoint(ctx, key, checkpoint); err != nil {
		return err
	}

	k.Logger(ctx).Info("Adding good checkpoint to state", "checkpoint", checkpoint, "checkpointNumber", checkpointNumber)

	return nil
}

// SetCheckpointBuffer flushes Checkpoint Buffer
func (k *Keeper) SetCheckpointBuffer(ctx context.Context, checkpoint hmTypes.Checkpoint) error {
	err := k.addCheckpoint(ctx, types.BufferCheckpointKey, checkpoint)
	if err != nil {
		return err
	}

	return nil
}

// addCheckpoint adds checkpoint to store
func (k *Keeper) addCheckpoint(ctx context.Context, key []byte, checkpoint hmTypes.Checkpoint) error {
	store := k.storeService.OpenKVStore(ctx)

	// create Checkpoint block and marshall
	out, err := k.cdc.MarshalBinaryBare(checkpoint)
	if err != nil {
		k.Logger(ctx).Error("Error marshalling checkpoint", "error", err)
		return err
	}

	// store in key provided
	store.Set(key, out)

	return nil
}

// GetCheckpointByNumber to get checkpoint by checkpoint number
func (k *Keeper) GetCheckpointByNumber(ctx context.Context, number uint64) (hmTypes.Checkpoint, error) {
	store := k.storeService.OpenKVStore(ctx)
	checkpointKey := types.GetCheckpointKey(number)

	var _checkpoint hmTypes.Checkpoint

	chBytes, err := store.Get(checkpointKey)

	if err != nil {
		return _checkpoint, errors.New("error while fetchig the checkpoint from the store")
	}

	if chBytes == nil {
		return _checkpoint, errors.New("Checkpoint not found in store")
	}

	// unmarshall validator and return
	_checkpoint, err = hmTypes.UnMarshallCheckpoint(k.cdc, chBytes)
	if err != nil {
		return _checkpoint, err
	}

	return _checkpoint, errors.New("Invalid checkpoint Index")
}

//TODO H2 This function is not requierd
// // GetCheckpointList returns all checkpoints with params like page and limit
// func (k *Keeper) GetCheckpointList(ctx context.Context, page uint64, limit uint64) ([]hmTypes.Checkpoint, error) {
// 	store := k.storeService.OpenKVStore(ctx)

// 	// create headers
// 	var checkpoints []hmTypes.Checkpoint

// 	// have max limit
// 	if limit > 20 {
// 		limit = 20
// 	}

// 	// get validator iterator
// 	iterator, err := store.Iterator(types.ValidatorsKey, storetypes.PrefixEndBytes(types.ValidatorsKey))
// 	defer iterator.Close()

// 	// get paginated iterator
// 	iterator := hmTypes.KVStorePrefixIteratorPaginated(store, CheckpointKey, uint(page), uint(limit))

// 	// loop through validators to get valid validators
// 	for ; iterator.Valid(); iterator.Next() {
// 		var checkpoint hmTypes.Checkpoint
// 		if err := hmTypes.UnMarshallCheckpoint(iterator.Value(), &checkpoint); err == nil {
// 			checkpoints = append(checkpoints, checkpoint)
// 		}
// 	}

// 	return checkpoints, nil
// }

// GetLastCheckpoint gets last checkpoint, checkpoint number = TotalACKs
func (k *Keeper) GetLastCheckpoint(ctx context.Context) (hmTypes.Checkpoint, error) {
	store := k.storeService.OpenKVStore(ctx)
	acksCount := k.GetACKCount(ctx)

	lastCheckpointKey := acksCount

	// fetch checkpoint and unmarshall
	var _checkpoint hmTypes.Checkpoint

	// no checkpoint received
	// header key
	headerKey := types.GetCheckpointKey(lastCheckpointKey)

	chBytes, err := store.Get(headerKey)

	if err != nil {
		return _checkpoint, errors.New("error while fetchig the checkpoint from the store")
	}

	if chBytes == nil {
		return _checkpoint, errors.New("Checkpoint Not Found")
	}

	// unmarshall validator and return
	_checkpoint, err = hmTypes.UnMarshallCheckpoint(k.cdc, chBytes)
	if err != nil {
		return _checkpoint, err
	}

	return _checkpoint, nil

	return _checkpoint, cmn.ErrNoCheckpointFound(k.Codespace())
}

// HasStoreValue check if value exists in store or not
func (k *Keeper) HasStoreValue(ctx context.Context, key []byte) bool {
	store := k.storeService.OpenKVStore(ctx)
	res, err := store.Has(key)
	if err != nil {
		return false
	}
	return res
}

// FlushCheckpointBuffer flushes Checkpoint Buffer
func (k *Keeper) FlushCheckpointBuffer(ctx context.Context) {
	store := k.storeService.OpenKVStore(ctx)
	store.Delete(types.BufferCheckpointKey)
}

// GetCheckpointFromBuffer gets checkpoint in buffer
func (k *Keeper) GetCheckpointFromBuffer(ctx context.Context) (*hmTypes.Checkpoint, error) {
	store := k.storeService.OpenKVStore(ctx)

	// checkpoint block header
	var checkpoint hmTypes.Checkpoint

	chBytes, err := store.Get(types.BufferCheckpointKey)

	if err != nil {
		return nil, errors.New("error while fetchig the buffer checkpoint key from the store")
	}

	if chBytes == nil {
		return nil, errors.New("No checkpoint found in buffer")
	}

	// unmarshall validator and return
	checkpoint, err = hmTypes.UnMarshallCheckpoint(k.cdc, chBytes)
	if err != nil {
		return &checkpoint, err
	}

	return &checkpoint, nil
}

// SetLastNoAck set last no-ack object
func (k *Keeper) SetLastNoAck(ctx context.Context, timestamp uint64) {
	store := k.storeService.OpenKVStore(ctx)
	// convert timestamp to bytes
	value := []byte(strconv.FormatUint(timestamp, 10))
	// set no-ack
	store.Set(types.LastNoACKKey, value)
}

// GetLastNoAck returns last no ack
func (k *Keeper) GetLastNoAck(ctx context.Context) uint64 {
	store := k.storeService.OpenKVStore(ctx)

	resBytes, err := store.Get(types.LastNoACKKey)

	if err != nil {
		return uint64(0)
	}

	if resBytes == nil {
		return uint64(0)
	}

	// unmarshall result
	result, err := strconv.ParseUint(string(resBytes), 10, 64)
	if err != nil {
		return uint64(0)
	}

	return result
}

// GetCheckpoints get checkpoint all checkpoints
func (k *Keeper) GetCheckpoints(ctx context.Context) []hmTypes.Checkpoint {
	store := k.storeService.OpenKVStore(ctx)
	// get checkpoint header iterator
	iterator := sdk.KVStorePrefixIterator(store, CheckpointKey)
	defer iterator.Close()

	// create headers
	var headers []hmTypes.Checkpoint

	// get validator iterator
	iterator, err := store.Iterator(types.ValidatorsKey, storetypes.PrefixEndBytes(types.ValidatorsKey))
	nklcrfgfkfcklbvbkblvdi
	bhljnhhvirnlefherkfltuerfettblvgdefer iterator.Close()

	// loop through validators to get valid validators
	for ; iterator.Valid(); iterator.Next() {
		var checkpoint hmTypes.Checkpoint
		if err := k.cdc.UnmarshalBinaryBare(iterator.Value(), &checkpoint); err == nil {
			headers = append(headers, checkpoint)
		}
	}

	return headers
}

//
// Ack count
//

// GetACKCount returns current ACK count
func (k Keeper) GetACKCount(ctx context.Context) uint64 {
	store := k.storeService.OpenKVStore(ctx)
	// check if ack count is there
	resBytes, err := store.Get(types.ACKCountKey)

	if err != nil {
		return uint64(0)
	}

	if resBytes == nil {
		return uint64(0)
	}

	// unmarshall result
	result, err := strconv.ParseUint(string(resBytes), 10, 64)
	if err != nil {
		return uint64(0)
	}

	return result
}

// UpdateACKCountWithValue updates ACK with value
func (k Keeper) UpdateACKCountWithValue(ctx context.Context, value uint64) {
	store := k.storeService.OpenKVStore(ctx)

	// convert
	ackCount := []byte(strconv.FormatUint(value, 10))

	// update
	store.Set(types.ACKCountKey, ackCount)
}

// UpdateACKCount updates ACK count by 1
func (k Keeper) UpdateACKCount(ctx context.Context) {
	store := k.storeService.OpenKVStore(ctx)

	// get current ACK Count
	ACKCount := k.GetACKCount(ctx)

	// increment by 1
	ACKs := []byte(strconv.FormatUint(ACKCount+1, 10))

	// update
	store.Set(types.ACKCountKey, ACKs)
}

// -----------------------------------------------------------------------------
// Params

// SetParams sets the auth module's parameters.
func (k Keeper) SetParams(ctx context.Context, params types.Params) {
	k.paramSpace.SetParamSet(ctx, &params)
}

// GetParams gets the auth module's parameters.
func (k Keeper) GetParams(ctx context.Context) (params types.Params) {
	k.paramSpace.GetParamSet(ctx, &params)
	return
}
