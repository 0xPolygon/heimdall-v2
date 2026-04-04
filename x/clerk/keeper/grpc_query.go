package keeper

import (
	"context"
	"errors"
	"math/big"
	"time"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/ethereum/go-ethereum/common"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/0xPolygon/heimdall-v2/common/hex"
	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/metrics/api"
	heimdallTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/clerk/types"
)

const (
	// MaxRecordListLimit is the maximum record list limit for queries.
	MaxRecordListLimit = 50

	errEmptyRequest = "empty request"
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
	var err error
	startTime := time.Now()
	defer recordClerkQueryMetric(api.GetRecordByIdMethod, startTime, &err)

	if request == nil {
		return nil, status.Error(codes.InvalidArgument, errEmptyRequest)
	}

	record, err := q.k.GetEventRecord(ctx, request.RecordId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.RecordResponse{Record: *record}, nil
}

func (q queryServer) GetRecordList(ctx context.Context, request *types.RecordListRequest) (*types.RecordListResponse, error) {
	var err error
	startTime := time.Now()
	defer recordClerkQueryMetric(api.GetRecordListMethod, startTime, &err)

	if request == nil {
		return nil, status.Error(codes.InvalidArgument, errEmptyRequest)
	}

	if request.Page == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "page cannot be 0")
	}
	if request.Limit == 0 || request.Limit > MaxRecordListLimit {
		return nil, status.Errorf(codes.InvalidArgument, "limit cannot be 0 or greater than %d", MaxRecordListLimit)
	}

	records, err := q.k.GetEventRecordList(ctx, request.Page, request.Limit)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.RecordListResponse{EventRecords: records}, nil
}

func (q queryServer) GetRecordListWithTime(ctx context.Context, request *types.RecordListWithTimeRequest) (_ *types.RecordListWithTimeResponse, err error) {
	startTime := time.Now()
	defer recordClerkQueryMetric(api.GetRecordListWithTimeMethod, startTime, &err)

	if request == nil {
		return nil, status.Error(codes.InvalidArgument, errEmptyRequest)
	}

	if isPaginationEmpty(request.Pagination) {
		return nil, status.Errorf(codes.InvalidArgument, "pagination request is empty (at least one argument must be set)")
	}
	if request.Pagination.Limit == 0 || request.Pagination.Limit > MaxRecordListLimit {
		return nil, status.Errorf(codes.InvalidArgument, "limit cannot be 0 or greater than %d", MaxRecordListLimit)
	}
	if request.FromId < 1 {
		return nil, status.Errorf(codes.InvalidArgument, "fromId cannot be less than 1")
	}
	if request.ToTime.IsZero() {
		return nil, status.Errorf(codes.InvalidArgument, "to_time must be set")
	}
	if request.ToTime.Unix() <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "to_time must be greater than Unix epoch")
	}

	// Collect the records based on pagination parameters.
	result := make([]types.EventRecord, 0, request.Pagination.Limit)

	// Use a range iterator starting from FromId.
	rng := (&collections.Range[uint64]{}).StartInclusive(request.FromId)

	iterator, err := q.k.RecordsWithID.Iterate(ctx, rng)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	defer func(iterator collections.Iterator[uint64, types.EventRecord]) {
		err := iterator.Close()
		if err != nil {
			q.k.Logger(ctx).Error("Error in closing event record iterator", "error", err)
		}
	}(iterator)

	skipped := uint64(0)   // Records skipped based on pagination offset.
	collected := uint64(0) // Records collected based on pagination limit.

	for ; iterator.Valid(); iterator.Next() {
		var value types.EventRecord
		value, err = iterator.Value()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "error reading event record from iterator: %v", err)
		}

		// Always filter by record_time for backward compatibility.
		// Post heimdall fork, clients switch to GetRecordListVisibleAtHeight which uses
		// visibility_height for deterministic filtering.
		if !value.RecordTime.Before(request.ToTime) {
			break
		}

		// Skip records based on the pagination offset.
		if skipped < request.Pagination.Offset {
			skipped++
			continue
		}

		// Collect records up to the limit.
		if collected < request.Pagination.Limit {
			result = append(result, value)
			collected++
		} else {
			// We have collected enough records, stop iterating.
			break
		}
	}

	if len(result) == 0 {
		return &types.RecordListWithTimeResponse{
			EventRecords: []types.EventRecord{},
		}, nil
	}

	return &types.RecordListWithTimeResponse{
		EventRecords: result,
	}, nil
}

