package types

const (
	// ModuleName defines the module name
	ModuleName = "bor"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey defines the module's message routing key
	RouterKey = ModuleName
)

var (
	// Keys for store prefixes

	LastSpanIDKey         = []byte{0x35} // Key to store last span start block
	SpanPrefixKey         = []byte{0x36} // Key to store span start block
	LastProcessedEthBlock = []byte{0x38} // key to store last processed eth block for seed
	ParamsKey             = []byte{0x39} // ParamsKey is the key to store the params in the store
)
