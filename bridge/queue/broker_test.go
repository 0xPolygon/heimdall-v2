package queue

import (
	"context"
	"errors"
	"testing"
	"time"

	"cosmossdk.io/log"
	"github.com/RichardKnop/machinery/v1"
	nullbackend "github.com/RichardKnop/machinery/v1/backends/null"
	eagerlock "github.com/RichardKnop/machinery/v1/locks/eager"
	"github.com/RichardKnop/machinery/v1/tasks"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
)

func TestDelayFor(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_760_000_000, 0)
	at := func(d time.Duration) *time.Time {
		eta := now.Add(d)
		return &eta
	}

	tests := []struct {
		name string
		eta  *time.Time
		want time.Duration
	}{
		{"no eta", nil, 0},
		{"past", at(-time.Second), 0},
		{"now", at(0), 0},
		{"a nanosecond", at(1), time.Second},
		{"sub-second", at(2 * time.Millisecond), time.Second},
		{"just under a second", at(time.Second - 1), time.Second},
		{"whole second", at(time.Second), time.Second},
		{"just over a second", at(time.Second + 1), 2 * time.Second},
		{"jittered", at(2999 * time.Millisecond), 3 * time.Second},
		{"minutes", at(10 * time.Minute), 10 * time.Minute},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, delayFor(tt.eta, now))
		})
	}
}

func TestDelayQueueMatchesMachinery(t *testing.T) {
	t.Parallel()

	require.Equal(t, "delay.3000.machinery_exchange.machinery_tasks", delayQueueName(3*time.Second))
	require.Equal(t, amqp.Table{
		"x-dead-letter-exchange":    "machinery_exchange",
		"x-dead-letter-routing-key": "machinery_tasks",
		"x-message-ttl":             int64(3000),
		"x-expires":                 int64(6000),
	}, delayQueueArgs(3*time.Second))
}

func TestDecodeSignature(t *testing.T) {
	t.Parallel()

	signature, err := decodeSignature([]byte(`{"Name":"task","Args":[{"Type":"uint64","Value":18446744073709551615}]}`))
	require.NoError(t, err)
	require.Equal(t, "task", signature.Name)
	require.Equal(t, "18446744073709551615", signature.Args[0].Value.(interface{ String() string }).String(), "large integers keep their precision")

	for name, body := range map[string]string{"empty": "", "not json": "task", "wrong shape": `{"Name":1}`} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := decodeSignature([]byte(body))
			require.ErrorContains(t, err, "decode task")
		})
	}
}

// settlement records how a delivery was settled.
type settlement struct{ acked, nacked, rejected, requeue bool }

func (s *settlement) Ack(uint64, bool) error { s.acked = true; return nil }

func (s *settlement) Nack(_ uint64, _, requeue bool) error {
	s.nacked, s.requeue = true, requeue
	return nil
}

func (s *settlement) Reject(_ uint64, requeue bool) error {
	s.rejected, s.requeue = true, requeue
	return nil
}

type processorFunc func(*tasks.Signature) error

func (f processorFunc) Process(signature *tasks.Signature) error { return f(signature) }
func (processorFunc) CustomQueue() string                        { return "" }
func (processorFunc) PreConsumeHandler() bool                    { return true }

func testBroker(url string) *broker {
	b := newBroker(newConfig(url), log.NewNopLogger())
	b.SetRegisteredTaskNames([]string{"task"})

	return b
}

const unreachableURL = "amqp://guest:guest@127.0.0.1:1/"

func TestBrokerHandleSettlesDelivery(t *testing.T) {
	t.Parallel()

	notRun := func(*broker, *tasks.Signature) error { return errors.New("ran a task that should be dropped") }

	tests := []struct {
		name    string
		body    string
		process func(*broker, *tasks.Signature) error
		want    settlement
	}{
		{"acks a processed task", `{"Name":"task"}`, func(*broker, *tasks.Signature) error { return nil }, settlement{acked: true}},
		{"drops an undecodable task", "{", notRun, settlement{rejected: true}},
		{"drops a task with no registered processor", `{"Name":"other"}`, notRun, settlement{rejected: true}},
		{"drops a task machinery could not run", `{"Name":"task"}`, func(*broker, *tasks.Signature) error {
			return errors.New("malformed")
		}, settlement{rejected: true}},
		{"requeues a task whose retry was not queued", `{"Name":"task"}`, func(b *broker, signature *tasks.Signature) error {
			return b.Publish(context.Background(), signature)
		}, settlement{nacked: true, requeue: true}},
		{"a failed publish of another signature does not requeue", `{"Name":"task"}`, func(b *broker, _ *tasks.Signature) error {
			_ = b.Publish(context.Background(), &tasks.Signature{Name: "task"})
			return nil
		}, settlement{acked: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			b, got := testBroker(unreachableURL), new(settlement)
			start := time.Now()
			b.handle(amqp.Delivery{Acknowledger: got, Body: []byte(tt.body)}, processorFunc(func(signature *tasks.Signature) error {
				return tt.process(b, signature)
			}))

			require.Equal(t, tt.want, *got)
			require.Equal(t, tt.want.requeue, time.Since(start) >= requeuePause, "only a requeue pauses")
			b.retryUnqueued.Range(func(any, any) bool {
				t.Error("signature still tracked after its delivery was settled")
				return false
			})
		})
	}
}

