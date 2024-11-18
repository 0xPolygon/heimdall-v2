package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/mock"

	"github.com/0xPolygon/heimdall-v2/sidetxs"
	"github.com/0xPolygon/heimdall-v2/x/milestone/testutil"
	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
	stakeSim "github.com/0xPolygon/heimdall-v2/x/stake/testutil"
)

func (s *KeeperTestSuite) sideHandler(ctx sdk.Context, msg sdk.Msg) sidetxs.Vote {
	cfg := s.sideMsgCfg
	return cfg.GetSideHandler(msg)(ctx, msg)
}

func (s *KeeperTestSuite) postHandler(ctx sdk.Context, msg sdk.Msg, vote sidetxs.Vote) {
	cfg := s.sideMsgCfg

	cfg.GetPostHandler(msg)(ctx, msg, vote)
}

func (s *KeeperTestSuite) TestSideHandleMsgMilestone() {
	ctx, require, keeper, sideHandler, contractCaller := s.ctx, s.Require(), s.milestoneKeeper, s.sideHandler, s.contractCaller

	params := types.DefaultParams()
	err := keeper.SetParams(ctx, params)
	require.NoError(err)

	start := uint64(0)
	minMilestoneLength := params.MinMilestoneLength

	milestone := testutil.GenRandMilestone(start, minMilestoneLength)

	s.Run("Success", func() {
		contractCaller.Mock = mock.Mock{}

		msgMilestone := types.NewMsgMilestoneBlock(
			milestone.Proposer,
			milestone.StartBlock,
			milestone.EndBlock,
			milestone.Hash,
			BorChainId,
			milestone.MilestoneId,
		)

		contractCaller.On("CheckIfBlocksExist", milestone.EndBlock+params.MilestoneTxConfirmations).Return(true)
		contractCaller.On("GetVoteOnHash", milestone.StartBlock, milestone.EndBlock, common.Bytes2Hex(milestone.Hash), milestone.MilestoneId).Return(true, nil)

		result := sideHandler(ctx, msgMilestone)
		require.Equal(result, sidetxs.Vote_VOTE_YES, "Side tx handler should succeed")

		milestoneReceived, _ := keeper.GetLastMilestone(ctx)
		require.Nil(milestoneReceived, "Should not store state")

	})

	s.Run("No Hash", func() {
		contractCaller.Mock = mock.Mock{}

		msgMilestone := types.NewMsgMilestoneBlock(
			milestone.Proposer,
			milestone.StartBlock,
			milestone.EndBlock,
			milestone.Hash,
			BorChainId,
			milestone.MilestoneId,
		)

		contractCaller.On("CheckIfBlocksExist", milestone.EndBlock+params.MilestoneTxConfirmations).Return(true)
		contractCaller.On("GetVoteOnHash", milestone.StartBlock, milestone.EndBlock, common.Bytes2Hex(milestone.Hash), milestone.MilestoneId).Return(false, nil)

		result := sideHandler(ctx, msgMilestone)
		require.Equal(result, sidetxs.Vote_VOTE_NO, "Side tx handler should fail")

		header, err := keeper.GetLastMilestone(ctx)
		require.Error(err)
		require.Nil(header, "Should not store state")
	})

	s.Run("invalid milestone because of shorter length", func() {
		contractCaller.Mock = mock.Mock{}

		msgMilestone := types.NewMsgMilestoneBlock(
			milestone.Proposer,
			milestone.StartBlock,
			milestone.EndBlock-1,
			milestone.Hash,
			BorChainId,
			milestone.MilestoneId,
		)

		contractCaller.On("CheckIfBlocksExist", milestone.EndBlock+params.MilestoneTxConfirmations).Return(true)
		contractCaller.On("GetVoteOnHash", milestone.StartBlock, milestone.EndBlock, common.Bytes2Hex(milestone.Hash), milestone.MilestoneId).Return(true, nil)

		result := sideHandler(ctx, msgMilestone)
		require.Equal(result, sidetxs.Vote_VOTE_NO, "Side tx handler should fail")
	})

	s.Run("Not in continuity", func() {
		contractCaller.Mock = mock.Mock{}
		err = keeper.AddMilestone(ctx, milestone)

		require.NoError(err)

		msgMilestone := types.NewMsgMilestoneBlock(
			milestone.Proposer,
			milestone.StartBlock,
			milestone.EndBlock,
			milestone.Hash,
			BorChainId,
			milestone.MilestoneId,
		)

		contractCaller.On("CheckIfBlocksExist", milestone.EndBlock+params.MilestoneTxConfirmations).Return(true)
		contractCaller.On("GetVoteOnHash", milestone.StartBlock, milestone.EndBlock, common.Bytes2Hex(milestone.Hash), milestone.MilestoneId).Return(true, nil)

		result := sideHandler(ctx, msgMilestone)
		require.Equal(result, sidetxs.Vote_VOTE_NO, "Side tx handler should fail")
	})
}

