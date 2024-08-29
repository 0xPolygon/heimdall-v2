package types

// x/topup module event types
const (
	AttributeValueCategory        = ModuleName
	EventTypeTopup                = "topup"
	EventTypeFeeWithdraw          = "fee-withdraw"
	EventTypeWithdraw             = "withdraw"
	AttributeKeyRecipient         = "recipient"
	AttributeKeySender            = "sender"
	AttributeKeyUser              = "user"
	AttributeKeyTopupAmount       = "topup-amount"
	AttributeKeyFeeWithdrawAmount = "fee-withdraw-amount"

	// TODO HV2: move the following to heimdall-v2/types as they are not specific to topup

	AttributeKeyTxHash       = "txhash"
	AttributeKeySideTxResult = "side-tx-result"
)
