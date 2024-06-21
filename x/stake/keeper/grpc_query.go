package keeper

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/0xPolygon/heimdall-v2/x/stake/types"
)

var _ types.QueryServer = queryServer{}

type queryServer struct {
	k *Keeper
}

// NewQueryServer creates a new querier for stake clients.
// It uses the underlying keeper and its contractCaller to interact with Ethereum chain.
func NewQueryServer(k *Keeper) types.QueryServer {
	return queryServer{
		k: k,
	}
}

// CurrentValidatorSet queries all validators which are currently active in validator set
func (q queryServer) CurrentValidatorSet(ctx context.Context, _ *types.QueryCurrentValidatorSetRequest) (*types.QueryCurrentValidatorSetResponse, error) {
	validatorSet, err := q.k.GetValidatorSet(ctx)

	return &types.QueryCurrentValidatorSetResponse{
		ValidatorSet: validatorSet,
	}, err
}

// Signer queries validator info for given validator address.
func (q queryServer) Signer(ctx context.Context, req *types.QuerySignerRequest) (*types.QuerySignerResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	validator, err := q.k.GetValidatorInfo(ctx, req.ValAddress)

	if err != nil {
		return nil, status.Errorf(codes.NotFound, "error in getting validator corresponding to the given address %s", req.ValAddress)
	}

	return &types.QuerySignerResponse{Validator: validator}, nil
}

// Validator queries validator info for a given validator id.
func (q queryServer) Validator(ctx context.Context, req *types.QueryValidatorRequest) (*types.QueryValidatorResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	validator, err := q.k.GetValidatorFromValID(ctx, req.Id)

	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("error in getting validator corresponding to the given id %d", req.Id))
	}

	return &types.QueryValidatorResponse{Validator: validator}, nil
}

// ValidatorStatus queries validator status for given validator address.
func (q queryServer) ValidatorStatus(ctx context.Context, req *types.QueryValidatorStatusRequest) (*types.QueryValidatorStatusResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	return &types.QueryValidatorStatusResponse{IsOld: q.k.IsCurrentValidatorByAddress(ctx, req.ValAddress)}, nil
}

// TotalPower queries the total power of a validator set
func (q queryServer) TotalPower(ctx context.Context, _ *types.QueryTotalPowerRequest) (*types.QueryTotalPowerResponse, error) {
	totalPower, err := q.k.GetTotalPower(ctx)
	if err != nil {
		return nil, err
	}

	return &types.QueryTotalPowerResponse{TotalPower: totalPower}, nil
}

// StakingIsOldTx queries for the staking sequence
func (q queryServer) StakingIsOldTx(ctx context.Context, req *types.QueryStakingIsOldTxRequest) (*types.QueryStakingIsOldTxResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	chainParams, err := q.k.cmKeeper.GetParams(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "chain params not found")
	}

	// get main tx receipt
	receipt, err := q.k.contractCaller.GetConfirmedTxReceipt(common.HexToHash(req.TxHash), chainParams.MainChainTxConfirmations)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if receipt == nil {
		return nil, status.Errorf(codes.NotFound, "receipt not found")
	}

	sequence := new(big.Int).Mul(receipt.BlockNumber, big.NewInt(types.DefaultLogIndexUnit))
	sequence.Add(sequence, new(big.Int).SetUint64(req.LogIndex))

	// check if incoming tx already exists
	if !q.k.HasStakingSequence(ctx, sequence.String()) {
		return &types.QueryStakingIsOldTxResponse{IsOld: false}, nil
	}

	return &types.QueryStakingIsOldTxResponse{IsOld: true}, nil
}
