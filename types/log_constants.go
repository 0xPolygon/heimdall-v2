package types

// Log attribute keys, used consistently across all modules for structured logging

const (
	// LogKeyError is the standard attribute key for error logging
	LogKeyError = "error"

	// LogKeySequence is used for logging sequence IDs in event processing
	LogKeySequence = "sequence"

	// Validator-related logging keys

	LogKeyValidatorID    = "validatorId"
	LogKeyValidatorNonce = "validatorNonce"

	// Message-related logging keys

	LogKeyMsgNonce       = "msgNonce"
	LogKeyMsgBlockNumber = "msgBlockNumber"
	LogKeyMsgID          = "msgId"

	// Transaction-related logging keys

	LogKeyLogIndex    = "logIndex"
	LogKeyBlockNumber = "blockNumber"

	// Receipt and event log keys

	LogKeyReceiptBlockNumber = "receiptBlockNumber"
	LogKeyNonceFromTx        = "nonceFromTx"
	LogKeyValidatorIdFromTx  = "validatorIdFromTx"

	// Amount and value keys

	LogKeyMsgAmount       = "msgAmount"
	LogKeyNewAmount       = "newAmount"
	LogKeyAmountFromEvent = "amountFromEvent"

	// Proposer and signer keys

	LogKeyProposer = "proposer"
	LogKeySigner   = "signer"
)
