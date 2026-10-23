package listener

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/RichardKnop/machinery/v1/tasks"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/metrics"
	checkpointTypes "github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
)

const heimdallLastBlockKey = "heimdall-last-block" // storage key

// HeimdallListener - Listens to and process events from heimdall
type HeimdallListener struct {
	BaseListener
}

// Start starts polling heimdall for new blocks
func (hl *HeimdallListener) Start() error {
	hl.Logger.Info("HeimdallListener: starting")

	// create cancellable context
	headerCtx, cancelHeaderProcess := context.WithCancel(context.Background())
	hl.cancelHeaderProcess = cancelHeaderProcess

	// Heimdall pollInterval = (minimal pollInterval of root chain and bor chain)
	pollInterval := helper.GetConfig().SyncerPollInterval
	if helper.GetConfig().CheckpointPollInterval < helper.GetConfig().SyncerPollInterval {
		pollInterval = helper.GetConfig().CheckpointPollInterval
	}

	hl.Logger.Info("HeimdallListener: start polling for events", "pollInterval", pollInterval)
	go hl.pollEvents(headerCtx, pollInterval)

	return nil
}

// Stop stops the heimdall listener
func (hl *HeimdallListener) Stop() {
	hl.Logger.Info("HeimdallListener: stopping")

	// cancel header processing if any, and clean up reference
	if hl.cancelHeaderProcess != nil {
		hl.cancelHeaderProcess()
		hl.cancelHeaderProcess = nil
	}

	hl.Logger.Info("HeimdallListener: stopped")
}

// pollEvents polls heimdall for new blocks and queues their checkpoint events
func (hl *HeimdallListener) pollEvents(ctx context.Context, pollInterval time.Duration) {
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
				hl.Logger.Error("HeimdallListener: error fetching from and to block, skipping events query", "fromBlock", fromBlock, "toBlock", toBlock, "error", err)
				continue
			}
			if fromBlock > toBlock {
				continue
			}

			hl.Logger.Debug("HeimdallListener: fetching new events between blocks", "fromBlock", fromBlock, "toBlock", toBlock)

			if next := hl.processBlockRange(ctx, fromBlock, toBlock); next > fromBlock {
				if err := hl.storageClient.Put([]byte(heimdallLastBlockKey), []byte(strconv.FormatUint(next-1, 10)), nil); err != nil {
					hl.Logger.Error("HeimdallListener: error persisting last processed block", "lastBlock", next-1, "error", err)
				} else {
					metrics.BridgeListenerProcessedBlock.WithLabelValues(heimdallListenerStr).Set(float64(next - 1))
				}
			}

		case <-ctx.Done():
			hl.Logger.Info("HeimdallListener: polling stopped")
			ticker.Stop()
			return
		}
	}
}

