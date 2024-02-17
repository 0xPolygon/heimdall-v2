package keeper

import (
	"context"

	addresscodec "cosmossdk.io/core/address"
	storetypes "cosmossdk.io/core/store"
	"cosmossdk.io/log"

	"github.com/0xPolygon/heimdall-v2/helper"
	//cmKeeper "github.com/0xPolygon/heimdall-v2/x/chainmanager/keeper"
	"github.com/0xPolygon/heimdall-v2/x/stake/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Implements ValidatorSet interface
// TODO H2 Please write the interface of Validator Set
var _ types.ValidatorSet = Keeper{}

// Keeper of the x/staking store
type Keeper struct {
	storeService       storetypes.KVStoreService
	cdc                codec.BinaryCodec
	hooks              types.StakingHooks
	authority          string
	moduleCommunicator types.ModuleCommunicator
	//chainManagerKeeper    *cmKeeper.Keeper
	validatorAddressCodec addresscodec.Codec
	IContractCaller       helper.IContractCaller
}

// NewKeeper creates a new staking Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec,
	storeService storetypes.KVStoreService,
	authority string,
	moduleCommunicator types.ModuleCommunicator,
	validatorAddressCodec addresscodec.Codec,
	contractCaller helper.IContractCaller,
) *Keeper {
	return &Keeper{
		storeService:          storeService,
		cdc:                   cdc,
		hooks:                 nil,
		authority:             authority,
		moduleCommunicator:    moduleCommunicator,
		validatorAddressCodec: validatorAddressCodec,
		IContractCaller:       contractCaller,
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx context.Context) log.Logger {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.Logger().With("module", "x/"+types.ModuleName)
}

// Hooks gets the hooks for staking *Keeper {
func (k *Keeper) Hooks() types.StakingHooks {
	if k.hooks == nil {
		// return a no-op implementation if no hooks are set
		return types.MultiStakingHooks{}
	}

	return k.hooks
}

// SetHooks sets the validator hooks.  In contrast to other receivers, this method must take a pointer due to nature
// of the hooks interface and SDK start up sequence.
func (k *Keeper) SetHooks(sh types.StakingHooks) {
	if k.hooks != nil {
		panic("cannot set validator hooks twice")
	}

	k.hooks = sh
}
