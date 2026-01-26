package helper

import (
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
