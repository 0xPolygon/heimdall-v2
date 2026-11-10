package processor

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"cosmossdk.io/log"
	"github.com/RichardKnop/machinery/v1/tasks"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/metrics"
)

func TestPendingRecords(t *testing.T) {
	// Not parallel: asserts an exact change of the global expiry counter.

	start := time.Unix(1_793_846_312, 0)

	var p pendingRecords
	require.Equal(t, uint64(50002), p.touch(50002, "", start))
	require.Equal(t, uint64(49992), p.touch(49992, "", start))
	require.Equal(t, uint64(49992), p.touch(50010, "", start))

	p.done(49992)
	require.Equal(t, uint64(50002), p.touch(50010, "", start))

	// 50002 stops retrying; exactly at the TTL it still counts, past it it is gone.
	require.Equal(t, uint64(50002), p.touch(50010, "", start.Add(pendingRecordTTL)))
	expired := testutil.ToFloat64(metrics.ClerkPendingExpired)
	require.Equal(t, uint64(50010), p.touch(50010, "", start.Add(pendingRecordTTL+time.Second)))
	require.Equal(t, expired+1, testutil.ToFloat64(metrics.ClerkPendingExpired))
	require.Len(t, p.leases.entries, 1)

	p.done(50010)
	p.done(12345)
	require.Equal(t, uint64(60000), p.touch(60000, "", start.Add(pendingRecordTTL+time.Second)))

	// Expiry found by a peek is counted too; one in the released set is not.
	expired = testutil.ToFloat64(metrics.ClerkPendingExpired)
	_, _, ok := p.lowest(start.Add(2*pendingRecordTTL + 2*time.Second))
	require.False(t, ok)
	require.Equal(t, expired+1, testutil.ToFloat64(metrics.ClerkPendingExpired))

	var r releasedRecords
	r.touch(49992, "a", start)
	r.touch(49993, "a", start.Add(pendingRecordTTL+time.Second))
	require.Equal(t, expired+1, testutil.ToFloat64(metrics.ClerkPendingExpired))
	require.Len(t, r.leases.entries, 1)

	// A seed has no task to lose, also when seedFrontier adds it.
	expired = testutil.ToFloat64(metrics.ClerkPendingExpired)
	p.seed(80000, start)
	_, _, ok = p.lowest(start.Add(pendingRecordTTL + time.Second))
	require.False(t, ok)
	require.Equal(t, expired, testutil.ToFloat64(metrics.ClerkPendingExpired))

	n, rest, comet := newBurstNode(t)
	n.recordCount = 89999
	cp, _ := newBurstClerkProcessor(t, n, rest, comet)
	cp.seeded.Store(false)
	cp.seedFrontier(context.Background(), start)
	_, _, ok = cp.pending.lowest(start.Add(pendingRecordTTL + time.Second))
	require.False(t, ok)
	require.Equal(t, expired, testutil.ToFloat64(metrics.ClerkPendingExpired))

	// Redeliveries of a done ID reuse its heap node.
	var redelivered idLeases
	for range 10 {
		redelivered.touch(90000, "", start, nil)
		redelivered.touch(90001, "", start, nil)
		redelivered.done(90001)
	}
	require.Len(t, redelivered.ids, 2)
	_, _, ok = redelivered.lowest(start, nil)
	require.True(t, ok)
	redelivered.done(90000)
	_, _, ok = redelivered.lowest(start, nil)
	require.False(t, ok)
	require.Empty(t, redelivered.ids)
	require.Empty(t, redelivered.queued, "a popped ID can be queued again")
	redelivered.touch(90000, "", start, nil)
	require.Len(t, redelivered.ids, 1)

	// A retry refreshes its lease without growing the heap.
	var l idLeases
	for range 10 {
		l.touch(70000, "", start, nil)
	}
	require.Len(t, l.ids, 1)
}

