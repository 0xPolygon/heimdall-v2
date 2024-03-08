package keeper

import (
	"context"

	"github.com/0xPolygon/heimdall-v2/x/clerk/types"
)

// query endpoints supported by the auth Querier
const (
	QueryRecord             = "record"
	QueryRecordList         = "record-list"
	QueryRecordListWithTime = "record-list-time"
	QueryRecordSequence     = "record-sequence"
)

type queryServer struct{ k Keeper }

var _ types.QueryServer = queryServer{}

func NewQueryServer(k Keeper) types.QueryServer {
	return queryServer{k: k}
}

func (s queryServer) NewQueryRecordParams(ctx context.Context, request *types.NewQueryRecordParamsRequest) (*types.QueryRecordParams, error) {
	return &types.QueryRecordParams{
		RecordID: request.RecordID,
	}, nil
}

func (s queryServer) NewQueryRecordSequenceParams(ctx context.Context, request *types.NewQueryRecordSequenceParamsRequest) (*types.QueryRecordSequenceParams, error) {
	return &types.QueryRecordSequenceParams{
		TxHash:   request.TxHash,
		LogIndex: request.LogIndex,
	}, nil
}

func (s queryServer) NewQueryTimeRangePaginationParams(ctx context.Context, request *types.NewQueryTimeRangePaginationParamsRequest) (*types.QueryRecordTimePaginationParams, error) {
	return &types.QueryRecordTimePaginationParams{
		FromTime: request.FromTime,
		ToTime:   request.ToTime,
		Page:     request.Page,
		Limit:    request.Limit,
	}, nil
}
