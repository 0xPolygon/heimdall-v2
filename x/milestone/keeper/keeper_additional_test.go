package keeper_test

import "github.com/0xPolygon/heimdall-v2/x/milestone/types"

func (s *KeeperTestSuite) TestGetLastMilestone_EmptyStore() {
	ctx, require, keeper := s.ctx, s.Require(), s.milestoneKeeper

	result, err := keeper.GetLastMilestone(ctx)
	require.Nil(result)
	require.ErrorIs(err, types.ErrNoMilestoneFound)
}

func (s *KeeperTestSuite) TestSetAndGetParamsAndCountAndMilestones() {
	ctx, require, keeper := s.ctx, s.Require(), s.milestoneKeeper

	params := types.Params{
		MaxMilestonePropositionLength: 17,
		FfMilestoneThreshold:          330,
		FfMilestoneBlockInterval:      33,
	}
	require.NoError(keeper.SetParams(ctx, params))

	gotParams, err := keeper.GetParams(ctx)
	require.NoError(err)
	require.Equal(params, gotParams)

	require.NoError(keeper.SetMilestoneCount(ctx, 7))
	count, err := keeper.GetMilestoneCount(ctx)
	require.NoError(err)
	require.Equal(uint64(7), count)

	milestones, err := keeper.GetMilestones(ctx)
	require.NoError(err)
	require.Empty(milestones)
}