func TestMempoolSizeDoesNotWaitForAnotherRead(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_793_846_312, 0)
	started, release := make(chan struct{}), make(chan struct{})
	reads := 0
	m := &mempoolSize{now: func() time.Time { return now }}
	m.read = func(any) (int, error) {
		reads++
		if reads == 2 {
			close(started)
			<-release
		}
		return 400 * reads, nil
	}

	_, _, err := m.size(nil)
	require.NoError(t, err)

	done := make(chan struct{})
	go func() {
		defer close(done)
		now = now.Add(mempoolCountTTL)
		_, _, _ = m.size(nil)
	}()
	<-started

	txs, fresh, err := m.size(nil)
	require.NoError(t, err)
	require.False(t, fresh)
	require.Equal(t, 400, txs, "the last size read while another read runs")

	close(release)
	<-done
	m.mu.Lock()
	require.Equal(t, 800, m.txs)
	require.False(t, m.reading)
	m.mu.Unlock()
}

func TestMempoolSizeServesTheLastSizeAfterForget(t *testing.T) {
	t.Parallel()

	started, release := make(chan struct{}), make(chan struct{})
	reads := 0
	m := &mempoolSize{}
	m.read = func(any) (int, error) {
		reads++
		if reads == 2 {
			close(started)
			<-release
		}
		return 400 * reads, nil
	}

	_, _, err := m.size(nil)
	require.NoError(t, err)
	m.forget()

	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _, _ = m.size(nil)
	}()
	<-started

	txs, fresh, err := m.size(nil)
	require.NoError(t, err, "a forgotten size is still the last one read")
	require.False(t, fresh)
	require.Equal(t, 400, txs)

	close(release)
	<-done
}

func TestMempoolSizeServesTheLastErrorDuringARead(t *testing.T) {
	t.Parallel()

	readErr := errors.New("rpc down")
	started, release := make(chan struct{}), make(chan struct{})
	reads := 0
	m := &mempoolSize{}
	m.read = func(any) (int, error) {
		reads++
		if reads == 2 {
			close(started)
			<-release
			return 400, nil
		}
		return 0, readErr
	}

	_, _, err := m.size(nil)
	require.ErrorIs(t, err, readErr)
	m.forget()

	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _, _ = m.size(nil)
	}()
	<-started

	_, _, err = m.size(nil)
	require.ErrorIs(t, err, readErr, "an outage isn't read as an empty mempool")

	close(release)
	<-done
}

func TestMempoolSizeAfterAPanickingRead(t *testing.T) {
	t.Parallel()

	reads := 0
	m := &mempoolSize{read: func(any) (int, error) {
		reads++
		if reads == 1 {
			panic("rpc client bug")
		}
		return 400, nil
	}}

	require.Panics(t, func() { _, _, _ = m.size(nil) })
	txs, fresh, err := m.size(nil)
	require.NoError(t, err)
	require.True(t, fresh, "the next task reads again")
	require.Equal(t, 400, txs)
}

func TestMempoolSizeBeforeTheFirstReadReturns(t *testing.T) {
	t.Parallel()

	started, release := make(chan struct{}), make(chan struct{})
	m := &mempoolSize{read: func(any) (int, error) {
		close(started)
		<-release
		return 400, nil
	}}

	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _, _ = m.size(nil)
	}()
	<-started

	_, fresh, err := m.size(nil)
	require.ErrorIs(t, err, errMempoolReadInProgress)
	require.False(t, fresh)

	close(release)
	<-done
}

func TestWholeSeconds(t *testing.T) {
	t.Parallel()

	require.Equal(t, 3*time.Second, wholeSeconds(3*time.Second))
	require.Equal(t, 4*time.Second, wholeSeconds(3*time.Second+time.Nanosecond))
	require.Equal(t, 4*time.Second, wholeSeconds(3600*time.Millisecond))
}

func TestSpread(t *testing.T) {
	t.Parallel()

	require.Zero(t, spread(49990, time.Minute))
	require.Equal(t, 30*time.Second, spread(49995, time.Minute))
	require.Equal(t, 54*time.Second, spread(49999, time.Minute))
}

