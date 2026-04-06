package keeper_test

import (
	"context"
	"time"

	"cosmossdk.io/collections"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/sidetxs"
	clerkKeeper "github.com/0xPolygon/heimdall-v2/x/clerk/keeper"
	"github.com/0xPolygon/heimdall-v2/x/clerk/types"
)

func (s *KeeperTestSuite) queryDeterministicStateSyncs(
	ctx sdk.Context,
	resolvedHeight int64,
	fromID uint64,
	toTime time.Time,
	limit uint64,
) (*types.StateSyncsByTimeResponse, error) {
	storeCtx := ctx.WithBlockHeight(resolvedHeight).WithBlockHeader(cmtproto.Header{
		Time:   toTime.Add(-time.Second),
		Height: resolvedHeight,
	})
	if err := s.keeper.StoreBlockTime(storeCtx); err != nil {
		return nil, err
	}

	queryCtx := ctx.WithBlockHeight(resolvedHeight + 1).WithBlockHeader(cmtproto.Header{
		Time:   toTime.Add(time.Second),
		Height: resolvedHeight + 1,
	})
	if err := s.keeper.StoreBlockTime(queryCtx); err != nil {
		return nil, err
	}

	queryServer := clerkKeeper.NewQueryServer(&s.keeper)

	return queryServer.(interface {
		GetStateSyncsByTime(context.Context, *types.StateSyncsByTimeRequest) (*types.StateSyncsByTimeResponse, error)
	}).GetStateSyncsByTime(queryCtx, &types.StateSyncsByTimeRequest{
		FromId:     fromID,
		ToTime:     toTime,
		Pagination: query.PageRequest{Limit: limit, Key: []byte{0x00}},
	})
}

// TestProcessPendingVisibilityEvents verifies that pending events get assigned
// visibility_height and are cleared from the pending collection.
func (s *KeeperTestSuite) TestProcessPendingVisibilityEvents() {
	ctx, ck := s.ctx, s.keeper
	require := s.Require()

	blockTime := time.Date(2026, 1, 12, 17, 5, 40, 0, time.UTC)
	blockHeight := int64(500)
	ctx = ctx.WithBlockHeight(blockHeight).WithBlockHeader(cmtproto.Header{Time: blockTime, Height: blockHeight})

	// Store 3 events and add them as pending
	for i := uint64(1); i <= 3; i++ {
		rec := types.NewEventRecord(TxHash1, i, i, Address1, make([]byte, 1), "1", blockTime.Add(-time.Minute))
		rec.RecordTime = rec.RecordTime.UTC()
		require.NoError(ck.SetEventRecord(ctx, rec))
		require.NoError(ck.AddPendingVisibilityEvent(ctx, i))
	}

	// Verify events are pending (no visibility height yet)
	for i := uint64(1); i <= 3; i++ {
		_, err := ck.GetVisibilityHeightForEvent(ctx, i)
		require.Error(err, "event %d should not have visibility height before processing", i)
	}

	// Process pending events
	require.NoError(ck.ProcessPendingVisibilityEvents(ctx))

	// All events should now have visibility_height = blockHeight
	for i := uint64(1); i <= 3; i++ {
		vh, err := ck.GetVisibilityHeightForEvent(ctx, i)
		require.NoError(err, "event %d should have visibility height after processing", i)
		require.Equal(uint64(blockHeight), vh, "event %d visibility_height should equal block height", i)
	}

	// Pending list should be cleared
	hasPending, err := ck.PendingVisibilityEvents.Has(ctx, 1)
	require.NoError(err)
	require.False(hasPending, "pending list should be empty after processing")
}

// TestProcessPendingVisibilityEventsEmpty verifies no-op when no pending events.
func (s *KeeperTestSuite) TestProcessPendingVisibilityEventsEmpty() {
	ctx, ck := s.ctx, s.keeper
	require := s.Require()

	blockTime := time.Date(2026, 1, 12, 17, 5, 40, 0, time.UTC)
	ctx = ctx.WithBlockHeader(cmtproto.Header{Time: blockTime})

	// Processing with no pending events should succeed (no-op)
	require.NoError(ck.ProcessPendingVisibilityEvents(ctx))
}

// TestVisibilityTimeUpgradeID verifies get/set of the upgrade boundary marker.
func (s *KeeperTestSuite) TestVisibilityTimeUpgradeID() {
	ctx, ck := s.ctx, s.keeper
	require := s.Require()

	// Not set initially
	_, err := ck.GetVisibilityTimeUpgradeID(ctx)
	require.Error(err, "upgrade ID should not exist initially")

	// Set and retrieve
	require.NoError(ck.SetVisibilityTimeUpgradeID(ctx, 42))
	id, err := ck.GetVisibilityTimeUpgradeID(ctx)
	require.NoError(err)
	require.Equal(uint64(42), id)
}

// TestGetRecordListWithTime_LegacyQueryPreserved verifies that when no upgrade ID
// is set, the query behaves identically to the original legacy path.
func (s *KeeperTestSuite) TestGetRecordListWithTime_LegacyQueryPreserved() {
	ctx, ck, queryClient := s.ctx, s.keeper, s.queryClient
	require := s.Require()

	now := time.Now().UTC()

	// Insert records with ascending record_time
	for i := uint64(1); i <= 5; i++ {
		rec := types.NewEventRecord(TxHash1, i, i, Address1, make([]byte, 1), "1",
			now.Add(-time.Duration(10-i)*time.Minute))
		require.NoError(ck.SetEventRecord(ctx, rec))
	}

	// No upgrade ID set should use the legacy path
	req := &types.RecordListWithTimeRequest{
		FromId:     1,
		ToTime:     now,
		Pagination: query.PageRequest{Limit: 10, Key: []byte{0x00}},
	}
	res, err := queryClient.GetRecordListWithTime(ctx, req)
	require.NoError(err)
	require.Len(res.EventRecords, 5)
}

// TestGetRecordListWithTime_AlwaysUsesRecordTime verifies that GetRecordListWithTime
// always filters by record_time, even for post-upgrade events with visibility_height set.
// This preserves backward compatibility for EL clients using the old (from_id, to_time) pattern.
func (s *KeeperTestSuite) TestGetRecordListWithTime_AlwaysUsesRecordTime() {
	ctx, ck, queryClient := s.ctx, s.keeper, s.queryClient
	require := s.Require()

	baseTime := time.Date(2026, 1, 12, 16, 50, 0, 0, time.UTC)

	require.NoError(ck.SetVisibilityTimeUpgradeID(ctx, 1))

	// Create an event with record_time within the query window
	rec := types.NewEventRecord(TxHash1, 1, 1, Address1, make([]byte, 1), "1",
		baseTime.Add(8*time.Minute+14*time.Second))
	require.NoError(ck.SetEventRecord(ctx, rec))

	// Set visibility_height (production uses height, not time)
	require.NoError(ck.VisibilityHeightByID.Set(ctx, 1, 200))

	// Query with to_time after record_time
	// Old endpoint uses record_time, so event IS returned
	toTime := baseTime.Add(8*time.Minute + 30*time.Second)
	req := &types.RecordListWithTimeRequest{
		FromId:     1,
		ToTime:     toTime,
		Pagination: query.PageRequest{Limit: 10, Key: []byte{0x00}},
	}
	res, err := queryClient.GetRecordListWithTime(ctx, req)
	require.NoError(err)
	require.Len(res.EventRecords, 1, "old endpoint uses record_time: event returned when record_time < to_time")
}

