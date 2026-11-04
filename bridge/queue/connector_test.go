package queue

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/RichardKnop/machinery/v1/tasks"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/metrics"
)

func TestTaskQueuedAt(t *testing.T) {
	t.Parallel()

	t.Run("survives the JSON round trip machinery makes on every retry", func(t *testing.T) {
		t.Parallel()

		queuedAt := time.UnixMilli(1_760_000_000_123)
		body, err := json.Marshal(&tasks.Signature{Name: "task", Headers: NewTaskHeaders(queuedAt)})
		require.NoError(t, err)

		var delivered tasks.Signature
		require.NoError(t, json.Unmarshal(body, &delivered))

		got, ok := taskQueuedAt(delivered.Headers)
		require.True(t, ok)
		require.True(t, queuedAt.Equal(got), "got %v, want %v", got, queuedAt)
	})

	for name, headers := range map[string]tasks.Headers{
		"no headers":      nil,
		"missing header":  {"other": "1"},
		"malformed value": {queuedAtHeader: "yesterday"},
		"non-string":      {queuedAtHeader: 1_760_000_000_123},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, ok := taskQueuedAt(headers)
			require.False(t, ok)
		})
	}
}

func TestObserveTaskDelivery(t *testing.T) {
	t.Parallel()

	const task = "observeDeliveryTestTask"
	signature := &tasks.Signature{Name: task, Headers: NewTaskHeaders(time.Now().Add(-time.Minute))}

	observeTaskDelivery(signature)
	observeTaskDelivery(&tasks.Signature{Name: task})

	require.InDelta(t, 2, testutil.ToFloat64(metrics.BridgeTaskDeliveries.WithLabelValues(task)), 0, "every delivery counts, with or without a queue time")

	var age dto.Metric
	require.NoError(t, metrics.BridgeTaskAge.WithLabelValues(task).(prometheus.Metric).Write(&age))
	require.Equal(t, uint64(1), age.GetHistogram().GetSampleCount(), "only a delivery with a queue time has an age")
	require.GreaterOrEqual(t, age.GetHistogram().GetSampleSum(), time.Minute.Seconds())
}
