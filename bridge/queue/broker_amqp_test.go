package queue

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
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

// amqpTestURLEnv names a RabbitMQ these tests may use and purge, for example
// amqp://guest:guest@127.0.0.1:5672/. They are skipped when it is unset.
const amqpTestURLEnv = "BRIDGE_TEST_AMQP_URL"

// amqpFixture is a machinery server on the bridge broker, with a worker.
type amqpFixture struct {
	t      *testing.T
	broker *broker
	server *machinery.Server
	conn   *amqp.Connection
	admin  *amqp.Channel
}

func newAMQPFixture(t *testing.T) *amqpFixture {
	t.Helper()

	url := os.Getenv(amqpTestURLEnv)
	if url == "" {
		t.Skipf("%s is not set", amqpTestURLEnv)
	}

	cnf := newConfig(url)
	b := newBroker(cnf, log.NewNopLogger())
	require.NoError(t, b.connect())

	conn, err := amqp.Dial(url)
	require.NoError(t, err)
	admin, err := conn.Channel()
	require.NoError(t, err)
	_, err = admin.QueuePurge(QName, false)
	require.NoError(t, err)

	f := &amqpFixture{
		t:      t,
		broker: b,
		server: machinery.NewServerWithBrokerBackendLock(cnf, b, nullbackend.New(), eagerlock.New()),
		conn:   conn,
		admin:  admin,
	}
	t.Cleanup(func() {
		b.StopConsuming()
		_ = conn.Close()
	})

	return f
}

func (f *amqpFixture) startWorker() {
	f.t.Helper()
	newWorker(f.server).LaunchAsync(make(chan error, 1))
}

func (f *amqpFixture) send(name string, delay time.Duration, id int) {
	f.t.Helper()

	signature := &tasks.Signature{Name: name, Args: []tasks.Arg{{Type: "int", Value: id}}}
	if delay > 0 {
		eta := time.Now().Add(delay)
		signature.ETA = &eta
	}

	_, err := f.server.SendTask(signature)
	require.NoError(f.t, err)
}

// inspect reads the task queue on a throwaway channel: a failed passive
// declare closes the channel it ran on.
func (f *amqpFixture) inspect(name string) (amqp.Queue, error) {
	ch, err := f.conn.Channel()
	require.NoError(f.t, err)
	defer func() { _ = ch.Close() }()

	return ch.QueueDeclarePassive(name, true, false, false, false, nil)
}

func (f *amqpFixture) queued() int {
	q, err := f.inspect(QName)
	require.NoError(f.t, err)

	return q.Messages
}

func TestBrokerAMQPDeliversDirectAndDelayedTasks(t *testing.T) {
	f := newAMQPFixture(t)

	var mu sync.Mutex
	ranAt := map[int]time.Time{}
	require.NoError(t, f.server.RegisterTask("deliver", func(id int) error {
		mu.Lock()
		defer mu.Unlock()
		ranAt[id] = time.Now()
		return nil
	}))
	f.startWorker()

	// a delay this short outlives its own queue on machinery's broker
	const n, shortDelay, longDelay = 50, 2 * time.Millisecond, 1500 * time.Millisecond
	start := time.Now()
	for id := range n {
		f.send("deliver", 0, id)
		f.send("deliver", shortDelay, n+id)
		f.send("deliver", longDelay, 2*n+id)
	}

	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(ranAt) == 3*n
	}, 10*time.Second, 50*time.Millisecond)

	for id := 2 * n; id < 3*n; id++ {
		require.GreaterOrEqual(t, ranAt[id].Sub(start), longDelay, "a task runs no earlier than its ETA")
	}

	_, err := f.inspect(delayQueueName(2 * time.Second))
	require.NoError(t, err, "delays share a queue per whole second")
}

func TestBrokerAMQPRequeuesTaskWhoseRetryWasNotQueued(t *testing.T) {
	f := newAMQPFixture(t)

	const n = 20
	var outage atomic.Bool
	var runs, completed atomic.Int64
	require.NoError(t, f.server.RegisterTask("retry", func(int) error {
		runs.Add(1)
		if outage.Load() {
			return tasks.NewErrRetryTaskLater("later", time.Second)
		}
		completed.Add(1)
		return nil
	}))

	f.startWorker()
	require.Eventually(t, func() bool {
		q, err := f.inspect(QName)
		return err == nil && q.Consumers == 1
	}, 5*time.Second, 20*time.Millisecond)

	outage.Store(true)
	for id := range n {
		f.send("retry", 0, id)
	}

	// the consumer stays connected while the publisher cannot dial
	url := f.broker.url
	f.broker.pubMu.Lock()
	f.broker.closePublisher()
	f.broker.url = "amqp://guest:guest@127.0.0.1:1/"
	f.broker.pubMu.Unlock()

	require.Eventually(t, func() bool { return runs.Load() >= 2*n }, 20*time.Second, 100*time.Millisecond,
		"tasks whose retry is not queued come back")
	require.Zero(t, completed.Load())

	outage.Store(false)
	f.broker.pubMu.Lock()
	f.broker.url = url
	f.broker.pubMu.Unlock()

	require.Eventually(t, func() bool { return completed.Load() == n }, 20*time.Second, 100*time.Millisecond,
		"every task survives the publisher outage")

	stopped := make(chan struct{})
	go func() {
		f.broker.StopConsuming()
		close(stopped)
	}()
	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		t.Fatal("StopConsuming blocked after failed publishes")
	}
}

