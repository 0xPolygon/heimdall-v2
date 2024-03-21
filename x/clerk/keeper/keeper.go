package keeper

import (
	"context"
	"errors"
	"strconv"
	"time"

	storetypes "cosmossdk.io/core/store"
	"cosmossdk.io/log"
	storetypes2 "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	// TODO HV2: uncomment when chainmanager is implemented
	// "github.com/0xPolygon/heimdall-v2/chainmanager"

	"github.com/0xPolygon/heimdall-v2/x/clerk/types"
)

var (
	StateRecordPrefixKey = []byte{0x11} // prefix key for when storing state

	// DefaultValue default value
	DefaultValue = []byte{0x01}

	// RecordSequencePrefixKey represents record sequence prefix key
	RecordSequencePrefixKey = []byte{0x12}

	StateRecordPrefixKeyWithTime = []byte{0x13} // prefix key for when storing state with time
)

// Keeper stores all related data
type Keeper struct {
	storeService storetypes.KVStoreService
	cdc          codec.BinaryCodec
	// chain param keeper
	// TODO HV2: uncomment when chainmanager is implemented
	// chainKeeper chainmanager.Keeper
}

// NewKeeper create new keeper
func NewKeeper(
	cdc codec.BinaryCodec,
	storeService storetypes.KVStoreService,
	// TODO HV2: uncomment when chainmanager is implemented
	// chainKeeper chainmanager.Keeper,
) Keeper {
	keeper := Keeper{
		storeService: storeService,
		cdc:          cdc,
		// TODO HV2: uncomment when chainmanager is implemented
		// chainKeeper: chainKeeper,
	}

	return keeper
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx context.Context) log.Logger {
	return sdk.UnwrapSDKContext(ctx).Logger().With("module", "x/"+types.ModuleName)
}

// SetEventRecordWithTime sets event record id with time
func (k *Keeper) SetEventRecordWithTime(ctx sdk.Context, record types.EventRecord) error {
	key := GetEventRecordKeyWithTime(record.ID, record.RecordTime)

	// TODO HV2: why is `record.ID` not accepted in the function?
	// value, err := k.cdc.MarshalInterface(record.ID)
	value, err := k.cdc.MarshalInterface(nil)
	if err != nil {
		k.Logger(ctx).Error("Error marshalling record", "error", err)
		return err
	}

	return k.setEventRecordStore(ctx, key, value)
}

// SetEventRecordWithID adds record to store with ID
func (k *Keeper) SetEventRecordWithID(ctx sdk.Context, record types.EventRecord) error {
	key := GetEventRecordKey(record.ID)

	value, err := k.cdc.MarshalInterface(&record)
	if err != nil {
		k.Logger(ctx).Error("Error marshalling record", "error", err)
		return err
	}

	return k.setEventRecordStore(ctx, key, value)
}

// setEventRecordStore adds value to store by key
func (k *Keeper) setEventRecordStore(ctx sdk.Context, key, value []byte) error {
	kvStore := k.storeService.OpenKVStore(ctx)
	// check if already set
	isPresent, err := kvStore.Has(key)
	if isPresent {
		return errors.New("key already exists")
	}

	if err != nil {
		k.Logger(ctx).Error("Error checking record", "error", err)
		return err
	}

	// store value in provided key
	err = kvStore.Set(key, value)
	if err != nil {
		k.Logger(ctx).Error("Error setting record", "error", err)
		return err
	}

	// return
	return nil
}

// SetEventRecord adds record to store
func (k *Keeper) SetEventRecord(ctx sdk.Context, record types.EventRecord) error {
	if err := k.SetEventRecordWithID(ctx, record); err != nil {
		return err
	}

	return k.SetEventRecordWithTime(ctx, record)
}

// GetEventRecord returns record from store
func (k *Keeper) GetEventRecord(ctx sdk.Context, stateID uint64) (*types.EventRecord, error) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := GetEventRecordKey(stateID)

	// check store has data
	isPresent, _ := kvStore.Has(key)
	if isPresent {
		var _record types.EventRecord
		value, _ := kvStore.Get(key)
		if err := k.cdc.UnmarshalInterface(value, &_record); err != nil {
			return nil, err
		}

		return &_record, nil
	}

	// return no error found
	return nil, errors.New("no record found")
}

// HasEventRecord check if state record
func (k *Keeper) HasEventRecord(ctx context.Context, stateID uint64) bool {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := GetEventRecordKey(stateID)

	isPresent, _ := kvStore.Has(key)

	return isPresent
}

// GetAllEventRecords get all state records
func (k *Keeper) GetAllEventRecords(ctx sdk.Context) (records []*types.EventRecord) {
	// iterate through spans and create span update array
	k.IterateRecordsAndApplyFn(ctx, func(record types.EventRecord) error {
		// append to list of validatorUpdates
		records = append(records, &record)
		return nil
	})

	return
}

// GetEventRecordList returns all records with params like page and limit
func (k *Keeper) GetEventRecordList(ctx sdk.Context, page uint64, limit uint64) ([]types.EventRecord, error) {
	// kvStore := k.storeService.OpenKVStore(ctx)

	// create records
	var records []types.EventRecord

	// have max limit
	if limit > 50 {
		limit = 50
	}

	// get paginated iterator
	// TODO HV2 - figure out why kvStore (defined in first line of this function) is not accepted in the function
	// iterator := storetypes2.KVStorePrefixIteratorPaginated(kvStore, StateRecordPrefixKey, uint(page), uint(limit))
	iterator := storetypes2.KVStorePrefixIteratorPaginated(nil, StateRecordPrefixKey, uint(page), uint(limit))

	// loop through records to get valid records
	for ; iterator.Valid(); iterator.Next() {
		var record types.EventRecord
		if err := k.cdc.UnmarshalInterface(iterator.Value(), &record); err == nil {
			records = append(records, record)
		}
	}

	return records, nil
}

