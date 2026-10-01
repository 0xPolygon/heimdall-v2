package peerpolicy

import (
	"container/list"
	"sync"
	"time"

	"github.com/cometbft/cometbft/p2p/observation"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	windowWidth     = 10 * time.Second
	windowCount     = 6
	maxPeers        = 1024
	maxObjects      = 128
	repeatAllowance = 32 << 20
)

type (
	usage  struct{ items, bytes uint64 }
	bucket struct {
		tick          int64
		reasons       uint16
		usage         [familyCount]usage
		repeatedBytes uint64
	}
)

type objectKey struct {
	family family
	id     [16]byte
}
type objectRecord struct {
	key    objectKey
	tick   int64
	copies uint8
}
type peerRecord struct {
	id      string
	windows [windowCount]bucket
	objects map[objectKey]int
	ring    [maxObjects]objectRecord
	next    int
}

// Tracker uses bounded memory and no background workers. Reconnection with the
// same authenticated node ID retains evidence until expiry or LRU eviction.
type Tracker struct {
	mu      sync.Mutex
	now     func() time.Duration
	peers   map[string]*list.Element
	order   *list.List
	metrics *telemetry
}

func New(reg prometheus.Registerer) (*Tracker, error) {
	start := time.Now()
	return newTracker(reg, func() time.Duration { return time.Since(start) })
}

func newTracker(reg prometheus.Registerer, now func() time.Duration) (*Tracker, error) {
	m := newTelemetry()
	if err := reg.Register(m); err != nil {
		return nil, err
	}
	return &Tracker{now: now, peers: make(map[string]*list.Element), order: list.New(), metrics: m}, nil
}

// Observe borrows the existing decoded message without hashing, copying payloads,
// or repeating cryptographic validation. Consensus traffic is not volume-scored.
func (t *Tracker) Observe(id string, event observation.Event) {
	if id == "" || len(id) > 128 {
		return
	}
	e := classify(event)
	if e.reason == none && e.family == other {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	tick := int64(t.now() / windowWidth)
	p := t.record(id)
	before := actionFor(p.risk(tick))
	b := &p.windows[tick%windowCount]
	if b.tick != tick {
		*b = bucket{tick: tick}
	}
	t.metrics.traffic[e.family].Add(float64(e.bytes))
	r := e.reason
	if r == none {
		r = p.traffic(b, tick, e)
	}
	t.observeReason(b, r)
	risk := p.risk(tick)
	t.metrics.risk.Observe(float64(risk))
	if action := actionFor(risk); action > before {
		t.metrics.actions[action-1].Inc()
	}
}

func (t *Tracker) observeReason(b *bucket, r reason) {
	if r == none {
		return
	}
	t.metrics.events[r-1].Inc()
	bit := uint16(1) << r
	if b.reasons&bit == 0 {
		t.metrics.windows[r-1].Inc()
	}
	b.reasons |= bit
}

func (t *Tracker) record(id string) *peerRecord {
	if elem := t.peers[id]; elem != nil {
		t.order.MoveToFront(elem)
		return elem.Value.(*peerRecord)
	}
	if t.order.Len() == maxPeers {
		last := t.order.Back()
		delete(t.peers, last.Value.(*peerRecord).id)
		t.order.Remove(last)
		t.metrics.evictions.Inc()
	}
	p := &peerRecord{id: id, objects: make(map[objectKey]int)}
	t.peers[id] = t.order.PushFront(p)
	return p
}

func (p *peerRecord) traffic(b *bucket, tick int64, e evidence) reason {
	a := allowances[e.family]
	u := &b.usage[e.family]
	u.items = cappedAdd(u.items, e.items, a.items+1)
	u.bytes = cappedAdd(u.bytes, e.bytes, a.bytes+1)
	repeated := p.repeated(tick, e)
	if repeated {
		b.repeatedBytes = cappedAdd(b.repeatedBytes, e.bytes, repeatAllowance+1)
	}
	if u.items > a.items || u.bytes > a.bytes {
		return a.reason
	}
	if b.repeatedBytes > repeatAllowance {
		return servingRepetition
	}
	return none
}

func (p *peerRecord) repeated(tick int64, e evidence) bool {
	if !e.hasObject {
		return false
	}
	key := objectKey{e.family, e.object}
	index, ok := p.objects[key]
	if !ok {
		index = p.next
		p.next = (p.next + 1) % maxObjects
		delete(p.objects, p.ring[index].key)
		p.objects[key] = index
		p.ring[index] = objectRecord{key: key, tick: tick}
	}
	entry := &p.ring[index]
	if tick-entry.tick >= windowCount {
		entry.tick, entry.copies = tick, 0
	}
	entry.copies = min(entry.copies+1, 3)
	return entry.copies == 3
}

func cappedAdd(current, delta, limit uint64) uint64 {
	if delta >= limit-current {
		return limit
	}
	return current + delta
}

// Taking the strongest signal per window avoids stacking transport and reactor
// evidence for one delivery. This intentionally undercounts independent incidents.
func (p *peerRecord) risk(tick int64) uint64 {
	var total uint64
	for _, b := range p.windows {
		if tick-b.tick >= windowCount || b.tick > tick {
			continue
		}
		var strongest uint64
		for r := invalidEncoding; r < reasonCount; r++ {
			if b.reasons&(uint16(1)<<r) != 0 {
				strongest = max(strongest, weight(r))
			}
		}
		total += strongest
	}
	return min(total, 100)
}

func actionFor(risk uint64) int {
	if risk >= 100 {
		return 2
	}
	if risk >= 40 {
		return 1
	}
	return 0
}

type Snapshot struct {
	Mode          string            `json:"mode"`
	Risk          uint64            `json:"risk"`
	WouldAction   string            `json:"wouldAction"`
	ReasonWindows map[string]uint64 `json:"reasonWindows"`
}

// Snapshot is for local callers and tests. Peer IDs never become metric labels.
func (t *Tracker) Snapshot(id string) Snapshot {
	t.mu.Lock()
	defer t.mu.Unlock()
	s := Snapshot{Mode: "observe", WouldAction: "none", ReasonWindows: make(map[string]uint64)}
	elem := t.peers[id]
	if elem == nil {
		return s
	}
	p := elem.Value.(*peerRecord)
	tick := int64(t.now() / windowWidth)
	s.Risk = p.risk(tick)
	s.WouldAction = [...]string{"none", "throttle", "jail"}[actionFor(s.Risk)]
	for _, b := range p.windows {
		if tick-b.tick >= windowCount || b.tick > tick {
			continue
		}
		for r := invalidEncoding; r < reasonCount; r++ {
			if b.reasons&(uint16(1)<<r) != 0 {
				s.ReasonWindows[reasonNames[r]]++
			}
		}
	}
	return s
}
