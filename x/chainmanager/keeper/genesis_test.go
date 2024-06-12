package keeper_test

import (
	"github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
)

// TestInitExportGenesis test import and export genesis state
func (s *KeeperTestSuite) TestInitExportGenesis() {
	chainmanagerKeeper, ctx := s.chainmanagerKeeper, s.ctx
	require := s.Require()
	params := types.DefaultParams()

	genesisState := &types.GenesisState{
		Params: params,
	}
	chainmanagerKeeper.InitGenesis(ctx, genesisState)

	actualParams := chainmanagerKeeper.ExportGenesis(ctx)
	require.Equal(genesisState, actualParams)
}
