package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/mock"

	"github.com/0xPolygon/heimdall-v2/contracts/rootchain"
	"github.com/0xPolygon/heimdall-v2/sidetxs"
	cmTypes "github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
	"github.com/0xPolygon/heimdall-v2/x/checkpoint/testutil"
	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
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

func (s *KeeperTestSuite) TestSideHandleMsgCheckpoint() {
	ctx, require := s.ctx, s.Require()
	keeper, cmKeeper, sideHandler, contractCaller := s.checkpointKeeper, s.cmKeeper, s.sideHandler, s.contractCaller

	start := uint64(0)
	maxSize := uint64(256)

	cmKeeper.EXPECT().GetParams(gomock.Any()).AnyTimes().Return(cmTypes.DefaultParams(), nil)

	checkpoint := testutil.GenRandCheckpoint(start, maxSize, uint64(1))

	borChainId := "1234"

	chainParams, err := cmKeeper.GetParams(ctx)
	require.NoError(err)

	borChainTxConfirmations := chainParams.BorChainTxConfirmations

	s.Run("Success", func() {
		contractCaller.Mock = mock.Mock{}

		// create checkpoint msg
		msgCheckpoint := types.NewMsgCheckpointBlock(
			checkpoint.Proposer,
			checkpoint.StartBlock,
			checkpoint.EndBlock,
			checkpoint.RootHash,
			checkpoint.RootHash,
			borChainId,
		)

		contractCaller.On("CheckIfBlocksExist", checkpoint.EndBlock+borChainTxConfirmations).Return(true)
		contractCaller.On("GetRootHash", checkpoint.StartBlock, checkpoint.EndBlock, uint64(1024)).Return(checkpoint.RootHash, nil)

		result := sideHandler(ctx, msgCheckpoint)
		require.Equal(result, sidetxs.Vote_VOTE_YES)

		doExist, err := keeper.HasCheckpointInBuffer(ctx)
		require.NoError(err)
		require.False(doExist)

		_, err = keeper.GetCheckpointFromBuffer(ctx)
		require.Error(err)
	})

	s.Run("No rootHash", func() {
		contractCaller.Mock = mock.Mock{}

		// create checkpoint msg
		msgCheckpoint := types.NewMsgCheckpointBlock(
			checkpoint.Proposer,
			checkpoint.StartBlock,
			checkpoint.EndBlock,
			checkpoint.RootHash,
			checkpoint.RootHash,
			borChainId,
		)

		contractCaller.On("CheckIfBlocksExist", checkpoint.EndBlock+borChainTxConfirmations).Return(true)
		contractCaller.On("GetRootHash", checkpoint.StartBlock, checkpoint.EndBlock, uint64(1024)).Return(nil, nil)

		result := sideHandler(ctx, msgCheckpoint)
		require.Equal(result, sidetxs.Vote_VOTE_NO, "Side tx handler should Fail")

		doExist, err := keeper.HasCheckpointInBuffer(ctx)
		require.NoError(err)
		require.False(doExist)

		_, err = keeper.GetCheckpointFromBuffer(ctx)
		require.Error(err)
	})

	s.Run("invalid rootHash", func() {
		contractCaller.Mock = mock.Mock{}

		// create checkpoint msg
		msgCheckpoint := types.NewMsgCheckpointBlock(
			checkpoint.Proposer,
			checkpoint.StartBlock,
			checkpoint.EndBlock,
			checkpoint.RootHash,
			checkpoint.RootHash,
			borChainId,
		)

		contractCaller.On("CheckIfBlocksExist", checkpoint.EndBlock+borChainTxConfirmations).Return(true)
		contractCaller.On("GetRootHash", checkpoint.StartBlock, checkpoint.EndBlock, uint64(1024)).Return([]byte{1}, nil)

		result := sideHandler(ctx, msgCheckpoint)
		require.Equal(result, sidetxs.Vote_VOTE_NO, "Side tx handler should fail")

		doExist, err := keeper.HasCheckpointInBuffer(ctx)
		require.NoError(err)
		require.False(doExist)

		_, err = keeper.GetCheckpointFromBuffer(ctx)
		require.Error(err)
	})
}

