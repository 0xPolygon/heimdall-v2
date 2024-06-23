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

// Querier is used as Keeper will have duplicate methods if used directly, and gRPC names take precedence over keeper
type Querier struct {
	*Keeper
}

var _ types.QueryServer = Querier{}

// Params gives the params
func (q Querier) Params(ctx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	// get validator set
	params, err := q.GetParams(ctx)
	if err != nil {
		return nil, err
	}

	return &types.QueryParamsResponse{
		Params: params,
	}, nil
}

// AckCount gives the checkpoint count
func (q Querier) AckCount(ctx context.Context, req *types.QueryAckCountRequest) (*types.QueryAckCountResponse, error) {
	count := q.GetACKCount(ctx)

	return &types.QueryAckCountResponse{Count: count}, nil
}

// Checkpoint gives the checkpoint based on number
func (q Querier) Checkpoint(ctx context.Context, req *types.QueryCheckpointRequest) (*types.QueryCheckpointResponse, error) {
	checkpoint, err := q.GetCheckpointByNumber(ctx, req.Number)
	if err != nil {
		return nil, err
	}

	return &types.QueryCheckpointResponse{Checkpoint: checkpoint}, nil
}

// CheckpointLatest gives the latest checkpoint
func (q Querier) CheckpointLatest(ctx context.Context, req *types.QueryCheckpointLatestRequest) (*types.QueryCheckpointLatestResponse, error) {
	checkpoint, err := q.GetLastCheckpoint(ctx)
	if err != nil {
		return nil, err
	}

	return &types.QueryCheckpointLatestResponse{Checkpoint: checkpoint}, nil
}

// CheckpointBuffer gives checkpoint from buffer
func (q Querier) CheckpointBuffer(ctx context.Context, req *types.QueryCheckpointBufferRequest) (*types.QueryCheckpointBufferResponse, error) {
	checkpoint, err := q.GetCheckpointFromBuffer(ctx)
	if err != nil {
		return nil, err
	}

	return &types.QueryCheckpointBufferResponse{Checkpoint: *checkpoint}, nil
}

// LastNoAck gives the last no ack
func (q Querier) LastNoAck(ctx context.Context, req *types.QueryLastNoAckRequest) (*types.QueryLastNoAckResponse, error) {
	noAck := q.GetLastNoAck(ctx)

	return &types.QueryLastNoAckResponse{Result: noAck}, nil
}

// NextCheckpoint gives next expected checkpoint
func (q Querier) NextCheckpoint(ctx context.Context, req *types.QueryNextCheckpointRequest) (*types.QueryNextCheckpointResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	// get validator set
	validatorSet := q.sk.GetValidatorSet(ctx)
	proposer := validatorSet.GetProposer()
	ackCount := q.GetACKCount(ctx)
	params, err := q.GetParams(ctx)
	if err != nil {
		return nil, err
	}

	var start uint64

	if ackCount != 0 {
		checkpointNumber := ackCount

		lastCheckpoint, err := q.GetCheckpointByNumber(ctx, checkpointNumber)
		if err != nil {
			return nil, err
		}

		start = lastCheckpoint.EndBlock + 1
	}

	endBlockNumber := start + params.AvgCheckpointLength

	contractCaller := q.IContractCaller

	rootHash, err := contractCaller.GetRootHash(start, end, params.MaxCheckpointLength)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("could not fetch roothash for start:%v end:%v error:%v", start, end, err.Error()))
	}

	accs := q.moduleCommunicator.GetAllDividendAccounts(ctx)

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
func (q Querier) CurrentProposer(ctx context.Context, req *types.QueryCurrentProposerRequest) (*types.QueryCurrentProposerResponse, error) {
	proposer := q.sk.GetCurrentProposer(ctx)

	return &types.QueryCurrentProposerResponse{Validator: *proposer}, nil
}

func (q Querier) Proposer(ctx context.Context, req *types.QueryProposerRequest) (*types.QueryProposerResponse, error) {
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

	return &types.QueryProposerResponse{Proposers: proposers}, nil
}
