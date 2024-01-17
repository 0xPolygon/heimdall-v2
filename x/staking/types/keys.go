package types

const (
	// ModuleName is the name of the staking module
	ModuleName = "staking"

	// StoreKey is the string store representation
	StoreKey = ModuleName

	// RouterKey is the msg router key for the staking module
	RouterKey = ModuleName
)

var (
	DefaultValue = []byte{0x01} // Value to store in CacheCheckpoint and CacheCheckpointACK & ValidatorSetChange Flag

	ValidatorsKey                   = []byte{0x21} // prefix for each key to a validator
	ValidatorMapKey                 = []byte{0x22} // prefix for each key for validator map
	CurrentValidatorSetKey          = []byte{0x23} // Key to store current validator set
	StakingSequenceKey              = []byte{0x24} // prefix for each key for staking sequence map
	CurrentMilestoneValidatorSetKey = []byte{0x25} // Key to store current validator set for milestone
)

// GetValidatorKey drafts the validator key for addresses
func GetValidatorKey(address []byte) []byte {
	return append(ValidatorsKey, address...)
}

// GetValidatorMapKey returns validator map
func GetValidatorMapKey(address []byte) []byte {
	return append(ValidatorMapKey, address...)
}

// GetStakingSequenceKey returns staking sequence key
func GetStakingSequenceKey(sequence string) []byte {
	return append(StakingSequenceKey, []byte(sequence)...)
}
