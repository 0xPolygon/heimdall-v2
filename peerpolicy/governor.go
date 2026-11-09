package peerpolicy

import (
	"strings"
	"sync"
	"time"

	"github.com/cometbft/cometbft/p2p/servebudget"
	"github.com/prometheus/client_golang/prometheus"
)

const windowWidth = 10 * time.Second

type peerState struct {
	catchupBytes, catchupWork tokenBucket
	id                        string
	last                      time.Duration
	bytes, work               tokenBucket
	windows                   [windowCount]scoreWindow
	objects                   map[servebudget.Request]*object
	historyTick               int64
	// held mirrors the outstanding byte reservations of each allowance class,
	// so a shortfall a peer's own in-flight requests explain stays neutral.
	held                 [2]uint64
	memory               [2]uint64
	inflight, repeatHeld uint64

	catchupInflight uint64
}

type Governor struct {
	mu          sync.Mutex
	cfg         Config
	now         func() time.Duration
	pools       [3]pool
	peers       map[string]*peerState
	protected   map[string]bool
	objects     uint64
	historyTick int64
	events      *prometheus.CounterVec
	windows     *prometheus.CounterVec
	risk        prometheus.Histogram

	objectLimit, historyLimit uint64
}

func New(cfg Config, registry prometheus.Registerer) (*Governor, error) {
	start := time.Now()
	return newGovernor(cfg, registry, func() time.Duration { return time.Since(start) })
}

func newGovernor(cfg Config, registry prometheus.Registerer, now func() time.Duration) (*Governor, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	g := &Governor{cfg: cfg, now: now, peers: make(map[string]*peerState), protected: make(map[string]bool)}
	g.objectLimit, g.historyLimit = cfg.historyLimits()
	g.events = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "heimdall_peer_serving_events_total", Help: "Native serving admission and observational reputation events."}, []string{"reason"})
	g.windows = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "heimdall_peer_reputation_windows_total", Help: "Peer-specific excess intervals after three refusals."}, []string{"reason"})
	g.risk = prometheus.NewHistogram(prometheus.HistogramOpts{Name: "heimdall_peer_reputation_risk", Help: "Event-sampled native peer risk in observation mode.", Buckets: []float64{0, 20, 25, 40, 50, 60, 80, 100}})
	if err := registry.Register(g); err != nil {
		return nil, err
	}
	for _, id := range cfg.ProtectedIDs {
		g.protected[strings.ToLower(id)] = true
	}
	// Disjoint shares prevent bulk peers from consuming catch-up or protected headroom.
	for i, divisor := range []uint64{2, 4, 4} {
		g.pools[i] = newPool(cfg, divisor, i == 1, now())
	}
	return g, nil
}

func (g *Governor) peer(id string, now time.Duration) *peerState {
	if len(id) != 40 {
		return nil
	}
	if p := g.peers[id]; p != nil {
		p.last = now
		return p
	}
	if len(g.peers) >= maxPeers && !g.evictIdle() {
		return nil
	}

	p := &peerState{id: id, last: now, bytes: newBucket(g.cfg.PeerBytesPerSecond, g.cfg.PeerBurstBytes, now), work: newBucket(g.cfg.PeerRequestsPerSecond, g.cfg.PeerRequestsPerSecond, now)}
	p.catchupBytes = newBucket(g.cfg.PeerBytesPerSecond, g.cfg.PeerBurstBytes, now)
	p.catchupWork = newBucket(g.cfg.PeerRequestsPerSecond, g.cfg.PeerRequestsPerSecond, now)
	g.peers[id] = p
	return p
}

func (g *Governor) Admit(id string, request servebudget.Request) (servebudget.Lease, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if request.MaxBytes == 0 || request.MaxBytes > g.cfg.MemoryBytes/2 || request.Family > servebudget.Catchup {
		return nil, false
	}
	now := g.now()
	index := 0
	if request.Family == servebudget.Catchup {
		index = 1
	} else if g.protected[id] {
		index = 2
	}
	pool := &g.pools[index]
	memory := 2 * request.MaxBytes
	if !pool.hasCapacity(memory, request.Family, now) {
		g.events.WithLabelValues("local_capacity").Inc()
		return nil, false
	}
	p := g.peer(id, now)
	if p == nil {
		return nil, false
	}
	defer func() {
		if g.cfg.Observe {
			g.risk.Observe(float64(p.risk(int64(now / windowWidth))))
		}
	}()
	obj, estimate, repeat, ok := g.admitPeer(p, pool, request, now)
	if !ok {
		return nil, false
	}
	return g.reserve(p, pool, obj, request, estimate, repeat)
}

func (g *Governor) refusal(w *scoreWindow, repeat bool, family servebudget.Family) {
	if family == servebudget.Catchup {
		g.events.WithLabelValues("catchup_capacity").Inc()
		return
	}
	name := "peer_allowance"
	count := &w.servingRefusals
	if repeat {
		name = "repeat_allowance"
		count = &w.repeatRefusals
	}
	g.events.WithLabelValues(name).Inc()
	if !g.cfg.Observe {
		return
	}
	previous := w.repeatRefusals + w.servingRefusals
	*count = min(*count+1, 3)
	if previous < 3 && w.repeatRefusals+w.servingRefusals >= 3 {
		if w.repeatRefusals > 0 && w.servingRefusals > 0 {
			name = "mixed_allowance"
		}
		g.windows.WithLabelValues(name).Inc()
	}
}

func (g *Governor) repeatRefusal(w *scoreWindow, family servebudget.Family, bytes uint64) {
	if bytes <= g.cfg.RepeatBytes-min(g.cfg.RepeatBytes, w.repeatedBytes) {
		g.events.WithLabelValues("repeat_reserved").Inc()
		return
	}
	g.refusal(w, true, family)
}

