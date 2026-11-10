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

// releasedRetryDelay is how long a released task waits when a lookup slot or
// the busy gate holds it back, unless it ranks as the lowest ID. Its check
// isn't urgent, and a burst's released tasks retrying every few seconds would
// flood the task queue.
const releasedRetryDelay = clerkMaxStoreWait / 2

// deferDelay is how soon a task held back by a lookup slot or the busy gate
// runs again.
func deferDelay(id, lowest uint64, released bool) time.Duration {
	if !released || id == lowest {
		return clerkDeferRetryDelay
	}
	return wholeSeconds(releasedRetryDelay + spread(id, releasedRetryDelay))
}

// clerkRecordLookups bounds the record lookups clerk tasks run at once, so a
// slow heimdall API can't take every shared bridge worker.
const clerkRecordLookups = 2

// lookupSlots hands out clerkRecordLookups slots without waiting, plus one
// kept for the lowest ID, so the record bor waits on isn't queued behind the
// others and still can't take more than one extra worker. The zero value is
// ready to use.
type lookupSlots struct {
	once     sync.Once
	slots    chan struct{}
	priority sync.Mutex
}

// tryAcquire returns the release func of the slot it took, or nil.
func (l *lookupSlots) tryAcquire(lowest bool) func() {
	l.once.Do(func() { l.slots = make(chan struct{}, clerkRecordLookups) })
	if lowest {
		// Only its own slot: duplicate tasks for the lowest ID can't take
		// the shared ones.
		if l.priority.TryLock() {
			return l.priority.Unlock
		}
		return nil
	}
	select {
	case l.slots <- struct{}{}:
		return func() { <-l.slots }
	default:
		return nil
	}
}
