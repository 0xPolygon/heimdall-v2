package keeper

import (
	"context"

	"cosmossdk.io/collections"
	storetypes "cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
)

// Keeper of the x/checkpoint store
type Keeper struct {
	storeService storetypes.KVStoreService
	cdc          codec.BinaryCodec
	schema       collections.Schema

	sk              types.StakeKeeper
	ck              types.ChainManagerKeeper
	topupKeeper     types.TopupKeeper
	IContractCaller helper.IContractCaller

	checkpoint         collections.Map[uint64, types.Checkpoint]
	bufferedCheckpoint collections.Item[types.Checkpoint]
	params             collections.Item[types.Params]
	lastNoAck          collections.Item[uint64]
	ackCount           collections.Item[uint64]
}

// NewKeeper creates a new checkpoint Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec,
	storeService storetypes.KVStoreService,
	stakingKeeper types.StakeKeeper,
	cmKeeper types.ChainManagerKeeper,
	topupKeeper types.TopupKeeper,
	contractCaller helper.IContractCaller,

) Keeper {
	sb := collections.NewSchemaBuilder(storeService)

	k := Keeper{
		storeService:    storeService,
		cdc:             cdc,
		sk:              stakingKeeper,
		ck:              cmKeeper,
		topupKeeper:     topupKeeper,
		IContractCaller: contractCaller,

		bufferedCheckpoint: collections.NewItem(sb, types.BufferedCheckpointPrefixKey, "buffered_checkpoint", codec.CollValue[types.Checkpoint](cdc)),
		checkpoint:         collections.NewMap(sb, types.CheckpointMapPrefixKey, "checkpoint", collections.Uint64Key, codec.CollValue[types.Checkpoint](cdc)),
		params:             collections.NewItem(sb, types.ParamsPrefixKey, "params", codec.CollValue[types.Params](cdc)),
		lastNoAck:          collections.NewItem(sb, types.LastNoAckPrefixKey, "last_no_ack", collections.Uint64Value),
		ackCount:           collections.NewItem(sb, types.AckCountPrefixKey, "ack_count", collections.Uint64Value),
	}

	// build the schema and set it in the keeper
	s, err := sb.Build()
	if err != nil {
		panic(err)
	}
	k.schema = s

	return k
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
		k.Logger(ctx).Error("error in adding the checkpoint to the store", "error", err)
		return err
	}

	return nil
}

// SetCheckpointBuffer sets the checkpoint in buffer
func (k *Keeper) SetCheckpointBuffer(ctx context.Context, checkpoint types.Checkpoint) error {
	err := k.bufferedCheckpoint.Set(ctx, checkpoint)
	if err != nil {
		k.Logger(ctx).Error("error in setting the buffered checkpoint in the store", "error", err)
		return err
	}

	return nil
}

// GetCheckpointByNumber gets the checkpoint by its number
func (k *Keeper) GetCheckpointByNumber(ctx context.Context, number uint64) (types.Checkpoint, error) {
	checkpoint, err := k.checkpoint.Get(ctx, number)
	if err != nil {
		k.Logger(ctx).Error("error while fetching checkpoint from store", "err", err)
		return types.Checkpoint{}, err
	}

	return checkpoint, nil
}

// GetLastCheckpoint gets last checkpoint, where its number is equal to the ack count
func (k *Keeper) GetLastCheckpoint(ctx context.Context) (checkpoint types.Checkpoint, err error) {
	acksCount, err := k.GetAckCount(ctx)
	if err != nil {
		k.Logger(ctx).Error("error while fetching the ack count", "err", err)
		return types.Checkpoint{}, err
	}

	checkpoint, err = k.checkpoint.Get(ctx, acksCount)
	if err != nil {
		k.Logger(ctx).Error("error while fetching last checkpoint from store", "err", err)
		return types.Checkpoint{}, err
	}

	return checkpoint, nil
}

