package queue

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/RichardKnop/machinery/v1"
	eagerbackend "github.com/RichardKnop/machinery/v1/backends/eager"
	"github.com/RichardKnop/machinery/v1/brokers/amqp"
	eagerbroker "github.com/RichardKnop/machinery/v1/brokers/eager"
	eagerlock "github.com/RichardKnop/machinery/v1/locks/eager"
	"github.com/RichardKnop/machinery/v1/tasks"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/metrics"
)

func TestNewConfigRoutesTasksToWorkerQueue(t *testing.T) {
	cnf := newConfig("amqp://localhost:5672")
	require.Equal(t, QName, cnf.DefaultQueue)

	broker := amqp.New(cnf)
	sig := &tasks.Signature{Name: "sendCheckpointToHeimdall"}
	broker.AdjustRoutingKey(sig)
	require.Equal(t, QName, sig.RoutingKey)
}

func TestNewConfigLeavesSignalsToHeimdalld(t *testing.T) {
	require.True(t, newConfig("amqp://localhost:5672").NoUnixSignals)
}

func TestNormalizeRoutingKey(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"legacy key", legacyRoutingKey, QName},
		{"current key", QName, QName},
		{"empty", "", ""},
		{"custom", "other", "other"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sig := &tasks.Signature{RoutingKey: tt.in}
			normalizeRoutingKey(sig)
			require.Equal(t, tt.want, sig.RoutingKey)
		})
	}
}

func TestNewWorkerNormalizesLegacyRoutingKey(t *testing.T) {
	server := machinery.NewServerWithBrokerBackendLock(
		newConfig("amqp://localhost:5672"), eagerbroker.New(), eagerbackend.New(), eagerlock.New())

	var seen string
	require.NoError(t, server.RegisterTask("probe", func(ctx context.Context) error {
		seen = tasks.SignatureFromContext(ctx).RoutingKey
		return nil
	}))

	sig := &tasks.Signature{UUID: "task_probe", Name: "probe", RoutingKey: legacyRoutingKey}
	require.NoError(t, newWorker(server).Process(sig))
	require.Equal(t, QName, seen)
}

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
