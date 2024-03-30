package keeper_test

import (
	"github.com/stretchr/testify/mock"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/0xPolygon/heimdall-v2/helper"
	hmModule "github.com/0xPolygon/heimdall-v2/module"
	milestoneSim "github.com/0xPolygon/heimdall-v2/x/milestone/testutil"
	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
	stakeSim "github.com/0xPolygon/heimdall-v2/x/stake/testutil"
)

func (s *KeeperTestSuite) sideHandler(ctx sdk.Context, msg sdk.Msg) hmModule.Vote {
	cfg := s.sideMsgCfg
	return cfg.SideHandler(msg)(ctx, msg)
}

func (s *KeeperTestSuite) postHandler(ctx sdk.Context, msg sdk.Msg, vote hmModule.Vote) {
	cfg := s.sideMsgCfg

	cfg.PostHandler(msg)(ctx, msg, vote)
}

//
// Test cases
//

// test sideHandler for side messages

func (s *KeeperTestSuite) TestSideHandleMsgMilestone() {
	ctx, _, keeper := s.ctx, s.msgServer, s.milestoneKeeper
	require := s.Require()

	params := types.DefaultParams()
	keeper.SetParams(ctx, params)

	start := uint64(0)
	milestoneLength := helper.MilestoneLength

	milestone, err := milestoneSim.GenRandMilestone(start, milestoneLength)
	require.NoError(err)

	borChainId := "1234"

	s.Run("Success", func() {
		s.contractCaller.Mock = mock.Mock{}

		// create milestone msg
		msgMilestone := types.NewMsgMilestoneBlock(
			milestone.Proposer,
			milestone.StartBlock,
			milestone.EndBlock,
			milestone.Hash,
			borChainId,
			milestone.MilestoneID,
		)

		s.contractCaller.On("CheckIfBlocksExist", milestone.EndBlock+params.MilestoneTxConfirmations).Return(true)
		s.contractCaller.On("GetVoteOnHash", milestone.StartBlock, milestone.EndBlock, milestoneLength, milestone.Hash.String(), milestone.MilestoneID).Return(true, nil)

		result := s.sideHandler(ctx, &msgMilestone)
		require.Equal(result, hmModule.Vote_VOTE_YES, "Side tx handler should succeed")

		milestoneReceived, _ := keeper.GetLastMilestone(ctx)
		require.Nil(milestoneReceived, "Should not store state")

	})

	s.Run("No Hash", func() {
		s.contractCaller.Mock = mock.Mock{}

		// create milestone msg
		msgMilestone := types.NewMsgMilestoneBlock(
			milestone.Proposer,
			milestone.StartBlock,
			milestone.EndBlock,
			milestone.Hash,
			borChainId,
			milestone.MilestoneID,
		)

		s.contractCaller.On("CheckIfBlocksExist", milestone.EndBlock+params.MilestoneTxConfirmations).Return(true)
		s.contractCaller.On("GetVoteOnHash", milestone.StartBlock, milestone.EndBlock, milestoneLength, milestone.Hash.String(), milestone.MilestoneID).Return(false, nil)

		result := s.sideHandler(ctx, &msgMilestone)
		require.Equal(result, hmModule.Vote_VOTE_NO, "Side tx handler should fail")

		Header, err := keeper.GetLastMilestone(ctx)
		require.Error(err)
		require.Nil(Header, "Should not store state")
	})

	s.Run("invalid milestone", func() {
		s.contractCaller.Mock = mock.Mock{}

		// create milestone msg
		msgMilestone := types.NewMsgMilestoneBlock(
			milestone.Proposer,
			milestone.StartBlock,
			milestone.EndBlock-1,
			milestone.Hash,
			borChainId,
			milestone.MilestoneID,
		)

		s.contractCaller.On("CheckIfBlocksExist", milestone.EndBlock+params.MilestoneTxConfirmations).Return(true)
		s.contractCaller.On("GetVoteOnHash", milestone.StartBlock, milestone.EndBlock, milestoneLength, milestone.Hash.String(), milestone.MilestoneID).Return(true, nil)

		result := s.sideHandler(ctx, &msgMilestone)
		require.Equal(result, hmModule.Vote_VOTE_NO, "Side tx handler should fail")
	})

	s.Run("Not in continuity", func() {
		s.contractCaller.Mock = mock.Mock{}
		err := keeper.AddMilestone(ctx, milestone)

		require.NoError(err)

		// create milestone msg
		msgMilestone := types.NewMsgMilestoneBlock(
			milestone.Proposer,
			milestone.StartBlock,
			milestone.EndBlock,
			milestone.Hash,
			borChainId,
			milestone.MilestoneID,
		)

		s.contractCaller.On("CheckIfBlocksExist", milestone.EndBlock+params.MilestoneTxConfirmations).Return(true)
		s.contractCaller.On("GetVoteOnHash", milestone.StartBlock, milestone.EndBlock, milestoneLength, milestone.Hash.String(), milestone.MilestoneID).Return(true, nil)

		result := s.sideHandler(ctx, &msgMilestone)
		require.Equal(result, hmModule.Vote_VOTE_NO, "Side tx handler should fail")
	})
}

