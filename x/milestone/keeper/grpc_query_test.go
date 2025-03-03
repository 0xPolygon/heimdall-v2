package keeper_test

import (
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"

	util "github.com/0xPolygon/heimdall-v2/common/address"
	"github.com/0xPolygon/heimdall-v2/x/milestone/testutil"
	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
)

func (s *KeeperTestSuite) TestQueryParams() {
	ctx, require, queryClient := s.ctx, s.Require(), s.queryClient

	req := &types.QueryParamsRequest{}
	defaultParams := types.DefaultParams()

	res, err := queryClient.GetMilestoneParams(ctx, req)
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
	proposerAddress := util.FormatAddress(secp256k1.GenPrivKey().PubKey().Address().String())
	timestamp := uint64(time.Now().Unix())
	milestoneID := TestMilestoneID
	borChainId := "1234"

	milestoneBlock := testutil.CreateMilestone(
		startBlock,
		endBlock,
		hash,
		proposerAddress,
		borChainId,
		milestoneID,
		timestamp,
	)

	res, err := queryClient.GetLatestMilestone(ctx, reqLatest)

	require.Error(err)
	require.Nil(res)

	resByNum, err := queryClient.GetMilestoneByNumber(ctx, reqByNumber)

	require.Error(err)
	require.Nil(resByNum)

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
