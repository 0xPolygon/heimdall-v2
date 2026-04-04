package keeper

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cosmossdk.io/collections"
	storetypes "cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/x/clerk/types"
)

// ErrNoBlockFound is returned when no committed block exists at or before the requested cutoff time.
var ErrNoBlockFound = errors.New("no block found before cutoff")

// Keeper stores all the related data.
type Keeper struct {
	storeService storetypes.KVStoreService
	cdc          codec.BinaryCodec

	ChainKeeper    types.ChainKeeper
	contractCaller helper.IContractCaller

	Schema                  collections.Schema
	RecordsWithID           collections.Map[uint64, types.EventRecord]
	RecordsWithTime         collections.Map[collections.Pair[time.Time, uint64], uint64]
	RecordSequences         collections.Map[string, []byte]
	VisibilityTimeUpgradeID collections.Item[uint64]
	PendingVisibilityEvents collections.Map[uint64, []byte]
	BlockTimeReverseIndex collections.Map[collections.Pair[uint64, uint64], uint64] // (blockTime, height) → height for O(log N) cutoff lookup
	VisibilityHeightByID    collections.Map[uint64, uint64]                           // event_id → heimdall block height where visibility was assigned
}

// NewKeeper creates a new keeper.
func NewKeeper(
	cdc codec.BinaryCodec,
	storeService storetypes.KVStoreService,
	ChainKeeper types.ChainKeeper,
	contractCaller helper.IContractCaller,
) Keeper {
	sb := collections.NewSchemaBuilder(storeService)
	keeper := Keeper{
		storeService:            storeService,
		cdc:                     cdc,
		ChainKeeper:             ChainKeeper,
		contractCaller:          contractCaller,
		RecordsWithID:           collections.NewMap(sb, types.RecordsWithIDKeyPrefix, "recordsWithID", collections.Uint64Key, codec.CollValue[types.EventRecord](cdc)),
		RecordsWithTime:         collections.NewMap(sb, types.RecordsWithTimeKeyPrefix, "recordsWithTime", collections.PairKeyCodec(sdk.TimeKey, collections.Uint64Key), collections.Uint64Value),
		RecordSequences:         collections.NewMap(sb, types.RecordSequencesKeyPrefix, "recordSequences", collections.StringKey, collections.BytesValue),
		VisibilityTimeUpgradeID: collections.NewItem(sb, types.VisibilityTimeUpgradeIDKeyPrefix, "visibilityTimeUpgradeID", collections.Uint64Value),
		PendingVisibilityEvents: collections.NewMap(sb, types.PendingVisibilityEventsKeyPrefix, "pendingVisibilityEvents", collections.Uint64Key, collections.BytesValue),
		BlockTimeReverseIndex: collections.NewMap(sb, types.BlockTimeReverseIndexKeyPrefix, "blockTimeReverseIndex", collections.PairKeyCodec(collections.Uint64Key, collections.Uint64Key), collections.Uint64Value),
		VisibilityHeightByID:    collections.NewMap(sb, types.VisibilityHeightByIDKeyPrefix, "visibilityHeightByID", collections.Uint64Key, collections.Uint64Value),
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}

	keeper.Schema = schema

	return keeper
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx context.Context) log.Logger {
	return sdk.UnwrapSDKContext(ctx).Logger().With("module", "x/"+types.ModuleName)
}

// SetEventRecordWithTime sets event record id with time.
func (k *Keeper) SetEventRecordWithTime(ctx context.Context, record types.EventRecord) error {
	isPresent, _ := k.RecordsWithTime.Has(ctx, collections.Join(record.RecordTime, record.Id))
	if isPresent {
		return fmt.Errorf("record with time %s and id %d already exists", record.RecordTime, record.Id)
	}

	return k.RecordsWithTime.Set(ctx, collections.Join(record.RecordTime, record.Id), record.Id)
}

