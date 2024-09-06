package keeper_test

import (
	"math/big"

	"github.com/0xPolygon/heimdall-v2/x/bor/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/golang/mock/gomock"
)

func (suite *KeeperTestSuite) TestGetLatestSpan() {
	require := suite.Require()
	ctx := suite.ctx
	res, err := suite.queryClient.GetLatestSpan(ctx, &types.QueryLatestSpanRequest{})
	require.NoError(err)
	require.Empty(res)

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

	lastEthBlock := big.NewInt(1)
	lastEthBlockHeader := &ethTypes.Header{Number: big.NewInt(1)}
	suite.contractCaller.On("GetMainChainBlock", lastEthBlock).Return(lastEthBlockHeader, nil).Times(1)

	suite.stakeKeeper.EXPECT().GetValidatorSet(ctx).Return(valSet, nil).Times(1)
	suite.stakeKeeper.EXPECT().GetSpanEligibleValidators(ctx).Return(vals).Times(1)

	// this actually doesn't get called because in this case spanEligibleValidators == producerCount
	suite.stakeKeeper.EXPECT().GetValidatorFromValID(ctx, gomock.Any()).AnyTimes()

	req := &types.QueryNextSpanRequest{
		SpanId:     2,
		StartBlock: 102,
		BorChainId: firstSpan[0].ChainId,
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
	lastEthBlock := big.NewInt(100)
	err := suite.borKeeper.SetLastEthBlock(ctx, lastEthBlock)
	require.NoError(err)
	nextEthBlock := lastEthBlock.Add(lastEthBlock, big.NewInt(1))
	nextEthBlockHeader := &ethTypes.Header{Number: nextEthBlock}
	suite.contractCaller.On("GetMainChainBlock", nextEthBlock).Return(nextEthBlockHeader, nil).Times(1)

	res, err := suite.queryClient.GetNextSpanSeed(ctx, &types.QueryNextSpanSeedRequest{})
	require.NoError(err)
	require.Equal(&types.QueryNextSpanSeedResponse{Seed: nextEthBlockHeader.Hash().String()}, res)
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
	require.Equal(expSpans, res.SpanList)

	res, err = suite.queryClient.GetSpanList(ctx, &types.QuerySpanListRequest{Pagination: &query.PageRequest{Limit: 2}})
	require.NoError(err)
	require.Equal(expSpans[:2], res.SpanList)
}