func (s *KeeperTestSuite) TestSideHandleMsgCpAck() {
	ctx, require := s.ctx, s.Require()
	keeper, cmKeeper, sideHandler, contractCaller := s.checkpointKeeper, s.cmKeeper, s.sideHandler, s.contractCaller

	start := uint64(0)
	maxSize := uint64(256)
	params, err := keeper.GetParams(ctx)
	require.NoError(err)

	cmKeeper.EXPECT().GetParams(gomock.Any()).AnyTimes().Return(cmTypes.DefaultParams(), nil)

	checkpoint := testutil.GenRandCheckpoint(start, maxSize, 1)
	cpNumber := uint64(1)

	s.Run("Success", func() {
		contractCaller.Mock = mock.Mock{}

		// prepare ack msg
		MsgCpAck := types.NewMsgCpAck(
			common.HexToAddress("0xdummyAddress123").String(),
			uint64(1),
			checkpoint.Proposer,
			checkpoint.StartBlock,
			checkpoint.EndBlock,
			checkpoint.RootHash,
			testutil.RandomBytes(),
			uint64(1),
		)
		rootChainInstance := &rootchain.Rootchain{}

		contractCaller.On("GetRootChainInstance", mock.Anything).Return(rootChainInstance, nil)
		contractCaller.On("GetHeaderInfo", cpNumber, rootChainInstance, params.ChildChainBlockInterval).Return(common.Hash(checkpoint.RootHash), checkpoint.StartBlock, checkpoint.EndBlock, checkpoint.Timestamp, checkpoint.Proposer, nil)

		result := sideHandler(ctx, &MsgCpAck)
		require.Equal(result, sidetxs.Vote_VOTE_YES, "Side tx handler should pass")
	})

	s.Run("No HeaderInfo", func() {
		contractCaller.Mock = mock.Mock{}

		// prepare ack msg
		MsgCpAck := types.NewMsgCpAck(
			common.HexToAddress("0xdummyAddress123").String(),
			uint64(1),
			checkpoint.Proposer,
			checkpoint.StartBlock,
			checkpoint.EndBlock,
			testutil.RandomBytes(),
			testutil.RandomBytes(),
			uint64(1),
		)
		rootChainInstance := &rootchain.Rootchain{}

		contractCaller.On("GetRootChainInstance", mock.Anything).Return(rootChainInstance, nil)
		contractCaller.On("GetHeaderInfo", cpNumber, rootChainInstance, params.ChildChainBlockInterval).Return(nil, checkpoint.StartBlock, checkpoint.EndBlock, checkpoint.Timestamp, checkpoint.Proposer, nil)

		result := sideHandler(ctx, &MsgCpAck)
		require.Equal(result, sidetxs.Vote_VOTE_NO, "Side tx handler should fail")
	})
}

func (s *KeeperTestSuite) TestPostHandleMsgCheckpoint() {
	ctx, require, keeper := s.ctx, s.Require(), s.checkpointKeeper
	cmKeeper, stakeKeeper, postHandler := s.cmKeeper, s.stakeKeeper, s.postHandler

	start := uint64(0)
	maxSize := uint64(256)

	validatorSet := stakeSim.GetRandomValidatorSet(2)
	stakeKeeper.EXPECT().GetValidatorSet(gomock.Any()).AnyTimes().Return(validatorSet, nil)
	stakeKeeper.EXPECT().GetCurrentProposer(gomock.Any()).AnyTimes().Return(validatorSet.Proposer)
	cmKeeper.EXPECT().GetParams(gomock.Any()).AnyTimes().Return(cmTypes.DefaultParams(), nil)

	lastCheckpoint, err := keeper.GetLastCheckpoint(ctx)
	if err == nil {
		start = start + lastCheckpoint.EndBlock + 1
	}

	require.NotNil(lastCheckpoint)

	checkpoint := testutil.GenRandCheckpoint(start, maxSize, lastCheckpoint.Id+1)

	// add current proposer to checkpoint
	checkpoint.Proposer = validatorSet.Proposer.Signer

	borChainId := "1234"

	s.Run("Failure", func() {
		// create checkpoint msg
		msgCheckpoint := types.NewMsgCheckpointBlock(
			checkpoint.Proposer,
			checkpoint.StartBlock,
			checkpoint.EndBlock,
			checkpoint.RootHash,
			checkpoint.RootHash,
			borChainId,
		)

		postHandler(ctx, msgCheckpoint, sidetxs.Vote_VOTE_NO)

		doExist, err := keeper.HasCheckpointInBuffer(ctx)
		require.NoError(err)
		require.False(doExist)

		_, err = keeper.GetCheckpointFromBuffer(ctx)
		require.Error(err)
	})

	s.Run("Success", func() {
		// create checkpoint msg
		msgCheckpoint := types.NewMsgCheckpointBlock(
			checkpoint.Proposer,
			checkpoint.StartBlock,
			checkpoint.EndBlock,
			checkpoint.RootHash,
			checkpoint.RootHash,
			borChainId,
		)

		postHandler(ctx, msgCheckpoint, sidetxs.Vote_VOTE_YES)

		bufCheckpoint, err := keeper.GetCheckpointFromBuffer(ctx)
		require.Equal(bufCheckpoint.StartBlock, checkpoint.StartBlock)
		require.Equal(bufCheckpoint.EndBlock, checkpoint.EndBlock)
		require.Equal(bufCheckpoint.RootHash, checkpoint.RootHash)
		require.Equal(bufCheckpoint.Proposer, checkpoint.Proposer)
		require.Equal(bufCheckpoint.BorChainId, checkpoint.BorChainId)
		require.NoError(err, "Unable to set checkpoint from buffer, Error: %v", err)
	})
}

