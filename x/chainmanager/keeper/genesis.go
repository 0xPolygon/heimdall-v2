package keeper

import (
	"context"

	"github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
)

// InitGenesis sets chainmanager information for genesis.
func InitGenesis(ctx context.Context, keeper Keeper, data types.GenesisState) {
	keeper.SetParams(ctx, data.Params)
}

// ExportGenesis returns a GenesisState for chainmanager.
func ExportGenesis(ctx context.Context, keeper Keeper) types.GenesisState {
	params := keeper.GetParams(ctx)

	return types.NewGenesisState(
		params,
	)
}
