// Package failover provides a transport-agnostic endpoint failover state
// machine shared by Heimdall's Bor HTTP and gRPC clients. It tracks N
// priority-ordered endpoints (index 0 is the primary), cascades to the next
// reachable endpoint when the active one fails, and a background prober reverts
// to a higher-priority endpoint once it recovers.
//
// Fallback endpoints are only used after a probe has confirmed their identity
// against the expected chain. The primary owns that expectation whenever it is
// reachable, so a wrong-network fallback cannot serve while the primary is up.
// The one exception is a window where the primary has never been reachable: the
// highest-priority reachable fallback may then provisionally define the expected
// identity so failover still engages, which means a misconfigured wrong-network
// fallback could serve during that window. Once the primary becomes reachable it
// reclaims the expectation and demotes every other endpoint (see Reclaim),
// ending the window.
package failover

import (
	"context"
	"sync"
	"time"

	"cosmossdk.io/log"
)

// Prober timing defaults. Hardcoded rather than operator config to keep the
// config surface small; revisit if operators ask to tune them. The probe
// timeout stays well under CometBFT's ~10s ABCI budget.
const (
	DefaultCheckInterval        = 10 * time.Second
	DefaultConsecutiveThreshold = 3
	DefaultPromotionCooldown    = 60 * time.Second
	DefaultProbeTimeout         = 3 * time.Second
)

type endpointHealth struct {
	healthy            bool
	consecutiveSuccess int
	healthySince       time.Time
	lastErr            error
}

// Metrics holds optional, nil-safe metric hooks. This package carries no
// metrics-library dependency; callers wire concrete counters/gauges.
type Metrics struct {
	Switch          func()    // in-call cascade switched the active endpoint
	ProactiveSwitch func()    // background prober switched (incl. revert-to-primary)
	ActiveIndex     func(int) // active endpoint index changed
	HealthyCount    func(int) // number of healthy endpoints after a probe cycle
}

func (m Metrics) onSwitch() {
	if m.Switch != nil {
		m.Switch()
	}
}

func (m Metrics) onProactiveSwitch() {
	if m.ProactiveSwitch != nil {
		m.ProactiveSwitch()
	}
}

func (m Metrics) onActiveIndex(i int) {
	if m.ActiveIndex != nil {
		m.ActiveIndex(i)
	}
}

func (m Metrics) onHealthyCount(c int) {
	if m.HealthyCount != nil {
		m.HealthyCount(c)
	}
}

// Health is a failover state machine for n priority-ordered endpoints. All
// exported methods are safe for concurrent use.
type Health struct {
	mu     sync.Mutex
	health []endpointHealth
	active int
	n      int

	checkInterval        time.Duration
	consecutiveThreshold int
	promotionCooldown    time.Duration
	probeTimeout         time.Duration

	probe  func(i int) error
	metric Metrics
	logger log.Logger

	quit      chan struct{}
	done      chan struct{}
	startOnce sync.Once
	stopOnce  sync.Once
}

// New builds a Health for n endpoints. probe tests reachability and identity of
// endpoint i. The primary (index 0) is trusted at boot; a fallback only becomes
// healthy — and thus a failover candidate — after an identity-validating probe.
func New(n int, probe func(i int) error, m Metrics, logger log.Logger) *Health {
	if n < 1 {
		panic("bor failover: endpoint count must be positive")
	}
	hs := make([]endpointHealth, n)
	hs[0] = endpointHealth{healthy: true}
	return &Health{
		health:               hs,
		n:                    n,
		checkInterval:        DefaultCheckInterval,
		consecutiveThreshold: DefaultConsecutiveThreshold,
		promotionCooldown:    DefaultPromotionCooldown,
		probeTimeout:         DefaultProbeTimeout,
		probe:                probe,
		metric:               m,
		logger:               logger,
		quit:                 make(chan struct{}),
		done:                 make(chan struct{}),
	}
}

// SetTuning overrides the prober timings. Call before Start; tests use it to
// compress intervals while production keeps the defaults.
func (h *Health) SetTuning(check time.Duration, threshold int, cooldown, probeTimeout time.Duration) {
	h.checkInterval = check
	h.consecutiveThreshold = threshold
	h.promotionCooldown = cooldown
	h.probeTimeout = probeTimeout
}

// ProbeTimeout exposes the per-probe timeout for callers whose probe closures
// need to bound their own context.
func (h *Health) ProbeTimeout() time.Duration { return h.probeTimeout }

// Active returns the current active endpoint index.
func (h *Health) Active() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.active
}

// SetActive records a new active endpoint and updates the gauge.
func (h *Health) SetActive(i int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.setActiveLocked(i)
}

func (h *Health) setActiveLocked(i int) {
	if h.active == i {
		return
	}
	h.active = i
	h.metric.onActiveIndex(i)
}

