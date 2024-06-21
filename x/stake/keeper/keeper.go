package keeper

import (
	"context"

	"cosmossdk.io/collections"
	addresscodec "cosmossdk.io/core/address"
	storetypes "cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/0xPolygon/heimdall-v2/helper"
	cmKeeper "github.com/0xPolygon/heimdall-v2/x/chainmanager/keeper"
	"github.com/0xPolygon/heimdall-v2/x/stake/types"
)

// Keeper stores all stake related data
type Keeper struct {
	cdc          codec.BinaryCodec
	storeService storetypes.KVStoreService
	schema       collections.Schema

	checkpointKeeper      types.CheckpointKeeper
	cmKeeper              cmKeeper.Keeper
	validatorAddressCodec addresscodec.Codec
	contractCaller        helper.IContractCaller

	validators   collections.Map[string, types.Validator]
	validatorSet collections.Map[[]byte, types.ValidatorSet]
	signer       collections.Map[uint64, string]
	sequences    collections.Map[string, bool]
}

// NewKeeper creates a new stake Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec,
	storeService storetypes.KVStoreService,
	checkpointKeeper types.CheckpointKeeper,
	cmKeeper cmKeeper.Keeper,
	validatorAddressCodec addresscodec.Codec,
	contractCaller helper.IContractCaller,
) Keeper {
	sb := collections.NewSchemaBuilder(storeService)

	k := Keeper{
		storeService:          storeService,
		cdc:                   cdc,
		checkpointKeeper:      checkpointKeeper,
		cmKeeper:              cmKeeper,
		validatorAddressCodec: validatorAddressCodec,
		contractCaller:        contractCaller,

		validators:   collections.NewMap(sb, types.ValidatorsKey, "validator", collections.StringKey, codec.CollValue[types.Validator](cdc)),
		validatorSet: collections.NewMap(sb, types.ValidatorSetKey, "validator_set", collections.BytesKey, codec.CollValue[types.ValidatorSet](cdc)),
		sequences:    collections.NewMap(sb, types.StakeSequenceKey, "stake_sequence", collections.StringKey, collections.BoolValue),
		signer:       collections.NewMap(sb, types.SignerKey, "signer", collections.Uint64Key, collections.StringValue),
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
