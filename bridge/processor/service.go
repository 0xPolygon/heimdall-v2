package processor

import (
	"sync"

	common "github.com/cometbft/cometbft/libs/service"
	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	servercmtlog "github.com/cosmos/cosmos-sdk/server/log"
	"github.com/spf13/viper"

	"github.com/0xPolygon/heimdall-v2/bridge/broadcaster"
	"github.com/0xPolygon/heimdall-v2/bridge/queue"
	"github.com/0xPolygon/heimdall-v2/helper"
)

const (
	processorServiceStr = "processor-service"
)

// retiredTasks are task names older bridges queued that no processor handles
// anymore. A no-op handler for each drains any left in RabbitMQ after an
// upgrade, which machinery would otherwise redeliver forever.
var retiredTasks = []string{"sendTickAckToHeimdall", "sendUnjailToHeimdall"}

// Service starts and stops all event processors
type Service struct {
	// Base service
	common.BaseService

	// queue connector
	queueConnector *queue.Connector

	processors        []Processor
	registerTasksOnce sync.Once
}

// NewProcessorService returns a new service object for processing queue msg
func NewProcessorService(
	queueConnector *queue.Connector,
	httpClient *rpchttp.HTTP,
	txBroadcaster *broadcaster.TxBroadcaster,
	contractCaller *helper.ContractCaller,
) *Service {
	// creating the processor object
	processorService := &Service{
		queueConnector: queueConnector,
	}

	processorService.BaseService = *common.NewBaseService(
		servercmtlog.CometLoggerWrapper{Logger: helper.Logger}.With("service", "processor"),
		processorServiceStr,
		processorService,
	)

	//
	// Initialize processors
	//

	// initialize checkpoint processor
	checkpointProcessor := NewCheckpointProcessor(&contractCaller.RootChainABI)
	checkpointProcessor.BaseProcessor = *NewBaseProcessor(queueConnector, httpClient, txBroadcaster, contractCaller, "checkpoint")

	// initialize fee processor
	feeProcessor := NewFeeProcessor(&contractCaller.StakingInfoABI)
	feeProcessor.BaseProcessor = *NewBaseProcessor(queueConnector, httpClient, txBroadcaster, contractCaller, "fee")

	// initialize staking processor
	stakingProcessor := NewStakingProcessor(&contractCaller.StakingInfoABI)
	stakingProcessor.BaseProcessor = *NewBaseProcessor(queueConnector, httpClient, txBroadcaster, contractCaller, "staking")

	// initialize clerk processor
	clerkProcessor := NewClerkProcessor(&contractCaller.StateSenderABI)
	clerkProcessor.BaseProcessor = *NewBaseProcessor(queueConnector, httpClient, txBroadcaster, contractCaller, "clerk")

	// initialize span processor
	spanProcessor := &SpanProcessor{}
	spanProcessor.BaseProcessor = *NewBaseProcessor(queueConnector, httpClient, txBroadcaster, contractCaller, "span")

	//
	// Select processors
	//

	// add into the processor list
	startAll := viper.GetBool(helper.AllProcessesFlag)
	onlyServices := viper.GetStringSlice(helper.OnlyProcessesFlag)

	if startAll {
		processorService.processors = append(processorService.processors,
			checkpointProcessor,
			stakingProcessor,
			clerkProcessor,
			feeProcessor,
			spanProcessor,
		)
	} else {
		for _, service := range onlyServices {
			switch service {
			case "checkpoint":
				processorService.processors = append(processorService.processors, checkpointProcessor)
			case "staking":
				processorService.processors = append(processorService.processors, stakingProcessor)
			case "clerk":
				processorService.processors = append(processorService.processors, clerkProcessor)
			case "fee":
				processorService.processors = append(processorService.processors, feeProcessor)
			case "span":
				processorService.processors = append(processorService.processors, spanProcessor)
			}
		}
	}

	if len(processorService.processors) == 0 {
		panic("processorService: no processors selected. Use --all or --only <comma-separated processors>")
	}

	return processorService
}

// RegisterTasks registers all selected processor tasks with machinery.
func (processorService *Service) RegisterTasks() {
	processorService.registerTasksOnce.Do(func() {
		for _, processor := range processorService.processors {
			processor.RegisterTasks()
		}

		for _, taskName := range retiredTasks {
			if err := processorService.queueConnector.Server.RegisterTask(taskName, processorService.dropRetiredTask(taskName)); err != nil {
				processorService.Logger.Error("ProcessorService: error registering retired task", "taskName", taskName, "error", err)
			}
		}
	})
}

// dropRetiredTask returns a handler that acknowledges a retired task without
// doing anything. Retired tasks carry the event name and the log JSON.
func (processorService *Service) dropRetiredTask(taskName string) func(string, string) error {
	return func(eventName, _ string) error {
		processorService.Logger.Info("ProcessorService: dropping a retired task", "taskName", taskName, "eventName", eventName)
		return nil
	}
}

// OnStart starts the new block subscription
func (processorService *Service) OnStart() error {
	if err := processorService.BaseService.OnStart(); err != nil {
		processorService.Logger.Error("ProcessorService | OnStart | OnStart", "Error", err)
	} // Always call the overridden method.

	processorService.RegisterTasks()

	// start processors
	for _, processor := range processorService.processors {
		go func(processor Processor) {
			if err := processor.Start(); err != nil {
				processorService.Logger.Error("ProcessorService | OnStart | processor.Start", "Error", err)
			}
		}(processor)
	}

	return nil
}

// OnStop stops all necessary go routines
func (processorService *Service) OnStop() {
	processorService.BaseService.OnStop() // always call the overridden method.
	// start chain listeners
	for _, processor := range processorService.processors {
		processor.Stop()
	}

	processorService.Logger.Info("ProcessorService: all processors stopped")
}
