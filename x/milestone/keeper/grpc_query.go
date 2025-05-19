package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
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

// GetMilestoneParams returns the milestones params
func (q queryServer) GetMilestoneParams(ctx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
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

	count, err := q.k.GetMilestoneCount(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get milestone count")
	}
	if req.Number == 0 || req.Number > count {
		return nil, status.Error(codes.NotFound, "milestone number out of range")
	}

	milestone, err := q.k.GetMilestoneByNumber(ctx, req.Number)
	if err != nil {
		return nil, err
	}

	return &types.QueryMilestoneResponse{Milestone: *milestone}, nil
}