func (q queryServer) GetRecordSequence(ctx context.Context, request *types.RecordSequenceRequest) (*types.RecordSequenceResponse, error) {
	var err error
	startTime := time.Now()
	defer recordClerkQueryMetric(api.GetRecordSequenceMethod, startTime, &err)

	if request == nil {
		return nil, status.Error(codes.InvalidArgument, errEmptyRequest)
	}

	if !hex.IsTxHashNonEmpty(request.TxHash) {
		return nil, status.Error(codes.InvalidArgument, "invalid tx hash")
	}

	chainParams, err := q.k.ChainKeeper.GetParams(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Get the main tx receipt.
	txHash := common.FromHex(request.TxHash)
	receipt, err := q.k.contractCaller.GetConfirmedTxReceipt(common.BytesToHash(txHash), chainParams.GetMainChainTxConfirmations())
	if err != nil || receipt == nil {
		return nil, status.Errorf(codes.Internal, "transaction is not confirmed yet. please wait for sometime and try again")
	}

	// Get the sequence id.
	sequence := new(big.Int).Mul(receipt.BlockNumber, big.NewInt(heimdallTypes.DefaultLogIndexUnit))
	sequence.Add(sequence, new(big.Int).SetUint64(request.LogIndex))
	// Check if the incoming tx already exists.
	if !q.k.HasRecordSequence(ctx, sequence.String()) {
		return nil, status.Error(codes.NotFound, "record sequence not found")
	}

	return &types.RecordSequenceResponse{Sequence: sequence.Uint64()}, nil
}

// IsClerkTxOld implements the gRPC service handler to query the status of a clerk tx
func (q queryServer) IsClerkTxOld(ctx context.Context, request *types.RecordSequenceRequest) (*types.IsClerkTxOldResponse, error) {
	var err error
	startTime := time.Now()
	defer recordClerkQueryMetric(api.IsClerkTxOldMethod, startTime, &err)

	if request == nil {
		return nil, status.Error(codes.InvalidArgument, errEmptyRequest)
	}

	if !hex.IsTxHashNonEmpty(request.TxHash) {
		return nil, status.Error(codes.InvalidArgument, "invalid tx hash")
	}

	chainParams, err := q.k.ChainKeeper.GetParams(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Get the main tx receipt.
	txHash := common.FromHex(request.TxHash)
	receipt, err := q.k.contractCaller.GetConfirmedTxReceipt(common.BytesToHash(txHash), chainParams.GetMainChainTxConfirmations())
	if err != nil || receipt == nil {
		return nil, status.Errorf(codes.Internal, "transaction is not confirmed yet. please wait for sometime and try again")
	}

	// Get the sequence id.
	sequence := new(big.Int).Mul(receipt.BlockNumber, big.NewInt(heimdallTypes.DefaultLogIndexUnit))
	sequence.Add(sequence, new(big.Int).SetUint64(request.LogIndex))

	// Check if the incoming tx already exists.
	if !q.k.HasRecordSequence(ctx, sequence.String()) {
		return nil, status.Error(codes.NotFound, "record sequence not found")
	}

	return &types.IsClerkTxOldResponse{IsOld: true}, nil
}

// GetLatestRecordId implements the gRPC service handler to query the latest record id from L1.
func (q queryServer) GetLatestRecordId(ctx context.Context, _ *types.LatestRecordIdRequest) (*types.LatestRecordIdResponse, error) {
	var err error
	startTime := time.Now()
	defer recordClerkQueryMetric(api.GetLatestRecordIdMethod, startTime, &err)

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
	var err error
	startTime := time.Now()
	defer recordClerkQueryMetric(api.GetRecordCountMethod, startTime, &err)

	return &types.RecordCountResponse{Count: q.k.GetEventRecordCount(ctx)}, nil
}

// GetBlockHeightByTime returns the greatest committed Heimdall height whose header
// timestamp is <= the given cutoff time. This is used by Bor to resolve a deterministic
// Heimdall height for height-pinned state sync queries.
func (q queryServer) GetBlockHeightByTime(ctx context.Context, request *types.BlockHeightByTimeRequest) (_ *types.BlockHeightByTimeResponse, err error) {
	startTime := time.Now()
	defer recordClerkQueryMetric(api.GetBlockHeightByTimeMethod, startTime, &err)

	if request == nil {
		return nil, status.Error(codes.InvalidArgument, errEmptyRequest)
	}
	if request.CutoffTime <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "cutoff_time must be a positive unix timestamp, got %d", request.CutoffTime)
	}

	height, err := q.k.GetBlockHeightByTime(ctx, request.CutoffTime)
	if err != nil {
		if errors.Is(err, ErrNoBlockFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.BlockHeightByTimeResponse{Height: height}, nil
}

// GetRecordListVisibleAtHeight queries events visible at a specific Heimdall height.
// Used by Bor for deterministic state syncs: all validators derive the same height
// from the same Bor block header and get identical results from the latest Heimdall state.
func (q queryServer) GetRecordListVisibleAtHeight(ctx context.Context, request *types.RecordListVisibleAtHeightRequest) (_ *types.RecordListVisibleAtHeightResponse, err error) {
	startTime := time.Now()
	defer recordClerkQueryMetric(api.GetRecordListVisibleAtHeightMethod, startTime, &err)

	if request == nil {
		return nil, status.Error(codes.InvalidArgument, errEmptyRequest)
	}

	if request.FromId < 1 {
		return nil, status.Errorf(codes.InvalidArgument, "from_id cannot be less than 1")
	}
	if request.HeimdallHeight <= helper.GetInitialHeight() {
		return nil, status.Errorf(codes.InvalidArgument, "heimdall_height must be greater than initial height %d", helper.GetInitialHeight())
	}
	if isPaginationEmpty(request.Pagination) {
		return nil, status.Errorf(codes.InvalidArgument, "pagination request is empty (at least one argument must be set)")
	}
	if request.Pagination.Limit == 0 || request.Pagination.Limit > MaxRecordListLimit {
		return nil, status.Errorf(codes.InvalidArgument, "limit cannot be 0 or greater than %d", MaxRecordListLimit)
	}
	if request.ToTime.IsZero() {
		return nil, status.Errorf(codes.InvalidArgument, "to_time must be set")
	}
	if request.ToTime.Unix() <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "to_time must be greater than Unix epoch")
	}

	// Reject future heights
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()
	if request.HeimdallHeight > currentHeight {
		return nil, status.Errorf(codes.InvalidArgument,
			"heimdall_height %d exceeds current committed height %d", request.HeimdallHeight, currentHeight)
	}

	records, err := q.recordListVisibleAtHeight(ctx, request.FromId, request.HeimdallHeight, request.ToTime, request.Pagination)
	if err != nil {
		return nil, err
	}

	return &types.RecordListVisibleAtHeightResponse{
		EventRecords: records,
	}, nil
}

