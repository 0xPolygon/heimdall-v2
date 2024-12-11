package keeper_test

import (
	"time"

	"github.com/0xPolygon/heimdall-v2/x/milestone/testutil"
	milestoneSim "github.com/0xPolygon/heimdall-v2/x/milestone/testutil"
	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
	stakeSim "github.com/0xPolygon/heimdall-v2/x/stake/testutil"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/golang/mock/gomock"
)

const (
	AccountHash = "0x000000000000000000000000000000000000dEaD"
	BorChainId  = "1234"
)

func (s *KeeperTestSuite) TestHandleMsgMilestone() {
	ctx, require, msgServer, keeper, stakeKeeper := s.ctx, s.Require(), s.msgServer, s.milestoneKeeper, s.stakeKeeper

	start := uint64(0)
	milestoneID := TestMilestoneID
	params := types.DefaultParams()
	err := keeper.SetParams(ctx, params)
	require.NoError(err)

	minMilestoneLength := params.MinMilestoneLength

	validatorSet := stakeSim.GetRandomValidatorSet(2)
	stakeKeeper.EXPECT().GetMilestoneValidatorSet(gomock.Any()).AnyTimes().Return(validatorSet, nil)
	stakeKeeper.EXPECT().MilestoneIncrementAccum(gomock.Any(), gomock.Any()).AnyTimes().Return()

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
		res, err := msgServer.Milestone(ctx, msgMilestone)
		require.Nil(res)
		require.ErrorContains(err, types.ErrProposerMismatch.Error())
	})

	milestoneValidatorSet, err := stakeKeeper.GetMilestoneValidatorSet(ctx)
	require.NoError(err)

	// add current proposer to header
	header.Proposer = milestoneValidatorSet.Proposer.Signer

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
		res, err := msgServer.Milestone(ctx, msgMilestone)
		require.Nil(res)
		require.ErrorContains(err, types.ErrMilestoneInvalid.Error())
	})

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
		res, err := msgServer.Milestone(ctx, msgMilestone)
		require.Nil(res)
		require.ErrorContains(err, types.ErrMilestoneInvalid.Error())
	})

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
		res, err := msgServer.Milestone(ctx, msgMilestone)
		require.NotNil(res)
		require.NoError(err)
		bufferedHeader, err := keeper.GetLastMilestone(ctx)
		require.Error(err)
		require.Nil(bufferedHeader)
		milestoneBlockNumber, err := keeper.GetMilestoneBlockNumber(ctx)
		require.NoError(err)
		require.Equal(int64(3), milestoneBlockNumber, "Mismatch in milestoneBlockNumber")
	})

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
		res, err := msgServer.Milestone(ctx, msgMilestone)
		require.Nil(res)
		require.ErrorContains(err, types.ErrPrevMilestoneInVoting.Error())
	})

	ctx = ctx.WithBlockHeight(int64(6))

	s.Run("Milestone not in continuity", func() {
		err := keeper.AddMilestone(ctx, header)
		require.NoError(err)

		_, err = keeper.GetLastMilestone(ctx)
		require.NoError(err)

		require.NoError(err)

		start = start + header.EndBlock + 2

		msgMilestone := types.NewMsgMilestoneBlock(
			header.Proposer,
			start,
			start+minMilestoneLength-1,
			header.Hash,
			BorChainId,
			milestoneID,
		)

		// send milestone to handler
		res, err := msgServer.Milestone(ctx, msgMilestone)
		require.Nil(res)
		require.ErrorContains(err, types.ErrMilestoneNotInContinuity.Error())

		start = start + header.EndBlock - 2

		msgMilestone = types.NewMsgMilestoneBlock(
			header.Proposer,
			start,
			start+minMilestoneLength-1,
			header.Hash,
			BorChainId,
			milestoneID,
		)

		// send milestone to handler
		res, err = msgServer.Milestone(ctx, msgMilestone)
		require.Nil(res)
		require.ErrorContains(err, types.ErrMilestoneNotInContinuity.Error())
	})
}

func (s *KeeperTestSuite) TestHandleMsgMilestoneExistInStore() {
	ctx, require, msgServer, keeper, stakeKeeper := s.ctx, s.Require(), s.msgServer, s.milestoneKeeper, s.stakeKeeper

	start := uint64(0)

	params := types.DefaultParams()

	minMilestoneLength := params.MinMilestoneLength

	validatorSet := stakeSim.GetRandomValidatorSet(2)
	stakeKeeper.EXPECT().GetMilestoneValidatorSet(gomock.Any()).AnyTimes().Return(validatorSet, nil)
	stakeKeeper.EXPECT().MilestoneIncrementAccum(gomock.Any(), gomock.Any()).AnyTimes().Return()

	milestoneValidatorSet, err := stakeKeeper.GetMilestoneValidatorSet(ctx)
	require.NoError(err)

	header := milestoneSim.GenRandMilestone(start, minMilestoneLength)
	require.NoError(err)

	// add current proposer to header
	header.Proposer = milestoneValidatorSet.Proposer.Signer

	msgMilestone := types.NewMsgMilestoneBlock(
		header.Proposer,
		header.StartBlock,
		header.EndBlock,
		header.Hash,
		header.BorChainId,
		header.MilestoneId,
	)

	// send old milestone
	ctx = ctx.WithBlockHeight(int64(3))

	res, err := msgServer.Milestone(ctx, msgMilestone)
	require.NoError(err)
	require.NotNil(res)

	ctx = ctx.WithBlockHeight(int64(6))

	// Add the milestone in the db
	err = keeper.AddMilestone(ctx, header)
	require.NoError(err)

	// send milestone to handler
	res, err = msgServer.Milestone(ctx, msgMilestone)
	require.Nil(res)
	require.ErrorContains(err, types.ErrMilestoneNotInContinuity.Error())
}

