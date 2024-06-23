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

	sequences collections.Map[string, bool]
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
	return &Keeper{
		storeService:    storeService,
		cdc:             cdc,
		authority:       authority,
		sk:              stakingKeeper,
		ck:              cmKeeper,
		topupKeeper:     topupKeeper,
		IContractCaller: contractCaller,
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
