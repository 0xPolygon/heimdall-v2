package queue

import (
	"sync"

	"cosmossdk.io/log"
	"github.com/RichardKnop/machinery/v1"
	"github.com/RichardKnop/machinery/v1/config"
	"github.com/RichardKnop/machinery/v1/tasks"
	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/0xPolygon/heimdall-v2/helper"
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

	// legacyRoutingKey is the routing key older bridges stamped on every task.
	legacyRoutingKey = "machinery_task"
)

// NewQueueConnector creates a new queue connector
func NewQueueConnector(dialer string) *Connector {
	// amqp dialer
	c, err := amqp.Dial(dialer)
	defer func(c *amqp.Connection) {
		err := c.Close()
		if err != nil {
			panic(err)
		}
	}(c)
	if err != nil {
		panic(err)
	}

	server, err := machinery.NewServer(newConfig(dialer))
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

// newConfig binds the worker queue with its own name as binding key: on a direct
// exchange machinery publishes to a queue named after the routing key, so any
// other key makes publishers declare and fill a queue nobody consumes.
func newConfig(dialer string) *config.Config {
	return &config.Config{
		Broker:        dialer,
		DefaultQueue:  QName,
		ResultBackend: dialer,
		AMQP: &config.AMQPConfig{
			Exchange:     "machinery_exchange",
			ExchangeType: "direct",
			BindingKey:   QName,
		},
	}
}

// normalizeRoutingKey moves tasks queued by older bridges onto QName, so their
// retries don't redeclare the unconsumed legacy queue.
func normalizeRoutingKey(s *tasks.Signature) {
	if s.RoutingKey == legacyRoutingKey {
		s.RoutingKey = QName
	}
}

func newWorker(server *machinery.Server) *machinery.Worker {
	worker := server.NewWorker("invoke-processor", 10)
	worker.SetPreTaskHandler(normalizeRoutingKey)

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
