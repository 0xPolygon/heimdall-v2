package keeper_test

import (
	"time"

	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/checkpoint/testutil"
	chSim "github.com/0xPolygon/heimdall-v2/x/checkpoint/testutil"
	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
	stakeSim "github.com/0xPolygon/heimdall-v2/x/stake/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/golang/mock/gomock"
)

func (s *KeeperTestSuite) TestHandleMsgCheckpoint() {
	ctx, require, msgServer := s.ctx, s.Require(), s.msgServer
	keeper, topupKeeper, stakeKeeper := s.checkpointKeeper, s.topupKeeper, s.stakeKeeper

	start := uint64(0)
	borChainId := "1234"
	params, _ := keeper.GetParams(ctx)

	topupKeeper.EXPECT().GetAllDividendAccounts(gomock.Any()).AnyTimes().Return(testutil.RandDividendAccounts(), nil)
	dividendAccounts, err := topupKeeper.GetAllDividendAccounts(ctx)
	require.NoError(err)

	validatorSet := stakeSim.GetRandomValidatorSet(2)
	stakeKeeper.EXPECT().GetValidatorSet(gomock.Any()).AnyTimes().Return(validatorSet, nil)
	stakeKeeper.EXPECT().GetCurrentProposer(gomock.Any()).AnyTimes().Return(validatorSet.Proposer)

	lastCheckpoint, err := keeper.GetLastCheckpoint(ctx)
	if err == nil {
		start = start + lastCheckpoint.EndBlock + 1
	}

	header := chSim.GenRandCheckpoint(start, params.MaxCheckpointLength)

	// add current proposer to header
	header.Proposer = validatorSet.Proposer.Signer

	accRootHash, err := hmTypes.GetAccountRootHash(dividendAccounts)

	s.Run("Success", func() {
		msgCheckpoint := types.NewMsgCheckpointBlock(
			header.Proposer,
			header.StartBlock,
			header.EndBlock,
			header.RootHash,
			accRootHash,
			borChainId,
		)

		// send checkpoint to handler
		res, err := msgServer.Checkpoint(ctx, msgCheckpoint)
		require.NotNil(res)
		require.NoError(err)

		doExist, err := keeper.HasCheckpointInBuffer(ctx)
		require.NoError(err)
		require.False(doExist)

		_, err = keeper.GetCheckpointFromBuffer(ctx)
		require.Error(err)
	})

	s.Run("Invalid Proposer", func() {
		header.Proposer = common.Address{}.String()

		msgCheckpoint := types.NewMsgCheckpointBlock(
			header.Proposer,
			header.StartBlock,
			header.EndBlock,
			header.RootHash,
			accRootHash,
			borChainId,
		)

		// send checkpoint to handler
		_, err := msgServer.Checkpoint(ctx, msgCheckpoint)
		require.Error(err)
		require.ErrorContains(err, types.ErrInvalidMsg.Error())
	})

	s.Run("Checkpoint not in continuity", func() {
		headerId := uint64(1)

		err = keeper.AddCheckpoint(ctx, headerId, header)
		require.NoError(err)

		_, err = keeper.GetCheckpointByNumber(ctx, headerId)
		require.NoError(err)

		err = keeper.IncrementAckCount(ctx)
		require.NoError(err)

		lastCheckpoint, err := keeper.GetLastCheckpoint(ctx)
		require.NoError(err)

		if err == nil {
			// pass wrong start
			start = start + lastCheckpoint.EndBlock + 2
		}

		msgCheckpoint := types.NewMsgCheckpointBlock(
			header.Proposer,
			start,
			start+256,
			header.RootHash,
			accRootHash,
			borChainId,
		)

		// send checkpoint to handler
		_, err = msgServer.Checkpoint(ctx, msgCheckpoint)
		require.Error(err)
		require.ErrorContains(err, types.ErrDiscontinuousCheckpoint.Error())
	})
}

