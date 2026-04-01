package keeper

import (
	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/0xPolygon/heimdall-v2/x/clerk/types"
)

// InitGenesis sets clerk information for genesis.
func (k *Keeper) InitGenesis(ctx sdk.Context, data *types.GenesisState) {
	if len(data.EventRecords) != 0 {
		for _, record := range data.EventRecords {
			if err := k.SetEventRecord(ctx, record); err != nil {
				k.Logger(ctx).Error("Error in storing event record", "error", err)
			}
		}
	}

	for _, sequence := range data.RecordSequences {
		k.SetRecordSequence(ctx, sequence)
	}

	// Deterministic state sync fields.
	if data.VisibilityTimeUpgradeId > 0 {
		if err := k.SetVisibilityTimeUpgradeID(ctx, data.VisibilityTimeUpgradeId); err != nil {
			k.Logger(ctx).Error("Error setting visibility time upgrade ID", "error", err)
		}
	}

	for _, eventID := range data.PendingVisibilityEventIds {
		if err := k.AddPendingVisibilityEvent(ctx, eventID); err != nil {
			k.Logger(ctx).Error("Error adding pending visibility event", "id", eventID, "error", err)
		}
	}

	for _, entry := range data.VisibilityTimesById {
		if err := k.VisibilityTimeByID.Set(ctx, entry.Key, entry.Value); err != nil {
			k.Logger(ctx).Error("Error setting visibility time by ID", "id", entry.Key, "error", err)
		}
	}

	for _, entry := range data.VisibilityHeightsById {
		if err := k.VisibilityHeightByID.Set(ctx, entry.Key, entry.Value); err != nil {
			k.Logger(ctx).Error("Error setting visibility height by ID", "id", entry.Key, "error", err)
		}
	}

	for _, entry := range data.BlockTimeEntries {
		if err := k.BlockTimeReverseIndex.Set(ctx, collections.Join(entry.BlockTime, entry.Height), entry.Height); err != nil {
			k.Logger(ctx).Error("Error setting block time entry", "time", entry.BlockTime, "height", entry.Height, "error", err)
		}
	}
}

// ExportGenesis returns a GenesisState for a given context and keeper.
func (k *Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	gs := &types.GenesisState{
		EventRecords:    k.GetAllEventRecords(ctx),
		RecordSequences: k.GetRecordSequences(ctx),
	}

	// Export visibility time upgrade ID.
	if upgradeID, err := k.GetVisibilityTimeUpgradeID(ctx); err == nil {
		gs.VisibilityTimeUpgradeId = upgradeID
	}

	// Export pending visibility events.
	pendingIter, err := k.PendingVisibilityEvents.Iterate(ctx, nil)
	if err == nil {
		defer func(pendingIter collections.Iterator[uint64, []byte]) {
			err := pendingIter.Close()
			if err != nil {
				k.Logger(ctx).Error("Error closing pending visibility events iterator", "error", err)
			}
		}(pendingIter)
		for ; pendingIter.Valid(); pendingIter.Next() {
			if key, err := pendingIter.Key(); err == nil {
				gs.PendingVisibilityEventIds = append(gs.PendingVisibilityEventIds, key)
			}
		}
	}

	// Export visibility times by ID.
	vtIter, err := k.VisibilityTimeByID.Iterate(ctx, nil)
	if err == nil {
		defer func(vtIter collections.Iterator[uint64, uint64]) {
			err := vtIter.Close()
			if err != nil {
				k.Logger(ctx).Error("Error closing visibility time iterator", "error", err)
			}
		}(vtIter)
		for ; vtIter.Valid(); vtIter.Next() {
			if kv, err := vtIter.KeyValue(); err == nil {
				gs.VisibilityTimesById = append(gs.VisibilityTimesById, types.Uint64Pair{
					Key:   kv.Key,
					Value: kv.Value,
				})
			}
		}
	}

	// Export visibility heights by ID.
	vhIter, err := k.VisibilityHeightByID.Iterate(ctx, nil)
	if err == nil {
		defer vhIter.Close()
		for ; vhIter.Valid(); vhIter.Next() {
			if kv, err := vhIter.KeyValue(); err == nil {
				gs.VisibilityHeightsById = append(gs.VisibilityHeightsById, types.Uint64Pair{
					Key:   kv.Key,
					Value: kv.Value,
				})
			}
		}
	}

	// Export block time reverse index.
	btIter, err := k.BlockTimeReverseIndex.Iterate(ctx, nil)
	if err == nil {
		defer func(btIter collections.Iterator[collections.Pair[uint64, uint64], uint64]) {
			err := btIter.Close()
			if err != nil {
				k.Logger(ctx).Error("Error closing block time reverse", "error", err)
			}
		}(btIter)
		for ; btIter.Valid(); btIter.Next() {
			if kv, err := btIter.KeyValue(); err == nil {
				gs.BlockTimeEntries = append(gs.BlockTimeEntries, types.BlockTimeEntry{
					BlockTime: kv.Key.K1(),
					Height:    kv.Key.K2(),
				})
			}
		}
	}

	return gs
}
