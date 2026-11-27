package queue

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"time"

	"cosmossdk.io/log"
	"github.com/RichardKnop/machinery/v1"
	nullbackend "github.com/RichardKnop/machinery/v1/backends/null"
	"github.com/RichardKnop/machinery/v1/config"
	eagerlock "github.com/RichardKnop/machinery/v1/locks/eager"
	"github.com/RichardKnop/machinery/v1/tasks"
	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/metrics"
)

// Connector is used to connect to the queue
type Connector struct {
	logger log.Logger
	Server *machinery.Server
	broker *broker

	mu     sync.Mutex
	worker *machinery.Worker
}

const (
	// QName is machinery task queue
	QName = "machinery_tasks"

	// queuedAtHeader holds the unix milliseconds a task was first queued at.
	// machinery republishes the same signature on retry, so it persists.
	queuedAtHeader = "bridgeQueuedAt"

	// workerConcurrency is how many tasks run at once, and the consumer's prefetch.
	workerConcurrency = 10
)

// NewTaskHeaders returns the headers for a task first queued at queuedAt.
func NewTaskHeaders(queuedAt time.Time) tasks.Headers {
	return tasks.Headers{queuedAtHeader: strconv.FormatInt(queuedAt.UnixMilli(), 10)}
}

// taskQueuedAt reads the time NewTaskHeaders recorded in headers.
func taskQueuedAt(headers tasks.Headers) (time.Time, bool) {
	raw, _ := headers[queuedAtHeader].(string)
	ms, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return time.Time{}, false
	}

	return time.UnixMilli(ms), true
}

// observeTaskDelivery records a delivery of a registered task, retries
// included, and how long ago the task was first queued.
func observeTaskDelivery(signature *tasks.Signature) {
	metrics.BridgeTaskDeliveries.WithLabelValues(signature.Name).Inc()

	if queuedAt, ok := taskQueuedAt(signature.Headers); ok {
		metrics.BridgeTaskAge.WithLabelValues(signature.Name).Observe(time.Since(queuedAt).Seconds())
	}
}

// NewQueueConnector connects to RabbitMQ and declares the task queue.
func NewQueueConnector(dialer string) (*Connector, error) {
	return newConnector(newConfig(dialer))
}

func newConnector(cnf *config.Config) (*Connector, error) {
	if _, err := amqp.ParseURI(cnf.Broker); err != nil {
		// the parse error can embed the url, credentials included
		return nil, errors.New("unsupported amqp url")
	}

	qlogger := helper.Logger.With("module", "bridge/queue")
	redirectMachineryLog(qlogger)

	broker := newBroker(cnf, qlogger)
	ctx, cancel := context.WithTimeout(context.Background(), publishTimeout)
	defer cancel()

	if err := broker.connect(ctx); err != nil {
		return nil, err
	}

	// The bridge never reads task results, so they go to the null backend.
	server := machinery.NewServerWithBrokerBackendLock(cnf, broker, nullbackend.New(), eagerlock.New())

	return &Connector{logger: qlogger, Server: server, broker: broker}, nil
}

func newConfig(dialer string) *config.Config {
	return &config.Config{
		Broker:       dialer,
		DefaultQueue: QName,
		// heimdalld owns SIGTERM; machinery's handler would quit the worker a second time after StopWorker
		NoUnixSignals: true,
	}
}

func newWorker(server *machinery.Server) *machinery.Worker {
	worker := server.NewWorker("invoke-processor", workerConcurrency)
	worker.SetPreTaskHandler(observeTaskDelivery)

	return worker
}

// StartWorker - starts worker to process registered tasks
func (qc *Connector) StartWorker() {
	qc.mu.Lock()
	defer qc.mu.Unlock()

	if qc.worker != nil {
		return
	}

	worker := newWorker(qc.Server)
	errors := make(chan error, 1)
	qc.worker = worker

	qc.logger.Info("Starting machinery worker")

	go qc.watchWorker(worker, errors)
	worker.LaunchAsync(errors)
}

// StopWorker stops the worker and prevents it from consuming more tasks.
func (qc *Connector) StopWorker() {
	qc.mu.Lock()
	worker := qc.worker
	qc.worker = nil
	qc.mu.Unlock()

	if worker != nil {
		qc.logger.Info("Stopping machinery worker")
		worker.Quit()
	}

	// also when no worker was started: the connector opens a connection
	if qc.broker != nil {
		qc.broker.StopConsuming()
	}
}

func (qc *Connector) watchWorker(worker *machinery.Worker, errors <-chan error) {
	err := <-errors

	qc.mu.Lock()
	if qc.worker == worker {
		qc.worker = nil
	}
	qc.mu.Unlock()

	if err != nil {
		qc.logger.Error("Machinery worker stopped", "err", err)
		return
	}

	qc.logger.Info("Machinery worker stopped")
}
