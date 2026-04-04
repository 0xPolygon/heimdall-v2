package keeper

import (
	"errors"

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
	logger := k.Logger(ctx)

	gs := &types.GenesisState{
		EventRecords:    k.GetAllEventRecords(ctx),
		RecordSequences: k.GetRecordSequences(ctx),
	}

	// Export visibility time upgrade ID.
	upgradeID, err := k.GetVisibilityTimeUpgradeID(ctx)
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			logger.Error("Error exporting visibility time upgrade ID", "error", err)
		}
	} else {
		gs.VisibilityTimeUpgradeId = upgradeID
	}

	// Export pending visibility events.
	pendingIter, err := k.PendingVisibilityEvents.Iterate(ctx, nil)
	if err != nil {
		logger.Error("Error creating pending visibility events iterator for export", "error", err)
	} else {
		defer func() {
			if err := pendingIter.Close(); err != nil {
				logger.Error("Error closing pending visibility events iterator", "error", err)
			}
		}()
		for ; pendingIter.Valid(); pendingIter.Next() {
			key, err := pendingIter.Key()
			if err != nil {
				logger.Error("Error reading pending visibility event key during export", "error", err)
				continue
			}
			gs.PendingVisibilityEventIds = append(gs.PendingVisibilityEventIds, key)
		}
	}

	// Export visibility heights by ID.
	vhIter, err := k.VisibilityHeightByID.Iterate(ctx, nil)
	if err != nil {
		logger.Error("Error creating visibility height iterator for export", "error", err)
	} else {
		defer func() {
			if err := vhIter.Close(); err != nil {
				logger.Error("Error closing visibility height iterator", "error", err)
			}
		}()
		for ; vhIter.Valid(); vhIter.Next() {
			kv, err := vhIter.KeyValue()
			if err != nil {
				logger.Error("Error reading visibility height entry during export", "error", err)
				continue
			}
			gs.VisibilityHeightsById = append(gs.VisibilityHeightsById, types.Uint64Pair{
				Key:   kv.Key,
				Value: kv.Value,
			})
		}
	}

	// Export block time reverse index.
	btIter, err := k.BlockTimeReverseIndex.Iterate(ctx, nil)
	if err != nil {
		logger.Error("Error creating block time reverse index iterator for export", "error", err)
	} else {
		defer func() {
			if err := btIter.Close(); err != nil {
				logger.Error("Error closing block time reverse index iterator", "error", err)
			}
		}()
		for ; btIter.Valid(); btIter.Next() {
			kv, err := btIter.KeyValue()
			if err != nil {
				logger.Error("Error reading block time entry during export", "error", err)
				continue
			}
			gs.BlockTimeEntries = append(gs.BlockTimeEntries, types.BlockTimeEntry{
				BlockTime: kv.Key.K1(),
				Height:    kv.Key.K2(),
			})
		}
	}

	return gs
}
