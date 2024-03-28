package keeper

import (
	"context"

	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Querier is used as Keeper will have duplicate methods if used directly, and gRPC names take precedence over keeper
type Querier struct {
	*Keeper
}

var _ types.QueryServer = Querier{}

func NewQuerier(keeper *Keeper) Querier {
	return Querier{Keeper: keeper}
}

// Params gives the params
func (k Querier) Params(ctx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	// get validator set
	params, err := k.GetParams(ctx)
	if err != nil {
		return nil, err
	}

	return &types.QueryParamsResponse{
		Params: params,
	}, nil
}

// Count gives the milestone count
func (k Querier) Count(ctx context.Context, req *types.QueryCountRequest) (*types.QueryCountResponse, error) {
	count := k.GetMilestoneCount(ctx)

	return &types.QueryCountResponse{Count: count}, nil
}

// LatestMilestone gives the latest milestone in the database
func (k Querier) LatestMilestone(ctx context.Context, req *types.QueryLatestMilestoneRequest) (*types.QueryLatestMilestoneResponse, error) {
	milestone, err := k.GetLastMilestone(ctx)
	if err != nil {
		return nil, err
	}

	return &types.QueryLatestMilestoneResponse{Milestone: *milestone}, nil
}

// Milestone return the milestone by number
func (k Querier) Milestone(ctx context.Context, req *types.QueryMilestoneRequest) (*types.QueryMilestoneResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	milestone, err := k.GetMilestoneByNumber(ctx, req.Number)
	if err != nil {
		return nil, err
	}

	return &types.QueryMilestoneResponse{Milestone: *milestone}, nil
}

// NoAckMilestoneByID gives the result by ID number
func (k Querier) LatestNoAckMilestone(ctx context.Context, req *types.QueryLatestNoAckMilestoneRequest) (*types.QueryLatestNoAckMilestoneResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	res := k.GetLastNoAckMilestone(ctx)

	return &types.QueryLatestNoAckMilestoneResponse{Result: res}, nil
}

// NoAckMilestoneByID gives the result by ID number
func (k Querier) NoAckMilestoneByID(ctx context.Context, req *types.QueryNoAckMilestoneByIDRequest) (*types.QueryNoAckMilestoneByIDResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	res := k.GetNoAckMilestone(ctx, req.Id)

	return &types.QueryNoAckMilestoneByIDResponse{Result: res}, nil
}
