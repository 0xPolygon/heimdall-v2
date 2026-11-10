package processor

import (
	"container/heap"
	"sync"
	"time"

	"github.com/0xPolygon/heimdall-v2/metrics"
)

// pendingRecordTTL drops an ID whose task stopped retrying without reaching a
// terminal path. A live task runs again within clerkMaxStoreWait at most; the
// margin covers a worker pool that runs tasks far behind schedule during a
// burst, which must not evict a live lowest ID.
const pendingRecordTTL = 30 * time.Minute

// releasedLive is how long after its last run a released task still counts
// as running.
const releasedLive = 2 * clerkMaxStoreWait

// idLeases is a set of state sync IDs, each held by one lease per holder: a
// task for pending IDs, an L1 log origin for released ones. A lease is dropped
// pendingRecordTTL after its holder last ran, and an ID leaves the set with its
// last lease.
//
// A min-heap keeps the lowest ID at hand; IDs removed by done or expired by
// the TTL are dropped from the heap lazily, when they reach its top, so every
// operation stays O(log n) during a burst of tens of thousands of IDs. The zero
// value is ready to use.
type idLeases struct {
	mu      sync.Mutex
	entries map[uint64]map[string]time.Time
	ids     idHeap
	// queued holds the IDs with a node in ids, so an ID done and touched
	// again reuses its node instead of adding one per redelivery.
	queued map[uint64]struct{}
	// swept is when every lease was last checked for expiry: the heap only
	// checks its top, and a lowest ID that stays live would otherwise keep
	// expired leases above it forever.
	swept time.Time
}

// touch records that holder ran for id at now and returns the lowest ID in
// the set. expired runs for each lease the call drops.
func (l *idLeases) touch(id uint64, holder string, now time.Time, expired func(holder string)) uint64 {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.entries == nil {
		l.entries = make(map[uint64]map[string]time.Time)
		l.queued = make(map[uint64]struct{})
	}
	if _, ok := l.queued[id]; !ok {
		heap.Push(&l.ids, id)
		l.queued[id] = struct{}{}
	}
	if _, ok := l.entries[id]; !ok {
		l.entries[id] = make(map[string]time.Time, 1)
	}
	l.entries[id][holder] = now
	if now.Sub(l.swept) > pendingRecordTTL {
		l.sweepLocked(now, expired)
	}

	// id was just refreshed, so the scan stops at it at the latest.
	lowest, _, _ := l.lowestLocked(now, expired)
	return lowest
}

// lowest returns the lowest ID in the set and when its task last ran.
func (l *idLeases) lowest(now time.Time, expired func(holder string)) (uint64, time.Time, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.lowestLocked(now, expired)
}

func (l *idLeases) lowestLocked(now time.Time, expired func(holder string)) (uint64, time.Time, bool) {
	for len(l.ids) > 0 {
		lowest := l.ids[0]
		if lastRun, ok := l.liveLocked(lowest, now, expired); ok {
			return lowest, lastRun, true
		}
		heap.Pop(&l.ids)
		delete(l.queued, lowest)
	}
	return 0, time.Time{}, false
}

// sweepLocked drops every expired lease and rebuilds the heap from the IDs
// left. It runs at most once per pendingRecordTTL, so its O(n) cost is spread
// over that period's touches.
func (l *idLeases) sweepLocked(now time.Time, expired func(holder string)) {
	l.swept = now
	l.ids = l.ids[:0]
	clear(l.queued)
	for id := range l.entries {
		if _, ok := l.liveLocked(id, now, expired); ok {
			l.ids = append(l.ids, id)
			l.queued[id] = struct{}{}
		}
	}
	heap.Init(&l.ids)
}

// liveLocked drops id's expired leases and returns when its most recent live
// lease last ran. An ID without live leases leaves the set.
func (l *idLeases) liveLocked(id uint64, now time.Time, expired func(holder string)) (time.Time, bool) {
	var lastRun time.Time
	for holder, seen := range l.entries[id] {
		if now.Sub(seen) <= pendingRecordTTL {
			if seen.After(lastRun) {
				lastRun = seen
			}
			continue
		}
		delete(l.entries[id], holder)
		if expired != nil {
			expired(holder)
		}
	}
	if len(l.entries[id]) == 0 {
		delete(l.entries, id)
		return time.Time{}, false
	}
	return lastRun, true
}

// has reports whether id holds a lease from holder that ran
// within live.
func (l *idLeases) has(id uint64, holder string, now time.Time, live time.Duration) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	seen, ok := l.entries[id][holder]
	return ok && now.Sub(seen) <= live
}

// only reports whether holder holds the one lease on id.
func (l *idLeases) only(id uint64, holder string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	_, ok := l.entries[id][holder]
	return ok && len(l.entries[id]) == 1
}

// done removes id with all its leases. Its heap entry is discarded the next
// time it reaches the top.
func (l *idLeases) done(id uint64) {
	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.entries, id)
}

