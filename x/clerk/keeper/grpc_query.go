package keeper

import (
	"context"
	"math/big"

	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/ethereum/go-ethereum/common"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/0xPolygon/heimdall-v2/common/hex"
	heimdallTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/clerk/types"
)

const (
	// DefaultPageLimit is the default page limit for queries.
	DefaultPageLimit = 1

	// DefaultRecordListLimit is the default record list limit for queries.
	DefaultRecordListLimit = 50

	// MaxRecordListLimitPerPage is the maximum record list limit per page for queries.
	MaxRecordListLimitPerPage = 1000
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
		request.Page = DefaultPageLimit
	}

	if request.Limit == 0 || request.Limit > MaxRecordListLimitPerPage {
		request.Limit = DefaultRecordListLimit
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

	if isPaginationEmpty(request.Pagination) && request.Pagination.Limit > MaxRecordListLimitPerPage {
		return nil, status.Errorf(codes.InvalidArgument, "pagination request is empty (at least one of offset, key, or limit must be set) and limit exceeds max allowed limit %d", MaxRecordListLimitPerPage)
	}

	if request.Pagination.Limit == 0 || request.Pagination.Limit > MaxRecordListLimitPerPage {
		request.Pagination.Limit = DefaultRecordListLimit
	}

	if request.FromId < 1 {
		return nil, status.Errorf(codes.InvalidArgument, "fromId should start from at least 1")
	}

	filtered := make([]types.EventRecord, 0, request.Pagination.Limit)

	for i := uint64(0); i < request.Pagination.Limit; i++ {
		value, err := q.k.RecordsWithID.Get(ctx, request.FromId)
		if err != nil {
			q.k.Logger(ctx).Debug("error in fetching event record", "error", err, "fromId", request.FromId)
			break
		}

		if value.RecordTime.Before(request.ToTime) {
			filtered = append(filtered, value)
			request.FromId++ // Increment FromId until we find a valid record or run out of records.
			continue
		}

		break
	}

	if len(filtered) == 0 {
		return &types.RecordListWithTimeResponse{
			EventRecords: []types.EventRecord{},
		}, nil
	}

	// Apply pagination over the filtered result.
	paginatedRecords := filterWithPage(filtered, &request.Pagination)

	return &types.RecordListWithTimeResponse{
		EventRecords: paginatedRecords,
	}, nil
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

// GetRecordCount implements the gRPC service handler to query the total count of event records.
func (q queryServer) GetRecordCount(ctx context.Context, _ *types.RecordCountRequest) (*types.RecordCountResponse, error) {
	return &types.RecordCountResponse{Count: q.k.GetEventRecordCount(ctx)}, nil
}

func isPaginationEmpty(p query.PageRequest) bool {
	return p.Key == nil &&
		p.Offset == 0 &&
		p.Limit == 0 &&
		!p.CountTotal &&
		!p.Reverse
}

func filterWithPage(records []types.EventRecord, pagination *query.PageRequest) []types.EventRecord {
	if pagination == nil {
		return records
	}

	start := int(pagination.Offset)
	end := start + int(pagination.Limit)

	if start >= len(records) {
		return []types.EventRecord{}
	}
	if end > len(records) {
		end = len(records)
	}
	return records[start:end]
}
