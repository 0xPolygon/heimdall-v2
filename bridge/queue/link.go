package queue

import (
	"context"
	"fmt"
	"net"
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
	deadline := time.Now().Add(dialTimeout)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}

	l := new(link)

	conn, err := amqp.DialConfig(url, amqp.Config{
		Locale: "en_US",
		Dial: func(network, addr string) (net.Conn, error) {
			sock, err := (&net.Dialer{Deadline: deadline}).DialContext(ctx, network, addr)
			if err != nil {
				return nil, err
			}

			// amqp091 clears the deadline once the handshake is done
			if err := sock.SetDeadline(deadline); err != nil {
				_ = sock.Close()
				return nil, err
			}

			l.sock = sock

			return sock, nil
		},
	})
	if err != nil {
		return nil, fmt.Errorf("dial amqp: %w", err)
	}

	l.conn = conn

	return l, nil
}

// abortOn closes the socket when ctx ends, until the returned stop is called.
// amqp091's methods take no deadline: one waiting on RabbitMQ returns only
// once the socket is dead.
func (l *link) abortOn(ctx context.Context) (stop func() bool) {
	return context.AfterFunc(ctx, func() { _ = l.sock.Close() })
}

// close ends the connection without waiting on RabbitMQ past closeTimeout.
func (l *link) close() {
	_ = l.conn.CloseDeadline(time.Now().Add(closeTimeout))
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