// doneHolder removes holder's lease on id.
func (l *idLeases) doneHolder(id uint64, holder string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.entries[id], holder)
	if len(l.entries[id]) == 0 {
		delete(l.entries, id)
	}
}

// pendingRecords tracks the state sync IDs this bridge still has to get into
// heimdall: seen by a task, not yet in the mempool, stored or given up on. Its
// lowest entry is
// the bridge's view of the next ID bor is waiting for. It starts empty after a
// restart and refills as tasks retry.
type pendingRecords struct {
	leases idLeases
}

// seedOrigin marks the pending lease seedFrontier adds without a task.
const seedOrigin = "seed"

// countExpired counts a pending ID whose task stopped running without a
// terminal outcome, e.g. it was lost on requeue; the window moves past it. A
// seed has no task to lose.
func countExpired(holder string) {
	if holder != seedOrigin {
		metrics.ClerkPendingExpired.Inc()
	}
}

// touch records that the task owner for id ran at now and returns the lowest
// pending ID.
func (p *pendingRecords) touch(id uint64, owner string, now time.Time) uint64 {
	return p.leases.touch(id, owner, now, countExpired)
}

// release removes the lease of the task owner for id; the ID stays pending
// while another task for it holds one.
func (p *pendingRecords) release(id uint64, owner string) { p.leases.doneHolder(id, owner) }

// lowest returns the lowest pending ID and when its task last ran, without
// adding one.
func (p *pendingRecords) lowest(now time.Time) (uint64, time.Time, bool) {
	return p.leases.lowest(now, countExpired)
}

// seedLive is how long a seed holds its ID. It only stands in for lower tasks
// arriving late after a start; a gap with no task behind it is left to that
// task, so it doesn't hold the window for a full pendingRecordTTL.
const seedLive = clerkMaxStoreWait

// seed adds id without a task behind it, for seedLive.
func (p *pendingRecords) seed(id uint64, now time.Time) {
	p.leases.touch(id, seedOrigin, now.Add(seedLive-pendingRecordTTL), countExpired)
}

// seedOnly reports whether id is pending only through its seed, with no task
// behind it.
func (p *pendingRecords) seedOnly(id uint64) bool { return p.leases.only(id, seedOrigin) }

// done removes id with every lease, once its record is stored or the ID
// stops holding the window.
func (p *pendingRecords) done(id uint64) { p.leases.done(id) }

// releasedRecords tracks the IDs whose task stopped holding the order window
// and keeps retrying, so a second task for the same event (a redelivery or a
// self-heal replay) ends instead of holding the window again. A task for the
// same ID from another L1 log is not a duplicate.
type releasedRecords struct {
	leases  idLeases
	standby standbys
}

// touch records that the released task for id ran at now and returns the
// lowest released ID.
func (r *releasedRecords) touch(id uint64, origin string, now time.Time) uint64 {
	return r.leases.touch(id, origin, now, nil)
}

// has reports whether a released task for id from the L1 log origin is still
// running. A released task runs at least every clerkMaxStoreWait, so one
// that missed two of those periods is taken as lost and no longer ends a
// redelivery of its event.
func (r *releasedRecords) has(id uint64, origin string, now time.Time) bool {
	return r.leases.has(id, origin, now, releasedLive)
}

func (r *releasedRecords) done(id uint64) { r.leases.done(id) }

func (r *releasedRecords) doneOrigin(id uint64, origin string) { r.leases.doneHolder(id, origin) }

// standbys holds, per released event, the one other task for it that waits
// to take over if the released task stops; further ones end, so replays of a
// record heimdall doesn't store don't pile up. The zero value is ready to use.
type standbys struct {
	mu    sync.Mutex
	tasks map[standbyKey]standbyTask
	swept time.Time
}

type standbyKey struct {
	id     uint64
	origin string
}

type standbyTask struct {
	owner string
	at    time.Time
}

// claim reports whether owner is the task waiting on the released event (id,
// origin), taking the place unless another task waited on it within
// releasedLive.
func (s *standbys) claim(id uint64, origin, owner string, now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.tasks == nil {
		s.tasks = make(map[standbyKey]standbyTask)
	}
	if now.Sub(s.swept) > releasedLive {
		s.swept = now
		for key, task := range s.tasks {
			if now.Sub(task.at) > releasedLive {
				delete(s.tasks, key)
			}
		}
	}
	key := standbyKey{id: id, origin: origin}
	if task, ok := s.tasks[key]; ok && task.owner != owner && now.Sub(task.at) <= releasedLive {
		return false
	}
	s.tasks[key] = standbyTask{owner: owner, at: now}
	return true
}

type idHeap []uint64

func (h idHeap) Len() int           { return len(h) }
func (h idHeap) Less(i, j int) bool { return h[i] < h[j] }
func (h idHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *idHeap) Push(x any) { *h = append(*h, x.(uint64)) }

func (h *idHeap) Pop() any {
	old := *h
	last := old[len(old)-1]
	*h = old[:len(old)-1]
	return last
}
