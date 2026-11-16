package queue

import (
	"context"
	"fmt"
	"net"
	"sync/atomic"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	exchangeName = "machinery_exchange"

	dialTimeout = 10 * time.Second

	// linkTimeout bounds a close, ack, nack or reject on a link.
	linkTimeout = 10 * time.Second
)

// Flags the AMQP declarations take positionally. They match what machinery
// declared the exchange and task queue with: RabbitMQ rejects a redeclaration
// that differs.
const (
	durable    = true
	autoDelete = false
	exclusive  = false
	internal   = false
	noWait     = false
)

// link is an AMQP connection, its channel and the socket under them.
type link struct {
	conn *amqp.Connection
	ch   *amqp.Channel
	sock net.Conn

	timeout time.Duration

	// returns carries the publishes RabbitMQ could not route, on a link set
	// up to publish
	returns <-chan amqp.Return
}

// openLink dials url, declares the exchange, the task queue and their binding,
// and runs setup, all within ctx.
func openLink(ctx context.Context, url, queue string, setup func(*link) error) (*link, error) {
	l, err := dial(ctx, url)
	if err != nil {
		return nil, err
	}

	stop := l.abortOn(ctx)
	defer stop()

	if err := l.prepare(queue, setup); err != nil {
		l.close()
		return nil, err
	}

	return l, nil
}

func (l *link) prepare(queue string, setup func(*link) error) (err error) {
	if l.ch, err = l.conn.Channel(); err != nil {
		return fmt.Errorf("open channel: %w", err)
	}

	if err := declareTaskQueue(l.ch, queue); err != nil {
		return err
	}

	return setup(l)
}

// dial connects and completes the AMQP handshake by one deadline: dialTimeout
// from now, or ctx's deadline if that is sooner.
func dial(ctx context.Context, url string) (*link, error) {
	ctx, cancel := context.WithTimeout(ctx, dialTimeout)
	defer cancel()

	// A socket deadline does not hold: amqp091 moves it on every frame once
	// heartbeats start, which is before the handshake ends. Closing the socket
	// when ctx ends does.
	var dialed atomic.Pointer[net.TCPConn]
	stop := context.AfterFunc(ctx, func() {
		if sock := dialed.Load(); sock != nil {
			_ = sock.Close()
		}
	})

	conn, err := amqp.DialConfig(url, amqp.Config{
		Locale: "en_US",
		Dial: func(network, addr string) (net.Conn, error) {
			sock, err := new(net.Dialer).DialContext(ctx, network, addr)
			if err != nil {
				return nil, err
			}

			dialed.Store(sock.(*net.TCPConn))

			// ctx may have ended before the socket could be closed for it
			if err := ctx.Err(); err != nil {
				return nil, err
			}

			return sock, nil
		},
	})

	sock := dialed.Load()
	if err == nil && !stop() {
		_ = conn.Close()
		err = ctx.Err()
	}

	if err != nil {
		// amqp091 does not close the socket on every failure after it dialed
		if sock != nil {
			_ = sock.Close()
		}

		return nil, fmt.Errorf("dial amqp: %w", err)
	}

	return &link{conn: conn, sock: sock, timeout: linkTimeout}, nil
}

// abortOn closes the socket when ctx ends, until the returned stop is called.
// amqp091's methods take no deadline, and it moves the socket's own on every
// heartbeat: one waiting on RabbitMQ returns only once the socket is dead.
func (l *link) abortOn(ctx context.Context) (stop func() bool) {
	return context.AfterFunc(ctx, func() { _ = l.sock.Close() })
}

// bounded runs f, closing the socket if f outlasts the link's timeout. It
// takes no context: a link is closed, and its deliveries settled, after the
// context of whatever opened it has ended.
func (l *link) bounded(f func() error) error {
	kill := time.AfterFunc(l.timeout, func() { _ = l.sock.Close() })
	defer kill.Stop()

	return f()
}

func (l *link) close() {
	_ = l.bounded(l.conn.Close)
	_ = l.sock.Close()
}

// settler bounds the ack, nack or reject of a delivery. A write RabbitMQ never
// reads would otherwise hold its task in flight, and with it shutdown.
type settler struct {
	amqp.Acknowledger
	link *link
}

func (s settler) Ack(tag uint64, multiple bool) error {
	return s.link.bounded(func() error { return s.Acknowledger.Ack(tag, multiple) })
}

func (s settler) Nack(tag uint64, multiple, requeue bool) error {
	return s.link.bounded(func() error { return s.Acknowledger.Nack(tag, multiple, requeue) })
}

func (s settler) Reject(tag uint64, requeue bool) error {
	return s.link.bounded(func() error { return s.Acknowledger.Reject(tag, requeue) })
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
