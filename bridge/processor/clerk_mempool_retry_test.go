package processor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	chainmanagertypes "github.com/0xPolygon/heimdall-v2/x/chainmanager/types"

	"cosmossdk.io/log"
	"github.com/RichardKnop/machinery/v1/tasks"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/bridge/broadcaster"
	"github.com/0xPolygon/heimdall-v2/bridge/util"
	clerktypes "github.com/0xPolygon/heimdall-v2/x/clerk/types"
)

func TestMempoolFullAlwaysRetries(t *testing.T) {
	t.Parallel()

	full := fmt.Errorf("broadcast succeeded but received non-ok response code: 20: %w", broadcaster.ErrMempoolFull)
	cp := &ClerkProcessor{BaseProcessor: BaseProcessor{Logger: log.NewNopLogger()}}
	cp.pending.touch(49992, "", time.Now())

	err := cp.handleBroadcastErr(context.Background(), "StateSynced", 49992, "a", full)

	var retryLater tasks.ErrRetryTaskLater
	require.True(t, errors.As(err, &retryLater), "got %v", err)
	require.Equal(t, util.RetryStateSyncTaskDelay, retryLater.RetryIn())
	require.Equal(t, uint64(49992), cp.pending.touch(50000, "", time.Now()), "a task still retrying keeps its ID pending")

	cp.releaseOnLastAttempt(context.Background(), 49992, "a", err)
	require.Equal(t, uint64(49992), cp.pending.touch(50000, "", time.Now()), "a retry-later error never releases the ID")
}

func TestReleaseOnLastAttemptKeepsOtherOrigins(t *testing.T) {
	t.Parallel()

	now := time.Now()
	cp := &ClerkProcessor{BaseProcessor: BaseProcessor{Logger: log.NewNopLogger()}}
	cp.released.touch(49992, "a", now)
	cp.released.touch(49992, "b", now)

	task, err := tasks.NewWithSignature(func(context.Context) error { return nil }, &tasks.Signature{RetryCount: 0})
	require.NoError(t, err)
	cp.releaseOnLastAttempt(task.Context, 49992, "a", errors.New("rpc down"))

	require.False(t, cp.released.has(49992, "a", now))
	require.True(t, cp.released.has(49992, "b", now), "another log's released task still runs")
}

func TestWarmingUp(t *testing.T) {
	t.Parallel()

	start := time.Unix(1_793_846_312, 0)
	cp := &ClerkProcessor{startedAt: start}
	require.True(t, cp.warmingUp(start.Add(clerkWarmup-time.Nanosecond)))
	require.False(t, cp.warmingUp(start.Add(clerkWarmup)))

	require.False(t, (&ClerkProcessor{}).warmingUp(start), "a processor without a start time doesn't warm up")
	require.True(t, NewClerkProcessor(nil).warmingUp(time.Now()), "a new processor warms up")
}

func TestHandleBroadcastErr(t *testing.T) {
	t.Parallel()

	other := errors.New("broadcast succeeded but received non-ok response code: 5")

	tests := []struct {
		name        string
		err         error
		wantDelay   time.Duration // zero: no retry-later
		wantErr     error
		wantSettled bool
	}{
		{name: "accepted broadcast settles the task", err: nil, wantDelay: clerkMaxStoreWait, wantSettled: true},
		{name: "mempool full defers the task", err: fmt.Errorf("broadcast succeeded but received non-ok response code: 20: %w", broadcaster.ErrMempoolFull), wantDelay: util.RetryStateSyncTaskDelay},
		{name: "other rejection spends a retry", err: other, wantErr: other},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			now := time.Now()
			cp := &ClerkProcessor{BaseProcessor: BaseProcessor{Logger: log.NewNopLogger()}}
			signature := &tasks.Signature{UUID: "task", RetryCount: 1, RetryTimeout: 8}
			task, err := tasks.NewWithSignature(func(context.Context) error { return nil }, signature)
			require.NoError(t, err)
			cp.pending.touch(49992, "task", now)

			err = cp.handleBroadcastErr(task.Context, "StateSynced", 49992, "a", tc.err)

			if tc.wantDelay == 0 {
				require.Equal(t, tc.wantErr, err)
			} else {
				var retryLater tasks.ErrRetryTaskLater
				require.True(t, errors.As(err, &retryLater), "got %v", err)
				require.Equal(t, tc.wantDelay, retryLater.RetryIn())
			}
			require.Equal(t, tc.wantSettled, windowReleased(signature))
			require.Equal(t, tc.wantSettled, cp.released.has(49992, "a", now))
			lowest := cp.pending.touch(50000, "other", now)
			if tc.wantSettled {
				require.Equal(t, uint64(50000), lowest, "an accepted ID leaves the order window")
				_, ok := retryStart(signature.Headers, acceptedAtHeader)
				require.True(t, ok)
				require.Equal(t, clerkTaskRetries, signature.RetryCount)
				require.Zero(t, signature.RetryTimeout)
			} else {
				require.Equal(t, uint64(49992), lowest, "a task still retrying keeps its ID pending")
			}
		})
	}
}

