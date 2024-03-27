package keeper

import (
	"context"
	"github.com/0xPolygon/heimdall-v2/x/topup/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"math/big"
)

var _ types.QueryServer = queryServer{}

type queryServer struct {
	k *Keeper
}

// NewQueryServer creates a new querier for topup clients.
// Besides the keeper, it also takes in the contractCaller to interact with ethereum chain
func NewQueryServer(k *Keeper /*, contractCaller helper.IContractCaller */) types.QueryServer {
	return queryServer{
		k: k,
	}
}

func (q queryServer) TopupTxStatus(ctx context.Context, req *types.QuerySequenceParams) (*types.QuerySequenceParamsResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	chainParams := q.k.chainKeeper.GetParams(sdkCtx)
	// get main tx receipt
	receipt, err := q.k.contractCaller.GetConfirmedTxReceipt(types.HexToHeimdallHash(req.TxHash).EthHash(), chainParams.MainchainTxConfirmations)
	if err != nil || receipt == nil {
		return nil, status.Errorf(codes.Internal, err.Error())
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

	return &types.QuerySequenceParamsResponse{Sequence: sequence.String()}, nil
}

// DividendAccountByAddress implements the gRPC service handler to query a dividend account by its address
func (q queryServer) DividendAccountByAddress(ctx context.Context, req *types.QueryDividendAccountParams) (*types.QueryDividendAccountParamsResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	dividendAccount, err := q.k.GetDividendAccount(sdkCtx, req.Address)
	if err != nil {
		return nil, err
	}
	return &types.QueryDividendAccountParamsResponse{DividendAccount: dividendAccount}, nil
}

func (q queryServer) DividendAccountRoot(ctx context.Context, req *types.QueryDividendAccountRootParams) (*types.QueryDividendAccountRootResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	// TODO HV2: replace _ with dividendAccounts
	_, err := q.k.GetAllDividendAccounts(sdkCtx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	/* TODO HV2: enable this when checkpoint is implemented in heimdall-v2
	accountRoot, err := checkpointTypes.GetAccountRootHash(dividendAccounts)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	*/

	// TODO HV2: return accountRoot instead of nil
	return &types.QueryDividendAccountRootResponse{AccountRootHash: nil}, nil
}

func (q queryServer) VerifyAccountProof(ctx context.Context, req *types.QueryVerifyAccountProofParams) (*types.QueryVerifyAccountProofResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// TODO HV2: replace _ with dividendAccounts
	_, err := q.k.GetAllDividendAccounts(sdkCtx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	// Verify account proof
	// TODO HV2: enable when checkpoint is implemented in heimdall-v2
	// accountProofStatus, err := checkpointTypes.VerifyAccountProof(dividendAccounts, req.Address, req.Proof)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	// TODO HV2: replace false with accountProofStatus
	return &types.QueryVerifyAccountProofResponse{Result: false}, nil

}

func (q queryServer) DividendAccountProof(ctx context.Context, req *types.QueryDividendAccountProofParams) (*types.QueryDividendAccountProofResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	/* TODO HV2: enable this when chainManager, checkpoint and contractCaller are implemented in heimdall-v2
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	chainParams := q.k.chainKeeper.GetParams(sdkCtx)
	stakingInfoAddress := chainParams.ChainParams.StakingInfoAddress.EthAddress()
	stakingInfoInstance, _ := q.k.contractCaller.GetStakingInfoInstance(stakingInfoAddress)
	accountRootOnChain, err := q.k.contractCaller.CurrentAccountStateRoot(stakingInfoInstance)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	dividendAccounts, err := q.k.GetAllDividendAccounts(sdkCtx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	currentStateAccountRoot, err := checkpointTypes.GetAccountRootHash(dividendAccounts)
	if !bytes.Equal(accountRootOnChain[:], currentStateAccountRoot) {
		return nil, status.Errorf(codes.Internal, err.Error())
	} else {
		// Calculate new account root hash
		merkleProof, index, err := checkpointTypes.GetAccountProof(dividendAccounts, req.Address)
		if err != nil {
			return nil, status.Errorf(codes.Internal, err.Error())
		}
		// build response and return
		dividendAccountProof := &types.DividendAccountProof{
			User:         req.Address,
			AccountProof: merkleProof,
			Index:        index,
		}

		return &types.QueryDividendAccountProofResponse{Result: dividendAccountProof}, nil
	}
	*/

	// TODO HV2: remove the "return nil, nil" when the above method is enabled
	return nil, nil
}
