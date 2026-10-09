package peerpolicy

import (
	"strings"
	"sync"
	"time"

	"github.com/cometbft/cometbft/p2p/servebudget"
	"github.com/prometheus/client_golang/prometheus"
)

const windowWidth = 10 * time.Second

type pool struct {
	bytes, work                              tokenBucket
	memory, inflight, memoryCap, inflightCap uint64
}
type peerState struct {
	catchupBytes, catchupWork tokenBucket
	id                        string
	last                      time.Duration
	bytes, work               tokenBucket
	windows                   [windowCount]scoreWindow
	objects                   [maxObjects]object
	next                      int
	inflight, repeatHeld      uint64
}

type Governor struct {
	mu        sync.Mutex
	cfg       Config
	now       func() time.Duration
	pools     [3]pool
	peers     map[string]*peerState
	protected map[string]bool
	events    *prometheus.CounterVec
	windows   *prometheus.CounterVec
	risk      prometheus.Histogram
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
		g.pools[i] = pool{bytes: newBucket(cfg.BytesPerSecond/divisor, cfg.BurstBytes/divisor, now()), work: newBucket(cfg.RequestsPerSecond/divisor, cfg.RequestsPerSecond/divisor, now()), memoryCap: cfg.MemoryBytes / divisor, inflightCap: cfg.MaxInflight / divisor}
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
	}
	if g.protected[id] {
		index = 2
	}
	pool := &g.pools[index]
	memory := 2 * request.MaxBytes
	if pool.memory+memory > pool.memoryCap || pool.inflight >= pool.inflightCap || pool.work.available(now) < 1 {
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

func (p *peerState) findObject(req servebudget.Request, tick int64) *object {
	for i := range p.objects {
		o := &p.objects[i]
		if o.request == req && (o.pending > 0 || tick-o.tick < windowCount) {
			return o
		}
	}
	for range p.objects {
		o := &p.objects[p.next]
		p.next = (p.next + 1) % maxObjects
		if o.pending == 0 {
			*o = object{request: req, tick: tick}
			return o
		}
	}
	return nil
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
	if *count == 2 {
		g.windows.WithLabelValues(name).Inc()
	}
	*count = min(*count+1, 3)
}

func (g *Governor) reserve(p *peerState, pool *pool, obj *object, req servebudget.Request, estimate uint64, repeat bool) (servebudget.Lease, bool) {
	memory := 2 * req.MaxBytes
	pool.bytes.spend(estimate)
	peerBytes, peerWork := p.allowance(req.Family)
	peerBytes.spend(estimate)
	if repeat {
		p.repeatHeld += estimate
	}
	pool.work.spend(1)
	peerWork.spend(1)
	pool.memory += memory
	pool.inflight++
	p.inflight++
	obj.pending++
	g.events.WithLabelValues("admitted").Inc()
	return &lease{g: g, p: p, pool: pool, obj: obj, memory: memory, bytes: estimate, repeat: repeat, peerBytes: peerBytes}, true
}

func (g *Governor) admitPeer(p *peerState, pool *pool, req servebudget.Request, now time.Duration) (*object, uint64, bool, bool) {
	tick := int64(now / windowWidth)
	obj := p.findObject(req, tick)
	if obj == nil || obj.pending > 0 || p.inflight >= 4 {
		g.events.WithLabelValues("peer_busy").Inc()
		return nil, 0, false, false
	}
	estimate := min(req.MaxBytes, uint64(128<<10))
	if obj.bytes > 0 {
		estimate = obj.bytes
	}
	if pool.bytes.available(now) < float64(estimate) {
		g.events.WithLabelValues("local_bytes").Inc()
		return nil, 0, false, false
	}
	w := p.window(tick)
	peerBytes, peerWork := p.allowance(req.Family)
	if peerWork.available(now) < 1 || peerBytes.available(now) < float64(estimate) {
		g.refusal(w, false, req.Family)
		return nil, 0, false, false
	}
	repeat := obj.copies >= 2 && req.Family != servebudget.Catchup && req.Family != servebudget.Snapshot
	if repeat && estimate > g.cfg.RepeatBytes-min(g.cfg.RepeatBytes, w.repeatedBytes+p.repeatHeld) {
		g.refusal(w, true, req.Family)
		return nil, 0, false, false
	}
	return obj, estimate, repeat, true
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
	g.events.WithLabelValues("evicted").Inc()
	return true
}

func (p *peerState) allowance(family servebudget.Family) (*tokenBucket, *tokenBucket) {
	if family == servebudget.Catchup {
		return &p.catchupBytes, &p.catchupWork
	}
	return &p.bytes, &p.work
}
