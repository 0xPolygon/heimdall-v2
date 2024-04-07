package keeper

import (
	"context"

	addresscodec "cosmossdk.io/core/address"
	storetypes "cosmossdk.io/core/store"
	"cosmossdk.io/log"

	"github.com/0xPolygon/heimdall-v2/helper"
	cmKeeper "github.com/0xPolygon/heimdall-v2/x/chainmanager/keeper"
	"github.com/0xPolygon/heimdall-v2/x/stake/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Keeper of the x/staking store
type Keeper struct {
	storeService          storetypes.KVStoreService
	cdc                   codec.BinaryCodec
	authority             string
	moduleCommunicator    types.ModuleCommunicator
	cmKeeper              *cmKeeper.Keeper
	validatorAddressCodec addresscodec.Codec
	IContractCaller       helper.IContractCaller
}

// NewKeeper creates a new staking Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec,
	storeService storetypes.KVStoreService,
	authority string,
	moduleCommunicator types.ModuleCommunicator,
	cmKeeper *cmKeeper.Keeper,
	validatorAddressCodec addresscodec.Codec,
	contractCaller helper.IContractCaller,
) *Keeper {
	return &Keeper{
		storeService:          storeService,
		cdc:                   cdc,
		authority:             authority,
		moduleCommunicator:    moduleCommunicator,
		cmKeeper:              cmKeeper,
		validatorAddressCodec: validatorAddressCodec,
		IContractCaller:       contractCaller,
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx context.Context) log.Logger {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.Logger().With("module", "x/"+types.ModuleName)
}
