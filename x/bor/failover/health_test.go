package failover

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"cosmossdk.io/log"
	"github.com/stretchr/testify/require"
)

var errBoom = errors.New("boom")

func always(error) bool { return true }

func nopProbe(int) error { return nil }

func newTestHealth(t *testing.T, n int) *Health {
	t.Helper()
	h := New(n, nopProbe, Metrics{}, log.NewNopLogger())
	h.SetTuning(5*time.Millisecond, 1, 10*time.Millisecond, 50*time.Millisecond)
	return h
}

// validate marks endpoints validated+healthy, as a successful identity probe
// would. With the test threshold of 1, a single applied probe suffices.
func validate(h *Health, idxs ...int) {
	for _, i := range idxs {
		h.applyProbe(i, nil)
	}
}

func okFn(i int) (int, error) { return i, nil }

func TestNew_PrimaryValidatedFallbacksNot(t *testing.T) {
	h := New(3, nopProbe, Metrics{}, log.NewNopLogger())
	require.Equal(t, 0, h.Active())
	// Only the primary is trusted at boot, so no fallback is a candidate yet.
	require.Empty(t, h.Candidates(0))
}

func TestCall_PassthroughWhenActiveHealthy(t *testing.T) {
	h := newTestHealth(t, 2)
	var calls int
	_, err := Call(h, context.Background(), time.Second, func(_ context.Context, i int) (int, error) {
		calls++
		return okFn(i)
	}, always)
	require.NoError(t, err)
	require.Equal(t, 1, calls) // only the primary is called
	require.Equal(t, 0, h.Active())
}

func TestCall_CascadesToValidatedFallback(t *testing.T) {
	h := newTestHealth(t, 2)
	validate(h, 1)
	res, err := Call(h, context.Background(), time.Second, func(_ context.Context, i int) (int, error) {
		if i == 0 {
			return 0, errBoom
		}
		return okFn(i)
	}, always)
	require.NoError(t, err)
	require.Equal(t, 1, res)
	require.Equal(t, 1, h.Active())
}

func TestCall_DoesNotCascadeToUnvalidatedFallback(t *testing.T) {
	h := newTestHealth(t, 2) // secondary never validated (e.g., wrong network)
	var tried []int
	_, err := Call(h, context.Background(), time.Second, func(_ context.Context, i int) (int, error) {
		tried = append(tried, i)
		return 0, errBoom
	}, always)
	require.ErrorIs(t, err, errBoom)
	require.Equal(t, []int{0}, tried) // unvalidated secondary is never used
}

func TestCall_NoCascadeOnNonRetriableError(t *testing.T) {
	h := newTestHealth(t, 2)
	validate(h, 1)
	var tried []int
	_, err := Call(h, context.Background(), time.Second, func(_ context.Context, i int) (int, error) {
		tried = append(tried, i)
		return 0, errBoom
	}, func(error) bool { return false })
	require.ErrorIs(t, err, errBoom)
	require.Equal(t, []int{0}, tried)
	require.Equal(t, 0, h.Active())
}

func TestCall_StopsOnNonRetriableMidCascade(t *testing.T) {
	errRetry := errors.New("retry")
	errStop := errors.New("stop")
	pred := func(e error) bool { return errors.Is(e, errRetry) }

	h := newTestHealth(t, 3)
	validate(h, 1, 2)
	var tried []int
	_, err := Call(h, context.Background(), time.Second, func(_ context.Context, i int) (int, error) {
		tried = append(tried, i)
		switch i {
		case 0:
			return 0, errRetry // retriable → cascade
		case 1:
			return 0, errStop // non-retriable → stop here
		default:
			return i, nil
		}
	}, pred)
	require.ErrorIs(t, err, errStop)
	require.Equal(t, []int{0, 1}, tried) // the third endpoint is never reached
}

func TestCall_AllDownReturnsLastError(t *testing.T) {
	h := newTestHealth(t, 3)
	validate(h, 1, 2)
	_, err := Call(h, context.Background(), time.Second, func(_ context.Context, _ int) (int, error) {
		return 0, errBoom
	}, always)
	require.ErrorIs(t, err, errBoom)
}

func TestCall_PerAttemptTimeoutCascades(t *testing.T) {
	h := newTestHealth(t, 2)
	validate(h, 1)
	res, err := Call(h, context.Background(), 20*time.Millisecond,
		func(ctx context.Context, i int) (int, error) {
			if i == 0 {
				<-ctx.Done() // primary hangs until the per-attempt timeout fires
				return 0, ctx.Err()
			}
			return okFn(i)
		},
		func(e error) bool { return errors.Is(e, context.DeadlineExceeded) })
	require.NoError(t, err)
	require.Equal(t, 1, res)
	require.Equal(t, 1, h.Active())
}

