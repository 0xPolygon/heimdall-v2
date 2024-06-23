package keeper

import (
	"context"

	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
)

// AddCheckpoint adds checkpoint into final blocks
func (k *Keeper) AddCheckpoint(ctx context.Context, checkpointNumber uint64, checkpoint types.Checkpoint) error {
	err := k.checkpoint.Set(ctx, checkpointNumber, checkpoint)
	if err != nil {
		k.Logger(ctx).Error("error in setting the checkpoint in store", "error", err)
		return err
	}

	return nil
}

// SetCheckpointBuffer flushes Checkpoint Buffer
func (k *Keeper) SetCheckpointBuffer(ctx context.Context, checkpoint types.Checkpoint) error {
	err := k.bufferedCheckpoint.Set(ctx, &checkpoint)
	if err != nil {
		k.Logger(ctx).Error("error in setting the buffered checkpoint in store", "error", err)
		return err
	}

	return nil
}

// addCheckpoint adds checkpoint to store
func (k *Keeper) addCheckpoint(ctx context.Context, key []byte, checkpoint types.Checkpoint) error {
	store := k.storeService.OpenKVStore(ctx)

	// create Checkpoint block and marshall
	out, err := k.cdc.Marshal(&checkpoint)
	if err != nil {
		k.Logger(ctx).Error("Error marshalling checkpoint", "error", err)
		return err
	}

	// store in key provided
	store.Set(key, out)

	return nil
}

// GetCheckpointByNumber to get checkpoint by checkpoint number
func (k *Keeper) GetCheckpointByNumber(ctx context.Context, number uint64) (types.Checkpoint, error) {
	checkpoint, err := k.checkpoint.Get(ctx, number)
	if err != nil {
		return checkpoint, err
	}

	return checkpoint, nil
}

// GetLastCheckpoint gets last checkpoint, checkpoint number = TotalACKs
func (k *Keeper) GetLastCheckpoint(ctx context.Context) (types.Checkpoint, error) {
	acksCount := k.GetACKCount(ctx)

	lastCheckpointNumber := acksCount

	checkpoint, err := k.checkpoint.Get(ctx, lastCheckpointNumber)
	if err != nil {
		k.Logger(ctx).Error("error while fetching last checkpoint from store", "err", err)
		return checkpoint, err
	}

	return checkpoint, nil
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
	k.bufferedCheckpoint.Remove(ctx)
}

// GetCheckpointFromBuffer gets checkpoint in buffer
func (k *Keeper) GetCheckpointFromBuffer(ctx context.Context) (*types.Checkpoint, error) {
	checkpoint, err := k.bufferedCheckpoint.Get(ctx)
	if err != nil {
		return checkpoint, err
	}

	return checkpoint, nil
}

// SetLastNoAck set last no-ack object
func (k *Keeper) SetLastNoAck(ctx context.Context, timestamp uint64) error {
	return k.lastNoAck.Set(ctx, timestamp)
}

// GetLastNoAck returns last no ack
func (k *Keeper) GetLastNoAck(ctx context.Context) uint64 {
	res, err := k.lastNoAck.Get(ctx)
	if err != nil {
		return uint64(0)
	}

	return res
}

// GetCheckpoints get checkpoint all checkpoints
func (k *Keeper) GetCheckpoints(ctx context.Context) ([]types.Checkpoint, error) {
	var checkpoints []types.Checkpoint

	iterator, err := k.checkpoint.Iterate(ctx, nil)
	if err != nil {
		k.Logger(ctx).Error("error in getting the iterator", "err", err)
		return checkpoints, err
	}

	defer iterator.Close()

	var checkpoint types.Checkpoint

	// loop through validators to get valid validators
	for ; iterator.Valid(); iterator.Next() {
		checkpoint, err = iterator.Value()
		if err != nil {
			k.Logger(ctx).Error("error while getting checkpoint from iterator", "err", err)
			return checkpoints, err
		}
		checkpoints = append(checkpoints, checkpoint)
	}

	return checkpoints, nil
}

//
// Ack count
//

// GetACKCount returns current ACK count
func (k Keeper) GetACKCount(ctx context.Context) uint64 {
	res, err := k.ackCount.Get(ctx)
	if err != nil {
		return uint64(0)
	}

	return res
}

// UpdateACKCountWithValue updates ACK with value
func (k Keeper) UpdateACKCountWithValue(ctx context.Context, value uint64) error {
	return k.ackCount.Set(ctx, value)
}

// UpdateACKCount updates ACK count by 1
func (k Keeper) UpdateACKCount(ctx context.Context) error {
	// get current ACK Count
	ackCount := k.GetACKCount(ctx)

	return k.ackCount.Set(ctx, ackCount+1)
}