func (s *KeeperTestSuite) TestPostHandleMsgMilestone() {
	ctx, _, keeper := s.ctx, s.msgServer, s.milestoneKeeper
	require := s.Require()
	stakingKeeper := s.stakeKeeper

	params := types.DefaultParams()
	keeper.SetParams(ctx, params)

	start := uint64(0)
	milestoneLength := helper.MilestoneLength

	// check valid milestone
	// generate proposer for validator set
	stakeSim.LoadValidatorSet(require, 2, stakingKeeper, ctx, false, 10)
	stakingKeeper.IncrementAccum(ctx, 1)

	lastMilestone, err := keeper.GetLastMilestone(ctx)
	if err == nil {
		start = start + lastMilestone.EndBlock + 1
	}

	milestone, err := milestoneSim.GenRandMilestone(start, milestoneLength)
	require.NoError(err)

	// add current proposer to header
	milestone.Proposer = stakingKeeper.GetValidatorSet(ctx).Proposer.Signer

	borChainId := "1234"

	s.Run("Failure", func() {
		// create milestone msg
		msgMilestone := types.NewMsgMilestoneBlock(
			milestone.Proposer,
			milestone.StartBlock,
			milestone.EndBlock,
			milestone.Hash,
			borChainId,
			"00000",
		)

		s.postHandler(ctx, &msgMilestone, hmModule.Vote_VOTE_NO)

		lastMilestone, err = keeper.GetLastMilestone(ctx)
		require.Nil(lastMilestone)
		require.Error(err)

		lastNoAckMilestone := keeper.GetLastNoAckMilestone(ctx)
		require.Equal(lastNoAckMilestone, "00000")

		IsNoAckMilestone := keeper.GetNoAckMilestone(ctx, "00000")
		require.True(IsNoAckMilestone)

		IsNoAckMilestone = keeper.GetNoAckMilestone(ctx, "WrongID")
		require.False(IsNoAckMilestone)

	})

	s.Run("Failure-Invalid Start Block", func() {
		// create milestone msg
		msgMilestone := types.NewMsgMilestoneBlock(
			milestone.Proposer,
			milestone.StartBlock+1,
			milestone.EndBlock+1,
			milestone.Hash,
			borChainId,
			"00000",
		)

		s.postHandler(ctx, &msgMilestone, hmModule.Vote_VOTE_YES)

		lastMilestone, err = keeper.GetLastMilestone(ctx)
		require.Nil(lastMilestone)
		require.Error(err)

		lastNoAckMilestone := keeper.GetLastNoAckMilestone(ctx)
		require.Equal(lastNoAckMilestone, "00000")

		IsNoAckMilestone := keeper.GetNoAckMilestone(ctx, "00000")
		require.True(IsNoAckMilestone)

		IsNoAckMilestone = keeper.GetNoAckMilestone(ctx, "WrongID")
		require.False(IsNoAckMilestone)
	})

	s.Run("Success", func() {
		// create milestone msg
		msgMilestone := types.NewMsgMilestoneBlock(
			milestone.Proposer,
			milestone.StartBlock,
			milestone.EndBlock,
			milestone.Hash,
			borChainId,
			"00001",
		)
		s.postHandler(ctx, &msgMilestone, hmModule.Vote_VOTE_YES)

		bufferedHeader, err := keeper.GetLastMilestone(ctx)
		require.Equal(bufferedHeader.StartBlock, milestone.StartBlock)
		require.Equal(bufferedHeader.EndBlock, milestone.EndBlock)
		require.Equal(bufferedHeader.Hash, milestone.Hash)
		require.Equal(bufferedHeader.Proposer, milestone.Proposer)
		require.Equal(bufferedHeader.BorChainID, milestone.BorChainID)
		require.Empty(err, "Unable to set milestone, Error: %v", err)

		lastNoAckMilestone := keeper.GetLastNoAckMilestone(ctx)
		require.NotEqual(lastNoAckMilestone, "00001")

		lastNoAckMilestone = keeper.GetLastNoAckMilestone(ctx)
		require.Equal(lastNoAckMilestone, "00000")

		IsNoAckMilestone := keeper.GetNoAckMilestone(ctx, "00001")
		require.False(IsNoAckMilestone)

	})

	s.Run("Pre Exist", func() {
		// create milestone msg
		msgMilestone := types.NewMsgMilestoneBlock(
			milestone.Proposer,
			milestone.StartBlock,
			milestone.EndBlock,
			milestone.Hash,
			borChainId,
			"00002",
		)
		s.postHandler(ctx, &msgMilestone, hmModule.Vote_VOTE_YES)

		lastNoAckMilestone := keeper.GetLastNoAckMilestone(ctx)
		require.Equal(lastNoAckMilestone, "00002")

		IsNoAckMilestone := keeper.GetNoAckMilestone(ctx, "00002")
		require.True(IsNoAckMilestone)

	})

	s.Run("Not in continuity", func() {
		// create milestone msg
		msgMilestone := types.NewMsgMilestoneBlock(
			milestone.Proposer,
			milestone.StartBlock+64+1,
			milestone.EndBlock+64+1,
			milestone.Hash,
			borChainId,
			"00003",
		)
		s.postHandler(ctx, &msgMilestone, hmModule.Vote_VOTE_YES)

		lastNoAckMilestone := keeper.GetLastNoAckMilestone(ctx)
		require.Equal(lastNoAckMilestone, "00003")

		IsNoAckMilestone := keeper.GetNoAckMilestone(ctx, "00003")
		require.True(IsNoAckMilestone)

	})

	s.Run("Replay", func() {
		// create milestone msg
		msgMilestone := types.NewMsgMilestoneBlock(
			milestone.Proposer,
			milestone.StartBlock,
			milestone.EndBlock,
			milestone.Hash,
			borChainId,
			"00004",
		)
		s.postHandler(ctx, &msgMilestone, hmModule.Vote_VOTE_NO)
		lastNoAckMilestone := keeper.GetLastNoAckMilestone(ctx)
		require.Equal(lastNoAckMilestone, "00004")
	})
}
