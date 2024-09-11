package keeper_test

import "github.com/0xPolygon/heimdall-v2/x/bor/types"

// TestInitExportGenesis test import and export genesis state
func (suite *KeeperTestSuite) TestInitExportGenesis() {
	borKeeper, ctx := suite.borKeeper, suite.ctx
	require := suite.Require()
	params := types.DefaultParams()

	genSpans := suite.genTestSpans(5)

	genesisState := &types.GenesisState{
		Params: params,
		Spans:  genSpans,
	}

	borKeeper.InitGenesis(ctx, genesisState)

	actualParams := borKeeper.ExportGenesis(ctx)
	require.Equal(genesisState, actualParams)
}
