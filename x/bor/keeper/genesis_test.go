package keeper_test

import "github.com/0xPolygon/heimdall-v2/x/bor/types"

// TestInitExportGenesis test import and export genesis state
func (s *KeeperTestSuite) TestInitExportGenesis() {
	keeper, ctx, require := s.borKeeper, s.ctx, s.Require()

	params := types.DefaultParams()
	genSpansPtr := s.genTestSpans(5)
	genSpans := make([]types.Span, len(genSpansPtr))
	for i, spanPtr := range genSpansPtr {
		genSpans[i] = *spanPtr
	}

	genesisState := &types.GenesisState{
		Params: params,
		Spans:  genSpans,
	}

	keeper.InitGenesis(ctx, genesisState)

	actualParams := keeper.ExportGenesis(ctx)
	require.Equal(genesisState, actualParams)
}