func TestWaitOnMempoolCopySettles(t *testing.T) {
	t.Parallel()

	now := time.Now()
	cp := &ClerkProcessor{BaseProcessor: BaseProcessor{Logger: log.NewNopLogger()}}
	signature := &tasks.Signature{UUID: "task"}
	task, err := tasks.NewWithSignature(func(context.Context) error { return nil }, signature)
	require.NoError(t, err)
	cp.pending.touch(49992, "task", now)

	requireRetryLater(t, cp.waitOnMempoolCopy(task.Context, "StateSynced", 49992, "a", now), msgAwaitingStore, clerkMaxStoreWait)
	require.True(t, windowReleased(signature))
	require.Equal(t, uint64(50000), cp.pending.touch(50000, "other", now), "a record already in the mempool doesn't hold the window")
	acceptedAt, ok := retryStart(signature.Headers, acceptedAtHeader)
	require.True(t, ok)
	require.Equal(t, now.Unix(), acceptedAt.Unix())
}

func TestAwaitStored(t *testing.T) {
	t.Parallel()

	acceptedAt := time.Unix(1_793_846_312, 0)
	checkDue := acceptedAt.Add(clerkMaxStoreWait)
	accepted := tasks.Headers{acceptedAtHeader: strconv.FormatInt(acceptedAt.Unix(), 10), windowReleasedHeader: headerTrue}
	rpcDown := errors.New("rpc down")

	tests := []struct {
		name         string
		headers      tasks.Headers
		noSignature  bool
		now          time.Time
		wantDelay    time.Duration // zero: no wait
		latestOnL1   uint64        // zero: far above the ID
		latestErr    error
		wantHeld     bool
		wantUnstored bool
	}{
		{name: "no task signature", noSignature: true, now: checkDue},
		{name: "never accepted", now: checkDue},
		{name: "accepted time in the future is ignored", headers: accepted, now: acceptedAt.Add(-time.Hour)},
		{name: "just accepted", headers: accepted, now: acceptedAt, wantDelay: clerkMaxStoreWait},
		{name: "woken early", headers: accepted, now: checkDue.Add(-1500 * time.Millisecond), wantDelay: 2 * time.Second},
		{name: "check due, record unstored", headers: accepted, now: checkDue, wantUnstored: true},
		{name: "check due, ID is the latest on L1", headers: accepted, now: checkDue, latestOnL1: 49992, wantUnstored: true},
		{name: "check due, ID above the latest on L1", headers: accepted, now: checkDue, latestOnL1: 49991, wantHeld: true},
		{name: "check due, latest on L1 unreadable", headers: accepted, now: checkDue, latestErr: rpcDown, wantUnstored: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			logs := &bytes.Buffer{}
			cp := &ClerkProcessor{BaseProcessor: BaseProcessor{Logger: log.NewLogger(logs, log.ColorOption(false))}}
			cp.latestOnL1.read = func() (uint64, error) {
				if tc.latestOnL1 == 0 {
					return 1 << 40, tc.latestErr
				}
				return tc.latestOnL1, tc.latestErr
			}

			ctx := context.Background()
			if !tc.noSignature {
				task, err := tasks.NewWithSignature(func(context.Context) error { return nil }, &tasks.Signature{Headers: tc.headers})
				require.NoError(t, err)
				ctx = task.Context
			}

			err := cp.awaitStored(ctx, "StateSynced", 49992, tc.now)
			require.Equal(t, tc.wantHeld, strings.Contains(logs.String(), warnMsgClerkNotOnL1))
			switch {
			case tc.wantHeld:
				requireRetryLater(t, err, msgNotOnL1, clerkMaxStoreWait)
			case tc.wantDelay == 0:
				require.NoError(t, err)
			default:
				requireRetryLater(t, err, msgAwaitingStore, tc.wantDelay)
			}
			require.Equal(t, tc.wantUnstored, strings.Contains(logs.String(), "ERR "+errMsgClerkUnstored))
		})
	}
}

