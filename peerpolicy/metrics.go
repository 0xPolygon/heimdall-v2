package peerpolicy

import "github.com/prometheus/client_golang/prometheus"

type telemetry struct {
	events, windows [reasonCount - 1]prometheus.Counter
	traffic         [familyCount]prometheus.Counter
	actions         [2]prometheus.Counter
	evictions       prometheus.Counter
	risk            prometheus.Histogram
	collectors      []prometheus.Collector
}

func newTelemetry(enforce bool) *telemetry {
	mode := modeObserve
	if enforce {
		mode = modeEnforce
	}
	m := &telemetry{}
	for r := invalidEncoding; r < reasonCount; r++ {
		m.events[r-1] = m.counter("reason_events_total", "Observed evidence events.", "reason", reasonNames[r])
		m.windows[r-1] = m.counter("reason_windows_total", "Ten-second peer windows containing evidence.", "reason", reasonNames[r])
	}
	for f := other; f < familyCount; f++ {
		m.traffic[f] = m.counter("bytes_total", "Observed encoded envelope bytes. Serving bytes are queued locally.", "family", familyNames[f])
	}
	m.actions[0] = m.counter("would_actions_total", "Upward score-band transitions; not a count of network actions.", "action", "throttle")
	m.actions[1] = m.counter("would_actions_total", "Upward score-band transitions; not a count of network actions.", "action", "jail")
	m.evictions = m.counter("evictions_total", "Peer records evicted at the capacity bound.", "mode", mode)
	m.risk = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "heimdall", Subsystem: "peer_reputation", Name: "risk", Help: "Event-sampled risk; not a peer-population distribution.", Buckets: []float64{0, 20, 40, 60, 80, 100},
	})
	m.collectors = append(m.collectors, m.risk)
	return m
}

func (m *telemetry) counter(name, help, key, value string) prometheus.Counter {
	c := prometheus.NewCounter(prometheus.CounterOpts{Namespace: "heimdall", Subsystem: "peer_reputation", Name: name, Help: help, ConstLabels: prometheus.Labels{key: value}})
	m.collectors = append(m.collectors, c)
	return c
}

func (m *telemetry) Describe(ch chan<- *prometheus.Desc) {
	for _, c := range m.collectors {
		c.Describe(ch)
	}
}

func (m *telemetry) Collect(ch chan<- prometheus.Metric) {
	for _, c := range m.collectors {
		c.Collect(ch)
	}
}
