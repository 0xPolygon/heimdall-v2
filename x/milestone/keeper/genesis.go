package keeper

import (
	"context"
	"fmt"

	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
)

// InitGenesis sets the milestone module's state from a given genesis state.
func (k Keeper) InitGenesis(ctx context.Context, data *types.GenesisState) {
	err := data.Params.Validate()
	if err != nil {
		panic(fmt.Sprint("error in validating the milestone params", "err", err))
	}

	err = k.params.Set(ctx, *data.Params)
	if err != nil {
		panic(fmt.Sprint("error in setting the milestone params", "err", err))
	}

}

// ExportGenesis returns milestone module's genesis state
func (k Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	params, err := k.GetParams(ctx)
	if err != nil {
		k.Logger(ctx).Error("error while getting milestone params")
	}

	return &types.GenesisState{Params: &params}
}
