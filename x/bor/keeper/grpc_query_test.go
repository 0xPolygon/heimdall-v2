package keeper_test

import (
	"github.com/0xPolygon/heimdall-v2/x/bor/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/golang/mock/gomock"
)

func (suite *KeeperTestSuite) TestGetLatestSpan() {
	require := suite.Require()
	ctx := suite.ctx
	res, err := suite.queryClient.GetLatestSpan(ctx, &types.QueryLatestSpanRequest{})
	require.NoError(err)
	require.Nil(res)

	spans := suite.genTestSpans(5)
	for _, span := range spans {
		err := suite.borKeeper.AddNewSpan(ctx, span)
		require.NoError(err)
	}

	res, err = suite.queryClient.GetLatestSpan(ctx, &types.QueryLatestSpanRequest{})
	expRes := &types.QueryLatestSpanResponse{Span: spans[len(spans)-1]}
	require.NoError(err)
	require.Equal(expRes, res)
}

func (suite *KeeperTestSuite) TestGetNextSpan() {
	require := suite.Require()
	ctx := suite.ctx

	valSet, vals := suite.genTestValidators()
	params := types.DefaultParams()
	params.ProducerCount = 5
	err := suite.borKeeper.SetParams(ctx, params)
	require.NoError(err)

	firstSpan := suite.genTestSpans(1)
	err = suite.borKeeper.AddNewSpan(ctx, firstSpan[0])
	require.NoError(err)

	// TODO HV2: uncomment when contract caller is merged
	// lastEthBlock, err := suite.borKeeper.GetLastEthBlock(ctx)
	// require.NoError(err)
	// suite.contractCaller.EXPECT().GetMainChainBlock(gomock.Any()).Return(lastEthBlock.Add(lastEthBlock, big.NewInt(1)), nil).AnyTimes()

	// suite.chainManagerKeeper.EXPECT().GetParams(ctx).Return(chainmanagertypes.DefaultParams(), nil).Times(1)
	suite.stakeKeeper.EXPECT().GetValidatorSet(ctx).Return(valSet).Times(1)
	suite.stakeKeeper.EXPECT().GetSpanEligibleValidators(ctx).Return(vals).Times(1)
	suite.stakeKeeper.EXPECT().GetValidatorFromValID(ctx, gomock.Any()).Times(1)

	req := &types.QueryNextSpanRequest{
		SpanId:     1,
		StartBlock: 100,
		BorChainId: "test-chain-id",
	}

	res, err := suite.queryClient.GetNextSpan(ctx, req)
	require.NoError(err)

	expRes := &types.QueryNextSpanResponse{
		Span: &types.Span{
			Id:                req.SpanId,
			StartBlock:        req.StartBlock,
			EndBlock:          req.StartBlock + params.SpanDuration - 1,
			ValidatorSet:      valSet,
			SelectedProducers: vals,
			ChainId:           req.BorChainId,
		},
	}

	require.Equal(expRes, res)
}

func (suite *KeeperTestSuite) TestGetNextSpanSeed() {
	require := suite.Require()
	ctx := suite.ctx

	/*
		TODO HV2: uncomment when contract caller is merged
		lastEthBlock, err := suite.borKeeper.GetLastEthBlock(ctx)
		require.NoError(err)
		incEthBlock := lastEthBlock.Add(lastEthBlock, big.NewInt(1))
		suite.contractCaller.EXPECT().GetMainChainBlock(gomock.Any()).Return(incEthBlock), nil).AnyTimes()
	*/

	// height := strconv.FormatInt(ctx.BlockHeight(), 10)

	_, err := suite.queryClient.GetNextSpanSeed(ctx, &types.QueryNextSpanSeedRequest{})
	require.NoError(err)

	// TODO HV2: uncomment when contract caller is merged
	// require.Equal(&types.QueryNextSpanSeedResponse{Height: height, Seed: incEthBlock.Hash().String()}, res)
}

func (suite *KeeperTestSuite) TestGetParams() {
	require := suite.Require()
	ctx := suite.ctx

	params := types.DefaultParams()
	err := suite.borKeeper.SetParams(ctx, params)
	require.NoError(err)

	res, err := suite.queryClient.GetParams(ctx, &types.QueryParamsRequest{})
	require.NoError(err)
	require.Equal(&types.QueryParamsResponse{Params: &params}, res)
}

func (suite *KeeperTestSuite) TestGetSpanById() {
	require := suite.Require()
	ctx := suite.ctx

	spans := suite.genTestSpans(1)
	err := suite.borKeeper.AddNewSpan(ctx, spans[0])
	require.NoError(err)

	req := &types.QuerySpanByIdRequest{Id: "1"}
	res, err := suite.queryClient.GetSpanById(ctx, req)
	require.NoError(err)
	require.Equal(&types.QuerySpanByIdResponse{Span: spans[0]}, res)
}

func (suite *KeeperTestSuite) TestGetSpanList() {
	require := suite.Require()
	ctx := suite.ctx

	spans := suite.genTestSpans(5)
	expSpans := make([]types.Span, 0, len(spans))
	for _, span := range spans {
		expSpans = append(expSpans, *span)
		err := suite.borKeeper.AddNewSpan(ctx, span)
		require.NoError(err)
	}

	res, err := suite.queryClient.GetSpanList(ctx, &types.QuerySpanListRequest{Pagination: &query.PageRequest{Limit: 5}})
	require.NoError(err)
	require.Equal(&types.QuerySpanListResponse{SpanList: expSpans}, res)

	res, err = suite.queryClient.GetSpanList(ctx, &types.QuerySpanListRequest{Pagination: &query.PageRequest{Limit: 2}})
	require.NoError(err)
	require.Equal(&types.QuerySpanListResponse{SpanList: expSpans[:2]}, res)
}
