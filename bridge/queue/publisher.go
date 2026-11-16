package queue

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/RichardKnop/machinery/v1/tasks"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	exchangeName = "machinery_exchange"

	// publishTimeout bounds one publish: the wait for the publisher, the dial
	// if it has no connection, and the broker's confirmation.
	publishTimeout = 15 * time.Second

	dialTimeout = 10 * time.Second

	// delayQueueSlack is how much longer than its messages a delay queue
	// lives, so a publish stalled after the queue was declared still lands in it.
	delayQueueSlack = time.Minute
)

// Flags the AMQP methods take positionally. The declarations match what
// machinery declared the exchange and task queue with: RabbitMQ rejects a
// redeclaration that differs.
const (
	durable    = true
	autoDelete = false
	exclusive  = false
	internal   = false
	noWait     = false

	// mandatory has RabbitMQ return a task it could not route, as when the
	// queue was deleted, instead of confirming and discarding it.
	mandatory = true
	immediate = false
)

// publisher publishes tasks on one long-lived channel in confirm mode.
type publisher struct {
	url   string
	queue string

	// lock serializes publishing. It is a channel so that waiting for it ends
	// with the caller's context.
	lock    chan struct{}
	conn    *amqp.Connection
	ch      *amqp.Channel
	returns <-chan amqp.Return
}

func newPublisher(url, queue string) *publisher {
	return &publisher{url: url, queue: queue, lock: make(chan struct{}, 1)}
}

func (p *publisher) acquire(ctx context.Context) error {
	select {
	case p.lock <- struct{}{}:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("wait for publisher: %w", ctx.Err())
	}
}

func (p *publisher) release() { <-p.lock }

// connect opens the connection and declares the task queue.
func (p *publisher) connect(ctx context.Context) error {
	if err := p.acquire(ctx); err != nil {
		return err
	}
	defer p.release()

	_, err := p.channel(ctx)

	return err
}

func (p *publisher) publish(ctx context.Context, signature *tasks.Signature, body []byte) error {
	if err := p.acquire(ctx); err != nil {
		return err
	}
	defer p.release()

	ch, err := p.channel(ctx)
	if err != nil {
		return err
	}

	if err := p.send(ctx, ch, signature, body); err != nil {
		// a failed channel is not reused: the next publish dials again and
		// redeclares the task queue
		p.drop()
		return err
	}

	return nil
}

func (p *publisher) send(ctx context.Context, ch *amqp.Channel, signature *tasks.Signature, body []byte) error {
	exchange, key := exchangeName, p.queue
	if delay := delayFor(signature.ETA, time.Now()); delay > 0 {
		name, err := declareDelayQueue(ch, p.queue, delay)
		if err != nil {
			return err
		}
		// the default exchange routes to the queue named by the key
		exchange, key = "", name
	}

	return publishConfirmed(ctx, ch, p.returns, exchange, key, amqp.Publishing{
		Headers:      amqp.Table(signature.Headers),
		ContentType:  "application/json",
		Body:         body,
		Priority:     signature.Priority,
		DeliveryMode: amqp.Persistent,
	})
}

// channel returns the open channel, dialing if there is none. The caller holds
// the lock.
func (p *publisher) channel(ctx context.Context) (*amqp.Channel, error) {
	if p.ch != nil && !p.ch.IsClosed() {
		return p.ch, nil
	}

	p.drop()

	conn, ch, err := openChannel(ctx, p.url, p.queue)
	if err != nil {
		return nil, err
	}

	if err := ch.Confirm(noWait); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("enable publisher confirms: %w", err)
	}

	p.conn, p.ch, p.returns = conn, ch, ch.NotifyReturn(make(chan amqp.Return, 1))

	return ch, nil
}

// drop closes the connection. The caller holds the lock.
func (p *publisher) drop() {
	if p.conn != nil {
		_ = p.conn.Close()
	}

	p.conn, p.ch, p.returns = nil, nil, nil
}

