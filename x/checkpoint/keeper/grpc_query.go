package keeper

import (
	"context"

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

// Params gives the params
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

// AckCount gives the checkpoint ack count
func (q queryServer) GetAckCount(ctx context.Context, _ *types.QueryAckCountRequest) (*types.QueryAckCountResponse, error) {
	count, err := q.k.GetAckCount(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &types.QueryAckCountResponse{AckCount: count}, err
}

// Checkpoint gives the checkpoint based on its number
func (q queryServer) GetCheckpoint(ctx context.Context, req *types.QueryCheckpointRequest) (*types.QueryCheckpointResponse, error) {
	checkpoint, err := q.k.GetCheckpointByNumber(ctx, req.Number)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &types.QueryCheckpointResponse{Checkpoint: checkpoint}, nil
}

// CheckpointLatest gives the latest checkpoint
func (q queryServer) GetCheckpointLatest(ctx context.Context, _ *types.QueryCheckpointLatestRequest) (*types.QueryCheckpointLatestResponse, error) {
	checkpoint, err := q.k.GetLastCheckpoint(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &types.QueryCheckpointLatestResponse{Checkpoint: checkpoint}, nil
}

// CheckpointBuffer gives checkpoint from buffer
func (q queryServer) GetCheckpointBuffer(ctx context.Context, req *types.QueryCheckpointBufferRequest) (*types.QueryCheckpointBufferResponse, error) {
	checkpoint, err := q.k.GetCheckpointFromBuffer(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &types.QueryCheckpointBufferResponse{Checkpoint: checkpoint}, nil
}

// LastNoAck gives the last no ack
func (q queryServer) GetLastNoAck(ctx context.Context, _ *types.QueryLastNoAckRequest) (*types.QueryLastNoAckResponse, error) {
	noAck, err := q.k.GetLastNoAck(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &types.QueryLastNoAckResponse{LastNoAckID: noAck}, err
}

// NextCheckpoint gives next expected checkpoint
func (q queryServer) GetNextCheckpoint(ctx context.Context, req *types.QueryNextCheckpointRequest) (*types.QueryNextCheckpointResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	// get validator set
	validatorSet, err := q.k.sk.GetValidatorSet(ctx)
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

	accRootHash, err := types.GetAccountRootHash(dividendAccounts)
	if err != nil {
		q.k.Logger(ctx).Error("could not get generate account root hash", "error", err)
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	checkpointMsg := types.MsgCheckpoint{
		Proposer:        proposer.Signer,
		StartBlock:      start,
		EndBlock:        endBlockNumber,
		RootHash:        hmTypes.HeimdallHash{Hash: rootHash},
		AccountRootHash: hmTypes.HeimdallHash{Hash: accRootHash},
		BorChainID:      req.BorChainID,
	}

	return &types.QueryNextCheckpointResponse{Checkpoint: checkpointMsg}, nil
}

// CurrentProposer queries validator info for the current proposer
func (q queryServer) GetCurrentProposer(ctx context.Context, _ *types.QueryCurrentProposerRequest) (*types.QueryCurrentProposerResponse, error) {
	proposer := q.k.sk.GetCurrentProposer(ctx)

	return &types.QueryCurrentProposerResponse{Validator: *proposer}, nil
}

func (q queryServer) GetProposer(ctx context.Context, req *types.QueryProposerRequest) (*types.QueryProposerResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	// get validator set
	validatorSet, err := q.k.sk.GetValidatorSet(ctx)
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
