package queue

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"cosmossdk.io/log"
	"github.com/RichardKnop/machinery/v1"
	nullbackend "github.com/RichardKnop/machinery/v1/backends/null"
	"github.com/RichardKnop/machinery/v1/config"
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

// amqpTestConfig skips the test without a RabbitMQ. It gives each test a queue
// of its own, so tests and test processes can share one.
func amqpTestConfig(tb testing.TB) *config.Config {
	tb.Helper()

	url := os.Getenv(amqpTestURLEnv)
	if url == "" {
		tb.Skipf("%s is not set", amqpTestURLEnv)
	}

	cnf := newConfig(url)
	cnf.DefaultQueue = fmt.Sprintf("%s.%s.%d", QName, tb.Name(), time.Now().UnixNano())

	return cnf
}

func deleteQueues(tb testing.TB, url string, names ...string) {
	tb.Helper()

	conn, err := amqp.Dial(url)
	require.NoError(tb, err)
	defer func() { _ = conn.Close() }()

	ch, err := conn.Channel()
	require.NoError(tb, err)

	for _, name := range names {
		_, _ = ch.QueueDelete(name, false, false, false)
	}
}

func newAMQPFixture(t *testing.T) *amqpFixture {
	t.Helper()

	cnf := amqpTestConfig(t)
	url := cnf.Broker

	t.Parallel()

	b := newBroker(cnf, log.NewNopLogger())
	b.requeuePause, b.consumeBackoff, b.handBackDelay = 50*time.Millisecond, 50*time.Millisecond, time.Second
	require.NoError(t, b.connect(t.Context()))

	conn, err := amqp.Dial(url)
	require.NoError(t, err)
	admin, err := conn.Channel()
	require.NoError(t, err)

	f := &amqpFixture{
		t:      t,
		broker: b,
		server: machinery.NewServerWithBrokerBackendLock(cnf, b, nullbackend.New(), eagerlock.New()),
		conn:   conn,
		admin:  admin,
	}
	t.Cleanup(func() {
		stopBroker(t, b)
		_, _ = admin.QueueDelete(b.queue, false, false, false)
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
	q, err := f.inspect(f.broker.queue)
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
	const n, shortDelay, longDelay = 50, 2 * time.Millisecond, 900 * time.Millisecond
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

	_, err := f.inspect(delayQueueName(f.broker.queue, time.Second))
	require.NoError(t, err, "delays share a queue per whole second")
}

func TestBrokerAMQPRequeuesTaskWhoseRetryWasNotQueued(t *testing.T) {
	f := newAMQPFixture(t)

	const n = workerConcurrency + 2
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
		q, err := f.inspect(f.broker.queue)
		return err == nil && q.Consumers == 1
	}, 5*time.Second, 20*time.Millisecond)

	outage.Store(true)
	for id := range n {
		f.send("retry", 0, id)
	}

	// the consumer stays connected while the publisher cannot dial
	pub := f.broker.pub
	url := pub.url
	require.NoError(t, pub.acquire(t.Context()))
	pub.drop()
	pub.url = unreachableURL
	pub.release()

	require.Eventually(t, func() bool { return runs.Load() >= 2*n }, 20*time.Second, 100*time.Millisecond,
		"tasks whose retry is not queued come back")
	require.Zero(t, completed.Load())

	outage.Store(false)
	require.NoError(t, pub.acquire(t.Context()))
	pub.url = url
	pub.release()

	require.Eventually(t, func() bool { return completed.Load() == n }, 20*time.Second, 100*time.Millisecond,
		"every task survives the publisher outage")

	stopBroker(t, f.broker)
}

func TestBrokerAMQPDropsTasksItCannotRun(t *testing.T) {
	f := newAMQPFixture(t)

	var ran atomic.Int64
	require.NoError(t, f.server.RegisterTask("known", func(int) error {
		ran.Add(1)
		return nil
	}))

	publish := func(body string) {
		require.NoError(t, f.admin.Publish(exchangeName, f.broker.queue, false, false, amqp.Publishing{Body: []byte(body)}))
	}
	for range 2 * workerConcurrency {
		publish(`not json`)
		publish(`{"Name":"known","Args":[{"Type":"no such type","Value":1}]}`)
	}
	f.send("known", 0, 1)
	f.startWorker()

	require.Eventually(t, func() bool { return ran.Load() == 1 && f.queued() == 0 }, 10*time.Second, 50*time.Millisecond,
		"tasks behind ones that cannot run still run, and none loops")
}

