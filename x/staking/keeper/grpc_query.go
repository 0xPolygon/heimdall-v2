package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/0xPolygon/heimdall-v2/x/staking/types"
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
	panic("IM")

}

// Signer queries validator info for given validator val_address.
func (k Querier) Signer(ctx context.Context, req *types.QuerySignerRequest) (*types.QuerySignerResponse, error) {
	panic("IM")

}

// Validator queries validator info for given validator id.
func (k Querier) Validator(ctx context.Context, req *types.QueryValidatorRequest) (*types.QueryValidatorResponse, error) {
	panic("IM")

}

// ValidatorStatus queries validator status for given validator val_address.
func (k Querier) ValidatorStatus(ctx context.Context, req *types.QueryValidatorStatusRequest) (*types.QueryValidatorStatusResponse, error) {
	panic("IM")

}

// TotalPower queries total power of a validator set
func (k Querier) TotalPower(ctx context.Context, req *types.QueryTotalPowerRequest) (*types.QueryTotalPowerResponse, error) {
	panic("IM")

}

// CurrentProposer queries validator info for the current proposer
func (k Querier) CurrentProposer(ctx context.Context, req *types.QueryCurrentProposerRequest) (*types.QueryCurrentProposerResponse, error) {
	panic("IM")

}

// Proposer queries for the proposer
func (k Querier) Proposer(ctx context.Context, req *types.QueryProposerRequest) (*types.QueryProposerResponse, error) {
	panic("IM")
}

// MilestoneProposer queries for the milestone proposer
func (k Querier) MilestoneProposer(ctx context.Context, req *types.QueryMilestoneProposerRequest) (*types.QueryMilestoneProposerResponse, error) {
	panic("IM")
}

// StakingSequence queries for the staking sequence
func (k Querier) StakingSequence(ctx context.Context, req *types.QueryStakingSequenceRequest) (*types.QueryStakingSequenceResponse, error) {
	panic("IM")
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
