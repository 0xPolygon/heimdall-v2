package types

import "strconv"

const (
	// ModuleName is the name of the staking module
	ModuleName = "checkpoint"

	// StoreKey is the string store representation
	StoreKey = ModuleName

	// RouterKey is the msg router key for the staking module
	RouterKey = ModuleName
)

var (
	DefaultValue = []byte{0x01} // Value to store in CacheCheckpoint and CacheCheckpointACK & ValidatorSetChange Flag

	ACKCountKey         = []byte{0x11} // key to store ACK count
	BufferCheckpointKey = []byte{0x12} // Key to store checkpoint in buffer
	CheckpointKey       = []byte{0x13} // prefix key for when storing checkpoint after ACK
	LastNoACKKey        = []byte{0x14} // key to store last no-ack

	ParamsKey = []byte{0x15} // prefix for parameters

)

// GetCheckpointKey appends prefix to checkpointNumber
func GetCheckpointKey(checkpointNumber uint64) []byte {
	checkpointNumberBytes := []byte(strconv.FormatUint(checkpointNumber, 10))
	return append(CheckpointKey, checkpointNumberBytes...)
}
