package keeper_test

import (
	"time"

	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	milestoneSim "github.com/0xPolygon/heimdall-v2/x/milestone/testutil"
	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
	stakeSim "github.com/0xPolygon/heimdall-v2/x/stake/testutil"
	"github.com/ethereum/go-ethereum/common"
)

func (s *KeeperTestSuite) TestHandleMsgMilestone() {
	ctx, msgServer, keeper := s.ctx, s.msgServer, s.milestoneKeeper
	require := s.Require()
	stakingKeeper := s.stakeKeeper
	start := uint64(0)
	borChainId := "1234"
	milestoneID := "0000"
	params := types.DefaultParams()
	err := keeper.SetParams(ctx, params)
	require.NoError(err)

	minMilestoneLength := params.MinMilestoneLength

	// check valid milestone
	// generate proposer for validator set
	stakeSim.LoadValidatorSet(require, 2, stakingKeeper, ctx, false, 10)
	stakingKeeper.IncrementAccum(ctx, 1)

	lastMilestone, err := keeper.GetLastMilestone(ctx)
	if err == nil {
		start = start + lastMilestone.EndBlock + 1
	}

	header := milestoneSim.GenRandMilestone(start, minMilestoneLength)

	ctx = ctx.WithBlockHeight(3)

	s.Run("Invalid Proposer", func() {
		header.Proposer = common.HexToAddress("1234").String()
		msgMilestone := types.NewMsgMilestoneBlock(
			header.Proposer,
			header.StartBlock,
			header.EndBlock,
			header.Hash,
			borChainId,
			milestoneID,
		)

		// send milestone to handler
		res, err := msgServer.Milestone(ctx, &msgMilestone)
		require.Nil(res)
		require.ErrorContains(err, types.ErrProposerMismatch.Error())
	})

	// add current proposer to header
	header.Proposer = stakingKeeper.GetMilestoneValidatorSet(ctx).Proposer.Signer

	s.Run("Invalid msg based on milestone length", func() {
		msgMilestone := types.NewMsgMilestoneBlock(
			header.Proposer,
			header.StartBlock,
			header.EndBlock-1,
			header.Hash,
			borChainId,
			milestoneID,
		)

		// send milestone to handler
		res, err := msgServer.Milestone(ctx, &msgMilestone)
		require.Nil(res)
		require.ErrorContains(err, types.ErrMilestoneInvalid.Error())
	})

	// add current proposer to header
	header.Proposer = stakingKeeper.GetMilestoneValidatorSet(ctx).Proposer.Signer

	s.Run("Invalid msg based on start block number", func() {
		msgMilestone := types.NewMsgMilestoneBlock(
			header.Proposer,
			uint64(1),
			header.EndBlock+1,
			header.Hash,
			borChainId,
			milestoneID,
		)

		// send milestone to handler
		res, err := msgServer.Milestone(ctx, &msgMilestone)
		require.Nil(res)
		require.ErrorContains(err, types.ErrMilestoneInvalid.Error())
	})

	header.Proposer = stakingKeeper.GetMilestoneValidatorSet(ctx).Proposer.Signer

	s.Run("Success", func() {
		msgMilestone := types.NewMsgMilestoneBlock(
			header.Proposer,
			header.StartBlock,
			header.EndBlock,
			header.Hash,
			borChainId,
			milestoneID,
		)

		// send milestone to handler
		res, err := msgServer.Milestone(ctx, &msgMilestone)
		require.NotNil(res)
		require.NoError(err)
		bufferedHeader, err := keeper.GetLastMilestone(ctx)
		require.NoError(err)
		require.Empty(bufferedHeader, "Should not store state")
		milestoneBlockNumber := keeper.GetMilestoneBlockNumber(ctx)
		require.Equal(int64(3), milestoneBlockNumber, "Mismatch in milestoneBlockNumber")
	})

	header.Proposer = stakingKeeper.GetMilestoneValidatorSet(ctx).Proposer.Signer

	ctx = ctx.WithBlockHeight(int64(4))

	s.Run("Previous milestone is still in voting phase", func() {

		msgMilestone := types.NewMsgMilestoneBlock(
			header.Proposer,
			start,
			start+minMilestoneLength-1,
			header.Hash,
			borChainId,
			milestoneID,
		)

		// send milestone to handler
		res, err := msgServer.Milestone(ctx, &msgMilestone)
		require.Nil(res)
		require.ErrorContains(err, types.ErrPrevMilestoneInVoting.Error())
	})

	header.Proposer = stakingKeeper.GetMilestoneValidatorSet(ctx).Proposer.Signer

	ctx = ctx.WithBlockHeight(int64(6))

	s.Run("Milestone not in continuity", func() {

		err := keeper.AddMilestone(ctx, header)
		require.NoError(err)

		_, err = keeper.GetLastMilestone(ctx)
		require.NoError(err)

		lastMilestone, err := keeper.GetLastMilestone(ctx)
		require.NoError(err)

		start = start + lastMilestone.EndBlock + 2

		msgMilestone := types.NewMsgMilestoneBlock(
			header.Proposer,
			start,
			start+minMilestoneLength-1,
			header.Hash,
			borChainId,
			milestoneID,
		)

		// send milestone to handler
		res, err := msgServer.Milestone(ctx, &msgMilestone)
		require.Nil(res)
		require.ErrorContains(err, types.ErrMilestoneNotInContinuity.Error())

		start = start + lastMilestone.EndBlock - 2

		msgMilestone = types.NewMsgMilestoneBlock(
			header.Proposer,
			start,
			start+minMilestoneLength-1,
			header.Hash,
			borChainId,
			milestoneID,
		)

		// send milestone to handler
		res, err = msgServer.Milestone(ctx, &msgMilestone)
		require.Nil(res)
		require.ErrorContains(err, types.ErrMilestoneNotInContinuity.Error())
	})
}

