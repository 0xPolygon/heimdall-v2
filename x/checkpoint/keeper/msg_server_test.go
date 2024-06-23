package keeper_test

import (
	"time"

	chSim "github.com/0xPolygon/heimdall-v2/x/checkpoint/testutil"
	stakeSim "github.com/0xPolygon/heimdall-v2/x/stake/testutil"
	"github.com/ethereum/go-ethereum/common"
	"github.com/0xPolygon/heimdall-v2/x/checkpoint/testutil"

	hmTypes "github.com/0xPolygon/heimdall-v2/types"

	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
)

func (s *KeeperTestSuite) TestHandleMsgCheckpoint() {
	ctx, msgServer, keeper := s.ctx, s.msgServer, s.checkpointKeeper
	require := s.Require()

	stakingKeeper := s.stakeKeeper

	start := uint64(0)
	maxSize := uint64(256)
	borChainId := "1234"
	params, _ := keeper.GetParams(ctx)
	dividendAccounts := s.moduleCommunicator.GetAllDividendAccounts(ctx)

	// check valid checkpoint
	// generate proposer for validator set
	stakeSim.LoadValidatorSet(require, 2, stakingKeeper, ctx, false, 10)
	stakingKeeper.IncrementAccum(ctx, 1)

	lastCheckpoint, err := keeper.GetLastCheckpoint(ctx)
	if err == nil {
		start = start + lastCheckpoint.EndBlock + 1
	}

	header, err := chSim.GenRandCheckpoint(start, maxSize, params.MaxCheckpointLength)
	require.NoError(err)

	// add current proposer to header
	header.Proposer = stakingKeeper.GetValidatorSet(ctx).Proposer.Signer

	accRootHash, err := types.GetAccountRootHash(dividendAccounts)
	require.NoError(err)

	accountRoot := hmTypes.BytesToHeimdallHash(accRootHash)

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
		require.Nil(res)
		require.NoError(err)

		bufferedHeader, err := keeper.GetCheckpointFromBuffer(ctx)
		require.NoError(err)
		require.Empty(bufferedHeader, "Should not store state")
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

		msgCheckpoint.Type()

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

		keeper.UpdateACKCount(ctx)
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
		require.ErrorContains(err, types.ErrDisCountinuousCheckpoint.Error())
	})
}

func (s *KeeperTestSuite) TestHandleMsgCheckpointAdjustCheckpointBuffer() {
	ctx, msgServer, keeper := s.ctx, s.msgServer, s.checkpointKeeper
	require := s.Require()

	checkpoint := types.Checkpoint{
		Proposer:   common.Address{}.String(),
		StartBlock: 0,
		EndBlock:   256,
		rootHash := hmTypes.HeimdallHash{testutil.RandomBytes()},
		BorChainID: "testchainid",
		TimeStamp:  1,
	}

	err := keeper.SetCheckpointBuffer(ctx, checkpoint)
	require.NoError(err)

	checkpointAdjust := types.MsgCheckpointAdjust{
		HeaderIndex: 1,
		Proposer:    common.Address([]byte("DifferentProposer123")).String(),
		StartBlock:  0,
		EndBlock:    512,
		RootHash:   hmTypes.HeimdallHash{testutil.RandomBytes()},
	}

	_, err = msgServer.CheckpointAdjust(ctx, &checkpointAdjust)
	require.Error(err)
}

func (s *KeeperTestSuite) TestHandleMsgCheckpointAdjustSameCheckpointAsMsg() {
	ctx, msgServer, keeper := s.ctx, s.msgServer, s.checkpointKeeper
	require := s.Require()

	checkpoint := types.Checkpoint{
		Proposer:   common.Address{}.String(),
		StartBlock: 0,
		EndBlock:   256,
		rootHash := hmTypes.HeimdallHash{testutil.RandomBytes()},
		BorChainID: "testchainid",
		TimeStamp:  1,
	}

	err := keeper.AddCheckpoint(ctx, 1, checkpoint)
	require.NoError(err)

	checkpointAdjust := types.MsgCheckpointAdjust{
		HeaderIndex: 1,
		Proposer:    common.Address{}.String(),
		StartBlock:  0,
		EndBlock:    256,
		rootHash := hmTypes.HeimdallHash{testutil.RandomBytes()},
	}

	_, err = msgServer.CheckpointAdjust(ctx, &checkpointAdjust)
	require.Error(err)
}

