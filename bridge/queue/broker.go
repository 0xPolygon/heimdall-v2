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
	exchangeName = "machinery_exchange"

	// publishTimeout bounds one publish, broker confirmation included.
	publishTimeout = 15 * time.Second

	// requeuePause keeps a task whose retry could not be queued from spinning
	// between the broker and the worker while the publisher is failing.
	requeuePause = time.Second

	minConsumeBackoff = time.Second
	maxConsumeBackoff = 30 * time.Second
)

// broker is machinery's transport to RabbitMQ, in place of machinery's own AMQP
// broker. Every task goes through the one QName queue.
type broker struct {
	common.Broker

	url    string
	logger log.Logger

	// publishing is serialized on one long-lived channel
	pubMu   sync.Mutex
	pubConn *amqp.Connection
	pubCh   *amqp.Channel

	// stopMu orders a consumer starting against StopConsuming waiting for them
	stopMu    sync.Mutex
	stop      chan struct{}
	consumers sync.WaitGroup

	// retryUnqueued flags the signatures in flight whose publish failed.
	// machinery retries a task by publishing the signature it was handed, and
	// reports a failure there with the same opaque error as a malformed task.
	retryUnqueued sync.Map
}

func newBroker(cnf *config.Config, logger log.Logger) *broker {
	return &broker{
		Broker: common.NewBroker(cnf),
		url:    cnf.Broker,
		logger: logger,
		stop:   make(chan struct{}),
	}
}

var _ iface.Broker = (*broker)(nil)

// connect opens the publisher connection and declares the task queue.
func (b *broker) connect() error {
	b.pubMu.Lock()
	defer b.pubMu.Unlock()

	_, err := b.publishChannel()

	return err
}

