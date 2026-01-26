package keeper_test

import (
	"errors"
	"math/big"

	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/mock"

	"github.com/0xPolygon/heimdall-v2/x/bor/types"
	staketypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

func (s *KeeperTestSuite) TestGetLatestSpan() {
	require, ctx, queryClient := s.Require(), s.ctx, s.queryClient

	res, _ := queryClient.GetLatestSpan(ctx, &types.QueryLatestSpanRequest{})
	require.Empty(res)

	spans := s.genTestSpans(5)
	for _, span := range spans {
		err := s.borKeeper.AddNewSpan(ctx, span)
		require.NoError(err)
	}

	res, err := queryClient.GetLatestSpan(ctx, &types.QueryLatestSpanRequest{})
	expRes := &types.QueryLatestSpanResponse{Span: *spans[len(spans)-1]}
	require.NoError(err)
	require.Equal(expRes, res)
}

func (s *KeeperTestSuite) TestGetNextSpan() {
	require, ctx, queryClient, contractCaller := s.Require(), s.ctx, s.queryClient, &s.contractCaller
	keeper, stakeKeeper := s.borKeeper, s.stakeKeeper

	valSet, vals := s.genTestValidators()
	params := types.DefaultParams()
	params.ProducerCount = 5
	err := keeper.SetParams(ctx, params)
	require.NoError(err)

	// add genesis span
	err = keeper.AddNewSpan(ctx, &types.Span{Id: 0})
	require.NoError(err)

	for _, v := range vals {
		add := common.HexToAddress(v.GetOperator())
		err = keeper.StoreSeedProducer(ctx, v.ValId, &add)
	}

	firstSpan := s.genTestSpans(1)
	err = keeper.AddNewSpan(ctx, firstSpan[0])
	require.NoError(err)
	valAddr := common.HexToAddress(vals[1].GetOperator())

	lastBorBlockHeader := &ethTypes.Header{Number: big.NewInt(0)}
	contractCaller.On("GetBorChainBlock", mock.Anything, big.NewInt(0)).Return(lastBorBlockHeader, nil).Times(1)
	contractCaller.On("GetBorChainBlockAuthor", big.NewInt(0)).Return(&valAddr, nil).Times(1)
	stakeKeeper.EXPECT().GetValidatorSet(ctx).Return(valSet, nil).Times(1)
	stakeKeeper.EXPECT().GetSpanEligibleValidators(ctx).Return(vals).Times(1)

	// this actually doesn't get called because in this case spanEligibleValidators == producerCount
	stakeKeeper.EXPECT().GetValidatorFromValID(ctx, gomock.Any()).AnyTimes()

	req := &types.QueryNextSpanRequest{
		SpanId:     2,
		StartBlock: 102,
		BorChainId: firstSpan[0].BorChainId,
	}

	res, err := queryClient.GetNextSpan(ctx, req)
	require.NoError(err)

	expRes := &types.QueryNextSpanResponse{
		Span: types.Span{
			Id:                req.SpanId,
			StartBlock:        req.StartBlock,
			EndBlock:          req.StartBlock + params.SpanDuration - 1,
			ValidatorSet:      valSet,
			SelectedProducers: vals,
			BorChainId:        req.BorChainId,
		},
	}

	require.Equal(expRes, res)
}

func (s *KeeperTestSuite) TestGetNextSpanSeed() {
	require, ctx, queryClient, contractCaller := s.Require(), s.ctx, s.queryClient, &s.contractCaller
	keeper := s.borKeeper

	// add genesis span
	err := keeper.AddNewSpan(ctx, &types.Span{Id: 0})
	require.NoError(err)

	valAddr := common.HexToAddress("0x91b54cD48FD796A5d0A120A4C5298a7fAEA59B")

	lastBorBlockHeader := &ethTypes.Header{Number: big.NewInt(1)}
	contractCaller.On("GetBorChainBlock", mock.Anything, big.NewInt(1)).Return(lastBorBlockHeader, nil).Times(1)
	contractCaller.On("GetBorChainBlockAuthor", big.NewInt(1)).Return(&valAddr, nil).Times(1)

	res, err := queryClient.GetNextSpanSeed(ctx, &types.QueryNextSpanSeedRequest{Id: 1})
	require.NoError(err)
	require.NotNil(res)
	require.Equal(&types.QueryNextSpanSeedResponse{Seed: lastBorBlockHeader.Hash().String(), SeedAuthor: valAddr.Hex()}, res)
}

func (s *KeeperTestSuite) TestGetParams() {
	require, ctx, queryClient, keeper := s.Require(), s.ctx, s.queryClient, s.borKeeper

	params := types.DefaultParams()
	err := keeper.SetParams(ctx, params)
	require.NoError(err)

	res, err := queryClient.GetBorParams(ctx, &types.QueryParamsRequest{})
	require.NoError(err)
	require.Equal(&types.QueryParamsResponse{Params: params}, res)
}

func (s *KeeperTestSuite) TestGetSpanById() {
	require, ctx, keeper, queryClient := s.Require(), s.ctx, s.borKeeper, s.queryClient

	spans := s.genTestSpans(1)
	err := keeper.AddNewSpan(ctx, spans[0])
	require.NoError(err)

	req := &types.QuerySpanByIdRequest{Id: "1"}
	res, err := queryClient.GetSpanById(ctx, req)
	require.NoError(err)
	require.Equal(&types.QuerySpanByIdResponse{Span: spans[0]}, res)
}

func (s *KeeperTestSuite) TestGetSpanList() {
	require, ctx, keeper, queryClient := s.Require(), s.ctx, s.borKeeper, s.queryClient

	spans := s.genTestSpans(5)
	expSpans := make([]types.Span, 0, len(spans))
	for _, span := range spans {
		expSpans = append(expSpans, *span)
		err := keeper.AddNewSpan(ctx, span)
		require.NoError(err)
	}

	res, err := queryClient.GetSpanList(ctx, &types.QuerySpanListRequest{Pagination: query.PageRequest{Limit: 5}})
	require.NoError(err)
	require.Equal(expSpans, res.SpanList)

	res, err = queryClient.GetSpanList(ctx, &types.QuerySpanListRequest{Pagination: query.PageRequest{Limit: 2}})
	require.NoError(err)
	require.Equal(expSpans[:2], res.SpanList)
}

func (s *KeeperTestSuite) TestGetProducerVotesByValidatorId() {
	require, ctx, queryClient, keeper := s.Require(), s.ctx, s.queryClient, s.borKeeper

	// Test case 1: Valid request
	valID := uint64(1)
	expectedVoteData := []uint64{100, 200}
	producerVotes := types.ProducerVotes{
		Votes: expectedVoteData,
	}

	// Use the actual SetProducerVotes from the keeper for setup
	err := keeper.SetProducerVotes(ctx, valID, producerVotes)
	require.NoError(err)

	req := &types.QueryProducerVotesByValidatorIdRequest{ValidatorId: valID}
	res, err := queryClient.GetProducerVotesByValidatorId(ctx, req)
	require.NoError(err)
	require.NotNil(res)
	require.Equal(expectedVoteData, res.Votes)

	// Test case 2: Empty request
	// Based on test observation, a nil request to the queryClient might result in
	// an empty (non-nil) request object at the handler, thus not triggering the req == nil check.
	// It would then proceed as if querying for validator ID 0, returning empty votes.
	var emptyRes *types.QueryProducerVotesByValidatorIdResponse
	emptyRes, err = queryClient.GetProducerVotesByValidatorId(ctx, nil)
	require.NoError(err)          // Expect no error based on observed behavior
	require.NotNil(emptyRes)      // Expect a non-nil response object
	require.Empty(emptyRes.Votes) // Expect empty votes for a zero-value/default request

	// Test case 3: Validator not found (GetProducerVotes returns empty votes)
	nonExistentValID := uint64(999)
	reqNonExistent := &types.QueryProducerVotesByValidatorIdRequest{ValidatorId: nonExistentValID}
	resNonExistent, err := queryClient.GetProducerVotesByValidatorId(ctx, reqNonExistent)
	require.NoError(err) // GetProducerVotes is expected to return empty votes and no error if not found
	require.NotNil(resNonExistent)
	require.Empty(resNonExistent.Votes)
}

func (s *KeeperTestSuite) TestGetProducerVotes() {
	require, ctx, queryClient, stakeKeeper, keeper := s.Require(), s.ctx, s.queryClient, s.stakeKeeper, s.borKeeper

	// Setup
	val1ID := uint64(1)
	val2ID := uint64(2)
	validators := []*staketypes.Validator{
		{ValId: val1ID, Signer: "val1_signer_address"},
		{ValId: val2ID, Signer: "val2_signer_address"},
	}
	validatorSet := &staketypes.ValidatorSet{Validators: validators}

	votes1Data := []uint64{100, 101}
	votes1 := types.ProducerVotes{
		Votes: votes1Data,
	}
	votes2Data := []uint64{200, 201}
	votes2 := types.ProducerVotes{
		Votes: votes2Data,
	}

	// Mocking and setup for the happy path
	stakeKeeper.EXPECT().GetValidatorSet(ctx).Return(*validatorSet, nil).Times(1)
	err := keeper.SetProducerVotes(ctx, val1ID, votes1)
	require.NoError(err)
	err = keeper.SetProducerVotes(ctx, val2ID, votes2)
	require.NoError(err)

	// Test case 1: Valid request
	req := &types.QueryProducerVotesRequest{}
	res, err := queryClient.GetProducerVotes(ctx, req)

	require.NoError(err)
	require.NotNil(res)
	require.Len(res.AllVotes, 2)
	require.Equal(votes1, res.AllVotes[val1ID])
	require.Equal(votes2, res.AllVotes[val2ID])

	// Test case 2: GetValidatorSet returns error
	mockStakeErr := errors.New("mock stake keeper error returning validator set")
	stakeKeeper.EXPECT().GetValidatorSet(ctx).Return(staketypes.ValidatorSet{}, mockStakeErr).Times(1)
	_, err = queryClient.GetProducerVotes(ctx, req)
	require.Error(err)
	require.Contains(err.Error(), mockStakeErr.Error())
}
