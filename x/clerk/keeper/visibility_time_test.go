package keeper_test

import (
	"context"
	"time"

	"cosmossdk.io/collections"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/sidetxs"
	clerkKeeper "github.com/0xPolygon/heimdall-v2/x/clerk/keeper"
	"github.com/0xPolygon/heimdall-v2/x/clerk/types"
)

// TestProcessPendingVisibilityEvents verifies that pending events get assigned
// visibility_time and are cleared from the pending collection.
func (s *KeeperTestSuite) TestProcessPendingVisibilityEvents() {
	ctx, ck := s.ctx, s.keeper
	require := s.Require()

	blockTime := time.Date(2026, 1, 12, 17, 5, 40, 0, time.UTC)
	ctx = ctx.WithBlockHeader(cmtproto.Header{Time: blockTime})

	// Store 3 events and add them as pending
	for i := uint64(1); i <= 3; i++ {
		rec := types.NewEventRecord(TxHash1, i, i, Address1, make([]byte, 1), "1", blockTime.Add(-time.Minute))
		rec.RecordTime = rec.RecordTime.UTC()
		require.NoError(ck.SetEventRecord(ctx, rec))
		require.NoError(ck.AddPendingVisibilityEvent(ctx, i))
	}

	// Verify events are pending (no visibility time yet)
	for i := uint64(1); i <= 3; i++ {
		_, err := ck.GetVisibilityTimeForEvent(ctx, i)
		require.Error(err, "event %d should not have visibility time before processing", i)
	}

	// Process pending events
	require.NoError(ck.ProcessPendingVisibilityEvents(ctx))

	// All events should now have visibility_time = blockTime
	for i := uint64(1); i <= 3; i++ {
		vt, err := ck.GetVisibilityTimeForEvent(ctx, i)
		require.NoError(err, "event %d should have visibility time after processing", i)
		require.True(vt.Equal(blockTime), "event %d visibility_time should equal block time, got %v", i, vt)
	}

	// Pending list should be cleared
	hasPending, _ := ck.PendingVisibilityEvents.Has(ctx, 1)
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

// TestGetRecordListWithTime_PostUpgradeVisibilityTimeFiltering verifies that
// post-upgrade events are filtered by visibility_time, not record_time.
func (s *KeeperTestSuite) TestGetRecordListWithTime_PostUpgradeVisibilityTimeFiltering() {
	ctx, ck, queryClient := s.ctx, s.keeper, s.queryClient
	require := s.Require()

	baseTime := time.Date(2026, 1, 12, 16, 50, 0, 0, time.UTC)

	// Set upgrade boundary at event ID 1
	require.NoError(ck.SetVisibilityTimeUpgradeID(ctx, 1))

	// Create an event with a record_time that falls within the query window
	rec := types.NewEventRecord(TxHash1, 1, 1, Address1, make([]byte, 1), "1",
		baseTime.Add(8*time.Minute+14*time.Second))
	require.NoError(ck.SetEventRecord(ctx, rec))

	// Set visibility_time to smth after the query to_time
	visTime := baseTime.Add(15*time.Minute + 40*time.Second)
	require.NoError(ck.SetEventRecordWithVisibilityTime(ctx, 1, visTime))

	// Query with to_time after record_time but before visibility_time
	// The event's record_time < to_time, but visibility_time >= to_time, hence the event should not be returned
	toTime := baseTime.Add(8*time.Minute + 30*time.Second)
	req := &types.RecordListWithTimeRequest{
		FromId:     1,
		ToTime:     toTime,
		Pagination: query.PageRequest{Limit: 10, Key: []byte{0x00}},
	}
	res, err := queryClient.GetRecordListWithTime(ctx, req)
	require.NoError(err)
	require.Empty(res.EventRecords, "event should be excluded because visibility_time >= to_time")
}

// TestGetRecordListWithTime_HaltSimulation simulates the scenario of a Heimdall halt,
// where an event has record_time within the query window but visibility_time outside it.
func (s *KeeperTestSuite) TestGetRecordListWithTime_HaltSimulation() {
	ctx, ck, queryClient := s.ctx, s.keeper, s.queryClient
	require := s.Require()

	recordTime := time.Date(2026, 1, 12, 16, 58, 14, 0, time.UTC)
	visibilityTime := time.Date(2026, 1, 12, 17, 5, 40, 0, time.UTC)
	borToTime := time.Date(2026, 1, 12, 16, 58, 30, 0, time.UTC)

	require.NoError(ck.SetVisibilityTimeUpgradeID(ctx, 3131120))

	rec := types.NewEventRecord(TxHash1, 1, 3131120, Address1, make([]byte, 1), "1", recordTime)
	require.NoError(ck.SetEventRecord(ctx, rec))
	require.NoError(ck.SetEventRecordWithVisibilityTime(ctx, 3131120, visibilityTime))

	// Query during heimdall halt, where to_time < visibility_time, hence the event is excluded
	req := &types.RecordListWithTimeRequest{
		FromId:     3131120,
		ToTime:     borToTime,
		Pagination: query.PageRequest{Limit: 10, Key: []byte{0x00}},
	}
	res, err := queryClient.GetRecordListWithTime(ctx, req)
	require.NoError(err)
	require.Empty(res.EventRecords, "halt: event should be excluded")

	// Query from history gives now the same result, proving determinism
	res2, err := queryClient.GetRecordListWithTime(ctx, req)
	require.NoError(err)
	require.Empty(res2.EventRecords, "history: same query should return same result")

	// After the halt is resolved, the query with to_time > visibility_time should ensure the event is included
	postHaltToTime := time.Date(2026, 1, 12, 17, 10, 0, 0, time.UTC)
	reqAfterHalt := &types.RecordListWithTimeRequest{
		FromId:     3131120,
		ToTime:     postHaltToTime,
		Pagination: query.PageRequest{Limit: 10, Key: []byte{0x00}},
	}
	resAfterHalt, err := queryClient.GetRecordListWithTime(ctx, reqAfterHalt)
	require.NoError(err)
	require.Len(resAfterHalt.EventRecords, 1, "after halt: event should be returned")
	require.Equal(uint64(3131120), resAfterHalt.EventRecords[0].Id)
}

// TestGetRecordListWithTime_PendingEventsExcluded verifies that
// the query does not return the events in the pending list (no visibility_time yet).
func (s *KeeperTestSuite) TestGetRecordListWithTime_PendingEventsExcluded() {
	ctx, ck, queryClient := s.ctx, s.keeper, s.queryClient
	require := s.Require()

	now := time.Now().UTC()

	require.NoError(ck.SetVisibilityTimeUpgradeID(ctx, 1))

	// Create an event with record_time in the past
	rec := types.NewEventRecord(TxHash1, 1, 1, Address1, make([]byte, 1), "1",
		now.Add(-5*time.Minute))
	require.NoError(ck.SetEventRecord(ctx, rec))

	// Add to pending but don't set visibility time
	require.NoError(ck.AddPendingVisibilityEvent(ctx, 1))

	req := &types.RecordListWithTimeRequest{
		FromId:     1,
		ToTime:     now,
		Pagination: query.PageRequest{Limit: 10, Key: []byte{0x00}},
	}
	res, err := queryClient.GetRecordListWithTime(ctx, req)
	require.NoError(err)
	require.Empty(res.EventRecords, "pending events should not be returned")
}

// TestGetRecordListWithTime_HybridQuery verifies the hybrid path where some events
// are pre-upgrade (filtered by record_time) and others are post-upgrade (filtered
// by visibility_time).
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

	// Events 3-4: post-upgrade, both have visibility_time within the query window.
	rec3 := types.NewEventRecord(TxHash1, 3, 3, Address1, make([]byte, 1), "1",
		baseTime.Add(3*time.Minute)) // 16:03
	require.NoError(ck.SetEventRecord(ctx, rec3))
	require.NoError(ck.SetEventRecordWithVisibilityTime(ctx, 3, baseTime.Add(3*time.Minute+3*time.Second)))

	rec4 := types.NewEventRecord(TxHash1, 4, 4, Address1, make([]byte, 1), "1",
		baseTime.Add(4*time.Minute)) // 16:04
	require.NoError(ck.SetEventRecord(ctx, rec4))
	require.NoError(ck.SetEventRecordWithVisibilityTime(ctx, 4, baseTime.Add(4*time.Minute+3*time.Second)))

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

// TestGetRecordListWithTime_HybridQueryHaltAtUpgradeBoundary verifies the hybrid path
// when a halt occurs right at the upgrade boundary: pre-upgrade events returned,
// but the first post-upgrade event's visibility_time exceeds to_time, hence the iteration stops there,
// preserving contiguous IDs.
func (s *KeeperTestSuite) TestGetRecordListWithTime_HybridQueryHaltAtUpgradeBoundary() {
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

	// Event 3: post-upgrade, stored during a halt, hence the visibility_time is 30 minutes later
	rec3 := types.NewEventRecord(TxHash1, 3, 3, Address1, make([]byte, 1), "1",
		baseTime.Add(3*time.Minute))
	require.NoError(ck.SetEventRecord(ctx, rec3))
	require.NoError(ck.SetEventRecordWithVisibilityTime(ctx, 3, baseTime.Add(33*time.Minute)))

	// Event 4: post-upgrade, also stored during the same halt, hence visibility_time also late
	rec4 := types.NewEventRecord(TxHash1, 4, 4, Address1, make([]byte, 1), "1",
		baseTime.Add(4*time.Minute))
	require.NoError(ck.SetEventRecord(ctx, rec4))
	require.NoError(ck.SetEventRecordWithVisibilityTime(ctx, 4, baseTime.Add(33*time.Minute+3*time.Second)))

	// Query with to_time=16:10: events 1,2 returned (legacy); event 3 has vis_time=16:33 -> break
	// the expected result is {1, 2} with contiguous prefixes
	toTime := baseTime.Add(10 * time.Minute)
	req := &types.RecordListWithTimeRequest{
		FromId:     1,
		ToTime:     toTime,
		Pagination: query.PageRequest{Limit: 10, Key: []byte{0x00}},
	}
	res, err := queryClient.GetRecordListWithTime(ctx, req)
	require.NoError(err)
	require.Len(res.EventRecords, 2, "should return only pre-upgrade events")
	require.Equal(uint64(1), res.EventRecords[0].Id)
	require.Equal(uint64(2), res.EventRecords[1].Id)
}

// TestMultipleEventsInSameBlock verifies that multiple events in the same block
// all get the same visibility_time but have distinct composite keys.
func (s *KeeperTestSuite) TestMultipleEventsInSameBlock() {
	ctx, ck := s.ctx, s.keeper
	require := s.Require()

	blockTime := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	ctx = ctx.WithBlockHeader(cmtproto.Header{Time: blockTime})

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

	// All should have the same visibility_time
	for i := uint64(1); i <= 5; i++ {
		vt, err := ck.GetVisibilityTimeForEvent(ctx, i)
		require.NoError(err)
		require.True(vt.Equal(blockTime), "event %d should have visibility_time = blockTime", i)
	}

	// Verify the secondary index has all entries (different composite keys)
	iter, err := ck.RecordsWithVisibilityTime.Iterate(ctx, nil)
	require.NoError(err)
	defer func(iter collections.Iterator[collections.Pair[time.Time, uint64], uint64]) {
		err = iter.Close()
		require.NoError(err)
	}(iter)

	count := 0
	for ; iter.Valid(); iter.Next() {
		count++
	}
	require.Equal(5, count, "should have 5 entries in RecordsWithVisibilityTime")
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

// TestGetRecordListWithTime_PendingAtFirstID verifies that if the first event in
// the query range is pending (no visibility_time), the query returns empty — it does
// not skip to later IDs. This preserves Bor's contiguous-ID invariant.
func (s *KeeperTestSuite) TestGetRecordListWithTime_PendingAtFirstID() {
	ctx, ck, queryClient := s.ctx, s.keeper, s.queryClient
	require := s.Require()

	now := time.Now().UTC()
	require.NoError(ck.SetVisibilityTimeUpgradeID(ctx, 1))

	// Events 1-3: all post-upgrade. Event 1 is pending, events 2-3 have visibility_time.
	for i := uint64(1); i <= 3; i++ {
		rec := types.NewEventRecord(TxHash1, i, i, Address1, make([]byte, 1), "1",
			now.Add(-time.Duration(5-i)*time.Minute))
		require.NoError(ck.SetEventRecord(ctx, rec))
	}
	// Only set visibility_time for events 2 and 3 (the first event remains pending)
	require.NoError(ck.SetEventRecordWithVisibilityTime(ctx, 2, now.Add(-2*time.Minute)))
	require.NoError(ck.SetEventRecordWithVisibilityTime(ctx, 3, now.Add(-1*time.Minute)))

	req := &types.RecordListWithTimeRequest{
		FromId:     1,
		ToTime:     now,
		Pagination: query.PageRequest{Limit: 10, Key: []byte{0x00}},
	}
	res, err := queryClient.GetRecordListWithTime(ctx, req)
	require.NoError(err)
	require.Empty(res.EventRecords, "should return empty: first event is pending, break preserves contiguity")
}

// TestGetRecordListWithTime_PendingGapInMiddle verifies that if event N is pending
// but N+1 has visibility_time, the query stops at N — returning a contiguous prefix.
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
	// Set visibility_time for 1, 2, and 4 — but not 3 (simulating a pending event)
	require.NoError(ck.SetEventRecordWithVisibilityTime(ctx, 1, now.Add(-8*time.Minute)))
	require.NoError(ck.SetEventRecordWithVisibilityTime(ctx, 2, now.Add(-7*time.Minute)))
	// event 3: pending (no visibility_time)
	require.NoError(ck.SetEventRecordWithVisibilityTime(ctx, 4, now.Add(-5*time.Minute)))

	req := &types.RecordListWithTimeRequest{
		FromId:     1,
		ToTime:     now,
		Pagination: query.PageRequest{Limit: 10, Key: []byte{0x00}},
	}
	res, err := queryClient.GetRecordListWithTime(ctx, req)
	require.NoError(err)
	require.Len(res.EventRecords, 2, "should return events 1,2 and stop at pending event 3")
	require.Equal(uint64(1), res.EventRecords[0].Id)
	require.Equal(uint64(2), res.EventRecords[1].Id)
}

// TestGetRecordListWithTime_VisTimeBreakPreservesContiguity verifies that when a
// post-upgrade event has visibility_time >= to_time, the query breaks (not continues),
// ensuring no later events with lower IDs are skipped and returned out of order.
func (s *KeeperTestSuite) TestGetRecordListWithTime_VisTimeBreakPreservesContiguity() {
	ctx, ck, queryClient := s.ctx, s.keeper, s.queryClient
	require := s.Require()

	baseTime := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	require.NoError(ck.SetVisibilityTimeUpgradeID(ctx, 1))

	// Events 1-5: all post-upgrade with monotonically increasing visibility_time
	for i := uint64(1); i <= 5; i++ {
		rec := types.NewEventRecord(TxHash1, i, i, Address1, make([]byte, 1), "1",
			baseTime.Add(time.Duration(i)*time.Minute))
		require.NoError(ck.SetEventRecord(ctx, rec))
		require.NoError(ck.SetEventRecordWithVisibilityTime(ctx, i,
			baseTime.Add(time.Duration(i)*time.Minute+3*time.Second)))
	}

	// Query with to_time between event 3's and event 4's visibility_time
	// Events 1-3 should be returned (vis_time < to_time), event 4 breaks
	toTime := baseTime.Add(4*time.Minute + 2*time.Second)
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

// TestQueryDeterminismAcrossCommittedStates verifies that the query result is
// deterministic for a given committed state, regardless of wall-clock timing.
//
// Scenario: event stored in block H. Block H+1 runs ProcessPendingVisibilityEvents
// and assigns visibility_time = blockTime(H+1). We test that:
//   - At committed state H (before ProcessPendingVisibilityEvents): the query returns empty
//   - At committed state H+1 (after ProcessPendingVisibilityEvents): the query returns the event
//   - Both results are deterministic — re-querying the same state returns the same answer
func (s *KeeperTestSuite) TestQueryDeterminismAcrossCommittedStates() {
	ctx, ck, queryClient := s.ctx, s.keeper, s.queryClient
	require := s.Require()

	require.NoError(ck.SetVisibilityTimeUpgradeID(ctx, 1))

	// Block H: event stored with pending status
	blockHTime := time.Date(2026, 1, 12, 16, 58, 14, 0, time.UTC)
	rec := types.NewEventRecord(TxHash1, 1, 1, Address1, make([]byte, 1), "1", blockHTime)
	require.NoError(ck.SetEventRecord(ctx, rec))
	require.NoError(ck.AddPendingVisibilityEvent(ctx, 1))

	// Simulate a query at committed state H (event pending, no visibility_time)
	// Even with to_time well after record_time, the event is not returned
	queryToTime := time.Date(2026, 1, 12, 17, 10, 0, 0, time.UTC)
	req := &types.RecordListWithTimeRequest{
		FromId:     1,
		ToTime:     queryToTime,
		Pagination: query.PageRequest{Limit: 10, Key: []byte{0x00}},
	}

	// Query 1 at state H: empty
	res1, err := queryClient.GetRecordListWithTime(ctx, req)
	require.NoError(err)
	require.Empty(res1.EventRecords, "state H: event should not be visible")

	// Query 2 at state H (same state, same query): still empty, hence deterministic
	res2, err := queryClient.GetRecordListWithTime(ctx, req)
	require.NoError(err)
	require.Empty(res2.EventRecords, "state H: repeat query must return same result")

	// Block H+1: ProcessPendingVisibilityEvents runs.
	// Simulate propose-to-commit delay: blockTime(H+1) = 17:06:00, but the block
	// doesn't "commit" until 17:12:00. The visibility_time is 17:06:00 regardless
	// of when the block commits — it's the header timestamp.
	blockH1Time := time.Date(2026, 1, 12, 17, 6, 0, 0, time.UTC)
	ctx = ctx.WithBlockHeader(cmtproto.Header{Time: blockH1Time})
	require.NoError(ck.ProcessPendingVisibilityEvents(ctx))

	// Query 3 at state H+1 (after ProcessPendingVisibilityEvents committed):
	// visibility_time = 17:06:00 < queryToTime(17:10:00) -> event returned
	res3, err := queryClient.GetRecordListWithTime(ctx, req)
	require.NoError(err)
	require.Len(res3.EventRecords, 1, "state H+1: event should be visible")

	// Query 4 at state H+1: same result, hence deterministic
	res4, err := queryClient.GetRecordListWithTime(ctx, req)
	require.NoError(err)
	require.Len(res4.EventRecords, 1, "state H+1: repeat query must return same result")

	// Edge case: query with to_time between record_time and visibility_time
	// to_time = 17:00:00 — after record_time (16:58:14) but before visibility_time (17:06:00)
	// With old code (record_time filtering): event would be returned
	// With new code (visibility_time filtering): event not returned → correct
	edgeToTime := time.Date(2026, 1, 12, 17, 0, 0, 0, time.UTC)
	edgeReq := &types.RecordListWithTimeRequest{
		FromId:     1,
		ToTime:     edgeToTime,
		Pagination: query.PageRequest{Limit: 10, Key: []byte{0x00}},
	}
	edgeRes, err := queryClient.GetRecordListWithTime(ctx, edgeReq)
	require.NoError(err)
	require.Empty(edgeRes.EventRecords,
		"to_time between record_time and visibility_time: event must not be returned")
}

// TestEndToEndVisibilityTimeLifecycle simulates the full event lifecycle:
// Block H: event created → Block H+1: stored with pending → Block H+2: visibility_time assigned
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

	// Event is stored but not yet queryable via visibility_time
	toTime := blockH1Time.Add(time.Minute)
	req := &types.RecordListWithTimeRequest{
		FromId:     1000,
		ToTime:     toTime,
		Pagination: query.PageRequest{Limit: 10, Key: []byte{0x00}},
	}
	res, err := queryClient.GetRecordListWithTime(ctx, req)
	require.NoError(err)
	require.Empty(res.EventRecords, "event should not be visible yet (pending)")

	// Block H+2: PreBlocker processes pending → assigns visibility_time
	blockH2Time := time.Date(2026, 3, 15, 10, 0, 3, 0, time.UTC) // ~3s later
	ctx = ctx.WithBlockHeight(101).WithBlockHeader(cmtproto.Header{Time: blockH2Time, Height: 101})

	require.NoError(ck.ProcessPendingVisibilityEvents(ctx))

	// Now the event should be queryable (visibility_time = blockH2Time < toTime)
	// Need to_time > blockH2Time for the event to appear
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

	// Verify visibility_time was set correctly
	vt, err := ck.GetVisibilityTimeForEvent(ctx, 1000)
	require.NoError(err)
	require.True(vt.Equal(blockH2Time))
}

// TestStoreBlockTime verifies that StoreBlockTime stores [height, blockTime] mappings
// and that they can be retrieved via the BlockTimeIndex.
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

	// Verify each mapping exists
	for _, tt := range times {
		storedTime, err := ck.BlockTimeIndex.Get(ctx, uint64(tt.height))
		require.NoError(err)
		require.Equal(uint64(tt.time.Unix()), storedTime)
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

// TestGetRecordListVisibleAtHeight_BasicFlow stores events with different visibility
// heights, queries at a specific height, and verifies correct filtering.
func (s *KeeperTestSuite) TestGetRecordListVisibleAtHeight_BasicFlow() {
	ctx, ck := s.ctx, s.keeper
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

	// Query at height 103 → events 1,2,3 visible (vis heights 101,102,103)
	qs := clerkKeeper.NewQueryServer(&ck)
	resp, err := qs.(interface {
		GetRecordListVisibleAtHeight(context.Context, *types.RecordListVisibleAtHeightRequest) (*types.RecordListVisibleAtHeightResponse, error)
	}).GetRecordListVisibleAtHeight(ctx, &types.RecordListVisibleAtHeightRequest{
		FromId:         1,
		HeimdallHeight: 103,
		ToTime:         baseTime.Add(time.Hour), // generous to_time for legacy compatibility
		Pagination:     query.PageRequest{Limit: 10, Key: []byte{0x00}},
	})
	require.NoError(err)
	require.Len(resp.EventRecords, 3)
	for i, rec := range resp.EventRecords {
		require.Equal(uint64(i+1), rec.Id, "IDs must be contiguous")
	}
}

// TestGetRecordListVisibleAtHeight_HybridQuery mixes pre-upgrade (record_time filtered)
// and post-upgrade (visibility_height filtered) events.
func (s *KeeperTestSuite) TestGetRecordListVisibleAtHeight_HybridQuery() {
	ctx, ck := s.ctx, s.keeper
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

	qs := clerkKeeper.NewQueryServer(&ck)
	resp, err := qs.(interface {
		GetRecordListVisibleAtHeight(context.Context, *types.RecordListVisibleAtHeightRequest) (*types.RecordListVisibleAtHeightResponse, error)
	}).GetRecordListVisibleAtHeight(ctx, &types.RecordListVisibleAtHeightRequest{
		FromId:         1,
		HeimdallHeight: 204, // includes all post-upgrade events
		ToTime:         baseTime.Add(time.Hour),
		Pagination:     query.PageRequest{Limit: 10, Key: []byte{0x00}},
	})
	require.NoError(err)
	require.Len(resp.EventRecords, 4, "should return all 4 events (2 legacy + 2 post-upgrade)")
	for i, rec := range resp.EventRecords {
		require.Equal(uint64(i+1), rec.Id, "IDs must be contiguous")
	}

	// Query at height 203 → only events 1,2,3 (event 4 has vis height 204)
	resp2, err := qs.(interface {
		GetRecordListVisibleAtHeight(context.Context, *types.RecordListVisibleAtHeightRequest) (*types.RecordListVisibleAtHeightResponse, error)
	}).GetRecordListVisibleAtHeight(ctx, &types.RecordListVisibleAtHeightRequest{
		FromId:         1,
		HeimdallHeight: 203,
		ToTime:         baseTime.Add(time.Hour),
		Pagination:     query.PageRequest{Limit: 10, Key: []byte{0x00}},
	})
	require.NoError(err)
	require.Len(resp2.EventRecords, 3, "should return 2 legacy + 1 post-upgrade")
}

// TestGetRecordListVisibleAtHeight_PendingExcluded verifies that post-upgrade events
// without visibility_height (still pending) break iteration (contiguous IDs).
func (s *KeeperTestSuite) TestGetRecordListVisibleAtHeight_PendingExcluded() {
	ctx, ck := s.ctx, s.keeper
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

	qs := clerkKeeper.NewQueryServer(&ck)
	resp, err := qs.(interface {
		GetRecordListVisibleAtHeight(context.Context, *types.RecordListVisibleAtHeightRequest) (*types.RecordListVisibleAtHeightResponse, error)
	}).GetRecordListVisibleAtHeight(ctx, &types.RecordListVisibleAtHeightRequest{
		FromId:         1,
		HeimdallHeight: 200,
		ToTime:         baseTime.Add(time.Hour),
		Pagination:     query.PageRequest{Limit: 10, Key: []byte{0x00}},
	})
	require.NoError(err)
	require.Len(resp.EventRecords, 1, "should return only event 1, break at pending event 2")
	require.Equal(uint64(1), resp.EventRecords[0].Id)
}

// TestGetRecordListVisibleAtHeight_DeterministicAcrossLatestState verifies that the
// same query against the latest state always returns the same result regardless of when called.
func (s *KeeperTestSuite) TestGetRecordListVisibleAtHeight_DeterministicAcrossLatestState() {
	ctx, ck := s.ctx, s.keeper
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

	qs := clerkKeeper.NewQueryServer(&ck)
	req := &types.RecordListVisibleAtHeightRequest{
		FromId:         1,
		HeimdallHeight: 101, // events 1,2 visible
		ToTime:         baseTime.Add(time.Hour),
		Pagination:     query.PageRequest{Limit: 10, Key: []byte{0x00}},
	}

	// Query twice and ensure results are identical (determinism)
	resp1, err := qs.(interface {
		GetRecordListVisibleAtHeight(context.Context, *types.RecordListVisibleAtHeightRequest) (*types.RecordListVisibleAtHeightResponse, error)
	}).GetRecordListVisibleAtHeight(ctx, req)
	require.NoError(err)
	require.Len(resp1.EventRecords, 2)

	resp2, err := qs.(interface {
		GetRecordListVisibleAtHeight(context.Context, *types.RecordListVisibleAtHeightRequest) (*types.RecordListVisibleAtHeightResponse, error)
	}).GetRecordListVisibleAtHeight(ctx, req)
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

	resp3, err := qs.(interface {
		GetRecordListVisibleAtHeight(context.Context, *types.RecordListVisibleAtHeightRequest) (*types.RecordListVisibleAtHeightResponse, error)
	}).GetRecordListVisibleAtHeight(ctx, req)
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

	// Cutoff between block 500 and 501 → should still return 500 (the greatest height with time <= cutoff)
	height, err = ck.GetBlockHeightByTime(ctx, baseUnix+499)
	require.NoError(err)
	require.Equal(int64(500), height, "cutoff between block 500 and 501 should return 500")

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

// TestGetRecordListVisibleAtHeight_ContiguousIDs stores 10 post-upgrade events where
// events 1-7 have visibility_height=100 and events 8-10 have visibility_height=200.
// Querying at height 150 should return events 1-7 (contiguous) and break at event 8.
func (s *KeeperTestSuite) TestGetRecordListVisibleAtHeight_ContiguousIDs() {
	ctx, ck := s.ctx, s.keeper
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

	// Query at height 150 → should return events 1-7 (vis_height 100 <= 150), break at event 8 (vis_height 200 > 150)
	qs := clerkKeeper.NewQueryServer(&ck)
	resp, err := qs.(interface {
		GetRecordListVisibleAtHeight(context.Context, *types.RecordListVisibleAtHeightRequest) (*types.RecordListVisibleAtHeightResponse, error)
	}).GetRecordListVisibleAtHeight(ctx, &types.RecordListVisibleAtHeightRequest{
		FromId:         1,
		HeimdallHeight: 150,
		ToTime:         baseTime.Add(time.Hour),
		Pagination:     query.PageRequest{Limit: 20, Key: []byte{0x00}},
	})
	require.NoError(err)
	require.Len(resp.EventRecords, 7, "should return events 1-7")

	// Verify IDs are strictly contiguous
	for i, rec := range resp.EventRecords {
		require.Equal(uint64(i+1), rec.Id, "event IDs must be strictly contiguous")
	}
}

// TestGetRecordListVisibleAtHeight_PreAndPostUpgradeMix stores pre-upgrade events
// (record_time based) and post-upgrade events (visibility_height based). Verifies
// that pre-upgrade events use record_time filtering and post-upgrade events use
// visibility_height filtering.
func (s *KeeperTestSuite) TestGetRecordListVisibleAtHeight_PreAndPostUpgradeMix() {
	ctx, ck := s.ctx, s.keeper
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

	// Query: height=100 (includes all post-upgrade events), to_time includes all the pre-upgrade events
	qs := clerkKeeper.NewQueryServer(&ck)
	resp, err := qs.(interface {
		GetRecordListVisibleAtHeight(context.Context, *types.RecordListVisibleAtHeightRequest) (*types.RecordListVisibleAtHeightResponse, error)
	}).GetRecordListVisibleAtHeight(ctx, &types.RecordListVisibleAtHeightRequest{
		FromId:         1,
		HeimdallHeight: 100,
		ToTime:         baseTime.Add(time.Hour), // generous to_time for all pre-upgrade events
		Pagination:     query.PageRequest{Limit: 20, Key: []byte{0x00}},
	})
	require.NoError(err)
	require.Len(resp.EventRecords, 6, "should return all 6 events (3 pre-upgrade + 3 post-upgrade)")

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

	// Now test that post-upgrade filtering actually works by querying at a lower height
	// that excludes event 6 (vis_height=56)
	resp2, err := qs.(interface {
		GetRecordListVisibleAtHeight(context.Context, *types.RecordListVisibleAtHeightRequest) (*types.RecordListVisibleAtHeightResponse, error)
	}).GetRecordListVisibleAtHeight(ctx, &types.RecordListVisibleAtHeightRequest{
		FromId:         1,
		HeimdallHeight: 55, // includes events 4 (54) and 5 (55), excludes event 6 (56)
		ToTime:         baseTime.Add(time.Hour),
		Pagination:     query.PageRequest{Limit: 20, Key: []byte{0x00}},
	})
	require.NoError(err)
	require.Len(resp2.EventRecords, 5, "should return events 1-5 (3 pre-upgrade + 2 post-upgrade)")
	for i, rec := range resp2.EventRecords {
		require.Equal(uint64(i+1), rec.Id, "IDs must be contiguous")
	}

	// Also test that pre-upgrade record_time filtering works by restricting to_time
	resp3, err := qs.(interface {
		GetRecordListVisibleAtHeight(context.Context, *types.RecordListVisibleAtHeightRequest) (*types.RecordListVisibleAtHeightResponse, error)
	}).GetRecordListVisibleAtHeight(ctx, &types.RecordListVisibleAtHeightRequest{
		FromId:         1,
		HeimdallHeight: 100,
		ToTime:         baseTime.Add(2*time.Minute + 30*time.Second), // includes events 1,2 but not event 3 (record_time = baseTime+3min)
		Pagination:     query.PageRequest{Limit: 20, Key: []byte{0x00}},
	})
	require.NoError(err)
	require.Len(resp3.EventRecords, 2, "restricting to_time should filter out pre-upgrade event 3 and break")
	require.Equal(uint64(1), resp3.EventRecords[0].Id)
	require.Equal(uint64(2), resp3.EventRecords[1].Id)
}

// TestStoreBlockTime_WritesReverseIndex verifies that StoreBlockTime correctly writes
// both the forward index (height → time) and the reverse index ((time, height) → height),
// and that GetBlockHeightByTime uses the reverse index to return correct results.
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

	// Verify forward index (BlockTimeIndex): height → time entries exist
	for _, b := range blocks {
		storedTime, err := ck.BlockTimeIndex.Get(ctx, uint64(b.height))
		require.NoError(err, "forward index should have entry for height %d", b.height)
		require.Equal(uint64(b.time.Unix()), storedTime,
			"forward index time mismatch for height %d", b.height)
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