func (s *KeeperTestSuite) TestHandleMsgMilestoneExistInStore() {
	ctx, msgServer, keeper := s.ctx, s.msgServer, s.milestoneKeeper
	require := s.Require()
	stakingKeeper := s.stakeKeeper
	start := uint64(0)

	params := types.DefaultParams()

	minMilestoneLength := params.MinMilestoneLength

	stakeSim.LoadValidatorSet(require, 2, stakingKeeper, ctx, false, 10)
	stakingKeeper.IncrementAccum(ctx, 1)

	lastMilestone, err := keeper.GetLastMilestone(ctx)
	if err == nil {
		start = start + lastMilestone.EndBlock + 1
	}

	header := milestoneSim.GenRandMilestone(start, minMilestoneLength)
	require.NoError(err)

	// add current proposer to header
	header.Proposer = stakingKeeper.GetValidatorSet(ctx).Proposer.Signer

	msgMilestone := types.NewMsgMilestoneBlock(
		header.Proposer,
		header.StartBlock,
		header.EndBlock,
		header.Hash,
		header.BorChainID,
		header.MilestoneID,
	)

	// send old milestone
	ctx = ctx.WithBlockHeight(int64(3))

	res, err := msgServer.Milestone(ctx, &msgMilestone)
	require.NoError(err)
	require.NotNil(res)

	ctx = ctx.WithBlockHeight(int64(6))

	// Add the milestone in the db
	err = keeper.AddMilestone(ctx, header)
	require.NoError(err)

	// send milestone to handler
	res, err = msgServer.Milestone(ctx, &msgMilestone)
	require.Nil(res)
	require.ErrorContains(err, types.ErrMilestoneNotInContinuity.Error())
}

func (s *KeeperTestSuite) TestHandleMsgMilestoneTimeout() {
	ctx, msgServer, keeper := s.ctx, s.msgServer, s.milestoneKeeper
	require := s.Require()
	stakingKeeper := s.stakeKeeper

	params := types.DefaultParams()

	startBlock := uint64(0)
	endBlock := uint64(63)
	hash := hmTypes.HexToHeimdallHash("123")
	proposerAddress := common.HexToAddress("123").String()
	timestamp := uint64(0)
	borChainId := "1234"
	milestoneID := "0000"

	proposer := common.Address{}.String()

	stakeSim.LoadValidatorSet(require, 2, stakingKeeper, ctx, false, 10)

	s.Run("Last milestone not found", func() {
		msgMilestoneTimeout := types.NewMsgMilestoneTimeout(
			proposer,
		)

		// send milestone to handler
		res, err := msgServer.MilestoneTimeout(ctx, &msgMilestoneTimeout)
		require.Nil(res)
		require.ErrorContains(err, types.ErrNoMilestoneFound.Error())
	})

	milestone := types.CreateMilestone(
		startBlock,
		endBlock,
		hash,
		proposerAddress,
		borChainId,
		milestoneID,
		timestamp,
	)
	err := keeper.AddMilestone(ctx, milestone)
	require.NoError(err)

	newTime := milestone.TimeStamp + uint64(params.MilestoneBufferTime) - 1
	ctx = ctx.WithBlockTime(time.Unix(0, int64(newTime)))

	msgMilestoneTimeout := types.NewMsgMilestoneTimeout(
		proposer,
	)
	// send milestone to handler
	res, err := msgServer.MilestoneTimeout(ctx, &msgMilestoneTimeout)
	require.Nil(res)
	require.ErrorContains(err, types.ErrInvalidMilestoneTimeout.Error())

	newTime = milestone.TimeStamp + 2*uint64(params.MilestoneBufferTime) + 10000000
	ctx = ctx.WithBlockTime(time.Unix(0, int64(newTime)))

	msgMilestoneTimeout = types.NewMsgMilestoneTimeout(
		proposer,
	)

	// send milestone to handler
	res, err = msgServer.MilestoneTimeout(ctx, &msgMilestoneTimeout)
	require.NotNil(res)
	require.Nil(err)
}