// SetEventRecordWithID adds record to store with ID.
func (k *Keeper) SetEventRecordWithID(ctx context.Context, record types.EventRecord) error {
	if k.HasEventRecord(ctx, record.Id) {
		return fmt.Errorf("record with id %d already exists", record.Id)
	}

	return k.RecordsWithID.Set(ctx, record.Id, record)
}

// SetEventRecord adds record to store.
func (k *Keeper) SetEventRecord(ctx context.Context, record types.EventRecord) error {
	if err := k.SetEventRecordWithID(ctx, record); err != nil {
		return err
	}

	return k.SetEventRecordWithTime(ctx, record)
}

// GetEventRecord returns record from the store.
func (k *Keeper) GetEventRecord(ctx context.Context, stateID uint64) (*types.EventRecord, error) {
	// Check if the record exists.
	record, err := k.RecordsWithID.Get(ctx, stateID)
	if err != nil {
		return nil, err
	}

	return &record, nil
}

// HasEventRecord checks if state record.
func (k *Keeper) HasEventRecord(ctx context.Context, stateID uint64) bool {
	isPresent, _ := k.RecordsWithID.Has(ctx, stateID)

	return isPresent
}

// GetAllEventRecords gets all state records.
func (k *Keeper) GetAllEventRecords(ctx context.Context) []types.EventRecord {
	records, _ := k.IterateRecords(ctx)
	// Iterate through state sync and append to list.
	return records
}

// GetEventRecordList returns all records with params like page and limit.
func (k *Keeper) GetEventRecordList(ctx context.Context, page, limit uint64) ([]types.EventRecord, error) {
	// Create the records' slice.
	var records []types.EventRecord

	if page == 0 {
		return nil, fmt.Errorf("page cannot be 0")
	}
	if limit == 0 || limit > MaxRecordListLimit {
		return nil, fmt.Errorf("limit cannot be 0 or greater than %d", MaxRecordListLimit)
	}

	// Calculate the starting record ID based on page and limit.
	startRecordID := (page-1)*limit + 1
	endRecordID := page*limit + 1

	// Use Range to efficiently query only the records we need.
	rng := new(collections.Range[uint64]).
		StartInclusive(startRecordID).
		EndExclusive(endRecordID)

	iterator, err := k.RecordsWithID.Iterate(ctx, rng)
	if err != nil {
		return nil, err
	}
	defer func(iterator collections.Iterator[uint64, types.EventRecord]) {
		err := iterator.Close()
		if err != nil {
			k.Logger(ctx).Error("Error closing iterator", "error", err)
		}
	}(iterator)

	// Collect the records from the iterator.
	records, err = iterator.Values()
	if err != nil {
		return nil, err
	}

	// Check if we have collected any records.
	if len(records) == 0 && page > 1 {
		return nil, fmt.Errorf("page %d does not exist", page)
	}

	return records, nil
}

// GetEventRecordListWithTime returns all records with params like fromTime and toTime.
func (k *Keeper) GetEventRecordListWithTime(ctx context.Context, fromTime, toTime time.Time, page, limit uint64) ([]types.EventRecord, error) {
	// Create the records' slice.
	var records []types.EventRecord

	if page == 0 {
		return nil, fmt.Errorf("page cannot be 0")
	}
	if limit == 0 || limit > MaxRecordListLimit {
		return nil, fmt.Errorf("limit cannot be 0 or greater than %d", MaxRecordListLimit)
	}

	rng := new(collections.Range[collections.Pair[time.Time, uint64]]).
		StartInclusive(collections.Join(fromTime, uint64(0))).
		EndExclusive(collections.Join(toTime, uint64(0)))

	iterator, err := k.RecordsWithTime.Iterate(ctx, rng)
	if err != nil {
		return records, err
	}

	stateIDs, err := iterator.Values()
	if err != nil {
		return records, err
	}

	allRecords := make([]types.EventRecord, 0, len(stateIDs))

	// Loop through records to get valid records.
	for _, stateID := range stateIDs {
		record, err := k.GetEventRecord(ctx, stateID)
		if err != nil {
			k.Logger(ctx).Error("Error in fetching event record", "error", err)
			continue
		}
		allRecords = append(allRecords, *record)
	}

	startIndex := int((page - 1) * limit)
	endIndex := int(page * limit)

	// Check if the startIndex is within bounds.
	if startIndex >= len(allRecords) {
		return nil, fmt.Errorf("page %d does not exist", page)
	}

	// Check if the endIndex exceeds the length of eventRecords.
	if endIndex > len(allRecords) {
		endIndex = len(allRecords)
	}

	// Retrieve the event records for the requested page.
	records = allRecords[startIndex:endIndex]

	return records, nil
}

