package keeper

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	heimdallTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/clerk/types"
)

type QueryServer struct{ K Keeper }

var _ types.QueryServer = QueryServer{}

func NewQueryServer(k Keeper) types.QueryServer {
	return QueryServer{K: k}
}

func (s QueryServer) Record(ctx context.Context, request *types.RecordRequest) (*types.RecordResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")

	}

	record, err := s.K.GetEventRecord(ctx, request.RecordID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())

	}

	return &types.RecordResponse{Record: record}, nil
}

func (s QueryServer) RecordList(ctx context.Context, request *types.RecordListRequest) (*types.RecordListResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")

	}

	records, err := s.K.GetEventRecordList(ctx, request.Page, request.Limit)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	newRecords := make([]*types.EventRecord, len(records))
	for i, record := range records {
		newRecords[i] = &record
	}

	return &types.RecordListResponse{EventRecords: newRecords}, nil
}

func (s QueryServer) RecordListWithTime(ctx context.Context, request *types.RecordListWithTimeRequest) (*types.RecordListWithTimeResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")

	}

	records, err := s.K.GetEventRecordListWithTime(ctx, request.FromTime, request.ToTime, request.Page, request.Limit)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	newRecords := make([]*types.EventRecord, len(records))
	for i, record := range records {
		newRecords[i] = &record
	}

	return &types.RecordListWithTimeResponse{EventRecords: newRecords}, nil
}

func (s QueryServer) RecordSequence(ctx context.Context, request *types.RecordSequenceRequest) (*types.RecordSequenceResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")

	}

	chainParams, err := s.K.ChainKeeper.GetParams(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	// get main tx receipt
	txHash := heimdallTypes.TxHash{Hash: common.FromHex(request.TxHash)}
	receipt, err := s.K.contractCaller.GetConfirmedTxReceipt(common.BytesToHash(txHash.Hash), chainParams.GetMainChainTxConfirmations())
	if err != nil || receipt == nil {
		return nil, status.Errorf(codes.Internal, "transaction is not confirmed yet. please wait for sometime and try again")
	}

	// sequence id
	sequence := new(big.Int).Mul(receipt.BlockNumber, big.NewInt(heimdallTypes.DefaultLogIndexUnit))
	sequence.Add(sequence, new(big.Int).SetUint64(request.LogIndex))
	// check if incoming tx already exists
	if !s.K.HasRecordSequence(ctx, sequence.String()) {
		return nil, nil
	}

	return &types.RecordSequenceResponse{Sequence: sequence.Uint64()}, nil
}
