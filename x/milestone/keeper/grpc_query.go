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

// Params fetches the parameters of the milestone module
func (q queryServer) Params(ctx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	// get validator set
	params, err := q.k.GetParams(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &types.QueryParamsResponse{
		Params: params,
	}, nil
}

// Count returns the milestone count
func (q queryServer) Count(ctx context.Context, _ *types.QueryCountRequest) (*types.QueryCountResponse, error) {
	count, err := q.k.GetMilestoneCount(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &types.QueryCountResponse{Count: count}, nil
}

// LatestMilestone gives the latest milestone in the database
func (q queryServer) LatestMilestone(ctx context.Context, _ *types.QueryLatestMilestoneRequest) (*types.QueryLatestMilestoneResponse, error) {
	milestone, err := q.k.GetLastMilestone(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &types.QueryLatestMilestoneResponse{Milestone: *milestone}, nil
}

// Milestone return the milestone by number
func (q queryServer) Milestone(ctx context.Context, req *types.QueryMilestoneRequest) (*types.QueryMilestoneResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	milestone, err := q.k.GetMilestoneByNumber(ctx, req.Number)
	if err != nil {
		return nil, err
	}

	return &types.QueryMilestoneResponse{Milestone: *milestone}, nil
}

// LatestNoAckMilestone fetches the latest no-ack milestone
func (q queryServer) LatestNoAckMilestone(ctx context.Context, req *types.QueryLatestNoAckMilestoneRequest) (*types.QueryLatestNoAckMilestoneResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	res, err := q.k.GetLastNoAckMilestone(ctx)

	return &types.QueryLatestNoAckMilestoneResponse{Result: res}, err
}

// NoAckMilestoneByID gives the result by ID number
func (q queryServer) NoAckMilestoneByID(ctx context.Context, req *types.QueryNoAckMilestoneByIDRequest) (*types.QueryNoAckMilestoneByIDResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	res, err := q.k.HasNoAckMilestone(ctx, req.Id)

	return &types.QueryNoAckMilestoneByIDResponse{Result: res}, err
}

// MilestoneProposer queries for the milestone proposer
func (q queryServer) MilestoneProposer(ctx context.Context, req *types.QueryMilestoneProposerRequest) (*types.QueryMilestoneProposerResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if req.Times >= math.MaxInt64 {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	// get milestone validator set
	validatorSet, err := q.k.StakeKeeper.GetMilestoneValidatorSet(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
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
