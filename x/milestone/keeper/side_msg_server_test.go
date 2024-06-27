package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/mock"

	hmModule "github.com/0xPolygon/heimdall-v2/module"
	milestoneSim "github.com/0xPolygon/heimdall-v2/x/milestone/testutil"
	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
	stakeSim "github.com/0xPolygon/heimdall-v2/x/stake/testutil"
)

func (s *KeeperTestSuite) sideHandler(ctx sdk.Context, msg sdk.Msg) hmModule.Vote {
	cfg := s.sideMsgCfg
	return cfg.GetSideHandler(msg)(ctx, msg)
}

func (s *KeeperTestSuite) postHandler(ctx sdk.Context, msg sdk.Msg, vote hmModule.Vote) {
	cfg := s.sideMsgCfg

	cfg.GetPostHandler(msg)(ctx, msg, vote)
}

//
// Test cases
//

// test sideHandler for side messages

func (s *KeeperTestSuite) TestSideHandleMsgMilestone() {
	ctx, keeper := s.ctx, s.milestoneKeeper
	require := s.Require()

	params := types.DefaultParams()
	err := keeper.SetParams(ctx, params)
	require.NoError(err)

	start := uint64(0)
	minMilestoneLength := params.MinMilestoneLength

	milestone := milestoneSim.GenRandMilestone(start, minMilestoneLength)

	s.Run("Success", func() {
		s.contractCaller.Mock = mock.Mock{}

		// create milestone msg
		msgMilestone := types.NewMsgMilestoneBlock(
			milestone.Proposer,
			milestone.StartBlock,
			milestone.EndBlock,
			milestone.Hash,
			BorChainId,
			milestone.MilestoneID,
		)

		s.contractCaller.On("CheckIfBlocksExist", milestone.EndBlock+params.MilestoneTxConfirmations).Return(true)
		s.contractCaller.On("GetVoteOnHash", milestone.StartBlock, milestone.EndBlock, minMilestoneLength, milestone.Hash.String(), milestone.MilestoneID).Return(true, nil)

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
			BorChainId,
			milestone.MilestoneID,
		)

		s.contractCaller.On("CheckIfBlocksExist", milestone.EndBlock+params.MilestoneTxConfirmations).Return(true)
		s.contractCaller.On("GetVoteOnHash", milestone.StartBlock, milestone.EndBlock, minMilestoneLength, milestone.Hash.String(), milestone.MilestoneID).Return(false, nil)

		result := s.sideHandler(ctx, &msgMilestone)
		require.Equal(result, hmModule.Vote_VOTE_NO, "Side tx handler should fail")

		header, err := keeper.GetLastMilestone(ctx)
		require.Error(err)
		require.Nil(header, "Should not store state")
	})

	s.Run("invalid milestone because of shorter length", func() {
		s.contractCaller.Mock = mock.Mock{}

		// create milestone msg
		msgMilestone := types.NewMsgMilestoneBlock(
			milestone.Proposer,
			milestone.StartBlock,
			milestone.EndBlock-1,
			milestone.Hash,
			BorChainId,
			milestone.MilestoneID,
		)

		s.contractCaller.On("CheckIfBlocksExist", milestone.EndBlock+params.MilestoneTxConfirmations).Return(true)
		s.contractCaller.On("GetVoteOnHash", milestone.StartBlock, milestone.EndBlock, minMilestoneLength, milestone.Hash.String(), milestone.MilestoneID).Return(true, nil)

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
			BorChainId,
			milestone.MilestoneID,
		)

		s.contractCaller.On("CheckIfBlocksExist", milestone.EndBlock+params.MilestoneTxConfirmations).Return(true)
		s.contractCaller.On("GetVoteOnHash", milestone.StartBlock, milestone.EndBlock, minMilestoneLength, milestone.Hash.String(), milestone.MilestoneID).Return(true, nil)

		result := s.sideHandler(ctx, &msgMilestone)
		require.Equal(result, hmModule.Vote_VOTE_NO, "Side tx handler should fail")
	})
}

