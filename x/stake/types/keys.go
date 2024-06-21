package types

const (
	// ModuleName is the name of the staking module
	ModuleName = "stake"
	// StoreKey is the string store representation
	StoreKey = ModuleName
	// DefaultLogIndexUnit represents the default unit for txHash + logIndex
	DefaultLogIndexUnit = 100000
)

var (
	ValidatorsKey                   = []byte{0x21} // prefix for each key to a validator
	ValidatorSetKey                 = []byte{0x22} // prefix for each key for validator map
	CurrentValidatorSetKey          = []byte{0x23} // Key to store current validator set
	StakeSequenceKey                = []byte{0x24} // prefix for each key for staking sequence map
	SignerKey                       = []byte{0x25} //prefix for signer address for signer map
	CurrentMilestoneValidatorSetKey = []byte{0x25} // Key to store current validator set for milestone
)

type PubKey [65]byte

// ZeroPubKey represents empty pub key
var ZeroPubKey = PubKey{}
