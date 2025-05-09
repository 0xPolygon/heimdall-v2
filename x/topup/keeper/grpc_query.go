package keeper

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	heimdallTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/topup/types"
)

const MaxProofLength = 1024

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

	if !isValidTxHash(req.TxHash) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tx hash")
	}

	chainParams, err := q.k.ChainKeeper.GetParams(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	// get main tx receipt
	txHash := common.FromHex(req.TxHash)
	receipt, err := q.k.contractCaller.GetConfirmedTxReceipt(common.BytesToHash(txHash), chainParams.MainChainTxConfirmations)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if receipt == nil {
		return nil, status.Errorf(codes.NotFound, "receipt not found")
	}

	// get sequence id
	sequence := new(big.Int).Mul(receipt.BlockNumber, big.NewInt(types.DefaultLogIndexUnit))
	sequence.Add(sequence, new(big.Int).SetUint64(req.LogIndex))

	// check if incoming tx already exists
	exists, err := q.k.HasTopupSequence(ctx, sequence.String())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
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

	if !isValidTxHash(req.TxHash) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tx hash")
	}

	chainParams, err := q.k.ChainKeeper.GetParams(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	// get main tx receipt
	txHash := common.FromHex(req.TxHash)
	receipt, err := q.k.contractCaller.GetConfirmedTxReceipt(common.BytesToHash(txHash), chainParams.MainChainTxConfirmations)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if receipt == nil {
		return nil, status.Errorf(codes.NotFound, "receipt not found")
	}

	// get sequence id
	sequence := new(big.Int).Mul(receipt.BlockNumber, big.NewInt(types.DefaultLogIndexUnit))
	sequence.Add(sequence, new(big.Int).SetUint64(req.LogIndex))

	// check if incoming tx already exists
	exists, err := q.k.HasTopupSequence(ctx, sequence.String())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryIsTopupTxOldResponse{IsOld: exists}, nil
}

// GetDividendAccountByAddress implements the gRPC service handler to query a dividend account by its address
func (q queryServer) GetDividendAccountByAddress(ctx context.Context, req *types.QueryDividendAccountRequest) (*types.QueryDividendAccountResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	exists, err := q.k.HasDividendAccount(ctx, req.Address)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !exists {
		return nil, status.Errorf(codes.NotFound, "dividend account with address %s not found", req.Address)
	}

	dividendAccount, err := q.k.GetDividendAccount(ctx, req.Address)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryDividendAccountResponse{DividendAccount: dividendAccount}, nil
}

// GetDividendAccountRootHash implements the gRPC service handler to query the root hash of all dividend accounts
func (q queryServer) GetDividendAccountRootHash(ctx context.Context, _ *types.QueryDividendAccountRootHashRequest) (*types.QueryDividendAccountRootHashResponse, error) {
	dividendAccounts, err := q.k.GetAllDividendAccounts(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	accountRoot, err := heimdallTypes.GetAccountRootHash(dividendAccounts)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if len(accountRoot) == 0 {
		return nil, status.Errorf(codes.NotFound, "account root not found")
	}

	return &types.QueryDividendAccountRootHashResponse{AccountRootHash: accountRoot}, nil
}

// VerifyAccountProofByAddress implements the gRPC service handler to verify the account proof by its address
func (q queryServer) VerifyAccountProofByAddress(ctx context.Context, req *types.QueryVerifyAccountProofRequest) (*types.QueryVerifyAccountProofResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	if !common.IsHexAddress(req.Address) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid address")
	}

	if err := validateProof(req.Proof); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid proof: %s", err.Error())
	}

	dividendAccounts, err := q.k.GetAllDividendAccounts(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Verify account proof
	accountProofStatus, err := heimdallTypes.VerifyAccountProof(dividendAccounts, req.Address, req.Proof)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryVerifyAccountProofResponse{IsVerified: accountProofStatus}, nil
}

// GetAccountProofByAddress implements the gRPC service handler to get the account proof by its address
func (q queryServer) GetAccountProofByAddress(ctx context.Context, req *types.QueryAccountProofRequest) (*types.QueryAccountProofResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	if !common.IsHexAddress(req.Address) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid address")
	}

	// Fetch the AccountRoot from RootChainContract, then the AccountRoot from current account
	// Finally, if they are equal, calculate the merkle path using GetAllDividendAccounts

	chainParams, err := q.k.ChainKeeper.GetParams(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	stakingInfoAddress := chainParams.ChainParams.StakingInfoAddress
	stakingInfoInstance, err := q.k.contractCaller.GetStakingInfoInstance(stakingInfoAddress)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	accountRootOnChain, err := q.k.contractCaller.CurrentAccountStateRoot(stakingInfoInstance)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	dividendAccounts, err := q.k.GetAllDividendAccounts(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	currentStateAccountRoot, err := heimdallTypes.GetAccountRootHash(dividendAccounts)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get account root hash: %v", err))
	}

	if !bytes.Equal(accountRootOnChain[:], currentStateAccountRoot) {
		return nil, status.Errorf(codes.Internal, "accountRootOnChain does not match with currentStateAccountRoot")
	}

	// Calculate new account root hash
	merkleProof, index, err := heimdallTypes.GetAccountProof(dividendAccounts, req.Address)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// build response and return
	dividendAccountProof := &types.QueryAccountProofResponse{
		Proof: types.AccountProof{
			Address:      req.Address,
			AccountProof: merkleProof,
			Index:        index,
		},
	}

	return dividendAccountProof, nil
}

func isValidTxHash(hash string) bool {
	return strings.HasPrefix(hash, "0x") && len(hash) == 66
}

func validateProof(proof string) error {
	proofBytes := common.FromHex(proof)
	if proofBytes == nil || len(proofBytes) == 0 {
		return errors.New("proof is empty")
	}
	if len(proofBytes)%32 != 0 {
		return errors.New("invalid proof length, not multiple of 32 bytes")
	}
	if len(proofBytes) > MaxProofLength {
		return fmt.Errorf("proof length exceeds maximum limit of %d bytes", MaxProofLength)
	}
	return nil
}
