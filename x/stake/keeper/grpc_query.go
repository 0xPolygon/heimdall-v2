package keeper

import (
	"context"
	"fmt"
	"math/big"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/0xPolygon/heimdall-v2/x/stake/types"

	hmTypes "github.com/0xPolygon/heimdall-v2/types"
)

// Querier is used as Keeper will have duplicate methods if used directly, and gRPC names take precedence over keeper
type Querier struct {
	*Keeper
}

var _ types.QueryServer = Querier{}

// CurrentValidatorSet queries all validators which are currently active in validator set
func (q Querier) CurrentValidatorSet(ctx context.Context, _ *types.QueryCurrentValidatorSetRequest) (*types.QueryCurrentValidatorSetResponse, error) {
	validatorSet, err := q.GetValidatorSet(ctx)

	return &types.QueryCurrentValidatorSetResponse{
		ValidatorSet: validatorSet,
	}, err
}

// Signer queries validator info for given validator validator address.
func (q Querier) Signer(ctx context.Context, req *types.QuerySignerRequest) (*types.QuerySignerResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	validator, err := q.GetValidatorInfo(ctx, req.ValAddress)

	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("error in getting validator corresposing to the given address", req.ValAddress, "err", err))
	}

	return &types.QuerySignerResponse{Validator: validator}, nil
}

// Validator queries validator info for a given validator id.
func (q Querier) Validator(ctx context.Context, req *types.QueryValidatorRequest) (*types.QueryValidatorResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	validator, ok := q.GetValidatorFromValID(ctx, req.Id)

	if !ok {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("error in getting validator corresposing to the given id", req.Id))
	}

	return &types.QueryValidatorResponse{Validator: validator}, nil
}

// ValidatorStatus queries validator status for given validator val_address.
func (q Querier) ValidatorStatus(ctx context.Context, req *types.QueryValidatorStatusRequest) (*types.QueryValidatorStatusResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	// get validator status by signer address
	status := q.IsCurrentValidatorByAddress(ctx, req.ValAddress)

	return &types.QueryValidatorStatusResponse{Status: status}, nil
}

// TotalPower queries the total power of a validator set
func (q Querier) TotalPower(ctx context.Context, _ *types.QueryTotalPowerRequest) (*types.QueryTotalPowerResponse, error) {
	totalPower := q.GetTotalPower(ctx)

	return &types.QueryTotalPowerResponse{TotalPower: totalPower}, nil
}

// StakingSequence queries for the staking sequence
func (q Querier) StakingSequence(ctx context.Context, req *types.QueryStakingSequenceRequest) (*types.QueryStakingSequenceResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	chainParams, err := q.cmKeeper.GetParams(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "chain params not found")
	}

	// get main tx receipt
	receipt, err := q.IContractCaller.GetConfirmedTxReceipt(hmTypes.HexToHeimdallHash(req.TxHash).EthHash(), chainParams.MainChainTxConfirmations)
	if err != nil || receipt == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	sequence := new(big.Int).Mul(receipt.BlockNumber, big.NewInt(hmTypes.DefaultLogIndexUnit))
	sequence.Add(sequence, new(big.Int).SetUint64(req.LogIndex))

	// check if incoming tx already exists
	if !q.HasStakingSequence(ctx, sequence.String()) {
		return &types.QueryStakingSequenceResponse{Status: true}, nil
	}

	return &types.QueryStakingSequenceResponse{Status: true}, nil
}
