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

// FetchAndValidateReceipt fetches and validates the confirmed tx receipt;
// returns nil if either the fetch fails or the receipt's block number
// disagrees with the message's. Callers should vote NO on nil.
func FetchAndValidateReceipt(
	ctx context.Context,
	contractCaller IContractCaller,
	params ReceiptValidationParams,
	logger log.Logger,
) *ethTypes.Receipt {
	receipt, err := contractCaller.GetConfirmedTxReceipt(
		ctx,
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

	receipts := caller.BatchGetMainChainTxReceipts(ctx, txHashes)
	if len(receipts) == 0 {
		logger.Debug("Batch RPC returned no receipts", "requested", len(txHashes))
	}

	logger.Debug("Receipt prefetch complete", "requested", len(txHashes), "fetched", len(receipts), "time", time.Since(t0))

	caller.prefetchMu.Lock()
	caller.prefetchedReceipts = receipts
	caller.prefetchMu.Unlock()
}