func (p *publisher) close() {
	p.lock <- struct{}{}
	defer p.release()

	p.drop()
}

// dial connects within dialTimeout, or ctx's deadline if that is sooner.
func dial(ctx context.Context, url string) (*amqp.Connection, error) {
	timeout := dialTimeout
	if deadline, ok := ctx.Deadline(); ok {
		timeout = min(timeout, time.Until(deadline))
	}

	if timeout <= 0 {
		return nil, fmt.Errorf("dial amqp: %w", context.DeadlineExceeded)
	}

	conn, err := amqp.DialConfig(url, amqp.Config{Locale: "en_US", Dial: amqp.DefaultDial(timeout)})
	if err != nil {
		return nil, fmt.Errorf("dial amqp: %w", err)
	}

	return conn, nil
}

// openChannel dials url and declares the exchange, the task queue and their
// binding.
func openChannel(ctx context.Context, url, queue string) (*amqp.Connection, *amqp.Channel, error) {
	conn, err := dial(ctx, url)
	if err != nil {
		return nil, nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("open channel: %w", err)
	}

	if err := declareTaskQueue(ch, queue); err != nil {
		_ = conn.Close()
		return nil, nil, err
	}

	return conn, ch, nil
}

func declareTaskQueue(ch *amqp.Channel, queue string) error {
	if err := ch.ExchangeDeclare(exchangeName, amqp.ExchangeDirect, durable, autoDelete, internal, noWait, nil); err != nil {
		return fmt.Errorf("declare exchange: %w", err)
	}

	if _, err := ch.QueueDeclare(queue, durable, autoDelete, exclusive, noWait, nil); err != nil {
		return fmt.Errorf("declare queue: %w", err)
	}

	if err := ch.QueueBind(queue, queue, exchangeName, noWait, nil); err != nil {
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

// delayQueueName differs from machinery's delay queue names: the arguments do.
func delayQueueName(queue string, delay time.Duration) string {
	return fmt.Sprintf("%s.delay.%d", queue, delay.Milliseconds())
}

// delayQueueArgs makes a queue hold each message for delay and then dead-letter
// it to the task queue. The queue expires once unused for long enough that
// every message it accepted has left; declaring it restarts that timer.
func delayQueueArgs(queue string, delay time.Duration) amqp.Table {
	return amqp.Table{
		"x-dead-letter-exchange":    exchangeName,
		"x-dead-letter-routing-key": queue,
		"x-message-ttl":             delay.Milliseconds(),
		"x-expires":                 (2*delay + delayQueueSlack).Milliseconds(),
	}
}

func declareDelayQueue(ch *amqp.Channel, queue string, delay time.Duration) (string, error) {
	name := delayQueueName(queue, delay)
	if _, err := ch.QueueDeclare(name, durable, autoDelete, exclusive, noWait, delayQueueArgs(queue, delay)); err != nil {
		return "", fmt.Errorf("declare delay queue: %w", err)
	}

	return name, nil
}

// publishConfirmed publishes msg and waits for RabbitMQ to take responsibility
// for it. RabbitMQ sends the return of an unroutable message before its
// confirmation.
func publishConfirmed(ctx context.Context, ch *amqp.Channel, returns <-chan amqp.Return, exchange, key string, msg amqp.Publishing) error {
	confirmation, err := ch.PublishWithDeferredConfirmWithContext(ctx, exchange, key, mandatory, immediate, msg)
	if err != nil {
		return fmt.Errorf("publish task: %w", err)
	}

	acked, err := confirmation.WaitContext(ctx)
	if err != nil {
		return fmt.Errorf("await publish confirmation: %w", err)
	}

	select {
	case returned, ok := <-returns:
		if ok {
			return fmt.Errorf("task not routed: %s", returned.ReplyText)
		}
	default:
	}

	if !acked {
		return errors.New("broker rejected the task")
	}

	return nil
}
