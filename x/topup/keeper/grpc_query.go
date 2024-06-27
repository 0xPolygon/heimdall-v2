package keeper

import (
	"bytes"
	"context"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	heimdallTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/topup/types"
)

var _ types.QueryServer = queryServer{}

type queryServer struct {
	k *Keeper
}

// NewQueryServer creates a new querier for topup clients.
// It uses the underlying keeper and its contractCaller to interact with Ethereum chain.
func NewQueryServer(k *Keeper) types.QueryServer {
	return queryServer{
		k: k,
	}
}

// GetTopupTxSequence implements the gRPC service handler to query the sequence of a topup tx
func (q queryServer) GetTopupTxSequence(ctx context.Context, req *types.QueryTopupSequenceRequest) (*types.QueryTopupSequenceResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	chainParams, err := q.k.ChainKeeper.GetParams(sdkCtx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	// get main tx receipt
	txHash := heimdallTypes.TxHash{Hash: common.FromHex(req.TxHash)}
	receipt, err := q.k.contractCaller.GetConfirmedTxReceipt(common.BytesToHash(txHash.Hash), chainParams.MainChainTxConfirmations)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if receipt == nil {
		return nil, status.Errorf(codes.NotFound, "receipt not found")
	}

	// get sequence id
	sequence := new(big.Int).Mul(receipt.BlockNumber, big.NewInt(types.DefaultLogIndexUnit))
	sequence.Add(sequence, new(big.Int).SetUint64(req.LogIndex))

	// check if incoming tx already exists
	exists, err := q.k.HasTopupSequence(sdkCtx, sequence.String())
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	if !exists {
		q.k.Logger(ctx).Error("sequence does not exist", "txHash", req.TxHash, "index", req.LogIndex)
		return nil, status.Errorf(codes.NotFound, "sequence with hash %s not found", req.TxHash)
	}

	return &types.QueryTopupSequenceResponse{Sequence: sequence.String()}, nil
}

// IsTopupTxOld implements the gRPC service handler to query the status of a topup tx
func (q queryServer) IsTopupTxOld(ctx context.Context, req *types.QueryTopupSequenceRequest) (*types.QueryIsTopupTxOldResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	chainParams, err := q.k.ChainKeeper.GetParams(sdkCtx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	// get main tx receipt
	txHash := heimdallTypes.TxHash{Hash: common.FromHex(req.TxHash)}
	receipt, err := q.k.contractCaller.GetConfirmedTxReceipt(common.BytesToHash(txHash.Hash), chainParams.MainChainTxConfirmations)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if receipt == nil {
		return nil, status.Errorf(codes.NotFound, "receipt not found")
	}

	// get sequence id
	sequence := new(big.Int).Mul(receipt.BlockNumber, big.NewInt(types.DefaultLogIndexUnit))
	sequence.Add(sequence, new(big.Int).SetUint64(req.LogIndex))

	// check if incoming tx already exists
	exists, err := q.k.HasTopupSequence(sdkCtx, sequence.String())
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &types.QueryIsTopupTxOldResponse{IsOld: exists}, nil
}

// GetDividendAccountByAddress implements the gRPC service handler to query a dividend account by its address
func (q queryServer) GetDividendAccountByAddress(ctx context.Context, req *types.QueryDividendAccountRequest) (*types.QueryDividendAccountResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	exists, err := q.k.HasDividendAccount(sdkCtx, req.Address)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if !exists {
		return nil, status.Errorf(codes.NotFound, "dividend account with address %s not found", req.Address)
	}

	dividendAccount, err := q.k.GetDividendAccount(sdkCtx, req.Address)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &types.QueryDividendAccountResponse{DividendAccount: dividendAccount}, nil
}

func (q queryServer) GetDividendAccountRootHash(ctx context.Context, req *types.QueryDividendAccountRootHashRequest) (*types.QueryDividendAccountRootHashResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	dividendAccounts, err := q.k.GetAllDividendAccounts(sdkCtx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	accountRoot, err := heimdallTypes.GetAccountRootHash(dividendAccounts)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if len(accountRoot) == 0 {
		return nil, status.Errorf(codes.NotFound, "account root not found")
	}

	return &types.QueryDividendAccountRootHashResponse{AccountRootHash: &heimdallTypes.HeimdallHash{Hash: accountRoot}}, nil
}

func (q queryServer) VerifyAccountProof(ctx context.Context, req *types.QueryVerifyAccountProofRequest) (*types.QueryVerifyAccountProofResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	dividendAccounts, err := q.k.GetAllDividendAccounts(sdkCtx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	// Verify account proof
	accountProofStatus, err := heimdallTypes.VerifyAccountProof(dividendAccounts, req.Address, req.Proof)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &types.QueryVerifyAccountProofResponse{IsVerified: accountProofStatus}, nil

}

func (q queryServer) GetAccountProof(ctx context.Context, req *types.QueryAccountProofRequest) (*types.QueryAccountProofResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	// Fetch the AccountRoot from RootChainContract, then the AccountRoot from current account
	// Finally, if they are equal, calculate the merkle path using GetAllDividendAccounts

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	chainParams, err := q.k.ChainKeeper.GetParams(sdkCtx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	stakingInfoAddress := chainParams.ChainParams.StakingInfoAddress
	stakingInfoInstance, err := q.k.contractCaller.GetStakingInfoInstance(stakingInfoAddress)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	accountRootOnChain, err := q.k.contractCaller.CurrentAccountStateRoot(stakingInfoInstance)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	dividendAccounts, err := q.k.GetAllDividendAccounts(sdkCtx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	currentStateAccountRoot, err := heimdallTypes.GetAccountRootHash(dividendAccounts)
	if !bytes.Equal(accountRootOnChain[:], currentStateAccountRoot) {
		return nil, status.Errorf(codes.Internal, "accountRootOnChain does not match with currentStateAccountRoot")
	}

	// Calculate new account root hash
	merkleProof, index, err := heimdallTypes.GetAccountProof(dividendAccounts, req.Address)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	// build response and return
	dividendAccountProof := &types.QueryAccountProofResponse{
		Proof: &types.AccountProof{
			Address:      req.Address,
			AccountProof: merkleProof,
			Index:        index,
		},
	}

	return dividendAccountProof, nil
}
