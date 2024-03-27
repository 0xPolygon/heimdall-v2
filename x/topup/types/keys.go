package types

import "cosmossdk.io/collections"

const (
	// ModuleName is the name of the module
	ModuleName = "topup"

	// StoreKey is the store key string for bor
	StoreKey = ModuleName

	// RouterKey is the message route for bor
	RouterKey = ModuleName
)

// TODO HV2: move these vars into types/keys.go ?
var (
	// DefaultTopupSequenceValue defines the default value of the topup sequence key
	DefaultTopupSequenceValue = true
	// TopupSequencePrefixKey represents the topup sequence prefix key
	TopupSequencePrefixKey = collections.NewPrefix([]byte{0x81})
	// DividendAccountMapKey represents the prefix for each key for the dividend account map
	DividendAccountMapKey = collections.NewPrefix([]byte{0x82})
)
