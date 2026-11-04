package queue

import (
	"strconv"
	"sync"
	"time"

	"cosmossdk.io/log"
	"github.com/RichardKnop/machinery/v1"
	"github.com/RichardKnop/machinery/v1/config"
	"github.com/RichardKnop/machinery/v1/tasks"
	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/metrics"
)

// Connector is used to connect to the queue
type Connector struct {
	logger log.Logger
	Server *machinery.Server

	mu     sync.Mutex
	worker *machinery.Worker
}

const (
	// QName is machinery task queue
	QName = "machinery_tasks"

	// queuedAtHeader holds the unix milliseconds a task was first queued at.
	// machinery republishes the same signature on retry, so it persists.
	queuedAtHeader = "bridgeQueuedAt"
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

// NewQueueConnector creates a new queue connector
func NewQueueConnector(dialer string) *Connector {
	// fail fast if the broker is unreachable
	c, err := amqp.Dial(dialer)
	if err != nil {
		panic(err)
	}
	if err := c.Close(); err != nil {
		panic(err)
	}

	cnf := &config.Config{
		Broker:       dialer,
		DefaultQueue: QName,
		// The bridge never reads task results, and an AMQP result backend opens a
		// connection and declares a queue for every task state change.
		ResultBackend: "null://",
		AMQP: &config.AMQPConfig{
			Exchange:     "machinery_exchange",
			ExchangeType: "direct",
			BindingKey:   "machinery_task",
		},
	}

	server, err := machinery.NewServer(cnf)
	if err != nil {
		panic(err)
	}

	qlogger := helper.Logger.With("module", "bridge/queue")
	redirectMachineryLog(qlogger)

	connector := Connector{
		logger: qlogger,
		Server: server,
	}

	return &connector
}

// StartWorker - starts worker to process registered tasks
func (qc *Connector) StartWorker() {
	qc.mu.Lock()
	defer qc.mu.Unlock()

	if qc.worker != nil {
		return
	}

	worker := qc.Server.NewWorker("invoke-processor", 10)
	worker.SetPreTaskHandler(observeTaskDelivery)
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

	if worker == nil {
		return
	}

	qc.logger.Info("Stopping machinery worker")
	worker.Quit()
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
