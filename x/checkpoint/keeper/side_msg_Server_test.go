package keeper_test

import (
	"github.com/0xPolygon/heimdall-v2/contracts/rootchain"
	"github.com/ethereum/go-ethereum/common"
	borCommon "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/mock"

	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	chSim "github.com/0xPolygon/heimdall-v2/x/checkpoint/testutil"
	stakeSim "github.com/0xPolygon/heimdall-v2/x/stake/testutil"

	hmModule "github.com/0xPolygon/heimdall-v2/module"
	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
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

// test handler for message
func (s *KeeperTestSuite) TestHandleMsgCheckpointAdjustSuccess() {
	ctx, msgServer, keeper := s.ctx, s.msgServer, s.checkpointKeeper
	require := s.Require()

	checkpoint := types.Checkpoint{
		Proposer:   common.HexToAddress("0xdummyAddress123").String(),
		StartBlock: 0,
		EndBlock:   256,
		RootHash:   hmTypes.HexToHeimdallHash("123"),
		BorChainID: "testchainid",
		TimeStamp:  1,
	}
	err := keeper.AddCheckpoint(ctx, 1, checkpoint)
	require.NoError(err)

	checkpointAdjust := types.MsgCheckpointAdjust{
		HeaderIndex: 1,
		Proposer:    common.HexToAddress("0xdummyAddress456").String(),
		StartBlock:  0,
		EndBlock:    512,
		RootHash:    hmTypes.HexToHeimdallHash("456"),
	}

	checkpointAdjust.String()

	rootchainInstance := &rootchain.Rootchain{}
	s.contractCaller.On("GetRootChainInstance", mock.Anything).Return(rootchainInstance, nil)
	s.contractCaller.On("GetHeaderInfo", mock.Anything, mock.Anything, mock.Anything).Return(borCommon.HexToHash("456"), uint64(0), uint64(512), uint64(1), common.HexToAddress("0xdummyAddress456").String(), nil)

	msgServer.CheckpointAdjust(ctx, &checkpointAdjust)
	sideResult := s.sideHandler(ctx, &checkpointAdjust)
	s.postHandler(ctx, &checkpointAdjust, sideResult)

	responseCheckpoint, _ := keeper.GetCheckpointByNumber(ctx, 1)
	require.Equal(responseCheckpoint.EndBlock, uint64(512))
	require.Equal(responseCheckpoint.Proposer, common.HexToAddress("0xdummyAddress456").String())
	require.Equal(responseCheckpoint.RootHash, hmTypes.HexToHeimdallHash("456"))
}

func (s *KeeperTestSuite) TestHandleMsgCheckpointAdjustSameCheckpointAsRootChain() {
	ctx, msgServer, keeper := s.ctx, s.msgServer, s.checkpointKeeper
	require := s.Require()

	checkpoint := types.Checkpoint{
		Proposer:   common.HexToAddress("0xdummyAddress123").String(),
		StartBlock: 0,
		EndBlock:   256,
		RootHash:   hmTypes.HexToHeimdallHash("123"),
		BorChainID: "testchainid",
		TimeStamp:  1,
	}
	err := keeper.AddCheckpoint(ctx, 1, checkpoint)
	require.NoError(err)

	checkpointAdjust := types.MsgCheckpointAdjust{
		HeaderIndex: 1,
		Proposer:    common.HexToAddress("0xdummyAddress123").String(),
		StartBlock:  0,
		EndBlock:    256,
		RootHash:    hmTypes.HexToHeimdallHash("456"),
	}
	rootchainInstance := &rootchain.Rootchain{}
	s.contractCaller.On("GetRootChainInstance", mock.Anything).Return(rootchainInstance, nil)
	s.contractCaller.On("GetHeaderInfo", mock.Anything, mock.Anything, mock.Anything).Return(borCommon.HexToHash("123"), uint64(0), uint64(256), uint64(1), common.HexToAddress("0xdummyAddress123").String(), nil)

	msgServer.CheckpointAdjust(ctx, &checkpointAdjust)
	sideResult := s.sideHandler(ctx, &checkpointAdjust)
	require.Equal(sideResult, hmModule.Vote_VOTE_NO)
}

func (s *KeeperTestSuite) TestHandleMsgCheckpointAdjustNotSameCheckpointAsRootChain() {
	ctx, _, keeper := s.ctx, s.msgServer, s.checkpointKeeper
	require := s.Require()

	checkpoint := types.Checkpoint{
		Proposer:   common.HexToAddress("0xdummyAddress123").String(),
		StartBlock: 0,
		EndBlock:   256,
		RootHash:   hmTypes.HexToHeimdallHash("123"),
		BorChainID: "testchainid",
		TimeStamp:  1,
	}
	err := keeper.AddCheckpoint(ctx, 1, checkpoint)
	require.NoError(err)

	checkpointAdjust := types.MsgCheckpointAdjust{
		HeaderIndex: 1,
		Proposer:    common.HexToAddress("0xdummyAddress123").String(),
		StartBlock:  0,
		EndBlock:    256,
		RootHash:    hmTypes.HexToHeimdallHash("123"),
	}

	rootchainInstance := &rootchain.Rootchain{}
	s.contractCaller.On("GetRootChainInstance", mock.Anything).Return(rootchainInstance, nil)
	s.contractCaller.On("GetHeaderInfo", mock.Anything, mock.Anything, mock.Anything).Return(borCommon.HexToHash("222"), uint64(0), uint64(256), uint64(1), common.HexToAddress("0xdummyAddress123").String(), nil)

	sideResult := s.sideHandler(ctx, &checkpointAdjust)
	require.Equal(sideResult, hmModule.Vote_VOTE_NO)
}

func (s *KeeperTestSuite) TestSideHandleMsgCheckpoint() {
	ctx, _, keeper := s.ctx, s.msgServer, s.checkpointKeeper
	require := s.Require()

	start := uint64(0)
	maxSize := uint64(256)
	params, err := keeper.GetParams(ctx)
	require.NoError(err)

	header, err := chSim.GenRandCheckpoint(start, maxSize, params.MaxCheckpointLength)
	require.NoError(err)

	borChainId := "1234"

	chainParams, err := s.cmKeeper.GetParams(ctx)
	require.NoError(err)

	maticTxConfirmations := chainParams.BorChainTxConfirmations

	s.Run("Success", func() {
		s.contractCaller.Mock = mock.Mock{}

		// create checkpoint msg
		msgCheckpoint := types.NewMsgCheckpointBlock(
			header.Proposer,
			header.StartBlock,
			header.EndBlock,
			header.RootHash,
			header.RootHash,
			borChainId,
		)

		s.contractCaller.On("CheckIfBlocksExist", header.EndBlock+maticTxConfirmations).Return(true)
		s.contractCaller.On("GetRootHash", header.StartBlock, header.EndBlock, uint64(1024)).Return(header.RootHash.Bytes(), nil)

		result := s.sideHandler(ctx, &msgCheckpoint)
		require.Equal(result, hmModule.Vote_VOTE_YES)

		bufferedHeader, _ := keeper.GetCheckpointFromBuffer(ctx)
		require.Nil(bufferedHeader, "Should not store state")
	})

	s.Run("No Roothash", func() {
		s.contractCaller.Mock = mock.Mock{}

		// create checkpoint msg
		msgCheckpoint := types.NewMsgCheckpointBlock(
			header.Proposer,
			header.StartBlock,
			header.EndBlock,
			header.RootHash,
			header.RootHash,
			borChainId,
		)

		s.contractCaller.On("CheckIfBlocksExist", header.EndBlock+maticTxConfirmations).Return(true)
		s.contractCaller.On("GetRootHash", header.StartBlock, header.EndBlock, uint64(1024)).Return(nil, nil)

		result := s.sideHandler(ctx, &msgCheckpoint)
		require.Equal(result, hmModule.Vote_VOTE_NO, "Side tx handler should Fail")

		bufferedHeader, err := keeper.GetCheckpointFromBuffer(ctx)
		require.Error(err)
		require.Nil(bufferedHeader, "Should not store state")
	})

	s.Run("invalid checkpoint", func() {
		s.contractCaller.Mock = mock.Mock{}

		// create checkpoint msg
		msgCheckpoint := types.NewMsgCheckpointBlock(
			header.Proposer,
			header.StartBlock,
			header.EndBlock,
			header.RootHash,
			header.RootHash,
			borChainId,
		)

		s.contractCaller.On("CheckIfBlocksExist", header.EndBlock+maticTxConfirmations).Return(true)
		s.contractCaller.On("GetRootHash", header.StartBlock, header.EndBlock, uint64(1024)).Return([]byte{1}, nil)

		result := s.sideHandler(ctx, &msgCheckpoint)
		require.Equal(result, hmModule.Vote_VOTE_NO, "Side tx handler should fail")
	})
}

func (s *KeeperTestSuite) TestSideHandleMsgCheckpointAck() {
	ctx, _, keeper := s.ctx, s.msgServer, s.checkpointKeeper
	require := s.Require()
	start := uint64(0)
	maxSize := uint64(256)
	params, err := keeper.GetParams(ctx)
	require.NoError(err)

	header, _ := chSim.GenRandCheckpoint(start, maxSize, params.MaxCheckpointLength)
	headerId := uint64(1)

	s.Run("Success", func() {
		s.contractCaller.Mock = mock.Mock{}

		// prepare ack msg
		msgCheckpointAck := types.NewMsgCheckpointAck(
			common.HexToAddress("0xdummyAddress123").String(),
			uint64(1),
			header.Proposer,
			header.StartBlock,
			header.EndBlock,
			header.RootHash,
			hmTypes.HexToHeimdallHash("123123"),
			uint64(1),
		)
		rootchainInstance := &rootchain.Rootchain{}

		s.contractCaller.On("GetRootChainInstance", mock.Anything).Return(rootchainInstance, nil)
		s.contractCaller.On("GetHeaderInfo", headerId, rootchainInstance, params.ChildBlockInterval).Return(header.RootHash.EthHash(), header.StartBlock, header.EndBlock, header.TimeStamp, header.Proposer, nil)

		result := s.sideHandler(ctx, &msgCheckpointAck)
		require.Equal(result, hmModule.Vote_VOTE_YES, "Side tx handler should pass")

	})

	s.Run("No HeaderInfo", func() {
		s.contractCaller.Mock = mock.Mock{}

		// prepare ack msg
		msgCheckpointAck := types.NewMsgCheckpointAck(
			common.HexToAddress("0xdummyAddress123").String(),
			uint64(1),
			header.Proposer,
			header.StartBlock,
			header.EndBlock,
			hmTypes.HexToHeimdallHash("123"),
			hmTypes.HexToHeimdallHash("123123"),
			uint64(1),
		)
		rootchainInstance := &rootchain.Rootchain{}

		s.contractCaller.On("GetRootChainInstance", mock.Anything).Return(rootchainInstance, nil)
		s.contractCaller.On("GetHeaderInfo", headerId, rootchainInstance, params.ChildBlockInterval).Return(nil, header.StartBlock, header.EndBlock, header.TimeStamp, header.Proposer, nil)

		result := s.sideHandler(ctx, &msgCheckpointAck)
		require.Equal(result, hmModule.Vote_VOTE_NO, "Side tx handler should fail")

	})
}

func (s *KeeperTestSuite) TestPostHandleMsgCheckpoint() {
	ctx, _, keeper := s.ctx, s.msgServer, s.checkpointKeeper
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

	borChainId := "1234"

	s.Run("Failure", func() {
		// create checkpoint msg
		msgCheckpoint := types.NewMsgCheckpointBlock(
			header.Proposer,
			header.StartBlock,
			header.EndBlock,
			header.RootHash,
			header.RootHash,
			borChainId,
		)

		s.postHandler(ctx, &msgCheckpoint, hmModule.Vote_VOTE_NO)

		bufferedHeader, err := keeper.GetCheckpointFromBuffer(ctx)
		require.Nil(bufferedHeader)
		require.Error(err)
	})

	s.Run("Success", func() {
		// create checkpoint msg
		msgCheckpoint := types.NewMsgCheckpointBlock(
			header.Proposer,
			header.StartBlock,
			header.EndBlock,
			header.RootHash,
			header.RootHash,
			borChainId,
		)

		s.postHandler(ctx, &msgCheckpoint, hmModule.Vote_VOTE_YES)

		bufferedHeader, err := keeper.GetCheckpointFromBuffer(ctx)
		require.Equal(bufferedHeader.StartBlock, header.StartBlock)
		require.Equal(bufferedHeader.EndBlock, header.EndBlock)
		require.Equal(bufferedHeader.RootHash, header.RootHash)
		require.Equal(bufferedHeader.Proposer, header.Proposer)
		require.Equal(bufferedHeader.BorChainID, header.BorChainID)
		require.NoError(err, "Unable to set checkpoint from buffer, Error: %v", err)
	})
}

func (s *KeeperTestSuite) TestPostHandleMsgCheckpointAck() {
	ctx, _, keeper := s.ctx, s.msgServer, s.checkpointKeeper
	require := s.Require()

	start := uint64(0)
	maxSize := uint64(256)
	params, err := keeper.GetParams(ctx)
	require.NoError(err)

	header, _ := chSim.GenRandCheckpoint(start, maxSize, params.MaxCheckpointLength)
	// generate proposer for validator set
	stakeSim.LoadValidatorSet(require, 2, s.stakeKeeper, ctx, false, 10)
	s.stakeKeeper.IncrementAccum(ctx, 1)

	// send ack
	checkpointNumber := uint64(1)

	s.Run("Failure", func() {
		msgCheckpointAck := types.NewMsgCheckpointAck(
			common.HexToAddress("0xdummyAddress123").String(),
			checkpointNumber,
			header.Proposer,
			header.StartBlock,
			header.EndBlock,
			header.RootHash,
			hmTypes.HexToHeimdallHash("123123"),
			uint64(1),
		)

		s.postHandler(ctx, &msgCheckpointAck, hmModule.Vote_VOTE_NO)

		afterAckBufferedCheckpoint, _ := keeper.GetCheckpointFromBuffer(ctx)
		require.Nil(afterAckBufferedCheckpoint)
	})

	s.Run("Success", func() {
		msgCheckpoint := types.NewMsgCheckpointBlock(
			header.Proposer,
			header.StartBlock,
			header.EndBlock,
			header.RootHash,
			header.RootHash,
			"1234",
		)

		s.postHandler(ctx, &msgCheckpoint, hmModule.Vote_VOTE_YES)

		msgCheckpointAck := types.NewMsgCheckpointAck(
			common.HexToAddress("0xdummyAddress123").String(),
			checkpointNumber,
			header.Proposer,
			header.StartBlock,
			header.EndBlock,
			header.RootHash,
			hmTypes.HexToHeimdallHash("123123"),
			uint64(1),
		)

		s.postHandler(ctx, &msgCheckpointAck, hmModule.Vote_VOTE_YES)

		afterAckBufferedCheckpoint, _ := keeper.GetCheckpointFromBuffer(ctx)
		require.Nil(afterAckBufferedCheckpoint)
	})

	s.Run("Replay", func() {
		msgCheckpointAck := types.NewMsgCheckpointAck(
			common.HexToAddress("0xdummyAddress123").String(),
			checkpointNumber,
			header.Proposer,
			header.StartBlock,
			header.EndBlock,
			header.RootHash,
			hmTypes.HexToHeimdallHash("123123"),
			uint64(1),
		)

		s.postHandler(ctx, &msgCheckpointAck, hmModule.Vote_VOTE_YES)

		afterAckBufferedCheckpoint, _ := keeper.GetCheckpointFromBuffer(ctx)
		require.Nil(afterAckBufferedCheckpoint)
	})

	s.Run("InvalidEndBlock", func() {
		header2, _ := chSim.GenRandCheckpoint(header.EndBlock+1, maxSize, params.MaxCheckpointLength)
		checkpointNumber = checkpointNumber + 1
		msgCheckpoint := types.NewMsgCheckpointBlock(
			header2.Proposer,
			header2.StartBlock,
			header2.EndBlock,
			header2.RootHash,
			header2.RootHash,
			"1234",
		)

		s.postHandler(ctx, &msgCheckpoint, hmModule.Vote_VOTE_YES)

		msgCheckpointAck := types.NewMsgCheckpointAck(
			common.HexToAddress("0xdummyAddress123").String(),
			checkpointNumber,
			header2.Proposer,
			header2.StartBlock,
			header2.EndBlock,
			header2.RootHash,
			hmTypes.HexToHeimdallHash("123123"),
			uint64(1),
		)

		s.postHandler(ctx, &msgCheckpointAck, hmModule.Vote_VOTE_YES)

		afterAckBufferedCheckpoint, _ := keeper.GetCheckpointFromBuffer(ctx)
		require.Nil(afterAckBufferedCheckpoint)
	})

	s.Run("BufferCheckpoint more than Ack", func() {
		latestCheckpoint, err := keeper.GetLastCheckpoint(ctx)
		require.Nil(err)

		header5, _ := chSim.GenRandCheckpoint(latestCheckpoint.EndBlock+1, maxSize, params.MaxCheckpointLength)
		checkpointNumber = checkpointNumber + 1

		msgCheckpoint := types.NewMsgCheckpointBlock(
			header5.Proposer,
			header5.StartBlock,
			header5.EndBlock,
			header5.RootHash,
			header5.RootHash,
			"1234",
		)

		ctx = ctx.WithBlockHeight(int64(1))

		s.postHandler(ctx, &msgCheckpoint, hmModule.Vote_VOTE_YES)

		msgCheckpointAck := types.NewMsgCheckpointAck(
			common.HexToAddress("0xdummyAddress123").String(),
			checkpointNumber,
			header5.Proposer,
			header5.StartBlock,
			header5.EndBlock-1,
			header5.RootHash,
			hmTypes.HexToHeimdallHash("123123"),
			uint64(1),
		)

		s.postHandler(ctx, &msgCheckpointAck, hmModule.Vote_VOTE_YES)

		afterAckBufferedCheckpoint, _ := keeper.GetCheckpointFromBuffer(ctx)
		require.Nil(afterAckBufferedCheckpoint)

		latestCheckpoint, err = keeper.GetLastCheckpoint(ctx)
		require.Nil(err)

		require.Equal(header5.EndBlock-1, latestCheckpoint.EndBlock, "expected latest checkpoint based on ack value")
	})

	s.Run("BufferCheckpoint less than Ack", func() {
		latestCheckpoint, err := keeper.GetLastCheckpoint(ctx)
		require.Nil(err)

		header6, _ := chSim.GenRandCheckpoint(latestCheckpoint.EndBlock+1, maxSize, params.MaxCheckpointLength)
		checkpointNumber = checkpointNumber + 1

		msgCheckpoint := types.NewMsgCheckpointBlock(
			header6.Proposer,
			header6.StartBlock,
			header6.EndBlock,
			header6.RootHash,
			header6.RootHash,
			"1234",
		)

		ctx = ctx.WithBlockHeight(int64(1))

		s.postHandler(ctx, &msgCheckpoint, hmModule.Vote_VOTE_YES)

		msgCheckpointAck := types.NewMsgCheckpointAck(
			common.HexToAddress("0xdummyAddress123").String(),
			checkpointNumber,
			header6.Proposer,
			header6.StartBlock,
			header6.EndBlock+1,
			header6.RootHash,
			hmTypes.HexToHeimdallHash("123123"),
			uint64(1),
		)

		s.postHandler(ctx, &msgCheckpointAck, hmModule.Vote_VOTE_YES)

		afterAckBufferedCheckpoint, _ := keeper.GetCheckpointFromBuffer(ctx)
		require.Nil(afterAckBufferedCheckpoint)

		latestCheckpoint, err = keeper.GetLastCheckpoint(ctx)
		require.Nil(err)

		require.Equal(header6.EndBlock+1, latestCheckpoint.EndBlock, "expected latest checkpoint based on ack value")
	})
}
