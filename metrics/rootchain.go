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