// IterateRecords iterates records and applies the given function.
func (k *Keeper) IterateRecords(ctx context.Context) ([]types.EventRecord, error) {
	iterator, err := k.RecordsWithID.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}

	records, err := iterator.Values()
	if err != nil {
		return nil, err
	}

	return records, nil
}

// GetRecordSequences checks if the record already exists.
func (k *Keeper) GetRecordSequences(ctx context.Context) (sequences []string) {
	k.IterateRecordSequencesAndApplyFn(ctx, func(sequence string) error {
		sequences = append(sequences, sequence)
		return nil
	})

	return
}

// IterateRecordSequencesAndApplyFn iterate records and apply the given function.
func (k *Keeper) IterateRecordSequencesAndApplyFn(ctx context.Context, f func(sequence string) error) {
	iterator, err := k.RecordSequences.Iterate(ctx, nil)
	if err != nil {
		return
	}

	// Loop through sequences.
	for ; iterator.Valid(); iterator.Next() {
		sequence, err := iterator.Key()
		if err != nil {
			return
		}

		// Call the function and return if required.
		if err := f(sequence); err != nil {
			return
		}
	}
}

// SetRecordSequence sets mapping for the sequence id to bool.
func (k *Keeper) SetRecordSequence(ctx context.Context, sequence string) {
	if sequence != "" {
		err := k.RecordSequences.Set(ctx, sequence, types.DefaultValue)
		if err != nil {
			k.Logger(ctx).Error("Error in storing record sequence", "error", err)
		}
	}
}

// HasRecordSequence checks if the record already exists.
func (k *Keeper) HasRecordSequence(ctx context.Context, sequence string) bool {
	isPresent, err := k.RecordSequences.Has(ctx, sequence)
	if err != nil {
		return false
	}

	return isPresent
}

// GetVisibilityHeightForEvent returns the visibility_height for the given event ID.
func (k *Keeper) GetVisibilityHeightForEvent(ctx context.Context, eventID uint64) (uint64, error) {
	return k.VisibilityHeightByID.Get(ctx, eventID)
}

// AddPendingVisibilityEvent adds an event ID to the pending visibility events list.
func (k *Keeper) AddPendingVisibilityEvent(ctx context.Context, eventID uint64) error {
	return k.PendingVisibilityEvents.Set(ctx, eventID, types.DefaultValue)
}

// ProcessPendingVisibilityEvents assigns visibility_time, visibility_height to events
// from the previous block, then clears the pending list.
// Pending events remain excluded from the height-pinned / visibility_height-based
// query path until this runs, ensuring deterministic results there: during halts,
// no new blocks arrive, so pending events stay excluded from that query path.
// Legacy time-based queries such as GetRecordListWithTime still use record_time
// and may return pending events before this processing occurs.
func (k *Keeper) ProcessPendingVisibilityEvents(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockHeight := uint64(sdkCtx.BlockHeight())

	iterator, err := k.PendingVisibilityEvents.Iterate(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to iterate pending visibility events: %w", err)
	}
	defer func() {
		if err := iterator.Close(); err != nil {
			k.Logger(ctx).Error("failed to close iterator in ProcessPendingVisibilityEvents", "error", err)
		}
	}()

	var eventIDs []uint64
	for ; iterator.Valid(); iterator.Next() {
		eventID, err := iterator.Key()
		if err != nil {
			return fmt.Errorf("failed to get pending event key: %w", err)
		}
		eventIDs = append(eventIDs, eventID)
	}

	for _, eventID := range eventIDs {
		if err := k.VisibilityHeightByID.Set(ctx, eventID, blockHeight); err != nil {
			return fmt.Errorf("failed to set visibility height for event %d: %w", eventID, err)
		}
		if err := k.PendingVisibilityEvents.Remove(ctx, eventID); err != nil {
			return fmt.Errorf("failed to remove pending event %d: %w", eventID, err)
		}
	}

	return nil
}

