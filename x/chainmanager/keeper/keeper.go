package keeper

import (
	"context"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	"cosmossdk.io/log"

	"github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Keeper stores all chainmanager related data
type Keeper struct {
	cdc codec.BinaryCodec
	// TODO HV2: clarify whether using KVStoreService instead of StoreKey is fine
	// storeKey storetypes.StoreKey
	storeService store.KVStoreService
	Params       collections.Item[types.Params]
}

// NewKeeper create new keeper
func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	// storeKey storetypes.StoreKey,
) Keeper {
	sb := collections.NewSchemaBuilder(storeService)
	return Keeper{
		cdc: cdc,
		// storeKey: storeKey,
		storeService: storeService,
		Params:       collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx context.Context) log.Logger {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.Logger().With("module", "x/"+types.ModuleName)
}

// -----------------------------------------------------------------------------
// Params

// SetParams sets the chainmanager module's parameters.
func (k Keeper) SetParams(ctx context.Context, params types.Params) error {
	return k.Params.Set(ctx, params)
}

// GetParams gets the chainmanager module's parameters.
func (k Keeper) GetParams(ctx context.Context) (params types.Params) {
	p, _ := k.Params.Get(ctx)
	return p
}
