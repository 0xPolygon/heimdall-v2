package keeper_test

// TestPendingBorBlockTracking covers the pending-bor-head tracking accessors:
// absent reads as zero/nil, and set/get round-trips the head, its identity, and the height.
func (s *KeeperTestSuite) TestPendingBorBlockTracking() {
	ctx, keeper, require := s.ctx, s.milestoneKeeper, s.Require()

	block, id, height, err := keeper.GetPendingBorBlockTracking(ctx)
	require.NoError(err)
	require.Zero(block)
	require.Nil(id)
	require.Zero(height)

	wantID := []byte{0x01, 0x02, 0x03}
	require.NoError(keeper.SetPendingBorBlockTracking(ctx, 4242, wantID, 777))

	block, id, height, err = keeper.GetPendingBorBlockTracking(ctx)
	require.NoError(err)
	require.Equal(uint64(4242), block)
	require.Equal(wantID, id)
	require.Equal(uint64(777), height)

	require.NoError(keeper.SetPendingBorBlockTracking(ctx, 5000, []byte{0x09}, 800))

	block, id, height, err = keeper.GetPendingBorBlockTracking(ctx)
	require.NoError(err)
	require.Equal(uint64(5000), block)
	require.Equal([]byte{0x09}, id)
	require.Equal(uint64(800), height)
}

// TestKeeperAccessorHelpers covers the small keeper methods that only read or
// mutate a single field so the accessors introduced for the pending-stall feature stay covered.
func (s *KeeperTestSuite) TestKeeperAccessorHelpers() {
	ctx, keeper, require := s.ctx, s.milestoneKeeper, s.Require()

	keeper.SetContractCaller(nil)

	block, err := keeper.GetLastMilestoneBlock(ctx)
	require.NoError(err)
	require.Zero(block)

	has, err := keeper.HasMilestone(ctx)
	require.NoError(err)
	require.False(has)

	require.NoError(keeper.SetLastMilestoneBlock(ctx, 1234))
	block, err = keeper.GetLastMilestoneBlock(ctx)
	require.NoError(err)
	require.Equal(uint64(1234), block)
}
