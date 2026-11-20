package keeper

import (
	"context"
	"errors"
	"fmt"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
)

func (k *Keeper) flushExpiredBuffer(ctx context.Context) error {
	if helper.IsPurley(sdk.UnwrapSDKContext(ctx).BlockHeight()) {
		// An expired approved buffer can still receive an L1 ACK.
		return nil
	}
	return k.FlushCheckpointBuffer(ctx)
}

// checkBufferReplacement fails unless the buffer is empty or, from Purley, expired at now.
func (k *Keeper) checkBufferReplacement(ctx sdk.Context, buffered types.Checkpoint, exists, purley bool, now uint64) error {
	if !exists || IsBufferedCheckpointZero(buffered) {
		return nil
	}
	params, err := k.GetParams(ctx)
	if err != nil {
		return err
	}
	bufferTime := uint64(params.CheckpointBufferTime.Seconds())
	if purley && isCheckpointBufferExpired(buffered, now, bufferTime) {
		return nil
	}
	return fmt.Errorf("checkpoint already exists in buffer")
}

// checkReplaceableBuffer applies checkBufferReplacement to the stored buffer at the block's time.
func (k *Keeper) checkReplaceableBuffer(ctx sdk.Context) error {
	buffered, err := k.GetCheckpointFromBuffer(ctx)
	if err != nil {
		return err
	}
	return k.checkBufferReplacement(ctx, buffered, true, true, uint64(ctx.BlockTime().Unix()))
}

// checkVotedBufferReplacement is checkBufferReplacement for a transaction voted on at txHeight.
// Voters at txHeight read the state committed at txHeight-1, so they never judged a buffer installed
// at txHeight or later, and their YES can't stand in for its expiry.
func (k *Keeper) checkVotedBufferReplacement(ctx sdk.Context, buffered types.Checkpoint, exists bool, txHeight int64, now uint64) error {
	purley := helper.IsPurley(txHeight)
	if err := k.checkBufferReplacement(ctx, buffered, exists, purley, now); err != nil {
		return err
	}
	if !purley || !exists || IsBufferedCheckpointZero(buffered) {
		return nil
	}
	installedAt, err := k.bufferHeight.Get(ctx)
	if errors.Is(err, collections.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if installedAt >= uint64(txHeight) {
		return fmt.Errorf("checkpoint buffer installed at height %d, after the vote at height %d", installedAt, txHeight)
	}
	return nil
}

// recordBufferHeight stores the height the buffer was installed at, from Purley only.
func (k *Keeper) recordBufferHeight(ctx context.Context) error {
	height := sdk.UnwrapSDKContext(ctx).BlockHeight()
	if !helper.IsPurley(height) {
		return nil
	}
	return k.bufferHeight.Set(ctx, uint64(height))
}
