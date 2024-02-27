package keeper_test

import (
	"github.com/0xPolygon/heimdall-v2/x/chainmanager/keeper"
	"github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
)

// TestInitExportGenesis test import and export genesis state
func (s *KeeperTestSuite) TestInitExportGenesis() {
	chainmanagerKeeper, ctx := s.chainmanagerKeeper, s.ctx
	require := s.Require()
	params := types.DefaultParams()

	genesisState := types.GenesisState{
		Params: params,
	}
	keeper.InitGenesis(ctx, chainmanagerKeeper, genesisState)

	actualParams, err := keeper.ExportGenesis(ctx, chainmanagerKeeper)
	require.NoError(err)
	require.Equal(genesisState, actualParams)
}
