package metrics

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

// TestBridgeTaskAgeBuckets checks the task age histogram can tell a task stuck
// for hours from one stuck for a day.
func TestBridgeTaskAgeBuckets(t *testing.T) {
	t.Parallel()

	var age dto.Metric
	require.NoError(t, BridgeTaskAge.WithLabelValues("bucketsTest").(prometheus.Metric).Write(&age))

	var bounds []float64
	for _, bucket := range age.GetHistogram().GetBucket() {
		bounds = append(bounds, bucket.GetUpperBound())
	}
	require.Subset(t, bounds, []float64{
		time.Hour.Seconds(), (3 * time.Hour).Seconds(), (6 * time.Hour).Seconds(), (24 * time.Hour).Seconds(),
	})
}