func TestLatestRecordIDOnL1Cache(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_793_846_312, 0)
	reads, latest := 0, uint64(49991)
	var readErr error
	cp := &ClerkProcessor{}
	cp.latestOnL1.read = func() (uint64, error) {
		reads++
		return latest, readErr
	}

	got, err := cp.latestRecordIDOnL1(now)
	require.NoError(t, err)
	require.Equal(t, uint64(49991), got)

	latest = 49992
	got, _ = cp.latestRecordIDOnL1(now.Add(latestOnL1TTL - time.Second))
	require.Equal(t, uint64(49991), got, "shared within the TTL")
	require.Equal(t, 1, reads)

	got, _ = cp.latestRecordIDOnL1(now.Add(latestOnL1TTL + time.Second))
	require.Equal(t, uint64(49992), got)
	require.Equal(t, 2, reads)

	readErr = errors.New("l1 down")
	later := now.Add(3 * latestOnL1TTL)
	_, err = cp.latestRecordIDOnL1(later)
	require.Error(t, err)
	_, err = cp.latestRecordIDOnL1(later.Add(latestOnL1FailTTL - time.Second))
	require.Error(t, err, "a failed read is shared for a shorter period")
	require.Equal(t, 3, reads)

	readErr = nil
	got, err = cp.latestRecordIDOnL1(later.Add(latestOnL1FailTTL + time.Second))
	require.NoError(t, err)
	require.Equal(t, uint64(49992), got)
	require.Equal(t, 4, reads)
}

func TestLatestRecordIDOnL1DoesNotWaitForAnotherRead(t *testing.T) {
	t.Parallel()

	started, release := make(chan struct{}), make(chan struct{})
	cp := &ClerkProcessor{}
	cp.latestOnL1.read = func() (uint64, error) {
		close(started)
		<-release
		return 49992, nil
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = cp.latestRecordIDOnL1(time.Unix(1_793_846_312, 0))
	}()
	<-started

	_, err := cp.latestRecordIDOnL1(time.Unix(1_793_846_312, 0))
	require.ErrorIs(t, err, errCounterReadInProgress)
	require.NoError(t, cp.holdIfNotOnL1("StateSynced", 1<<40, time.Unix(1_793_846_312, 0)), "an unreadable counter never holds")

	close(release)
	<-done
}

func TestLatestRecordIDOnL1StampsWhenTheReadReturns(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_793_846_312, 0)
	reads := 0
	cp := &ClerkProcessor{}
	cp.latestOnL1.read = func() (uint64, error) {
		reads++
		time.Sleep(50 * time.Millisecond)
		return 0, errors.New("l1 timeout")
	}

	_, err := cp.latestRecordIDOnL1(now)
	require.Error(t, err)
	_, err = cp.latestRecordIDOnL1(now.Add(latestOnL1FailTTL))
	require.Error(t, err)
	require.Equal(t, 1, reads, "a slow failed read is still shared for the whole period")
}

func TestCachedChainParamsDuringARead(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_793_846_312, 0)
	c := &cachedChainParams{reading: true}
	got, ok := c.cachedLocked(now)
	require.True(t, ok)
	require.ErrorIs(t, got.err, errChainParamsReadInProgress)

	params := &chainmanagertypes.Params{}
	c.params, c.at = params, now.Add(-time.Hour)
	got, ok = c.cachedLocked(now)
	require.True(t, ok, "a stale value while another task reads")
	require.NoError(t, got.err)
	require.Same(t, params, got.params)

	c.reading = false
	_, ok = c.cachedLocked(now)
	require.False(t, ok, "stale and nobody reading: this caller reads")
}