func TestBrokerAMQPSharedQueueWithDifferentProcessors(t *testing.T) {
	f := newAMQPFixture(t)

	// a second bridge on the same queue, with another processor set
	other := newBroker(f.broker.GetConfig(), log.NewNopLogger())
	other.handBackDelay = f.broker.handBackDelay
	otherServer := machinery.NewServerWithBrokerBackendLock(other.GetConfig(), other, nullbackend.New(), eagerlock.New())
	t.Cleanup(func() { stopBroker(t, other) })

	var ranHere, ranThere atomic.Int64
	require.NoError(t, f.server.RegisterTask("here", func(int) error {
		ranHere.Add(1)
		return nil
	}))
	require.NoError(t, otherServer.RegisterTask("there", func(int) error {
		ranThere.Add(1)
		return nil
	}))

	// more of the other bridge's tasks than the prefetch, ahead of this one's
	const n = workerConcurrency + 2
	for id := range n {
		f.send("there", 0, id)
	}
	for id := range n {
		f.send("here", 0, id)
	}

	f.startWorker()
	require.Eventually(t, func() bool { return ranHere.Load() == n }, 10*time.Second, 20*time.Millisecond,
		"tasks behind another bridge's tasks still run")
	require.Zero(t, ranThere.Load())

	newWorker(otherServer).LaunchAsync(make(chan error, 1))
	require.Eventually(t, func() bool { return ranThere.Load() == n }, 30*time.Second, 50*time.Millisecond,
		"tasks handed back reach the bridge that has their processor")
}

func TestBrokerAMQPReturnsUnroutableTask(t *testing.T) {
	f := newAMQPFixture(t)

	// the publisher's channel outlives the queue
	_, err := f.admin.QueueDelete(f.broker.queue, false, false, false)
	require.NoError(t, err)

	_, err = f.server.SendTask(&tasks.Signature{Name: "unroutable"})
	require.ErrorContains(t, err, "task not routed")

	f.send("unroutable", 0, 1)
	require.Equal(t, 1, f.queued(), "the next publish redeclares the queue")
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
	_, err := f.admin.QueueDelete(f.broker.queue, false, false, false)
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		q, err := f.inspect(f.broker.queue)
		return err == nil && q.Consumers == 1
	}, 10*time.Second, 20*time.Millisecond, "the consumer redeclares the queue and resumes")

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
		select {
		case <-started:
		case <-time.After(10 * time.Second):
			t.Fatal("tasks not delivered")
		}
	}
	time.Sleep(50 * time.Millisecond)
	require.Len(t, started, 0, "no more than workerConcurrency tasks run at once")

	stopped := make(chan struct{})
	go func() {
		f.broker.StopConsuming()
		close(stopped)
	}()

	select {
	case <-stopped:
		t.Fatal("StopConsuming returned with tasks in flight")
	case <-time.After(100 * time.Millisecond):
	}

	close(release)
	select {
	case <-stopped:
	case <-time.After(10 * time.Second):
		t.Fatal("StopConsuming blocked after tasks finished")
	}
	require.EqualValues(t, workerConcurrency, finished.Load())
	require.Equal(t, workerConcurrency, f.queued(), "tasks not started stay queued for the next run")

	_, err := f.server.SendTask(&tasks.Signature{Name: "slow"})
	require.ErrorContains(t, err, errPublisherClosed.Error(), "a stopped broker opens no new connection")

	require.NoError(t, f.broker.pub.acquire(t.Context()))
	defer f.broker.pub.release()
	require.Nil(t, f.broker.pub.link, "stopping closes the publisher connection")
}

func TestBrokerAMQPRedialsAfterFailedPublish(t *testing.T) {
	f := newAMQPFixture(t)

	// a delay queue declared with other arguments fails the publish and its channel
	const delay = 7 * time.Second
	_, err := f.admin.QueueDeclare(delayQueueName(f.broker.queue, delay), true, false, false, false, amqp.Table{"x-expires": int64(60_000)})
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = f.admin.QueueDelete(delayQueueName(f.broker.queue, delay), false, false, false) })

	eta := time.Now().Add(delay)
	_, err = f.server.SendTask(&tasks.Signature{Name: "redial", ETA: &eta})
	require.ErrorContains(t, err, "declare delay queue")

	require.NoError(t, f.broker.pub.acquire(t.Context()))
	require.Nil(t, f.broker.pub.link, "a failed link is dropped")
	f.broker.pub.release()

	f.send("redial", 0, 1)
	require.Equal(t, 1, f.queued())
}

