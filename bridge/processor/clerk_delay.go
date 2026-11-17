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
// the busy gate turns it away, unless it ranks as the lowest ID. Its check
// isn't urgent, and a burst's released tasks retrying every few seconds would
// flood the task queue.
const releasedRetryDelay = clerkMaxStoreWait / 2

// deferDelay is how soon a task held back by the lookup gate or the busy gate
// runs again.
func deferDelay(id, lowest uint64, released bool) time.Duration {
	if !released || id == lowest {
		return clerkDeferRetryDelay
	}
	return wholeSeconds(releasedRetryDelay + spread(id, releasedRetryDelay))
}

const (
	// clerkLookupTimeout bounds a record lookup, so a heimdall API that stops
	// answering holds a worker no longer than this.
	clerkLookupTimeout = 2 * time.Second

	// clerkSlowLookup is how long a record lookup may take before heimdall's
	// API counts as slow. A healthy one answers in a few milliseconds.
	clerkSlowLookup = 500 * time.Millisecond

	// clerkSlowLookups bounds the record lookups clerk tasks run at once while
	// heimdall's API is slow, so it can't take every shared bridge worker.
	clerkSlowLookups = 2
)

// lookupGate lets every record lookup run while heimdall's API answers
// quickly: a task turned away keeps its place in the order window, so a burst
// delivered to every worker at once would stall behind the few deferred. Once
// a lookup is slow or fails it lets clerkSlowLookups run at a time, until one
// started after that answers quickly. The zero value is ready to use.
type lookupGate struct {
	mu        sync.Mutex
	slowSince time.Time
	running   int
}

// lookupDone reports how a record lookup went: quick, or slow or failed.
type lookupDone func(quick bool, now time.Time)

// enter returns what the lookup calls when it ends, or nil if it may not run
// now. The lowest ID's lookup, the one bor waits on, always may.
func (g *lookupGate) enter(lowest bool, now time.Time) lookupDone {
	g.mu.Lock()
	defer g.mu.Unlock()

	if !lowest {
		if !g.slowSince.IsZero() && g.running >= clerkSlowLookups {
			return nil
		}
		g.running++
	}
	return func(quick bool, end time.Time) { g.finish(now, !lowest, quick, end) }
}

// finish records the end of a lookup started at start. A quick one that
// started before the API turned slow says nothing about it since.
func (g *lookupGate) finish(start time.Time, counted, quick bool, end time.Time) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if counted {
		g.running--
	}
	switch {
	case !quick:
		g.slowSince = end
	case !start.Before(g.slowSince):
		g.slowSince = time.Time{}
	}
}
