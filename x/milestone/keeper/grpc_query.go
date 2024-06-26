package keeper

import (
	"context"

	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
	stakeTypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// HV2 Following path for API's has been changed
// "/milestone/lastNoAck" -> "/milestone/last-no-ack"
// "/milestone/noAck/{id}"->"/milestone/no-ack/{id}"
// "/milestone/ID/{id}" it has been removed
// "/staking/milestoneProposer/{times}" -> "/milestone/proposer"

type QueryServer struct {
	*Keeper
}

var _ types.QueryServer = QueryServer{}

func NewQueryServer(keeper *Keeper) QueryServer {
	return QueryServer{Keeper: keeper}
}

// Params fetches the parameters of the milestone module
func (q QueryServer) Params(ctx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	// get validator set
	params, err := q.GetParams(ctx)
	if err != nil {
		return nil, err
	}

	return &types.QueryParamsResponse{
		Params: params,
	}, nil
}

// Count gives the milestone count
func (q QueryServer) Count(ctx context.Context, req *types.QueryCountRequest) (*types.QueryCountResponse, error) {
	count := q.GetMilestoneCount(ctx)

	return &types.QueryCountResponse{Count: count}, nil
}

// LatestMilestone gives the latest milestone in the database
func (q QueryServer) LatestMilestone(ctx context.Context, req *types.QueryLatestMilestoneRequest) (*types.QueryLatestMilestoneResponse, error) {
	milestone, err := q.GetLastMilestone(ctx)
	if err != nil {
		return nil, err
	}

	return &types.QueryLatestMilestoneResponse{Milestone: *milestone}, nil
}

// Milestone return the milestone by number
func (q QueryServer) Milestone(ctx context.Context, req *types.QueryMilestoneRequest) (*types.QueryMilestoneResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	milestone, err := q.GetMilestoneByNumber(ctx, req.Number)
	if err != nil {
		return nil, err
	}

	return &types.QueryMilestoneResponse{Milestone: *milestone}, nil
}

// LatestNoAckMilestone fetches the latest no-ack milestone
func (q QueryServer) LatestNoAckMilestone(ctx context.Context, req *types.QueryLatestNoAckMilestoneRequest) (*types.QueryLatestNoAckMilestoneResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	res := q.GetLastNoAckMilestone(ctx)

	return &types.QueryLatestNoAckMilestoneResponse{Result: res}, nil
}

// NoAckMilestoneByID gives the result by ID number
func (q QueryServer) NoAckMilestoneByID(ctx context.Context, req *types.QueryNoAckMilestoneByIDRequest) (*types.QueryNoAckMilestoneByIDResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	res := q.GetNoAckMilestone(ctx, req.Id)

	return &types.QueryNoAckMilestoneByIDResponse{Result: res}, nil
}

// MilestoneProposer queries for the milestone proposer
func (q QueryServer) MilestoneProposer(ctx context.Context, req *types.QueryMilestoneProposerRequest) (*types.QueryMilestoneProposerResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	// get validator set
	validatorSet := q.sk.GetValidatorSet(ctx)

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
