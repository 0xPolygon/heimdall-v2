package keeper

import (
	"bytes"
	"context"
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/address"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/0xPolygon/heimdall-v2/x/engine/types"
)

type (
	Keeper struct {
		cdc          codec.BinaryCodec
		storeService store.KVStoreService
		logger       log.Logger

		// the address capable of executing a MsgUpdateParams message. Typically, this
		// should be the x/gov module account.
		authority string

		executionStateMetadata collections.Item[types.ExecutionStateMetadata]
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	logger log.Logger,
	authority string,

) Keeper {
	bz, err := address.NewHexCodec().StringToBytes(authority)
	if err != nil {
		panic(fmt.Errorf("invalid engine authority address: %w", err))
	}

	// ensure only gov has the authority to update the params
	if !bytes.Equal(bz, authtypes.NewModuleAddress(govtypes.ModuleName)) {
		panic(fmt.Errorf("invalid engine authority address: %s", authority))
	}

	sb := collections.NewSchemaBuilder(storeService)

	return Keeper{
		cdc:          cdc,
		storeService: storeService,
		authority:    authority,
		logger:       logger,

		executionStateMetadata: collections.NewItem(sb, types.ExecutionStateMetadataPrefixKey, "execution_state_metadata", codec.CollValue[types.ExecutionStateMetadata](cdc)),
	}
}

// GetAuthority returns the module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// Logger returns a module-specific logger.
func (k Keeper) Logger() log.Logger {
	return k.logger.With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// SetExecutionStateMetadata stores the execution state metadata
func (k Keeper) SetExecutionStateMetadata(ctx context.Context, metadata types.ExecutionStateMetadata) error {
	return k.executionStateMetadata.Set(ctx, metadata)
}

// GetExecutionStateMetadata retrieves the execution state metadata
func (k Keeper) GetExecutionStateMetadata(ctx context.Context) (types.ExecutionStateMetadata, error) {
	return k.executionStateMetadata.Get(ctx)
}