func TestOrderRetryDelay(t *testing.T) {
	t.Parallel()

	tests := []struct {
		ahead uint64
		want  time.Duration
	}{
		{ahead: clerkOrderWindow, want: clerkDeferRetryDelay},
		{ahead: 2*clerkOrderWindow - 1, want: clerkDeferRetryDelay},
		{ahead: 2 * clerkOrderWindow, want: 2 * clerkDeferRetryDelay},
		{ahead: 7 * clerkOrderWindow, want: 7 * clerkDeferRetryDelay},
		{ahead: 8 * clerkOrderWindow, want: 8 * clerkDeferRetryDelay},
		{ahead: 40*clerkOrderWindow - 1, want: 39 * clerkDeferRetryDelay},
		{ahead: 40 * clerkOrderWindow, want: 40 * clerkDeferRetryDelay},
		{ahead: 200*clerkOrderWindow - 1, want: 199 * clerkDeferRetryDelay},
		{ahead: 200 * clerkOrderWindow, want: clerkMaxOrderDelay},
		{ahead: 50000, want: clerkMaxOrderDelay},
		{ahead: math.MaxUint64, want: clerkMaxOrderDelay},
	}

	for _, tc := range tests {
		require.Equal(t, tc.want, orderRetryDelay(tc.ahead), "ahead %d", tc.ahead)
	}
}

func TestPendingRecordsSeedHoldsForSeedLive(t *testing.T) {
	var p pendingRecords
	start := time.Now()

	p.seed(49992, start)
	lowest, _, ok := p.lowest(start.Add(seedLive))
	require.True(t, ok)
	require.Equal(t, uint64(49992), lowest)
	_, _, ok = p.lowest(start.Add(seedLive + time.Second))
	require.False(t, ok, "a gap with no task doesn't hold the window for pendingRecordTTL")

	p.seed(49992, start)
	p.touch(49992, "task", start)
	lowest, _, ok = p.lowest(start.Add(seedLive + time.Second))
	require.True(t, ok, "the ID's own task keeps it")
	require.Equal(t, uint64(49992), lowest)
}

func TestPendingRecordTTLOutlastsQueueLag(t *testing.T) {
	t.Parallel()

	// A live task can sit far behind its ETA in a backlogged worker pool; the
	// TTL must stay well above that so a live lowest ID isn't dropped.
	require.GreaterOrEqual(t, pendingRecordTTL, 10*time.Minute)
}

func TestPendingRecordsRetouchAfterDone(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_793_846_312, 0)

	var p pendingRecords
	p.touch(100, "", now)
	p.touch(200, "", now)
	p.done(100)
	require.Equal(t, uint64(200), p.touch(200, "", now))

	// A redelivered task for 100 runs again: it is pending again and lowest.
	require.Equal(t, uint64(100), p.touch(100, "", now))
	p.done(100)
	require.Equal(t, uint64(200), p.touch(300, "", now))
	require.Len(t, p.leases.entries, 2)
}

func BenchmarkPendingRecordsTouch(b *testing.B) {
	const pending = 50_000
	now := time.Unix(1_793_846_312, 0)

	var p pendingRecords
	for id := uint64(1); id <= pending; id++ {
		p.touch(id, "", now)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.touch(uint64(i%pending)+1, "", now)
	}
}

func TestMempoolSizeCache(t *testing.T) {
	t.Parallel()

	clock := time.Unix(1_793_846_312, 0)
	reads, txs := 0, 400
	m := &mempoolSize{
		read: func(any) (int, error) { reads++; return txs, nil },
		now:  func() time.Time { return clock },
	}

	got, fresh, err := m.size(nil)
	require.NoError(t, err)
	require.True(t, fresh)
	require.Equal(t, 400, got)

	txs = 900
	clock = clock.Add(mempoolCountTTL - time.Nanosecond)
	got, fresh, _ = m.size(nil)
	require.False(t, fresh, "a read inside the TTL is shared")
	require.Equal(t, 400, got)
	require.Equal(t, 1, reads)

	clock = clock.Add(time.Nanosecond)
	got, fresh, _ = m.size(nil)
	require.True(t, fresh, "a read at the TTL refreshes")
	require.Equal(t, 900, got)

	m.forget()
	_, fresh, _ = m.size(nil)
	require.True(t, fresh, "forget forces the next read")
	require.Equal(t, 3, reads)

	readErr := errors.New("rpc down")
	m.read = func(any) (int, error) {
		reads++
		clock = clock.Add(10 * time.Second) // a slow read
		return 0, readErr
	}
	m.forget()
	_, fresh, err = m.size(nil)
	require.ErrorIs(t, err, readErr)
	require.True(t, fresh)
	_, fresh, err = m.size(nil)
	require.ErrorIs(t, err, readErr, "errors are cached too, from when the slow read returned")
	require.False(t, fresh)
	require.Equal(t, 4, reads)
}

