package keeper_test

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/golang/mock/gomock"

	"github.com/0xPolygon/heimdall-v2/x/checkpoint/testutil"
	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
	stakeSim "github.com/0xPolygon/heimdall-v2/x/stake/testutil"
)

func (s *KeeperTestSuite) TestQueryParams() {
	ctx, queryClient, require := s.ctx, s.queryClient, s.Require()

	req := &types.QueryParamsRequest{}
	defaultParams := types.DefaultParams()

	res, err := queryClient.GetParams(ctx, req)
	require.NoError(err)
	require.NotNil(res)
	require.True(defaultParams.Equal(res.Params))
}

func (s *KeeperTestSuite) TestQueryAckCount() {
	ctx, keeper, queryClient, require := s.ctx, s.checkpointKeeper, s.queryClient, s.Require()

	req := &types.QueryAckCountRequest{}
	ackCount := uint64(1)

	err := keeper.UpdateAckCountWithValue(ctx, ackCount)
	require.NoError(err)

	res, err := queryClient.GetAckCount(ctx, req)
	require.NoError(err)
	require.NotNil(res)

	actualAckCount := res.GetAckCount()
	require.Equal(actualAckCount, ackCount)
}

func (s *KeeperTestSuite) TestQueryCheckpoint() {
	ctx, keeper, queryClient, require := s.ctx, s.checkpointKeeper, s.queryClient, s.Require()

	req := &types.QueryCheckpointRequest{Number: uint64(1)}

	res, err := queryClient.GetCheckpoint(ctx, req)
	require.NotNil(err)
	require.Nil(res)

	headerNumber := uint64(1)
	startBlock := uint64(0)
	endBlock := uint64(255)
	rootHash := testutil.RandomBytes()
	proposerAddress := common.HexToAddress(AccountHash).String()
	timestamp := uint64(time.Now().Unix())

	checkpointBlock := types.CreateCheckpoint(
		startBlock,
		endBlock,
		rootHash,
		proposerAddress,
		BorChainID,
		timestamp,
	)

	err = keeper.AddCheckpoint(ctx, headerNumber, checkpointBlock)
	require.NoError(err)

	res, err = queryClient.GetCheckpoint(ctx, req)
	require.NoError(err)

	require.NotNil(res)
	require.Equal(res.Checkpoint, checkpointBlock)
}

func (s *KeeperTestSuite) TestQueryCheckpointBuffer() {
	ctx, keeper, queryClient := s.ctx, s.checkpointKeeper, s.queryClient
	require := s.Require()

	req := &types.QueryCheckpointBufferRequest{}

	res, err := queryClient.GetCheckpointBuffer(ctx, req)
	require.NotNil(err)
	require.Nil(res)

	startBlock := uint64(0)
	endBlock := uint64(255)
	rootHash := testutil.RandomBytes()
	proposerAddress := common.HexToAddress(AccountHash).String()
	timestamp := uint64(time.Now().Unix())

	checkpointBlock := types.CreateCheckpoint(
		startBlock,
		endBlock,
		rootHash,
		proposerAddress,
		BorChainID,
		timestamp,
	)
	err = keeper.SetCheckpointBuffer(ctx, checkpointBlock)
	require.NoError(err)

	res, err = queryClient.GetCheckpointBuffer(ctx, req)

	require.NoError(err)
	require.Equal(res.Checkpoint, checkpointBlock)
}

func (s *KeeperTestSuite) TestQueryLastNoAck() {
	ctx, keeper, queryClient := s.ctx, s.checkpointKeeper, s.queryClient
	require := s.Require()

	noAck := uint64(time.Now().Unix())
	err := keeper.SetLastNoAck(ctx, noAck)
	require.NoError(err)

	req := &types.QueryLastNoAckRequest{}

	res, err := queryClient.GetLastNoAck(ctx, req)
	require.NoError(err)
	require.NotNil(res)

	require.Equal(res.LastNoAckId, noAck)
}

func (s *KeeperTestSuite) TestQueryNextCheckpoint() {
	ctx, keeper, queryClient := s.ctx, s.checkpointKeeper, s.queryClient
	require := s.Require()

	validatorSet := stakeSim.GetRandomValidatorSet(2)
	s.topupKeeper.EXPECT().GetAllDividendAccounts(gomock.Any()).AnyTimes().Return(testutil.RandDividendAccounts(), nil)

	headerNumber := uint64(1)
	startBlock := uint64(0)
	endBlock := uint64(256)
	rootHash := testutil.RandomBytes()
	proposerAddress := common.HexToAddress(AccountHash).String()
	timestamp := uint64(time.Now().Unix())

	checkpointBlock := types.CreateCheckpoint(
		startBlock,
		endBlock,
		rootHash,
		proposerAddress,
		BorChainID,
		timestamp,
	)

	s.contractCaller.On("GetRootHash", checkpointBlock.StartBlock, checkpointBlock.EndBlock, uint64(1024)).Return(checkpointBlock.RootHash, nil)
	err := keeper.AddCheckpoint(ctx, headerNumber, checkpointBlock)
	require.NoError(err)

	req := types.QueryNextCheckpointRequest{BorChainId: BorChainID}

	s.stakeKeeper.EXPECT().GetValidatorSet(gomock.Any()).AnyTimes().Return(validatorSet, nil)
	res, err := queryClient.GetNextCheckpoint(ctx, &req)
	require.NoError(err)

	require.Equal(checkpointBlock.StartBlock, res.Checkpoint.StartBlock)
	require.Equal(checkpointBlock.EndBlock, res.Checkpoint.EndBlock)
	require.Equal(checkpointBlock.RootHash, res.Checkpoint.RootHash)
	require.Equal(checkpointBlock.BorChainId, res.Checkpoint.BorChainId)
}

func (s *KeeperTestSuite) TestHandleCurrentQueryProposer() {
	ctx, queryClient := s.ctx, s.queryClient
	require := s.Require()

	validatorSet := stakeSim.GetRandomValidatorSet(2)

	s.stakeKeeper.EXPECT().GetCurrentProposer(ctx).AnyTimes().Return(validatorSet.Proposer)
	req := &types.QueryCurrentProposerRequest{}

	res, err := queryClient.GetCurrentProposer(ctx, req)
	require.NoError(err)
	require.NotNil(res)

	require.Equal(res.Validator.Signer, validatorSet.Proposer.Signer)
}

func (s *KeeperTestSuite) TestHandleQueryProposer() {
	ctx, queryClient := s.ctx, s.queryClient
	require := s.Require()

	validatorSet := stakeSim.GetRandomValidatorSet(2)

	s.stakeKeeper.EXPECT().GetValidatorSet(gomock.Any()).AnyTimes().Return(validatorSet, nil)
	req := &types.QueryProposerRequest{Times: 2}

	res, err := queryClient.GetProposer(ctx, req)
	require.NoError(err)
	require.NotNil(res)

	require.Equal(len(res.Proposers), 2)

	require.Equal(res.Proposers[0].Signer, validatorSet.Proposer.Signer)
}
