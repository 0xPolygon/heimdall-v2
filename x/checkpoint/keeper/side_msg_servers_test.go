package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/mock"

	"github.com/0xPolygon/heimdall-v2/contracts/rootchain"
	hmModule "github.com/0xPolygon/heimdall-v2/module"
	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	cmTypes "github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
	"github.com/0xPolygon/heimdall-v2/x/checkpoint/testutil"
	chSim "github.com/0xPolygon/heimdall-v2/x/checkpoint/testutil"
	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
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

func (s *KeeperTestSuite) TestSideHandleMsgCheckpoint() {
	ctx, keeper := s.ctx, s.checkpointKeeper
	require := s.Require()

	start := uint64(0)
	maxSize := uint64(256)

	s.cmKeeper.EXPECT().GetParams(gomock.Any()).AnyTimes().Return(cmTypes.DefaultParams(), nil)

	header := chSim.GenRandCheckpoint(start, maxSize)

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
		s.contractCaller.On("GetRootHash", header.StartBlock, header.EndBlock, uint64(1024)).Return(header.RootHash.GetHash(), nil)

		result := s.sideHandler(ctx, &msgCheckpoint)
		require.Equal(result, hmModule.Vote_VOTE_YES)

		doExist, err := keeper.HasCheckpointInBuffer(ctx)
		require.NoError(err)
		require.False(doExist)

		_, err = keeper.GetCheckpointFromBuffer(ctx)
		require.Error(err)
	})

	s.Run("No rootHash", func() {
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

		doExist, err := keeper.HasCheckpointInBuffer(ctx)
		require.NoError(err)
		require.False(doExist)

		_, err = keeper.GetCheckpointFromBuffer(ctx)
		require.Error(err)
	})

	s.Run("invalid rootHash", func() {
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

		doExist, err := keeper.HasCheckpointInBuffer(ctx)
		require.NoError(err)
		require.False(doExist)

		_, err = keeper.GetCheckpointFromBuffer(ctx)
		require.Error(err)
	})
}

func (s *KeeperTestSuite) TestSideHandleMsgCheckpointAck() {
	ctx, keeper := s.ctx, s.checkpointKeeper
	require := s.Require()
	start := uint64(0)
	maxSize := uint64(256)
	params, err := keeper.GetParams(ctx)
	require.NoError(err)

	s.cmKeeper.EXPECT().GetParams(gomock.Any()).AnyTimes().Return(cmTypes.DefaultParams(), nil)

	header := chSim.GenRandCheckpoint(start, maxSize)
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
			hmTypes.HeimdallHash{Hash: testutil.RandomBytes()},
			uint64(1),
		)
		rootChainInstance := &rootchain.Rootchain{}

		s.contractCaller.On("GetRootChainInstance", mock.Anything).Return(rootChainInstance, nil)
		s.contractCaller.On("GetHeaderInfo", headerId, rootChainInstance, params.ChildBlockInterval).Return(common.Hash(header.RootHash.GetHash()), header.StartBlock, header.EndBlock, header.Timestamp, header.Proposer, nil)

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
			hmTypes.HeimdallHash{Hash: testutil.RandomBytes()},
			hmTypes.HeimdallHash{Hash: testutil.RandomBytes()},
			uint64(1),
		)
		rootChainInstance := &rootchain.Rootchain{}

		s.contractCaller.On("GetRootChainInstance", mock.Anything).Return(rootChainInstance, nil)
		s.contractCaller.On("GetHeaderInfo", headerId, rootChainInstance, params.ChildBlockInterval).Return(nil, header.StartBlock, header.EndBlock, header.Timestamp, header.Proposer, nil)

		result := s.sideHandler(ctx, &msgCheckpointAck)
		require.Equal(result, hmModule.Vote_VOTE_NO, "Side tx handler should fail")

	})
}

