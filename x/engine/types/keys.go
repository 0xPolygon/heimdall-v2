package types

import "cosmossdk.io/collections"

const (
	// ModuleName defines the module name
	ModuleName = "engine"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_engine"
)

var (
	ParamsKey = []byte("p_engine")

	// ExecutionStateMetadataPrefixKey represents the prefix for execution client metadata
	ExecutionStateMetadataPrefixKey = collections.NewPrefix([]byte{0x87})
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}
