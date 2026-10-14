package peerpolicy

import "time"

type lease struct {
	peerBytes                  *tokenBucket
	g                          *Governor
	p                          *peerState
	pool                       *pool
	obj                        *object
	memory, bytes              uint64
	repeat, prepared, finished bool
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
		l.pool.bytes.refund(l.bytes - bytes)
		l.peerBytes.refund(l.bytes - bytes)
	}
	if l.repeat {
		l.p.repeatHeld = l.p.repeatHeld - l.bytes + bytes
	}
	// The maximum reservation protects the store read and encoding. Once the
	// payload is prepared, retain only its actual size until the writer flushes.
	// Otherwise tiny blocks occupy a full maximum-block reservation in the queue.
	memory := 2 * bytes
	l.pool.memory -= l.memory - memory
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
	l.pool.inflight--
	l.p.inflight--
	l.obj.pending--
	if l.repeat {
		l.p.repeatHeld -= l.bytes
	}
	if written && l.prepared {
		tick := int64(g.now() / windowWidth)
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
			l.pool.bytes.refund(l.bytes)
			l.peerBytes.refund(l.bytes)
		}
		g.events.WithLabelValues("incomplete").Inc()
	}
}

func (l *lease) reserveMore(delta uint64, now time.Duration, w *scoreWindow) bool {
	g := l.g
	if l.pool.bytes.available(now) < float64(delta) {
		g.events.WithLabelValues("local_bytes").Inc()
		return false
	}
	if l.peerBytes.available(now) < float64(delta) {
		g.refusal(w, false, l.obj.request.Family)
		return false
	}
	if l.repeat && delta > g.cfg.RepeatBytes-min(g.cfg.RepeatBytes, w.repeatedBytes+l.p.repeatHeld) {
		g.refusal(w, true, l.obj.request.Family)
		return false
	}
	l.pool.bytes.spend(delta)
	l.peerBytes.spend(delta)
	return true
}