func (s *KeeperTestSuite) TestPostHandleMsgCpAck() {
	ctx, require, keeper := s.ctx, s.Require(), s.checkpointKeeper
	cmKeeper, stakeKeeper, postHandler := s.cmKeeper, s.stakeKeeper, s.postHandler

	start := uint64(0)
	maxSize := uint64(256)
	checkpointNumber := uint64(1)

	checkpoint := testutil.GenRandCheckpoint(start, maxSize, checkpointNumber)

	validatorSet := stakeSim.GetRandomValidatorSet(2)
	stakeKeeper.EXPECT().GetValidatorSet(gomock.Any()).AnyTimes().Return(validatorSet, nil)
	stakeKeeper.EXPECT().GetCurrentProposer(gomock.Any()).AnyTimes().Return(validatorSet.Proposer)
	stakeKeeper.EXPECT().IncrementAccum(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
	cmKeeper.EXPECT().GetParams(gomock.Any()).AnyTimes().Return(cmTypes.DefaultParams(), nil)

	s.Run("Failure", func() {
		MsgCpAck := types.NewMsgCpAck(
			common.HexToAddress("0xdummyAddress123").String(),
			checkpointNumber,
			checkpoint.Proposer,
			checkpoint.StartBlock,
			checkpoint.EndBlock,
			checkpoint.RootHash,
			testutil.RandomBytes(),
			uint64(1),
		)

		postHandler(ctx, &MsgCpAck, sidetxs.Vote_VOTE_NO)

		doExist, err := keeper.HasCheckpointInBuffer(ctx)
		require.NoError(err)
		require.False(doExist)

		_, err = keeper.GetCheckpointFromBuffer(ctx)
		require.Error(err)
	})

	s.Run("Success", func() {
		msgCheckpoint := types.NewMsgCheckpointBlock(
			checkpoint.Proposer,
			checkpoint.StartBlock,
			checkpoint.EndBlock,
			checkpoint.RootHash,
			checkpoint.RootHash,
			"1234",
		)

		postHandler(ctx, msgCheckpoint, sidetxs.Vote_VOTE_YES)

		MsgCpAck := types.NewMsgCpAck(
			common.HexToAddress("0xdummyAddress123").String(),
			checkpointNumber,
			checkpoint.Proposer,
			checkpoint.StartBlock,
			checkpoint.EndBlock,
			checkpoint.RootHash,
			testutil.RandomBytes(),

			uint64(1),
		)

		postHandler(ctx, &MsgCpAck, sidetxs.Vote_VOTE_YES)

		doExist, err := keeper.HasCheckpointInBuffer(ctx)
		require.NoError(err)
		require.False(doExist)

		_, err = keeper.GetCheckpointFromBuffer(ctx)
		require.Error(err)
	})

	s.Run("Replay", func() {
		MsgCpAck := types.NewMsgCpAck(
			common.HexToAddress("0xdummyAddress123").String(),
			checkpointNumber,
			checkpoint.Proposer,
			checkpoint.StartBlock,
			checkpoint.EndBlock,
			checkpoint.RootHash,
			testutil.RandomBytes(),
			uint64(1),
		)

		postHandler(ctx, &MsgCpAck, sidetxs.Vote_VOTE_YES)

		doExist, err := keeper.HasCheckpointInBuffer(ctx)
		require.NoError(err)
		require.False(doExist)

		_, err = keeper.GetCheckpointFromBuffer(ctx)
		require.Error(err)
	})

	s.Run("InvalidEndBlock", func() {
		checkpointNumber = checkpointNumber + 1
		checkpoint2 := testutil.GenRandCheckpoint(checkpoint.EndBlock+1, maxSize, checkpointNumber)
		msgCheckpoint := types.NewMsgCheckpointBlock(
			checkpoint2.Proposer,
			checkpoint2.StartBlock,
			checkpoint2.EndBlock,
			checkpoint2.RootHash,
			checkpoint2.RootHash,
			"1234",
		)

		postHandler(ctx, msgCheckpoint, sidetxs.Vote_VOTE_YES)

		MsgCpAck := types.NewMsgCpAck(
			common.HexToAddress("0xdummyAddress123").String(),
			checkpointNumber,
			checkpoint2.Proposer,
			checkpoint2.StartBlock,
			checkpoint2.EndBlock,
			checkpoint2.RootHash,
			testutil.RandomBytes(),
			uint64(1),
		)

		postHandler(ctx, &MsgCpAck, sidetxs.Vote_VOTE_YES)

		doExist, err := keeper.HasCheckpointInBuffer(ctx)
		require.NoError(err)
		require.False(doExist)

		_, err = keeper.GetCheckpointFromBuffer(ctx)
		require.Error(err)
	})

	s.Run("BufferCheckpoint more than Ack", func() {
		latestCheckpoint, err := keeper.GetLastCheckpoint(ctx)
		require.Nil(err)

		checkpoint5 := testutil.GenRandCheckpoint(latestCheckpoint.EndBlock+1, maxSize, latestCheckpoint.Id+1)
		checkpointNumber = checkpointNumber + 1

		msgCheckpoint := types.NewMsgCheckpointBlock(
			checkpoint5.Proposer,
			checkpoint5.StartBlock,
			checkpoint5.EndBlock,
			checkpoint5.RootHash,
			checkpoint5.RootHash,
			"1234",
		)

		ctx = ctx.WithBlockHeight(int64(1))

		postHandler(ctx, msgCheckpoint, sidetxs.Vote_VOTE_YES)

		msgCpAck := types.NewMsgCpAck(
			common.HexToAddress("0xdummyAddress123").String(),
			checkpointNumber,
			checkpoint5.Proposer,
			checkpoint5.StartBlock,
			checkpoint5.EndBlock-1,
			checkpoint5.RootHash,
			testutil.RandomBytes(),
			uint64(1),
		)

		postHandler(ctx, &msgCpAck, sidetxs.Vote_VOTE_YES)

		doExist, err := keeper.HasCheckpointInBuffer(ctx)
		require.NoError(err)
		require.False(doExist)

		_, err = keeper.GetCheckpointFromBuffer(ctx)
		require.Error(err)

		latestCheckpoint, err = keeper.GetLastCheckpoint(ctx)
		require.Nil(err)

		require.Equal(checkpoint5.EndBlock-1, latestCheckpoint.EndBlock, "expected latest checkpoint based on ack value")
	})

	s.Run("BufferCheckpoint less than Ack", func() {
		latestCheckpoint, err := keeper.GetLastCheckpoint(ctx)
		require.Nil(err)

		checkpoint6 := testutil.GenRandCheckpoint(latestCheckpoint.EndBlock+1, maxSize, latestCheckpoint.Id+1)
		checkpointNumber = checkpointNumber + 1

		msgCheckpoint := types.NewMsgCheckpointBlock(
			checkpoint6.Proposer,
			checkpoint6.StartBlock,
			checkpoint6.EndBlock,
			checkpoint6.RootHash,
			checkpoint6.RootHash,
			"1234",
		)

		ctx = ctx.WithBlockHeight(int64(1))

		postHandler(ctx, msgCheckpoint, sidetxs.Vote_VOTE_YES)

		msgCheckpointAck := types.NewMsgCpAck(
			common.HexToAddress("0xdummyAddress123").String(),
			checkpointNumber,
			checkpoint6.Proposer,
			checkpoint6.StartBlock,
			checkpoint6.EndBlock+1,
			checkpoint6.RootHash,
			testutil.RandomBytes(),
			uint64(1),
		)

		postHandler(ctx, &msgCheckpointAck, sidetxs.Vote_VOTE_YES)

		doExist, err := keeper.HasCheckpointInBuffer(ctx)
		require.NoError(err)
		require.False(doExist)

		_, err = keeper.GetCheckpointFromBuffer(ctx)
		require.Error(err)

		latestCheckpoint, err = keeper.GetLastCheckpoint(ctx)
		require.Nil(err)

		require.Equal(checkpoint6.EndBlock+1, latestCheckpoint.EndBlock, "expected latest checkpoint based on ack value")
	})
}