func TestDeferDelay(t *testing.T) {
	t.Parallel()

	require.Equal(t, clerkDeferRetryDelay, deferDelay(49992, 49900, false))
	require.Equal(t, clerkDeferRetryDelay, deferDelay(49992, 49992, true), "the lowest ID retries quickly, released or not")
	require.Equal(t, wholeSeconds(releasedRetryDelay+spread(49992, releasedRetryDelay)), deferDelay(49992, 49900, true))
	require.Equal(t, releasedRetryDelay, deferDelay(49990, 49900, true), "no spread for an ID ending in 0")
}

func TestRateGate(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_793_846_312, 0)
	var g rateGate
	require.True(t, g.due(now))
	require.False(t, g.due(now.Add(clerkStoreCheckDelay-time.Second)))
	require.True(t, g.due(now.Add(clerkStoreCheckDelay)))
}

func TestClerkThrottleSettings(t *testing.T) {
	t.Parallel()

	// Chosen values: changing one is a behaviour change to review, not a refactor.
	require.Equal(t, 300, clerkMempoolBusyTxs)
	require.Equal(t, 1000, clerkFrontierBusyTxs)
	require.Equal(t, 2500, clerkLowestBusyTxs)
	require.Equal(t, uint64(10), uint64(clerkLowestIDs))
	require.Equal(t, uint64(25), uint64(clerkFrontierIDs))
	require.Equal(t, uint64(100), uint64(clerkOrderWindow))
	require.Equal(t, 10*time.Minute, clerkMaxStoreWait)
	require.Equal(t, 5*time.Minute, releasedRetryDelay)
	require.Equal(t, 5*time.Second, clerkStoreCheckDelay)
	require.Equal(t, time.Minute, clerkLowestIdle)
	require.Equal(t, 30*time.Second, latestOnL1TTL)
	require.Equal(t, 10*time.Second, latestOnL1FailTTL)
	require.Equal(t, 30*time.Second, clerkWarmup)
	require.Greater(t, clerkWarmup, util.RetryStateSyncTaskDelay, "every in-window task runs once during warm-up")
	require.Equal(t, 20*time.Minute, releasedLive)
	require.Equal(t, 2*time.Second, borStateTimeout)
	require.Less(t, borStateTimeout, seedTimeout, "the gap scan keeps part of the seed's time")
	require.Equal(t, 5*time.Second, seedTimeout)
	require.Equal(t, 10*time.Minute, clerkSeedRetryWindow)
	require.Equal(t, 10*time.Minute, clerkMaxOrderDelay)
	require.Equal(t, 3*time.Minute, clerkSeedOrderDelay)
	require.Less(t, clerkSeedOrderDelay, seedLive)
	require.LessOrEqual(t, clerkMaxOrderDelay, pendingRecordTTL/2, "a deferred task refreshes its pending lease well within the TTL")
	require.Less(t, clerkLowestBusyTxs, 5000, "below CometBFT's default mempool size")
}

func TestSameEventRecord(t *testing.T) {
	t.Parallel()

	base := burstRecord(49992)
	tests := []struct {
		name   string
		mutate func(m *clerktypes.MsgEventRecord)
		want   bool
	}{
		{name: "identical event, different submitter", mutate: func(m *clerktypes.MsgEventRecord) { m.From = "0x01" }, want: true},
		{name: "different L1 tx hash", mutate: func(m *clerktypes.MsgEventRecord) { m.TxHash = "0x02" }},
		{name: "different log index", mutate: func(m *clerktypes.MsgEventRecord) { m.LogIndex++ }},
		{name: "different id", mutate: func(m *clerktypes.MsgEventRecord) { m.Id++ }},
		{name: "different block number", mutate: func(m *clerktypes.MsgEventRecord) { m.BlockNumber++ }},
		{name: "different chain id", mutate: func(m *clerktypes.MsgEventRecord) { m.ChainId = "137" }},
		{name: "different contract", mutate: func(m *clerktypes.MsgEventRecord) { m.ContractAddress = "0x0000000000000000000000000000000000000def" }},
		{name: "different data", mutate: func(m *clerktypes.MsgEventRecord) { m.Data = []byte{0x01} }},
		{name: "nil and empty data", mutate: func(m *clerktypes.MsgEventRecord) { m.Data = nil }, want: len(base.Data) == 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			other := *base
			tc.mutate(&other)
			require.Equal(t, tc.want, sameEventRecord(base, &other))
		})
	}
}
