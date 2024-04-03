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
)
