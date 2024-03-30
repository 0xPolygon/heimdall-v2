package keeper

import (
	"context"

	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// InitGenesis sets the pool and parameters for the provided keeper
func (k Keeper) InitGenesis(ctx context.Context, data *types.GenesisState) {
	k.SetParams(ctx, types.GetDefaultParams())

	return
}

// ExportGenesis returns a GenesisState for a given context and keeper. The
// GenesisState will contain the pool, params, validators, and bonds found in
// the keeper.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	return &types.GenesisState{}
}
