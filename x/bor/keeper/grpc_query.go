package keeper

import (
	"context"
	"strconv"

	"github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/ethereum/go-ethereum/common"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/0xPolygon/heimdall-v2/x/bor/types"
	staketypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

const maxSpanListLimitPerPage = 1000

var _ types.QueryServer = queryServer{}

type queryServer struct {
	k *Keeper
}

func isPaginationEmpty(p query.PageRequest) bool {
	return p.Key == nil &&
		p.Offset == 0 &&
		p.Limit == 0 &&
		!p.CountTotal &&
		!p.Reverse
}

func NewQueryServer(k *Keeper) types.QueryServer {
	return queryServer{
		k: k,
	}
}

func (q queryServer) GetLatestSpan(ctx context.Context, _ *types.QueryLatestSpanRequest) (*types.QueryLatestSpanResponse, error) {
	spans, err := q.k.GetAllSpans(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if len(spans) == 0 {
		return nil, status.Error(codes.NotFound, "no spans found")
	}

	latestSpan := spans[len(spans)-1]
	return &types.QueryLatestSpanResponse{Span: *latestSpan}, nil
}

func (q queryServer) GetNextSpan(ctx context.Context, req *types.QueryNextSpanRequest) (*types.QueryNextSpanResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	lastSpan, err := q.k.GetLastSpan(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if req.SpanId != lastSpan.Id+1 {
		return nil, status.Errorf(codes.InvalidArgument, "invalid span id")
	}

	if req.StartBlock != lastSpan.EndBlock+1 {
		return nil, status.Errorf(codes.InvalidArgument, "invalid start block")
	}

	if req.BorChainId != lastSpan.BorChainId {
		return nil, status.Errorf(codes.InvalidArgument, "invalid chain id")
	}

	// fetch params
	params, err := q.k.FetchParams(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// fetch current validator set
	validatorSet, err := q.k.sk.GetValidatorSet(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Convert []*Validator to []staketypes.Validator
	validators := make([]staketypes.Validator, len(validatorSet.Validators))
	for i, v := range validatorSet.Validators {
		validators[i] = *v
	}

	// fetch next selected block producers
	nextSpanSeed, _, err := q.k.FetchNextSpanSeed(ctx, req.SpanId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	selectedProducers, err := q.k.SelectNextProducers(ctx, nextSpanSeed, validators)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	selectedProducers = types.SortValidatorByAddress(selectedProducers)

	var addressCodec = address.HexCodec{}

	for idx := range selectedProducers {
		addrBytes, _ := addressCodec.StringToBytes(selectedProducers[idx].Signer)
		checksummedAddress := common.BytesToAddress(addrBytes).Hex()
		selectedProducers[idx].Signer = checksummedAddress
	}

	for idx := range validatorSet.Validators {
		addrBytes, _ := addressCodec.StringToBytes(validatorSet.Validators[idx].Signer)
		checksummedAddress := common.BytesToAddress(addrBytes).Hex()
		validatorSet.Validators[idx].Signer = checksummedAddress
	}

	// create next span
	nextSpan := &types.Span{
		Id:                req.SpanId,
		StartBlock:        req.StartBlock,
		EndBlock:          req.StartBlock + params.SpanDuration - 1,
		ValidatorSet:      validatorSet,
		SelectedProducers: selectedProducers,
		BorChainId:        req.BorChainId,
	}

	return &types.QueryNextSpanResponse{Span: *nextSpan}, nil
}

// GetNextSpanSeed returns the next span seed
func (q queryServer) GetNextSpanSeed(ctx context.Context, req *types.QueryNextSpanSeedRequest) (*types.QueryNextSpanSeedResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}
	spanId := req.GetId()

	// fetch next span seed
	nextSpanSeed, nextSpanSeedAuthor, err := q.k.FetchNextSpanSeed(ctx, spanId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryNextSpanSeedResponse{
		Seed:       nextSpanSeed.String(),
		SeedAuthor: nextSpanSeedAuthor.Hex(),
	}, nil
}

// GetBorParams returns the bor module parameters
func (q queryServer) GetBorParams(ctx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	params, err := q.k.FetchParams(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryParamsResponse{Params: params}, nil
}

// GetSpanById returns the span by id
func (q queryServer) GetSpanById(ctx context.Context, req *types.QuerySpanByIdRequest) (*types.QuerySpanByIdResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	spanId, err := strconv.Atoi(req.Id)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	span, err := q.k.GetSpan(ctx, uint64(spanId))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QuerySpanByIdResponse{Span: &span}, nil
}

// GetSpanList returns the list of spans
func (q queryServer) GetSpanList(ctx context.Context, req *types.QuerySpanListRequest) (*types.QuerySpanListResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	if isPaginationEmpty(req.Pagination) && req.Pagination.Limit > maxSpanListLimitPerPage {
		return nil, status.Errorf(codes.InvalidArgument, "limit must be less than or equal to 1000")
	}

	spans, pageRes, err := query.CollectionPaginate(
		ctx,
		q.k.spans,
		&req.Pagination, func(id uint64, span types.Span) (types.Span, error) {
			return q.k.GetSpan(ctx, id)
		},
	)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "paginate: %v", err)
	}

	return &types.QuerySpanListResponse{SpanList: spans, Pagination: *pageRes}, nil
}
