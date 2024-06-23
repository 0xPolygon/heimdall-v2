package keeper_test

import (
	"time"

	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/checkpoint/testutil"
	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
	stakeSim "github.com/0xPolygon/heimdall-v2/x/stake/testutil"
	"github.com/ethereum/go-ethereum/common"
)

func (s *KeeperTestSuite) TestQueryParams() {
	ctx, queryClient := s.ctx, s.queryClient
	require := s.Require()

	req := &types.QueryParamsRequest{}

	defaultParams := types.DefaultParams()

	res, err := queryClient.Params(ctx, req)
	require.NoError(err)
	require.NotNil(res)

	require.Equal(defaultParams.AvgCheckpointLength, res.Params.AvgCheckpointLength)
	require.Equal(defaultParams.MaxCheckpointLength, res.Params.MaxCheckpointLength)
}

func (s *KeeperTestSuite) TestQueryAckCount() {
	ctx, keeper, queryClient := s.ctx, s.checkpointKeeper, s.queryClient
	require := s.Require()

	req := &types.QueryAckCountRequest{}

	ackCount := uint64(1)
	keeper.UpdateACKCountWithValue(ctx, ackCount)

	res, err := queryClient.AckCount(ctx, req)
	require.NoError(err)
	require.NotNil(res)

	actualAckCount := res.GetCount()
	require.Equal(actualAckCount, ackCount)
}

func (s *KeeperTestSuite) TestQueryCheckpoint() {
	ctx, keeper, queryClient := s.ctx, s.checkpointKeeper, s.queryClient
	require := s.Require()

	req := &types.QueryCheckpointRequest{Number: uint64(1)}

	res, err := queryClient.Checkpoint(ctx, req)
	require.NotNil(err)
	require.Nil(res)

	headerNumber := uint64(1)
	startBlock := uint64(0)
	endBlock := uint64(255)
	rootHash := hmTypes.HeimdallHash{testutil.RandomBytes()}
	proposerAddress := common.HexToAddress("0xdummyAddress123").String()
	timestamp := uint64(time.Now().Unix())
	borChainId := "1234"

	checkpointBlock := types.CreateBlock(
		startBlock,
		endBlock,
		rootHash,
		proposerAddress,
		borChainId,
		timestamp,
	)

	err = keeper.AddCheckpoint(ctx, headerNumber, checkpointBlock)
	require.NoError(err)

	res, err = queryClient.Checkpoint(ctx, req)
	require.NoError(err)

	require.NotNil(res)
	require.Equal(res.Checkpoint, checkpointBlock)
}

func (s *KeeperTestSuite) TestQueryCheckpointBuffer() {
	ctx, keeper, queryClient := s.ctx, s.checkpointKeeper, s.queryClient
	require := s.Require()

	req := &types.QueryCheckpointBufferRequest{}

	res, err := queryClient.CheckpointBuffer(ctx, req)
	require.NotNil(err)
	require.Nil(res)

	startBlock := uint64(0)
	endBlock := uint64(255)
	rootHash := hmTypes.HeimdallHash{testutil.RandomBytes()}
	proposerAddress := common.HexToAddress("0xdummyAddress123").String()
	timestamp := uint64(time.Now().Unix())
	borChainId := "1234"

	checkpointBlock := types.CreateBlock(
		startBlock,
		endBlock,
		rootHash,
		proposerAddress,
		borChainId,
		timestamp,
	)
	err = keeper.SetCheckpointBuffer(ctx, checkpointBlock)
	require.NoError(err)

	res, err = queryClient.CheckpointBuffer(ctx, req)

	require.NoError(err)
	require.NotNil(res)

	require.Equal(res.Checkpoint, checkpointBlock)
}

func (s *KeeperTestSuite) TestQueryLastNoAck() {
	ctx, keeper, queryClient := s.ctx, s.checkpointKeeper, s.queryClient
	require := s.Require()

	noAck := uint64(time.Now().Unix())
	keeper.SetLastNoAck(ctx, noAck)

	req := &types.QueryLastNoAckRequest{}

	res, err := queryClient.LastNoAck(ctx, req)
	require.NoError(err)
	require.NotNil(res)

	require.Equal(res.Result, noAck)
}

func (s *KeeperTestSuite) TestQueryNextCheckpoint() {
	ctx, keeper, queryClient := s.ctx, s.checkpointKeeper, s.queryClient
	require := s.Require()

	stakeSim.LoadValidatorSet(require, 2, s.stakeKeeper, ctx, false, 10)

	headerNumber := uint64(1)
	startBlock := uint64(0)
	endBlock := uint64(256)
	rootHash := hmTypes.HeimdallHash{testutil.RandomBytes()}
	proposerAddress := common.HexToAddress("0xdummyAddress123").String()
	timestamp := uint64(time.Now().Unix())
	borChainId := "1234"

	checkpointBlock := types.CreateBlock(
		startBlock,
		endBlock,
		rootHash,
		proposerAddress,
		borChainId,
		timestamp,
	)

	s.contractCaller.On("GetRootHash", checkpointBlock.StartBlock, checkpointBlock.EndBlock, uint64(1024)).Return(checkpointBlock.RootHash.Bytes(), nil)
	err := keeper.AddCheckpoint(ctx, headerNumber, checkpointBlock)
	require.NoError(err)

	req := types.QueryNextCheckpointRequest{BorChainId: borChainId}

	res, err := queryClient.NextCheckpoint(ctx, &req)
	require.NoError(err)

	require.Equal(checkpointBlock.StartBlock, res.Checkpoint.StartBlock)
	require.Equal(checkpointBlock.EndBlock, res.Checkpoint.EndBlock)
	require.Equal(checkpointBlock.RootHash, res.Checkpoint.RootHash)
	require.Equal(checkpointBlock.BorChainID, res.Checkpoint.BorChainID)
}

func (s *KeeperTestSuite) TestHandleCurrentQueryProposer() {
	ctx, keeper, queryClient := s.ctx, s.stakeKeeper, s.queryClient
	require := s.Require()
	validatorSet := stakeSim.LoadValidatorSet(require, 4, keeper, ctx, false, 10)
	require.NotNil(validatorSet)

	req := &types.QueryCurrentProposerRequest{}

	res, err := queryClient.CurrentProposer(ctx, req)
	// check no error found
	require.NoError(err)
	require.NotNil(res)

	require.Equal(res.Validator.Signer, validatorSet.Proposer.Signer)
}

func (s *KeeperTestSuite) TestHandleQueryProposer() {
	ctx, keeper, queryClient := s.ctx, s.stakeKeeper, s.queryClient
	require := s.Require()
	validatorSet := stakeSim.LoadValidatorSet(require, 4, keeper, ctx, false, 10)
	require.NotNil(validatorSet)

	req := &types.QueryProposerRequest{Times: 2}

	res, err := queryClient.Proposer(ctx, req)
	// check no error found
	require.NoError(err)
	require.NotNil(res)

	require.Equal(len(res.Proposers), 2)

	require.Equal(res.Proposers[0].Signer, validatorSet.Proposer.Signer)
}
