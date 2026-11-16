package queue

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"cosmossdk.io/log"
	"github.com/RichardKnop/machinery/v1/brokers/iface"
	"github.com/RichardKnop/machinery/v1/common"
	"github.com/RichardKnop/machinery/v1/config"
	"github.com/RichardKnop/machinery/v1/tasks"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	defaultRequeuePause = time.Second

	// defaultHandBackDelay is how long a task this bridge has no processor for
	// waits before it is delivered again.
	defaultHandBackDelay = 5 * time.Second

	defaultConsumeBackoff = time.Second
	maxConsumeBackoff     = 30 * time.Second
)

// broker is machinery's transport to RabbitMQ, in place of machinery's own AMQP
// broker. Every task goes through the one queue.
type broker struct {
	common.Broker

	url    string
	queue  string
	logger log.Logger
	pub    *publisher

	requeuePause   time.Duration
	handBackDelay  time.Duration
	consumeBackoff time.Duration

	// stopMu orders a consumer starting against StopConsuming waiting for them
	stopMu    sync.Mutex
	stop      chan struct{}
	consumers sync.WaitGroup

	// retryUnqueued flags the signatures in flight whose publish failed.
	// machinery retries a task by publishing the signature it was handed, and
	// reports a failure there with the same opaque error as a malformed task.
	retryUnqueued sync.Map

	// unregisteredWarned holds the task names already reported as unregistered
	unregisteredWarned sync.Map
}

func newBroker(cnf *config.Config, logger log.Logger) *broker {
	return &broker{
		Broker: common.NewBroker(cnf),
		url:    cnf.Broker,
		queue:  cnf.DefaultQueue,
		logger: logger,
		pub:    newPublisher(cnf.Broker, cnf.DefaultQueue),

		requeuePause:   defaultRequeuePause,
		handBackDelay:  defaultHandBackDelay,
		consumeBackoff: defaultConsumeBackoff,
		stop:           make(chan struct{}),
	}
}

var _ iface.Broker = (*broker)(nil)

// connect opens the publisher connection and declares the task queue.
func (b *broker) connect(ctx context.Context) error { return b.pub.connect(ctx) }

