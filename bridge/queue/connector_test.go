package queue

import (
	"context"
	"testing"

	"github.com/RichardKnop/machinery/v1"
	eagerbackend "github.com/RichardKnop/machinery/v1/backends/eager"
	"github.com/RichardKnop/machinery/v1/brokers/amqp"
	eagerbroker "github.com/RichardKnop/machinery/v1/brokers/eager"
	eagerlock "github.com/RichardKnop/machinery/v1/locks/eager"
	"github.com/RichardKnop/machinery/v1/tasks"
	"github.com/stretchr/testify/require"
)

func TestNewConfigRoutesTasksToWorkerQueue(t *testing.T) {
	cnf := newConfig("amqp://localhost:5672")
	require.Equal(t, QName, cnf.DefaultQueue)

	broker := amqp.New(cnf)
	sig := &tasks.Signature{Name: "sendCheckpointToHeimdall"}
	broker.AdjustRoutingKey(sig)
	require.Equal(t, QName, sig.RoutingKey)
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