// TestGetRecordListWithTime_HaltSimulation simulates the January 2026 halt scenario.
// The old /clerk/time endpoint uses record_time, so it returns the event regardless
// of visibility_height. This is the pre-existing non-deterministic behavior.
// The FIX for determinism is in GetStateSyncsByTime (the deterministic endpoint),
// not in the old endpoint — the old endpoint must remain backward-compatible.
func (s *KeeperTestSuite) TestGetRecordListWithTime_HaltSimulation() {
	ctx, ck, queryClient := s.ctx, s.keeper, s.queryClient
	require := s.Require()

	recordTime := time.Date(2026, 1, 12, 16, 58, 14, 0, time.UTC)
	borToTime := time.Date(2026, 1, 12, 16, 58, 30, 0, time.UTC)

	require.NoError(ck.SetVisibilityTimeUpgradeID(ctx, 3131120))

	rec := types.NewEventRecord(TxHash1, 1, 3131120, Address1, make([]byte, 1), "1", recordTime)
	require.NoError(ck.SetEventRecord(ctx, rec))
	require.NoError(ck.VisibilityHeightByID.Set(ctx, 3131120, 500))

	// Old endpoint: record_time (16:58:14) < borToTime (16:58:30), so event IS returned.
	// This is the non-deterministic behavior the old endpoint has always had.
	req := &types.RecordListWithTimeRequest{
		FromId:     3131120,
		ToTime:     borToTime,
		Pagination: query.PageRequest{Limit: 10, Key: []byte{0x00}},
	}
	res, err := queryClient.GetRecordListWithTime(ctx, req)
	require.NoError(err)
	require.Len(res.EventRecords, 1, "old endpoint uses record_time: event returned")

	// Deterministic query uses record_time too — same result, hence deterministic
	// for the same committed state (the non-determinism was across different Heimdall states,
	// not across repeated queries of the same state)
	res2, err := queryClient.GetRecordListWithTime(ctx, req)
	require.NoError(err)
	require.Len(res2.EventRecords, 1, "repeat query returns same result")
}

// TestGetRecordListWithTime_PendingEventsIncluded verifies that GetRecordListWithTime
// always filters by record_time, so pending events (no visibility_height yet) ARE returned
// as long as their record_time < to_time. Visibility-time filtering is only in
// GetStateSyncsByTime.
func (s *KeeperTestSuite) TestGetRecordListWithTime_PendingEventsIncluded() {
	ctx, ck, queryClient := s.ctx, s.keeper, s.queryClient
	require := s.Require()

	now := time.Now().UTC()

	require.NoError(ck.SetVisibilityTimeUpgradeID(ctx, 1))

	// Create an event with record_time in the past
	rec := types.NewEventRecord(TxHash1, 1, 1, Address1, make([]byte, 1), "1",
		now.Add(-5*time.Minute))
	require.NoError(ck.SetEventRecord(ctx, rec))

	// Add to pending but don't assign visibility_height
	require.NoError(ck.AddPendingVisibilityEvent(ctx, 1))

	req := &types.RecordListWithTimeRequest{
		FromId:     1,
		ToTime:     now,
		Pagination: query.PageRequest{Limit: 10, Key: []byte{0x00}},
	}
	res, err := queryClient.GetRecordListWithTime(ctx, req)
	require.NoError(err)
	require.Len(res.EventRecords, 1, "old endpoint uses record_time: pending event returned when record_time < to_time")
}

// TestGetRecordListWithTime_HybridQuery verifies that the legacy time-based
// endpoint always filters by record_time, even when some events are post-upgrade
// and already have visibility_height set.
func (s *KeeperTestSuite) TestGetRecordListWithTime_HybridQuery() {
	ctx, ck, queryClient := s.ctx, s.keeper, s.queryClient
	require := s.Require()

	baseTime := time.Date(2026, 1, 12, 16, 0, 0, 0, time.UTC)

	// Upgrade boundary at event ID 3
	require.NoError(ck.SetVisibilityTimeUpgradeID(ctx, 3))

	// Events 1-2: pre-upgrade (legacy), record_time within the query window
	for i := uint64(1); i <= 2; i++ {
		rec := types.NewEventRecord(TxHash1, i, i, Address1, make([]byte, 1), "1",
			baseTime.Add(time.Duration(i)*time.Minute))
		require.NoError(ck.SetEventRecord(ctx, rec))
	}

	// Events 3-4: post-upgrade, both have visibility_height set.
	rec3 := types.NewEventRecord(TxHash1, 3, 3, Address1, make([]byte, 1), "1",
		baseTime.Add(3*time.Minute)) // 16:03
	require.NoError(ck.SetEventRecord(ctx, rec3))
	require.NoError(ck.VisibilityHeightByID.Set(ctx, 3, 100))

	rec4 := types.NewEventRecord(TxHash1, 4, 4, Address1, make([]byte, 1), "1",
		baseTime.Add(4*time.Minute)) // 16:04
	require.NoError(ck.SetEventRecord(ctx, rec4))
	require.NoError(ck.VisibilityHeightByID.Set(ctx, 4, 101))

	// Query with from_id=1 and to_time=16:10 should return all 4 contiguous events
	toTime := baseTime.Add(10 * time.Minute)
	req := &types.RecordListWithTimeRequest{
		FromId:     1,
		ToTime:     toTime,
		Pagination: query.PageRequest{Limit: 10, Key: []byte{0x00}},
	}
	res, err := queryClient.GetRecordListWithTime(ctx, req)
	require.NoError(err)
	require.Len(res.EventRecords, 4, "should return all 4 events")
	for i, rec := range res.EventRecords {
		require.Equal(uint64(i+1), rec.Id, "IDs must be contiguous")
	}
}

// TestGetRecordListWithTime_AllEventsReturnedByRecordTime verifies that GetRecordListWithTime
// always filters by record_time regardless of the upgrade boundary or visibility_height.
// Even when visibility_height is far in the future (e.g., due to a halt), the old endpoint
// returns events based solely on record_time < to_time.
func (s *KeeperTestSuite) TestGetRecordListWithTime_AllEventsReturnedByRecordTime() {
	ctx, ck, queryClient := s.ctx, s.keeper, s.queryClient
	require := s.Require()

	baseTime := time.Date(2026, 1, 12, 16, 0, 0, 0, time.UTC)

	// Upgrade boundary at event ID 3
	require.NoError(ck.SetVisibilityTimeUpgradeID(ctx, 3))

	// Events 1-2: pre-upgrade, record_time within the query window
	for i := uint64(1); i <= 2; i++ {
		rec := types.NewEventRecord(TxHash1, i, i, Address1, make([]byte, 1), "1",
			baseTime.Add(time.Duration(i)*time.Minute))
		require.NoError(ck.SetEventRecord(ctx, rec))
	}

	// Event 3: post-upgrade, stored during a halt, hence the visibility_height is much later
	rec3 := types.NewEventRecord(TxHash1, 3, 3, Address1, make([]byte, 1), "1",
		baseTime.Add(3*time.Minute))
	require.NoError(ck.SetEventRecord(ctx, rec3))
	require.NoError(ck.VisibilityHeightByID.Set(ctx, 3, 1000))

	// Event 4: post-upgrade, also stored during the same halt
	rec4 := types.NewEventRecord(TxHash1, 4, 4, Address1, make([]byte, 1), "1",
		baseTime.Add(4*time.Minute))
	require.NoError(ck.SetEventRecord(ctx, rec4))
	require.NoError(ck.VisibilityHeightByID.Set(ctx, 4, 1001))

	// Query with to_time=16:10: all 4 events have record_time < 16:10, so all are returned.
	// The old endpoint ignores visibility_height entirely.
	toTime := baseTime.Add(10 * time.Minute)
	req := &types.RecordListWithTimeRequest{
		FromId:     1,
		ToTime:     toTime,
		Pagination: query.PageRequest{Limit: 10, Key: []byte{0x00}},
	}
	res, err := queryClient.GetRecordListWithTime(ctx, req)
	require.NoError(err)
	require.Len(res.EventRecords, 4, "old endpoint uses record_time: all 4 events returned when record_time < to_time")
	for i, rec := range res.EventRecords {
		require.Equal(uint64(i+1), rec.Id, "IDs must be contiguous")
	}
}

