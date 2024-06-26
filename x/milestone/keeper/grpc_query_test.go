package keeper_test

import (
	"time"

	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/milestone/testutil"
	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
	stakeTestUtil "github.com/0xPolygon/heimdall-v2/x/stake/testutil"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
)

func (s *KeeperTestSuite) TestQueryParams() {
	ctx, queryClient := s.ctx, s.queryClient
	require := s.Require()

	req := &types.QueryParamsRequest{}

	defaultParams := types.DefaultParams()

	res, err := queryClient.Params(ctx, req)
	require.NoError(err)
	require.NotNil(res)

	require.True(defaultParams.Equal(res.Params))
}

func (s *KeeperTestSuite) TestQueryLatestMilestone() {
	ctx, keeper, queryClient := s.ctx, s.milestoneKeeper, s.queryClient
	require := s.Require()

	reqLatest := &types.QueryLatestMilestoneRequest{}
	reqByNumber := &types.QueryMilestoneRequest{Number: uint64(1)}
	reqCount := &types.QueryCountRequest{}

	startBlock := uint64(0)
	endBlock := uint64(255)
	hash := hmTypes.HeimdallHash{Hash: testutil.RandomBytes()}
	proposerAddress := secp256k1.GenPrivKey().PubKey().Address().String()
	timestamp := uint64(time.Now().Unix())
	borChainId := "1234"
	milestoneID := "00000"

	milestoneBlock := testutil.CreateMilestone(
		startBlock,
		endBlock,
		hash,
		proposerAddress,
		borChainId,
		milestoneID,
		timestamp,
	)

	res, err := queryClient.LatestMilestone(ctx, reqLatest)

	require.Error(err)
	require.Nil(res)

	resByNum, err := queryClient.Milestone(ctx, reqByNumber)

	require.Error(err)
	require.Nil(res)

	res, err = queryClient.LatestMilestone(ctx, reqLatest)

	require.Error(err)
	require.Nil(res)

	err = keeper.AddMilestone(ctx, milestoneBlock)
	require.NoError(err)

	res, err = queryClient.LatestMilestone(ctx, reqLatest)

	require.NoError(err)
	require.NotNil(res)
	require.Equal(res.Milestone, milestoneBlock)

	resByNum, err = queryClient.Milestone(ctx, reqByNumber)

	require.NoError(err)
	require.NotNil(res)
	require.Equal(resByNum.Milestone, milestoneBlock)

	resCount, err := queryClient.Count(ctx, reqCount)

	require.NoError(err)
	require.NotNil(resCount)

	require.Equal(resCount.Count, uint64(1))
}

func (s *KeeperTestSuite) TestQueryLastNoAckMilestone() {
	ctx, keeper, queryClient := s.ctx, s.milestoneKeeper, s.queryClient
	require := s.Require()

	req := &types.QueryLatestNoAckMilestoneRequest{}

	res, err := queryClient.LatestNoAckMilestone(ctx, req)
	require.NoError(err)
	require.Equal(res.Result, "")
	require.NotNil(res)

	milestoneID := "00000"
	keeper.SetNoAckMilestone(ctx, milestoneID)

	res, err = queryClient.LatestNoAckMilestone(ctx, req)
	require.NoError(err)
	require.NotNil(res)

	require.Equal(res.Result, milestoneID)

	milestoneID = "00001"
	keeper.SetNoAckMilestone(ctx, milestoneID)

	res, err = queryClient.LatestNoAckMilestone(ctx, req)
	require.NoError(err)
	require.NotNil(res)

	require.Equal(res.Result, milestoneID)
}
func (s *KeeperTestSuite) TestQueryNoAckMilestoneByID() {
	ctx, keeper, queryClient := s.ctx, s.milestoneKeeper, s.queryClient
	require := s.Require()

	milestoneID := "00000"
	req := &types.QueryNoAckMilestoneByIDRequest{Id: milestoneID}

	res, err := queryClient.NoAckMilestoneByID(ctx, req)
	require.NotNil(res)
	require.Nil(err)

	require.Equal(res.Result, false)

	keeper.SetNoAckMilestone(ctx, milestoneID)

	res, err = queryClient.NoAckMilestoneByID(ctx, req)
	require.NotNil(res)
	require.Nil(err)

	require.Equal(res.Result, true)

	milestoneID = "00001"

	keeper.SetNoAckMilestone(ctx, milestoneID)

	req = &types.QueryNoAckMilestoneByIDRequest{Id: milestoneID}

	res, err = queryClient.NoAckMilestoneByID(ctx, req)
	require.NotNil(res)
	require.Nil(err)

	require.Equal(res.Result, true)
}

func (s *KeeperTestSuite) TestHandleQueryMilestoneProposer() {
	ctx, queryClient := s.ctx, s.queryClient
	require := s.Require()

	stakingKeeper := s.stakeKeeper

	validatorSet := stakeTestUtil.LoadValidatorSet(require, 4, stakingKeeper, ctx, false, 10)

	req := &types.QueryMilestoneProposerRequest{Times: 1}

	res, err := queryClient.MilestoneProposer(ctx, req)
	require.NotNil(res)
	require.Nil(err)

	require.Equal(res.Proposers[0].Signer, validatorSet.Proposer.Signer)
}
