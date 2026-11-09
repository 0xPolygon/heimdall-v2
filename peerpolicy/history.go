package peerpolicy

import "github.com/cometbft/cometbft/p2p/servebudget"

// lookupObject returns a peer's live entry for req in constant time, dropping
// the one stale entry it passes over. It never allocates, so a request that
// admission goes on to refuse leaves the table untouched.
func (g *Governor) lookupObject(p *peerState, req servebudget.Request, tick int64) *object {
	obj := p.objects[req]
	if obj == nil {
		return nil
	}
	if obj.pending > 0 || tick-obj.tick < windowCount {
		return obj
	}
	delete(p.objects, req)
	g.objects--
	return nil
}

// newObject records history only once a request has cleared every budget.
// Live entries are never displaced to make room: at either bound the request
// is refused neutrally, because a peer must not be able to reset its own
// repeat history by filling the table.
func (g *Governor) newObject(p *peerState, req servebudget.Request, tick int64) *object {
	g.expire(p, tick)
	if g.objects >= maxHistory {
		g.sweep(tick)
	}
	if len(p.objects) >= maxObjects || g.objects >= maxHistory {
		g.events.WithLabelValues("history_capacity").Inc()
		return nil
	}
	if p.objects == nil {
		p.objects = make(map[servebudget.Request]*object)
	}
	obj := &object{request: req, tick: tick}
	p.objects[req] = obj
	g.objects++
	return obj
}

// expire drops one peer's settled, aged entries at most once per interval, so
// a served request never pays for a scan.
func (g *Governor) expire(p *peerState, tick int64) {
	if p.historyTick == tick {
		return
	}
	p.historyTick = tick
	for req, obj := range p.objects {
		if obj.pending == 0 && tick-obj.tick >= windowCount {
			delete(p.objects, req)
			g.objects--
		}
	}
	if len(p.objects) == 0 {
		p.objects = nil
	}
}

// sweep reclaims node-wide history at most once per interval, and only under
// capacity pressure, where quiet peers hold entries their own admissions would
// otherwise never revisit.
func (g *Governor) sweep(tick int64) {
	if g.historyTick == tick {
		return
	}
	g.historyTick = tick
	for _, p := range g.peers {
		g.expire(p, tick)
	}
}

// forget returns a departing peer's entries to the node-wide budget.
func (g *Governor) forget(p *peerState) {
	g.objects -= uint64(len(p.objects))
}
