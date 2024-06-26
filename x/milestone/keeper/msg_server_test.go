package keeper_test

import (
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/ethereum/go-ethereum/common"

	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/milestone/testutil"
	milestoneSim "github.com/0xPolygon/heimdall-v2/x/milestone/testutil"
	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
	stakeSim "github.com/0xPolygon/heimdall-v2/x/stake/testutil"
)

const (
	AccountHash = "0x000000000000000000000000000000000000dEaD"
	BorChainId  = "1234"
)

func (s *KeeperTestSuite) TestHandleMsgMilestone() {
	ctx, msgServer, keeper := s.ctx, s.msgServer, s.milestoneKeeper
	require := s.Require()
	stakingKeeper := s.stakeKeeper
	start := uint64(0)
	milestoneID := "0000"
	params := types.DefaultParams()
	err := keeper.SetParams(ctx, params)
	require.NoError(err)

	minMilestoneLength := params.MinMilestoneLength

	// check valid milestone
	// generate proposer for validator set
	stakeSim.LoadRandomValidatorSet(require, 2, stakingKeeper, ctx, false, 10)
	err = stakingKeeper.IncrementAccum(ctx, 1)
	require.NoError(err)

	lastMilestone, err := keeper.GetLastMilestone(ctx)
	if err == nil {
		start = start + lastMilestone.EndBlock + 1
	}

	header := milestoneSim.GenRandMilestone(start, minMilestoneLength)

	ctx = ctx.WithBlockHeight(3)

	s.Run("Invalid Proposer", func() {
		header.Proposer = common.HexToAddress(AccountHash).String()
		msgMilestone := types.NewMsgMilestoneBlock(
			header.Proposer,
			header.StartBlock,
			header.EndBlock,
			header.Hash,
			BorChainId,
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
			BorChainId,
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
			BorChainId,
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
			BorChainId,
			milestoneID,
		)

		// send milestone to handler
		res, err := msgServer.Milestone(ctx, &msgMilestone)
		require.NotNil(res)
		require.NoError(err)
		bufferedHeader, err := keeper.GetLastMilestone(ctx)
		require.NoError(err)
		require.Empty(bufferedHeader, "Should not store state")
		milestoneBlockNumber, err := keeper.GetMilestoneBlockNumber(ctx)
		require.NoError(err)
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
			BorChainId,
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
			BorChainId,
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
			BorChainId,
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
	err := stakingKeeper.IncrementAccum(ctx, 1)
	require.NoError(err)

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
	hash := hmTypes.HeimdallHash{Hash: testutil.RandomBytes()}
	proposerAddress := secp256k1.GenPrivKey().PubKey().Address().String()
	timestamp := uint64(0)
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
		BorChainId,
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