// Publish queues signature, through a delay queue when its ETA is in the future.
func (b *broker) Publish(ctx context.Context, signature *tasks.Signature) error {
	body, err := json.Marshal(signature)
	if err != nil {
		return fmt.Errorf("marshal task: %w", err)
	}

	// like a marshal error, this fails every attempt: it is not a retry to hand back
	if err := amqp.Table(signature.Headers).Validate(); err != nil {
		return fmt.Errorf("task headers: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, publishTimeout)
	defer cancel()

	if err := b.pub.publish(ctx, signature, body); err != nil {
		if unqueued, ok := b.retryUnqueued.Load(signature); ok {
			unqueued.(*atomic.Bool).Store(true)
		}

		return err
	}

	return nil
}

// StartConsuming delivers tasks to processor until StopConsuming, reconnecting
// with backoff whenever the connection or channel fails. It never asks
// machinery to retry: that loop has no backoff.
func (b *broker) StartConsuming(consumerTag string, concurrency int, processor iface.TaskProcessor) (bool, error) {
	b.stopMu.Lock()
	select {
	case <-b.stop:
		b.stopMu.Unlock()
		return false, nil
	default:
	}
	b.consumers.Add(1)
	b.stopMu.Unlock()

	defer b.consumers.Done()

	backoff := b.consumeBackoff
	for {
		session, err := b.openConsumer(consumerTag, concurrency)
		if err == nil {
			backoff = b.consumeBackoff
			err = b.serve(session, processor)
		}

		if err == nil {
			return false, nil
		}

		b.logger.Error("Bridge queue consumer failed, reconnecting", "err", err, "in", backoff)

		if !b.wait(backoff) {
			return false, nil
		}

		backoff = nextBackoff(backoff)
	}
}

func nextBackoff(backoff time.Duration) time.Duration {
	return min(2*backoff, maxConsumeBackoff)
}

// consumer is one consuming session on its own link.
type consumer struct {
	link       *link
	deliveries <-chan amqp.Delivery
	closed     <-chan *amqp.Error
}

func (b *broker) openConsumer(consumerTag string, concurrency int) (*consumer, error) {
	ctx, cancel := context.WithTimeout(context.Background(), publishTimeout)
	defer cancel()

	session := new(consumer)

	l, err := openLink(ctx, b.url, b.queue, func(l *link) error {
		// Without a prefetch limit RabbitMQ pushes the whole backlog at once, and
		// closes the channel when a delivery then waits past its ack timeout.
		if err := l.ch.Qos(concurrency, 0, false); err != nil {
			return fmt.Errorf("set prefetch: %w", err)
		}

		deliveries, err := l.ch.Consume(b.queue, consumerTag, false, false, false, false, nil)
		if err != nil {
			return fmt.Errorf("consume: %w", err)
		}

		session.deliveries, session.closed = deliveries, l.ch.NotifyClose(make(chan *amqp.Error, 1))

		return nil
	})
	if err != nil {
		return nil, err
	}

	session.link = l

	return session, nil
}

// serve hands deliveries to processor until the session fails or the broker
// stops, which is the only time it returns nil.
func (b *broker) serve(session *consumer, processor iface.TaskProcessor) error {
	defer session.link.close()

	// tasks in flight finish, and ack, before the connection closes
	var inFlight sync.WaitGroup
	defer inFlight.Wait()

	for {
		select {
		case <-b.stop:
			return nil
		case delivery, ok := <-session.deliveries:
			if !ok {
				return consumerEnded(session.closed)
			}

			delivery.Acknowledger = settler{delivery.Acknowledger, session.link}

			inFlight.Add(1)
			go func() {
				defer inFlight.Done()
				b.handle(delivery, processor)
			}()
		}
	}
}

// consumerEnded explains why deliveries stopped. RabbitMQ can cancel the
// consumer, as when the queue is deleted, and leave the channel open.
func consumerEnded(closed <-chan *amqp.Error) error {
	select {
	case reason := <-closed:
		if reason != nil {
			return fmt.Errorf("consumer channel closed: %w", reason)
		}
	default:
	}

	return errors.New("consumer cancelled by RabbitMQ")
}

// handle runs one delivered task and settles it with the broker.
func (b *broker) handle(delivery amqp.Delivery, processor iface.TaskProcessor) {
	signature, err := decodeSignature(delivery.Body)
	if err != nil {
		b.logger.Error("Bridge queue: dropping undecodable task", "err", err)
		b.settle(delivery.Reject(false))
		return
	}

	if !b.IsTaskRegistered(signature.Name) {
		b.handBack(delivery, signature)
		return
	}

	unqueued := new(atomic.Bool)
	b.retryUnqueued.Store(signature, unqueued)
	err = processor.Process(signature)
	b.retryUnqueued.Delete(signature)

	switch {
	case unqueued.Load():
		b.logger.Error("Bridge queue: retry not queued, returning task to the queue", "taskName", signature.Name, "uuid", signature.UUID, "err", err)
		b.requeue(delivery)
	case err != nil:
		b.logger.Error("Bridge queue: dropping task machinery could not run", "taskName", signature.Name, "uuid", signature.UUID, "err", err)
		b.settle(delivery.Reject(false))
	default:
		b.settle(delivery.Ack(false))
	}
}

// handBack returns a task this bridge has no processor for to the queue after
// a delay: a bridge sharing the queue may have one. Requeueing it directly
// would redeliver it at once, ahead of the tasks behind it.
func (b *broker) handBack(delivery amqp.Delivery, signature *tasks.Signature) {
	if _, warned := b.unregisteredWarned.LoadOrStore(signature.Name, struct{}{}); !warned {
		b.logger.Warn("Bridge queue: handing back tasks with no registered processor", "taskName", signature.Name)
	}

	eta := time.Now().Add(b.handBackDelay)
	signature.ETA = &eta

	if err := b.Publish(context.Background(), signature); err != nil {
		b.logger.Error("Bridge queue: could not hand task back, returning it to the queue", "taskName", signature.Name, "uuid", signature.UUID, "err", err)
		b.requeue(delivery)
		return
	}

	b.settle(delivery.Ack(false))
}

// requeue returns a task to the head of the queue, after a pause so it does
// not spin between RabbitMQ and the worker while the publisher is failing.
func (b *broker) requeue(delivery amqp.Delivery) {
	b.wait(b.requeuePause)
	b.settle(delivery.Nack(false, true))
}

// settle logs a failed ack, nack or reject.
func (b *broker) settle(err error) {
	if err != nil {
		b.logger.Warn("Bridge queue: could not settle task, RabbitMQ will redeliver it", "err", err)
	}
}

// decodeSignature keeps numeric arguments as json.Number, as machinery does.
func decodeSignature(body []byte) (*tasks.Signature, error) {
	signature := new(tasks.Signature)

	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()

	if err := decoder.Decode(signature); err != nil {
		return nil, fmt.Errorf("decode task: %w", err)
	}

	return signature, nil
}

// wait pauses for d and reports false if the broker stopped meanwhile.
func (b *broker) wait(d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-b.stop:
		return false
	case <-timer.C:
		return true
	}
}

// StopConsuming stops delivering tasks and returns once those in flight have
// finished. It is safe to call more than once.
func (b *broker) StopConsuming() {
	b.stopMu.Lock()
	select {
	case <-b.stop:
	default:
		close(b.stop)
	}
	b.stopMu.Unlock()

	b.consumers.Wait()

	b.pub.close()
}
