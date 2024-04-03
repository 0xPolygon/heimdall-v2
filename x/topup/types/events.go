package types

// topup module event types
const (
	AttributeValueCategory        = ModuleName
	EventTypeTopup                = "topup"
	EventTypeFeeWithdraw          = "fee-withdraw"
	EventTypeWithdraw             = "withdraw"
	EventTypeTransfer             = "transfer"
	AttributeKeyRecipient         = "recipient"
	AttributeKeySender            = "sender"
	AttributeKeyUser              = "user"
	AttributeKeyTopupAmount       = "topup-amount"
	AttributeKeyFeeWithdrawAmount = "fee-withdraw-amount"

	// TODO HV2: move the following to heimdall-v2/types as they are not specific to topup

	AttributeKeyTxHash       = "txhash"
	AttributeKeyTxLogIndex   = "tx-log-index"
	AttributeKeySideTxResult = "side-tx-result"
)
