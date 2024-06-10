package types

// stake module event types
var (
	EventTypeValidatorJoin = "validator-join"
	EventTypeSignerUpdate  = "signer-update"
	EventTypeStakeUpdate   = "stake-update"
	EventTypeValidatorExit = "validator-exit"

	AttributeKeySigner     = "signer"
	AttributeValueCategory = ModuleName
)