// recordListVisibleAtHeight is the shared implementation used by both
// GetRecordListVisibleAtHeight and GetStateSyncsByTime. Extracting it avoids
// double-counting Prometheus metrics when the combined endpoint delegates.
func (q queryServer) recordListVisibleAtHeight(
	ctx context.Context,
	fromId uint64,
	heimdallHeight int64,
	toTime time.Time,
	pagination query.PageRequest,
) ([]types.EventRecord, error) {
	// Determine the upgrade boundary. If not set, all events use the legacy path.
	upgradeId, err := q.k.GetVisibilityTimeUpgradeID(ctx)
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return nil, status.Errorf(codes.Internal, "failed to get visibility time upgrade ID: %v", err)
		}
		upgradeId = ^uint64(0)
	}

	requestedHeight := uint64(heimdallHeight)

	result := make([]types.EventRecord, 0, pagination.Limit)

	rng := (&collections.Range[uint64]{}).StartInclusive(fromId)

	iterator, err := q.k.RecordsWithID.Iterate(ctx, rng)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	defer func(iterator collections.Iterator[uint64, types.EventRecord]) {
		err := iterator.Close()
		if err != nil {
			q.k.Logger(ctx).Error("Error in closing event record iterator", "error", err)
		}
	}(iterator)

	skipped := uint64(0)
	collected := uint64(0)

	for ; iterator.Valid(); iterator.Next() {
		var value types.EventRecord
		value, err = iterator.Value()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "error reading event record from iterator: %v", err)
		}

		if value.Id < upgradeId {
			// Legacy event: filter by record_time < to_time for backward compatibility
			if !value.RecordTime.Before(toTime) {
				break
			}
		} else {
			// Post-upgrade event: filter by visibility_height and record_time.
			var visibilityHeight uint64
			visibilityHeight, err = q.k.GetVisibilityHeightForEvent(ctx, value.Id)
			if err != nil {
				if errors.Is(err, collections.ErrNotFound) {
					// Event exists but has no visibility_height yet (still pending).
					break
				}
				return nil, status.Errorf(codes.Internal, "failed to get visibility height for event %d: %v", value.Id, err)
			}
			if visibilityHeight > requestedHeight {
				break
			}
			if !value.RecordTime.Before(toTime) {
				break
			}
		}

		if skipped < pagination.Offset {
			skipped++
			continue
		}

		if collected < pagination.Limit {
			result = append(result, value)
			collected++
		} else {
			break
		}
	}

	if len(result) == 0 {
		return []types.EventRecord{}, nil
	}

	return result, nil
}

