package types

import (
	"cosmossdk.io/collections"
	types "github.com/0xPolygon/heimdall-v2/types"
	"github.com/ethereum/go-ethereum/common"
)

const (
	// ModuleName is the name of the staking module
	ModuleName = "checkpoint"

	// StoreKey is the string store representation
	StoreKey = ModuleName

	// RouterKey is the msg router key for the staking module
	RouterKey = ModuleName
)

var (
	// ParamsPrefixKey represents the prefix for param
	ParamsPrefixKey = collections.NewPrefix([]byte{0x80})

	// CheckpointMapPrefixKey represents the key for each key for the checkpoint map
	CheckpointMapPrefixKey = collections.NewPrefix([]byte{0x81})
	// BufferedCheckpointPrefixKey represents the prefix for buffered checkpoint
	BufferedCheckpointPrefixKey = collections.NewPrefix([]byte{0x82})

	// AckCountPrefixKey represents the prefix for ack count
	AckCountPrefixKey = collections.NewPrefix([]byte{0x83})

	// LastNoAckPrefixKey represents the prefix for last no ack
	LastNoAckPrefixKey = collections.NewPrefix([]byte{0x84})
)

// ZeroHeimdallHash represents empty pub key
var ZeroHeimdallHash = types.HeimdallHash{Hash: common.Hash{}.Bytes()}
