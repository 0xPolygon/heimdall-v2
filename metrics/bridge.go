package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Bridge task and listener metrics. The task label is a registered task name
// and the listener label is a listener name, so both are bounded sets.
var (
	// BridgeTasksSent counts listener task sends by outcome: "sent",
	// "failed" (the queue rejected it) or "skipped" (no processor registered
	// the task, e.g. disabled with --only).
	BridgeTasksSent = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: "bridge",
		Name:      "tasks_sent_total",
		Help:      "Total number of bridge tasks sent by the listeners, labeled by task and outcome",
	}, []string{"task", "outcome"})

	// BridgeTaskDeliveries counts every delivery of a task to a processor,
	// retries included. Deliveries growing much faster than sends for the
	// same task means tasks are retrying in a loop.
	BridgeTaskDeliveries = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: "bridge",
		Name:      "task_deliveries_total",
		Help:      "Total number of bridge task deliveries to processors, retries included, labeled by task",
	}, []string{"task"})

	// BridgeTaskAge tracks how long ago a delivered task was first queued,
	// including its scheduled delay and any retries. A tail of hours means
	// tasks are stuck retrying.
	BridgeTaskAge = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: Namespace,
		Subsystem: "bridge",
		Name:      "task_age_seconds",
		Help:      "Time between a bridge task being first queued and each delivery of it, labeled by task",
		Buckets:   []float64{1, 5, 15, 30, 60, 300, 900, 1800, 3600, 3 * 3600, 6 * 3600, 24 * 3600},
	}, []string{"task"})

	// BridgeListenerHeadBlock is the newest block a listener can process:
	// the confirmed L1 block for the root chain listener, the latest height
	// for the heimdall listener.
	BridgeListenerHeadBlock = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: "bridge",
		Name:      "listener_head_block",
		Help:      "Newest block the bridge listener can process, labeled by listener",
	}, []string{"listener"})

	// BridgeListenerProcessedBlock is the last block a listener persisted as
	// fully processed. Head minus processed is the listener's lag; a lag
	// that keeps growing means the listener is stuck on a block.
	BridgeListenerProcessedBlock = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: "bridge",
		Name:      "listener_processed_block",
		Help:      "Last block the bridge listener persisted as processed, labeled by listener",
	}, []string{"listener"})

	// BridgeListenerRangeRetries counts block ranges a listener left for its
	// next poll, by reason: "not_queued" (their tasks couldn't be queued) or
	// "query_failed" (fetching or validating the blocks' events failed).
	BridgeListenerRangeRetries = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: "bridge",
		Name:      "listener_range_retries_total",
		Help:      "Total number of block ranges a bridge listener left unprocessed for its next poll, labeled by listener and reason",
	}, []string{"listener", "reason"})

	// BridgeListenerBlocksSkipped counts blocks a listener skipped because
	// their events kept failing to fetch. Each one is an event this node
	// never queued, so any increase needs a look.
	BridgeListenerBlocksSkipped = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: "bridge",
		Name:      "listener_blocks_skipped_total",
		Help:      "Total number of blocks a bridge listener skipped after repeatedly failing to fetch their events, labeled by listener",
	}, []string{"listener"})
)