func (s *KeeperTestSuite) TestPostHandleMsgCheckpoint() {
	ctx, keeper := s.ctx, s.checkpointKeeper
	require := s.Require()

	start := uint64(0)
	maxSize := uint64(256)

	validatorSet := stakeSim.GetRandomValidatorSet(2)
	s.stakeKeeper.EXPECT().GetValidatorSet(gomock.Any()).AnyTimes().Return(validatorSet, nil)
	s.stakeKeeper.EXPECT().GetCurrentProposer(gomock.Any()).AnyTimes().Return(validatorSet.Proposer)
	s.cmKeeper.EXPECT().GetParams(gomock.Any()).AnyTimes().Return(cmTypes.DefaultParams(), nil)

	lastCheckpoint, err := keeper.GetLastCheckpoint(ctx)
	if err == nil {
		start = start + lastCheckpoint.EndBlock + 1
	}

	header := chSim.GenRandCheckpoint(start, maxSize)

	// add current proposer to header
	header.Proposer = validatorSet.Proposer.Signer

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

		doExist, err := keeper.HasCheckpointInBuffer(ctx)
		require.NoError(err)
		require.False(doExist)

		_, err = keeper.GetCheckpointFromBuffer(ctx)
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
	ctx, keeper := s.ctx, s.checkpointKeeper
	require := s.Require()

	start := uint64(0)
	maxSize := uint64(256)

	header := chSim.GenRandCheckpoint(start, maxSize)

	validatorSet := stakeSim.GetRandomValidatorSet(2)
	s.stakeKeeper.EXPECT().GetValidatorSet(gomock.Any()).AnyTimes().Return(validatorSet, nil)
	s.stakeKeeper.EXPECT().GetCurrentProposer(gomock.Any()).AnyTimes().Return(validatorSet.Proposer)
	s.stakeKeeper.EXPECT().IncrementAccum(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
	s.cmKeeper.EXPECT().GetParams(gomock.Any()).AnyTimes().Return(cmTypes.DefaultParams(), nil)

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
			hmTypes.HeimdallHash{Hash: testutil.RandomBytes()},
			uint64(1),
		)

		s.postHandler(ctx, &msgCheckpointAck, hmModule.Vote_VOTE_NO)

		doExist, err := keeper.HasCheckpointInBuffer(ctx)
		require.NoError(err)
		require.False(doExist)

		_, err = keeper.GetCheckpointFromBuffer(ctx)
		require.Error(err)
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
			hmTypes.HeimdallHash{Hash: testutil.RandomBytes()},

			uint64(1),
		)

		s.postHandler(ctx, &msgCheckpointAck, hmModule.Vote_VOTE_YES)

		doExist, err := keeper.HasCheckpointInBuffer(ctx)
		require.NoError(err)
		require.False(doExist)

		_, err = keeper.GetCheckpointFromBuffer(ctx)
		require.Error(err)
	})

	s.Run("Replay", func() {
		msgCheckpointAck := types.NewMsgCheckpointAck(
			common.HexToAddress("0xdummyAddress123").String(),
			checkpointNumber,
			header.Proposer,
			header.StartBlock,
			header.EndBlock,
			header.RootHash,
			hmTypes.HeimdallHash{Hash: testutil.RandomBytes()},
			uint64(1),
		)

		s.postHandler(ctx, &msgCheckpointAck, hmModule.Vote_VOTE_YES)

		doExist, err := keeper.HasCheckpointInBuffer(ctx)
		require.NoError(err)
		require.False(doExist)

		_, err = keeper.GetCheckpointFromBuffer(ctx)
		require.Error(err)
	})

	s.Run("InvalidEndBlock", func() {
		header2 := chSim.GenRandCheckpoint(header.EndBlock+1, maxSize)
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
			hmTypes.HeimdallHash{Hash: testutil.RandomBytes()},
			uint64(1),
		)

		s.postHandler(ctx, &msgCheckpointAck, hmModule.Vote_VOTE_YES)

		doExist, err := keeper.HasCheckpointInBuffer(ctx)
		require.NoError(err)
		require.False(doExist)

		_, err = keeper.GetCheckpointFromBuffer(ctx)
		require.Error(err)
	})

	s.Run("BufferCheckpoint more than Ack", func() {
		latestCheckpoint, err := keeper.GetLastCheckpoint(ctx)
		require.Nil(err)

		header5 := chSim.GenRandCheckpoint(latestCheckpoint.EndBlock+1, maxSize)
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
			hmTypes.HeimdallHash{Hash: testutil.RandomBytes()},
			uint64(1),
		)

		s.postHandler(ctx, &msgCheckpointAck, hmModule.Vote_VOTE_YES)

		doExist, err := keeper.HasCheckpointInBuffer(ctx)
		require.NoError(err)
		require.False(doExist)

		_, err = keeper.GetCheckpointFromBuffer(ctx)
		require.Error(err)

		latestCheckpoint, err = keeper.GetLastCheckpoint(ctx)
		require.Nil(err)

		require.Equal(header5.EndBlock-1, latestCheckpoint.EndBlock, "expected latest checkpoint based on ack value")
	})

	s.Run("BufferCheckpoint less than Ack", func() {
		latestCheckpoint, err := keeper.GetLastCheckpoint(ctx)
		require.Nil(err)

		header6 := chSim.GenRandCheckpoint(latestCheckpoint.EndBlock+1, maxSize)
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
			hmTypes.HeimdallHash{Hash: testutil.RandomBytes()},
			uint64(1),
		)

		s.postHandler(ctx, &msgCheckpointAck, hmModule.Vote_VOTE_YES)

		doExist, err := keeper.HasCheckpointInBuffer(ctx)
		require.NoError(err)
		require.False(doExist)

		_, err = keeper.GetCheckpointFromBuffer(ctx)
		require.Error(err)

		latestCheckpoint, err = keeper.GetLastCheckpoint(ctx)
		require.Nil(err)

		require.Equal(header6.EndBlock+1, latestCheckpoint.EndBlock, "expected latest checkpoint based on ack value")
	})
}
