package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	PreBlockerDuration = promauto.NewSummary(
		prometheus.SummaryOpts{
			Namespace: Namespace,
			Subsystem: "abci",
			Name:      "pre_blocker_duration_seconds",
			Help:      "Time taken by PreBlocker function in seconds",
			Objectives: map[float64]float64{
				0.50: 0.05,  // 50th percentile +/-5% error
				0.90: 0.01,  // 90th percentile +/-1% error
				0.99: 0.001, // 99th percentile +/-0.1% error
			},
		},
	)

	BeginBlockerDuration = promauto.NewSummary(
		prometheus.SummaryOpts{
			Namespace: Namespace,
			Subsystem: "abci",
			Name:      "begin_blocker_duration_seconds",
			Help:      "Time taken by BeginBlocker function in seconds",
			Objectives: map[float64]float64{
				0.50: 0.05,  // 50th percentile +/-5% error
				0.90: 0.01,  // 90th percentile +/-1% error
				0.99: 0.001, // 99th percentile +/-0.1% error
			},
		},
	)

	EndBlockerDuration = promauto.NewSummary(
		prometheus.SummaryOpts{
			Namespace: Namespace,
			Subsystem: "abci",
			Name:      "end_blocker_duration_seconds",
			Help:      "Time taken by EndBlocker function in seconds",
			Objectives: map[float64]float64{
				0.50: 0.05,  // 50th percentile +/-5% error
				0.90: 0.01,  // 90th percentile +/-1% error
				0.99: 0.001, // 99th percentile +/-0.1% error
			},
		},
	)
)

func RecordPreBlockerDuration(start time.Time) {
	duration := time.Since(start)
	PreBlockerDuration.Observe(duration.Seconds())
}

func RecordBeginBlockerDuration(start time.Time) {
	duration := time.Since(start)
	BeginBlockerDuration.Observe(duration.Seconds())
}

func RecordEndBlockerDuration(start time.Time) {
	duration := time.Since(start)
	EndBlockerDuration.Observe(duration.Seconds())
}
