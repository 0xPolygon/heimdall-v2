package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// RootChainListenerLogRejected counts rootchain event logs rejected because
// they didn't match the query that produced them (unexpected address, block
// number outside the queried range, or a topic outside the configured event
// set). A non-zero rate means the configured L1 RPC endpoint is returning
// log data that doesn't match what was asked for.
var RootChainListenerLogRejected = promauto.NewCounter(prometheus.CounterOpts{
	Namespace: Namespace,
	Subsystem: "rootchain_listener",
	Name:      "log_rejected_total",
	Help:      "Total number of rootchain event logs rejected for not matching the query that produced them",
})

// RootChainListenerLogQuarantined counts rootchain logs that failed
// validation on maxRootChainLogRejections poll cycles (not necessarily
// back-to-back — a cycle that aborts before content validation runs still
// preserves progress) and were therefore quarantined: the cursor advanced
// past the block containing them
// instead of withholding it forever. Distinct from log_rejected_total, which
// also fires for a rejection that later self-resolves — a non-zero rate here
// means a specific log needs manual operator investigation, not just a
// noisy endpoint.
var RootChainListenerLogQuarantined = promauto.NewCounter(prometheus.CounterOpts{
	Namespace: Namespace,
	Subsystem: "rootchain_listener",
	Name:      "log_quarantined_total",
	Help:      "Total number of rootchain event logs quarantined after repeatedly failing validation",
})