// FlushCheckpointBuffer flushes the checkpoint buffer
func (k *Keeper) FlushCheckpointBuffer(ctx context.Context) error {
	err := k.bufferedCheckpoint.Remove(ctx)
	if err != nil {
		k.Logger(ctx).Error("error in flushing the checkpoint buffer", "error", err)
		return err
	}
	return nil
}

// GetCheckpointFromBuffer gets the buffered checkpoint from store
func (k *Keeper) GetCheckpointFromBuffer(ctx context.Context) (types.Checkpoint, error) {
	checkpoint, err := k.bufferedCheckpoint.Get(ctx)
	if err != nil {
		k.Logger(ctx).Error("error while fetching the buffered checkpoint from store", "err", err)
		return types.Checkpoint{}, err
	}

	return checkpoint, nil
}

// HasCheckpointInBuffer checks if the buffered checkpoint exists in the store
func (k *Keeper) HasCheckpointInBuffer(ctx context.Context) (bool, error) {
	res, err := k.bufferedCheckpoint.Has(ctx)
	if err != nil {
		k.Logger(ctx).Error("error while checking the buffered checkpoint from store", "err", err)
		return false, err
	}

	return res, nil
}

// SetLastNoAck sets the last no-ack object
func (k *Keeper) SetLastNoAck(ctx context.Context, timestamp uint64) error {
	return k.lastNoAck.Set(ctx, timestamp)
}

// GetLastNoAck returns last no ack
func (k *Keeper) GetLastNoAck(ctx context.Context) (uint64, error) {
	exists, err := k.lastNoAck.Has(ctx)
	if err != nil {
		k.Logger(ctx).Error("error while checking for existence of the last no-ack in store", "err", err)
		return uint64(0), err
	}

	if !exists {
		return uint64(0), nil
	}

	res, err := k.lastNoAck.Get(ctx)
	if err != nil {
		k.Logger(ctx).Error("error while fetching the last no-ack from store", "err", err)
		return uint64(0), err
	}

	return res, nil
}

// GetCheckpoints gets all the checkpoints from the store
func (k *Keeper) GetCheckpoints(ctx context.Context) (checkpoints []types.Checkpoint, e error) {
	iterator, err := k.checkpoint.Iterate(ctx, nil)
	if err != nil {
		k.Logger(ctx).Error("error in getting the iterator", "err", err)
		return nil, err
	}

	defer func() {
		err := iterator.Close()
		if err != nil {
			k.Logger(ctx).Error("error in closing the checkpoint iterator", "error", err)
		}
		checkpoints = nil
		e = err
	}()

	var checkpoint types.Checkpoint

	for ; iterator.Valid(); iterator.Next() {
		checkpoint, err = iterator.Value()
		if err != nil {
			k.Logger(ctx).Error("error while getting checkpoint from iterator", "err", err)
			return nil, err
		}
		checkpoints = append(checkpoints, checkpoint)
	}

	return checkpoints, nil
}

// GetAckCount returns the current ack count
func (k Keeper) GetAckCount(ctx context.Context) (uint64, error) {
	exists, err := k.ackCount.Has(ctx)
	if err != nil {
		k.Logger(ctx).Error("error while checking for existence of the ack count in store", "err", err)
		return uint64(0), err
	}

	if !exists {
		return uint64(0), nil
	}

	res, err := k.ackCount.Get(ctx)

	if err != nil {
		k.Logger(ctx).Error("error while fetching the ack count from the store", "err", err)
		return uint64(0), err
	}

	return res, nil
}

// UpdateAckCountWithValue updates the ACK count with a value
func (k Keeper) UpdateAckCountWithValue(ctx context.Context, value uint64) error {
	return k.ackCount.Set(ctx, value)
}

// IncrementAckCount updates the ack count by 1
func (k Keeper) IncrementAckCount(ctx context.Context) error {
	// get current ACK Count
	ackCount, err := k.GetAckCount(ctx)
	if err != nil {
		return nil
	}

	return k.ackCount.Set(ctx, ackCount+1)
}