func TestBrokerAMQPClosedChannelErrors(t *testing.T) {
	f := newAMQPFixture(t)

	ch, err := f.conn.Channel()
	require.NoError(t, err)
	require.NoError(t, ch.Close())

	require.ErrorContains(t, declareTaskQueue(ch, f.broker.queue), "declare exchange")
	require.ErrorContains(t, publishConfirmed(t.Context(), ch, nil, exchangeName, f.broker.queue, amqp.Publishing{}), "publish task")
	_, err = declareDelayQueue(ch, f.broker.queue, time.Second)
	require.ErrorContains(t, err, "declare delay queue")
}

func TestConnectorAMQPRunsTasks(t *testing.T) {
	cnf := amqpTestConfig(t)
	qc, err := newConnector(cnf)
	require.NoError(t, err)
	t.Cleanup(func() { deleteQueues(t, cnf.Broker, cnf.DefaultQueue) })

	ran := make(chan struct{}, 1)
	require.NoError(t, qc.Server.RegisterTask("connector", func() error {
		ran <- struct{}{}
		return nil
	}))

	qc.StartWorker()
	qc.StartWorker()

	_, err = qc.Server.SendTask(&tasks.Signature{Name: "connector"})
	require.NoError(t, err)

	within(t, 10*time.Second, func() { <-ran })
	within(t, 10*time.Second, func() {
		qc.StopWorker()
		qc.StopWorker()
	})
}

func TestBrokerAMQPConsumerReturnsWhenStopped(t *testing.T) {
	f := newAMQPFixture(t)

	type result struct {
		retry bool
		err   error
	}
	done := make(chan result, 1)
	go func() {
		retry, err := f.broker.StartConsuming("tag", 1, processorFunc(func(*tasks.Signature) error { return nil }))
		done <- result{retry, err}
	}()

	require.Eventually(t, func() bool {
		q, err := f.inspect(f.broker.queue)
		return err == nil && q.Consumers == 1
	}, 10*time.Second, 20*time.Millisecond)

	stopBroker(t, f.broker)

	select {
	case got := <-done:
		require.Equal(t, result{}, got, "machinery is never asked to retry")
	case <-time.After(10 * time.Second):
		t.Fatal("consumer still running after StopConsuming")
	}
}

func TestLinkAMQPEndsWithItsContext(t *testing.T) {
	f := newAMQPFixture(t)

	l, err := openLink(t.Context(), f.broker.url, f.broker.queue, func(*link) error { return nil })
	require.NoError(t, err)
	require.False(t, l.conn.IsClosed(), "the link outlives the context it was opened with only until that ends")

	ctx, cancel := context.WithCancel(t.Context())
	stop := l.abortOn(ctx)
	require.True(t, stop(), "a stopped abort leaves the link alone")
	cancel()
	require.NoError(t, declareTaskQueue(l.ch, f.broker.queue))

	ctx, cancel = context.WithCancel(t.Context())
	l.abortOn(ctx)
	cancel()
	require.Eventually(t, l.conn.IsClosed, 5*time.Second, 10*time.Millisecond, "ending the context kills the socket")
	require.Error(t, declareTaskQueue(l.ch, f.broker.queue))

	closing, err := openLink(t.Context(), f.broker.url, f.broker.queue, func(*link) error { return nil })
	require.NoError(t, err)
	closing.close()
	require.True(t, closing.conn.IsClosed())
}

func BenchmarkBrokerAMQPPublishDelayed(b *testing.B) {
	const delay = 3 * time.Second

	cnf := amqpTestConfig(b)
	br := newBroker(cnf, log.NewNopLogger())
	server := machinery.NewServerWithBrokerBackendLock(cnf, br, nullbackend.New(), eagerlock.New())
	b.Cleanup(func() {
		stopBroker(b, br)
		deleteQueues(b, cnf.Broker, delayQueueName(cnf.DefaultQueue, delay), cnf.DefaultQueue)
	})

	for i := 0; b.Loop(); i++ {
		eta := time.Now().Add(delay)
		if _, err := server.SendTask(&tasks.Signature{Name: "bench", ETA: &eta, Args: []tasks.Arg{{Type: "string", Value: fmt.Sprint(i)}}}); err != nil {
			b.Fatal(err)
		}
	}
}
