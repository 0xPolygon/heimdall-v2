package keeper

import (
	"context"
	"math"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
	stakeTypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

var _ types.QueryServer = queryServer{}

type queryServer struct {
	k *Keeper
}

// NewQueryServer creates a new querier for milestone clients.
func NewQueryServer(k *Keeper) types.QueryServer {
	return queryServer{
		k: k,
	}
}

// GetParams returns the milestones params
func (q queryServer) GetParams(ctx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	params, err := q.k.GetParams(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryParamsResponse{
		Params: params,
	}, nil
}

// GetMilestoneCount returns the milestone count
func (q queryServer) GetMilestoneCount(ctx context.Context, _ *types.QueryCountRequest) (*types.QueryCountResponse, error) {
	count, err := q.k.GetMilestoneCount(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryCountResponse{Count: count}, nil
}

// GetLatestMilestone gives the latest milestone in the database
func (q queryServer) GetLatestMilestone(ctx context.Context, _ *types.QueryLatestMilestoneRequest) (*types.QueryLatestMilestoneResponse, error) {
	milestone, err := q.k.GetLastMilestone(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryLatestMilestoneResponse{Milestone: *milestone}, nil
}

// GetMilestoneByNumber return the milestone by number
func (q queryServer) GetMilestoneByNumber(ctx context.Context, req *types.QueryMilestoneRequest) (*types.QueryMilestoneResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	milestone, err := q.k.GetMilestoneByNumber(ctx, req.Number)
	if err != nil {
		return nil, err
	}

	return &types.QueryMilestoneResponse{Milestone: *milestone}, nil
}

// GetLatestNoAckMilestone fetches the latest no-ack milestone
func (q queryServer) GetLatestNoAckMilestone(ctx context.Context, _ *types.QueryLatestNoAckMilestoneRequest) (*types.QueryLatestNoAckMilestoneResponse, error) {
	res, err := q.k.GetLastNoAckMilestone(ctx)

	return &types.QueryLatestNoAckMilestoneResponse{Result: res}, err
}

// GetNoAckMilestoneById gives the result by id
func (q queryServer) GetNoAckMilestoneById(ctx context.Context, req *types.QueryNoAckMilestoneByIDRequest) (*types.QueryNoAckMilestoneByIDResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	res, err := q.k.HasNoAckMilestone(ctx, req.Id)

	return &types.QueryNoAckMilestoneByIDResponse{Result: res}, err
}

// GetMilestoneProposerByTimes queries for the milestone proposer given the number of subsequent milestone's proposers
func (q queryServer) GetMilestoneProposerByTimes(ctx context.Context, req *types.QueryMilestoneProposerRequest) (*types.QueryMilestoneProposerResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if req.Times >= math.MaxInt64 {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	// get milestone validator set
	validatorSet, err := q.k.stakeKeeper.GetMilestoneValidatorSet(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	times := int(req.Times)
	if times > len(validatorSet.Validators) {
		times = len(validatorSet.Validators)
	}

	// init proposers
	proposers := make([]stakeTypes.Validator, times)

	// get proposers
	for index := 0; index < times; index++ {
		proposers[index] = *(validatorSet.GetProposer())
		validatorSet.IncrementProposerPriority(1)
	}

	return &types.QueryMilestoneProposerResponse{Proposers: proposers}, nil
}
