package keeper

import (
	"context"
	"errors"
	"fmt"

	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
	stakeTypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ types.QueryServer = queryServer{}

type queryServer struct {
	k *Keeper
}

// NewQueryServer creates a new querier for checkpoint clients.
// It uses the underlying keeper and its contractCaller to interact with Ethereum chain.
func NewQueryServer(k *Keeper) types.QueryServer {
	return queryServer{
		k: k,
	}
}

// Params gives the params
func (q queryServer) Params(ctx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	// get validator set
	params, err := q.k.GetParams(ctx)
	if err != nil {
		return nil, err
	}

	return &types.QueryParamsResponse{
		Params: params,
	}, nil
}

// AckCount gives the checkpoint count
func (q queryServer) AckCount(ctx context.Context, req *types.QueryAckCountRequest) (*types.QueryAckCountResponse, error) {
	count, err := q.k.GetACKCount(ctx)

	return &types.QueryAckCountResponse{Count: count}, err
}

// Checkpoint gives the checkpoint based on number
func (q queryServer) Checkpoint(ctx context.Context, req *types.QueryCheckpointRequest) (*types.QueryCheckpointResponse, error) {
	checkpoint, err := q.k.GetCheckpointByNumber(ctx, req.Number)
	if err != nil {
		return nil, err
	}

	return &types.QueryCheckpointResponse{Checkpoint: checkpoint}, nil
}

// CheckpointLatest gives the latest checkpoint
func (q queryServer) CheckpointLatest(ctx context.Context, req *types.QueryCheckpointLatestRequest) (*types.QueryCheckpointLatestResponse, error) {
	checkpoint, err := q.k.GetLastCheckpoint(ctx)
	if err != nil {
		return nil, err
	}

	return &types.QueryCheckpointLatestResponse{Checkpoint: checkpoint}, nil
}

// CheckpointBuffer gives checkpoint from buffer
func (q queryServer) CheckpointBuffer(ctx context.Context, req *types.QueryCheckpointBufferRequest) (*types.QueryCheckpointBufferResponse, error) {
	checkpoint, err := q.k.GetCheckpointFromBuffer(ctx)
	if err != nil {
		return nil, err
	}

	return &types.QueryCheckpointBufferResponse{Checkpoint: *checkpoint}, nil
}

// LastNoAck gives the last no ack
func (q queryServer) LastNoAck(ctx context.Context, req *types.QueryLastNoAckRequest) (*types.QueryLastNoAckResponse, error) {
	noAck, err := q.k.GetLastNoAck(ctx)

	return &types.QueryLastNoAckResponse{Result: noAck}, err
}

// NextCheckpoint gives next expected checkpoint
func (q queryServer) NextCheckpoint(ctx context.Context, req *types.QueryNextCheckpointRequest) (*types.QueryNextCheckpointResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	// get validator set
	validatorSet, err := q.k.sk.GetValidatorSet(ctx)
	if err != nil {
		return nil, err
	}

	proposer := validatorSet.GetProposer()
	ackCount, err := q.k.GetACKCount(ctx)
	if err != nil {
		return nil, err
	}

	params, err := q.k.GetParams(ctx)
	if err != nil {
		return nil, err
	}

	var start uint64

	if ackCount != 0 {
		checkpointNumber := ackCount

		lastCheckpoint, err := q.k.GetCheckpointByNumber(ctx, checkpointNumber)
		if err != nil {
			return nil, err
		}

		start = lastCheckpoint.EndBlock + 1
	}

	endBlockNumber := start + params.AvgCheckpointLength

	contractCaller := q.k.IContractCaller

	rootHash, err := contractCaller.GetRootHash(start, endBlockNumber, params.MaxCheckpointLength)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("could not fetch roothash for start:%v end:%v error:%v", start, end, err.Error()))
	}

	accs := q.k.topupKeeper.GetAllDividendAccounts(ctx)

	accRootHash, err := types.GetAccountRootHash(accs)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("could not get generate account root hash. Error:%v", err.Error()))
	}

	checkpointMsg := types.MsgCheckpoint{
		Proposer:        proposer.Signer,
		StartBlock:      start,
		EndBlock:        endBlockNumber,
		RootHash:        hmTypes.BytesToHeimdallHash(rootHash),
		AccountRootHash: hmTypes.BytesToHeimdallHash(accRootHash),
		BorChainID:      req.GetBorChainId(),
	}

	return &types.QueryNextCheckpointResponse{Checkpoint: checkpointMsg}, nil
}

// CurrentProposer queries validator info for the current proposer
func (q queryServer) CurrentProposer(ctx context.Context, req *types.QueryCurrentProposerRequest) (*types.QueryCurrentProposerResponse, error) {
	proposer := q.k.sk.GetCurrentProposer(ctx)

	return &types.QueryCurrentProposerResponse{Validator: *proposer}, nil
}

func (q queryServer) Proposer(ctx context.Context, req *types.QueryProposerRequest) (*types.QueryProposerResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	// get validator set
	validatorSet, err := q.k.sk.GetValidatorSet(ctx)
	if err != nil {
		return nil, err
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

	return &types.QueryProposerResponse{Proposers: proposers}, nil
}