func TestBrokerAMQPDropsTasksItCannotRun(t *testing.T) {
	f := newAMQPFixture(t)

	var ran atomic.Int64
	require.NoError(t, f.server.RegisterTask("known", func(int) error {
		ran.Add(1)
		return nil
	}))

	publish := func(body string) {
		require.NoError(t, f.admin.Publish(exchangeName, QName, false, false, amqp.Publishing{Body: []byte(body)}))
	}
	for range 2 * workerConcurrency {
		publish(`{"Name":"unregistered"}`)
		publish(`not json`)
		publish(`{"Name":"known","Args":[{"Type":"no such type","Value":1}]}`)
	}
	f.send("known", 0, 1)
	f.startWorker()

	require.Eventually(t, func() bool { return ran.Load() == 1 && f.queued() == 0 }, 10*time.Second, 50*time.Millisecond,
		"tasks behind undeliverable ones still run, and none loops")
}

func TestBrokerAMQPConsumerReconnects(t *testing.T) {
	f := newAMQPFixture(t)

	var ran atomic.Int64
	require.NoError(t, f.server.RegisterTask("reconnect", func(int) error {
		ran.Add(1)
		return nil
	}))
	f.startWorker()

	f.send("reconnect", 0, 1)
	require.Eventually(t, func() bool { return ran.Load() == 1 }, 5*time.Second, 20*time.Millisecond)

	// deleting the queue cancels the consumer, as a failed channel would
	_, err := f.admin.QueueDelete(QName, false, false, false)
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		q, err := f.inspect(QName)
		return err == nil && q.Consumers == 1
	}, 3*minConsumeBackoff+5*time.Second, 100*time.Millisecond, "the consumer redeclares the queue and resumes")

	f.send("reconnect", 0, 2)
	require.Eventually(t, func() bool { return ran.Load() == 2 }, 5*time.Second, 20*time.Millisecond)
}

func TestBrokerAMQPStopWaitsForTasksInFlight(t *testing.T) {
	f := newAMQPFixture(t)

	started, release := make(chan struct{}, 2*workerConcurrency), make(chan struct{})
	var finished atomic.Int64
	require.NoError(t, f.server.RegisterTask("slow", func(int) error {
		started <- struct{}{}
		<-release
		finished.Add(1)
		return nil
	}))
	f.startWorker()

	for id := range 2 * workerConcurrency {
		f.send("slow", 0, id)
	}
	for range workerConcurrency {
		<-started
	}
	time.Sleep(200 * time.Millisecond)
	require.Len(t, started, 0, "no more than workerConcurrency tasks run at once")

	stopped := make(chan struct{})
	go func() {
		f.broker.StopConsuming()
		close(stopped)
	}()

	select {
	case <-stopped:
		t.Fatal("StopConsuming returned with tasks in flight")
	case <-time.After(300 * time.Millisecond):
	}

	close(release)
	<-stopped
	require.EqualValues(t, workerConcurrency, finished.Load())
	require.Equal(t, workerConcurrency, f.queued(), "tasks not started stay queued for the next run")
}

func BenchmarkBrokerAMQPPublishDelayed(b *testing.B) {
	url := os.Getenv(amqpTestURLEnv)
	if url == "" {
		b.Skipf("%s is not set", amqpTestURLEnv)
	}

	br := newBroker(newConfig(url), log.NewNopLogger())
	server := machinery.NewServerWithBrokerBackendLock(br.GetConfig(), br, nullbackend.New(), eagerlock.New())
	defer br.StopConsuming()

	for i := 0; b.Loop(); i++ {
		eta := time.Now().Add(3 * time.Second)
		if _, err := server.SendTask(&tasks.Signature{Name: "bench", ETA: &eta, Args: []tasks.Arg{{Type: "string", Value: fmt.Sprint(i)}}}); err != nil {
			b.Fatal(err)
		}
	}
}
