package keeper

import (
	"context"

	"cosmossdk.io/collections"
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
	cdc          codec.BinaryCodec
	storeService storetypes.KVStoreService

	schema collections.Schema

	authority             string
	moduleCommunicator    types.ModuleCommunicator
	cmKeeper              *cmKeeper.Keeper
	validatorAddressCodec addresscodec.Codec
	IContractCaller       helper.IContractCaller

	// Validators key: valAddr | value: Validator
	Validators collections.Map[string, types.Validator]

	// ValidatorSet
	ValidatorSet collections.Map[[]byte, types.ValidatorSet]

	SignerIDMap collections.Map[uint64, string]

	StakingSequenceMap collections.Map[string, []byte]
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
	sb := collections.NewSchemaBuilder(storeService)

	k := &Keeper{
		storeService:          storeService,
		cdc:                   cdc,
		authority:             authority,
		moduleCommunicator:    moduleCommunicator,
		cmKeeper:              cmKeeper,
		validatorAddressCodec: validatorAddressCodec,
		IContractCaller:       contractCaller,
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