func TestReleasedRecords(t *testing.T) {
	t.Parallel()

	start := time.Unix(1_793_846_312, 0)
	var r releasedRecords
	require.False(t, r.has(49992, "a", start))

	r.touch(49992, "a", start)
	require.True(t, r.has(49992, "a", start.Add(releasedLive)))
	require.False(t, r.has(49992, "a", start.Add(releasedLive+time.Second)), "a task that missed two slow periods is taken as lost")
	require.False(t, r.has(49993, "a", start))
	require.False(t, r.has(49992, "b", start), "the same ID from another L1 log is not a duplicate")

	r.touch(49992, "a", start.Add(releasedLive))
	require.True(t, r.has(49992, "a", start.Add(2*releasedLive)), "a run refreshes the entry")

	r.touch(49994, "a", start)
	r.done(49994)
	require.False(t, r.has(49994, "a", start))
}

func TestIDLeasesSweepExpiredLeasesAboveALiveLowest(t *testing.T) {
	t.Parallel()

	start := time.Unix(1_793_846_312, 0)
	var l idLeases
	l.touch(49992, "", start, nil)
	for id := uint64(49993); id < 50093; id++ {
		l.touch(id, "", start, nil)
	}
	require.Len(t, l.entries, 101)

	require.Equal(t, start, l.swept)

	// 49992 keeps running, the tasks above it were lost.
	l.touch(49992, "", start.Add(pendingRecordTTL), nil)
	require.Equal(t, start, l.swept, "swept at most once per TTL")
	survivors := make([]uint64, 0, 20)
	for i := range uint64(20) {
		survivors = append(survivors, 60000+(i*7)%20)
	}
	for _, id := range survivors {
		l.touch(id, "", start.Add(pendingRecordTTL), nil)
	}
	expired := 0
	l.touch(49992, "", start.Add(pendingRecordTTL+time.Second), func(string) { expired++ })
	require.Equal(t, start.Add(pendingRecordTTL+time.Second), l.swept)
	require.Len(t, l.entries, 21)
	require.Len(t, l.ids, 21)
	require.Len(t, l.queued, 21)
	require.Equal(t, 100, expired)
	l.done(49992)
	for want := uint64(60000); want < 60020; want++ {
		lowest, _, ok := l.lowest(start.Add(pendingRecordTTL+time.Second), nil)
		require.True(t, ok)
		require.Equal(t, want, lowest, "the rebuilt heap is ordered")
		l.done(want)
	}
	l.touch(49992, "", start.Add(pendingRecordTTL+time.Second), nil)
	lowest, _, ok := l.lowest(start.Add(pendingRecordTTL+time.Second), nil)
	require.True(t, ok)
	require.Equal(t, uint64(49992), lowest)
}

func TestPendingRecordsLeasePerTask(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_793_846_312, 0)
	cp := &ClerkProcessor{BaseProcessor: BaseProcessor{Logger: log.NewNopLogger()}}
	cp.pending.touch(49992, "task-a", now)
	cp.pending.touch(49992, "task-b", now)

	task, err := tasks.NewWithSignature(func(context.Context) error { return nil }, &tasks.Signature{UUID: "task-a", RetryCount: 0})
	require.NoError(t, err)
	cp.releaseOnLastAttempt(task.Context, 49992, "origin", errors.New("rpc down"))
	require.Equal(t, uint64(49992), cp.pending.touch(50000, "task-c", now), "a duplicate delivery still holds the ID")

	cp.pending.release(49992, "task-b")
	require.Equal(t, uint64(50000), cp.pending.touch(50000, "task-c", now))
}

