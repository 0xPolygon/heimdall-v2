package keeper

import (
	"bytes"
	"context"
	"fmt"

	"cosmossdk.io/collections"
	storetypes "cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	util "github.com/0xPolygon/heimdall-v2/common/hex"
	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
)

// Keeper of the x/milestone store
type Keeper struct {
	storeService storetypes.KVStoreService
	cdc          codec.BinaryCodec
	authority    string
	schema       collections.Schema

	IContractCaller helper.IContractCaller

	milestone          collections.Map[uint64, types.Milestone]
	params             collections.Item[types.Params]
	count              collections.Item[uint64]
	lastMilestoneBlock collections.Item[uint64]
}

// NewKeeper creates a new milestone Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec,
	authority string,
	storeService storetypes.KVStoreService,
	contractCaller helper.IContractCaller,
) Keeper {
	bz, err := address.NewHexCodec().StringToBytes(authority)
	if err != nil {
		panic(fmt.Errorf("invalid milestone authority address: %w", err))
	}

	// ensure only gov has the authority to update the params
	if !bytes.Equal(bz, authtypes.NewModuleAddress(govtypes.ModuleName)) {
		panic(fmt.Errorf("invalid milestone authority address: %s", authority))
	}

	sb := collections.NewSchemaBuilder(storeService)

	k := Keeper{
		storeService:    storeService,
		authority:       authority,
		cdc:             cdc,
		IContractCaller: contractCaller,

		milestone:          collections.NewMap(sb, types.MilestoneMapPrefixKey, "milestone", collections.Uint64Key, codec.CollValue[types.Milestone](cdc)),
		params:             collections.NewItem(sb, types.ParamsPrefixKey, "params", codec.CollValue[types.Params](cdc)),
		count:              collections.NewItem(sb, types.CountPrefixKey, "count", collections.Uint64Value),
		lastMilestoneBlock: collections.NewItem(sb, types.LastMilestoneBlockPrefixKey, "lastMilestoneBlock", collections.Uint64Value),
	}

	// build the schema and set it in the keeper
	s, err := sb.Build()
	if err != nil {
		panic(err)
	}
	k.schema = s

	return k
}

func (k Keeper) SetLastMilestoneBlock(ctx context.Context, block uint64) error {
	err := k.lastMilestoneBlock.Set(ctx, block)
	if err != nil {
		k.Logger(ctx).Error("error while setting last milestone block in store", "err", err)
		return err
	}

	return nil
}

func (k Keeper) GetLastMilestoneBlock(ctx context.Context) (uint64, error) {
	doExist, err := k.lastMilestoneBlock.Has(ctx)
	if err != nil {
		k.Logger(ctx).Error("error while checking the existence of last milestone block in store", "err", err)
		return 0, err
	}

	if !doExist {
		return 0, nil
	}
	block, err := k.lastMilestoneBlock.Get(ctx)
	if err != nil {
		k.Logger(ctx).Error("error while getting last milestone block from store", "err", err)
		return 0, err
	}

	return block, nil
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx context.Context) log.Logger {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.Logger().With("module", "x/"+types.ModuleName)
}

// GetAuthority returns x/milestone module's authority
func (k Keeper) GetAuthority() string {
	return k.authority
}

// SetParams sets the x/milestone module parameters.
func (k Keeper) SetParams(ctx context.Context, params types.Params) error {
	err := params.ValidateBasic()
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

	return params, nil
}

// AddMilestone adds a milestone to the store
func (k *Keeper) AddMilestone(ctx context.Context, milestone types.Milestone) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// GetMilestoneCount gives the number of previous milestone
	milestoneNumber, err := k.GetMilestoneCount(ctx)
	if err != nil {
		return err
	}

	milestoneNumber = milestoneNumber + 1

	milestone.Proposer = util.FormatAddress(milestone.Proposer)
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

	// emit milestone event
	sdkCtx.EventManager().EmitEvent(
		types.NewMilestoneEvent(milestone, milestoneNumber),
	)

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

// HasMilestone checks for existence of milestone
func (k *Keeper) HasMilestone(ctx context.Context) (bool, error) {
	lastMilestoneNumber, err := k.GetMilestoneCount(ctx)
	if err != nil {
		return false, err
	}

	if lastMilestoneNumber == 0 {
		return false, nil
	}

	return true, nil
}

// GetLastMilestone gets last milestone, where number = GetCount()
func (k *Keeper) GetLastMilestone(ctx context.Context) (*types.Milestone, error) {
	lastMilestoneNumber, err := k.GetMilestoneCount(ctx)
	if err != nil {
		return nil, err
	}

	if lastMilestoneNumber == 0 {
		k.Logger(ctx).Warn("no milestones found in store yet")
		return nil, types.ErrNoMilestoneFound
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
	doExist, err := k.count.Has(ctx)
	if err != nil {
		k.Logger(ctx).Error("error while checking the existence of milestone count in store", "err", err)
		return 0, err
	}

	if !doExist {
		return 0, nil
	}

	count, err := k.count.Get(ctx)
	if err != nil {
		k.Logger(ctx).Error("error while fetching milestone count in store", "err", err)
		return 0, err
	}

	return count, nil
}

// GetMilestones gets all milestones
func (k *Keeper) GetMilestones(ctx context.Context) ([]types.Milestone, error) {
	iterator, err := k.milestone.Iterate(ctx, nil)
	if err != nil {
		k.Logger(ctx).Error("error in getting the iterator", "err", err)
		return nil, err
	}
	defer func(iterator collections.Iterator[uint64, types.Milestone]) {
		err := iterator.Close()
		if err != nil {
			k.Logger(ctx).Error("error in closing iterator", "err", err)
		}
	}(iterator)

	milestones, err := iterator.Values()
	if err != nil {
		k.Logger(ctx).Error("error in getting the iterator values", "err", err)
		return nil, err
	}

	return milestones, nil
}
