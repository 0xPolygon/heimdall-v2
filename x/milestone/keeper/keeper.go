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

	util "github.com/0xPolygon/heimdall-v2/common/address"
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

	milestone collections.Map[uint64, types.Milestone]
	params    collections.Item[types.Params]
	count     collections.Item[uint64]
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

		milestone: collections.NewMap(sb, types.MilestoneMapPrefixKey, "milestone", collections.Uint64Key, codec.CollValue[types.Milestone](cdc)),
		params:    collections.NewItem(sb, types.ParamsPrefixKey, "params", codec.CollValue[types.Params](cdc)),
		count:     collections.NewItem(sb, types.CountPrefixKey, "count", collections.Uint64Value),
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

// GetAuthority returns x/milestone module's authority
func (k Keeper) GetAuthority() string {
	return k.authority
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
		k.Logger(ctx).Error("milestone doesn't exist in store")
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
