package types

// TODO HV2: remove unused constants

// stake module event types
var (
	EventTypeValidatorJoin = "validator-join"
	EventTypeSignerUpdate  = "signer-update"
	EventTypeStakeUpdate   = "stake-update"
	EventTypeValidatorExit = "validator-exit"

	AttributeKeySigner            = "signer"
	AttributeKeyDeactivationEpoch = "deactivation-epoch"
	AttributeKeyActivationEpoch   = "activation-epoch"
	AttributeKeyValidatorID       = "validator-id"
	AttributeKeyValidatorNonce    = "validator-nonce"
	AttributeKeyUpdatedAt         = "updated-at"

	AttributeValueCategory = ModuleName
)
