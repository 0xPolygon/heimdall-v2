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

	// ClerkMempoolBusyDeferrals counts clerk broadcasts the bridge deferred
	// because the local heimdall mempool was busy.
	ClerkMempoolBusyDeferrals = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: "bridge",
		Name:      "clerk_mempool_busy_deferrals_total",
		Help:      "Total number of clerk broadcasts deferred because the local heimdall mempool was busy",
	})

	// ClerkOrderDeferrals counts clerk broadcasts the bridge deferred because
	// the state sync was too far ahead of the lowest one it still had pending.
	ClerkOrderDeferrals = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: "bridge",
		Name:      "clerk_order_deferrals_total",
		Help:      "Total number of clerk broadcasts deferred because the state sync was too far ahead of the lowest pending one",
	})

	// ClerkPendingExpired counts state sync IDs the bridge stopped waiting on
	// because their task stopped running without being stored, broadcast or
	// given up on.
	ClerkPendingExpired = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: "bridge",
		Name:      "clerk_pending_expired_total",
		Help:      "Total number of pending state sync IDs the bridge stopped waiting on after their task went quiet past the TTL",
	})

	// ClerkRecordsUnstored counts state syncs a bridge broadcast again because
	// heimdall still hadn't stored them long after they entered the mempool.
	ClerkRecordsUnstored = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: "bridge",
		Name:      "clerk_records_unstored_total",
		Help:      "Total number of state syncs broadcast again because heimdall still hadn't stored them long after they entered the mempool",
	})

	// ClerkRecordsNotOnL1 counts rebroadcasts of released state syncs a bridge
	// skipped because their ID was above the latest state sync ID on L1.
	ClerkRecordsNotOnL1 = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: "bridge",
		Name:      "clerk_records_not_on_l1_total",
		Help:      "Total number of state sync rebroadcasts skipped because their ID was above the latest one on L1",
	})

	// ClerkPendingReconciled counts lowest pending state sync IDs a bridge
	// dropped from its order window because heimdall already stored them while
	// their task had stopped running.
	ClerkPendingReconciled = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: "bridge",
		Name:      "clerk_pending_reconciled_total",
		Help:      "Total number of idle lowest pending state syncs dropped because heimdall already stored them",
	})

	// ClerkTasksExhausted counts state sync tasks that ran out of retries
	// before heimdall stored their record.
	ClerkTasksExhausted = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: "bridge",
		Name:      "clerk_tasks_exhausted_total",
		Help:      "Total number of state sync tasks that ran out of retries before heimdall stored their record",
	})

	// SelfHealCheckpointAcksProcessed counts missing NewHeaderBlock checkpoint
	// acks successfully queued for replay.
	SelfHealCheckpointAcksProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: "self_healing",
		Name:      "checkpoint_acks_processed_total",
		Help:      "Total number of missing NewHeaderBlock checkpoint ACKs queued by the self-heal loop",
	})

	// SelfHealValidationRejected counts subgraph hits (checkpoint ack, stake
	// event, or state sync) rejected because the L1 receipt they resolved to
	// didn't structurally match the request, or decoded to different content
	// than the one the subgraph was queried for. Unlike a *Processed counter
	// staying flat, this fires on every rejection whether or not the recovery
	// path keeps retrying it — a non-zero rate means the subgraph or L1 RPC is
	// feeding self-heal mismatched data, the same class of untrusted response
	// the live listener path already tracks via rootchain_listener_log_rejected_total.
	SelfHealValidationRejected = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: "self_healing",
		Name:      "log_validation_rejected_total",
		Help:      "Total number of self-heal subgraph hits rejected for not matching the L1 receipt they resolved to",
	})
)
