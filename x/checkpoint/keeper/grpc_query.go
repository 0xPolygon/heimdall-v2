package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
	stakeTypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

var _ types.QueryServer = queryServer{}

type queryServer struct {
	k *Keeper
}

// NewQueryServer creates a new querier for the checkpoint client.
// It uses the underlying keeper and its contractCaller to interact with Ethereum chain.
func NewQueryServer(k *Keeper) types.QueryServer {
	return queryServer{
		k: k,
	}
}

// GetParams returns the checkpoint params
func (q queryServer) GetParams(ctx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	// get validator set
	params, err := q.k.GetParams(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &types.QueryParamsResponse{
		Params: params,
	}, nil
}

// GetAckCount returns the checkpoint ack count
func (q queryServer) GetAckCount(ctx context.Context, _ *types.QueryAckCountRequest) (*types.QueryAckCountResponse, error) {
	count, err := q.k.GetAckCount(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &types.QueryAckCountResponse{AckCount: count}, err
}

// GetCheckpoint returns the checkpoint based on its number
func (q queryServer) GetCheckpoint(ctx context.Context, req *types.QueryCheckpointRequest) (*types.QueryCheckpointResponse, error) {
	checkpoint, err := q.k.GetCheckpointByNumber(ctx, req.Number)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &types.QueryCheckpointResponse{Checkpoint: checkpoint}, nil
}

// GetCheckpointLatest returns the latest checkpoint
func (q queryServer) GetCheckpointLatest(ctx context.Context, _ *types.QueryCheckpointLatestRequest) (*types.QueryCheckpointLatestResponse, error) {
	checkpoint, err := q.k.GetLastCheckpoint(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &types.QueryCheckpointLatestResponse{Checkpoint: checkpoint}, nil
}

// GetCheckpointBuffer returns the checkpoint from buffer
func (q queryServer) GetCheckpointBuffer(ctx context.Context, _ *types.QueryCheckpointBufferRequest) (*types.QueryCheckpointBufferResponse, error) {
	checkpoint, err := q.k.GetCheckpointFromBuffer(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &types.QueryCheckpointBufferResponse{Checkpoint: checkpoint}, nil
}

// GetLastNoAck returns the last no ack
func (q queryServer) GetLastNoAck(ctx context.Context, _ *types.QueryLastNoAckRequest) (*types.QueryLastNoAckResponse, error) {
	noAck, err := q.k.GetLastNoAck(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &types.QueryLastNoAckResponse{LastNoAckId: noAck}, err
}

// GetNextCheckpoint returns the next expected checkpoint
func (q queryServer) GetNextCheckpoint(ctx context.Context, req *types.QueryNextCheckpointRequest) (*types.QueryNextCheckpointResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	// get validator set
	validatorSet, err := q.k.stakeKeeper.GetValidatorSet(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	proposer := validatorSet.GetProposer()
	ackCount, err := q.k.GetAckCount(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	params, err := q.k.GetParams(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	var start uint64

	if ackCount != 0 {
		checkpointNumber := ackCount

		lastCheckpoint, err := q.k.GetCheckpointByNumber(ctx, checkpointNumber)
		if err != nil {
			return nil, status.Errorf(codes.Internal, err.Error())
		}

		start = lastCheckpoint.EndBlock + 1
	}

	endBlockNumber := start + params.AvgCheckpointLength

	contractCaller := q.k.IContractCaller

	rootHash, err := contractCaller.GetRootHash(start, endBlockNumber, params.MaxCheckpointLength)
	if err != nil {
		q.k.Logger(ctx).Error("could not fetch rootHash", "start", start, "end", endBlockNumber, "error", err)
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	dividendAccounts, err := q.k.topupKeeper.GetAllDividendAccounts(ctx)
	if err != nil {
		q.k.Logger(ctx).Error("could not get the dividends accounts", "error", err)
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	accRootHash, err := hmTypes.GetAccountRootHash(dividendAccounts)
	if err != nil {
		q.k.Logger(ctx).Error("could not get generate account root hash", "error", err)
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	checkpointMsg := types.MsgCheckpoint{
		Proposer:        proposer.Signer,
		StartBlock:      start,
		EndBlock:        endBlockNumber,
		RootHash:        rootHash,
		AccountRootHash: accRootHash,
		BorChainId:      req.BorChainId,
	}

	return &types.QueryNextCheckpointResponse{Checkpoint: checkpointMsg}, nil
}

// GetCurrentProposer queries validator info for the current proposer
func (q queryServer) GetCurrentProposer(ctx context.Context, _ *types.QueryCurrentProposerRequest) (*types.QueryCurrentProposerResponse, error) {
	proposer := q.k.stakeKeeper.GetCurrentProposer(ctx)

	return &types.QueryCurrentProposerResponse{Validator: *proposer}, nil
}

// GetProposers queries validator info for the current proposers
func (q queryServer) GetProposers(ctx context.Context, req *types.QueryProposerRequest) (*types.QueryProposerResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	// get validator set
	validatorSet, err := q.k.stakeKeeper.GetValidatorSet(ctx)
	if err != nil {
		q.k.Logger(ctx).Error("could not get get validators set", "error", err)
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	times := int(req.Times)
	if times > len(validatorSet.Validators) {
		times = len(validatorSet.Validators)
	}

	proposers := make([]stakeTypes.Validator, times)
	for i := 0; i < times; i++ {
		proposers[i] = *(validatorSet.GetProposer())
		validatorSet.IncrementProposerPriority(1)
	}

	return &types.QueryProposerResponse{Proposers: proposers}, nil
}

// GetCheckpointList returns the list of checkpoints
func (q queryServer) GetCheckpointList(ctx context.Context, req *types.QueryCheckpointListRequest) (*types.QueryCheckpointListResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	if req.Pagination != nil && req.Pagination.Limit > 1000 {
		return nil, status.Errorf(codes.InvalidArgument, "limit must be less than or equal to 1000")
	}

	checkpoints, pageRes, err := query.CollectionPaginate(
		ctx,
		q.k.checkpoints,
		req.Pagination, func(number uint64, checkpoint types.Checkpoint) (types.Checkpoint, error) {
			return q.k.GetCheckpointByNumber(ctx, number)
		},
	)

	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "paginate: %v", err)
	}

	return &types.QueryCheckpointListResponse{CheckpointList: checkpoints, Pagination: pageRes}, nil
}
