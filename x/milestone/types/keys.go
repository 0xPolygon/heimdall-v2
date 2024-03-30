package types

import "strconv"

const (
	// ModuleName is the name of the milestone module
	ModuleName = "milestone"

	// StoreKey is the string store representation
	StoreKey = ModuleName

	// RouterKey is the msg router key for the milestone module
	RouterKey = ModuleName
)

var (
	MilestoneKey          = []byte{0x10} // Key to store milestone
	CountKey              = []byte{0x11} //Key to store the count
	MilestoneNoAckKey     = []byte{0x12} //Key to store the NoAckMilestone
	MilestoneLastNoAckKey = []byte{0x13} //Key to store the Latest NoAckMilestone
	LastMilestoneTimeout  = []byte{0x14} //Key to store the Last Milestone Timeout
	BlockNumberKey        = []byte{0x15} //Key to store the count

	ParamsKey = []byte{0x16} // prefix for parameters
)

// GetMilestoneKey appends prefix to milestoneNumber
func GetMilestoneKey(milestoneNumber uint64) []byte {
	milestoneNumberBytes := []byte(strconv.FormatUint(milestoneNumber, 10))
	return append(MilestoneKey, milestoneNumberBytes...)
}

// GetMilestoneNoAckKey appends prefix to milestoneId
func GetMilestoneNoAckKey(milestoneId string) []byte {
	milestoneNoAckBytes := []byte(milestoneId)
	return append(MilestoneNoAckKey, milestoneNoAckBytes...)
}