// GetEventRecordListWithTime returns all records with params like fromTime and toTime
func (k *Keeper) GetEventRecordListWithTime(ctx sdk.Context, fromTime, toTime time.Time, page, limit uint64) ([]types.EventRecord, error) {
	var iterator storetypes.Iterator

	kvStore := k.storeService.OpenKVStore(ctx)

	// create records
	var records []types.EventRecord

	iterator, _ = kvStore.Iterator(GetEventRecordKeyWithTimePrefix(fromTime), GetEventRecordKeyWithTimePrefix(toTime))

	// get range iterator
	defer iterator.Close()
	// loop through records to get valid records
	for ; iterator.Valid(); iterator.Next() {
		var stateID uint64
		if err := k.cdc.UnmarshalInterface(iterator.Value(), &stateID); err == nil {
			record, err := k.GetEventRecord(ctx, stateID)
			if err != nil {
				k.Logger(ctx).Error("GetEventRecordListWithTime | GetEventRecord", "error", err)
				continue
			}

			records = append(records, *record)
		}
	}

	return records, nil
}

// GetEventRecordKey appends prefix to state id
func GetEventRecordKey(stateID uint64) []byte {
	stateIDBytes := []byte(strconv.FormatUint(stateID, 10))
	return append(StateRecordPrefixKey, stateIDBytes...)
}

// GetEventRecordKeyWithTime appends prefix to state id and record time
func GetEventRecordKeyWithTime(stateID uint64, recordTime time.Time) []byte {
	stateIDBytes := []byte(strconv.FormatUint(stateID, 10))
	return append(GetEventRecordKeyWithTimePrefix(recordTime), stateIDBytes...)
}

// GetEventRecordKeyWithTimePrefix gives prefix for record time key
func GetEventRecordKeyWithTimePrefix(recordTime time.Time) []byte {
	recordTimeBytes := sdk.FormatTimeBytes(recordTime)
	return append(StateRecordPrefixKeyWithTime, recordTimeBytes...)
}

// GetRecordSequenceKey returns record sequence key
func GetRecordSequenceKey(sequence string) []byte {
	return append(RecordSequencePrefixKey, []byte(sequence)...)
}

// IterateRecordsAndApplyFn iterate records and apply the given function.
func (k *Keeper) IterateRecordsAndApplyFn(ctx sdk.Context, f func(record types.EventRecord) error) {
	// kvStore := k.storeService.OpenKVStore(ctx)

	// get span iterator
	// TODO HV2 - figure out why kvStore (defined in first line of this function) is not accepted in the function
	// iterator := storetypes2.KVStorePrefixIterator(kvStore, StateRecordPrefixKey)
	iterator := storetypes2.KVStorePrefixIterator(nil, StateRecordPrefixKey)
	defer iterator.Close()

	// loop through spans to get valid spans
	for ; iterator.Valid(); iterator.Next() {
		// unmarshall span
		var result types.EventRecord
		if err := k.cdc.UnmarshalInterface(iterator.Value(), &result); err != nil {
			k.Logger(ctx).Error("IterateRecordsAndApplyFn | UnmarshalInterface", "error", err)
			return
		}
		// call function and return if required
		if err := f(result); err != nil {
			return
		}
	}
}

// GetRecordSequences checks if record already exists
func (k *Keeper) GetRecordSequences(ctx sdk.Context) (sequences []string) {
	k.IterateRecordSequencesAndApplyFn(ctx, func(sequence string) error {
		sequences = append(sequences, sequence)
		return nil
	})

	return
}

// IterateRecordSequencesAndApplyFn iterate records and apply the given function.
func (k *Keeper) IterateRecordSequencesAndApplyFn(ctx sdk.Context, f func(sequence string) error) {
	// kvStore = k.storeService.OpenKVStore(ctx)

	// get sequence iterator
	// TODO HV2 - figure out why kvStore (defined in first line of this function) is not accepted in the function
	// iterator := storetypes2.KVStorePrefixIterator(kvStore, RecordSequencePrefixKey)
	iterator := storetypes2.KVStorePrefixIterator(nil, RecordSequencePrefixKey)
	defer iterator.Close()

	// loop through sequences
	for ; iterator.Valid(); iterator.Next() {
		sequence := string(iterator.Key()[len(RecordSequencePrefixKey):])

		// call function and return if required
		if err := f(sequence); err != nil {
			return
		}
	}
}

// SetRecordSequence sets mapping for sequence id to bool
func (k *Keeper) SetRecordSequence(ctx sdk.Context, sequence string) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := GetRecordSequenceKey(sequence)
	if key != nil {
		kvStore.Set(GetRecordSequenceKey(sequence), DefaultValue)
	}
}

// HasRecordSequence checks if record already exists
func (k *Keeper) HasRecordSequence(ctx context.Context, sequence string) bool {
	kvStore := k.storeService.OpenKVStore(ctx)
	isPresent, _ := kvStore.Has(GetRecordSequenceKey(sequence))
	return isPresent
}
