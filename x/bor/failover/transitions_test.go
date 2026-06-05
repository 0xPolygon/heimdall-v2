package failover

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"cosmossdk.io/log"
	"github.com/stretchr/testify/require"
)

func TestPrimaryStartsHealthy(t *testing.T) {
	h := New(2, nopProbe, Metrics{}, log.NewNopLogger())
	h.SetActive(1)
	// Primary (0) is validated+healthy with a zero healthySince, so it is cooled
	// and maybePromote must revert to it.
	h.maybePromote()
	require.Equal(t, 0, h.Active())
}

func TestProbe_WrongNetworkNeverValidated(t *testing.T) {
	probe := func(i int) error {
		if i == 1 {
			return errBoom // wrong network / unreachable
		}
		return nil
	}
	h := New(2, probe, Metrics{}, log.NewNopLogger())
	h.SetTuning(time.Second, 1, 0, time.Second)

	h.probeAll()
	require.Empty(t, h.Candidates(0)) // never validated → never a failover candidate
}

func TestProbeAll_ValidatesAndCounts(t *testing.T) {
	var lastHealthy atomic.Int64
	m := Metrics{HealthyCount: func(c int) { lastHealthy.Store(int64(c)) }}
	probe := func(i int) error {
		if i == 1 {
			return errBoom
		}
		return nil
	}
	h := New(3, probe, m, log.NewNopLogger())
	h.SetTuning(time.Second, 1, 0, time.Second)

	h.probeAll()
	require.Equal(t, int64(2), lastHealthy.Load()) // index 0 and 2 up, index 1 down
	require.Equal(t, []int{2}, h.Candidates(0))    // only validated+up fallback
}

func TestMarkSuccess_ThresholdBoundary(t *testing.T) {
	h := New(3, nopProbe, Metrics{}, log.NewNopLogger())
	h.SetTuning(time.Second, 2, 0, time.Second) // threshold 2, cooldown 0

	h.applyProbe(2, nil)
	h.applyProbe(2, nil) // index 2 healthy+cooled after 2 successes
	h.applyProbe(1, nil) // index 1 one success → below threshold, not yet healthy
	require.Equal(t, []int{2}, h.Candidates(0))

	h.applyProbe(1, nil) // index 1 crosses threshold → healthy+cooled
	require.Equal(t, []int{1, 2}, h.Candidates(0))
}

func TestMaybePromote_RevertsToHigherPriority(t *testing.T) {
	h := New(3, nopProbe, Metrics{}, log.NewNopLogger())
	h.SetTuning(time.Second, 1, 0, time.Second)
	h.SetActive(2)
	h.MarkUnhealthy(0, errBoom) // primary down, so index 1 is the best higher-priority target
	h.applyProbe(1, nil)        // index 1 validated + healthy
	h.maybePromote()
	require.Equal(t, 1, h.Active())
}

func TestMaybeProactiveSwitch_MovesOffUnhealthyActive(t *testing.T) {
	h := New(2, nopProbe, Metrics{}, log.NewNopLogger())
	h.SetTuning(time.Second, 1, 0, time.Second)
	h.MarkUnhealthy(0, errBoom)
	h.applyProbe(1, nil) // index 1 validated + healthy
	h.maybeProactiveSwitch()
	require.Equal(t, 1, h.Active())
}

func TestMaybeProactiveSwitch_StaysWhenNoAlternative(t *testing.T) {
	var prosw atomic.Int64
	m := Metrics{ProactiveSwitch: func() { prosw.Add(1) }}
	h := New(2, nopProbe, m, log.NewNopLogger())
	h.MarkUnhealthy(0, errBoom) // active down; index 1 never validated
	h.maybeProactiveSwitch()
	require.Equal(t, 0, h.Active())
	require.Zero(t, prosw.Load())
}

func TestMaybeProactiveSwitch_NoSwitchWhenActiveHealthy(t *testing.T) {
	h := New(2, nopProbe, Metrics{}, log.NewNopLogger())
	h.SetTuning(time.Second, 1, 0, time.Second)
	h.applyProbe(1, nil) // a healthy alternative exists, but active (0) is healthy
	h.maybeProactiveSwitch()
	require.Equal(t, 0, h.Active())
}

func TestBestHealthy_PrefersCooled(t *testing.T) {
	h := New(3, nopProbe, Metrics{}, log.NewNopLogger())
	h.SetTuning(time.Second, 1, 20*time.Millisecond, time.Second)
	h.MarkUnhealthy(0, errBoom)       // active down
	h.applyProbe(2, nil)              // index 2 healthy now
	time.Sleep(30 * time.Millisecond) // index 2 becomes cooled
	h.applyProbe(1, nil)              // index 1 healthy now (uncooled)
	h.maybeProactiveSwitch()
	require.Equal(t, 2, h.Active()) // prefers the cooled endpoint over the uncooled one
}

