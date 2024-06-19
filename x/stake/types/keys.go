package types

const (
	// ModuleName is the name of the staking module
	ModuleName = "stake"

	// StoreKey is the string store representation
	StoreKey = ModuleName

	// RouterKey is the msg router key for the staking module
	RouterKey = ModuleName

	// DefaultLogIndexUnit represents the default unit for txHash + logIndex
	DefaultLogIndexUnit = 100000
)

var (
	DefaultValue = true // Value to store in CacheCheckpoint and CacheCheckpointACK & ValidatorSetChange Flag

	ValidatorsKey                   = []byte{0x21} // prefix for each key to a validator
	ValidatorSetKey                 = []byte{0x22} // prefix for each key for validator map
	CurrentValidatorSetKey          = []byte{0x23} // Key to store current validator set
	StakeSequenceKey                = []byte{0x24} // prefix for each key for staking sequence map
	SignerKey                       = []byte{0x25} //prefix for signer address for signer map
	CurrentMilestoneValidatorSetKey = []byte{0x25} // Key to store current validator set for milestone
)

// PubKey pubkey
type PubKey [65]byte

// ZeroPubKey represents empty pub key
var ZeroPubKey = PubKey{}

// GetUpdatedValidators updates validators in validator set
func GetUpdatedValidators(
	currentSet *ValidatorSet,
	validators []*Validator,
	ackCount uint64,
) []*Validator {
	updates := make([]*Validator, 0, len(validators))

	for _, v := range validators {
		// create copy of validator
		validator := v.Copy()

		address := validator.Signer

		_, val := currentSet.GetByAddress(address)
		if val != nil && !validator.IsCurrentValidator(ackCount) {
			// remove validator
			validator.VotingPower = 0
			updates = append(updates, validator)
		} else if val == nil && validator.IsCurrentValidator(ackCount) {
			// add validator
			updates = append(updates, validator)
		} else if val != nil && validator.VotingPower != val.VotingPower {
			updates = append(updates, validator)
		}
	}

	return updates
}