func (s *KeeperTestSuite) TestHandleMsgCheckpointAfterBufferTimeOut() {
	ctx, require, msgServer := s.ctx, s.Require(), s.msgServer
	keeper, topupKeeper, stakeKeeper := s.checkpointKeeper, s.topupKeeper, s.stakeKeeper

	start := uint64(0)
	maxSize := uint64(256)
	borChainId := "1234"
	params, err := keeper.GetParams(ctx)
	require.NoError(err)
	checkpointBufferTime := params.CheckpointBufferTime

	topupKeeper.EXPECT().GetAllDividendAccounts(gomock.Any()).AnyTimes().Return(testutil.RandDividendAccounts(), nil)
	dividendAccounts, err := topupKeeper.GetAllDividendAccounts(ctx)
	require.NoError(err)

	// generate proposer for validator set
	validatorSet := stakeSim.GetRandomValidatorSet(2)
	stakeKeeper.EXPECT().GetValidatorSet(gomock.Any()).AnyTimes().Return(validatorSet, nil)
	stakeKeeper.EXPECT().GetCurrentProposer(gomock.Any()).AnyTimes().Return(validatorSet.Proposer)

	lastCheckpoint, err := keeper.GetLastCheckpoint(ctx)
	if err == nil {
		start = start + lastCheckpoint.EndBlock + 1
	}

	header := chSim.GenRandCheckpoint(start, maxSize)

	// add current proposer to header
	header.Proposer = validatorSet.Proposer.Signer

	accRootHash, err := hmTypes.GetAccountRootHash(dividendAccounts)
	require.NoError(err)

	msgCheckpoint := types.NewMsgCheckpointBlock(
		header.Proposer,
		header.StartBlock,
		header.EndBlock,
		header.RootHash,
		accRootHash,
		borChainId,
	)

	// send checkpoint to handler
	_, err = msgServer.Checkpoint(ctx, msgCheckpoint)
	require.NoError(err)

	err = keeper.SetCheckpointBuffer(ctx, header)
	require.NoError(err)

	checkpointBuffer, err := keeper.GetCheckpointFromBuffer(ctx)
	require.NoError(err)

	// set time buffered checkpoint timestamp + checkpointBufferTime
	newTime := checkpointBuffer.Timestamp + uint64(checkpointBufferTime)
	ctx = ctx.WithBlockTime(time.Unix(int64(newTime), 0))

	// send new checkpoint which should replace old one
	// send checkpoint to handler
	_, err = msgServer.Checkpoint(ctx, msgCheckpoint)
	require.NoError(err)
}

func (s *KeeperTestSuite) TestHandleMsgCheckpointExistInBuffer() {
	ctx, require, msgServer := s.ctx, s.Require(), s.msgServer
	keeper, topupKeeper, stakeKeeper := s.checkpointKeeper, s.topupKeeper, s.stakeKeeper

	start := uint64(0)
	maxSize := uint64(256)

	borChainId := "1234"

	topupKeeper.EXPECT().GetAllDividendAccounts(gomock.Any()).AnyTimes().Return(testutil.RandDividendAccounts(), nil)
	dividendAccounts, err := topupKeeper.GetAllDividendAccounts(ctx)
	require.NoError(err)

	validatorSet := stakeSim.GetRandomValidatorSet(2)
	stakeKeeper.EXPECT().GetValidatorSet(gomock.Any()).AnyTimes().Return(validatorSet, nil)
	stakeKeeper.EXPECT().GetCurrentProposer(gomock.Any()).AnyTimes().Return(validatorSet.Proposer)

	lastCheckpoint, err := keeper.GetLastCheckpoint(ctx)
	if err == nil {
		start = start + lastCheckpoint.EndBlock + 1
	}

	header := chSim.GenRandCheckpoint(start, maxSize)

	// add current proposer to header
	header.Proposer = validatorSet.Proposer.Signer

	accRootHash, err := hmTypes.GetAccountRootHash(dividendAccounts)
	require.NoError(err)

	msgCheckpoint := types.NewMsgCheckpointBlock(
		header.Proposer,
		header.StartBlock,
		header.EndBlock,
		header.RootHash,
		accRootHash,
		borChainId,
	)

	// send checkpoint to handler
	_, err = msgServer.Checkpoint(ctx, msgCheckpoint)
	require.NoError(err)

	err = keeper.SetCheckpointBuffer(ctx, header)
	require.NoError(err)

	// send checkpoint to handler
	_, err = msgServer.Checkpoint(ctx, msgCheckpoint)
	require.ErrorContains(err, types.ErrNoAck.Error())
}