func (s *KeeperTestSuite) TestHandleMsgMilestoneTimeout() {
	ctx, require, msgServer, keeper, stakeKeeper := s.ctx, s.Require(), s.msgServer, s.milestoneKeeper, s.stakeKeeper

	params := types.DefaultParams()

	validatorSet := stakeSim.GetRandomValidatorSet(2)
	stakeKeeper.EXPECT().GetMilestoneValidatorSet(gomock.Any()).AnyTimes().Return(validatorSet, nil)
	stakeKeeper.EXPECT().MilestoneIncrementAccum(gomock.Any(), gomock.Any()).AnyTimes().Return()

	startBlock := uint64(0)
	endBlock := uint64(63)
	hash := testutil.RandomBytes()
	proposerAddress := secp256k1.GenPrivKey().PubKey().Address().String()
	timestamp := uint64(0)
	milestoneID := TestMilestoneID

	proposer := common.Address{}.String()

	s.Run("Last milestone not found", func() {
		msgMilestoneTimeout := types.NewMsgMilestoneTimeout(
			proposer,
		)

		// send milestone to handler
		res, err := msgServer.MilestoneTimeout(ctx, msgMilestoneTimeout)
		require.Nil(res)
		require.ErrorContains(err, types.ErrNoMilestoneFound.Error())
	})

	milestone := testutil.CreateMilestone(
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

	newTime := milestone.Timestamp + uint64(params.MilestoneBufferTime) - 1
	ctx = ctx.WithBlockTime(time.Unix(0, int64(newTime)))

	msgMilestoneTimeout := types.NewMsgMilestoneTimeout(
		proposer,
	)
	// send milestone to handler
	res, err := msgServer.MilestoneTimeout(ctx, msgMilestoneTimeout)
	require.Nil(res)
	require.ErrorContains(err, types.ErrInvalidMilestoneTimeout.Error())

	newTime = milestone.Timestamp + 2*uint64(params.MilestoneBufferTime) + 10000000
	ctx = ctx.WithBlockTime(time.Unix(0, int64(newTime)))

	msgMilestoneTimeout = types.NewMsgMilestoneTimeout(
		proposer,
	)

	// send milestone to handler
	res, err = msgServer.MilestoneTimeout(ctx, msgMilestoneTimeout)
	require.NotNil(res)
	require.Nil(err)
}

func (s *KeeperTestSuite) TestMsgUpdateParams() {
	ctx, require, keeper, queryClient, msgServer, params := s.ctx, s.Require(), s.milestoneKeeper, s.queryClient, s.msgServer, types.DefaultParams()

	testCases := []struct {
		name      string
		input     *types.MsgUpdateParams
		expErr    bool
		expErrMsg string
	}{
		{
			name: "invalid authority",
			input: &types.MsgUpdateParams{
				Authority: "invalid",
				Params:    params,
			},
			expErr:    true,
			expErrMsg: "invalid authority",
		},
		{
			name: "invalid min milestone length",
			input: &types.MsgUpdateParams{
				Authority: keeper.GetAuthority(),
				Params: types.Params{
					MinMilestoneLength:       0,
					MilestoneBufferTime:      params.MilestoneBufferTime,
					MilestoneBufferLength:    params.MilestoneBufferLength,
					MilestoneTxConfirmations: params.MilestoneTxConfirmations,
				},
			},
			expErr:    true,
			expErrMsg: "min milestone length should not be zero",
		},
		{
			name: "invalid milestone buffer time",
			input: &types.MsgUpdateParams{
				Authority: keeper.GetAuthority(),
				Params: types.Params{
					MinMilestoneLength:       params.MinMilestoneLength,
					MilestoneBufferTime:      0,
					MilestoneBufferLength:    params.MilestoneBufferLength,
					MilestoneTxConfirmations: params.MilestoneTxConfirmations,
				},
			},
			expErr:    true,
			expErrMsg: "milestone buffer time should not be zero",
		},
		{
			name: "invalid milestone tx confirmations",
			input: &types.MsgUpdateParams{
				Authority: keeper.GetAuthority(),
				Params: types.Params{
					MinMilestoneLength:       params.MinMilestoneLength,
					MilestoneBufferTime:      params.MilestoneBufferTime,
					MilestoneBufferLength:    params.MilestoneBufferLength,
					MilestoneTxConfirmations: 0,
				},
			},
			expErr:    true,
			expErrMsg: "milestone tx confirmations should not be zero",
		},
		{
			name: "invalid milestone buffer time",
			input: &types.MsgUpdateParams{
				Authority: keeper.GetAuthority(),
				Params: types.Params{
					MinMilestoneLength:       params.MinMilestoneLength,
					MilestoneBufferTime:      0,
					MilestoneBufferLength:    params.MilestoneBufferLength,
					MilestoneTxConfirmations: params.MilestoneTxConfirmations,
				},
			},
			expErr:    true,
			expErrMsg: "milestone buffer time should not be zero",
		},
		{
			name: "all good",
			input: &types.MsgUpdateParams{
				Authority: keeper.GetAuthority(),
				Params:    params,
			},
			expErr: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.Run(tc.name, func() {
			_, err := msgServer.UpdateParams(ctx, tc.input)

			if tc.expErr {
				require.Error(err)
				require.Contains(err.Error(), tc.expErrMsg)
			} else {
				require.Equal(authtypes.NewModuleAddress(govtypes.ModuleName).String(), keeper.GetAuthority())
				require.NoError(err)

				res, err := queryClient.GetMilestoneParams(ctx, &types.QueryParamsRequest{})
				require.NoError(err)
				require.Equal(params, res.Params)
			}
		})
	}
}
