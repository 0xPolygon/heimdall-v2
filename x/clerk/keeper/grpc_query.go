package keeper

import (
	"context"
	"math/big"

	heimdallTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/clerk/types"
	"github.com/ethereum/go-ethereum/common"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ types.QueryServer = queryServer{}

type queryServer struct {
	k *Keeper
}

// NewQueryServer creates a new querier for clerk clients.
func NewQueryServer(k *Keeper) types.QueryServer {
	return queryServer{
		k: k,
	}
}

func (q queryServer) GetRecordByID(ctx context.Context, request *types.RecordRequest) (*types.RecordResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	record, err := q.k.GetEventRecord(ctx, request.RecordId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.RecordResponse{Record: *record}, nil
}

func (q queryServer) GetRecordList(ctx context.Context, request *types.RecordListRequest) (*types.RecordListResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	records, err := q.k.GetEventRecordList(ctx, request.Page, request.Limit)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	newRecords := make([]types.EventRecord, len(records))
	copy(newRecords, records)

	return &types.RecordListResponse{EventRecords: newRecords}, nil
}

func (q queryServer) GetRecordListWithTime(ctx context.Context, request *types.RecordListWithTimeRequest) (*types.RecordListWithTimeResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	records, err := q.k.GetEventRecordListWithTime(ctx, request.FromTime, request.ToTime, request.Page, request.Limit)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	newRecords := make([]types.EventRecord, len(records))
	copy(newRecords, records)

	return &types.RecordListWithTimeResponse{EventRecords: newRecords}, nil
}

func (q queryServer) GetRecordSequence(ctx context.Context, request *types.RecordSequenceRequest) (*types.RecordSequenceResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	chainParams, err := q.k.ChainKeeper.GetParams(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// get main tx receipt
	txHash := common.FromHex(request.TxHash)
	receipt, err := q.k.contractCaller.GetConfirmedTxReceipt(common.BytesToHash(txHash), chainParams.GetMainChainTxConfirmations())
	if err != nil || receipt == nil {
		return nil, status.Errorf(codes.Internal, "transaction is not confirmed yet. please wait for sometime and try again")
	}

	// sequence id
	sequence := new(big.Int).Mul(receipt.BlockNumber, big.NewInt(heimdallTypes.DefaultLogIndexUnit))
	sequence.Add(sequence, new(big.Int).SetUint64(request.LogIndex))
	// check if incoming tx already exists
	if !q.k.HasRecordSequence(ctx, sequence.String()) {
		return nil, status.Error(codes.NotFound, "record sequence not found")
	}

	return &types.RecordSequenceResponse{Sequence: sequence.Uint64()}, nil
}

// IsClerkTxOld implements the gRPC service handler to query the status of a clerk tx
func (q queryServer) IsClerkTxOld(ctx context.Context, request *types.RecordSequenceRequest) (*types.IsClerkTxOldResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	chainParams, err := q.k.ChainKeeper.GetParams(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// get main tx receipt
	txHash := common.FromHex(request.TxHash)
	receipt, err := q.k.contractCaller.GetConfirmedTxReceipt(common.BytesToHash(txHash), chainParams.GetMainChainTxConfirmations())
	if err != nil || receipt == nil {
		return nil, status.Errorf(codes.Internal, "transaction is not confirmed yet. please wait for sometime and try again")
	}

	// sequence id
	sequence := new(big.Int).Mul(receipt.BlockNumber, big.NewInt(heimdallTypes.DefaultLogIndexUnit))
	sequence.Add(sequence, new(big.Int).SetUint64(request.LogIndex))

	// check if incoming tx already exists
	if !q.k.HasRecordSequence(ctx, sequence.String()) {
		return nil, status.Error(codes.NotFound, "record sequence not found")
	}

	return &types.IsClerkTxOldResponse{IsOld: true}, nil
}
