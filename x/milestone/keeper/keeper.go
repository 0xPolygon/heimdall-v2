package keeper

import (
	"context"

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
// CONTRACT: This method performs no validation of the parameters.
func (k Keeper) SetParams(ctx context.Context, params types.Params) error {
	store := k.storeService.OpenKVStore(ctx)

	err := params.Validate()
	if err != nil {
		return err
	}

	bz, err := k.cdc.Marshal(&params)
	if err != nil {
		return err
	}
	return store.Set(types.ParamsKey, bz)
}

// GetParams gets the x/Milestone module parameters.
func (k Keeper) GetParams(ctx context.Context) (params types.Params, err error) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.ParamsKey)
	if err != nil {
		return params, err
	}

	if bz == nil {
		return params, nil
	}

	err = k.cdc.Unmarshal(bz, &params)
	return params, err
}