// TestMultipleEventsInSameBlock verifies that multiple events in the same block
// all get the same visibility_height but have distinct composite keys.
func (s *KeeperTestSuite) TestMultipleEventsInSameBlock() {
	ctx, ck := s.ctx, s.keeper
	require := s.Require()

	blockTime := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	blockHeight := int64(500)
	ctx = ctx.WithBlockHeight(blockHeight).WithBlockHeader(cmtproto.Header{Time: blockTime, Height: blockHeight})

	// Store 5 events and add them as pending
	for i := uint64(1); i <= 5; i++ {
		rec := types.NewEventRecord(TxHash1, i, i, Address1, make([]byte, 1), "1",
			blockTime.Add(-time.Minute))
		rec.RecordTime = rec.RecordTime.UTC()
		require.NoError(ck.SetEventRecord(ctx, rec))
		require.NoError(ck.AddPendingVisibilityEvent(ctx, i))
	}

	// Process all pending events
	require.NoError(ck.ProcessPendingVisibilityEvents(ctx))

	// All should have the same visibility_height
	for i := uint64(1); i <= 5; i++ {
		vh, err := ck.GetVisibilityHeightForEvent(ctx, i)
		require.NoError(err)
		require.Equal(uint64(blockHeight), vh, "event %d should have visibility_height = blockHeight", i)
	}

}

// TestVisibilityTimeFeatureFlag tests the IsVisibilityTimeEnabled helper function.
func (s *KeeperTestSuite) TestVisibilityTimeFeatureFlag() {
	require := s.Require()

	// Save and restore the original value
	original := helper.GetVisibilityTimeHeight()
	defer helper.SetVisibilityTimeHeight(original)

	// Height 0 means disabled
	helper.SetVisibilityTimeHeight(0)
	require.False(helper.IsVisibilityTimeEnabled(100))
	require.False(helper.IsVisibilityTimeEnabled(0))

	// Height 256 means enabled at block 256+
	helper.SetVisibilityTimeHeight(256)
	require.False(helper.IsVisibilityTimeEnabled(255))
	require.True(helper.IsVisibilityTimeEnabled(256))
	require.True(helper.IsVisibilityTimeEnabled(300))
}

// TestPostHandlerAddsToVisibilityPending verifies that PostHandleMsgEventRecord
// adds events to the pending visibility list when the feature is enabled.
func (s *KeeperTestSuite) TestPostHandlerAddsToVisibilityPending() {
	ctx, ck, postHandler := s.ctx, s.keeper, s.postHandler
	require := s.Require()

	// Save and restore the original value
	original := helper.GetVisibilityTimeHeight()
	defer helper.SetVisibilityTimeHeight(original)

	// Enable visibility time at block 1
	helper.SetVisibilityTimeHeight(1)

	// Set block height to an enabled height
	ctx = ctx.WithBlockHeight(10)

	postHandler(ctx, new(types.NewMsgEventRecord(
		Address1, TxHash1, 1, 100, 42, []byte(Address2), make([]byte, 0), s.chainId,
	)), sidetxs.Vote_VOTE_YES)

	// Verify the event was stored
	rec, err := ck.GetEventRecord(ctx, 42)
	require.NoError(err)
	require.NotNil(rec)

	// Verify event is in the pending list
	hasPending, err := ck.PendingVisibilityEvents.Has(ctx, 42)
	require.NoError(err)
	require.True(hasPending, "event should be in pending visibility events")

	// Verify upgrade ID was set
	upgradeID, err := ck.GetVisibilityTimeUpgradeID(ctx)
	require.NoError(err)
	require.Equal(uint64(42), upgradeID, "upgrade ID should be set to first post-upgrade event")
}

// TestPostHandlerSkipsVisibilityWhenDisabled verifies that PostHandleMsgEventRecord
// does not add events to pending when the feature is disabled.
func (s *KeeperTestSuite) TestPostHandlerSkipsVisibilityWhenDisabled() {
	ctx, ck, postHandler := s.ctx, s.keeper, s.postHandler
	require := s.Require()

	// Save and restore
	original := helper.GetVisibilityTimeHeight()
	defer helper.SetVisibilityTimeHeight(original)

	// Disable visibility time
	helper.SetVisibilityTimeHeight(0)

	ctx = ctx.WithBlockHeight(10)

	postHandler(ctx, new(types.NewMsgEventRecord(
		Address1, TxHash1, 1, 100, 99, []byte(Address2), make([]byte, 0), s.chainId,
	)), sidetxs.Vote_VOTE_YES)

	// Verify event was stored
	rec, err := ck.GetEventRecord(ctx, 99)
	require.NoError(err)
	require.NotNil(rec)

	// Verify the event is not in the pending list
	hasPending, err := ck.PendingVisibilityEvents.Has(ctx, 99)
	require.NoError(err)
	require.False(hasPending, "event should not be in pending when feature is disabled")

	// Verify upgrade ID was not set
	_, err = ck.GetVisibilityTimeUpgradeID(ctx)
	require.Error(err, "upgrade ID should not be set when feature is disabled")
}

// TestUpgradeIDBoundarySetOnce verifies that the visibility time upgrade ID
// is only set on the first post-upgrade event.
func (s *KeeperTestSuite) TestUpgradeIDBoundarySetOnce() {
	ctx, ck, postHandler := s.ctx, s.keeper, s.postHandler
	require := s.Require()

	original := helper.GetVisibilityTimeHeight()
	defer helper.SetVisibilityTimeHeight(original)

	helper.SetVisibilityTimeHeight(1)
	ctx = ctx.WithBlockHeight(10)

	// The first event sets the upgrade ID
	postHandler(ctx, new(types.NewMsgEventRecord(
		Address1, TxHash1, 1, 100, 10, []byte(Address2), make([]byte, 0), s.chainId,
	)), sidetxs.Vote_VOTE_YES)

	upgradeID, err := ck.GetVisibilityTimeUpgradeID(ctx)
	require.NoError(err)
	require.Equal(uint64(10), upgradeID)

	// The second event should not change the upgrade ID
	postHandler(ctx, new(types.NewMsgEventRecord(
		Address1, TxHash1, 2, 200, 20, []byte(Address2), make([]byte, 0), s.chainId,
	)), sidetxs.Vote_VOTE_YES)

	upgradeID, err = ck.GetVisibilityTimeUpgradeID(ctx)
	require.NoError(err)
	require.Equal(uint64(10), upgradeID, "upgrade ID should remain at first event")
}

// TestGetRecordListWithTime_PendingAtFirstID verifies that GetRecordListWithTime
// returns all events based on record_time, regardless of pending status.
// Pending events (no visibility_height yet) are still returned because the old endpoint
// filters only by record_time.
func (s *KeeperTestSuite) TestGetRecordListWithTime_PendingAtFirstID() {
	ctx, ck, queryClient := s.ctx, s.keeper, s.queryClient
	require := s.Require()

	now := time.Now().UTC()
	require.NoError(ck.SetVisibilityTimeUpgradeID(ctx, 1))

	// Events 1-3: all post-upgrade. Event 1 is pending, events 2-3 have visibility_height.
	for i := uint64(1); i <= 3; i++ {
		rec := types.NewEventRecord(TxHash1, i, i, Address1, make([]byte, 1), "1",
			now.Add(-time.Duration(5-i)*time.Minute))
		require.NoError(ck.SetEventRecord(ctx, rec))
	}
	// Only set visibility_height for events 2 and 3 (the first event remains pending)
	require.NoError(ck.VisibilityHeightByID.Set(ctx, 2, 100))
	require.NoError(ck.VisibilityHeightByID.Set(ctx, 3, 101))

	req := &types.RecordListWithTimeRequest{
		FromId:     1,
		ToTime:     now,
		Pagination: query.PageRequest{Limit: 10, Key: []byte{0x00}},
	}
	res, err := queryClient.GetRecordListWithTime(ctx, req)
	require.NoError(err)
	require.Len(res.EventRecords, 3, "old endpoint uses record_time: all 3 events returned regardless of pending status")
	for i, rec := range res.EventRecords {
		require.Equal(uint64(i+1), rec.Id, "IDs must be contiguous")
	}
}

