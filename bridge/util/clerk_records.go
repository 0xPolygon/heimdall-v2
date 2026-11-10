package util

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/cosmos/cosmos-sdk/codec"

	"github.com/0xPolygon/heimdall-v2/helper"
	clerktypes "github.com/0xPolygon/heimdall-v2/x/clerk/types"
)

// GetRecordIDsPage returns the IDs of the records heimdall stored on one page
// of its record list: page p of size limit covers IDs (p-1)*limit+1 .. p*limit.
// Heimdall answers a page above 1 with no records with an error; that answer
// is an empty page, not a failure.
func GetRecordIDsPage(ctx context.Context, cdc codec.Codec, page, limit uint64) ([]uint64, error) {
	response, err := helper.FetchFromAPIWithContext(ctx, helper.GetHeimdallServerEndpoint(fmt.Sprintf(ClerkRecordListURL, page, limit)))
	var statusErr *helper.APIStatusError
	if errors.As(err, &statusErr) && strings.Contains(statusErr.Body, fmt.Sprintf("page %d does not exist", page)) {
		return []uint64{}, nil
	}
	if err != nil {
		return nil, err
	}

	var list clerktypes.RecordListResponse
	if err = cdc.UnmarshalJSON(response, &list); err != nil {
		return nil, err
	}
	ids := make([]uint64, 0, len(list.EventRecords))
	for _, record := range list.EventRecords {
		ids = append(ids, record.Id)
	}
	return ids, nil
}

// recordNotFound is in the body of heimdall's answer to a query for a record it
// doesn't hold.
const recordNotFound = "collections: not found"

// ClerkRecordStored reports whether heimdall stored the record with id. Only a
// 404 or heimdall's not-found answer means missing; any other failure, or an
// answer about another record, is an error.
func ClerkRecordStored(ctx context.Context, cdc codec.Codec, id uint64) (bool, error) {
	response, err := helper.FetchFromAPIWithContext(ctx, helper.GetHeimdallServerEndpoint(fmt.Sprintf(ClerkEventRecordURL, id)))
	var statusErr *helper.APIStatusError
	if errors.As(err, &statusErr) && (statusErr.Status == http.StatusNotFound || strings.Contains(statusErr.Body, recordNotFound)) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	var record clerktypes.RecordResponse
	if err = cdc.UnmarshalJSON(response, &record); err != nil {
		return false, err
	}
	if record.Record.Id != id {
		return false, fmt.Errorf("record query for %d answered with record %d", id, record.Record.Id)
	}
	return true, nil
}

// GetLatestRecordID returns the latest state sync ID on L1, as heimdall reads
// it from the StateSender contract.
func GetLatestRecordID(cdc codec.Codec) (uint64, error) {
	response, err := helper.FetchFromAPI(helper.GetHeimdallServerEndpoint(ClerkLatestRecordIDURL))
	if err != nil {
		return 0, err
	}

	var latest clerktypes.LatestRecordIdResponse
	if err = cdc.UnmarshalJSON(response, &latest); err != nil {
		return 0, err
	}
	return latest.LatestRecordId, nil
}

// GetRecordCount returns how many state sync records heimdall stored.
func GetRecordCount(ctx context.Context, cdc codec.Codec) (uint64, error) {
	response, err := helper.FetchFromAPIWithContext(ctx, helper.GetHeimdallServerEndpoint(ClerkRecordCountURL))
	if err != nil {
		return 0, err
	}

	var count clerktypes.RecordCountResponse
	if err = cdc.UnmarshalJSON(response, &count); err != nil {
		return 0, err
	}
	return count.Count, nil
}
