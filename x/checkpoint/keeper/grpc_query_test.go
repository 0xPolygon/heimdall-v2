package keeper_test

import (
	"time"

	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/ethereum/go-ethereum/common"
	"github.com/golang/mock/gomock"

	util "github.com/0xPolygon/heimdall-v2/common/address"
	cmTypes "github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
	"github.com/0xPolygon/heimdall-v2/x/checkpoint/testutil"
	chSim "github.com/0xPolygon/heimdall-v2/x/checkpoint/testutil"
	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
	stakeSim "github.com/0xPolygon/heimdall-v2/x/stake/testutil"
)

func (s *KeeperTestSuite) TestQueryParams() {
	ctx, queryClient, require := s.ctx, s.queryClient, s.Require()

	req := &types.QueryParamsRequest{}
	defaultParams := types.DefaultParams()

	res, err := queryClient.GetCheckpointParams(ctx, req)
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

	cpNumber := uint64(1)
	startBlock := uint64(0)
	endBlock := uint64(255)
	rootHash := testutil.RandomBytes()
	proposerAddress := util.FormatAddress(common.HexToAddress(AccountHash).String())
	timestamp := uint64(time.Now().Unix())

	checkpointBlock := types.CreateCheckpoint(
		cpNumber,
		startBlock,
		endBlock,
		rootHash,
		proposerAddress,
		BorChainID,
		timestamp,
	)

	err = keeper.AddCheckpoint(ctx, checkpointBlock)
	require.NoError(err)

	res, err = queryClient.GetCheckpoint(ctx, req)
	require.NoError(err)

	require.NotNil(res)
	require.Equal(res.Checkpoint, checkpointBlock)
}

func (s *KeeperTestSuite) TestQueryCheckpointBuffer() {
	ctx, require, keeper, queryClient := s.ctx, s.Require(), s.checkpointKeeper, s.queryClient

	req := &types.QueryCheckpointBufferRequest{}

	res, err := queryClient.GetCheckpointBuffer(ctx, req)
	require.Nil(err)
	require.Equal(types.Checkpoint{}, res.Checkpoint)

	startBlock := uint64(0)
	endBlock := uint64(255)
	rootHash := testutil.RandomBytes()
	proposerAddress := util.FormatAddress(common.HexToAddress(AccountHash).String())
	timestamp := uint64(time.Now().Unix())

	checkpointBlock := types.CreateCheckpoint(
		1,
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
	ctx, require, keeper, queryClient := s.ctx, s.Require(), s.checkpointKeeper, s.queryClient

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
	ctx, require, keeper := s.ctx, s.Require(), s.checkpointKeeper
	topupKeeper, stakeKeeper, queryClient, contractCaller := s.topupKeeper, s.stakeKeeper, s.queryClient, s.contractCaller

	validatorSet := stakeSim.GetRandomValidatorSet(2)
	topupKeeper.EXPECT().GetAllDividendAccounts(gomock.Any()).AnyTimes().Return(testutil.RandDividendAccounts(), nil)

	cpNumber := uint64(1)
	startBlock := uint64(0)
	endBlock := uint64(256)
	rootHash := testutil.RandomBytes()
	proposerAddress := common.HexToAddress(AccountHash).String()
	timestamp := uint64(time.Now().Unix())

	checkpointBlock := types.CreateCheckpoint(
		cpNumber,
		startBlock,
		endBlock,
		rootHash,
		proposerAddress,
		BorChainID,
		timestamp,
	)

	contractCaller.On("GetRootHash", checkpointBlock.StartBlock, checkpointBlock.EndBlock, uint64(1024)).Return(checkpointBlock.RootHash, nil)
	err := keeper.AddCheckpoint(ctx, checkpointBlock)
	require.NoError(err)

	req := types.QueryNextCheckpointRequest{BorChainId: BorChainID}

	stakeKeeper.EXPECT().GetValidatorSet(gomock.Any()).AnyTimes().Return(validatorSet, nil)
	res, err := queryClient.GetNextCheckpoint(ctx, &req)
	require.NoError(err)

	require.Equal(checkpointBlock.StartBlock, res.Checkpoint.StartBlock)
	require.Equal(checkpointBlock.EndBlock, res.Checkpoint.EndBlock)
	require.Equal(checkpointBlock.RootHash, res.Checkpoint.RootHash)
	require.Equal(checkpointBlock.BorChainId, res.Checkpoint.BorChainId)
}

func (s *KeeperTestSuite) TestHandleCurrentQueryProposer() {
	ctx, require, stakeKeeper, queryClient := s.ctx, s.Require(), s.stakeKeeper, s.queryClient

	validatorSet := stakeSim.GetRandomValidatorSet(2)

	stakeKeeper.EXPECT().GetCurrentProposer(ctx).AnyTimes().Return(validatorSet.Proposer)
	req := &types.QueryCurrentProposerRequest{}

	res, err := queryClient.GetCurrentProposer(ctx, req)
	require.NoError(err)
	require.NotNil(res)

	require.Equal(res.Validator.Signer, validatorSet.Proposer.Signer)
}

func (s *KeeperTestSuite) TestHandleQueryProposer() {
	ctx, require, queryClient, stakeKeeper := s.ctx, s.Require(), s.queryClient, s.stakeKeeper

	validatorSet := stakeSim.GetRandomValidatorSet(2)

	stakeKeeper.EXPECT().GetValidatorSet(gomock.Any()).AnyTimes().Return(validatorSet, nil)
	req := &types.QueryProposerRequest{Times: 2}

	res, err := queryClient.GetProposers(ctx, req)
	require.NoError(err)
	require.NotNil(res)

	require.Equal(len(res.Proposers), 2)

	require.Equal(res.Proposers[0].Signer, validatorSet.Proposer.Signer)
}

func (s *KeeperTestSuite) TestGetCheckpointList() {
	ctx, keeper, cmKeeper, queryClient, require := s.ctx, s.checkpointKeeper, s.cmKeeper, s.queryClient, s.Require()

	start := uint64(0)
	maxSize := uint64(256)

	cmKeeper.EXPECT().GetParams(gomock.Any()).AnyTimes().Return(cmTypes.DefaultParams(), nil)

	var checkpoints []*types.Checkpoint
	for i := 0; i < 5; i++ {
		checkpoint := chSim.GenRandCheckpoint(start, maxSize, uint64(i))
		checkpoints = append(checkpoints, &checkpoint)
	}

	expCheckpoints := make([]types.Checkpoint, 0, len(checkpoints))
	for i, cp := range checkpoints {
		expCheckpoints = append(expCheckpoints, *cp)
		cp.Id = uint64(i)
		err := keeper.AddCheckpoint(ctx, *cp)
		require.NoError(err)
	}

	res, err := queryClient.GetCheckpointList(ctx, &types.QueryCheckpointListRequest{Pagination: query.PageRequest{Limit: 5}})
	require.NoError(err)
	require.Equal(expCheckpoints, res.CheckpointList)

	res, err = queryClient.GetCheckpointList(ctx, &types.QueryCheckpointListRequest{Pagination: query.PageRequest{Limit: 2}})
	require.NoError(err)
	require.Equal(expCheckpoints[:2], res.CheckpointList)
}