// TestGetRecordListWithTime_PendingGapInMiddle verifies that GetRecordListWithTime
// returns all events based on record_time, even when some events in the middle
// are pending (no visibility_height yet). The old endpoint ignores visibility_height entirely.
func (s *KeeperTestSuite) TestGetRecordListWithTime_PendingGapInMiddle() {
	ctx, ck, queryClient := s.ctx, s.keeper, s.queryClient
	require := s.Require()

	now := time.Now().UTC()
	require.NoError(ck.SetVisibilityTimeUpgradeID(ctx, 1))

	// Events 1-4: all post-upgrade
	for i := uint64(1); i <= 4; i++ {
		rec := types.NewEventRecord(TxHash1, i, i, Address1, make([]byte, 1), "1",
			now.Add(-time.Duration(10-i)*time.Minute))
		require.NoError(ck.SetEventRecord(ctx, rec))
	}
	// Set visibility_height for 1, 2, and 4 — but not 3 (simulating a pending event)
	require.NoError(ck.VisibilityHeightByID.Set(ctx, 1, 100))
	require.NoError(ck.VisibilityHeightByID.Set(ctx, 2, 101))
	// event 3: pending (no visibility_height)
	require.NoError(ck.VisibilityHeightByID.Set(ctx, 4, 103))

	req := &types.RecordListWithTimeRequest{
		FromId:     1,
		ToTime:     now,
		Pagination: query.PageRequest{Limit: 10, Key: []byte{0x00}},
	}
	res, err := queryClient.GetRecordListWithTime(ctx, req)
	require.NoError(err)
	require.Len(res.EventRecords, 4, "old endpoint uses record_time: all 4 events returned regardless of pending status")
	for i, rec := range res.EventRecords {
		require.Equal(uint64(i+1), rec.Id, "IDs must be contiguous")
	}
}

// TestGetRecordListWithTime_RecordTimeBreakPreservesContiguity verifies that
// GetRecordListWithTime always uses record_time for filtering (even post-visibility-upgrade),
// and breaks at the first event with record_time >= to_time, preserving contiguous ID order.
func (s *KeeperTestSuite) TestGetRecordListWithTime_RecordTimeBreakPreservesContiguity() {
	ctx, ck, queryClient := s.ctx, s.keeper, s.queryClient
	require := s.Require()

	baseTime := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	require.NoError(ck.SetVisibilityTimeUpgradeID(ctx, 1))

	// Events 1-5: post-upgrade with increasing record_time
	for i := uint64(1); i <= 5; i++ {
		rec := types.NewEventRecord(TxHash1, i, i, Address1, make([]byte, 1), "1",
			baseTime.Add(time.Duration(i)*time.Minute))
		require.NoError(ck.SetEventRecord(ctx, rec))
		require.NoError(ck.VisibilityHeightByID.Set(ctx, i, 100+i))
	}

	// Query with to_time between event 3's and event 4's record_time
	// Events 1-3 should be returned (record_time < to_time), event 4 breaks
	toTime := baseTime.Add(3*time.Minute + 30*time.Second)
	req := &types.RecordListWithTimeRequest{
		FromId:     1,
		ToTime:     toTime,
		Pagination: query.PageRequest{Limit: 10, Key: []byte{0x00}},
	}
	res, err := queryClient.GetRecordListWithTime(ctx, req)
	require.NoError(err)
	require.Len(res.EventRecords, 3)
	for i, rec := range res.EventRecords {
		require.Equal(uint64(i+1), rec.Id, "IDs must be contiguous starting from 1")
	}
}

// TestQueryDeterminismAcrossCommittedStates verifies that GetRecordListWithTime
// (the old /clerk/time endpoint) always uses record_time for filtering, even after
// the visibility-height upgrade. This preserves backward compatibility: EL clients
// using the old (from_id, to_time) query pattern continue to get consistent results
// until they switch to GetStateSyncsByTime after the EL HF.
//
// Determinism for post-EL-fork nodes is provided by GetStateSyncsByTime,
// which filters by visibility_height — tested separately.
func (s *KeeperTestSuite) TestQueryDeterminismAcrossCommittedStates() {
	ctx, ck, queryClient := s.ctx, s.keeper, s.queryClient
	require := s.Require()

	require.NoError(ck.SetVisibilityTimeUpgradeID(ctx, 1))

	// Block H: event stored with pending status
	blockHTime := time.Date(2026, 1, 12, 16, 58, 14, 0, time.UTC)
	rec := types.NewEventRecord(TxHash1, 1, 1, Address1, make([]byte, 1), "1", blockHTime)
	require.NoError(ck.SetEventRecord(ctx, rec))
	require.NoError(ck.AddPendingVisibilityEvent(ctx, 1))

	// Query with to_time after record_time: event IS returned because the old endpoint
	// always filters by record_time, not visibility_height. This is correct for backward
	// compatibility with EL clients still using the old query pattern.
	queryToTime := time.Date(2026, 1, 12, 17, 10, 0, 0, time.UTC)
	req := &types.RecordListWithTimeRequest{
		FromId:     1,
		ToTime:     queryToTime,
		Pagination: query.PageRequest{Limit: 10, Key: []byte{0x00}},
	}

	res1, err := queryClient.GetRecordListWithTime(ctx, req)
	require.NoError(err)
	require.Len(res1.EventRecords, 1, "old endpoint uses record_time: event visible when record_time < to_time")

	// Re-query: same result (deterministic)
	res2, err := queryClient.GetRecordListWithTime(ctx, req)
	require.NoError(err)
	require.Len(res2.EventRecords, 1, "repeat query returns same result")

	// Query with to_time before record_time: event NOT returned
	earlyToTime := time.Date(2026, 1, 12, 16, 50, 0, 0, time.UTC)
	earlyReq := &types.RecordListWithTimeRequest{
		FromId:     1,
		ToTime:     earlyToTime,
		Pagination: query.PageRequest{Limit: 10, Key: []byte{0x00}},
	}
	earlyRes, err := queryClient.GetRecordListWithTime(ctx, earlyReq)
	require.NoError(err)
	require.Empty(earlyRes.EventRecords, "to_time before record_time: event not returned")
}

// TestEndToEndVisibilityTimeLifecycle simulates the full event lifecycle:
// Block H: event created → Block H+1: stored with pending → Block H+2: visibility_height assigned
//
// GetRecordListWithTime always uses record_time, so the event is visible immediately
// after being stored (even while pending). The visibility_height is only relevant for
// GetStateSyncsByTime (the deterministic endpoint).
func (s *KeeperTestSuite) TestEndToEndVisibilityTimeLifecycle() {
	ctx, ck, queryClient, postHandler := s.ctx, s.keeper, s.queryClient, s.postHandler
	require := s.Require()

	original := helper.GetVisibilityTimeHeight()
	defer helper.SetVisibilityTimeHeight(original)
	helper.SetVisibilityTimeHeight(1)

	// Block H+1: PostHandler stores event and adds to pending
	blockH1Time := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	ctx = ctx.WithBlockHeight(100).WithBlockHeader(cmtproto.Header{Time: blockH1Time, Height: 100})

	postHandler(ctx, new(types.NewMsgEventRecord(
		Address1, TxHash1, 1, 500, 1000, []byte(Address2), make([]byte, 0), s.chainId,
	)), sidetxs.Vote_VOTE_YES)

	// Old endpoint uses record_time: event IS returned even while pending,
	// because record_time < to_time. The event's record_time is set by NewMsgEventRecord.
	toTime := blockH1Time.Add(time.Minute)
	req := &types.RecordListWithTimeRequest{
		FromId:     1000,
		ToTime:     toTime,
		Pagination: query.PageRequest{Limit: 10, Key: []byte{0x00}},
	}
	res, err := queryClient.GetRecordListWithTime(ctx, req)
	require.NoError(err)
	require.Len(res.EventRecords, 1, "old endpoint uses record_time: pending event returned when record_time < to_time")
	require.Equal(uint64(1000), res.EventRecords[0].Id)

	// Block H+2: PreBlocker processes pending → assigns visibility_height
	blockH2Time := time.Date(2026, 3, 15, 10, 0, 3, 0, time.UTC) // ~3s later
	ctx = ctx.WithBlockHeight(101).WithBlockHeader(cmtproto.Header{Time: blockH2Time, Height: 101})

	require.NoError(ck.ProcessPendingVisibilityEvents(ctx))

	// Event is still returned (record_time hasn't changed)
	toTimeAfter := blockH2Time.Add(time.Minute)
	reqAfter := &types.RecordListWithTimeRequest{
		FromId:     1000,
		ToTime:     toTimeAfter,
		Pagination: query.PageRequest{Limit: 10, Key: []byte{0x00}},
	}
	resAfter, err := queryClient.GetRecordListWithTime(ctx, reqAfter)
	require.NoError(err)
	require.Len(resAfter.EventRecords, 1)
	require.Equal(uint64(1000), resAfter.EventRecords[0].Id)

	// Verify visibility_height was set correctly (used by GetStateSyncsByTime)
	vh, err := ck.GetVisibilityHeightForEvent(ctx, 1000)
	require.NoError(err)
	require.Equal(uint64(101), vh)
}

