package keeper

import (
	"context"
	"strconv"

	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/x/bor/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Querier struct {
	keeper Keeper
}

var _ types.QueryServer = Querier{}

func NewQuerier(keeper Keeper) Querier {
	return Querier{keeper: keeper}
}

func (k Querier) LatestSpan(ctx context.Context, req *types.QueryLatestSpanRequest) (*types.QueryLatestSpanResponse, error) {

	var emptySpan *types.Span

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	spans := k.keeper.GetAllSpans(ctx)
	if len(spans) == 0 {
		return &types.QueryLatestSpanResponse{Height: strconv.FormatInt(sdkCtx.BlockHeight(), 10), Span: emptySpan}, nil
	}

	latestSpan := spans[len(spans)-1]
	return &types.QueryLatestSpanResponse{Height: strconv.FormatInt(sdkCtx.BlockHeight(), 10), Span: latestSpan}, nil
}

func (k Querier) NextSpan(ctx context.Context, req *types.QueryNextSpanRequest) (*types.QueryNextSpanResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// fetch params
	params, err := k.keeper.GetParams(ctx)
	if err != nil {
		return nil, err
	}

	// fetch current validator set
	validatorSet := k.keeper.sk.GetValidatorSet(ctx)

	// fetch next selected block producers
	nextSpanSeed, err := k.keeper.GetNextSpanSeed(ctx)
	if err != nil {
		return nil, err
	}

	selectedProducers, err := k.keeper.SelectNextProducers(ctx, nextSpanSeed)
	if err != nil {
		return nil, err
	}

	selectedProducers = helper.SortValidatorByAddress(selectedProducers)

	// create next span
	nextSpan := &types.Span{
		Id:                req.SpanId,
		StartBlock:        req.StartBlock,
		EndBlock:          req.StartBlock + params.SpanDuration - 1,
		ValidatorSet:      validatorSet,
		SelectedProducers: selectedProducers,
		ChainId:           req.BorChainId,
	}

	return &types.QueryNextSpanResponse{Height: strconv.FormatInt(sdkCtx.BlockHeight(), 10), Span: nextSpan}, nil
}

func (k Querier) NextSpanSeed(ctx context.Context, req *types.QueryNextSpanSeedRequest) (*types.QueryNextSpanSeedResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// fetch next span seed
	nextSpanSeed, err := k.keeper.GetNextSpanSeed(ctx)
	if err != nil {
		return nil, err
	}

	return &types.QueryNextSpanSeedResponse{Height: strconv.FormatInt(sdkCtx.BlockHeight(), 10), Seed: nextSpanSeed.String()}, nil
}

func (k Querier) Params(ctx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	params, err := k.keeper.GetParams(ctx)
	if err != nil {
		return nil, err
	}

	return &types.QueryParamsResponse{Height: strconv.FormatInt(sdkCtx.BlockHeight(), 10), Params: &params}, nil
}

func (k Querier) SpanById(ctx context.Context, req *types.QuerySpanByIdRequest) (*types.QuerySpanByIdResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	spanId, err := strconv.Atoi(req.SpanId)
	if err != nil {
		return nil, err
	}

	span, err := k.keeper.GetSpan(ctx, uint64(spanId))
	if err != nil {
		return nil, err
	}

	return &types.QuerySpanByIdResponse{Height: strconv.FormatInt(sdkCtx.BlockHeight(), 10), Span: span}, nil
}

func (k Querier) SpanList(ctx context.Context, req *types.QuerySpanListRequest) (*types.QuerySpanListResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	spansList, err := k.keeper.GetSpanList(ctx, req.Page, req.Limit)
	if err != nil {
		return nil, err
	}

	return &types.QuerySpanListResponse{Height: strconv.FormatInt(sdkCtx.BlockHeight(), 10), SpanList: spansList}, nil
}