// byteRefusal scores a peer-byte shortfall only when the peer's own completed
// serving caused it. Reservations held by its other in-flight requests are
// payloads in progress, which Prepare refunds down to their encoded size, so a
// conformant pipeline that outruns its burst is congestion rather than excess.
func (g *Governor) byteRefusal(w *scoreWindow, family servebudget.Family, need uint64, avail float64, held uint64) {
	if family != servebudget.Catchup && avail+float64(held) >= float64(need) {
		g.events.WithLabelValues("peer_reserved").Inc()
		return
	}
	g.refusal(w, false, family)
}

func (g *Governor) reserve(p *peerState, pool *pool, obj *object, req servebudget.Request, estimate uint64, repeat bool) (servebudget.Lease, bool) {
	memory := 2 * req.MaxBytes
	poolBytes, familyWork := pool.allowance(req.Family)
	poolBytes.spend(estimate)
	peerBytes, peerWork, held := p.allowance(req.Family)
	peerBytes.spend(estimate)
	*held += estimate
	if repeat {
		p.repeatHeld += estimate
	}
	pool.work.spend(1)
	familyWork.spend(1)
	peerWork.spend(1)
	pool.memory += memory
	peerMemory := p.reservedMemory(req.Family)
	*peerMemory += memory
	pool.inflight++
	state := stateFamily(req.Family)
	if state {
		pool.stateMemory += memory
		pool.stateInflight++
	}
	p.inflight++
	if req.Family == servebudget.Catchup {
		p.catchupInflight++
	}
	obj.pending++
	g.events.WithLabelValues("admitted").Inc()
	return &lease{
		g: g, p: p, pool: pool, obj: obj, memory: memory, bytes: estimate, repeat: repeat, state: state,
		peerBytes: peerBytes, poolBytes: poolBytes, held: held, peerMemory: peerMemory,
	}, true
}

func (g *Governor) admitPeer(p *peerState, pool *pool, req servebudget.Request, now time.Duration) (*object, uint64, bool, bool) {
	tick := int64(now / windowWidth)
	obj := g.lookupObject(p, req, tick)
	if *p.reservedMemory(req.Family)+2*req.MaxBytes > pool.memoryCap/3 {
		g.events.WithLabelValues("peer_memory_capacity").Inc()
		return nil, 0, false, false
	}
	// Leave room for at least three peers in each pool.
	if p.pending(req.Family) >= min(maxPeerInflight, pool.inflightCap/3) || (obj != nil && obj.pending > 0) {
		g.events.WithLabelValues("peer_busy").Inc()
		return nil, 0, false, false
	}
	// Unseen payloads need their full allowance before the store read and encoding.
	estimate := req.MaxBytes
	if obj != nil && obj.bytes > 0 {
		estimate = obj.bytes
	}
	poolBytes, _ := pool.allowance(req.Family)
	if poolBytes.available(now) < float64(estimate) {
		g.events.WithLabelValues("local_bytes").Inc()
		return nil, 0, false, false
	}
	w := p.window(tick)
	repeat := obj != nil && obj.copies >= 2 && req.Family != servebudget.Catchup && req.Family != servebudget.Snapshot
	if repeat && estimate > g.cfg.RepeatBytes-min(g.cfg.RepeatBytes, w.repeatedBytes+p.repeatHeld) {
		g.repeatRefusal(w, req.Family, estimate)
		return nil, 0, false, false
	}
	if !g.admitAllowance(p, w, req, estimate, now) {
		return nil, 0, false, false
	}
	// History is recorded last so a refused request cannot claim a slot.
	if obj == nil {
		obj = g.newObject(p, req, tick)
	}
	return obj, estimate, repeat, obj != nil
}

// admitAllowance charges nothing; it reports whether the peer's own rate and
// byte allowances can fund estimate, recording why when they cannot.
func (g *Governor) admitAllowance(p *peerState, w *scoreWindow, req servebudget.Request, estimate uint64, now time.Duration) bool {
	peerBytes, peerWork, held := p.allowance(req.Family)
	if avail := peerBytes.available(now); avail < float64(estimate) {
		g.byteRefusal(w, req.Family, estimate, avail, *held)
		return false
	}
	// Native sync has no negotiated request rate or retry-after signal.
	// Work backpressure alone cannot distinguish catch-up from abuse.
	if peerWork.available(now) < 1 {
		g.events.WithLabelValues("peer_work_capacity").Inc()
		return false
	}
	return true
}

func (g *Governor) evictIdle() bool {
	var oldest *peerState
	for _, p := range g.peers {
		if p.inflight == 0 && (oldest == nil || p.last < oldest.last) {
			oldest = p
		}
	}
	if oldest == nil {
		return false
	}
	delete(g.peers, oldest.id)
	g.forget(oldest)
	g.events.WithLabelValues("evicted").Inc()
	return true
}

func (p *peerState) allowance(family servebudget.Family) (*tokenBucket, *tokenBucket, *uint64) {
	if family == servebudget.Catchup {
		return &p.catchupBytes, &p.catchupWork, &p.held[1]
	}
	return &p.bytes, &p.work, &p.held[0]
}

func (p *peerState) pending(family servebudget.Family) uint64 {
	if family == servebudget.Catchup {
		return p.catchupInflight
	}
	return p.inflight - p.catchupInflight
}

func (p *peerState) reservedMemory(family servebudget.Family) *uint64 {
	if family == servebudget.Catchup {
		return &p.memory[1]
	}
	return &p.memory[0]
}
