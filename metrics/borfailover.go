package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/0xPolygon/heimdall-v2/x/bor/failover"
)

// Bor endpoint-failover metrics. Labeled by transport ("http" or "grpc") so a
// single dashboard shows failover behavior across both Bor client paths.
// Counter names use the _total suffix per Prometheus convention.
var (
	BorFailoverSwitches = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: "bor_failover",
		Name:      "switches_total",
		Help:      "Total in-call failover switches of the active Bor endpoint, by transport",
	}, []string{"transport"})

	BorFailoverProactiveSwitches = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: "bor_failover",
		Name:      "proactive_switches_total",
		Help:      "Total background-prober switches of the active Bor endpoint (including revert-to-primary), by transport",
	}, []string{"transport"})

	BorFailoverActiveIndex = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: "bor_failover",
		Name:      "active_index",
		Help:      "Index of the currently active Bor endpoint (0 = primary), by transport",
	}, []string{"transport"})

	BorFailoverHealthyEndpoints = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: "bor_failover",
		Name:      "healthy_endpoints",
		Help:      "Number of Bor endpoints currently considered healthy, by transport",
	}, []string{"transport"})
)

// BorFailover wires the failover state machine's metric hooks to the labeled
// Prometheus vectors for the given transport ("http" or "grpc").
func BorFailover(transport string) failover.Metrics {
	return failover.Metrics{
		Switch:          func() { BorFailoverSwitches.WithLabelValues(transport).Inc() },
		ProactiveSwitch: func() { BorFailoverProactiveSwitches.WithLabelValues(transport).Inc() },
		ActiveIndex:     func(i int) { BorFailoverActiveIndex.WithLabelValues(transport).Set(float64(i)) },
		HealthyCount:    func(c int) { BorFailoverHealthyEndpoints.WithLabelValues(transport).Set(float64(c)) },
	}
}
