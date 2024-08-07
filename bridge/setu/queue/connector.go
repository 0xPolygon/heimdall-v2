package queue

import (
	"github.com/cometbft/cometbft/libs/log"

	"github.com/RichardKnop/machinery/v1"
	"github.com/RichardKnop/machinery/v1/config"
	"github.com/streadway/amqp"
)

// QueueConnector is used to connect to the queue
type QueueConnector struct {
	logger log.Logger
	Server *machinery.Server
}

const (
	// machinery task queue
	QueueName = "machinery_tasks"
)

// NewQueueConnector creates a new queue connector
func NewQueueConnector(dialer string) *QueueConnector {
	// amqp dialer
	_, err := amqp.Dial(dialer)
	if err != nil {
		panic(err)
	}

	var cnf = &config.Config{
		Broker:        dialer,
		DefaultQueue:  QueueName,
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

	// queue connector
	connector := QueueConnector{
		logger: log.NewNopLogger().With("module", "QueueConnector"),
		Server: server,
	}

	// connector
	return &connector
}

// StartWorker - starts worker to process registered tasks
func (qc *QueueConnector) StartWorker() {
	worker := qc.Server.NewWorker("invoke-processor", 10)

	qc.logger.Info("Starting machinery worker")

	errors := make(chan error)

	worker.LaunchAsync(errors)
}
