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

//var _ types.QueryServer = Querier{}

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

// AckCount gives the checkpoint count
func (k Querier) AckCount(ctx context.Context, req *types.QueryAckCountRequest) (*types.QueryAckCountResponse, error) {
	count := k.GetACKCount(ctx)

	return &types.QueryAckCountResponse{Count: count}, nil
}

// Checkpoint gives the checkpoint based on number
func (k Querier) Checkpoint(ctx context.Context, req *types.QueryCheckpointRequest) (*types.QueryCheckpointResponse, error) {
	checkpoint, err := k.GetCheckpointByNumber(ctx, req.Number)
	if err != nil {
		return nil, err
	}

	return &types.QueryCheckpointResponse{Checkpoint: checkpoint}, nil
}

// CheckpointLatest gives the latest checkpoint
func (k Querier) CheckpointLatest(ctx context.Context, req *types.QueryCheckpointLatestRequest) (*types.QueryCheckpointLatestResponse, error) {
	checkpoint, err := k.GetLastCheckpoint(ctx)
	if err != nil {
		return nil, err
	}

	return &types.QueryCheckpointLatestResponse{Checkpoint: checkpoint}, nil
}

// CheckpointBuffer gives checkpoint from buffer
func (k Querier) CheckpointBuffer(ctx context.Context, req *types.QueryCheckpointBufferRequest) (*types.QueryCheckpointBufferResponse, error) {
	checkpoint, err := k.GetCheckpointFromBuffer(ctx)
	if err != nil {
		return nil, err
	}

	return &types.QueryCheckpointBufferResponse{Checkpoint: *checkpoint}, nil
}

// LastNoAck gives the last no ack
func (k Querier) LastNoAck(ctx context.Context, req *types.QueryLastNoAckRequest) (*types.QueryLastNoAckResponse, error) {
	noAck := k.GetLastNoAck(ctx)

	return &types.QueryLastNoAckResponse{Result: noAck}, nil
}

// NextCheckpoint gives next expected checkpoint
func (k Querier) NextCheckpoint(ctx context.Context, req *types.QueryNextCheckpointRequest) (*types.QueryNextCheckpointResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	// get validator set
	validatorSet := k.sk.GetValidatorSet(ctx)
	proposer := validatorSet.GetProposer()
	ackCount := k.GetACKCount(ctx)
	params, err := k.GetParams(ctx)
	if err != nil {
		return nil, err
	}

	var start uint64

	if ackCount != 0 {
		checkpointNumber := ackCount

		lastCheckpoint, err := k.GetCheckpointByNumber(ctx, checkpointNumber)
		if err != nil {
			return nil, err
		}

		start = lastCheckpoint.EndBlock + 1
	}

	end := start + params.AvgCheckpointLength

	contractCaller := k.IContractCaller

	rootHash, err := contractCaller.GetRootHash(start, end, params.MaxCheckpointLength)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("could not fetch roothash for start:%v end:%v error:%v", start, end, err.Error()))
	}

	accs := k.moduleCommunicator.GetAllDividendAccounts(ctx)

	accRootHash, err := types.GetAccountRootHash(accs)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("could not get generate account root hash. Error:%v", err.Error()))
	}

	checkpointMsg := types.MsgCheckpoint{
		Proposer:        proposer.Signer,
		StartBlock:      start,
		EndBlock:        start + params.AvgCheckpointLength,
		RootHash:        hmTypes.BytesToHeimdallHash(rootHash),
		AccountRootHash: hmTypes.BytesToHeimdallHash(accRootHash),
		BorChainID:      req.GetBorChainId(),
	}

	return &types.QueryNextCheckpointResponse{Checkpoint: checkpointMsg}, nil
}

// CurrentProposer queries validator info for the current proposer
func (k Querier) CurrentProposer(ctx context.Context, req *types.QueryCurrentProposerRequest) (*types.QueryCurrentProposerResponse, error) {
	proposer := k.sk.GetCurrentProposer(ctx)

	return &types.QueryCurrentProposerResponse{Validator: *proposer}, nil
}

func (k Querier) Proposer(ctx context.Context, req *types.QueryProposerRequest) (*types.QueryProposerResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	// get validator set
	validatorSet := k.sk.GetValidatorSet(ctx)

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
