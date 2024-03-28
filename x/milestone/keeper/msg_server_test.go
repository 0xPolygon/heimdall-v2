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
	keeper.SetParams(ctx, params)

	milestoneLength := params.MinMilestoneLength

	// check valid milestone
	// generate proposer for validator set
	stakeSim.LoadValidatorSet(require, 2, stakingKeeper, ctx, false, 10)
	stakingKeeper.IncrementAccum(ctx, 1)

	lastMilestone, err := keeper.GetLastMilestone(ctx)
	if err == nil {
		start = start + lastMilestone.EndBlock + 1
	}

	header, err := milestoneSim.GenRandMilestone(start, milestoneLength)
	require.NoError(err)

	ctx = ctx.WithBlockHeight(3)

	//Test1- When milestone is proposed by wrong proposer
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

	//Test2- When milestone is proposed of length shorter than configured minimum length
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

	//Test3- When the first milestone is composed of incorrect start number
	s.Run("Failure-Invalid Start Block Number", func() {
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

	//Test4- When the correct milestone is proposed
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
		bufferedHeader, _ := keeper.GetLastMilestone(ctx)
		require.Empty(bufferedHeader, "Should not store state")
		milestoneBlockNumber := keeper.GetMilestoneBlockNumber(ctx)
		require.Equal(int64(3), milestoneBlockNumber, "Mismatch in milestoneBlockNumber")
	})

	header.Proposer = stakingKeeper.GetMilestoneValidatorSet(ctx).Proposer.Signer

	ctx = ctx.WithBlockHeight(int64(4))

	//Test5- When previous milestone is still in voting phase
	s.Run("Previous milestone is still in voting phase", func() {

		msgMilestone := types.NewMsgMilestoneBlock(
			header.Proposer,
			start,
			start+milestoneLength-1,
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

	//Test5- When milestone is not in continuity
	s.Run("Milestone not in countinuity", func() {

		err := keeper.AddMilestone(ctx, header)
		require.NoError(err)

		_, err = keeper.GetLastMilestone(ctx)
		require.NoError(err)

		lastMilestone, err := keeper.GetLastMilestone(ctx)
		if err == nil {
			// pass wrong start
			start = start + lastMilestone.EndBlock + 2 //Start block is 2 more than last milestone's end block
		}

		msgMilestone := types.NewMsgMilestoneBlock(
			header.Proposer,
			start,
			start+milestoneLength-1,
			header.Hash,
			borChainId,
			milestoneID,
		)

		// send milestone to handler
		res, err := msgServer.Milestone(ctx, &msgMilestone)
		require.Nil(res)
		require.ErrorContains(err, types.ErrMilestoneNotInContinuity.Error())
	})

	header.Proposer = stakingKeeper.GetMilestoneValidatorSet(ctx).Proposer.Signer

	//Test6- When milestone is not in continuity
	s.Run("Milestone not in countinuity", func() {

		_, err = keeper.GetLastMilestone(ctx)
		require.NoError(err)

		lastMilestone, err := keeper.GetLastMilestone(ctx)
		if err == nil {
			// pass wrong start
			start = start + lastMilestone.EndBlock - 2 //Start block is 2 less than last milestone's end block
		}

		msgMilestone := types.NewMsgMilestoneBlock(
			header.Proposer,
			start,
			start+milestoneLength-1,
			header.Hash,
			borChainId,
			milestoneID,
		)

		// send milestone to handler
		res, err := msgServer.Milestone(ctx, &msgMilestone)
		require.Nil(res)
		require.ErrorContains(err, types.ErrMilestoneNotInContinuity.Error())
	})
}

// Test to check that passed milestone should be in the store
func (s *KeeperTestSuite) TestHandleMsgMilestoneExistInStore() {
	ctx, msgServer, keeper := s.ctx, s.msgServer, s.milestoneKeeper
	require := s.Require()
	stakingKeeper := s.stakeKeeper
	start := uint64(0)

	params := types.DefaultParams()

	milestoneLength := params.MinMilestoneLength

	stakeSim.LoadValidatorSet(require, 2, stakingKeeper, ctx, false, 10)
	stakingKeeper.IncrementAccum(ctx, 1)

	lastMilestone, err := keeper.GetLastMilestone(ctx)
	if err == nil {
		start = start + lastMilestone.EndBlock + 1
	}

	header, err := milestoneSim.GenRandMilestone(start, milestoneLength)
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
	keeper.AddMilestone(ctx, header)

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
	_ = keeper.AddMilestone(ctx, milestone)

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