func TestMaybeProactiveSwitch_UncooledEmergencyFallback(t *testing.T) {
	h := New(2, nopProbe, Metrics{}, log.NewNopLogger())
	h.SetTuning(time.Second, 1, time.Minute, time.Second) // long cooldown → stays uncooled
	h.MarkUnhealthy(0, errBoom)
	h.applyProbe(1, nil) // validated + healthy but not cooled
	h.maybeProactiveSwitch()
	require.Equal(t, 1, h.Active()) // still switches in the emergency (uncooled) pass
}

func TestMetricsHooksFire(t *testing.T) {
	var sw, prosw, active, healthy atomic.Int64
	m := Metrics{
		Switch:          func() { sw.Add(1) },
		ProactiveSwitch: func() { prosw.Add(1) },
		ActiveIndex:     func(int) { active.Add(1) },
		HealthyCount:    func(int) { healthy.Add(1) },
	}
	h := New(2, nopProbe, m, log.NewNopLogger())
	h.SetTuning(time.Second, 1, 0, time.Second)
	validate(h, 1)

	_, err := Call(h, context.Background(), time.Second, func(_ context.Context, i int) (int, error) {
		if i == 0 {
			return 0, errBoom
		}
		return okFn(i)
	}, always)
	require.NoError(t, err)
	require.Positive(t, sw.Load())
	require.Positive(t, active.Load())

	h.probeAll()
	require.Positive(t, healthy.Load())

	h.maybePromote()
	require.Positive(t, prosw.Load())
	require.Equal(t, 0, h.Active())
}

func TestProber_ProactiveSwitchThenRevert(t *testing.T) {
	var primaryUp atomic.Bool
	probe := func(i int) error {
		if i == 0 && !primaryUp.Load() {
			return errBoom
		}
		return nil
	}
	h := newTestHealthWithProbe(t, 2, probe)
	h.Start()
	defer h.Stop()

	require.Eventually(t, func() bool { return h.Active() == 1 }, 2*time.Second, 5*time.Millisecond,
		"should proactively switch off the down primary")

	primaryUp.Store(true)
	require.Eventually(t, func() bool { return h.Active() == 0 }, 2*time.Second, 5*time.Millisecond,
		"should revert to the recovered primary after cooldown")
}

func TestProber_ContinuesProbingOnTicker(t *testing.T) {
	probed := make(chan int, 8)
	h := newTestHealthWithProbe(t, 2, func(i int) error {
		probed <- i
		return nil
	})
	go h.run()
	defer func() {
		close(h.quit)
		<-h.done
	}()

	for range h.n {
		requireProbe(t, probed)
	}
	for range h.n {
		requireProbe(t, probed)
	}
}

func TestStart_ProbesOnEveryInterval(t *testing.T) {
	var probes atomic.Int64
	h := newTestHealthWithProbe(t, 2, func(int) error {
		probes.Add(1)
		return nil
	})
	h.Start()
	defer h.Stop()

	require.Eventually(t, func() bool {
		return probes.Load() >= int64(2*h.n)
	}, time.Second, 5*time.Millisecond)
}

func TestReclaim_AdoptsPrimaryAndDemotesOthers(t *testing.T) {
	var prosw atomic.Int64
	h := New(3, nopProbe, Metrics{ProactiveSwitch: func() { prosw.Add(1) }}, log.NewNopLogger())
	h.SetTuning(time.Second, 1, 0, time.Second) // threshold 1, cooldown 0
	h.MarkUnhealthy(0, errBoom)                 // primary was down during the outage
	h.applyProbe(1, nil)                        // a fallback validated (against a provisional identity) and is healthy
	h.SetActive(1)
	require.Equal(t, []int{1}, h.Candidates(0)) // the stale fallback is an eligible in-call candidate

	h.Reclaim(0)
	require.Equal(t, 0, h.Active()) // primary adopted as active at once, without the threshold
	require.Equal(t, int64(1), prosw.Load())
	require.Empty(t, h.Candidates(0)) // every other endpoint demoted → must re-validate before reuse

	h.Reclaim(0)
	require.Equal(t, int64(1), prosw.Load()) // no duplicate switch metric when already active
}

func TestStart_NoOpForSingleEndpoint(t *testing.T) {
	var probed atomic.Int64
	h := New(1, func(int) error { probed.Add(1); return nil }, Metrics{}, log.NewNopLogger())
	h.Start()
	time.Sleep(50 * time.Millisecond) // a (wrongly) spawned prober would probe within this window
	got := probed.Load()
	h.Stop()
	require.Zero(t, got)
}

func newTestHealthWithProbe(t *testing.T, n int, probe func(int) error) *Health {
	t.Helper()
	h := New(n, probe, Metrics{}, log.NewNopLogger())
	h.SetTuning(5*time.Millisecond, 1, 10*time.Millisecond, 50*time.Millisecond)
	return h
}

func requireProbe(t *testing.T, probed <-chan int) {
	t.Helper()
	select {
	case <-probed:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for probe")
	}
}
