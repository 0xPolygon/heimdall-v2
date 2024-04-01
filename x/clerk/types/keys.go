package types

import "cosmossdk.io/collections"

const (
	// ModuleName is the name of the module
	ModuleName = "clerk"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey defines the module's message routing key
	RouterKey = ModuleName

	// QuerierRoute is the querier route for bor
	QuerierRoute = ModuleName

	// DefaultParamSpace default name for parameter store
	DefaultParamSpace = ModuleName
)

var (
	// RecordKeyPrefix is the prefix for the record key
	RecordsWithIDKeyPrefix   = collections.NewPrefix(0)
	RecordsWithTimeKeyPrefix = collections.NewPrefix(1)
	RecordSequencesKeyPrefix = collections.NewPrefix(2)
)
