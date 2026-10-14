package peerpolicy

import "github.com/prometheus/client_golang/prometheus"

func (g *Governor) Describe(ch chan<- *prometheus.Desc) {
	g.events.Describe(ch)
	g.windows.Describe(ch)
	g.risk.Describe(ch)
}

func (g *Governor) Collect(ch chan<- prometheus.Metric) {
	g.events.Collect(ch)
	g.windows.Collect(ch)
	g.risk.Collect(ch)
}