// TestStoreBlockTime verifies that StoreBlockTime stores block time mappings
// that can be resolved via GetBlockHeightByTime (the reverse index).
func (s *KeeperTestSuite) TestStoreBlockTime() {
	ctx, ck := s.ctx, s.keeper
	require := s.Require()

	// Store block times for several heights
	times := []struct {
		height int64
		time   time.Time
	}{
		{100, time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)},
		{101, time.Date(2026, 3, 15, 10, 0, 3, 0, time.UTC)},
		{102, time.Date(2026, 3, 15, 10, 0, 6, 0, time.UTC)},
	}

	for _, tt := range times {
		ctx = ctx.WithBlockHeight(tt.height).WithBlockHeader(cmtproto.Header{Time: tt.time, Height: tt.height})
		require.NoError(ck.StoreBlockTime(ctx))
	}

	// Verify each mapping is resolvable via GetBlockHeightByTime
	for _, tt := range times {
		resolvedHeight, err := ck.GetBlockHeightByTime(ctx, tt.time.Unix())
		require.NoError(err)
		require.Equal(tt.height, resolvedHeight)
	}
}

// TestGetBlockHeightByTime verifies that GetBlockHeightByTime returns the greatest
// height with blockTime <= cutoff for various cutoff values.
func (s *KeeperTestSuite) TestGetBlockHeightByTime() {
	ctx, ck := s.ctx, s.keeper
	require := s.Require()

	// Store block times: heights 100-104 with 3-second intervals
	baseTime := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	for i := int64(0); i < 5; i++ {
		h := 100 + i
		bt := baseTime.Add(time.Duration(i*3) * time.Second)
		ctx = ctx.WithBlockHeight(h).WithBlockHeader(cmtproto.Header{Time: bt, Height: h})
		require.NoError(ck.StoreBlockTime(ctx))
	}

	// Cutoff at exactly block 102's time should return height 102
	height, err := ck.GetBlockHeightByTime(ctx, baseTime.Add(6*time.Second).Unix())
	require.NoError(err)
	require.Equal(int64(102), height)

	// Cutoff between block 102 and block 103 should return 102
	height, err = ck.GetBlockHeightByTime(ctx, baseTime.Add(7*time.Second).Unix())
	require.NoError(err)
	require.Equal(int64(102), height)

	// Cutoff at block 104's time should return 104
	height, err = ck.GetBlockHeightByTime(ctx, baseTime.Add(12*time.Second).Unix())
	require.NoError(err)
	require.Equal(int64(104), height)

	// Cutoff well after all blocks should return 104 (the highest)
	height, err = ck.GetBlockHeightByTime(ctx, baseTime.Add(time.Hour).Unix())
	require.NoError(err)
	require.Equal(int64(104), height)
}

// TestGetBlockHeightByTime_CutoffBeforeAnyBlock verifies that an error is returned
// when the cutoff time is before any stored block.
func (s *KeeperTestSuite) TestGetBlockHeightByTime_CutoffBeforeAnyBlock() {
	ctx, ck := s.ctx, s.keeper
	require := s.Require()

	// Store a single block
	bt := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	ctx = ctx.WithBlockHeight(100).WithBlockHeader(cmtproto.Header{Time: bt, Height: 100})
	require.NoError(ck.StoreBlockTime(ctx))

	// Cutoff 1 second before the only block triggers an error
	_, err := ck.GetBlockHeightByTime(ctx, bt.Add(-time.Second).Unix())
	require.Error(err)
	require.Contains(err.Error(), "no block found")
}

// TestGetBlockHeightByTime_ExactMatch verifies that when the cutoff exactly matches
// a block's timestamp, that block's height is returned.
func (s *KeeperTestSuite) TestGetBlockHeightByTime_ExactMatch() {
	ctx, ck := s.ctx, s.keeper
	require := s.Require()

	// Store blocks at heights 200, 201, 202
	times := []time.Time{
		time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC),
		time.Date(2026, 6, 1, 12, 0, 3, 0, time.UTC),
		time.Date(2026, 6, 1, 12, 0, 6, 0, time.UTC),
	}

	for i, bt := range times {
		h := int64(200 + i)
		ctx = ctx.WithBlockHeight(h).WithBlockHeader(cmtproto.Header{Time: bt, Height: h})
		require.NoError(ck.StoreBlockTime(ctx))
	}

	// Exact match at each block's time
	for i, bt := range times {
		height, err := ck.GetBlockHeightByTime(ctx, bt.Unix())
		require.NoError(err)
		require.Equal(int64(200+i), height, "exact match at block %d", 200+i)
	}
}

// TestGetBlockHeightByTime_TieBreaking verifies that when multiple blocks have the
// same timestamp (same unix second), the greatest height wins.
func (s *KeeperTestSuite) TestGetBlockHeightByTime_TieBreaking() {
	ctx, ck := s.ctx, s.keeper
	require := s.Require()

	// Store 3 blocks all at the same unix second
	sameTime := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	for i := int64(0); i < 3; i++ {
		h := 300 + i
		ctx = ctx.WithBlockHeight(h).WithBlockHeader(cmtproto.Header{Time: sameTime, Height: h})
		require.NoError(ck.StoreBlockTime(ctx))
	}

	// Cutoff at that exact second should return the greatest height (302)
	// because descending iteration visits 302 first and its time <= cutoff
	height, err := ck.GetBlockHeightByTime(ctx, sameTime.Unix())
	require.NoError(err)
	require.Equal(int64(302), height, "should return greatest height when timestamps tie")
}

// TestGetBlockHeightByTime_EmptyIndex verifies that an error is returned when
// no blocks have been indexed.
func (s *KeeperTestSuite) TestGetBlockHeightByTime_EmptyIndex() {
	ctx, ck := s.ctx, s.keeper
	require := s.Require()

	_, err := ck.GetBlockHeightByTime(ctx, time.Now().Unix())
	require.Error(err)
	require.Contains(err.Error(), "no block found")
}

// TestProcessPendingVisibilityEvents_StoresHeight verifies that after processing,
// events have visibility_height set to the processing block's height.
func (s *KeeperTestSuite) TestProcessPendingVisibilityEvents_StoresHeight() {
	ctx, ck := s.ctx, s.keeper
	require := s.Require()

	blockTime := time.Date(2026, 1, 12, 17, 5, 40, 0, time.UTC)
	blockHeight := int64(500)
	ctx = ctx.WithBlockHeight(blockHeight).WithBlockHeader(cmtproto.Header{Time: blockTime, Height: blockHeight})

	// Store 3 events and add them as pending
	for i := uint64(1); i <= 3; i++ {
		rec := types.NewEventRecord(TxHash1, i, i, Address1, make([]byte, 1), "1", blockTime.Add(-time.Minute))
		rec.RecordTime = rec.RecordTime.UTC()
		require.NoError(ck.SetEventRecord(ctx, rec))
		require.NoError(ck.AddPendingVisibilityEvent(ctx, i))
	}

	// Process pending events
	require.NoError(ck.ProcessPendingVisibilityEvents(ctx))

	// All events should have visibility_height = blockHeight
	for i := uint64(1); i <= 3; i++ {
		vh, err := ck.GetVisibilityHeightForEvent(ctx, i)
		require.NoError(err, "event %d should have visibility height after processing", i)
		require.Equal(uint64(blockHeight), vh, "event %d visibility_height should equal block height", i)
	}

}

