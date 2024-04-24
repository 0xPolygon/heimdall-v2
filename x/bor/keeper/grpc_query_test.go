package keeper_test

import (
	"strconv"

	"github.com/0xPolygon/heimdall-v2/x/bor/types"
	chainmanagertypes "github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/golang/mock/gomock"
)

func (suite *KeeperTestSuite) TestLatestSpan() {
	require := suite.Require()

	sdkCtx := sdk.UnwrapSDKContext(suite.ctx)
	height := strconv.FormatInt(sdkCtx.BlockHeight(), 10)
	var emptySpan *types.Span
	res, err := suite.queryClient.LatestSpan(suite.ctx, &types.QueryLatestSpanRequest{})
	expRes := &types.QueryLatestSpanResponse{Height: height, Span: emptySpan}
	require.NoError(err)
	require.Equal(expRes, res)

	spans := suite.genTestSpans(5)
	for _, span := range spans {
		err := suite.borKeeper.AddNewSpan(suite.ctx, span)
		require.NoError(err)
	}

	res, err = suite.queryClient.LatestSpan(suite.ctx, &types.QueryLatestSpanRequest{})
	expRes = &types.QueryLatestSpanResponse{Height: height, Span: spans[len(spans)-1]}
	require.NoError(err)
	require.Equal(expRes, res)
}

func (suite *KeeperTestSuite) TestNextSpan() {
	require := suite.Require()

	sdkCtx := sdk.UnwrapSDKContext(suite.ctx)
	height := strconv.FormatInt(sdkCtx.BlockHeight(), 10)

	valset, vals := suite.genTestValidators()
	params := types.DefaultParams()
	params.ProducerCount = 5
	err := suite.borKeeper.SetParams(suite.ctx, params)
	require.NoError(err)

	// TODO HV2: uncomment when contract caller is merged
	// lastEthBlock, err := suite.borKeeper.GetLastEthBlock(suite.ctx)
	// require.NoError(err)
	// suite.contractCaller.EXPECT().GetMainChainBlock(gomock.Any()).Return(lastEthBlock.Add(lastEthBlock, big.NewInt(1)), nil).AnyTimes()

	suite.chainManagerKeeper.EXPECT().GetParams(suite.ctx).Return(chainmanagertypes.DefaultParams(), nil).AnyTimes()
	suite.stakeKeeper.EXPECT().GetValidatorSet(suite.ctx).Return(valset).AnyTimes()
	suite.stakeKeeper.EXPECT().GetSpanEligibleValidators(suite.ctx).Return(vals).AnyTimes()
	suite.stakeKeeper.EXPECT().GetValidatorFromValID(suite.ctx, gomock.Any()).AnyTimes()

	req := &types.QueryNextSpanRequest{
		SpanId:     1,
		StartBlock: 100,
		BorChainId: "test-chain-id",
	}

	res, err := suite.queryClient.NextSpan(suite.ctx, req)
	require.NoError(err)

	expRes := &types.QueryNextSpanResponse{
		Height: height,
		Span: &types.Span{
			Id:                req.SpanId,
			StartBlock:        req.StartBlock,
			EndBlock:          req.StartBlock + params.SpanDuration - 1,
			ValidatorSet:      valset,
			SelectedProducers: vals,
			ChainId:           req.BorChainId,
		},
	}

	require.Equal(expRes, res)
}

func (suite *KeeperTestSuite) TestNextSpanSeed() {
	require := suite.Require()

	/*
		TODO HV2: uncomment when contract caller is merged
		lastEthBlock, err := suite.borKeeper.GetLastEthBlock(suite.ctx)
		require.NoError(err)
		incEthBlock := lastEthBlock.Add(lastEthBlock, big.NewInt(1))
		suite.contractCaller.EXPECT().GetMainChainBlock(gomock.Any()).Return(incEthBlock), nil).AnyTimes()
	*/

	// sdkCtx := sdk.UnwrapSDKContext(suite.ctx)
	// height := strconv.FormatInt(sdkCtx.BlockHeight(), 10)

	_, err := suite.queryClient.NextSpanSeed(suite.ctx, &types.QueryNextSpanSeedRequest{})
	require.NoError(err)

	// TODO HV2: uncomment when contract caller is merged
	// require.Equal(&types.QueryNextSpanSeedResponse{Height: height, Seed: incEthBlock.Hash().String()}, res)
}

func (suite *KeeperTestSuite) TestParams() {
	require := suite.Require()
	sdkCtx := sdk.UnwrapSDKContext(suite.ctx)
	height := strconv.FormatInt(sdkCtx.BlockHeight(), 10)

	params := types.DefaultParams()
	err := suite.borKeeper.SetParams(suite.ctx, params)
	require.NoError(err)

	res, err := suite.queryClient.Params(suite.ctx, &types.QueryParamsRequest{})
	require.NoError(err)
	require.Equal(&types.QueryParamsResponse{Height: height, Params: &params}, res)
}

func (suite *KeeperTestSuite) TestSpanById() {
	require := suite.Require()
	sdkCtx := sdk.UnwrapSDKContext(suite.ctx)
	height := strconv.FormatInt(sdkCtx.BlockHeight(), 10)

	spans := suite.genTestSpans(1)
	err := suite.borKeeper.AddNewSpan(suite.ctx, spans[0])
	require.NoError(err)

	req := &types.QuerySpanByIdRequest{SpanId: "1"}
	res, err := suite.queryClient.SpanById(suite.ctx, req)
	require.NoError(err)
	require.Equal(&types.QuerySpanByIdResponse{Height: height, Span: spans[0]}, res)
}

func (suite *KeeperTestSuite) TestSpanList() {
	require := suite.Require()
	sdkCtx := sdk.UnwrapSDKContext(suite.ctx)
	height := strconv.FormatInt(sdkCtx.BlockHeight(), 10)

	spans := suite.genTestSpans(5)
	expSpans := make([]types.Span, 0, len(spans))
	for _, span := range spans {
		expSpans = append(expSpans, *span)
		err := suite.borKeeper.AddNewSpan(suite.ctx, span)
		require.NoError(err)
	}

	res, err := suite.queryClient.SpanList(suite.ctx, &types.QuerySpanListRequest{Page: 1, Limit: 5})
	require.NoError(err)
	require.Equal(&types.QuerySpanListResponse{Height: height, SpanList: expSpans}, res)

	res, err = suite.queryClient.SpanList(suite.ctx, &types.QuerySpanListRequest{Page: 1, Limit: 2})
	require.NoError(err)
	require.Equal(&types.QuerySpanListResponse{Height: height, SpanList: expSpans[:2]}, res)
}
