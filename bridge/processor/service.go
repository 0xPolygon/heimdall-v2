package processor

import (
	"github.com/0xPolygon/heimdall-v2/bridge/broadcaster"
	"github.com/0xPolygon/heimdall-v2/bridge/queue"
	"github.com/0xPolygon/heimdall-v2/helper"
	common "github.com/cometbft/cometbft/libs/service"
	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/spf13/viper"
)

const (
	processorServiceStr = "processor-service"
)

// ProcessorService starts and stops all event processors
// revive:disable-next-line exported
type ProcessorService struct {
	// Base service
	common.BaseService

	// queue connector
	queueConnector *queue.QueueConnector

	processors []Processor
}

// NewProcessorService returns new service object for processing queue msg
func NewProcessorService(
	cdc codec.Codec,
	queueConnector *queue.QueueConnector,
	httpClient *rpchttp.HTTP,
	txBroadcaster *broadcaster.TxBroadcaster,
) *ProcessorService {
	// creating processor object
	processorService := &ProcessorService{
		queueConnector: queueConnector,
	}

	contractCaller, err := helper.NewContractCaller()
	if err != nil {
		panic(err)
	}

	processorService.BaseService = *common.NewBaseService(nil, processorServiceStr, processorService)

	//
	// Initialize processors
	//

	// initialize checkpoint processor
	checkpointProcessor := NewCheckpointProcessor(&contractCaller.RootChainABI)
	checkpointProcessor.BaseProcessor = *NewBaseProcessor(cdc, queueConnector, httpClient, txBroadcaster, "checkpoint", checkpointProcessor)

	// initialize checkpoint processor
	milestoneProcessor := &MilestoneProcessor{}
	milestoneProcessor.BaseProcessor = *NewBaseProcessor(cdc, queueConnector, httpClient, txBroadcaster, "milestone", milestoneProcessor)

	// initialize fee processor
	feeProcessor := NewFeeProcessor(&contractCaller.StakingInfoABI)
	feeProcessor.BaseProcessor = *NewBaseProcessor(cdc, queueConnector, httpClient, txBroadcaster, "fee", feeProcessor)

	// initialize staking processor
	stakingProcessor := NewStakingProcessor(&contractCaller.StakingInfoABI)
	stakingProcessor.BaseProcessor = *NewBaseProcessor(cdc, queueConnector, httpClient, txBroadcaster, "staking", stakingProcessor)

	// initialize clerk processor
	clerkProcessor := NewClerkProcessor(&contractCaller.StateSenderABI)
	clerkProcessor.BaseProcessor = *NewBaseProcessor(cdc, queueConnector, httpClient, txBroadcaster, "clerk", clerkProcessor)

	// initialize span processor
	spanProcessor := &SpanProcessor{}
	spanProcessor.BaseProcessor = *NewBaseProcessor(cdc, queueConnector, httpClient, txBroadcaster, "span", spanProcessor)

	// HV2 - not adding slashing
	/*
		// initialize slashing processor
		slashingProcessor := NewSlashingProcessor(&contractCaller.StakingInfoABI)
		slashingProcessor.BaseProcessor = *NewBaseProcessor(cdc, queueConnector, httpClient, txBroadcaster, "slashing", slashingProcessor)
	*/

	//
	// Select processors
	//

	// add into processor list
	startAll := viper.GetBool(helper.AllProcessesFlag)
	onlyServices := viper.GetStringSlice(helper.OnlyProcessesFlag)

	if startAll {
		processorService.processors = append(processorService.processors,
			checkpointProcessor,
			milestoneProcessor,
			stakingProcessor,
			clerkProcessor,
			feeProcessor,
			spanProcessor,
			// HV2 - not adding slashing
			// slashingProcessor,
		)
	} else {
		for _, service := range onlyServices {
			switch service {
			case "checkpoint":
				processorService.processors = append(processorService.processors, checkpointProcessor)
			case "milestone":
				processorService.processors = append(processorService.processors, milestoneProcessor)
			case "staking":
				processorService.processors = append(processorService.processors, stakingProcessor)
			case "clerk":
				processorService.processors = append(processorService.processors, clerkProcessor)
			case "fee":
				processorService.processors = append(processorService.processors, feeProcessor)
			case "span":
				processorService.processors = append(processorService.processors, spanProcessor)
				// HV2 - not adding slashing
				/*
					case "slashing":
						processorService.processors = append(processorService.processors, slashingProcessor)
				*/
			}
		}
	}

	if len(processorService.processors) == 0 {
		panic("No processors selected. Use --all or --only <comma-separated processors>")
	}

	return processorService
}

// OnStart starts new block subscription
func (processorService *ProcessorService) OnStart() error {
	if err := processorService.BaseService.OnStart(); err != nil {
		processorService.Logger.Error("OnStart | OnStart", "Error", err)
	} // Always call the overridden method.

	// start processors
	for _, processor := range processorService.processors {
		processor.RegisterTasks()

		go func(processor Processor) {
			if err := processor.Start(); err != nil {
				processorService.Logger.Error("OnStart | processor.Start", "Error", err)
			}
		}(processor)
	}

	processorService.Logger.Info("all processors Started")

	return nil
}

// OnStop stops all necessary go routines
func (processorService *ProcessorService) OnStop() {
	processorService.BaseService.OnStop() // Always call the overridden method.
	// start chain listeners
	for _, processor := range processorService.processors {
		processor.Stop()
	}

	processorService.Logger.Info("all processors stopped")
}
