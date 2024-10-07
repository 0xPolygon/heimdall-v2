package keeper_test

import (
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/golang/mock/gomock"

	"github.com/0xPolygon/heimdall-v2/x/milestone/testutil"
	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
	stakeSim "github.com/0xPolygon/heimdall-v2/x/stake/testutil"
)

func (s *KeeperTestSuite) TestQueryParams() {
	ctx, require, queryClient := s.ctx, s.Require(), s.queryClient

	req := &types.QueryParamsRequest{}
	defaultParams := types.DefaultParams()

	res, err := queryClient.GetParams(ctx, req)
	require.NoError(err)
	require.NotNil(res)

	require.True(defaultParams.Equal(res.Params))
}

func (s *KeeperTestSuite) TestQueryLatestMilestone() {
	ctx, require, keeper, queryClient := s.ctx, s.Require(), s.milestoneKeeper, s.queryClient

	reqLatest := &types.QueryLatestMilestoneRequest{}
	reqByNumber := &types.QueryMilestoneRequest{Number: uint64(1)}
	reqCount := &types.QueryCountRequest{}

	startBlock := uint64(0)
	endBlock := uint64(255)
	hash := testutil.RandomBytes()
	proposerAddress := secp256k1.GenPrivKey().PubKey().Address().String()
	timestamp := uint64(time.Now().Unix())
	milestoneID := "00000"

	milestoneBlock := testutil.CreateMilestone(
		startBlock,
		endBlock,
		hash,
		proposerAddress,
		BorChainId,
		milestoneID,
		timestamp,
	)

	res, err := queryClient.GetLatestMilestone(ctx, reqLatest)

	require.Error(err)
	require.Nil(res)

	resByNum, err := queryClient.GetMilestoneByNumber(ctx, reqByNumber)

	require.Error(err)
	require.Nil(res)

	res, err = queryClient.GetLatestMilestone(ctx, reqLatest)

	require.Error(err)
	require.Nil(res)

	err = keeper.AddMilestone(ctx, milestoneBlock)
	require.NoError(err)

	res, err = queryClient.GetLatestMilestone(ctx, reqLatest)

	require.NoError(err)
	require.NotNil(res)
	require.Equal(res.Milestone, milestoneBlock)

	resByNum, err = queryClient.GetMilestoneByNumber(ctx, reqByNumber)

	require.NoError(err)
	require.NotNil(res)
	require.Equal(resByNum.Milestone, milestoneBlock)

	resCount, err := queryClient.GetMilestoneCount(ctx, reqCount)

	require.NoError(err)
	require.NotNil(resCount)

	require.Equal(resCount.Count, uint64(1))
}

func (s *KeeperTestSuite) TestQueryLastNoAckMilestone() {
	ctx, require, keeper, queryClient := s.ctx, s.Require(), s.milestoneKeeper, s.queryClient

	req := &types.QueryLatestNoAckMilestoneRequest{}
	res, err := queryClient.GetLatestNoAckMilestone(ctx, req)
	require.Nil(res)

	milestoneID := "00000"
	err = keeper.SetNoAckMilestone(ctx, milestoneID)
	require.NoError(err)

	res, err = queryClient.GetLatestNoAckMilestone(ctx, req)
	require.NoError(err)
	require.NotNil(res)

	require.Equal(res.Result, milestoneID)

	milestoneID = "00001"
	err = keeper.SetNoAckMilestone(ctx, milestoneID)
	require.NoError(err)

	res, err = queryClient.GetLatestNoAckMilestone(ctx, req)
	require.NoError(err)
	require.NotNil(res)

	require.Equal(res.Result, milestoneID)
}
func (s *KeeperTestSuite) TestQueryNoAckMilestoneByID() {
	ctx, require, keeper, queryClient := s.ctx, s.Require(), s.milestoneKeeper, s.queryClient

	milestoneID := "00000"
	req := &types.QueryNoAckMilestoneByIDRequest{Id: milestoneID}

	res, err := queryClient.GetNoAckMilestoneById(ctx, req)
	require.NotNil(res)
	require.Nil(err)

	require.Equal(res.Result, false)

	err = keeper.SetNoAckMilestone(ctx, milestoneID)
	require.NoError(err)

	res, err = queryClient.GetNoAckMilestoneById(ctx, req)
	require.NotNil(res)
	require.Nil(err)

	require.Equal(res.Result, true)

	milestoneID = "00001"

	err = keeper.SetNoAckMilestone(ctx, milestoneID)
	require.NoError(err)

	req = &types.QueryNoAckMilestoneByIDRequest{Id: milestoneID}

	res, err = queryClient.GetNoAckMilestoneById(ctx, req)
	require.NotNil(res)
	require.Nil(err)

	require.Equal(res.Result, true)
}

func (s *KeeperTestSuite) TestHandleQueryMilestoneProposer() {
	ctx, require, queryClient, stakeKeeper := s.ctx, s.Require(), s.queryClient, s.stakeKeeper

	validatorSet := stakeSim.GetRandomValidatorSet(2)
	stakeKeeper.EXPECT().GetMilestoneValidatorSet(gomock.Any()).AnyTimes().Return(validatorSet, nil)
	stakeKeeper.EXPECT().MilestoneIncrementAccum(gomock.Any(), gomock.Any()).AnyTimes().Return()

	req := &types.QueryMilestoneProposerRequest{Times: 1}

	res, err := queryClient.GetMilestoneProposerByTimes(ctx, req)
	require.NotNil(res)
	require.Nil(err)

	require.Equal(res.Proposers[0].Signer, validatorSet.Proposer.Signer)
}
