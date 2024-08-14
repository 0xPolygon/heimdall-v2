package keeper

import (
	"context"
	"strconv"

	"github.com/0xPolygon/heimdall-v2/x/bor/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type QueryServer struct {
	Keeper
}

var _ types.QueryServer = QueryServer{}

func NewQueryServer(keeper Keeper) QueryServer {
	return QueryServer{Keeper: keeper}
}

func (q QueryServer) GetLatestSpan(ctx context.Context, _ *types.QueryLatestSpanRequest) (*types.QueryLatestSpanResponse, error) {

	spans, err := q.GetAllSpans(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	if len(spans) == 0 {
		return nil, nil
	}

	latestSpan := spans[len(spans)-1]
	return &types.QueryLatestSpanResponse{Span: latestSpan}, nil
}

func (q QueryServer) GetNextSpan(ctx context.Context, req *types.QueryNextSpanRequest) (*types.QueryNextSpanResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	lastSpan, err := q.GetLastSpan(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if req.SpanId != lastSpan.Id+1 {
		return nil, status.Errorf(codes.InvalidArgument, "invalid span id")
	}

	if req.StartBlock != lastSpan.EndBlock+1 {
		return nil, status.Errorf(codes.InvalidArgument, "invalid start block")
	}

	if req.BorChainId != lastSpan.ChainId {
		return nil, status.Errorf(codes.InvalidArgument, "invalid chain id")
	}

	// fetch params
	params, err := q.FetchParams(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	// fetch current validator set
	validatorSet := q.sk.GetValidatorSet(ctx)

	// fetch next selected block producers
	nextSpanSeed, err := q.FetchNextSpanSeed(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	selectedProducers, err := q.SelectNextProducers(ctx, nextSpanSeed)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	// TODO HV2: uncomment when helper is merged
	// selectedProducers = helper.SortValidatorByAddress(selectedProducers)

	// create next span
	nextSpan := &types.Span{
		Id:                req.SpanId,
		StartBlock:        req.StartBlock,
		EndBlock:          req.StartBlock + params.SpanDuration - 1,
		ValidatorSet:      validatorSet,
		SelectedProducers: selectedProducers,
		ChainId:           req.BorChainId,
	}

	return &types.QueryNextSpanResponse{Span: nextSpan}, nil
}

func (q QueryServer) GetNextSpanSeed(ctx context.Context, _ *types.QueryNextSpanSeedRequest) (*types.QueryNextSpanSeedResponse, error) {

	// fetch next span seed
	nextSpanSeed, err := q.FetchNextSpanSeed(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &types.QueryNextSpanSeedResponse{Seed: nextSpanSeed.String()}, nil
}

func (q QueryServer) GetParams(ctx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {

	params, err := q.FetchParams(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &types.QueryParamsResponse{Params: &params}, nil
}

func (q QueryServer) GetSpanById(ctx context.Context, req *types.QuerySpanByIdRequest) (*types.QuerySpanByIdResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	spanId, err := strconv.Atoi(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	span, err := q.GetSpan(ctx, uint64(spanId))
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &types.QuerySpanByIdResponse{Span: span}, nil
}

func (q QueryServer) GetSpanList(ctx context.Context, req *types.QuerySpanListRequest) (*types.QuerySpanListResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	// spansList, err := q.FetchSpanList(ctx, req.Page, req.Limit)
	// if err != nil {
	// 	return nil, status.Errorf(codes.Internal, err.Error())
	// }

	spans, pageRes, err := query.CollectionPaginate(
		ctx,
		q.spans,
		req.Pagination,
		func(_ uint64, _ *types.Span) ([]*types.Span, error) {
			spans, err := q.GetAllSpans(ctx)
			if err != nil {
				return nil, err
			}
			return spans, nil
		},
	)

	// TODO @Raneet: This is a workaround to flatten the 2D slice of spans and solve the compile error in
	//  the return statement. However, the error was there even before my fix to `query.CollectionPaginate`,
	//  which is basically declaring spans as collections.Map[uint64, *types.Span] (note the pointer).
	//  Hence, you might want to remove this (untested) and implement your own solution

	// Create a one-dimensional slice to hold the result
	var flatSpans []types.Span

	// Flatten the 2D slice and dereference the pointers
	for _, innerSlice := range spans {
		for _, spanPtr := range innerSlice {
			if spanPtr != nil {
				flatSpans = append(flatSpans, *spanPtr)
			}
		}
	}

	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "paginate: %v", err)
	}
	return &types.QuerySpanListResponse{SpanList: flatSpans, Pagination: pageRes}, nil
}
