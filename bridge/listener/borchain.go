package listener

import (
	"context"

	"github.com/0xPolygon/heimdall-v2/helper"
)

// BorChainListener - Listens to and processes headerBlocks from the bor chain
type BorChainListener struct {
	BaseListener
}

// Start starts polling for new Bor headers
func (ml *BorChainListener) Start() error {
	ml.Logger.Info("BorChainListener: starting")

	// create cancellable context
	ctx, cancelSubscription := context.WithCancel(context.Background())
	ml.cancelSubscription = cancelSubscription

	// create cancellable context
	headerCtx, cancelHeaderProcess := context.WithCancel(context.Background())
	ml.cancelHeaderProcess = cancelHeaderProcess

	// start the header process
	go ml.StartHeaderProcess(headerCtx)

	// start go routine to poll for the new header using the client object
	ml.Logger.Info("BorChainListener: start polling for header blocks", "pollInterval", helper.GetConfig().CheckpointPollInterval)

	// start polling for the latest block in the bor child chain (replace with finalized block once we have it implemented)
	go ml.StartPolling(ctx, helper.GetConfig().CheckpointPollInterval, nil)

	return nil
}

// ProcessHeader - process header block from the bor chain
func (ml *BorChainListener) ProcessHeader(newHeader *blockHeader) {
	ml.Logger.Debug("BorChainListener: new header block detected", "blockNumber", newHeader.header.Number)

	// Marshall header block and publish to queue
	headerBytes, err := newHeader.header.MarshalJSON()
	if err != nil {
		ml.Logger.Error("BorChainListener: error marshalling header block", "error", err)
		return
	}

	// A failed send needs no retry: the next poll queues the then-latest header.
	_ = ml.sendTask("sendCheckpointToHeimdall", blockTaskRetryCount, 0, stringArg(string(headerBytes)))
}
