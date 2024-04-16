package types

import "cosmossdk.io/collections"

const (
	// ModuleName is the name of the module
	ModuleName = "topup"

	// StoreKey is the store key string for bor
	StoreKey = ModuleName

	// DefaultDenom represents the default denominator for Polygon PoS coin
	DefaultDenom = "matic"

	// DefaultLogIndexUnit represents the default unit for txHash + logIndex
	DefaultLogIndexUnit = 100000
)

var (
	// DefaultTopupSequenceValue defines the default value of the topup sequence key
	DefaultTopupSequenceValue = true
	// TopupSequencePrefixKey represents the topup sequence prefix key
	TopupSequencePrefixKey = collections.NewPrefix([]byte{0x81})
	// DividendAccountMapKey represents the prefix for each key for the dividend account map
	DividendAccountMapKey = collections.NewPrefix([]byte{0x82})
)
