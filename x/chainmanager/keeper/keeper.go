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
	cdc          codec.BinaryCodec
	storeService store.KVStoreService
	Schema       collections.Schema
	params       collections.Item[types.Params]
}

// NewKeeper create new keeper
func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
) Keeper {
	sb := collections.NewSchemaBuilder(storeService)

	k := Keeper{
		cdc:          cdc,
		storeService: storeService,
		params:       collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}

	k.Schema = schema

	return k
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx context.Context) log.Logger {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.Logger().With("module", "x/"+types.ModuleName)
}

// Params

// SetParams sets the chainmanager module's parameters.
func (k Keeper) SetParams(ctx context.Context, params types.Params) error {
	return k.params.Set(ctx, params)
}

// GetParams gets the chainmanager module's parameters.
func (k Keeper) GetParams(ctx context.Context) (types.Params, error) {
	p, err := k.params.Get(ctx)
	if err != nil {
		return types.Params{}, err
	}
	return p, nil
}
