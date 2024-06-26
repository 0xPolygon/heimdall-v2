package keeper

import (
	"context"

	"cosmossdk.io/collections"
	storetypes "cosmossdk.io/core/store"
	"cosmossdk.io/log"

	"github.com/0xPolygon/heimdall-v2/helper"
	stakeKeeper "github.com/0xPolygon/heimdall-v2/x/stake/keeper"

	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Keeper of the x/staking store
type Keeper struct {
	storeService    storetypes.KVStoreService
	cdc             codec.BinaryCodec
	authority       string
	sk              stakeKeeper.Keeper
	IContractCaller helper.IContractCaller

	milestone   collections.Map[uint64, types.Milestone]
	params      collections.Item[types.Params]
	blockNumber collections.Item[int64]
	count       collections.Item[uint64]
	timeout     collections.Item[uint64]
}

// NewKeeper creates a new staking Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec,
	storeService storetypes.KVStoreService,
	authority string,
	stakingKeeper stakeKeeper.Keeper,
	contractCaller helper.IContractCaller,

) *Keeper {
	return &Keeper{
		storeService:    storeService,
		cdc:             cdc,
		authority:       authority,
		sk:              stakingKeeper,
		IContractCaller: contractCaller,
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx context.Context) log.Logger {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.Logger().With("module", "x/"+types.ModuleName)
}

// SetParams sets the x/milestone module parameters.
func (k Keeper) SetParams(ctx context.Context, params types.Params) error {
	err := params.Validate()
	if err != nil {
		k.Logger(ctx).Error("error while validating params", "err", err)
		return err
	}

	err = k.params.Set(ctx, params)
	if err != nil {
		k.Logger(ctx).Error("error while storing params in store", "err", err)
		return err
	}

	return nil
}

// GetParams gets the x/Milestone module parameters.
func (k Keeper) GetParams(ctx context.Context) (params types.Params, err error) {
	params, err = k.params.Get(ctx)
	if err != nil {
		k.Logger(ctx).Error("error while fetching params from store", "err", err)
		return
	}

	return params, err
}

// AddMilestone adds a milestone to the store
func (k *Keeper) AddMilestone(ctx context.Context, milestone types.Milestone) error {
	// GetMilestoneCount gives the number of previous milestone
	milestoneNumber, err := k.GetMilestoneCount(ctx)
	if err != nil {
		return err
	}

	milestoneNumber = milestoneNumber + 1

	err = k.milestone.Set(ctx, milestoneNumber, milestone)
	if err != nil {
		k.Logger(ctx).Error("error while storing milestone in store", "err", err)
		return err
	}

	err = k.SetMilestoneCount(ctx, milestoneNumber)
	if err != nil {
		k.Logger(ctx).Error("error while storing milestone count in store", "err", err)
		return err
	}

	return nil
}

// GetMilestoneByNumber gets a milestone by its number
func (k *Keeper) GetMilestoneByNumber(ctx context.Context, number uint64) (*types.Milestone, error) {
	milestone, err := k.milestone.Get(ctx, number)
	if err != nil {
		k.Logger(ctx).Error("error while fetching milestone from store", "err", err)
		return nil, err
	}

	return &milestone, nil
}

// GetLastMilestone gets last milestone, where number = GetCount()
func (k *Keeper) DoLastMilestoneExist(ctx context.Context) (bool, error) {
	lastMilestoneNumber, err := k.GetMilestoneCount(ctx)
	if err != nil {
		return nil, err
	}

	milestone, err := k.milestone.Get(ctx, lastMilestoneNumber)
	if err != nil {
		k.Logger(ctx).Error("error while fetching milestone from store", "number", lastMilestoneNumber, "err", err)
		return nil, err
	}

	return &milestone, nil
}

// GetLastMilestone gets last milestone, where number = GetCount()
func (k *Keeper) GetLastMilestone(ctx context.Context) (*types.Milestone, error) {
	lastMilestoneNumber, err := k.GetMilestoneCount(ctx)
	if err != nil {
		return nil, err
	}

	milestone, err := k.milestone.Get(ctx, lastMilestoneNumber)
	if err != nil {
		k.Logger(ctx).Error("error while fetching milestone from store", "number", lastMilestoneNumber, "err", err)
		return nil, err
	}

	return &milestone, nil
}

// SetMilestoneCount sets the milestone's count number
func (k *Keeper) SetMilestoneCount(ctx context.Context, number uint64) error {
	err := k.count.Set(ctx, number)
	if err != nil {
		k.Logger(ctx).Error("error while setting milestone count in store", "err", err)
		return err
	}

	return nil
}

// GetMilestoneCount returns the milestone count
func (k *Keeper) GetMilestoneCount(ctx context.Context) (uint64, error) {
	count, err := k.count.Get(ctx)
	if err != nil {
		k.Logger(ctx).Error("error while fetching milestone count in store", "err", err)
		return 0, err
	}

	return count, nil
}

// SetMilestoneBlockNumber set the block number when the latest milestone enter the handler
func (k *Keeper) SetMilestoneBlockNumber(ctx context.Context, number int64) error {
	err := k.blockNumber.Set(ctx, number)
	if err != nil {
		k.Logger(ctx).Error("error while setting block number in store", "err", err)
		return err
	}

	return nil
}

// GetMilestoneBlockNumber returns the block number when the latest milestone enter the handler
func (k *Keeper) GetMilestoneBlockNumber(ctx context.Context) (int64, error) {
	number, err := k.blockNumber.Get(ctx)
	if err != nil {
		k.Logger(ctx).Error("error while fetching block number from store", "err", err)
		return 0, err
	}

	return number, nil
}

// SetNoAckMilestone sets the last no-ack milestone
func (k *Keeper) SetNoAckMilestone(ctx context.Context, milestoneId string) {
	store := k.storeService.OpenKVStore(ctx)

	milestoneNoAckKey := types.GetMilestoneNoAckKey(milestoneId)
	value := []byte(milestoneId)

	store.Set(milestoneNoAckKey, value)
	store.Set(types.MilestoneLastNoAckKey, value)
}

// GetLastNoAckMilestone returns the last no-ack milestone
func (k *Keeper) GetLastNoAckMilestone(ctx context.Context) string {
	store := k.storeService.OpenKVStore(ctx)
	lastNoAckBytes, err := store.Get(types.MilestoneLastNoAckKey)

	if err != nil || lastNoAckBytes == nil {
		return ""
	}

	return string(lastNoAckBytes)
}

// GetNoAckMilestone returns the last no-ack milestone
func (k *Keeper) GetNoAckMilestone(ctx context.Context, milestoneId string) bool {
	store := k.storeService.OpenKVStore(ctx)

	res, err := store.Has(types.GetMilestoneNoAckKey(milestoneId))

	if err != nil {
		return false
	}

	return res
}

// SetLastMilestoneTimeout set lastMilestone timeout time
func (k *Keeper) SetLastMilestoneTimeout(ctx context.Context, timestamp uint64) error {
	err := k.timeout.Set(ctx, timestamp)
	if err != nil {
		k.Logger(ctx).Error("error while setting milestone timeout in store", "err", err)
		return err
	}

	return nil
}

// GetLastMilestoneTimeout returns lastMilestone timeout time
func (k *Keeper) GetLastMilestoneTimeout(ctx context.Context) (uint64, error) {
	timeout, err := k.timeout.Get(ctx)
	if err != nil {
		k.Logger(ctx).Error("error while fetching milestone timeout from store", "err", err)
		return timeout, err
	}

	return timeout, nil
}
