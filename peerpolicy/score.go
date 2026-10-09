package peerpolicy

import "github.com/cometbft/cometbft/p2p/servebudget"

const (
	windowCount = 6
	maxPeers    = 1024
	maxObjects  = 128
)

type scoreWindow struct {
	tick                            int64
	invalid, malformed              uint64
	repeatRefusals, servingRefusals uint8
	repeatedBytes                   uint64
}

type object struct {
	request         servebudget.Request
	tick            int64
	copies, pending uint64
	bytes           uint64
}

type Snapshot struct {
	Risk        uint64
	Score       int64
	Mode        string
	WouldAction string
}

func (p *peerState) window(tick int64) *scoreWindow {
	w := &p.windows[tick%windowCount]
	if w.tick != tick {
		*w = scoreWindow{tick: tick}
	}
	return w
}

func (p *peerState) risk(tick int64) uint64 {
	var score uint64
	for _, w := range p.windows {
		if w.tick > tick || tick-w.tick >= windowCount {
			continue
		}
		score += w.invalid*60 + w.malformed*20
		if w.repeatRefusals >= 3 || w.servingRefusals >= 3 {
			score += 20
		}
	}
	return min(score, 100)
}

func (g *Governor) Observe(id string, evidence servebudget.Evidence) {
	if !g.cfg.Observe {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	now := g.now()
	p := g.peer(id, now)
	if p == nil {
		return
	}
	defer func() { g.risk.Observe(float64(p.risk(int64(now / windowWidth)))) }()
	w := p.window(int64(now / windowWidth))
	switch evidence {
	case servebudget.InvalidResponse:
		w.invalid = min(w.invalid+1, 2)
		g.events.WithLabelValues("invalid_response").Inc()
	case servebudget.MalformedRequest:
		w.malformed = min(w.malformed+1, 5)
		g.events.WithLabelValues("malformed_request").Inc()
	}
}

func (g *Governor) Snapshot(id string) Snapshot {
	g.mu.Lock()
	defer g.mu.Unlock()
	s := Snapshot{Mode: "observe", WouldAction: "none"}
	if p := g.peers[id]; p != nil {
		s.Risk = p.risk(int64(g.now() / windowWidth))
		s.Score = -int64(s.Risk)
	}
	switch {
	case s.Risk >= 100:
		s.WouldAction = "jail"
	case s.Risk >= 50:
		s.WouldAction = "stop_bulk"
	case s.Risk >= 25:
		s.WouldAction = "reduce_bulk"
	}
	return s
}
