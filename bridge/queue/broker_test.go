package queue

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
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

func TestDelayQueue(t *testing.T) {
	t.Parallel()

	require.Equal(t, "machinery_tasks.delay.3000", delayQueueName(QName, 3*time.Second))
	require.Equal(t, amqp.Table{
		"x-dead-letter-exchange":    "machinery_exchange",
		"x-dead-letter-routing-key": "machinery_tasks",
		"x-message-ttl":             int64(3000),
		"x-expires":                 int64(66_000),
	}, delayQueueArgs(QName, 3*time.Second))
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
	b.requeuePause = 50 * time.Millisecond

	return b
}

const unreachableURL = "amqp://guest:guest@127.0.0.1:1/"

func TestBrokerHandleSettlesDelivery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		body    string
		process func(*broker, *tasks.Signature) error
		want    settlement
	}{
		{"acks a processed task", `{"Name":"task"}`, func(*broker, *tasks.Signature) error { return nil }, settlement{acked: true}},
		{"drops an undecodable task", "{", nil, settlement{rejected: true}},
		{"requeues a task with no registered processor it cannot hand back", `{"Name":"other"}`, nil, settlement{nacked: true, requeue: true}},
		{"drops a task whose retry can never be queued", `{"Name":"task","Headers":{"n":1}}`, func(b *broker, signature *tasks.Signature) error {
			return b.Publish(context.Background(), signature)
		}, settlement{rejected: true}},
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
				if tt.process == nil {
					t.Error("ran a task that should be dropped")
					return nil
				}
				return tt.process(b, signature)
			}))

			require.Equal(t, tt.want, *got)
			if tt.want.requeue {
				require.GreaterOrEqual(t, time.Since(start), b.requeuePause, "a requeue pauses first")
			}
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

	badHeaders := &tasks.Signature{Name: "task", Headers: tasks.Headers{"n": struct{}{}}}
	require.ErrorContains(t, b.Publish(context.Background(), badHeaders), "task headers")

	expired, cancel := context.WithCancel(context.Background())
	cancel()
	require.NoError(t, b.pub.acquire(context.Background()))
	require.ErrorIs(t, b.Publish(expired, &tasks.Signature{Name: "task"}), context.Canceled, "waiting for the publisher ends with the context")
	b.pub.release()
}

// stopBroker fails the test rather than hang it when StopConsuming blocks.
func stopBroker(t *testing.T, b *broker) {
	t.Helper()

	stopped := make(chan struct{})
	go func() {
		b.StopConsuming()
		close(stopped)
	}()

	select {
	case <-stopped:
	case <-time.After(10 * time.Second):
		t.Error("StopConsuming blocked")
	}
}

func TestNextBackoff(t *testing.T) {
	t.Parallel()

	require.Equal(t, 2*time.Second, nextBackoff(time.Second))
	require.Equal(t, 30*time.Second, nextBackoff(15*time.Second))
	require.Equal(t, 30*time.Second, nextBackoff(16*time.Second))
	require.Equal(t, 30*time.Second, nextBackoff(30*time.Second))
}

