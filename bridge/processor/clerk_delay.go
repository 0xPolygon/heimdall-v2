package processor

import (
	"sync"
	"time"
)

// rateGate lets a call through at most once per every, or per
// clerkStoreCheckDelay when every is zero.
type rateGate struct {
	mu    sync.Mutex
	next  time.Time
	every time.Duration
}

func (g *rateGate) due(now time.Time) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	if now.Before(g.next) {
		return false
	}
	every := g.every
	if every == 0 {
		every = clerkStoreCheckDelay
	}
	g.next = now.Add(every)
	return true
}

// holdUntil keeps the gate shut until t.
func (g *rateGate) holdUntil(t time.Time) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.next = t
}

// spread returns a per-ID share of up to limit, so tasks deferred together
// don't all come back in the same instant.
func spread(id uint64, limit time.Duration) time.Duration {
	return time.Duration(id%10) * limit / 10
}

// wholeSeconds rounds a retry delay up to whole seconds: machinery declares a
// RabbitMQ delay queue per distinct delay (in ms, measured at publish), so
// fewer distinct delays mean far fewer queues during a burst.
func wholeSeconds(d time.Duration) time.Duration {
	return (d + time.Second - 1).Truncate(time.Second)
}

func orderRetryDelay(aheadOfLowest uint64) time.Duration {
	// Clamped before multiplying, so an ID far ahead can't overflow the delay.
	windows := min(aheadOfLowest/clerkOrderWindow, uint64(clerkMaxOrderDelay/clerkDeferRetryDelay))
	return time.Duration(windows) * clerkDeferRetryDelay
}

// releasedRetryDelay is how long a released task waits when the lookup gate or
// the busy gate holds it back, unless it ranks as the lowest ID. Its check
// isn't urgent, and a burst's released tasks retrying every few seconds would
// flood the task queue.
const releasedRetryDelay = clerkMaxStoreWait / 2

// deferDelay is how soon a task held back by the busy gate runs again.
func deferDelay(id, lowest uint64, released bool) time.Duration {
	if !released || id == lowest {
		return clerkDeferRetryDelay
	}
	return wholeSeconds(releasedRetryDelay + spread(id, releasedRetryDelay))
}

const (
	// clerkLookupTimeout bounds a record lookup; one that takes longer has
	// failed. A healthy heimdall answers in a few milliseconds.
	clerkLookupTimeout = time.Second

	// clerkLookupHold is how long record lookups stay held after a failed one.
	clerkLookupHold = 5 * time.Second
)

// lookupGate holds record lookups after one failed, so a slow heimdall API
// can't keep the shared bridge workers: once the hold is over a single lookup
// tries the API, and the rest follow when it succeeds. While lookups succeed
// every one runs: a task turned away keeps its place in the order window, so
// a burst delivered at once would stall behind the few deferred. The zero
// value is ready to use.
type lookupGate struct {
	mu        sync.Mutex
	heldUntil time.Time
	probing   bool
}

// enter reports whether a lookup may run now, and if not how long until one may.
func (g *lookupGate) enter(now time.Time) (bool, time.Duration) {
	g.mu.Lock()
	defer g.mu.Unlock()

	switch {
	case g.heldUntil.IsZero():
		return true, 0
	case now.Before(g.heldUntil):
		return false, g.heldUntil.Sub(now)
	case g.probing:
		return false, clerkLookupTimeout
	}
	g.probing = true
	return true, 0
}

// done records how a lookup ended.
func (g *lookupGate) done(failed bool, now time.Time) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.probing = false
	g.heldUntil = time.Time{}
	if failed {
		g.heldUntil = now.Add(clerkLookupHold)
	}
}
