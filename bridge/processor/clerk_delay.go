package processor

import (
	"sync"
	"time"
)

// rateGate lets a call through at most once per clerkStoreCheckDelay.
type rateGate struct {
	mu   sync.Mutex
	next time.Time
}

func (g *rateGate) due(now time.Time) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	if now.Before(g.next) {
		return false
	}
	g.next = now.Add(clerkStoreCheckDelay)
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

// lookupSlots hands out clerkRecordLookups slots without waiting. The zero
// value is ready to use.
type lookupSlots struct {
	once  sync.Once
	slots chan struct{}
}

func (l *lookupSlots) tryAcquire() bool {
	l.once.Do(func() { l.slots = make(chan struct{}, clerkRecordLookups) })
	select {
	case l.slots <- struct{}{}:
		return true
	default:
		return false
	}
}

func (l *lookupSlots) release() { <-l.slots }