func TestReleasedRecordsPerOrigin(t *testing.T) {
	t.Parallel()

	start := time.Unix(1_793_846_312, 0)
	var r releasedRecords
	r.touch(49992, "a", start)
	r.touch(49992, "b", start.Add(time.Minute))
	require.True(t, r.has(49992, "a", start.Add(time.Minute)), "another origin doesn't overwrite a lease")
	require.True(t, r.has(49992, "b", start.Add(time.Minute)))

	r.doneOrigin(49992, "a")
	require.False(t, r.has(49992, "a", start.Add(time.Minute)))
	require.True(t, r.has(49992, "b", start.Add(time.Minute)))
	require.Equal(t, uint64(49992), r.touch(49995, "c", start.Add(time.Minute)))

	r.doneOrigin(49992, "b")
	require.NotContains(t, r.leases.entries, uint64(49992), "an ID leaves with its last lease")
	require.Equal(t, uint64(49995), r.touch(49995, "c", start.Add(time.Minute)))

	r.touch(49996, "a", start)
	r.touch(49996, "b", start)
	r.done(49996)
	require.False(t, r.has(49996, "a", start))
	require.False(t, r.has(49996, "b", start))
}

func TestIDLeasesLastRunIsTheNewestLease(t *testing.T) {
	t.Parallel()

	start := time.Unix(1_793_846_312, 0)
	var l idLeases
	l.touch(49992, "b", start.Add(time.Minute), nil)
	l.touch(49992, "a", start, nil)
	_, lastRun, ok := l.lowest(start.Add(time.Minute), nil)
	require.True(t, ok)
	require.Equal(t, start.Add(time.Minute), lastRun)

	expired := 0
	_, lastRun, ok = l.lowest(start.Add(pendingRecordTTL+time.Second), func(string) { expired++ })
	require.True(t, ok)
	require.Equal(t, start.Add(time.Minute), lastRun)
	require.Equal(t, 1, expired, "only the expired lease is dropped")
}

func TestReleasedRecordsLowest(t *testing.T) {
	t.Parallel()

	start := time.Unix(1_793_846_312, 0)
	var r releasedRecords
	require.Equal(t, uint64(49995), r.touch(49995, "a", start))
	require.Equal(t, uint64(49993), r.touch(49993, "a", start))
	require.Equal(t, uint64(49993), r.touch(49997, "a", start.Add(time.Minute)))

	require.Equal(t, uint64(49993), r.touch(49997, "a", start.Add(pendingRecordTTL)), "an entry lives for the whole TTL")

	later := start.Add(pendingRecordTTL + time.Second)
	require.Equal(t, uint64(49997), r.touch(49997, "a", later), "expired entries neither rank nor stay")
	require.Len(t, r.leases.entries, 1)
}

func TestPendingRecordsLowest(t *testing.T) {
	t.Parallel()

	start := time.Unix(1_793_846_312, 0)
	var p pendingRecords
	_, _, ok := p.lowest(start)
	require.False(t, ok)

	p.touch(49992, "", start)
	p.touch(49990, "", start)
	lowest, lastRun, ok := p.lowest(start)
	require.True(t, ok)
	require.Equal(t, uint64(49990), lowest)
	require.Equal(t, start, lastRun)
	require.Len(t, p.leases.ids, 2, "lowest adds nothing")

	p.done(49990)
	lowest, _, ok = p.lowest(start)
	require.True(t, ok)
	require.Equal(t, uint64(49992), lowest)

	_, _, ok = p.lowest(start.Add(pendingRecordTTL + time.Second))
	require.False(t, ok, "expired IDs aren't pending")
}

func TestStandbys(t *testing.T) {
	var s standbys
	t0 := time.Now()

	require.True(t, s.claim(7, "a", "task-1", t0))
	require.False(t, s.claim(7, "a", "task-2", t0), "one standby per event")
	require.True(t, s.claim(7, "a", "task-1", t0), "the standby itself keeps its place")
	require.True(t, s.claim(7, "b", "task-2", t0), "another origin is another event")
	require.True(t, s.claim(8, "a", "task-2", t0), "another ID is another event")
}

