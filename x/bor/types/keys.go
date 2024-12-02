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

// Keys for store prefixes
var (
	LastSpanIDKey            = collections.NewPrefix(0x35) // Key to store last span
	SpanPrefixKey            = collections.NewPrefix(0x36) // Prefix key to store span
	LastProcessedEthBlock    = collections.NewPrefix(0x38) // key to store last processed eth block for seed
	SeedLastBlockProducerKey = collections.NewPrefix(0x39) // key to store the last bor blocks producer seed
	ParamsKey                = collections.NewPrefix(0x40) // Key to store the params in the store
)
