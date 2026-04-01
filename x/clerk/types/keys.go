package types

import "cosmossdk.io/collections"

const (
	// ModuleName is the name of the module
	ModuleName = "clerk"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey defines the module's message routing key
	RouterKey = ModuleName
)

var (
	RecordsWithIDKeyPrefix             = collections.NewPrefix(0)
	RecordsWithTimeKeyPrefix           = collections.NewPrefix(1)
	RecordSequencesKeyPrefix           = collections.NewPrefix(2)
	RecordsWithVisibilityTimeKeyPrefix = collections.NewPrefix(3)
	VisibilityTimeUpgradeIDKeyPrefix   = collections.NewPrefix(4)
	PendingVisibilityEventsKeyPrefix   = collections.NewPrefix(5)
	VisibilityTimeByIDKeyPrefix        = collections.NewPrefix(6)
	BlockTimeIndexKeyPrefix            = collections.NewPrefix(7)
	VisibilityHeightByIDKeyPrefix      = collections.NewPrefix(8)
	BlockTimeReverseIndexKeyPrefix     = collections.NewPrefix(9)

	// DefaultValue of the record sequence
	DefaultValue = []byte{0x01}
)
