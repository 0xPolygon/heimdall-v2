package keeper

import (
	"context"

	"github.com/0xPolygon/heimdall-v2/x/clerk/types"
)

// query endpoints supported by the auth querier
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

func (s queryServer) Record(ctx context.Context, request *types.RecordRequest) (*types.RecordResponse, error) {
	record, err := s.k.GetEventRecord(ctx, request.RecordID)
	if err != nil {
		return nil, err
	}

	return &types.RecordResponse{Record: record}, nil
}

func (s queryServer) RecordList(ctx context.Context, request *types.RecordListRequest) (*types.RecordListResponse, error) {
	records, err := s.k.GetEventRecordList(ctx, request.Page, request.Limit)
	if err != nil {
		return nil, err
	}

	newRecords := make([]*types.EventRecord, len(records))
	for i, record := range records {
		newRecords[i] = &record
	}

	return &types.RecordListResponse{EventRecords: newRecords}, nil
}

func (s queryServer) RecordListWithTime(ctx context.Context, request *types.RecordListWithTimeRequest) (*types.RecordListWithTimeResponse, error) {
	records, err := s.k.GetEventRecordListWithTime(ctx, request.FromTime, request.ToTime, request.Page, request.Limit)
	if err != nil {
		return nil, err
	}

	newRecords := make([]*types.EventRecord, len(records))
	for i, record := range records {
		newRecords[i] = &record
	}

	return &types.RecordListWithTimeResponse{EventRecords: newRecords}, nil
}

func (s queryServer) RecordSequence(ctx context.Context, request *types.RecordSequenceRequest) (*types.RecordSequenceResponse, error) {
	// TODO HV2 - implement after contractCallerObj is available
	/*
		var params types.QueryRecordSequenceParams
		if err := types.ModuleCdc.UnmarshalJSON(req.Data, &params); err != nil {
			return nil, sdk.ErrInternal(fmt.Sprintf("failed to parse params: %s", err))
		}
		chainParams := keeper.chainKeeper.GetParams(ctx)
		// get main tx receipt
		receipt, err := contractCallerObj.GetConfirmedTxReceipt(hmTypes.HexToHeimdallHash(params.TxHash).EthHash(), chainParams.MainchainTxConfirmations)
		if err != nil || receipt == nil {
			return nil, sdk.ErrInternal("Transaction is not confirmed yet. Please wait for sometime and try again")
		}
		// sequence id
		sequence := new(big.Int).Mul(receipt.BlockNumber, big.NewInt(hmTypes.DefaultLogIndexUnit))
		sequence.Add(sequence, new(big.Int).SetUint64(params.LogIndex))
		// check if incoming tx already exists
		if !keeper.HasRecordSequence(ctx, sequence.String()) {
			return nil, nil
		}
		bz, err := codec.MarshalJSONIndent(types.ModuleCdc, sequence)
		if err != nil {
			return nil, sdk.ErrInternal(sdk.AppendMsgToErr("could not marshal result to JSON", err.Error()))
		}
		return bz, nil
	*/

	return &types.RecordSequenceResponse{}, nil
}