func (s *KeeperTestSuite) TestHandleMsgCheckpointAfterBufferTimeOut() {
	ctx, msgServer, keeper := s.ctx, s.msgServer, s.checkpointKeeper
	require := s.Require()
	stakeKeeper := s.stakeKeeper
	start := uint64(0)
	maxSize := uint64(256)
	borChainId := "1234"
	params, err := keeper.GetParams(ctx)
	require.NoError(err)
	checkpointBufferTime := params.CheckpointBufferTime
	dividendAccounts := s.moduleCommunicator.GetAllDividendAccounts(ctx)

	// generate proposer for validator set
	stakeSim.LoadValidatorSet(require, 2, stakeKeeper, ctx, false, 10)
	stakeKeeper.IncrementAccum(ctx, 1)

	lastCheckpoint, err := keeper.GetLastCheckpoint(ctx)
	if err == nil {
		start = start + lastCheckpoint.EndBlock + 1
	}

	header, err := chSim.GenRandCheckpoint(start, maxSize, params.MaxCheckpointLength)
	require.NoError(err)

	// add current proposer to header
	header.Proposer = stakeKeeper.GetValidatorSet(ctx).Proposer.Signer

	accRootHash, err := types.GetAccountRootHash(dividendAccounts)
	require.NoError(err)

	accountRoot := hmTypes.BytesToHeimdallHash(accRootHash)

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

	err=keeper.SetCheckpointBuffer(ctx, header)
	require.NoError(err)

	checkpointBuffer, err := keeper.GetCheckpointFromBuffer(ctx)
	require.NoError(err)

	// set time buffered checkpoint timestamp + checkpointBufferTime
	newTime := checkpointBuffer.TimeStamp + uint64(checkpointBufferTime)
	ctx = ctx.WithBlockTime(time.Unix(int64(newTime), 0))

	// send new checkpoint which should replace old one
	// send checkpoint to handler
	_, err = msgServer.Checkpoint(ctx, &msgCheckpoint)
	require.NoError(err)
}

func (s *KeeperTestSuite) TestHandleMsgCheckpointExistInBuffer() {
	ctx, msgServer, keeper := s.ctx, s.msgServer, s.checkpointKeeper
	require := s.Require()
	stakeKeeper := s.stakeKeeper

	start := uint64(0)
	maxSize := uint64(256)
	params, err := keeper.GetParams(ctx)
	borChainId := "1234"

	require.NoError(err)
	dividendAccounts := s.moduleCommunicator.GetAllDividendAccounts(ctx)

	stakeSim.LoadValidatorSet(require, 2, stakeKeeper, ctx, false, 10)
	stakeKeeper.IncrementAccum(ctx, 1)

	lastCheckpoint, err := keeper.GetLastCheckpoint(ctx)
	if err == nil {
		start = start + lastCheckpoint.EndBlock + 1
	}

	header, err := chSim.GenRandCheckpoint(start, maxSize, params.MaxCheckpointLength)
	require.NoError(err)

	// add current proposer to header
	header.Proposer = stakeKeeper.GetValidatorSet(ctx).Proposer.Signer

	accRootHash, err := types.GetAccountRootHash(dividendAccounts)
	require.NoError(err)

	accountRoot := hmTypes.BytesToHeimdallHash(accRootHash)

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

	err=keeper.SetCheckpointBuffer(ctx, header)
	require.NoError(err)

	// send checkpoint to handler
	_, err = msgServer.Checkpoint(ctx, &msgCheckpoint)
	require.ErrorContains(err, types.ErrNoACK.Error())
}

