package keeper

import (
	"context"

	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
)

// InitGenesis sets the milestone module's state from a given genesis state.
func (k Keeper) InitGenesis(ctx context.Context, data *types.GenesisState) {
}

// ExportGenesis returns milestone module's genesis state
func (k Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	return &types.GenesisState{}
}
