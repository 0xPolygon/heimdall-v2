package keeper

import (
	"context"

	"github.com/0xPolygon/heimdall-v2/x/bor/types"
)

// InitGenesis sets bor information for genesis.
func (k Keeper) InitGenesis(ctx context.Context, data *types.GenesisState) {
	k.SetParams(ctx, data.Params)

	if len(data.Spans) > 0 {
		// sort data spans before inserting to ensure lastspanId fetched is correct
		// TODO HV2: uncomment when helper is merged
		// helper.SortSpanByID(data.Spans)
		// add new span
		for _, span := range data.Spans {
			if err := k.AddNewRawSpan(ctx, span); err != nil {
				k.Logger(ctx).Error("Error AddNewRawSpan", "error", err)
			}
		}

		// update last span
		k.UpdateLastSpan(ctx, data.Spans[len(data.Spans)-1].Id)
	}

}

// ExportGenesis returns a GenesisState for bor.
func (k Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	params, err := k.GetParams(ctx)
	if err != nil {
		panic(err)
	}

	allSpans := k.GetAllSpans(ctx)
	// TODO HV2: uncomment when helper is merged
	// helper.SortSpanByID(allSpans)

	return types.NewGenesisState(
		params,
		allSpans,
	)
}
