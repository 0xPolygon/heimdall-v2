package listener

import (
	"context"
	"encoding/json"
	"math/big"
	"strconv"
	"time"

	"github.com/RichardKnop/machinery/v1/tasks"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/0xPolygon/heimdall-v2/helper"
	checkpointTypes "github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
)

const (
	heimdallLastBlockKey = "heimdall-last-block" // storage key
)

// HeimdallListener - Listens to and process events from heimdall
type HeimdallListener struct {
	BaseListener
}

// Start starts new block subscription
func (hl *HeimdallListener) Start() error {
	hl.Logger.Info("Starting heimdall listener")

	// create cancellable context
	headerCtx, cancelHeaderProcess := context.WithCancel(context.Background())
	hl.cancelHeaderProcess = cancelHeaderProcess

	// Heimdall pollInterval = (minimal pollInterval of root chain and bor chain)
	pollInterval := helper.GetConfig().SyncerPollInterval
	if helper.GetConfig().CheckpointPollInterval < helper.GetConfig().SyncerPollInterval {
		pollInterval = helper.GetConfig().CheckpointPollInterval
	}

	hl.Logger.Info("Start polling for events", "pollInterval", pollInterval)
	hl.StartPolling(headerCtx, pollInterval, nil)

	return nil
}

// ProcessHeader -
func (hl *HeimdallListener) ProcessHeader(_ *blockHeader) {
}

// StartPolling - starts polling for heimdall events
func (hl *HeimdallListener) StartPolling(ctx context.Context, pollInterval time.Duration, _ *big.Int) {
	// How often to fire the passed in function in second
	interval := pollInterval

	// Set up the ticket and the channel to signal
	// the ending of the interval
	ticker := time.NewTicker(interval)

	// start listening
	for {
		select {
		case <-ticker.C:
			fromBlock, toBlock, err := hl.fetchFromAndToBlock(ctx)
			if err != nil {
				hl.Logger.Error("Error fetching from and toBlock, skipping events query", "fromBlock", fromBlock, "toBlock", toBlock, "error", err)
			} else if fromBlock < toBlock {

				hl.Logger.Info("Fetching new events between", "fromBlock", fromBlock, "toBlock", toBlock)

				// Querying and processing Begin events
				for i := fromBlock; i <= toBlock; i++ {
					events, err := helper.GetBeginBlockEvents(ctx, hl.httpClient, int64(i))
					if err != nil {
						hl.Logger.Error("Error fetching begin block events", "error", err)
					}
					for _, event := range events {
						hl.ProcessBlockEvent(sdk.StringifyEvent(event), int64(i))
					}
				}

				// set last block to storage
				if err := hl.storageClient.Put([]byte(heimdallLastBlockKey), []byte(strconv.FormatUint(toBlock, 10)), nil); err != nil {
					hl.Logger.Error("hl.storageClient.Put", "Error", err)
				}
			}

		case <-ctx.Done():
			hl.Logger.Info("Polling stopped")
			ticker.Stop()

			return
		}
	}
}

func (hl *HeimdallListener) fetchFromAndToBlock(ctx context.Context) (uint64, uint64, error) {
	// toBlock - get latest blockheight from heimdall node
	fromBlock := uint64(0)
	toBlock := uint64(0)

	nodeStatus, err := helper.GetNodeStatus(hl.cliCtx, ctx)
	if err != nil {
		hl.Logger.Error("Error while fetching heimdall node status", "error", err)
		return fromBlock, toBlock, err
	}

	toBlock = uint64(nodeStatus.SyncInfo.LatestBlockHeight)

	// fromBlock - get last block from storage
	hasLastBlock, _ := hl.storageClient.Has([]byte(heimdallLastBlockKey), nil)
	if hasLastBlock {
		lastBlockBytes, err := hl.storageClient.Get([]byte(heimdallLastBlockKey), nil)
		if err != nil {
			hl.Logger.Info("Error while fetching last block bytes from storage", "error", err)
			return fromBlock, toBlock, err
		}

		if result, err := strconv.ParseUint(string(lastBlockBytes), 10, 64); err == nil {
			hl.Logger.Debug("Got last block from bridge storage", "lastBlock", result)
			fromBlock = result + 1
		} else {
			hl.Logger.Info("Error parsing last block bytes from storage", "error", err)
			toBlock = 0

			return fromBlock, toBlock, err
		}
	}

	return fromBlock, toBlock, err
}

// ProcessBlockEvent - process Blockevents (BeginBlock, EndBlock events) from heimdall.
func (hl *HeimdallListener) ProcessBlockEvent(event sdk.StringEvent, blockHeight int64) {
	hl.Logger.Info("Received block event from Heimdall", "eventType", event.Type)

	eventBytes, err := json.Marshal(event)
	if err != nil {
		hl.Logger.Error("Error while parsing block event", "eventType", event.Type, "error", err)
		return
	}

	switch event.Type {
	case checkpointTypes.EventTypeCheckpoint:
		hl.sendBlockTask("sendCheckpointToRootchain", eventBytes, blockHeight)
	default:
		hl.Logger.Debug("BlockEvent Type mismatch", "eventType", event.Type)
	}
}

func (hl *HeimdallListener) sendBlockTask(taskName string, eventBytes []byte, blockHeight int64) {
	// create machinery task
	signature := &tasks.Signature{
		Name: taskName,
		Args: []tasks.Arg{
			{
				Type:  "string",
				Value: string(eventBytes),
			},
			{
				Type:  "int64",
				Value: blockHeight,
			},
		},
	}
	signature.RetryCount = 3

	hl.Logger.Info("Sending block level task", "taskName", taskName, "currentTime", time.Now(), "blockHeight", blockHeight)

	// send task
	_, err := hl.queueConnector.Server.SendTask(signature)
	if err != nil {
		hl.Logger.Error("Error sending block level task", "taskName", taskName, "blockHeight", blockHeight, "error", err)
	}
}