// GetVisibilityTimeUpgradeID returns the first event ID that uses visibility_time filtering.
func (k *Keeper) GetVisibilityTimeUpgradeID(ctx context.Context) (uint64, error) {
	return k.VisibilityTimeUpgradeID.Get(ctx)
}

// SetVisibilityTimeUpgradeID sets the first event ID that uses visibility_time filtering.
func (k *Keeper) SetVisibilityTimeUpgradeID(ctx context.Context, id uint64) error {
	return k.VisibilityTimeUpgradeID.Set(ctx, id)
}

// StoreBlockTime stores the current block's (blockTime, height) → height mapping in the
// reverse index, enabling O(log N) cutoff lookups via GetBlockHeightByTime.
// Called in PreBlocker for each block from the visibility_time activation height onward.
func (k *Keeper) StoreBlockTime(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	height := uint64(sdkCtx.BlockHeight())
	blockTime := uint64(sdkCtx.BlockTime().Unix())

	return k.BlockTimeReverseIndex.Set(ctx, collections.Join(blockTime, height), height)
}

// GetBlockHeightByTime returns the greatest committed Heimdall height H such that
// header.time(H) <= cutoff. If multiple heights share the same timestamp, the greatest
// height wins.
func (k *Keeper) GetBlockHeightByTime(ctx context.Context, cutoffUnix int64) (int64, error) {
	if cutoffUnix <= 0 {
		return 0, fmt.Errorf("cutoff time must be positive, got %d", cutoffUnix)
	}
	cutoff := uint64(cutoffUnix)

	rng := new(collections.Range[collections.Pair[uint64, uint64]]).
		EndInclusive(collections.Join(cutoff, ^uint64(0))).
		Descending()

	iterator, err := k.BlockTimeReverseIndex.Iterate(ctx, rng)
	if err != nil {
		return 0, fmt.Errorf("failed to create iterator for BlockTimeReverseIndex: %w", err)
	}
	defer func() {
		if err := iterator.Close(); err != nil {
			k.Logger(ctx).Error("failed to close iterator for BlockTimeReverseIndex", "error", err)
		}
	}()

	if !iterator.Valid() {
		return 0, fmt.Errorf("%w: time <= %d", ErrNoBlockFound, cutoffUnix)
	}

	kv, err := iterator.KeyValue()
	if err != nil {
		return 0, fmt.Errorf("failed to read from BlockTimeReverseIndex: %w", err)
	}

	return int64(kv.Value), nil
}

// GetEventRecordCount returns the total count of event records.
func (k *Keeper) GetEventRecordCount(ctx context.Context) uint64 {
	// Create a reverse iterator to get the highest key efficiently.
	iterator, err := k.RecordsWithID.Iterate(ctx, (&collections.Range[uint64]{}).Descending())
	if err != nil {
		k.Logger(ctx).Error("Failed to create reverse iterator for counting records", "error", err)
		return 0
	}
	defer func(iterator collections.Iterator[uint64, types.EventRecord]) {
		err := iterator.Close()
		if err != nil {
			k.Logger(ctx).Error("Failed to close reverse iterator for counting records", "error", err)
		}
	}(iterator)

	// Get the first (highest) key from the reverse iterator.
	if !iterator.Valid() {
		return 0 // No records exist.
	}

	highestKey, err := iterator.Key()
	if err != nil {
		k.Logger(ctx).Error("Failed to get highest key for counting records", "error", err)
		return 0
	}

	// Since record IDs are sequential starting from 1, the highest key equals the count.
	return highestKey
}