func (s *KeeperTestSuite) TestHandleMsgCheckpointAck() {
	ctx, require, msgServer := s.ctx, s.Require(), s.msgServer
	keeper, stakeKeeper := s.checkpointKeeper, s.stakeKeeper

	start := uint64(0)
	maxSize := uint64(256)

	validatorSet := stakeSim.GetRandomValidatorSet(2)
	stakeKeeper.EXPECT().GetValidatorSet(gomock.Any()).AnyTimes().Return(validatorSet, nil)
	stakeKeeper.EXPECT().GetCurrentProposer(gomock.Any()).AnyTimes().Return(validatorSet.Proposer)

	lastCheckpoint, err := keeper.GetLastCheckpoint(ctx)
	if err == nil {
		start = start + lastCheckpoint.EndBlock + 1
	}

	header := chSim.GenRandCheckpoint(start, maxSize)

	// add current proposer to header
	header.Proposer = validatorSet.Proposer.Signer

	headerId := uint64(1)

	s.Run("No checkpoint in buffer", func() {
		MsgCpAck := types.NewMsgCpAck(
			common.Address{}.String(),
			headerId,
			header.Proposer,
			header.StartBlock,
			header.EndBlock,
			header.RootHash,
			testutil.RandomBytes(),
			uint64(1),
		)

		_, err = msgServer.CheckpointAck(ctx, &MsgCpAck)
		require.ErrorContains(err, types.ErrBadAck.Error())
	})

	err = keeper.SetCheckpointBuffer(ctx, header)
	require.NoError(err)

	s.Run("Success", func() {
		MsgCpAck := types.NewMsgCpAck(
			common.Address{}.String(),
			headerId,
			header.Proposer,
			header.StartBlock,
			header.EndBlock,
			header.RootHash,
			testutil.RandomBytes(),
			uint64(1),
		)

		_, err = msgServer.CheckpointAck(ctx, &MsgCpAck)
		require.NoError(err)

		afterAckBufferedCheckpoint, _ := keeper.GetCheckpointFromBuffer(ctx)
		require.NotNil(afterAckBufferedCheckpoint, "should not remove from buffer")
	})

	s.Run("Invalid start", func() {
		MsgCpAck := types.NewMsgCpAck(
			common.Address{}.String(),
			headerId,
			header.Proposer,
			uint64(123),
			header.EndBlock,
			header.RootHash,
			testutil.RandomBytes(),
			uint64(1),
		)

		_, err = msgServer.CheckpointAck(ctx, &MsgCpAck)
		require.ErrorContains(err, types.ErrBadAck.Error())
	})

	s.Run("Invalid RootHash", func() {
		MsgCpAck := types.NewMsgCpAck(
			common.Address{}.String(),
			headerId,
			header.Proposer,
			header.StartBlock,
			header.EndBlock,
			testutil.RandomBytes(),
			testutil.RandomBytes(),
			uint64(1),
		)

		_, err = msgServer.CheckpointAck(ctx, &MsgCpAck)
		require.ErrorContains(err, types.ErrBadAck.Error())
	})
}

