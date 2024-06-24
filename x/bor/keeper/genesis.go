package keeper

import (
	"context"
	"fmt"

	"github.com/0xPolygon/heimdall-v2/x/bor/types"
)

// InitGenesis sets bor information for genesis.
func (k Keeper) InitGenesis(ctx context.Context, data *types.GenesisState) {
	if err := k.SetParams(ctx, data.Params); err != nil {
		panic(fmt.Sprintf("error while setting bor params during InitGenesis: %v", err))
	}

	// sort data spans before inserting to ensure lastspanId fetched is correct
	// TODO HV2: uncomment when helper is merged
	// helper.SortSpanByID(data.Spans)
	// add new span
	for _, span := range data.Spans {
		if err := k.AddNewRawSpan(ctx, span); err != nil {
			panic(fmt.Sprintf("error while adding span during InitGenesis: %v", err))
		}
	}

	if len(data.Spans) > 0 {
		// update last span
		if err := k.UpdateLastSpan(ctx, data.Spans[len(data.Spans)-1].Id); err != nil {
			panic(fmt.Sprintf("error while updating last span during InitGenesis: %v", err))
		}
	}

}

// ExportGenesis returns a GenesisState for bor.
func (k Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	params, err := k.FetchParams(ctx)
	if err != nil {
		panic(err)
	}

	allSpans, err := k.GetAllSpans(ctx)
	if err != nil {
		panic(err)
	}
	// TODO HV2: uncomment when helper is merged
	// helper.SortSpanByID(allSpans)

	return types.NewGenesisState(
		params,
		allSpans,
	)
}
