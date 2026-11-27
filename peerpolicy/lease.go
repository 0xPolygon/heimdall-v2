package peerpolicy

import (
	"time"

	"github.com/cometbft/cometbft/p2p/servebudget"
)

type lease struct {
	peerBytes, poolBytes       *tokenBucket
	held, peerMemory           *uint64
	g                          *Governor
	p                          *peerState
	pool                       *pool
	obj                        *object
	memory, bytes              uint64
	repeat, prepared, finished bool
	state                      bool
	observe                    bool
}

func (l *lease) Prepare(bytes uint64) bool {
	g := l.g
	g.mu.Lock()
	defer g.mu.Unlock()
	if l.finished || l.prepared || bytes > l.obj.request.MaxBytes {
		return false
	}
	now := g.now()
	w := l.p.window(int64(now / windowWidth))
	if bytes > l.bytes {
		if !l.reserveMore(bytes-l.bytes, now, w) {
			return false
		}
	} else {
		l.poolBytes.refund(l.bytes - bytes)
		l.peerBytes.refund(l.bytes - bytes)
		*l.held -= l.bytes - bytes
	}
	if l.repeat {
		l.p.repeatHeld = l.p.repeatHeld - l.bytes + bytes
	}
	// The maximum reservation protects the store read and encoding. Once the
	// payload is prepared, retain only its actual size until the writer flushes.
	// Otherwise tiny blocks occupy a full maximum-block reservation in the queue.
	memory := 2 * bytes
	l.pool.memory -= l.memory - memory
	if l.state {
		l.pool.stateMemory -= l.memory - memory
	}
	*l.peerMemory -= l.memory - memory
	l.memory = memory
	l.bytes = bytes
	l.prepared = true
	return true
}

func (l *lease) Finish(written bool) {
	g := l.g
	g.mu.Lock()
	defer g.mu.Unlock()
	if l.finished {
		return
	}
	l.finished = true
	l.pool.memory -= l.memory
	*l.peerMemory -= l.memory
	l.pool.inflight--
	if l.state {
		l.pool.stateMemory -= l.memory
		l.pool.stateInflight--
	}
	l.p.inflight--
	if l.obj.request.Family == servebudget.Catchup {
		l.p.catchupInflight--
	}
	l.obj.pending--
	*l.held -= l.bytes
	if l.repeat {
		l.p.repeatHeld -= l.bytes
	}
	if written && l.prepared {
		now := g.now()
		tick := int64(now / windowWidth)
		if l.obj.request.Family != servebudget.Catchup {
			l.p.completed.tokens = max(0, l.p.completed.available(now)-float64(l.bytes))
		}
		l.obj.bytes = l.bytes
		l.obj.tick = tick
		l.obj.copies = min(l.obj.copies+1, 2)
		if l.repeat {
			w := l.p.window(tick)
			w.repeatedBytes += l.bytes
		}
		g.events.WithLabelValues("written").Inc()
	} else {
		// Bytes may have partially reached the socket. Keep prepared charges on failure.
		if !l.prepared {
			l.poolBytes.refund(l.bytes)
			l.peerBytes.refund(l.bytes)
		}
		g.events.WithLabelValues("incomplete").Inc()
	}
}

func (l *lease) reserveMore(delta uint64, now time.Duration, w *scoreWindow) bool {
	g := l.g
	if l.poolBytes.available(now) < float64(delta) {
		g.events.WithLabelValues("local_bytes").Inc()
		return false
	}
	if avail := l.peerBytes.available(now); avail < float64(delta) {
		g.byteRefusal(l.p, w, l.obj.request.Family, l.bytes+delta, now, l.observe)
		return false
	}
	if l.repeat && delta > g.cfg.RepeatBytes-min(g.cfg.RepeatBytes, w.repeatedBytes+l.p.repeatHeld) {
		g.repeatRefusal(w, l.obj.request.Family, l.bytes+delta, l.observe)
		return false
	}
	l.poolBytes.spend(delta)
	l.peerBytes.spend(delta)
	*l.held += delta
	return true
}
