package failover

import (
	"sync"
	"time"
)

// Start launches the background prober. It is a no-op for single-endpoint
// setups (nothing to fail over to) and starts at most once.
func (h *Health) Start() {
	if h.n <= 1 {
		return
	}
	h.startOnce.Do(func() { go h.run() })
}

// Stop terminates the background prober and waits for it to exit. Safe to call
// even if Start was never run.
func (h *Health) Stop() {
	if h.n <= 1 {
		return
	}
	// If Start never ran, close done here so the wait below doesn't block (and
	// consume startOnce so a later Start can't spawn run()).
	h.startOnce.Do(func() { close(h.done) })
	h.stopOnce.Do(func() { close(h.quit) })
	<-h.done
}

// run probes every endpoint on a ticker, reverting to a recovered
// higher-priority endpoint and switching away from an unhealthy active one.
func (h *Health) run() {
	defer close(h.done)

	h.cycle() // immediate cycle so a primary that is down at boot is caught quickly

	ticker := time.NewTicker(h.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-h.quit:
			return
		case <-ticker.C:
			h.cycle()
		}
	}
}

func (h *Health) cycle() {
	h.probeAll()
	h.maybePromote()
	h.maybeProactiveSwitch()
}

// probeAll probes every endpoint concurrently and applies each result as it
// completes, so a request arriving mid-cycle sees fresh data for finished
// probes rather than stale data for all of them.
func (h *Health) probeAll() {
	var wg sync.WaitGroup
	wg.Add(h.n)
	for i := 0; i < h.n; i++ {
		go func(idx int) {
			defer wg.Done()
			h.applyProbe(idx, h.probe(idx))
		}(i)
	}
	wg.Wait()

	h.metric.onHealthyCount(h.healthyCount())
}

func (h *Health) applyProbe(i int, err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if err != nil {
		h.health[i].healthy = false
		h.health[i].consecutiveSuccess = 0
		h.health[i].lastErr = err
		return
	}
	// A nil probe error means the endpoint answered and matched the expected
	// chain identity; markSuccessLocked promotes it toward healthy so it becomes
	// a failover candidate.
	h.markSuccessLocked(i)
}

func (h *Health) healthyCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	c := 0
	for i := 0; i < h.n; i++ {
		if h.health[i].healthy {
			c++
		}
	}
	return c
}

// maybePromote reverts to the highest-priority endpoint above the active one
// that is healthy and has passed the promotion cooldown.
func (h *Health) maybePromote() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.active == 0 {
		return
	}
	for i := 0; i < h.active; i++ {
		if h.health[i].healthy && time.Since(h.health[i].healthySince) >= h.promotionCooldown {
			h.promoteLocked(i, "revert to higher-priority bor endpoint")
			return
		}
	}
}

// maybeProactiveSwitch moves off an unhealthy active endpoint to the best
// healthy alternative before a request has to discover the failure in-call.
func (h *Health) maybeProactiveSwitch() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.health[h.active].healthy {
		return
	}
	if i, ok := h.bestHealthyLocked(); ok {
		h.promoteLocked(i, "switch away from unhealthy active bor endpoint")
	}
}

// bestHealthyLocked returns the highest-priority healthy endpoint other than
// the active one, preferring a cooled endpoint but falling back to an uncooled
// one in an emergency.
func (h *Health) bestHealthyLocked() (int, bool) {
	uncooled := -1
	for i := 0; i < h.n; i++ {
		if i == h.active || !h.health[i].healthy {
			continue
		}
		if time.Since(h.health[i].healthySince) >= h.promotionCooldown {
			return i, true
		}
		if uncooled == -1 {
			uncooled = i
		}
	}
	if uncooled != -1 {
		return uncooled, true
	}
	return 0, false
}

func (h *Health) promoteLocked(i int, msg string) {
	prev := h.active
	h.active = i
	h.metric.onActiveIndex(i)
	h.metric.onProactiveSwitch()
	h.logger.Info("bor failover: "+msg, "from", prev, "to", i)
}