// machinery must keep retrying a task by publishing the signature it was
// handed: retryUnqueued is keyed by it.
func TestBrokerHandleRequeuesWhenMachineryCannotQueueRetry(t *testing.T) {
	t.Parallel()

	b := testBroker(unreachableURL)
	server := machinery.NewServerWithBrokerBackendLock(b.GetConfig(), b, nullbackend.New(), eagerlock.New())
	for name, fail := range map[string]func() error{
		"retryLater": func() error { return tasks.NewErrRetryTaskLater("later", time.Second) },
		"retryCount": func() error { return errors.New("failed") },
	} {
		require.NoError(t, server.RegisterTask(name, fail))
	}

	for _, body := range []string{`{"Name":"retryLater"}`, `{"Name":"retryCount","RetryCount":1}`} {
		got := new(settlement)
		b.handle(amqp.Delivery{Acknowledger: got, Body: []byte(body)}, newWorker(server))
		require.Equal(t, settlement{nacked: true, requeue: true}, *got, body)
	}
}

func TestBrokerPublish(t *testing.T) {
	t.Parallel()

	b := testBroker(unreachableURL)
	require.ErrorContains(t, b.Publish(context.Background(), &tasks.Signature{Name: "task"}), "dial amqp")

	unmarshalable := &tasks.Signature{Name: "task", Args: []tasks.Arg{{Value: make(chan int)}}}
	require.ErrorContains(t, b.Publish(context.Background(), unmarshalable), "marshal task")
}

func TestBrokerStop(t *testing.T) {
	t.Parallel()

	t.Run("wait reports whether the broker is still running", func(t *testing.T) {
		t.Parallel()

		b := testBroker(unreachableURL)
		require.True(t, b.wait(time.Millisecond))

		b.StopConsuming()
		start := time.Now()
		require.False(t, b.wait(time.Minute))
		require.Less(t, time.Since(start), time.Second)
	})

	t.Run("stopping twice is safe and later consumers return at once", func(t *testing.T) {
		t.Parallel()

		b := testBroker(unreachableURL)
		b.StopConsuming()
		b.StopConsuming()

		retry, err := b.StartConsuming("tag", 1, processorFunc(func(*tasks.Signature) error { return nil }))
		require.False(t, retry)
		require.NoError(t, err)
	})

	t.Run("stop ends a consumer that cannot connect", func(t *testing.T) {
		t.Parallel()

		b := testBroker(unreachableURL)
		done := make(chan struct{})
		go func() {
			defer close(done)

			retry, err := b.StartConsuming("tag", 1, processorFunc(func(*tasks.Signature) error { return nil }))
			require.False(t, retry)
			require.NoError(t, err)
		}()

		time.Sleep(100 * time.Millisecond)
		b.StopConsuming()

		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("consumer still running after StopConsuming")
		}
	})
}

func TestConsumerEnded(t *testing.T) {
	t.Parallel()

	failed := make(chan *amqp.Error, 1)
	failed <- &amqp.Error{Code: amqp.PreconditionFailed, Reason: "delivery acknowledgement timed out"}
	err := consumerEnded(failed)
	require.ErrorContains(t, err, "consumer channel closed")
	require.ErrorContains(t, err, "delivery acknowledgement timed out")

	closedCleanly := make(chan *amqp.Error)
	close(closedCleanly)
	require.EqualError(t, consumerEnded(closedCleanly), "consumer cancelled by RabbitMQ")

	require.EqualError(t, consumerEnded(make(chan *amqp.Error)), "consumer cancelled by RabbitMQ", "an open channel does not block")
}

func TestBrokerHandleLogsFailedSettlement(t *testing.T) {
	t.Parallel()

	var warned []string
	nop := func(string, ...any) {}
	b := testBroker(unreachableURL)
	b.logger = fakeLogger{debug: nop, info: nop, err: nop, warn: func(msg string, _ ...any) { warned = append(warned, msg) }}

	// a delivery with no acknowledger cannot be settled
	b.handle(amqp.Delivery{Body: []byte(`{"Name":"task"}`)}, processorFunc(func(*tasks.Signature) error { return nil }))

	require.Len(t, warned, 1)
	require.Contains(t, warned[0], "could not settle task")
}