// TestGetStateSyncsByTime_BasicFlow stores events with different visibility
// heights, resolves a specific height from cutoff time, and verifies filtering.
func (s *KeeperTestSuite) TestGetStateSyncsByTime_BasicFlow() {
	ctx, ck := s.ctx, s.keeper
	ctx = ctx.WithBlockHeight(10000)
	require := s.Require()

	require.NoError(ck.SetVisibilityTimeUpgradeID(ctx, 1))

	baseTime := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)

	// Create 5 events with ascending visibility heights
	for i := uint64(1); i <= 5; i++ {
		rec := types.NewEventRecord(TxHash1, i, i, Address1, make([]byte, 1), "1",
			baseTime.Add(time.Duration(i)*time.Minute))
		require.NoError(ck.SetEventRecord(ctx, rec))
		// Assign visibility at height 100+i
		visHeight := 100 + i
		require.NoError(ck.VisibilityHeightByID.Set(ctx, i, visHeight))
	}

	resp, err := s.queryDeterministicStateSyncs(ctx, 103, 1, baseTime.Add(time.Hour), 10)
	require.NoError(err)
	require.Len(resp.EventRecords, 3)
	for i, rec := range resp.EventRecords {
		require.Equal(uint64(i+1), rec.Id, "IDs must be contiguous")
	}
}

// TestGetStateSyncsByTime_HybridQuery mixes pre-upgrade (record_time filtered)
// and post-upgrade (visibility_height filtered) events. It verifies the
// deterministic endpoint resolves the height internally while preserving the same
// legacy-vs-post-upgrade filtering rules.
func (s *KeeperTestSuite) TestGetStateSyncsByTime_HybridQuery() {
	ctx, ck := s.ctx, s.keeper
	ctx = ctx.WithBlockHeight(10000)
	require := s.Require()

	baseTime := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)

	// Upgrade boundary at event ID 3
	require.NoError(ck.SetVisibilityTimeUpgradeID(ctx, 3))

	// Events 1-2: pre-upgrade (legacy), record_time within the query window
	for i := uint64(1); i <= 2; i++ {
		rec := types.NewEventRecord(TxHash1, i, i, Address1, make([]byte, 1), "1",
			baseTime.Add(time.Duration(i)*time.Minute))
		require.NoError(ck.SetEventRecord(ctx, rec))
	}

	// Events 3-4: post-upgrade with visibility heights
	for i := uint64(3); i <= 4; i++ {
		rec := types.NewEventRecord(TxHash1, i, i, Address1, make([]byte, 1), "1",
			baseTime.Add(time.Duration(i)*time.Minute))
		require.NoError(ck.SetEventRecord(ctx, rec))
		visHeight := 200 + i
		require.NoError(ck.VisibilityHeightByID.Set(ctx, i, visHeight))
	}

	resp2, err := s.queryDeterministicStateSyncs(ctx, 203, 1, baseTime.Add(time.Hour), 10)
	require.NoError(err)
	require.Len(resp2.EventRecords, 3, "should return 2 legacy + 1 post-upgrade")

	resp, err := s.queryDeterministicStateSyncs(ctx, 204, 1, baseTime.Add(time.Hour+time.Second), 10)
	require.NoError(err)
	require.Len(resp.EventRecords, 4, "should return all 4 events (2 legacy + 2 post-upgrade)")
	for i, rec := range resp.EventRecords {
		require.Equal(uint64(i+1), rec.Id, "IDs must be contiguous")
	}
}

// TestGetStateSyncsByTime_LegacyOutOfOrderRecordTimes verifies that a
// legacy event with record_time >= to_time does not hide a later-ID legacy event
// whose record_time is still before to_time.
func (s *KeeperTestSuite) TestGetStateSyncsByTime_LegacyOutOfOrderRecordTimes() {
	ctx, ck := s.ctx, s.keeper
	ctx = ctx.WithBlockHeight(10000)
	require := s.Require()

	baseTime := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	toTime := baseTime.Add(51 * time.Minute)

	require.NoError(ck.SetVisibilityTimeUpgradeID(ctx, 200))

	// Event 100 is legacy but recorded after event 101.
	require.NoError(ck.SetEventRecord(ctx, types.NewEventRecord(
		TxHash1, 100, 100, Address1, make([]byte, 1), "1", baseTime.Add(52*time.Minute),
	)))
	require.NoError(ck.SetEventRecord(ctx, types.NewEventRecord(
		TxHash1, 101, 101, Address1, make([]byte, 1), "1", baseTime.Add(50*time.Minute),
	)))

	resp, err := s.queryDeterministicStateSyncs(ctx, 1000, 100, toTime, 10)
	require.NoError(err)
	require.Len(resp.EventRecords, 1, "later eligible legacy events must not be hidden by an earlier ineligible ID")
	require.Equal(uint64(101), resp.EventRecords[0].Id)
}

// TestGetStateSyncsByTime_PendingExcluded verifies that post-upgrade events
// without visibility_height are skipped, while later visible events are still returned.
func (s *KeeperTestSuite) TestGetStateSyncsByTime_PendingExcluded() {
	ctx, ck := s.ctx, s.keeper
	ctx = ctx.WithBlockHeight(10000)
	require := s.Require()

	baseTime := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)

	require.NoError(ck.SetVisibilityTimeUpgradeID(ctx, 1))

	// Create 3 events, only set visibility_height for events 1 and 3 (event 2 pending)
	for i := uint64(1); i <= 3; i++ {
		rec := types.NewEventRecord(TxHash1, i, i, Address1, make([]byte, 1), "1",
			baseTime.Add(time.Duration(i)*time.Minute))
		require.NoError(ck.SetEventRecord(ctx, rec))
	}
	require.NoError(ck.VisibilityHeightByID.Set(ctx, 1, 100))
	// event 2: no visibility height (pending)
	require.NoError(ck.VisibilityHeightByID.Set(ctx, 3, 102))

	resp, err := s.queryDeterministicStateSyncs(ctx, 200, 1, baseTime.Add(time.Hour), 10)
	require.NoError(err)
	require.Len(resp.EventRecords, 2, "should return visible events 1 and 3, skipping pending event 2")
	require.Equal(uint64(1), resp.EventRecords[0].Id)
	require.Equal(uint64(3), resp.EventRecords[1].Id)
}

// TestGetStateSyncsByTime_OutOfOrderVisibilityHeights verifies that a
// lower-ID event becoming visible later does not hide a higher-ID event that is
// already visible at the requested height.
func (s *KeeperTestSuite) TestGetStateSyncsByTime_OutOfOrderVisibilityHeights() {
	ctx, ck := s.ctx, s.keeper
	ctx = ctx.WithBlockHeight(10000)
	require := s.Require()

	baseTime := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)

	require.NoError(ck.SetVisibilityTimeUpgradeID(ctx, 1))

	for i := uint64(1); i <= 2; i++ {
		rec := types.NewEventRecord(TxHash1, i, i, Address1, make([]byte, 1), "1",
			baseTime.Add(time.Duration(i)*time.Minute))
		require.NoError(ck.SetEventRecord(ctx, rec))
	}

	// Event 1 becomes visible later than event 2.
	require.NoError(ck.VisibilityHeightByID.Set(ctx, 1, 300))
	require.NoError(ck.VisibilityHeightByID.Set(ctx, 2, 200))

	resp, err := s.queryDeterministicStateSyncs(ctx, 250, 1, baseTime.Add(time.Hour), 10)
	require.NoError(err)
	require.Len(resp.EventRecords, 1, "should return event 2 even though event 1 becomes visible later")
	require.Equal(uint64(2), resp.EventRecords[0].Id)
}

