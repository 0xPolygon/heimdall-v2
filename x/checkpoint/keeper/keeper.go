package keeper

import (
	"context"

	"cosmossdk.io/collections"
	storetypes "cosmossdk.io/core/store"
	"cosmossdk.io/log"

	"github.com/0xPolygon/heimdall-v2/helper"
	stakeKeeper "github.com/0xPolygon/heimdall-v2/x/stake/keeper"

	cmKeeper "github.com/0xPolygon/heimdall-v2/x/chainmanager/keeper"
	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Keeper of the x/checkpoint store
type Keeper struct {
	storeService storetypes.KVStoreService
	cdc          codec.BinaryCodec
	schema       collections.Schema

	authority       string
	sk              stakeKeeper.Keeper
	ck              cmKeeper.Keeper
	topupKeeper     types.TopupKeeper
	IContractCaller helper.IContractCaller

	checkpoint         collections.Map[uint64, types.Checkpoint]
	bufferedCheckpoint collections.Item[*types.Checkpoint]
	params             collections.Item[types.Params]
	lastNoAck          collections.Item[uint64]
	ackCount           collections.Item[uint64]
}

// NewKeeper creates a new checkpoint Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec,
	storeService storetypes.KVStoreService,
	authority string,
	stakingKeeper stakeKeeper.Keeper,
	cmKeeper cmKeeper.Keeper,
	topupKeeper types.TopupKeeper,
	contractCaller helper.IContractCaller,

) *Keeper {
	sb := collections.NewSchemaBuilder(storeService)

	k := &Keeper{
		storeService:    storeService,
		cdc:             cdc,
		authority:       authority,
		sk:              stakingKeeper,
		ck:              cmKeeper,
		topupKeeper:     topupKeeper,
		IContractCaller: contractCaller,

		bufferedCheckpoint: collections.NewItem(sb, types.BufferedCheckpointPrefixKey, "buffered_checkpoint", codec.CollValue[types.Checkpoint](cdc)),
		checkpoint:         collections.NewMap(sb, types.CheckpointMapPrefixKey, "checkpoint", collections.Uint64Key, codec.CollValue[types.Checkpoint](cdc)),
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx context.Context) log.Logger {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.Logger().With("module", "x/"+types.ModuleName)
}

// SetParams sets the x/checkpoint module parameters.
func (k Keeper) SetParams(ctx context.Context, params types.Params) error {
	err := k.params.Set(ctx, params)
	if err != nil {
		k.Logger(ctx).Error("error in setting the checkpoint params", "error", err)
		return err
	}

	return nil
}

// GetParams gets the x/checkpoint module parameters.
func (k Keeper) GetParams(ctx context.Context) (params types.Params, err error) {
	params, err = k.params.Get(ctx)
	if err != nil {
		k.Logger(ctx).Error("error in fetching the checkpoint params", "error", err)
		return params, err
	}

	return params, err
}

// AddCheckpoint adds checkpoint into the db store
func (k *Keeper) AddCheckpoint(ctx context.Context, checkpointNumber uint64, checkpoint types.Checkpoint) error {
	err := k.checkpoint.Set(ctx, checkpointNumber, checkpoint)
	if err != nil {
		k.Logger(ctx).Error("error in setting the checkpoint in store", "error", err)
		return err
	}

	return nil
}

// SetCheckpointBuffer sets the checkpoint in buffer
func (k *Keeper) SetCheckpointBuffer(ctx context.Context, checkpoint types.Checkpoint) error {
	err := k.bufferedCheckpoint.Set(ctx, &checkpoint)
	if err != nil {
		k.Logger(ctx).Error("error in setting the buffered checkpoint in store", "error", err)
		return err
	}

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

// GetLastCheckpoint gets last checkpoint, last checkpoint number is equal to total checkpoint ack count
func (k *Keeper) GetLastCheckpoint(ctx context.Context) (checkpoint types.Checkpoint, err error) {
	acksCount, err := k.GetACKCount(ctx)
	if err != nil {
		k.Logger(ctx).Error("error while calling the get ack count", "err", err)
		return checkpoint, err
	}

	lastCheckpointNumber := acksCount

	checkpoint, err = k.checkpoint.Get(ctx, lastCheckpointNumber)
	if err != nil {
		k.Logger(ctx).Error("error while fetching last checkpoint from store", "err", err)
		return checkpoint, err
	}

	return checkpoint, nil
}

// FlushCheckpointBuffer flushes Checkpoint Buffer
func (k *Keeper) FlushCheckpointBuffer(ctx context.Context) {
	k.bufferedCheckpoint.Remove(ctx)
}

// GetCheckpointFromBuffer gets buffered checkpoint from store
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
func (k *Keeper) GetLastNoAck(ctx context.Context) (uint64, error) {
	doExist, err := k.lastNoAck.Has(ctx)
	if err != nil {
		k.Logger(ctx).Error("error while checking the last no-ack in store", "err", err)
		return uint64(0), err
	}

	if !doExist {
		return uint64(0), nil
	}

	res, err := k.lastNoAck.Get(ctx)

	if err != nil {
		k.Logger(ctx).Error("error while fetching the last no-ack from store", "err", err)
		return uint64(0), err
	}

	return res, nil
}

// GetCheckpoints get all the checkpoints from the store
func (k *Keeper) GetCheckpoints(ctx context.Context) ([]types.Checkpoint, error) {
	var checkpoints []types.Checkpoint

	iterator, err := k.checkpoint.Iterate(ctx, nil)
	if err != nil {
		k.Logger(ctx).Error("error in getting the iterator", "err", err)
		return checkpoints, err
	}

	defer func() {
		err := iterator.Close()
		if err != nil {
			k.Logger(ctx).Error("error in closing the checkpoint iterator", "error", err)
		}
	}()

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

// GetACKCount returns current ack count
func (k Keeper) GetACKCount(ctx context.Context) (uint64, error) {
	doExist, err := k.ackCount.Has(ctx)
	if err != nil {
		k.Logger(ctx).Error("error while checking the ack count in store", "err", err)
		return uint64(0), err
	}

	if !doExist {
		return uint64(0), nil
	}

	res, err := k.ackCount.Get(ctx)

	if err != nil {
		k.Logger(ctx).Error("error while fetching the ack count in store", "err", err)
		return uint64(0), err
	}

	return res, nil
}

// UpdateACKCountWithValue updates ACK with value
func (k Keeper) UpdateACKCountWithValue(ctx context.Context, value uint64) error {
	return k.ackCount.Set(ctx, value)
}

// UpdateACKCount updates ACK count by 1
func (k Keeper) UpdateACKCount(ctx context.Context) error {
	// get current ACK Count
	ackCount, err := k.GetACKCount(ctx)
	if err != nil {
		return nil
	}

	return k.ackCount.Set(ctx, ackCount+1)
}
