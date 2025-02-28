package keeper_test

import (
	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
)

func (s *KeeperTestSuite) TestInitExportGenesis() {
	ctx, keeper := s.ctx, s.milestoneKeeper
	require := s.Require()

	genesisState := types.NewGenesisState()

	keeper.InitGenesis(ctx, &genesisState)

	require.NotNil(keeper.ExportGenesis(ctx))

}
