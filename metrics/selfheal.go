package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Self-healing counters track recovery events processed by the bridge listener's
// self-heal loop. Names follow the heimdallv2 metrics namespace and the
// Prometheus convention (snake_case + _total suffix for monotonic counters),
// so they're discoverable by standard Prometheus tooling and Datadog scrapers.
var (
	// SelfHealStakeEventsProcessed counts missing nonce-gated stake events
	// (StakeUpdate, SignerChange, UnstakeInit) successfully recovered.
	SelfHealStakeEventsProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: "self_healing",
		Name:      "stake_events_processed_total",
		Help:      "Total number of missing nonce-gated stake events (StakeUpdate, SignerChange, UnstakeInit) processed by the self-heal loop",
	})

	// SelfHealStateSyncsProcessed counts missing StateSynced events
	// successfully recovered.
	SelfHealStateSyncsProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: "self_healing",
		Name:      "state_syncs_processed_total",
		Help:      "Total number of missing StateSynced events processed by the self-heal loop",
	})

	// SelfHealStakeNonceStuck counts self-heal cycles that skipped a validator
	// because heimdall marks its next stake event processed while its nonce
	// never advanced. Any increase needs a state repair, not a replay.
	SelfHealStakeNonceStuck = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: "self_healing",
		Name:      "stake_nonce_stuck_total",
		Help:      "Total number of self-heal cycles that skipped a validator whose next stake event is already processed but whose nonce did not advance",
	})

	// StakeNonceRetriesDropped counts bridge stake tasks dropped after
	// retrying an out-of-order nonce for longer than the retry age limit.
	StakeNonceRetriesDropped = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: "bridge",
		Name:      "stake_nonce_retries_dropped_total",
		Help:      "Total number of stake tasks dropped after retrying an out-of-order nonce past the retry age limit",
	})

	// SelfHealCheckpointAcksProcessed counts missing NewHeaderBlock checkpoint
	// acks successfully queued for replay.
	SelfHealCheckpointAcksProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: "self_healing",
		Name:      "checkpoint_acks_processed_total",
		Help:      "Total number of missing NewHeaderBlock checkpoint ACKs queued by the self-heal loop",
	})
)