func (s *KeeperTestSuite) TestPostHandleMsgMilestone() {
	ctx, keeper := s.ctx, s.milestoneKeeper
	require := s.Require()
	stakingKeeper := s.stakeKeeper

	params := types.DefaultParams()
	err := keeper.SetParams(ctx, params)
	require.NoError(err)

	validatorSet := stakeSim.GetRandomValidatorSet(2)
	s.stakeKeeper.EXPECT().GetMilestoneValidatorSet(gomock.Any()).AnyTimes().Return(validatorSet, nil)
	s.stakeKeeper.EXPECT().MilestoneIncrementAccum(gomock.Any(), gomock.Any()).AnyTimes().Return()

	start := uint64(0)
	minMilestoneLength := params.MinMilestoneLength

	milestone := milestoneSim.GenRandMilestone(start, minMilestoneLength)

	milestoneValidatorSet, err := stakingKeeper.GetMilestoneValidatorSet(ctx)
	require.NoError(err)

	// add current proposer to header
	milestone.Proposer = milestoneValidatorSet.Proposer.Signer

	s.Run("Failure", func() {
		// create milestone msg
		msgMilestone := types.NewMsgMilestoneBlock(
			milestone.Proposer,
			milestone.StartBlock,
			milestone.EndBlock,
			milestone.Hash,
			BorChainId,
			"00000",
		)

		s.postHandler(ctx, &msgMilestone, hmModule.Vote_VOTE_NO)

		lastMilestone, err := keeper.GetLastMilestone(ctx)
		require.Nil(lastMilestone)
		require.Error(err)

		lastNoAckMilestone, err := keeper.GetLastNoAckMilestone(ctx)
		require.NoError(err)
		require.Equal(lastNoAckMilestone, "00000")

		IsNoAckMilestone, err := keeper.HasNoAckMilestone(ctx, "00000")
		require.NoError(err)
		require.True(IsNoAckMilestone)

		IsNoAckMilestone, err = keeper.HasNoAckMilestone(ctx, "WrongID")
		require.NoError(err)
		require.False(IsNoAckMilestone)
	})

	s.Run("Failure-Invalid Start Block", func() {
		// create milestone msg
		msgMilestone := types.NewMsgMilestoneBlock(
			milestone.Proposer,
			milestone.StartBlock+1,
			milestone.EndBlock+1,
			milestone.Hash,
			BorChainId,
			"00000",
		)

		s.postHandler(ctx, &msgMilestone, hmModule.Vote_VOTE_YES)

		lastMilestone, err := keeper.GetLastMilestone(ctx)
		require.Nil(lastMilestone)
		require.Error(err)

		lastNoAckMilestone, err := keeper.GetLastNoAckMilestone(ctx)
		require.NoError(err)
		require.Equal(lastNoAckMilestone, "00000")

		IsNoAckMilestone, err := keeper.HasNoAckMilestone(ctx, "00000")
		require.NoError(err)
		require.True(IsNoAckMilestone)

		IsNoAckMilestone, err = keeper.HasNoAckMilestone(ctx, "WrongID")
		require.NoError(err)
		require.False(IsNoAckMilestone)
	})

	s.Run("Success", func() {
		// create milestone msg
		msgMilestone := types.NewMsgMilestoneBlock(
			milestone.Proposer,
			milestone.StartBlock,
			milestone.EndBlock,
			milestone.Hash,
			BorChainId,
			"00001",
		)
		s.postHandler(ctx, &msgMilestone, hmModule.Vote_VOTE_YES)

		bufferedHeader, err := keeper.GetLastMilestone(ctx)
		require.True(bufferedHeader.Equal(milestone))

		require.Empty(err, "Unable to set milestone, Error: %v", err)

		lastNoAckMilestone, err := keeper.GetLastNoAckMilestone(ctx)
		require.NoError(err)
		require.NotEqual(lastNoAckMilestone, "00001")

		lastNoAckMilestone, err = keeper.GetLastNoAckMilestone(ctx)
		require.NoError(err)
		require.Equal(lastNoAckMilestone, "00000")

		IsNoAckMilestone, err := keeper.HasNoAckMilestone(ctx, "00001")
		require.NoError(err)
		require.False(IsNoAckMilestone)
	})

	s.Run("Pre Exist", func() {
		// create milestone msg
		msgMilestone := types.NewMsgMilestoneBlock(
			milestone.Proposer,
			milestone.StartBlock,
			milestone.EndBlock,
			milestone.Hash,
			BorChainId,
			"00002",
		)
		s.postHandler(ctx, &msgMilestone, hmModule.Vote_VOTE_YES)

		lastNoAckMilestone, err := keeper.GetLastNoAckMilestone(ctx)
		require.NoError(err)
		require.Equal(lastNoAckMilestone, "00002")

		IsNoAckMilestone, err := keeper.HasNoAckMilestone(ctx, "00002")
		require.NoError(err)
		require.True(IsNoAckMilestone)
	})

	s.Run("Not in continuity", func() {
		// create milestone msg
		msgMilestone := types.NewMsgMilestoneBlock(
			milestone.Proposer,
			milestone.StartBlock+64+1,
			milestone.EndBlock+64+1,
			milestone.Hash,
			BorChainId,
			"00003",
		)
		s.postHandler(ctx, &msgMilestone, hmModule.Vote_VOTE_YES)

		lastNoAckMilestone, err := keeper.GetLastNoAckMilestone(ctx)
		require.NoError(err)
		require.Equal(lastNoAckMilestone, "00003")

		IsNoAckMilestone, err := keeper.HasNoAckMilestone(ctx, "00003")
		require.NoError(err)
		require.True(IsNoAckMilestone)

	})

	s.Run("Replay", func() {
		// create milestone msg
		msgMilestone := types.NewMsgMilestoneBlock(
			milestone.Proposer,
			milestone.StartBlock,
			milestone.EndBlock,
			milestone.Hash,
			BorChainId,
			"00004",
		)
		s.postHandler(ctx, &msgMilestone, hmModule.Vote_VOTE_NO)
		lastNoAckMilestone, err := keeper.GetLastNoAckMilestone(ctx)
		require.NoError(err)
		require.Equal(lastNoAckMilestone, "00004")
	})
}
