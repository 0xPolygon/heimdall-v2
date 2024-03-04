package keeper

import (
	"context"

	"github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
)

// InitGenesis sets chainmanager information for genesis.
func (k Keeper) InitGenesis(ctx context.Context, data *types.GenesisState) {
	k.SetParams(ctx, data.Params)
}

// ExportGenesis returns a GenesisState for chainmanager.
func (k Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	params, err := k.GetParams(ctx)
	if err != nil {
		k.Logger(ctx).Error("failed to get params", "error", err)
		return nil
	}

	return types.NewGenesisState(
		params,
	)
}
