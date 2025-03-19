package types

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
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}
