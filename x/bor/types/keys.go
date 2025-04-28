package types

import "cosmossdk.io/collections"

const (
	// ModuleName defines the module name
	ModuleName = "bor"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey defines the module's message routing key
	RouterKey = ModuleName
)

// Where is the right place for these?
const (
	TimeBasedSpanBlockHeightKey = "time-based-span-block-height"
	TimeBasedSpanProducersKey   = "time-based-span-producers"
)

// Keys for store prefixes
var (
	LastSpanIDKey            = collections.NewPrefix(0x35) // Key to store last span
	SpanPrefixKey            = collections.NewPrefix(0x36) // Prefix key to store span
	SeedLastBlockProducerKey = collections.NewPrefix(0x37) // key to store the last bor blocks producer seed
	ParamsKey                = collections.NewPrefix(0x38) // Key to store the params in the store
)
