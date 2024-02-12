package keeper

import (
	"context"

	storetypes "cosmossdk.io/core/store"
	"cosmossdk.io/log"

	"github.com/0xPolygon/heimdall-v2/helper"
	chainmanager "github.com/0xPolygon/heimdall-v2/x/chainmanager/keeper"
	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
	staking "github.com/0xPolygon/heimdall-v2/x/staking/keeper"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Keeper of the x/staking store
type Keeper struct {
	storeService       storetypes.KVStoreService
	cdc                codec.BinaryCodec
	authority          string
	sk                 staking.Keeper
	ck                 chainmanager.Keeper
	moduleCommunicator types.ModuleCommunicator
	IContractCaller    helper.IContractCaller
}

// NewKeeper creates a new staking Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec,
	storeService storetypes.KVStoreService,
	authority string,
	stakingKeeper staking.Keeper,
	chainmanagerKeeper chainmanager.Keeper,
	moduleCommunicator types.ModuleCommunicator,
	contractCaller helper.IContractCaller,

) *Keeper {
	return &Keeper{
		storeService:       storeService,
		cdc:                cdc,
		authority:          authority,
		sk:                 stakingKeeper,
		ck:                 chainmanagerKeeper,
		moduleCommunicator: moduleCommunicator,
		IContractCaller:    contractCaller,
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx context.Context) log.Logger {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.Logger().With("module", "x/"+types.ModuleName)
}