func (s *KeeperTestSuite) TestHandleMsgCheckpointNoAck() {
	ctx, require, msgServer := s.ctx, s.Require(), s.msgServer
	keeper, stakeKeeper := s.checkpointKeeper, s.stakeKeeper

	start := uint64(0)
	maxSize := uint64(256)
	params, err := keeper.GetParams(ctx)
	require.NoError(err)
	checkpointBufferTime := params.CheckpointBufferTime

	validatorSet := stakeSim.GetRandomValidatorSet(4)

	stakeKeeper.EXPECT().GetValidatorSet(gomock.Any()).AnyTimes().Return(validatorSet, nil)
	stakeKeeper.EXPECT().GetCurrentProposer(gomock.Any()).AnyTimes().Return(validatorSet.Proposer)
	stakeKeeper.EXPECT().IncrementAccum(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)

	lastCheckpoint, err := keeper.GetLastCheckpoint(ctx)
	if err == nil {
		start = start + lastCheckpoint.EndBlock + 1
	}

	header := chSim.GenRandCheckpoint(start, maxSize)

	// add current proposer to header
	header.Proposer = validatorSet.Proposer.Signer

	err = keeper.AddCheckpoint(ctx, uint64(1), header)
	require.NoError(err)

	ackCount, err := keeper.GetAckCount(ctx)
	require.NoError(err)

	// set time lastCheckpoint timestamp + checkpointBufferTime-10
	newTime := lastCheckpoint.Timestamp + uint64(checkpointBufferTime.Seconds()) - uint64(5)
	ctx = ctx.WithBlockTime(time.Unix(int64(newTime), 0))

	// Rotate the list to get the next proposer in line
	dupValidatorSet := validatorSet.Copy()
	dupValidatorSet.IncrementProposerPriority(1)
	noAckProposer := dupValidatorSet.Proposer.Signer

	msgNoAck := types.MsgCpNoAck{
		From: noAckProposer,
	}

	_, err = msgServer.CheckpointNoAck(ctx, &msgNoAck)
	require.ErrorContains(err, types.ErrInvalidNoAck.Error())

	updatedAckCount, err := keeper.GetAckCount(ctx)
	require.NoError(err)

	require.Equal(ackCount, updatedAckCount, "Should not update state")

	// set time lastCheckpoint timestamp + noAckWaitTime
	newTime = lastCheckpoint.Timestamp + uint64(checkpointBufferTime.Seconds())
	ctx = ctx.WithBlockTime(time.Unix(int64(newTime), 0))

	msgNoAck = types.MsgCpNoAck{
		From: header.Proposer,
	}

	_, err = msgServer.CheckpointNoAck(ctx, &msgNoAck)
	require.ErrorContains(err, types.ErrInvalidNoAckProposer.Error())

	updatedAckCount, err = keeper.GetAckCount(ctx)
	require.NoError(err)
	require.Equal(ackCount, updatedAckCount, "should not update state")

	dupValidatorSet = validatorSet.Copy()
	dupValidatorSet.IncrementProposerPriority(1)
	noAckProposer = dupValidatorSet.Proposer.Signer

	msgNoAck = types.MsgCpNoAck{
		From: noAckProposer,
	}

	_, err = msgServer.CheckpointNoAck(ctx, &msgNoAck)
	require.NoError(err)

	updatedAckCount, err = keeper.GetAckCount(ctx)
	require.NoError(err)

	require.Equal(ackCount, updatedAckCount, "Should not update state")
}

func (s *KeeperTestSuite) TestMsgUpdateParams() {
	ctx, require, keeper, queryClient, msgServer, params := s.ctx, s.Require(), s.checkpointKeeper, s.queryClient, s.msgServer, types.DefaultParams()

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
			name: "invalid max checkpoint length",
			input: &types.MsgUpdateParams{
				Authority: keeper.GetAuthority(),
				Params: types.Params{
					MaxCheckpointLength:     0,
					CheckpointBufferTime:    params.CheckpointBufferTime,
					AvgCheckpointLength:     params.AvgCheckpointLength,
					ChildChainBlockInterval: params.ChildChainBlockInterval,
				},
			},
			expErr:    true,
			expErrMsg: "max checkpoint length should be non-zero",
		},
		{
			name: "invalid avg checkpoint length",
			input: &types.MsgUpdateParams{
				Authority: keeper.GetAuthority(),
				Params: types.Params{
					MaxCheckpointLength:     params.MaxCheckpointLength,
					CheckpointBufferTime:    params.CheckpointBufferTime,
					AvgCheckpointLength:     0,
					ChildChainBlockInterval: params.ChildChainBlockInterval,
				},
			},
			expErr:    true,
			expErrMsg: "value of avg checkpoint length should be non-zero",
		},
		{
			name: "invalid avg checkpoint length against max checkpoint length",
			input: &types.MsgUpdateParams{
				Authority: keeper.GetAuthority(),
				Params: types.Params{
					MaxCheckpointLength:     params.MaxCheckpointLength,
					CheckpointBufferTime:    params.CheckpointBufferTime,
					AvgCheckpointLength:     params.MaxCheckpointLength + 1,
					ChildChainBlockInterval: params.ChildChainBlockInterval,
				},
			},
			expErr:    true,
			expErrMsg: "avg checkpoint length should not be greater than max checkpoint length",
		},
		{
			name: "invalid child chain block interval",
			input: &types.MsgUpdateParams{
				Authority: keeper.GetAuthority(),
				Params: types.Params{
					MaxCheckpointLength:     params.MaxCheckpointLength,
					CheckpointBufferTime:    params.CheckpointBufferTime,
					AvgCheckpointLength:     params.AvgCheckpointLength,
					ChildChainBlockInterval: 0,
				},
			},
			expErr:    true,
			expErrMsg: "child chain block interval should be greater than zero",
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

				res, err := queryClient.GetCheckpointParams(ctx, &types.QueryParamsRequest{})
				require.NoError(err)
				require.Equal(params, res.Params)
			}
		})
	}
}