func (s *KeeperTestSuite) TestHandleMsgCheckpointAck() {
	ctx, msgServer, keeper := s.ctx, s.msgServer, s.checkpointKeeper
	require := s.Require()
	stakingKeeper := s.stakeKeeper
	start := uint64(0)
	maxSize := uint64(256)
	params, err := keeper.GetParams(ctx)
	require.NoError(err)

	// check valid checkpoint
	// generate proposer for validator set
	stakeSim.LoadValidatorSet(require, 2, stakingKeeper, ctx, false, 10)
	stakingKeeper.IncrementAccum(ctx, 1)

	lastCheckpoint, err := keeper.GetLastCheckpoint(ctx)
	if err == nil {
		start = start + lastCheckpoint.EndBlock + 1
	}

	header, err := chSim.GenRandCheckpoint(start, maxSize, params.MaxCheckpointLength)
	require.NoError(err)

	// add current proposer to header
	header.Proposer = stakingKeeper.GetValidatorSet(ctx).Proposer.Signer

	headerId := uint64(1)

	s.Run("No checkpoint in buffer", func() {
		msgCheckpointAck := types.NewMsgCheckpointAck(
			common.Address{}.String(),
			headerId,
			header.Proposer,
			header.StartBlock,
			header.EndBlock,
			header.RootHash,
			hmTypes.HeimdallHash{testutil.RandomBytes()},
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
			hmTypes.HeimdallHash{testutil.RandomBytes()},
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
			hmTypes.HeimdallHash{testutil.RandomBytes()},
			uint64(1),
		)

		_, err = msgServer.CheckpointAck(ctx, &msgCheckpointAck)
		require.ErrorContains(err, types.ErrBadAck.Error())
	})

	s.Run("Invalid Roothash", func() {
		msgCheckpointAck := types.NewMsgCheckpointAck(
			common.Address{}.String(),
			headerId,
			header.Proposer,
			header.StartBlock,
			header.EndBlock,
			hmTypes.HeimdallHash{testutil.RandomBytes()},
			hmTypes.HeimdallHash{testutil.RandomBytes()},
			uint64(1),
		)

		_, err = msgServer.CheckpointAck(ctx, &msgCheckpointAck)
		require.ErrorContains(err, types.ErrBadAck.Error())
	})
}

func (s *KeeperTestSuite) TestHandleMsgCheckpointNoAck() {
	ctx, msgServer, keeper := s.ctx, s.msgServer, s.checkpointKeeper
	require := s.Require()
	stakeKeeper := s.stakeKeeper
	start := uint64(0)
	maxSize := uint64(256)
	params, err := keeper.GetParams(ctx)
	require.NoError(err)
	checkpointBufferTime := params.CheckpointBufferTime

	// check valid checkpoint
	// generate proposer for validator set
	stakeSim.LoadValidatorSet(require, 4, stakeKeeper, ctx, false, 10)
	stakeKeeper.IncrementAccum(ctx, 1)

	lastCheckpoint, err := keeper.GetLastCheckpoint(ctx)
	if err == nil {
		start = start + lastCheckpoint.EndBlock + 1
	}

	header, err := chSim.GenRandCheckpoint(start, maxSize, params.MaxCheckpointLength)
	require.NoError(err)

	// add current proposer to header
	header.Proposer = stakeKeeper.GetValidatorSet(ctx).Proposer.Signer

	keeper.AddCheckpoint(ctx, uint64(1), header)
	ackCount := keeper.GetACKCount(ctx)

	// set time lastCheckpoint timestamp + checkpointBufferTime-10
	newTime := lastCheckpoint.TimeStamp + uint64(checkpointBufferTime.Seconds()) - uint64(5)
	ctx = ctx.WithBlockTime(time.Unix(int64(newTime), 0))

	validatorSet := stakeKeeper.GetValidatorSet(ctx)

	//Rotate the list to get the next proposer in line
	validatorSet.IncrementProposerPriority(1)
	noAckProposer := validatorSet.Proposer.Signer

	msgNoAck := types.MsgCheckpointNoAck{
		From: noAckProposer,
	}

	_, err = msgServer.CheckpointNoAck(ctx, &msgNoAck)
	require.ErrorContains(err, types.ErrInvalidNoACK.Error())

	updatedAckCount := keeper.GetACKCount(ctx)
	require.Equal(ackCount, updatedAckCount, "Should not update state")

	// set time lastCheckpoint timestamp + noAckWaitTime
	newTime = lastCheckpoint.TimeStamp + uint64(checkpointBufferTime.Seconds())
	ctx = ctx.WithBlockTime(time.Unix(int64(newTime), 0))

	msgNoAck = types.MsgCheckpointNoAck{
		From: header.Proposer,
	}

	_, err = msgServer.CheckpointNoAck(ctx, &msgNoAck)
	require.ErrorContains(err, types.ErrInvalidNoACK.Error())

	updatedAckCount = keeper.GetACKCount(ctx)
	require.Equal(ackCount, updatedAckCount, "Should not update state")

	msgNoAck = types.MsgCheckpointNoAck{
		From: noAckProposer,
	}

	_, err = msgServer.CheckpointNoAck(ctx, &msgNoAck)
	require.NoError(err)

	updatedAckCount = keeper.GetACKCount(ctx)
	require.Equal(ackCount, updatedAckCount, "Should not update state")
}
