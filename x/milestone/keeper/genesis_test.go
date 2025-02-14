package keeper_test

import (
	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
)

func (s *KeeperTestSuite) TestInitExportGenesis() {
	ctx, keeper := s.ctx, s.milestoneKeeper
	require := s.Require()

	params := types.DefaultParams()
	genesisState := types.NewGenesisState(
		params,
	)

	keeper.InitGenesis(ctx, &genesisState)

	actualParams := keeper.ExportGenesis(ctx)

	require.True(actualParams.Params.Equal(params))
}
