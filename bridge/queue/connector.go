package queue

import (
	"sync"

	"cosmossdk.io/log"
	"github.com/RichardKnop/machinery/v1"
	"github.com/RichardKnop/machinery/v1/config"
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

	cnf := &config.Config{
		Broker:        dialer,
		DefaultQueue:  QName,
		ResultBackend: dialer,
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

	connector := Connector{
		logger: helper.Logger.With("module", "bridge/queue"),
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