// TestGetStateSyncsByTime_LegacyEventDoesNotHidePostUpgradeEvents verifies
// that an out-of-order legacy event near the upgrade boundary does not truncate the
// scan before eligible post-upgrade events.
func (s *KeeperTestSuite) TestGetStateSyncsByTime_LegacyEventDoesNotHidePostUpgradeEvents() {
	ctx, ck := s.ctx, s.keeper
	ctx = ctx.WithBlockHeight(10000)
	require := s.Require()

	baseTime := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	toTime := baseTime.Add(45 * time.Minute)

	require.NoError(ck.SetVisibilityTimeUpgradeID(ctx, 101))

	require.NoError(ck.SetEventRecord(ctx, types.NewEventRecord(
		TxHash1, 100, 100, Address1, make([]byte, 1), "1", baseTime.Add(50*time.Minute),
	)))

	for i := uint64(101); i <= 102; i++ {
		rec := types.NewEventRecord(TxHash1, i, i, Address1, make([]byte, 1), "1",
			baseTime.Add(time.Duration(i-58)*time.Minute))
		require.NoError(ck.SetEventRecord(ctx, rec))
		require.NoError(ck.VisibilityHeightByID.Set(ctx, i, 200+i))
	}

	resp, err := s.queryDeterministicStateSyncs(ctx, 500, 100, toTime, 10)
	require.NoError(err)
	require.Len(resp.EventRecords, 2, "eligible post-upgrade events must still be returned")
	require.Equal(uint64(101), resp.EventRecords[0].Id)
	require.Equal(uint64(102), resp.EventRecords[1].Id)
}

// TestGetStateSyncsByTime_DeterministicAcrossLatestState verifies that the
// same query against the latest state always returns the same result regardless of when called.
func (s *KeeperTestSuite) TestGetStateSyncsByTime_DeterministicAcrossLatestState() {
	ctx, ck := s.ctx, s.keeper
	ctx = ctx.WithBlockHeight(10000)
	require := s.Require()

	baseTime := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	require.NoError(ck.SetVisibilityTimeUpgradeID(ctx, 1))

	// Create events 1-3 with visibility heights 100, 101, 102
	for i := uint64(1); i <= 3; i++ {
		rec := types.NewEventRecord(TxHash1, i, i, Address1, make([]byte, 1), "1",
			baseTime.Add(time.Duration(i)*time.Minute))
		require.NoError(ck.SetEventRecord(ctx, rec))
		visHeight := 99 + i
		require.NoError(ck.VisibilityHeightByID.Set(ctx, i, visHeight))
	}

	// Query twice and ensure results are identical (determinism)
	resp1, err := s.queryDeterministicStateSyncs(ctx, 101, 1, baseTime.Add(time.Hour), 10)
	require.NoError(err)
	require.Len(resp1.EventRecords, 2)

	resp2, err := s.queryDeterministicStateSyncs(ctx, 101, 1, baseTime.Add(time.Hour), 10)
	require.NoError(err)
	require.Len(resp2.EventRecords, 2)

	// Verify identical results
	for i := range resp1.EventRecords {
		require.Equal(resp1.EventRecords[i].Id, resp2.EventRecords[i].Id,
			"query must be deterministic: event IDs must match on repeated calls")
	}

	// The immutable visibility_height means even if new blocks are produced,
	// querying at the same height always returns the same set. Simulate by
	// adding more events at higher heights and re-querying at height 101.
	for i := uint64(4); i <= 6; i++ {
		rec := types.NewEventRecord(TxHash1, i, i, Address1, make([]byte, 1), "1",
			baseTime.Add(time.Duration(i)*time.Minute))
		require.NoError(ck.SetEventRecord(ctx, rec))
		visHeight := 99 + i // heights 103, 104, 105
		require.NoError(ck.VisibilityHeightByID.Set(ctx, i, visHeight))
	}

	resp3, err := s.queryDeterministicStateSyncs(ctx, 101, 1, baseTime.Add(time.Hour), 10)
	require.NoError(err)
	require.Len(resp3.EventRecords, 2, "same height query returns same count even after new events added")
	require.Equal(uint64(1), resp3.EventRecords[0].Id)
	require.Equal(uint64(2), resp3.EventRecords[1].Id)
}

// TestGetBlockHeightByTime_LargeDataset stores 1000 blocks with 1-second intervals
// and validates correct lookups at various cutoff points. This exercises the O(log N)
// reverse index on a larger dataset to guard against performance regressions.
func (s *KeeperTestSuite) TestGetBlockHeightByTime_LargeDataset() {
	ctx, ck := s.ctx, s.keeper
	require := s.Require()

	const numBlocks = 1000
	baseUnix := int64(1000000)
	baseTime := time.Unix(baseUnix, 0).UTC()

	// Store 1000 blocks: height 1..1000, time baseUnix..baseUnix+999
	for i := int64(1); i <= numBlocks; i++ {
		bt := baseTime.Add(time.Duration(i-1) * time.Second)
		ctx = ctx.WithBlockHeight(i).WithBlockHeader(cmtproto.Header{Time: bt, Height: i})
		require.NoError(ck.StoreBlockTime(ctx))
	}

	// Cutoff at the 500th block's exact time → returns height 500
	// Block 500 has time = baseUnix + 499
	height, err := ck.GetBlockHeightByTime(ctx, baseUnix+499)
	require.NoError(err)
	require.Equal(int64(500), height, "exact cutoff at block 500 should return height 500")

	// Cutoff at block 501's exact time → returns height 501
	// Block 501 has time = baseUnix + 500
	height, err = ck.GetBlockHeightByTime(ctx, baseUnix+500)
	require.NoError(err)
	require.Equal(int64(501), height, "exact cutoff at block 501 should return height 501")

	// Cutoff after the last block (block 1000 has time = baseUnix+999) → returns 1000
	height, err = ck.GetBlockHeightByTime(ctx, baseUnix+1500)
	require.NoError(err)
	require.Equal(int64(1000), height, "cutoff after all blocks should return the last height")

	// Cutoff before the first block (block 1 has time = baseUnix) → error
	_, err = ck.GetBlockHeightByTime(ctx, baseUnix-1)
	require.Error(err, "cutoff before first block should return error")
	require.Contains(err.Error(), "no block found")
}

// TestGetBlockHeightByTime_TieBreakingLargeScale stores 100 blocks where blocks 50-59
// share the same unix timestamp (simulating sub-second block production). Verifies that
// a cutoff at the shared timestamp returns height 59 (the greatest height wins ties).
func (s *KeeperTestSuite) TestGetBlockHeightByTime_TieBreakingLargeScale() {
	ctx, ck := s.ctx, s.keeper
	require := s.Require()

	baseTime := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	sharedTime := baseTime.Add(50 * time.Second) // unix second for blocks 50-59

	// Store 100 blocks: heights 1-100
	for i := int64(1); i <= 100; i++ {
		var bt time.Time
		if i >= 50 && i <= 59 {
			// Blocks 50-59 all share the same timestamp
			bt = sharedTime
		} else if i < 50 {
			bt = baseTime.Add(time.Duration(i) * time.Second)
		} else {
			// Blocks 60+ resume at sharedTime + (i-59) seconds
			bt = sharedTime.Add(time.Duration(i-59) * time.Second)
		}
		ctx = ctx.WithBlockHeight(i).WithBlockHeader(cmtproto.Header{Time: bt, Height: i})
		require.NoError(ck.StoreBlockTime(ctx))
	}

	// Cutoff at the shared timestamp → should return height 59 (the greatest height with that time)
	height, err := ck.GetBlockHeightByTime(ctx, sharedTime.Unix())
	require.NoError(err)
	require.Equal(int64(59), height, "tie-breaking should return greatest height (59) among blocks 50-59")
}

// TestGetBlockHeightByTime_ActivationBoundary tests the scenario where block time
// indexing starts at an activation height (e.g., 128). Querying with a cutoff before
// any indexed blocks should error; querying at the first indexed block's time should succeed.
func (s *KeeperTestSuite) TestGetBlockHeightByTime_ActivationBoundary() {
	ctx, ck := s.ctx, s.keeper
	require := s.Require()

	activationHeight := int64(128)
	baseTime := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

	// Store block times starting at activation height (128-132)
	for i := int64(0); i < 5; i++ {
		h := activationHeight + i
		bt := baseTime.Add(time.Duration(i*3) * time.Second)
		ctx = ctx.WithBlockHeight(h).WithBlockHeader(cmtproto.Header{Time: bt, Height: h})
		require.NoError(ck.StoreBlockTime(ctx))
	}

	// Query with cutoff before the first indexed block's time → error
	_, err := ck.GetBlockHeightByTime(ctx, baseTime.Add(-time.Second).Unix())
	require.Error(err, "cutoff before activation height's block time should error")
	require.Contains(err.Error(), "no block found")

	// Query at exactly the first indexed block's time → returns activation height
	height, err := ck.GetBlockHeightByTime(ctx, baseTime.Unix())
	require.NoError(err)
	require.Equal(activationHeight, height, "cutoff at first indexed block should return activation height")

	// Query at the second indexed block's time → returns activation height + 1
	height, err = ck.GetBlockHeightByTime(ctx, baseTime.Add(3*time.Second).Unix())
	require.NoError(err)
	require.Equal(activationHeight+1, height)
}

