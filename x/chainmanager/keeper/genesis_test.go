package keeper_test

import (
	"github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
)

// TestInitExportGenesis test import and export genesis state
func (s *KeeperTestSuite) TestInitExportGenesis() {
	cmKeeper, ctx := s.cmKeeper, s.ctx
	require := s.Require()
	params := types.DefaultParams()

	genesisState := &types.GenesisState{
		Params: params,
	}
	cmKeeper.InitGenesis(ctx, genesisState)

	actualParams := cmKeeper.ExportGenesis(ctx)
	require.Equal(genesisState, actualParams)
}
