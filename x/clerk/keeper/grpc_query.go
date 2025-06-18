package keeper

import (
	"context"
	"math/big"
	"time"

	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/ethereum/go-ethereum/common"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/0xPolygon/heimdall-v2/common/hex"
	heimdallTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/clerk/types"
)

const (
	defaultPageLimit          = 1
	defaultRecordListLimit    = 50
	maxRecordListLimitPerPage = 1000
	maxTimeWindow             = time.Hour * 24 * 2 // 2 days of data
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

func (q queryServer) GetRecordById(ctx context.Context, request *types.RecordRequest) (*types.RecordResponse, error) {
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

	if request.Page == 0 {
		request.Page = defaultPageLimit
	}

	if request.Limit == 0 {
		request.Limit = defaultRecordListLimit
	} else if request.Limit > maxRecordListLimitPerPage {
		return nil, status.Errorf(codes.InvalidArgument, "limit cannot exceed %d", maxRecordListLimitPerPage)
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

	if isPaginationEmpty(request.Pagination) {
		return nil, status.Errorf(codes.InvalidArgument, "pagination request is empty: at least one of offset, key, or limit must be set")
	}

	if request.Pagination.Limit == 0 {
		request.Pagination.Limit = defaultRecordListLimit
	} else if request.Pagination.Limit > maxRecordListLimitPerPage {
		return nil, status.Errorf(codes.InvalidArgument, "pagination limit cannot exceed %d", maxRecordListLimitPerPage)
	}

	if request.ToTime.After(time.Now()) {
		return nil, status.Errorf(codes.InvalidArgument, "to_time cannot be in the future")
	}

	record, err := q.GetRecordById(ctx, &types.RecordRequest{RecordId: request.FromId})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error during GetRecordListWithTime while getting record by id: %d, %v", request.FromId, err)
	}

	if request.ToTime.Sub(record.Record.RecordTime) > maxTimeWindow {
		return nil, status.Errorf(codes.InvalidArgument, "time range too long: from %v to %v exceeds max allowed duration %v",
			record.Record.RecordTime, request.ToTime, maxTimeWindow)
	}

	res, _, err := query.CollectionPaginate(
		ctx,
		q.k.RecordsWithID,
		&request.Pagination, func(id uint64, record types.EventRecord) (*types.EventRecord, error) {
			return q.k.GetEventRecord(ctx, id)
		},
	)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "paginate: %v", err)
	}

	records := make([]types.EventRecord, 0, len(res))
	for _, r := range res {
		if r.Id >= request.FromId && r.RecordTime.Before(request.ToTime) {
			records = append(records, *r)
			if len(records) >= maxRecordListLimitPerPage {
				break
			}
		}
	}

	return &types.RecordListWithTimeResponse{EventRecords: records}, nil
}

func (q queryServer) GetRecordSequence(ctx context.Context, request *types.RecordSequenceRequest) (*types.RecordSequenceResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if !hex.IsTxHashNonEmpty(request.TxHash) {
		return nil, status.Error(codes.InvalidArgument, "invalid tx hash")
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

	if !hex.IsTxHashNonEmpty(request.TxHash) {
		return nil, status.Error(codes.InvalidArgument, "invalid tx hash")
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

// GetLatestRecordId implements the gRPC service handler to query the latest record id from L1.
func (q queryServer) GetLatestRecordId(ctx context.Context, _ *types.LatestRecordIdRequest) (*types.LatestRecordIdResponse, error) {
	// Get chain params to get the StateSender contract address.
	chainParams, err := q.k.ChainKeeper.GetParams(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Get the StateSender contract instance.
	stateSenderInstance, err := q.k.contractCaller.GetStateSenderInstance(chainParams.ChainParams.StateSenderAddress)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get state sender instance")
	}

	// Get the current state counter from L1.
	stateCounter := q.k.contractCaller.CurrentStateCounter(stateSenderInstance)
	if stateCounter == nil {
		return nil, status.Error(codes.Internal, "failed to get latest state counter from L1")
	}

	latestRecordId := stateCounter.Uint64()
	eventRecordExists := q.k.HasEventRecord(ctx, latestRecordId)
	return &types.LatestRecordIdResponse{LatestRecordId: latestRecordId, IsProcessedByHeimdall: eventRecordExists}, nil
}

func isPaginationEmpty(p query.PageRequest) bool {
	return p.Key == nil &&
		p.Offset == 0 &&
		p.Limit == 0 &&
		!p.CountTotal &&
		!p.Reverse
}
