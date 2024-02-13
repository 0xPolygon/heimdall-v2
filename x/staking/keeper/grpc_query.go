package keeper

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/0xPolygon/heimdall-v2/x/staking/types"
	hmTypes "github.com/0xPolygon/heimdall-v2/x/types"
)

// Querier is used as Keeper will have duplicate methods if used directly, and gRPC names take precedence over keeper
type Querier struct {
	*Keeper
}

var _ types.QueryServer = Querier{}

func NewQuerier(keeper *Keeper) Querier {
	return Querier{Keeper: keeper}
}

// Validators queries all validators that match the given status
func (k Querier) CurrentValidatorSet(ctx context.Context, req *types.QueryCurrentValidatorSetRequest) (*types.QueryCurrentValidatorSetResponse, error) {
	// get validator set
	validatorSet := k.GetValidatorSet(ctx)

	return &types.QueryCurrentValidatorSetResponse{
		ValidatorSet: validatorSet,
	}, nil
}

// Signer queries validator info for given validator val_address.
func (k Querier) Signer(ctx context.Context, req *types.QuerySignerRequest) (*types.QuerySignerResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	validator, err := k.GetValidatorInfo(ctx, req.ValAddress)

	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("Error in getting validator corresposing to the given address Err:%s", err))
	}

	return &types.QuerySignerResponse{Validator: validator}, nil
}

// Validator queries validator info for given validator id.
func (k Querier) Validator(ctx context.Context, req *types.QueryValidatorRequest) (*types.QueryValidatorResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	fmt.Print("-------IN VALIDATOR", req.Id)
	validator, ok := k.GetValidatorFromValID(ctx, req.Id)
	fmt.Print("-------OUT VALIDATOR", ok)

	if !ok {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("Error in getting validator corresposing to the given id "))
	}

	return &types.QueryValidatorResponse{Validator: validator}, nil
}

// ValidatorStatus queries validator status for given validator val_address.
func (k Querier) ValidatorStatus(ctx context.Context, req *types.QueryValidatorStatusRequest) (*types.QueryValidatorStatusResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	// get validator status by signer address
	status := k.IsCurrentValidatorByAddress(ctx, req.ValAddress)

	return &types.QueryValidatorStatusResponse{Status: status}, nil
}

// TotalPower queries total power of a validator set
func (k Querier) TotalPower(ctx context.Context, req *types.QueryTotalPowerRequest) (*types.QueryTotalPowerResponse, error) {
	totalPower := k.GetTotalPower(ctx)

	return &types.QueryTotalPowerResponse{TotalPower: totalPower}, nil
}

// CurrentProposer queries validator info for the current proposer
func (k Querier) CurrentProposer(ctx context.Context, req *types.QueryCurrentProposerRequest) (*types.QueryCurrentProposerResponse, error) {
	proposer := k.GetCurrentProposer(ctx)

	return &types.QueryCurrentProposerResponse{Validator: *proposer}, nil
}

// Proposer queries for the proposer
func (k Querier) Proposer(ctx context.Context, req *types.QueryProposerRequest) (*types.QueryProposerResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	// get validator set
	validatorSet := k.GetValidatorSet(ctx)

	times := int(req.Times)
	if times > len(validatorSet.Validators) {
		times = len(validatorSet.Validators)
	}

	// init proposers
	proposers := make([]hmTypes.Validator, times)

	// get proposers
	for index := 0; index < times; index++ {
		proposers[index] = *(validatorSet.GetProposer())
		validatorSet.IncrementProposerPriority(1)
	}

	return &types.QueryProposerResponse{Proposers: proposers}, nil
}

// MilestoneProposer queries for the milestone proposer
func (k Querier) MilestoneProposer(ctx context.Context, req *types.QueryMilestoneProposerRequest) (*types.QueryMilestoneProposerResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	// get validator set
	validatorSet := k.GetValidatorSet(ctx)

	times := int(req.Times)
	if times > len(validatorSet.Validators) {
		times = len(validatorSet.Validators)
	}

	// init proposers
	proposers := make([]hmTypes.Validator, times)

	// get proposers
	for index := 0; index < times; index++ {
		proposers[index] = *(validatorSet.GetProposer())
		validatorSet.IncrementProposerPriority(1)
	}

	return &types.QueryMilestoneProposerResponse{Proposers: proposers}, nil
}

// StakingSequence queries for the staking sequence
func (k Querier) StakingSequence(ctx context.Context, req *types.QueryStakingSequenceRequest) (*types.QueryStakingSequenceResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	// //TODO H2 Please implement this
	// //chainParams := keeper.chainKeeper.GetParams(ctx

	// // get main tx receipt
	// receipt, err := k.IContractCaller.GetConfirmedTxReceipt(hmTypes.HexToHeimdallHash(req.TxHash).EthHash(),)
	// if err != nil || receipt == nil {
	// 	return nil, status.Error(codes.InvalidArgument, "empty request")
	// }

	// // sequence id

	// sequence := new(big.Int).Mul(receipt.BlockNumber, big.NewInt(hmTypes.DefaultLogIndexUnit))
	// sequence.Add(sequence, new(big.Int).SetUint64(req.LogIndex))

	// // check if incoming tx already exists
	// if !k.HasStakingSequence(ctx, sequence.String()) {
	// 	return &types.QueryStakingSequenceResponse{Status: true}, nil
	// }

	return &types.QueryStakingSequenceResponse{Status: true}, nil
}
