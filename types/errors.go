package types

// Common error messages used across modules

const (
	// Event processing errors

	ErrMsgEventAlreadyProcessed = "event already processed"
	ErrMsgOldEventsNotAllowed   = "old events are not allowed"

	// Side transaction errors

	ErrMsgSideTxRejected = "sideTx didn't get yes votes"

	// Validation errors

	ErrMsgValidationFailed = "failed to validate msg"

	// Validator-related errors

	ErrMsgFailedToFetchValidator   = "failed to fetch validator from store"
	ErrMsgFailedToGetValidatorSet  = "failed to get validator set"
	ErrMsgValidatorAlreadyExists   = "validator has been a validator before, hence cannot join with same id"
	ErrMsgNoProposerInValidatorSet = "no proposer in validator set"
	ErrMsgInvalidProposerInMsg     = "invalid proposer in msg"
	ErrMsgInvalidProposerAddress   = "invalid proposer address"

	// Nonce and ID mismatch errors

	ErrMsgNonceMismatch = "nonce in message doesn't match with nonce in log"
	ErrMsgIDMismatch    = "id in message doesn't match with id in log"

	// Sequence errors

	ErrMsgUnableToSetSequence = "unable to set the sequence"

	// Block errors

	ErrMsgBlocksNotInContinuity = "blocks not in continuity"
	ErrMsgBlockNumberMismatch   = "blockNumber in message doesn't match blockNumber in receipt"

	// Public key validation errors

	ErrMsgPubKeyFirstByteMismatch = "public key first byte mismatch"
	ErrMsgInvalidPubKey           = "public key is invalid"

	// Address conversion errors

	ErrMsgConvertSignerToBytes         = "error in converting signer address to bytes"
	ErrMsgConvertEventLogSignerToBytes = "error in converting event log signer address to bytes"
	ErrMsgConvertHexToBytes            = "error in converting hex address to bytes"

	// Parameter fetching errors

	ErrMsgFailedToFetchParams              = "error in fetching params from store"
	ErrMsgFailedToGetChainParams           = "failed to get chain params"
	ErrMsgFailedToGetChainManagerParams    = "failed to get chain manager params"
	ErrMsgErrorInGettingChainManagerParams = "error in getting chain manager params"

	// Receipt and transaction errors

	ErrMsgErrorFetchingLog = "error fetching log from txHash"

	// Generic errors for specific types

	ErrMsgInvalidAmount  = "invalid amount for validator"
	ErrMsgAmountMismatch = "amount in message doesn't match Amount in event logs"

	// Vote extension errors

	ErrAlertVoteExtensionRejected                     = "alert, vote extension rejected. This should not happen; the validator could be malicious!"
	ErrAlertNonRpVoteExtensionRejected                = "alert, non-rp vote extension rejected. This should not happen; the validator could be malicious!"
	ErrAlertMilestonePropositionVoteExtensionRejected = "alert, milestone proposition vote extension rejected. This should not happen; the validator could be malicious!"
)
