package keeper_test

import (
	"time"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/0xPolygon/heimdall-v2/x/clerk/types"
)

// TestRecordListWithTime_Deterministic_PreHFEventAboveBoundary covers an
// out-of-order event crossing the visibility-time activation height: a lower-ID
// event is processed after the HF while a higher-ID event was already
// processed before it. The pre-HF event has no visibility_height and is not
// pending, so it is classified as legacy and filtered on record_time. It must
// still be delivered rather than permanently dropped — otherwise bor's strict
// lastStateId+1 consumption freezes every later state sync.
func (s *KeeperTestSuite) TestRecordListWithTime_Deterministic_PreHFEventAboveBoundary() {
	ctx, ck := s.ctx, s.keeper
	require := s.Require()

	base := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	toTime := base.Add(2 * time.Hour)
	resolvedHeight := int64(100)

	setRecord := func(id uint64) {
		rec := types.NewEventRecord(TxHash1, id, id, Address1, make([]byte, 1), "1", base)
		rec.RecordTime = rec.RecordTime.UTC()
		require.NoError(ck.SetEventRecord(ctx, rec))
	}

	// Pre-HF processing (no visibility_height assigned): 1,2,3 in order; 5 out of
	// order (event 4 lagged behind on the bridge retry path).
	setRecord(1)
	setRecord(2)
	setRecord(3)
	setRecord(5)

	for _, id := range []uint64{1, 2, 3, 5} {
		_, err := ck.GetVisibilityHeightForEvent(ctx, id)
		require.Error(err, "pre-HF event %d must have no visibility_height", id)
	}

	// HF activates; event 4 (the hole) is processed AFTER the HF, mirroring
	// PostHandleMsgEventRecord: store and mark pending.
	setRecord(4)
	assignHeight := int64(10)
	assignCtx := ctx.WithBlockHeight(assignHeight).
		WithBlockHeader(cmtproto.Header{Time: base.Add(-time.Hour), Height: assignHeight})
	require.NoError(ck.AddPendingVisibilityEvent(assignCtx, 4))

	// Next block PreBlocker assigns visibility_height to pending event 4.
	require.NoError(ck.ProcessPendingVisibilityEvents(assignCtx))
	vh4, err := ck.GetVisibilityHeightForEvent(ctx, 4)
	require.NoError(err)
	require.Equal(uint64(assignHeight), vh4)

	resp, err := s.queryDeterministicStateSyncs(ctx, resolvedHeight, 1, toTime, 50)
	require.NoError(err)

	got := make(map[uint64]bool)
	for _, r := range resp.EventRecords {
		got[r.Id] = true
	}

	for id := uint64(1); id <= 5; id++ {
		require.True(got[id], "event %d must be delivered (no permanent gap)", id)
	}
}

// TestRecordListWithTime_Deterministic_InOrderControl is the control: with no
// cross-HF hole, every pre-HF event was processed in order, so the same query
// returns the full contiguous set.
func (s *KeeperTestSuite) TestRecordListWithTime_Deterministic_InOrderControl() {
	ctx, ck := s.ctx, s.keeper
	require := s.Require()

	base := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	toTime := base.Add(2 * time.Hour)
	resolvedHeight := int64(100)

	setRecord := func(id uint64) {
		rec := types.NewEventRecord(TxHash1, id, id, Address1, make([]byte, 1), "1", base)
		rec.RecordTime = rec.RecordTime.UTC()
		require.NoError(ck.SetEventRecord(ctx, rec))
	}

	for id := uint64(1); id <= 5; id++ {
		setRecord(id)
	}

	// First post-HF event is 6 -> gets a visibility_height; pre-HF events 1-5
	// are all legacy (no visibility_height, not pending).
	setRecord(6)
	assignHeight := int64(10)
	assignCtx := ctx.WithBlockHeight(assignHeight).
		WithBlockHeader(cmtproto.Header{Time: base.Add(-time.Hour), Height: assignHeight})
	require.NoError(ck.AddPendingVisibilityEvent(assignCtx, 6))
	require.NoError(ck.ProcessPendingVisibilityEvents(assignCtx))

	resp, err := s.queryDeterministicStateSyncs(ctx, resolvedHeight, 1, toTime, 50)
	require.NoError(err)

	got := make(map[uint64]bool)
	for _, r := range resp.EventRecords {
		got[r.Id] = true
	}
	for id := uint64(1); id <= 6; id++ {
		require.True(got[id], "control: event %d should be delivered with no cross-HF hole", id)
	}
}
