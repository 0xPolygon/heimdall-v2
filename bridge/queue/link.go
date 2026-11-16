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
	dialTimeout  = 10 * time.Second
	closeTimeout = 5 * time.Second
)

// link is an AMQP connection, its channel and the socket under them.
type link struct {
	conn *amqp.Connection
	ch   *amqp.Channel
	sock net.Conn
}

// openLink dials url and declares the exchange, the task queue and their
// binding, all within ctx.
func openLink(ctx context.Context, url, queue string) (*link, error) {
	l, err := dial(ctx, url)
	if err != nil {
		return nil, err
	}

	stop := l.abortOn(ctx)
	defer stop()

	if l.ch, err = l.conn.Channel(); err != nil {
		l.close()
		return nil, fmt.Errorf("open channel: %w", err)
	}

	if err := declareTaskQueue(l.ch, queue); err != nil {
		l.close()
		return nil, err
	}

	return l, nil
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

	return &link{conn: conn, sock: sock}, nil
}

// abortOn closes the socket when ctx ends, until the returned stop is called.
// amqp091's methods take no deadline: one waiting on RabbitMQ returns only
// once the socket is dead.
func (l *link) abortOn(ctx context.Context) (stop func() bool) {
	return context.AfterFunc(ctx, func() { _ = l.sock.Close() })
}

// close ends the connection without waiting on RabbitMQ past closeTimeout.
func (l *link) close() { l.closeWithin(closeTimeout) }

func (l *link) closeWithin(timeout time.Duration) {
	// amqp091's own close deadline is moved on by every heartbeat
	kill := time.AfterFunc(timeout, func() { _ = l.sock.Close() })
	defer kill.Stop()

	_ = l.conn.Close()
	_ = l.sock.Close()
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