// TestGetStateSyncsByTime_ContiguousIDs stores 10 post-upgrade events where
// events 1-7 have visibility_height=100 and events 8-10 have visibility_height=200.
// Querying at height 150 should return only events 1-7.
func (s *KeeperTestSuite) TestGetStateSyncsByTime_ContiguousIDs() {
	ctx, ck := s.ctx, s.keeper
	ctx = ctx.WithBlockHeight(10000)
	require := s.Require()

	baseTime := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	require.NoError(ck.SetVisibilityTimeUpgradeID(ctx, 1))

	// Create 10 events
	for i := uint64(1); i <= 10; i++ {
		rec := types.NewEventRecord(TxHash1, i, i, Address1, make([]byte, 1), "1",
			baseTime.Add(time.Duration(i)*time.Minute))
		require.NoError(ck.SetEventRecord(ctx, rec))

		var visHeight uint64
		if i <= 7 {
			visHeight = 100
		} else {
			visHeight = 200
		}
		require.NoError(ck.VisibilityHeightByID.Set(ctx, i, visHeight))
	}

	resp, err := s.queryDeterministicStateSyncs(ctx, 150, 1, baseTime.Add(time.Hour), 20)
	require.NoError(err)
	require.Len(resp.EventRecords, 7, "should return events 1-7")

	// Verify IDs are strictly contiguous
	for i, rec := range resp.EventRecords {
		require.Equal(uint64(i+1), rec.Id, "event IDs must be strictly contiguous")
	}
}

// TestGetStateSyncsByTime_PreAndPostUpgradeMix stores pre-upgrade events
// (record_time based) and post-upgrade events (visibility_height based). Verifies
// that pre-upgrade events use record_time filtering and post-upgrade events use
// visibility_height filtering.
func (s *KeeperTestSuite) TestGetStateSyncsByTime_PreAndPostUpgradeMix() {
	ctx, ck := s.ctx, s.keeper
	ctx = ctx.WithBlockHeight(10000)
	require := s.Require()

	baseTime := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)

	// Set upgrade boundary at event ID 4
	require.NoError(ck.SetVisibilityTimeUpgradeID(ctx, 4))

	// Events 1-3: pre-upgrade (legacy), filtered by record_time
	for i := uint64(1); i <= 3; i++ {
		rec := types.NewEventRecord(TxHash1, i, i, Address1, make([]byte, 1), "1",
			baseTime.Add(time.Duration(i)*time.Minute))
		require.NoError(ck.SetEventRecord(ctx, rec))
	}

	// Events 4-6: post-upgrade, filtered by visibility_height
	for i := uint64(4); i <= 6; i++ {
		rec := types.NewEventRecord(TxHash1, i, i, Address1, make([]byte, 1), "1",
			baseTime.Add(time.Duration(i)*time.Minute))
		require.NoError(ck.SetEventRecord(ctx, rec))

		visHeight := 50 + i // heights 54, 55, 56
		require.NoError(ck.VisibilityHeightByID.Set(ctx, i, visHeight))
	}

	// First test that pre-upgrade record_time filtering works by restricting to_time.
	resp3, err := s.queryDeterministicStateSyncs(ctx, 100, 1, baseTime.Add(2*time.Minute+30*time.Second), 20)
	require.NoError(err)
	require.Len(resp3.EventRecords, 2, "restricting to_time should filter out pre-upgrade event 3")
	require.Equal(int64(100), resp3.HeimdallHeight, "first query should resolve the requested height")
	require.Equal(uint64(1), resp3.EventRecords[0].Id)
	require.Equal(uint64(2), resp3.EventRecords[1].Id)

	// Then test that post-upgrade filtering works by querying at a lower height
	// that excludes event 6 (vis_height=56).
	resp2, err := s.queryDeterministicStateSyncs(ctx, 55, 1, baseTime.Add(time.Hour), 20)
	require.NoError(err)
	require.Len(resp2.EventRecords, 5, "should return events 1-5 (3 pre-upgrade + 2 post-upgrade)")
	require.Equal(int64(55), resp2.HeimdallHeight, "second query should resolve the requested lower height")
	for i, rec := range resp2.EventRecords {
		require.Equal(uint64(i+1), rec.Id, "IDs must be contiguous")
	}

	// Use a later cutoff than the previous query so the reverse index resolves the
	// freshly-added height 100 instead of reusing height 56 from the earlier call.
	resp, err := s.queryDeterministicStateSyncs(ctx, 100, 1, baseTime.Add(2*time.Hour), 20)
	require.NoError(err)
	require.Len(resp.EventRecords, 6, "should return all 6 events (3 pre-upgrade + 3 post-upgrade)")
	require.Equal(int64(100), resp.HeimdallHeight, "final query should resolve the intended height")

	// Verify events 1-3 are pre-upgrade (id < upgradeID=4)
	for i := 0; i < 3; i++ {
		require.Equal(uint64(i+1), resp.EventRecords[i].Id)
		require.True(resp.EventRecords[i].Id < 4, "events 1-3 are pre-upgrade (record_time filtered)")
	}

	// Verify events 4-6 are post-upgrade (id >= upgradeID=4)
	for i := 3; i < 6; i++ {
		require.Equal(uint64(i+1), resp.EventRecords[i].Id)
		require.True(resp.EventRecords[i].Id >= 4, "events 4-6 are post-upgrade (visibility_height filtered)")
	}
}

// TestStoreBlockTime_WritesReverseIndex verifies that StoreBlockTime correctly writes
// the reverse index ((time, height) → height), and that GetBlockHeightByTime uses
// the reverse index to return correct results.
func (s *KeeperTestSuite) TestStoreBlockTime_WritesReverseIndex() {
	ctx, ck := s.ctx, s.keeper
	require := s.Require()

	blocks := []struct {
		height int64
		time   time.Time
	}{
		{100, time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)},
		{101, time.Date(2026, 3, 15, 10, 0, 3, 0, time.UTC)},
		{102, time.Date(2026, 3, 15, 10, 0, 6, 0, time.UTC)},
	}

	for _, b := range blocks {
		ctx = ctx.WithBlockHeight(b.height).WithBlockHeader(cmtproto.Header{Time: b.time, Height: b.height})
		require.NoError(ck.StoreBlockTime(ctx))
	}

	// Verify reverse index (BlockTimeReverseIndex): (time, height) → height entries exist
	for _, b := range blocks {
		storedHeight, err := ck.BlockTimeReverseIndex.Get(ctx,
			collections.Join(uint64(b.time.Unix()), uint64(b.height)))
		require.NoError(err, "reverse index should have entry for (time=%d, height=%d)",
			b.time.Unix(), b.height)
		require.Equal(uint64(b.height), storedHeight,
			"reverse index height mismatch for (time=%d, height=%d)", b.time.Unix(), b.height)
	}

	// Verify GetBlockHeightByTime returns correct results using the reverse index
	// Cutoff at block 101's time → should return 101
	height, err := ck.GetBlockHeightByTime(ctx, blocks[1].time.Unix())
	require.NoError(err)
	require.Equal(int64(101), height)

	// Cutoff between block 101 and 102 → should return 101
	height, err = ck.GetBlockHeightByTime(ctx, blocks[1].time.Unix()+1)
	require.NoError(err)
	require.Equal(int64(101), height)

	// Cutoff at block 102's time → should return 102
	height, err = ck.GetBlockHeightByTime(ctx, blocks[2].time.Unix())
	require.NoError(err)
	require.Equal(int64(102), height)
}