func TestCall_CascadesWhenParentBudgetExceedsAttempt(t *testing.T) {
	// Production shape: the caller budgets attemptTimeout per endpoint, so a hung
	// primary times out at one attempt budget while the parent context (sized for
	// the whole cascade) is still alive to reach a fallback.
	h := newTestHealth(t, 2)
	validate(h, 1)
	attempt := 30 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), 3*attempt)
	defer cancel()

	res, err := Call(h, ctx, attempt, func(actx context.Context, i int) (int, error) {
		if i == 0 {
			<-actx.Done() // primary hangs for its whole attempt budget
			return 0, actx.Err()
		}
		return okFn(i)
	}, func(e error) bool { return errors.Is(e, context.DeadlineExceeded) })
	require.NoError(t, err)
	require.Equal(t, 1, res)
	require.Equal(t, 1, h.Active())
}

func TestCall_NoCascadeWhenBudgetIsSingleAttempt(t *testing.T) {
	// If the caller budgets only one attemptTimeout, a hung primary consumes the
	// whole budget and there is no room to try a fallback — the call fails
	// deterministically (the background prober routes subsequent calls instead).
	h := newTestHealth(t, 2)
	validate(h, 1)
	d := 30 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()

	_, err := Call(h, ctx, d, func(actx context.Context, i int) (int, error) {
		if i == 0 {
			<-actx.Done()
			return 0, actx.Err()
		}
		return okFn(i)
	}, func(e error) bool { return errors.Is(e, context.DeadlineExceeded) })
	require.Error(t, err)
	require.Equal(t, 0, h.Active())
}

func TestCall_CancellationPropagatesToInFlightAttempt(t *testing.T) {
	// An in-flight attempt must abort promptly when the caller cancels, well
	// before the (here, very long) per-attempt timeout would fire.
	h := newTestHealth(t, 2)
	validate(h, 1)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	_, err := Call(h, ctx, time.Hour, func(actx context.Context, _ int) (int, error) {
		<-actx.Done() // would block for an hour if cancellation did not propagate
		return 0, actx.Err()
	}, func(e error) bool { return errors.Is(e, context.Canceled) })
	require.Error(t, err)
	require.Less(t, time.Since(start), time.Second) // returned promptly on cancel, not after an hour
}

func TestCall_StopsCascadeWhenParentCancelled(t *testing.T) {
	h := newTestHealth(t, 2)
	validate(h, 1)
	ctx, cancel := context.WithCancel(context.Background())
	var tried []int
	_, err := Call(h, ctx, time.Second, func(_ context.Context, i int) (int, error) {
		tried = append(tried, i)
		cancel() // caller abandons the whole call during the first attempt
		return 0, errBoom
	}, always)
	require.Error(t, err)
	require.Equal(t, []int{0}, tried) // cancellation stops the cascade
}

func TestCandidates_OnlyHealthyInPriorityOrder(t *testing.T) {
	h := New(4, nopProbe, Metrics{}, log.NewNopLogger())
	h.SetTuning(time.Second, 1, 0, time.Second)
	validate(h, 1, 3) // index 1 and 3 validated+healthy; index 2 never probed
	require.Equal(t, []int{1, 3}, h.Candidates(0))

	h.MarkUnhealthy(1, errBoom)
	require.Equal(t, []int{3}, h.Candidates(0)) // a down endpoint drops out
}

func TestCandidates_CooledBeforeUncooled(t *testing.T) {
	h := New(3, nopProbe, Metrics{}, log.NewNopLogger())
	h.SetTuning(time.Second, 1, 20*time.Millisecond, time.Second)
	h.applyProbe(2, nil)              // index 2 healthy now
	time.Sleep(30 * time.Millisecond) // index 2 becomes cooled
	h.applyProbe(1, nil)              // index 1 healthy now (uncooled)
	require.Equal(t, []int{2, 1}, h.Candidates(0))
}

func TestSetActive_NoOpWhenSameIndex(t *testing.T) {
	var active int
	m := Metrics{ActiveIndex: func(int) { active++ }}
	h := New(2, nopProbe, m, log.NewNopLogger())
	h.SetActive(0)
	require.Zero(t, active)
	h.SetActive(1)
	require.Equal(t, 1, active)
}

func TestCall_ConcurrentWithProber(t *testing.T) {
	h := newTestHealth(t, 3) // nopProbe validates all endpoints over time
	h.Start()
	defer h.Stop()

	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				_, _ = Call(h, context.Background(), 50*time.Millisecond, func(_ context.Context, idx int) (int, error) {
					if idx == 0 {
						return 0, errBoom
					}
					return okFn(idx)
				}, always)
			}
		}()
	}
	wg.Wait()
}

func TestDefaults(t *testing.T) {
	require.Equal(t, 10*time.Second, DefaultCheckInterval)
	require.Equal(t, 3, DefaultConsecutiveThreshold)
	require.Equal(t, 60*time.Second, DefaultPromotionCooldown)
	require.Equal(t, 3*time.Second, DefaultProbeTimeout)
}
