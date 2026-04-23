package helper

import (
	"context"
	"time"

	"cosmossdk.io/log"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
)

// ReceiptValidationParams holds parameters for receipt validation.
// ModuleName is used for logging to identify which module is performing the validation.
type ReceiptValidationParams struct {
	TxHash         []byte
	MsgBlockNumber uint64
	Confirmations  uint64
	ModuleName     string
}

// FetchAndValidateReceipt fetches the confirmed transaction receipt and validates it.
// It performs two key validations:
// 1. Ensures the receipt exists and was fetched successfully
// 2. Ensures the block number in the receipt matches the block number in the message
//
// Returns the receipt if validation succeeds, or nil if validation fails.
// Callers should vote NO if this function returns nil.
func FetchAndValidateReceipt(
	contractCaller IContractCaller,
	params ReceiptValidationParams,
	logger log.Logger,
) *ethTypes.Receipt {
	// Get confirmed tx receipt
	receipt, err := contractCaller.GetConfirmedTxReceipt(
		common.BytesToHash(params.TxHash),
		params.Confirmations,
	)

	if receipt == nil || err != nil {
		logger.Error("Failed to get confirmed tx receipt",
			"module", params.ModuleName,
			"txHash", common.Bytes2Hex(params.TxHash),
			"error", err)
		return nil
	}

	// Validate block number matches
	if receipt.BlockNumber.Uint64() != params.MsgBlockNumber {
		logger.Error("Block number mismatch between message and receipt",
			"module", params.ModuleName,
			"msgBlockNumber", params.MsgBlockNumber,
			"receiptBlockNumber", receipt.BlockNumber.Uint64())
		return nil
	}

	return receipt
}

// PrefetchReceipts batch-fetches L1 receipts in a single JSON-RPC call and caches them.
// Finality is checked later by GetConfirmedTxReceipt when side handlers run.
func PrefetchReceipts(ctx context.Context, contractCaller IContractCaller, txHashes []common.Hash, logger log.Logger) {
	t0 := time.Now()

	if len(txHashes) == 0 {
		return
	}

	caller, ok := contractCaller.(*ContractCaller)
	if !ok {
		logger.Debug("Prefetch skipped: contractCaller is not *ContractCaller")
		return
	}

	// Filter out hashes already in cache.
	uncached := make([]common.Hash, 0, len(txHashes))
	for _, h := range txHashes {
		if caller.receiptCache == nil || !caller.receiptCache.Contains(h) {
			uncached = append(uncached, h)
		}
	}

	if len(uncached) == 0 {
		logger.Debug("Prefetch skipped: all receipts already cached", "total", len(txHashes))
		return
	}

	receipts := caller.BatchGetMainChainTxReceipts(ctx, uncached)
	if len(receipts) == 0 {
		logger.Debug("Batch RPC returned no receipts", "requested", len(uncached))
	}

	for hash, receipt := range receipts {
		caller.cacheReceipt(hash, receipt)
	}

	logger.Info("Receipt prefetch complete", "requested", len(txHashes), "uncached", len(uncached), "fetched", len(receipts), "time", time.Since(t0))
}