// MarkUnhealthy flags endpoint i as failed, removing it from the failover
// candidate set until a probe re-confirms it.
func (h *Health) MarkUnhealthy(i int, err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.health[i].healthy = false
	h.health[i].consecutiveSuccess = 0
	h.health[i].lastErr = err
}

// MarkSuccess records a successful use of endpoint i, promoting it to healthy
// once it clears the consecutive-success threshold.
func (h *Health) MarkSuccess(i int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.markSuccessLocked(i)
}

func (h *Health) markSuccessLocked(i int) {
	h.health[i].consecutiveSuccess++
	h.health[i].lastErr = nil
	if !h.health[i].healthy && h.health[i].consecutiveSuccess >= h.consecutiveThreshold {
		h.health[i].healthy = true
		h.health[i].healthySince = time.Now()
	}
}

// Reclaim makes endpoint i the sole trusted endpoint: healthy and active at once
// (bypassing the consecutive-success threshold), and marks every other endpoint
// unhealthy so it must re-validate before it can be used again. It is called when
// the authoritative primary reclaims the chain identity after an outage: other
// endpoints may have been validated against a now-overwritten provisional
// identity, so their health is stale and must not be trusted — including as an
// in-call failover candidate — until a fresh probe re-confirms them against the
// reclaimed identity.
func (h *Health) Reclaim(i int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for j := range h.health {
		if j == i {
			continue
		}
		h.health[j].healthy = false
		h.health[j].consecutiveSuccess = 0
	}
	h.health[i].healthy = true
	h.health[i].consecutiveSuccess = h.consecutiveThreshold
	h.health[i].healthySince = time.Now()
	h.health[i].lastErr = nil
	h.setActiveLocked(i)
}

// Candidates returns the in-call failover targets to try after `failed` failed:
// only currently-healthy endpoints, ordered cooled (past the promotion cooldown)
// then uncooled, each in priority (index) order. A fallback is healthy only
// after an identity-validating probe, so a wrong-network or down endpoint is
// never a candidate.
func (h *Health) Candidates(failed int) []int {
	h.mu.Lock()
	defer h.mu.Unlock()
	var cooled, uncooled []int
	for i := 0; i < h.n; i++ {
		if i == failed || !h.health[i].healthy {
			continue
		}
		if time.Since(h.health[i].healthySince) >= h.promotionCooldown {
			cooled = append(cooled, i)
		} else {
			uncooled = append(uncooled, i)
		}
	}
	return append(cooled, uncooled...)
}

// Promote records a successful in-call failover switch from one endpoint to
// another, updating active, health, the switch metric, and the log.
func (h *Health) Promote(from, to int) {
	h.SetActive(to)
	h.MarkSuccess(to)
	h.metric.onSwitch()
	h.logger.Info("bor failover: switched active endpoint", "from", from, "to", to)
}

// Call executes fn against the active endpoint. On a retriable error it marks
// the active endpoint unhealthy and cascades through Candidates (validated
// endpoints only), promoting the first that succeeds. Each attempt is bounded
// by attemptTimeout; the cascade stops once ctx is done (caller deadline or
// cancellation). The caller must therefore budget more than one attemptTimeout
// for a cascade to reach a fallback — ContractCaller uses GetBorChainCallTimeout
// (attemptTimeout times the endpoint count) for exactly that.
func Call[T any](h *Health, ctx context.Context, attemptTimeout time.Duration, fn func(context.Context, int) (T, error), retriable func(error) bool) (T, error) {
	active := h.Active()
	res, err := callOnce(ctx, attemptTimeout, active, fn)
	if err == nil {
		return res, nil
	}
	if ctx.Err() != nil || !retriable(err) {
		return res, err
	}

	h.MarkUnhealthy(active, err)
	lastErr := err
	var zero T
	for _, i := range h.Candidates(active) {
		res, err := callOnce(ctx, attemptTimeout, i, fn)
		if err == nil {
			h.Promote(active, i)
			return res, nil
		}
		if ctx.Err() != nil || !retriable(err) {
			return zero, err
		}
		lastErr = err
		h.MarkUnhealthy(i, err)
	}
	return zero, lastErr
}

func callOnce[T any](ctx context.Context, attemptTimeout time.Duration, i int, fn func(context.Context, int) (T, error)) (T, error) {
	attemptCtx, cancel := WithAttemptTimeout(ctx, attemptTimeout)
	defer cancel()
	return fn(attemptCtx, i)
}

// WithAttemptTimeout derives a per-attempt context bounding a single endpoint
// attempt to timeout. It is a child of parent, so caller cancellation
// propagates immediately to the in-flight attempt and the attempt never
// outlives the caller's overall budget. Failover relies on that budget
// exceeding a single attemptTimeout — ContractCaller sizes it via
// GetBorChainCallTimeout (per-attempt timeout times the endpoint count) — so a
// cascade has the room to reach a fallback within the same call.
func WithAttemptTimeout(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, timeout)
}