func (s *KeeperTestSuite) TestPostHandleMsgMilestone() {
	ctx, require, keeper, stakeKeeper, postHandler := s.ctx, s.Require(), s.milestoneKeeper, s.stakeKeeper, s.postHandler

	milestoneId := TestMilestoneID

	params := types.DefaultParams()
	err := keeper.SetParams(ctx, params)
	require.NoError(err)

	validatorSet := stakeSim.GetRandomValidatorSet(2)
	stakeKeeper.EXPECT().GetMilestoneValidatorSet(gomock.Any()).AnyTimes().Return(validatorSet, nil)
	stakeKeeper.EXPECT().MilestoneIncrementAccum(gomock.Any(), gomock.Any()).AnyTimes().Return()

	start := uint64(0)
	minMilestoneLength := params.MinMilestoneLength

	milestone := testutil.GenRandMilestone(start, minMilestoneLength)
	milestone.BorChainId = BorChainId
	milestone.Timestamp = uint64(ctx.BlockTime().Unix())
	milestone.MilestoneId = milestoneId

	milestoneValidatorSet, err := stakeKeeper.GetMilestoneValidatorSet(ctx)
	require.NoError(err)

	// add current proposer to header
	milestone.Proposer = milestoneValidatorSet.Proposer.Signer

	s.Run("Failure", func() {
		msgMilestone := types.NewMsgMilestoneBlock(
			milestone.Proposer,
			milestone.StartBlock,
			milestone.EndBlock,
			milestone.Hash,
			BorChainId,
			milestoneId,
		)

		postHandler(ctx, msgMilestone, sidetxs.Vote_VOTE_NO)

		lastMilestone, err := keeper.GetLastMilestone(ctx)
		require.Nil(lastMilestone)
		require.Error(err)

		lastNoAckMilestone, err := keeper.GetLastNoAckMilestone(ctx)
		require.NoError(err)
		require.Equal(lastNoAckMilestone, milestoneId)

		IsNoAckMilestone, err := keeper.HasNoAckMilestone(ctx, milestoneId)
		require.NoError(err)
		require.True(IsNoAckMilestone)

		IsNoAckMilestone, err = keeper.HasNoAckMilestone(ctx, "WrongID")
		require.NoError(err)
		require.False(IsNoAckMilestone)
	})

	milestoneId = "00000"
	milestone.MilestoneId = milestoneId

	s.Run("Failure-Invalid Start Block", func() {
		msgMilestone := types.NewMsgMilestoneBlock(
			milestone.Proposer,
			milestone.StartBlock+1,
			milestone.EndBlock+1,
			milestone.Hash,
			BorChainId,
			milestoneId,
		)

		postHandler(ctx, msgMilestone, sidetxs.Vote_VOTE_YES)

		lastMilestone, err := keeper.GetLastMilestone(ctx)
		require.Nil(lastMilestone)
		require.Error(err)

		lastNoAckMilestone, err := keeper.GetLastNoAckMilestone(ctx)
		require.NoError(err)
		require.Equal(lastNoAckMilestone, milestoneId)

		IsNoAckMilestone, err := keeper.HasNoAckMilestone(ctx, milestoneId)
		require.NoError(err)
		require.True(IsNoAckMilestone)

		IsNoAckMilestone, err = keeper.HasNoAckMilestone(ctx, "WrongID")
		require.NoError(err)
		require.False(IsNoAckMilestone)
	})

	milestoneId = "00001"
	milestone.MilestoneId = milestoneId

	s.Run("Success", func() {
		msgMilestone := types.NewMsgMilestoneBlock(
			milestone.Proposer,
			milestone.StartBlock,
			milestone.EndBlock,
			milestone.Hash,
			BorChainId,
			milestoneId,
		)
		postHandler(ctx, msgMilestone, sidetxs.Vote_VOTE_YES)

		bufferedHeader, err := keeper.GetLastMilestone(ctx)
		require.NoError(err)
		require.NotNil(bufferedHeader)

		require.True(testutil.IsEqual(bufferedHeader, &milestone))

		require.Empty(err, "Unable to set milestone, Error: %v", err)

		lastNoAckMilestone, err := keeper.GetLastNoAckMilestone(ctx)
		require.NoError(err)
		require.NotEqual(lastNoAckMilestone, milestoneId)
		require.Equal(lastNoAckMilestone, "00000")

		IsNoAckMilestone, err := keeper.HasNoAckMilestone(ctx, milestoneId)
		require.NoError(err)
		require.False(IsNoAckMilestone)
	})

	milestoneId = "00002"
	milestone.MilestoneId = milestoneId

	s.Run("Pre Exist", func() {
		msgMilestone := types.NewMsgMilestoneBlock(
			milestone.Proposer,
			milestone.StartBlock,
			milestone.EndBlock,
			milestone.Hash,
			BorChainId,
			milestoneId,
		)
		postHandler(ctx, msgMilestone, sidetxs.Vote_VOTE_YES)
		lastNoAckMilestone, err := keeper.GetLastNoAckMilestone(ctx)
		require.NoError(err)
		require.Equal(lastNoAckMilestone, milestoneId)

		IsNoAckMilestone, err := keeper.HasNoAckMilestone(ctx, milestoneId)
		require.NoError(err)
		require.True(IsNoAckMilestone)
	})

	milestoneId = "00003"
	milestone.MilestoneId = milestoneId

	s.Run("Not in continuity", func() {
		msgMilestone := types.NewMsgMilestoneBlock(
			milestone.Proposer,
			milestone.StartBlock+64+1,
			milestone.EndBlock+64+1,
			milestone.Hash,
			BorChainId,
			milestoneId,
		)
		postHandler(ctx, msgMilestone, sidetxs.Vote_VOTE_YES)

		lastNoAckMilestone, err := keeper.GetLastNoAckMilestone(ctx)
		require.NoError(err)
		require.Equal(lastNoAckMilestone, milestoneId)

		IsNoAckMilestone, err := keeper.HasNoAckMilestone(ctx, milestoneId)
		require.NoError(err)
		require.True(IsNoAckMilestone)

	})

	milestoneId = "00004"
	milestone.MilestoneId = milestoneId

	s.Run("Replay", func() {
		msgMilestone := types.NewMsgMilestoneBlock(
			milestone.Proposer,
			milestone.StartBlock,
			milestone.EndBlock,
			milestone.Hash,
			BorChainId,
			milestoneId,
		)
		postHandler(ctx, msgMilestone, sidetxs.Vote_VOTE_NO)
		lastNoAckMilestone, err := keeper.GetLastNoAckMilestone(ctx)
		require.NoError(err)
		require.Equal(lastNoAckMilestone, "00004")
	})
}
