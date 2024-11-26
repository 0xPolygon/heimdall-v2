package keeper

import (
	"context"
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

// Keeper stores all related data
type Keeper struct {
	storeService storetypes.KVStoreService
	cdc          codec.BinaryCodec

	ChainKeeper    types.ChainKeeper
	contractCaller helper.IContractCaller

	Schema        collections.Schema
	RecordsWithID collections.Map[uint64, types.EventRecord]
	// TODO HV2 - is RecordsWithTime this needed? We can regenerate this from RecordsWithID
	RecordsWithTime collections.Map[collections.Pair[time.Time, uint64], uint64]
	RecordSequences collections.Map[string, []byte]
}

// NewKeeper create new keeper
func NewKeeper(
	cdc codec.BinaryCodec,
	storeService storetypes.KVStoreService,
	ChainKeeper types.ChainKeeper,
	contractCaller helper.IContractCaller,
) Keeper {
	sb := collections.NewSchemaBuilder(storeService)
	keeper := Keeper{
		storeService:    storeService,
		cdc:             cdc,
		ChainKeeper:     ChainKeeper,
		contractCaller:  contractCaller,
		RecordsWithID:   collections.NewMap(sb, types.RecordsWithIDKeyPrefix, "recordsWithID", collections.Uint64Key, codec.CollValue[types.EventRecord](cdc)),
		RecordsWithTime: collections.NewMap(sb, types.RecordsWithTimeKeyPrefix, "recordsWithTime", collections.PairKeyCodec(sdk.TimeKey, collections.Uint64Key), collections.Uint64Value),
		RecordSequences: collections.NewMap(sb, types.RecordSequencesKeyPrefix, "recordSequences", collections.StringKey, collections.BytesValue),
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

// SetEventRecordWithTime sets event record id with time
func (k *Keeper) SetEventRecordWithTime(ctx context.Context, record types.EventRecord) error {
	isPresent, _ := k.RecordsWithTime.Has(ctx, collections.Join(record.RecordTime, record.Id))
	if isPresent {
		return fmt.Errorf("record with time %s and id %d already exists", record.RecordTime, record.Id)
	}

	return k.RecordsWithTime.Set(ctx, collections.Join(record.RecordTime, record.Id), record.Id)
}

// SetEventRecordWithID adds record to store with ID
func (k *Keeper) SetEventRecordWithID(ctx context.Context, record types.EventRecord) error {
	if k.HasEventRecord(ctx, record.Id) {
		return fmt.Errorf("record with id %d already exists", record.Id)
	}

	return k.RecordsWithID.Set(ctx, record.Id, record)
}

// SetEventRecord adds record to store
func (k *Keeper) SetEventRecord(ctx context.Context, record types.EventRecord) error {
	if err := k.SetEventRecordWithID(ctx, record); err != nil {
		return err
	}

	return k.SetEventRecordWithTime(ctx, record)
}

// GetEventRecord returns record from store
func (k *Keeper) GetEventRecord(ctx context.Context, stateID uint64) (*types.EventRecord, error) {
	// check if record exists
	record, err := k.RecordsWithID.Get(ctx, stateID)
	if err != nil {
		return nil, err
	}

	return &record, nil
}

// HasEventRecord check if state record
func (k *Keeper) HasEventRecord(ctx context.Context, stateID uint64) bool {
	isPresent, _ := k.RecordsWithID.Has(ctx, stateID)

	return isPresent
}

// GetAllEventRecords get all state records
func (k *Keeper) GetAllEventRecords(ctx context.Context) []types.EventRecord {
	records, _ := k.IterateRecords(ctx)
	// iterate through state sync and append to list
	return records
}

// GetEventRecordList returns all records with params like page and limit
func (k *Keeper) GetEventRecordList(ctx context.Context, page uint64, limit uint64) ([]types.EventRecord, error) {
	// create records slice
	var records []types.EventRecord

	// set max limit
	if limit > 50 {
		limit = 50
	}

	startIndex := int((page - 1) * limit)
	endIndex := int(page * limit)

	// Initialize a counter to track the number of records processed
	counter := 0

	// Use Walk to iterate over the records
	err := k.RecordsWithID.Walk(ctx, nil, func(key uint64, record types.EventRecord) (bool, error) {
		// If the current index is within the desired range, add the record to the slice
		if counter >= startIndex && counter < endIndex {
			records = append(records, record)
		}

		// Increment the counter
		counter++

		// Stop walking if we've collected enough records
		if counter >= endIndex {
			return true, nil // Stop the walk
		}

		return false, nil // Continue walking
	})

	if err != nil {
		return nil, err
	}

	// Check if we have collected any records
	if len(records) == 0 && startIndex > 0 {
		return nil, fmt.Errorf("page %d does not exist", page)
	}

	return records, nil
}

// GetEventRecordListWithTime returns all records with params like fromTime and toTime
func (k *Keeper) GetEventRecordListWithTime(ctx context.Context, fromTime, toTime time.Time, page, limit uint64) ([]types.EventRecord, error) {
	// create records
	var records []types.EventRecord
	var allRecords = make([]types.EventRecord, 0)

	// have max limit
	if limit > 50 {
		limit = 50
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

	// loop through records to get valid records
	for _, stateID := range stateIDs {
		record, err := k.GetEventRecord(ctx, stateID)
		if err != nil {
			k.Logger(ctx).Error("error in fetching event record", "error", err)
			continue
		}
		allRecords = append(allRecords, *record)
	}

	if page == 0 && limit == 0 {
		return allRecords, nil
	}

	startIndex := int((page - 1) * limit)
	endIndex := int(page * limit)

	// Check if the startIndex is within bounds
	if startIndex >= len(allRecords) {
		return nil, fmt.Errorf("page %d does not exist", page)
	}

	// Check if the endIndex exceeds the length of eventRecords
	if endIndex > len(allRecords) {
		endIndex = len(allRecords)
	}

	// Retrieve the event records for the requested page
	records = allRecords[startIndex:endIndex]

	return records, nil
}

// IterateRecords iterates records and apply the given function
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

// GetRecordSequences checks if record already exists
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

	// loop through sequences
	for ; iterator.Valid(); iterator.Next() {
		sequence, err := iterator.Key()
		if err != nil {
			return
		}

		// call function and return if required
		if err := f(sequence); err != nil {
			return
		}
	}
}

// SetRecordSequence sets mapping for sequence id to bool
func (k *Keeper) SetRecordSequence(ctx context.Context, sequence string) {
	if sequence != "" {
		err := k.RecordSequences.Set(ctx, sequence, types.DefaultValue)
		if err != nil {
			k.Logger(ctx).Error("error in storing record sequence", "error", err)
		}
	}
}

// HasRecordSequence checks if record already exists
func (k *Keeper) HasRecordSequence(ctx context.Context, sequence string) bool {
	isPresent, err := k.RecordSequences.Has(ctx, sequence)

	if err != nil {
		return false
	}

	return isPresent
}