func (hl *HeimdallListener) fetchFromAndToBlock(ctx context.Context) (uint64, uint64, error) {
	nodeStatus, err := helper.GetNodeStatus(ctx, hl.cliCtx)
	if err != nil {
		hl.Logger.Error("HeimdallListener: error while fetching heimdall node status", "error", err)
		return 0, 0, err
	}

	chainId := hl.cliCtx.ChainID
	if chainId == "" {
		// If chainId is not set in cliCtx, fetch it from node status.
		// It should not happen because chainId is set in cliCtx during bridge initialization.
		hl.Logger.Debug("HeimdallListener: chainID is empty in cliCtx")
		if nodeStatus.NodeInfo.Network == "" {
			return 0, 0, errors.New("network is empty in node status; cannot determine initial fromBlock")
		}
		chainId = nodeStatus.NodeInfo.Network
	}

	// fromBlock - get the initial block height from config
	fromBlock := helper.GetInitialBlockHeight(chainId)
	// fromBlock - get last block from storage
	hasLastBlock, _ := hl.storageClient.Has([]byte(heimdallLastBlockKey), nil)
	if hasLastBlock {
		lastBlockBytes, err := hl.storageClient.Get([]byte(heimdallLastBlockKey), nil)
		if err != nil {
			hl.Logger.Info("HeimdallListener: error while fetching last block bytes from storage", "error", err)
			return 0, 0, err
		}

		result, err := strconv.ParseUint(string(lastBlockBytes), 10, 64)
		if err != nil {
			hl.Logger.Info("HeimdallListener: error parsing last block bytes from storage", "error", err)
			return 0, 0, err
		}

		hl.Logger.Debug("HeimdallListener: got last block from bridge storage", "lastBlock", result)
		metrics.BridgeListenerProcessedBlock.WithLabelValues(heimdallListenerStr).Set(float64(result))
		if result >= fromBlock {
			hl.Logger.Debug("HeimdallListener: overriding fromBlock using last processed block from storage", "oldFromBlock", fromBlock, "lastProcessedBlock", result, "newFromBlock", result+1)
			fromBlock = result + 1
		}
	}

	// Clamp fromBlock to the node's earliest available block (this handles pruned snapshots)
	earliestBlock := uint64(nodeStatus.SyncInfo.EarliestBlockHeight)
	if earliestBlock > 0 && fromBlock < earliestBlock {
		hl.Logger.Info("HeimdallListener: fromBlock is before the node's earliest available block, skipping ahead",
			"fromBlock", fromBlock, "earliestBlock", earliestBlock)
		fromBlock = earliestBlock
	}

	// toBlock - get the latest block height from heimdall node
	toBlock := uint64(nodeStatus.SyncInfo.LatestBlockHeight)
	metrics.BridgeListenerHeadBlock.WithLabelValues(heimdallListenerStr).Set(float64(toBlock))

	return fromBlock, toBlock, nil
}

// processBlockRange queues the checkpoint events of each block in
// [fromBlock, toBlock] in order and returns the first height it didn't
// finish. It stops at the first block whose events can't be fetched or
// queued, so the next poll retries that block instead of skipping it.
func (hl *HeimdallListener) processBlockRange(ctx context.Context, fromBlock, toBlock uint64) uint64 {
	height := fromBlock
	for ; height <= toBlock && ctx.Err() == nil; height++ {
		if err := hl.processBlock(ctx, height); err != nil {
			hl.Logger.Error("HeimdallListener: error processing block, retrying from it next poll", "height", height, "error", err)
			break
		}
	}

	return height
}

func (hl *HeimdallListener) processBlock(ctx context.Context, height uint64) error {
	events, err := helper.GetBeginBlockEvents(ctx, hl.httpClient, int64(height))
	if err != nil {
		metrics.BridgeListenerRangeRetries.WithLabelValues(heimdallListenerStr, "query_failed").Inc()
		return err
	}

	for _, event := range events {
		if err := hl.ProcessBlockEvent(sdk.StringifyEvent(event), int64(height)); err != nil {
			metrics.BridgeListenerRangeRetries.WithLabelValues(heimdallListenerStr, "not_queued").Inc()
			return err
		}
	}

	return nil
}

// ProcessBlockEvent queues a heimdall block event for its processor. It
// returns an error only when the task couldn't be queued.
func (hl *HeimdallListener) ProcessBlockEvent(event sdk.StringEvent, blockHeight int64) error {
	hl.Logger.Debug("HeimdallListener: received block event from Heimdall", "eventType", event.Type)

	eventBytes, err := json.Marshal(event)
	if err != nil {
		hl.Logger.Error("HeimdallListener: error while marshalling block event", "eventType", event.Type, "error", err)
		return nil
	}

	switch event.Type {
	case checkpointTypes.EventTypeCheckpoint:
		hl.Logger.Info("HeimdallListener: sending task for checkpoint event", "blockHeight", blockHeight)
		return hl.sendTask("sendCheckpointToRootchain", blockTaskRetryCount, 0,
			stringArg(string(eventBytes)), tasks.Arg{Type: "int64", Value: blockHeight})
	default:
		hl.Logger.Debug("HeimdallListener: block event type mismatch", "eventType", event.Type)
		return nil
	}
}
