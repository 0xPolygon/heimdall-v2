package keeper_test

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/golang/mock/gomock"

	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/checkpoint/testutil"
	chSim "github.com/0xPolygon/heimdall-v2/x/checkpoint/testutil"
	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
	stakeSim "github.com/0xPolygon/heimdall-v2/x/stake/testutil"
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

	accountRoot := hmTypes.HeimdallHash{Hash: accRootHash}

	s.Run("Success", func() {
		msgCheckpoint := types.NewMsgCheckpointBlock(
			header.Proposer,
			header.StartBlock,
			header.EndBlock,
			header.RootHash,
			accountRoot,
			borChainId,
		)

		// send checkpoint to handler
		res, err := msgServer.Checkpoint(ctx, &msgCheckpoint)
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
			accountRoot,
			borChainId,
		)

		// send checkpoint to handler
		_, err := msgServer.Checkpoint(ctx, &msgCheckpoint)
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
			accountRoot,
			borChainId,
		)

		// send checkpoint to handler
		_, err = msgServer.Checkpoint(ctx, &msgCheckpoint)
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

	accountRoot := hmTypes.HeimdallHash{Hash: accRootHash}

	msgCheckpoint := types.NewMsgCheckpointBlock(
		header.Proposer,
		header.StartBlock,
		header.EndBlock,
		header.RootHash,
		accountRoot,
		borChainId,
	)

	// send checkpoint to handler
	_, err = msgServer.Checkpoint(ctx, &msgCheckpoint)
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
	_, err = msgServer.Checkpoint(ctx, &msgCheckpoint)
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

	accountRoot := hmTypes.HeimdallHash{Hash: accRootHash}

	msgCheckpoint := types.NewMsgCheckpointBlock(
		header.Proposer,
		header.StartBlock,
		header.EndBlock,
		header.RootHash,
		accountRoot,
		borChainId,
	)

	// send checkpoint to handler
	_, err = msgServer.Checkpoint(ctx, &msgCheckpoint)
	require.NoError(err)

	err = keeper.SetCheckpointBuffer(ctx, header)
	require.NoError(err)

	// send checkpoint to handler
	_, err = msgServer.Checkpoint(ctx, &msgCheckpoint)
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
		msgCheckpointAck := types.NewMsgCheckpointAck(
			common.Address{}.String(),
			headerId,
			header.Proposer,
			header.StartBlock,
			header.EndBlock,
			header.RootHash,
			hmTypes.HeimdallHash{Hash: testutil.RandomBytes()},
			uint64(1),
		)

		_, err = msgServer.CheckpointAck(ctx, &msgCheckpointAck)
		require.ErrorContains(err, types.ErrBadAck.Error())

	})

	err = keeper.SetCheckpointBuffer(ctx, header)
	require.NoError(err)

	s.Run("Success", func() {
		msgCheckpointAck := types.NewMsgCheckpointAck(
			common.Address{}.String(),
			headerId,
			header.Proposer,
			header.StartBlock,
			header.EndBlock,
			header.RootHash,
			hmTypes.HeimdallHash{Hash: testutil.RandomBytes()},
			uint64(1),
		)

		_, err = msgServer.CheckpointAck(ctx, &msgCheckpointAck)
		require.NoError(err)

		afterAckBufferedCheckpoint, _ := keeper.GetCheckpointFromBuffer(ctx)
		require.NotNil(afterAckBufferedCheckpoint, "should not remove from buffer")
	})

	s.Run("Invalid start", func() {
		msgCheckpointAck := types.NewMsgCheckpointAck(
			common.Address{}.String(),
			headerId,
			header.Proposer,
			uint64(123),
			header.EndBlock,
			header.RootHash,
			hmTypes.HeimdallHash{Hash: testutil.RandomBytes()},
			uint64(1),
		)

		_, err = msgServer.CheckpointAck(ctx, &msgCheckpointAck)
		require.ErrorContains(err, types.ErrBadAck.Error())
	})

	s.Run("Invalid RootHash", func() {
		msgCheckpointAck := types.NewMsgCheckpointAck(
			common.Address{}.String(),
			headerId,
			header.Proposer,
			header.StartBlock,
			header.EndBlock,
			hmTypes.HeimdallHash{Hash: testutil.RandomBytes()},
			hmTypes.HeimdallHash{Hash: testutil.RandomBytes()},
			uint64(1),
		)

		_, err = msgServer.CheckpointAck(ctx, &msgCheckpointAck)
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

	//Rotate the list to get the next proposer in line
	dupValidatorSet := validatorSet.Copy()
	dupValidatorSet.IncrementProposerPriority(1)
	noAckProposer := dupValidatorSet.Proposer.Signer

	msgNoAck := types.MsgCheckpointNoAck{
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

	msgNoAck = types.MsgCheckpointNoAck{
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

	msgNoAck = types.MsgCheckpointNoAck{
		From: noAckProposer,
	}

	_, err = msgServer.CheckpointNoAck(ctx, &msgNoAck)
	require.NoError(err)

	updatedAckCount, err = keeper.GetAckCount(ctx)
	require.NoError(err)

	require.Equal(ackCount, updatedAckCount, "Should not update state")
}