// GetStateSyncsByTime resolves a cutoff time to a Heimdall height, then returns
// events visible at that height. Combines GetBlockHeightByTime + GetRecordListVisibleAtHeight.
func (q queryServer) GetStateSyncsByTime(ctx context.Context, request *types.StateSyncsByTimeRequest) (_ *types.StateSyncsByTimeResponse, err error) {
	startTime := time.Now()
	defer recordClerkQueryMetric(api.GetStateSyncsByTimeMethod, startTime, &err)

	if request == nil {
		return nil, status.Error(codes.InvalidArgument, errEmptyRequest)
	}
	if request.FromId < 1 {
		return nil, status.Errorf(codes.InvalidArgument, "from_id cannot be less than 1")
	}
	if request.ToTime.IsZero() {
		return nil, status.Errorf(codes.InvalidArgument, "to_time must be set")
	}
	if request.ToTime.Unix() <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "to_time must be greater than Unix epoch")
	}
	if isPaginationEmpty(request.Pagination) {
		return nil, status.Errorf(codes.InvalidArgument, "pagination request is empty (at least one argument must be set)")
	}
	if request.Pagination.Limit == 0 || request.Pagination.Limit > MaxRecordListLimit {
		return nil, status.Errorf(codes.InvalidArgument, "limit cannot be 0 or greater than %d", MaxRecordListLimit)
	}

	// Stability gate: the resolved height is only frozen once a heimdall block with
	// time > cutoff has been committed (block times are monotonically increasing, so no
	// new blocks can then have time <= cutoff, and no new events can get a visibility_height
	// <= the resolved height).  Before this point the result could change between queries,
	// breaking cross-client determinism.  Returning empty matches bor's pre-fork behaviour
	// when heimdall is behind.
	cutoffUnix := request.ToTime.Unix()
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if sdkCtx.BlockTime().Unix() <= cutoffUnix {
		return &types.StateSyncsByTimeResponse{
			EventRecords: []types.EventRecord{},
		}, nil
	}

	height, err := q.k.GetBlockHeightByTime(ctx, cutoffUnix)
	if err != nil {
		if errors.Is(err, ErrNoBlockFound) {
			return &types.StateSyncsByTimeResponse{
				EventRecords: []types.EventRecord{},
			}, nil
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Use the shared helper to avoid double-counting metrics (the public
	// GetRecordListVisibleAtHeight handler has its own recordClerkQueryMetric).
	records, err := q.recordListVisibleAtHeight(ctx, request.FromId, height, request.ToTime, request.Pagination)
	if err != nil {
		return nil, err
	}

	return &types.StateSyncsByTimeResponse{
		EventRecords:   records,
		HeimdallHeight: height,
	}, nil
}

func isPaginationEmpty(p query.PageRequest) bool {
	return p.Key == nil &&
		p.Offset == 0 &&
		p.Limit == 0 &&
		!p.CountTotal &&
		!p.Reverse
}

func recordClerkQueryMetric(method string, start time.Time, err *error) {
	success := *err == nil
	api.RecordAPICallWithStart(api.ClerkSubsystem, method, api.QueryType, success, start)
}