func TestBrokerStop(t *testing.T) {
	t.Parallel()

	consume := func(b *broker) (bool, error) {
		return b.StartConsuming("tag", 1, processorFunc(func(*tasks.Signature) error { return nil }))
	}

	t.Run("wait reports whether the broker is still running", func(t *testing.T) {
		t.Parallel()

		b := testBroker(unreachableURL)
		require.True(t, b.wait(time.Millisecond))

		stopBroker(t, b)
		start := time.Now()
		require.False(t, b.wait(time.Minute))
		require.Less(t, time.Since(start), time.Second)
	})

	t.Run("stopping twice is safe and later consumers return at once", func(t *testing.T) {
		t.Parallel()

		b := testBroker(unreachableURL)
		stopBroker(t, b)
		stopBroker(t, b)

		retry, err := consume(b)
		require.False(t, retry)
		require.NoError(t, err)
	})

	t.Run("a consumer that cannot connect backs off until stopped", func(t *testing.T) {
		t.Parallel()

		waits := make(chan time.Duration, 64)
		nop := func(string, ...any) {}
		b := testBroker(unreachableURL)
		b.consumeBackoff = 5 * time.Millisecond
		b.logger = fakeLogger{debug: nop, info: nop, warn: nop, err: func(_ string, keyVals ...any) {
			select {
			case waits <- keyVals[3].(time.Duration):
			default:
			}
		}}

		type result struct {
			retry bool
			err   error
		}
		done := make(chan result, 1)
		go func() {
			retry, err := consume(b)
			done <- result{retry, err}
		}()

		for _, want := range []time.Duration{5 * time.Millisecond, 10 * time.Millisecond, 20 * time.Millisecond} {
			select {
			case got := <-waits:
				require.Equal(t, want, got)
			case <-time.After(10 * time.Second):
				t.Fatal("consumer did not retry")
			}
		}

		stopBroker(t, b)

		select {
		case got := <-done:
			require.Equal(t, result{}, got, "machinery is never asked to retry")
		case <-time.After(10 * time.Second):
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

func TestDial(t *testing.T) {
	t.Parallel()

	expired, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()

	_, err := dial(expired, unreachableURL)
	require.ErrorContains(t, err, "dial amqp")

	// a listener that never answers the AMQP handshake
	silent, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = silent.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	within(t, 5*time.Second, func() {
		_, err := dial(ctx, "amqp://guest:guest@"+silent.Addr().String()+"/")
		require.ErrorContains(t, err, "dial amqp")
	})
}

// within fails the test rather than hang it when f blocks.
func within(t *testing.T, limit time.Duration, f func()) {
	t.Helper()

	done := make(chan struct{})
	go func() {
		defer close(done)
		f()
	}()

	select {
	case <-done:
	case <-time.After(limit):
		t.Fatalf("still blocked after %s", limit)
	}
}

// heartbeatingPeer is an AMQP server that goes as far as connection.tune, and
// connection.open-ok if answerOpen, then sends only heartbeats.
func heartbeatingPeer(t *testing.T, answerOpen bool) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })

	method := func(id byte, args ...byte) []byte {
		payload := append([]byte{0, 10, 0, id}, args...)
		frame := binary.BigEndian.AppendUint32([]byte{1, 0, 0}, uint32(len(payload)))
		return append(append(frame, payload...), 0xCE)
	}
	skipFrame := func(conn net.Conn) error {
		header := make([]byte, 7)
		if _, err := io.ReadFull(conn, header); err != nil {
			return err
		}
		_, err := io.CopyN(io.Discard, conn, int64(binary.BigEndian.Uint32(header[3:]))+1)
		return err
	}

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()

		start := method(10, 0, 9, 0, 0, 0, 0, 0, 0, 0, 5, 'P', 'L', 'A', 'I', 'N', 0, 0, 0, 5, 'e', 'n', '_', 'U', 'S')
		tune := method(30, 0, 0, 0, 2, 0, 0, 0, 1) // heartbeat every second
		_, _ = io.CopyN(io.Discard, conn, 8)       // protocol header
		_, _ = conn.Write(start)
		_ = skipFrame(conn) // start-ok
		_, _ = conn.Write(tune)
		_ = skipFrame(conn) // tune-ok
		_ = skipFrame(conn) // open
		if answerOpen {
			_, _ = conn.Write(method(41, 0))
		}

		go func() { _, _ = io.Copy(io.Discard, conn) }()
		for {
			if _, err := conn.Write([]byte{8, 0, 0, 0, 0, 0, 0, 0xCE}); err != nil {
				return
			}
			time.Sleep(50 * time.Millisecond)
		}
	}()

	return "amqp://guest:guest@" + listener.Addr().String() + "/"
}

// amqp091 moves its socket deadline on every heartbeat, so neither the
// handshake nor the close may rely on it.
func TestLinkGivesUpOnHeartbeatingPeer(t *testing.T) {
	t.Parallel()

	t.Run("handshake", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel()

		within(t, 5*time.Second, func() {
			_, err := dial(ctx, heartbeatingPeer(t, false))
			require.ErrorContains(t, err, "dial amqp")
		})
	})

	t.Run("close", func(t *testing.T) {
		t.Parallel()

		l, err := dial(context.Background(), heartbeatingPeer(t, true))
		require.NoError(t, err)

		start := time.Now()
		within(t, 5*time.Second, func() { l.closeWithin(300 * time.Millisecond) })
		require.GreaterOrEqual(t, time.Since(start), 300*time.Millisecond, "RabbitMQ gets the time to answer")
		require.True(t, l.conn.IsClosed())
	})
}

func TestPublisherClosed(t *testing.T) {
	t.Parallel()

	b := testBroker(unreachableURL)
	b.pub.close()
	b.pub.close()

	require.ErrorIs(t, b.Publish(context.Background(), &tasks.Signature{Name: "task"}), errPublisherClosed)
	require.ErrorIs(t, b.pub.connect(context.Background()), errPublisherClosed)
}

func TestConnectorStopWorkerWithoutWorker(t *testing.T) {
	t.Parallel()

	b := testBroker(unreachableURL)
	qc := &Connector{logger: log.NewNopLogger(), broker: b}
	qc.StopWorker()

	retry, err := b.StartConsuming("tag", 1, processorFunc(func(*tasks.Signature) error { return nil }))
	require.False(t, retry, "the broker is stopped even though no worker ran")
	require.NoError(t, err)

	(&Connector{logger: log.NewNopLogger()}).StopWorker()
}
