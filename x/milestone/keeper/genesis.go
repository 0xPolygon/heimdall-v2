package keeper

import (
	"context"
	"fmt"

	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// InitGenesis sets the milestone module's state from a given genesis state.
func (k Keeper) InitGenesis(ctx context.Context, data *types.GenesisState) {
	if err := k.SetParams(ctx, types.GetDefaultParams()); err != nil {
		panic(fmt.Errorf("failed to set default milestone params: %w", err))
	}

	return
}

// ExportGenesis returns milestone module's genesis state
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	return &types.GenesisState{}
}