func TestStandbysGiveWayWhenStale(t *testing.T) {
	var s standbys
	t0 := time.Now()
	half := releasedLive / 2

	require.True(t, s.claim(1, "x", "other", t0))
	require.True(t, s.claim(7, "a", "task-1", t0.Add(half)))
	// A sweep here keeps task-1, so the next claims see its entry.
	require.True(t, s.claim(2, "x", "other", t0.Add(releasedLive+time.Millisecond)))

	require.False(t, s.claim(7, "a", "task-2", t0.Add(half+releasedLive)))
	require.True(t, s.claim(7, "a", "task-2", t0.Add(half+releasedLive+time.Nanosecond)), "a standby that stopped running gives way")

	require.True(t, s.claim(3, "x", "other", t0.Add(5*releasedLive)))
	require.Len(t, s.tasks, 1, "the sweep drops entries older than releasedLive")
}

func TestSetHeaderWithoutSignature(t *testing.T) {
	require.NotPanics(t, func() { setHeader(nil, standbyHeader) })

	signature := &tasks.Signature{}
	setHeader(signature, standbyHeader)
	require.Equal(t, headerTrue, signature.Headers[standbyHeader])
}

func TestDeferOutOfOrderBehindASeedOnlyLowest(t *testing.T) {
	n, rest, comet := newBurstNode(t)
	cp, _ := newBurstClerkProcessor(t, n, rest, comet)
	cp.lowestCheck.holdUntil(time.Now().Add(time.Hour))
	far := uint64(1 + 300*clerkOrderWindow)

	full := wholeSeconds(clerkMaxOrderDelay + spread(far, clerkMaxOrderDelay/5))
	capped := wholeSeconds(clerkSeedOrderDelay + spread(far, clerkSeedOrderDelay/5))

	cp.pending.touch(1, "task", time.Now().Add(-2*clerkLowestIdle))
	_, err := cp.deferOutOfOrder("StateSynced", far, "far")
	requireRetryLater(t, err, msgAheadOfPending, full)

	cp.pending.seed(1, time.Now())
	_, err = cp.deferOutOfOrder("StateSynced", far, "far")
	requireRetryLater(t, err, msgAheadOfPending, full)

	cp.pending.release(1, "task")
	_, err = cp.deferOutOfOrder("StateSynced", far, "far")
	requireRetryLater(t, err, msgAheadOfPending, capped)

	near := uint64(1 + 20*clerkOrderWindow)
	_, err = cp.deferOutOfOrder("StateSynced", near, "near")
	requireRetryLater(t, err, msgAheadOfPending, wholeSeconds(20*clerkDeferRetryDelay+spread(near, 20*clerkDeferRetryDelay/5)))
}

func TestAbandonedRecords(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_793_846_312, 0)
	var a abandonedRecords
	require.False(t, a.has(7, "a", now))

	a.add(7, "a", now)
	require.True(t, a.has(7, "a", now.Add(clerkMaxReleasedAge)))
	require.False(t, a.has(7, "a", now.Add(clerkMaxReleasedAge+time.Second)), "a replay after the cap is a fresh attempt")
	require.False(t, a.has(7, "b", now), "another L1 log with the same ID")
	require.False(t, a.has(8, "a", now))

	a.add(8, "a", now.Add(clerkMaxReleasedAge+time.Second))
	require.NotContains(t, a.at, standbyKey{id: 7, origin: "a"}, "expired entries are swept")
	require.Contains(t, a.at, standbyKey{id: 8, origin: "a"})
}

func TestAbandonRetryDelayIsOnTheReleasedCadence(t *testing.T) {
	t.Parallel()

	for _, id := range []uint64{49990, 49999} {
		require.GreaterOrEqual(t, abandonRetryDelay(id), releasedRetryDelay)
		require.Less(t, abandonRetryDelay(id), 2*releasedRetryDelay)
	}
	require.NotEqual(t, abandonRetryDelay(49990), abandonRetryDelay(49999), "spread per ID")
}