// Publish queues signature, through a delay queue when its ETA is in the future.
func (b *broker) Publish(ctx context.Context, signature *tasks.Signature) error {
	body, err := json.Marshal(signature)
	if err != nil {
		return fmt.Errorf("marshal task: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, publishTimeout)
	defer cancel()

	if err := b.publish(ctx, signature, body); err != nil {
		if unqueued, ok := b.retryUnqueued.Load(signature); ok {
			unqueued.(*atomic.Bool).Store(true)
		}

		return err
	}

	return nil
}

func (b *broker) publish(ctx context.Context, signature *tasks.Signature, body []byte) error {
	b.pubMu.Lock()
	defer b.pubMu.Unlock()

	ch, err := b.publishChannel()
	if err != nil {
		return err
	}

	if err := send(ctx, ch, signature, body); err != nil {
		// a failed channel is not reused: the next publish dials again
		b.closePublisher()
		return err
	}

	return nil
}

func send(ctx context.Context, ch *amqp.Channel, signature *tasks.Signature, body []byte) error {
	exchange, key := exchangeName, QName
	if delay := delayFor(signature.ETA, time.Now()); delay > 0 {
		name, err := declareDelayQueue(ch, delay)
		if err != nil {
			return err
		}
		// the default exchange routes to the queue named by the key
		exchange, key = "", name
	}

	return publishConfirmed(ctx, ch, exchange, key, amqp.Publishing{
		Headers:      amqp.Table(signature.Headers),
		ContentType:  "application/json",
		Body:         body,
		Priority:     signature.Priority,
		DeliveryMode: amqp.Persistent,
	})
}

// publishChannel returns the open publisher channel, dialing if there is none.
// The caller holds pubMu.
func (b *broker) publishChannel() (*amqp.Channel, error) {
	if b.pubCh != nil && !b.pubCh.IsClosed() {
		return b.pubCh, nil
	}

	b.closePublisher()

	conn, ch, err := openChannel(b.url)
	if err != nil {
		return nil, err
	}

	if err := ch.Confirm(false); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("enable publisher confirms: %w", err)
	}

	b.pubConn, b.pubCh = conn, ch

	return ch, nil
}

// closePublisher drops the publisher connection. The caller holds pubMu.
func (b *broker) closePublisher() {
	if b.pubConn != nil {
		_ = b.pubConn.Close()
	}

	b.pubConn, b.pubCh = nil, nil
}

// openChannel dials url and declares the exchange, the task queue and their
// binding, with the arguments machinery declared them with.
func openChannel(url string) (*amqp.Connection, *amqp.Channel, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, nil, fmt.Errorf("dial amqp: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("open channel: %w", err)
	}

	if err := declareTaskQueue(ch); err != nil {
		_ = conn.Close()
		return nil, nil, err
	}

	return conn, ch, nil
}

func declareTaskQueue(ch *amqp.Channel) error {
	if err := ch.ExchangeDeclare(exchangeName, amqp.ExchangeDirect, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare exchange: %w", err)
	}

	if _, err := ch.QueueDeclare(QName, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare queue: %w", err)
	}

	if err := ch.QueueBind(QName, QName, exchangeName, false, nil); err != nil {
		return fmt.Errorf("bind queue: %w", err)
	}

	return nil
}

// delayFor returns how long a task with the given ETA waits in a delay queue,
// zero if it is due. Delays round up to whole seconds so tasks share a queue
// per delay instead of declaring one per millisecond of jitter.
func delayFor(eta *time.Time, now time.Time) time.Duration {
	if eta == nil || !eta.After(now) {
		return 0
	}

	return (eta.Sub(now) + time.Second - 1).Truncate(time.Second)
}

func delayQueueName(delay time.Duration) string {
	return fmt.Sprintf("delay.%d.%s.%s", delay.Milliseconds(), exchangeName, QName)
}

// delayQueueArgs makes a queue hold each message for delay and then dead-letter
// it to the task queue. The queue expires once unused for twice the delay, by
// when every message it accepted has left; declaring it restarts that timer.
func delayQueueArgs(delay time.Duration) amqp.Table {
	return amqp.Table{
		"x-dead-letter-exchange":    exchangeName,
		"x-dead-letter-routing-key": QName,
		"x-message-ttl":             delay.Milliseconds(),
		"x-expires":                 2 * delay.Milliseconds(),
	}
}

func declareDelayQueue(ch *amqp.Channel, delay time.Duration) (string, error) {
	name := delayQueueName(delay)
	if _, err := ch.QueueDeclare(name, true, false, false, false, delayQueueArgs(delay)); err != nil {
		return "", fmt.Errorf("declare delay queue: %w", err)
	}

	return name, nil
}

func publishConfirmed(ctx context.Context, ch *amqp.Channel, exchange, key string, msg amqp.Publishing) error {
	confirmation, err := ch.PublishWithDeferredConfirmWithContext(ctx, exchange, key, false, false, msg)
	if err != nil {
		return fmt.Errorf("publish task: %w", err)
	}

	acked, err := confirmation.WaitContext(ctx)
	if err != nil {
		return fmt.Errorf("await publish confirmation: %w", err)
	}

	if !acked {
		return errors.New("broker rejected the task")
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

	backoff := minConsumeBackoff
	for {
		connected, err := b.consume(consumerTag, concurrency, processor)
		if err == nil {
			return false, nil
		}

		if connected {
			backoff = minConsumeBackoff
		}

		b.logger.Error("Bridge queue consumer failed, reconnecting", "err", err, "in", backoff)

		if !b.wait(backoff) {
			return false, nil
		}

		backoff = min(2*backoff, maxConsumeBackoff)
	}
}

// consume runs one consumer session and reports whether it got as far as
// consuming. It returns a nil error only when the broker is stopped, after the
// tasks in flight have finished.
func (b *broker) consume(consumerTag string, concurrency int, processor iface.TaskProcessor) (bool, error) {
	conn, ch, err := openChannel(b.url)
	if err != nil {
		return false, err
	}
	defer func() { _ = conn.Close() }()

	// Without a prefetch limit RabbitMQ pushes the whole backlog at once, and
	// closes the channel when a delivery then waits past its ack timeout.
	if err := ch.Qos(concurrency, 0, false); err != nil {
		return false, fmt.Errorf("set prefetch: %w", err)
	}

	deliveries, err := ch.Consume(QName, consumerTag, false, false, false, false, nil)
	if err != nil {
		return false, fmt.Errorf("consume: %w", err)
	}

	closed := ch.NotifyClose(make(chan *amqp.Error, 1))

	// tasks in flight finish, and ack, before the connection closes
	var inFlight sync.WaitGroup
	defer inFlight.Wait()

	for {
		select {
		case <-b.stop:
			return true, nil
		case delivery, ok := <-deliveries:
			if !ok {
				return true, consumerEnded(closed)
			}

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

	// machinery would ack it without a word
	if !b.IsTaskRegistered(signature.Name) {
		b.logger.Warn("Bridge queue: dropping task with no registered processor", "taskName", signature.Name, "uuid", signature.UUID)
		b.settle(delivery.Reject(false))
		return
	}

	unqueued := new(atomic.Bool)
	b.retryUnqueued.Store(signature, unqueued)
	err = processor.Process(signature)
	b.retryUnqueued.Delete(signature)

	switch {
	case unqueued.Load():
		b.logger.Error("Bridge queue: retry not queued, returning task to the queue", "taskName", signature.Name, "uuid", signature.UUID, "err", err)
		b.wait(requeuePause)
		b.settle(delivery.Nack(false, true))
	case err != nil:
		b.logger.Error("Bridge queue: dropping task machinery could not run", "taskName", signature.Name, "uuid", signature.UUID, "err", err)
		b.settle(delivery.Reject(false))
	default:
		b.settle(delivery.Ack(false))
	}
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

	b.pubMu.Lock()
	b.closePublisher()
	b.pubMu.Unlock()
}
