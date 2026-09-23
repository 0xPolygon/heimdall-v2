package listener

import (
	"math/big"
	"strconv"
)

// rootChainBlockRangeToProcess computes the [from, to] block range this
// header makes newly available, capped by finality/confirmations and by the
// last block already persisted to storage. ok is false when there's nothing
// new to process, or the range/storage state couldn't be determined.
func (rl *RootChainListener) rootChainBlockRangeToProcess(rootChainContext *RootChainListenerContext, newHeader *blockHeader) (from, to *big.Int, ok bool) {
	to, ok = rl.confirmedHeadBlock(rootChainContext, newHeader)
	if !ok {
		return nil, nil, false
	}

	from, ok = rl.fromBlockAfterLastPersisted(to)
	if !ok {
		return nil, nil, false
	}

	// Prepare block range
	if to.Cmp(from) == -1 {
		from = to
	}

	return from, to, true
}

// confirmedHeadBlock returns the block number this header confirms as final:
// the header itself if it's already a `finalized` header, or the header
// minus requiredConfirmations if it's only a `latest` header. ok is false
// when the header is too recent to have accrued enough confirmations yet.
func (rl *RootChainListener) confirmedHeadBlock(rootChainContext *RootChainListenerContext, newHeader *blockHeader) (*big.Int, bool) {
	headerNumber := newHeader.header.Number
	if newHeader.isFinalized {
		return headerNumber, true
	}

	requiredConfirmations := rootChainContext.ChainmanagerParams.MainChainTxConfirmations
	confirmationBlocks := big.NewInt(0).SetUint64(requiredConfirmations)
	if headerNumber.Cmp(confirmationBlocks) <= 0 {
		rl.Logger.Error("RootChainListener: block number less than confirmations required", "blockNumber", headerNumber.Uint64, "confirmationsRequired", confirmationBlocks.Uint64)
		return nil, false
	}

	return headerNumber.Sub(headerNumber, confirmationBlocks), true
}

// fromBlockAfterLastPersisted returns the first block to process: the block
// after the last one persisted to storage, or headerNumber itself if nothing
// has been persisted yet. ok is false when storage couldn't be read, or the
// last persisted block is already at or past headerNumber (nothing new).
func (rl *RootChainListener) fromBlockAfterLastPersisted(headerNumber *big.Int) (*big.Int, bool) {
	hasLastBlock, _ := rl.storageClient.Has([]byte(lastRootBlockKey), nil)
	if !hasLastBlock {
		return headerNumber, true
	}

	lastBlockBytes, err := rl.storageClient.Get([]byte(lastRootBlockKey), nil)
	if err != nil {
		rl.Logger.Error("RootChainListener: error while fetching last block bytes from storage", "error", err)
		return nil, false
	}

	rl.Logger.Debug("RootChainListener: got last block from bridge storage", "lastBlock", string(lastBlockBytes))

	result, err := strconv.ParseUint(string(lastBlockBytes), 10, 64)
	if err != nil {
		return headerNumber, true
	}
	if result >= headerNumber.Uint64() {
		return nil, false
	}

	return big.NewInt(0).SetUint64(result + 1), true
}

// processRootChainBlockRangeInChunks processes [from, to] in
// maxRootChainBlockRange-sized chunks to avoid oversized FilterLogs
// responses, stopping (without advancing the cursor further) on the first
// chunk that fails.
func (rl *RootChainListener) processRootChainBlockRangeInChunks(rootChainContext *RootChainListenerContext, from, to *big.Int) {
	// Shared across every chunk and every bisection level this call makes —
	// see rootChainRejectionState.
	state := newRootChainRejectionState()

	for chunkFrom := new(big.Int).Set(from); chunkFrom.Cmp(to) <= 0; {
		chunkTo := new(big.Int).Add(chunkFrom, big.NewInt(maxRootChainBlockRange-1))
		if chunkTo.Cmp(to) > 0 {
			chunkTo = to
		}

		if err := rl.processRootChainBlockRange(rootChainContext, chunkFrom, chunkTo, state); err != nil {
			rl.Logger.Error(
				"queryAndBroadcastEvents failed",
				"error", err,
				"from", chunkFrom,
				"to", chunkTo,
			)
			// do not advance the cursor, as we want to retry this range on the next header
			rl.pruneStaleLogFailureCounts(state.countedThisCycle)
			return
		}

		chunkFrom = new(big.Int).Add(chunkTo, big.NewInt(1))
	}

	rl.pruneStaleLogFailureCounts(state.countedThisCycle)
}

// processRootChainBlockRange queries and handles logs for a block range. If the
// range fails, it is split into smaller ranges until either processing succeeds
// or a single-block query fails. The root block cursor is advanced only after the
// current range has been fully processed. A single block whose log has been
// quarantined (see rootChainRejectionState) is treated as processed rather
// than retried forever.
func (rl *RootChainListener) processRootChainBlockRange(rootChainContext *RootChainListenerContext, fromBlock *big.Int, toBlock *big.Int, state *rootChainRejectionState) error {
	if err := rl.queryAndBroadcastEvents(rootChainContext, fromBlock, toBlock, state); err != nil {
		// A single-block failure cannot be split further.
		if fromBlock.Cmp(toBlock) >= 0 {
			if state.quarantine != nil {
				rl.quarantineRootChainLog(state.quarantine)
				state.quarantine = nil
				return rl.persistLastRootBlock(toBlock)
			}
			// Return the error so the caller keeps the cursor unchanged and
			// retries this block later.
			return err
		}

		// Split the failed range and retry smaller ranges. If the left half
		// also fails, it will be split again by the recursive call below.
		midBlock := splitBlockRange(fromBlock, toBlock)
		rl.Logger.Warn(
			"RootChainListener: splitting rootChain event log query after RPC failure",
			"error", err,
			"fromBlock", fromBlock,
			"toBlock", toBlock,
			"leftToBlock", midBlock,
		)

		// Process the earlier half first to preserve root-chain block order.
		if err := rl.processRootChainBlockRange(rootChainContext, fromBlock, midBlock, state); err != nil {
			return err
		}

		// Process the later half only after the earlier half has succeeded.
		nextBlock := new(big.Int).Add(midBlock, big.NewInt(1))
		return rl.processRootChainBlockRange(rootChainContext, nextBlock, toBlock, state)
	}

	// Persist only after the full range has been handled successfully.
	return rl.persistLastRootBlock(toBlock)
}

func (rl *RootChainListener) persistLastRootBlock(block *big.Int) error {
	if err := rl.storageClient.Put([]byte(lastRootBlockKey), []byte(block.String()), nil); err != nil {
		rl.Logger.Error("RootChainListener: error persisting last root block in storage", "error", err, "lastRootBlock", block.String())
		return err
	}

	return nil
}

func splitBlockRange(fromBlock *big.Int, toBlock *big.Int) *big.Int {
	return new(big.Int).Add(
		fromBlock,
		new(big.Int).Div(new(big.Int).Sub(toBlock, fromBlock), big.NewInt(2)),
	)
}
