package keeper_test

import (
	"time"

	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
	"github.com/ethereum/go-ethereum/common"
)

func (s *KeeperTestSuite) TestQueryParams() {
	ctx, _, queryClient := s.ctx, s.milestoneKeeper, s.queryClient
	require := s.Require()

	req := &types.QueryParamsRequest{}

	defaultParams := types.DefaultParams()

	res, err := queryClient.Params(ctx, req)
	require.NoError(err)
	require.NotNil(res)

	require.Equal(defaultParams.MilestoneTxConfirmations, res.Params.MilestoneTxConfirmations)
	require.Equal(defaultParams.MilestoneBufferLength, res.Params.MilestoneBufferLength)
	require.Equal(defaultParams.MilestoneBufferTime, res.Params.MilestoneBufferTime)
	require.Equal(defaultParams.MinMilestoneLength, res.Params.MinMilestoneLength)
}

func (s *KeeperTestSuite) TestQueryLatestMilestone() {
	ctx, keeper, queryClient := s.ctx, s.milestoneKeeper, s.queryClient
	require := s.Require()

	reqLatest := &types.QueryLatestMilestoneRequest{}
	reqByNumber := &types.QueryMilestoneRequest{Number: uint64(1)}
	reqCount := &types.QueryCountRequest{}

	startBlock := uint64(0)
	endBlock := uint64(255)
	hash := hmTypes.HexToHeimdallHash("123")
	proposerAddress := common.HexToAddress("123").String()
	timestamp := uint64(time.Now().Unix())
	borChainId := "1234"
	milestoneID := "00000"

	milestoneBlock := types.CreateMilestone(
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

	errNew := keeper.AddMilestone(ctx, milestoneBlock)
	require.NoError(errNew)

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

	require.NoError(errNew)
	require.Equal(resCount.Count, uint64(1))
}

func (s *KeeperTestSuite) TestQueryLastNoAckMilestone() {
	ctx, keeper, queryClient := s.ctx, s.milestoneKeeper, s.queryClient
	require := s.Require()

	req := &types.QueryLatestNoAckMilestoneRequest{}

	res, err := queryClient.LatestNoAckMilestone(ctx, req)
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
	ctx, keeper, queryClient := s.ctx, s.stakeKeeper, s.queryClient
	require := s.Require()
	testutil.LoadValidatorSet(require, 4, keeper, ctx, false, 10)

	path := []string{types.QueryMilestoneProposer}

	route := fmt.Sprintf("custom/%s/%s", types.QuerierRoute, types.QueryMilestoneProposer)

	req := abci.RequestQuery{
		Path: route,
		Data: app.Codec().MustMarshalJSON(types.NewQueryProposerParams(uint64(2))),
	}
	res, err := querier(ctx, path, req)
	// check no error found
	require.NoError(t, err)

	// check response is not nil
	require.NotNil(t, res)
}


  // MilestoneProposer queries for the milestone proposer
  rpc MilestoneProposer(QueryMilestoneProposerRequest)
      returns (QueryMilestoneProposerResponse) {
    option (cosmos.query.v1.module_query_safe) = true;
    option (google.api.http).get = "/staking/milestone-proposer";
  }


  // QueryCurrentMilestoneProposerRequest is request type for the
// Query/MilestoneProposer RPC method
message QueryMilestoneProposerRequest {
	uint64 times = 1 [ (amino.dont_omitempty) = true ];
  }
  
  // QueryCurrentMilestoneProposerResponse is response type for the
  // Query/MilestoneProposer RPC method
  message QueryMilestoneProposerResponse {
	// validator defines the validator info.
	repeated Validator proposers = 1
		[ (gogoproto.nullable) = false, (amino.dont_omitempty) = true ];
  }