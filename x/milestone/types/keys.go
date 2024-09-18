package types

import (
	"cosmossdk.io/collections"
	"github.com/ethereum/go-ethereum/common"
)

const (
	// ModuleName is the name of the milestone module
	ModuleName = "milestone"

	// StoreKey is the string store representation
	StoreKey = ModuleName

	// RouterKey is the msg router key for the milestone module
	RouterKey = ModuleName
)

var (
	ParamsPrefixKey                = collections.NewPrefix([]byte{0x80})
	MilestoneMapPrefixKey          = collections.NewPrefix([]byte{0x81})
	CountPrefixKey                 = collections.NewPrefix([]byte{0x83})
	BlockNumberPrefixKey           = collections.NewPrefix([]byte{0x84})
	MilestoneTimeoutKPrefixKey     = collections.NewPrefix([]byte{0x85})
	MilestoneNoAckPrefixKey        = collections.NewPrefix([]byte{0x86})
	MilestoneLastNoAckKeyPrefixKey = collections.NewPrefix([]byte{0x87})
)

const StartBlock uint64 = 0

// ZeroHeimdallHash represents empty heimdall hash
var ZeroHeimdallHash = common.Hash{}.Bytes()
