package testutil

import (
	"time"

	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
	"github.com/ethereum/go-ethereum/common"
)

// GenRandCheckpoint returns a random checkpoint header
func GenRandCheckpoint(start uint64, headerSize uint64) (headerBlock types.Checkpoint) {
	end := start + headerSize
	borChainID := "1234"
	rootHash := RandomBytes()
	proposer := common.Address{}.String()

	headerBlock = types.CreateCheckpoint(
		start,
		end,
		rootHash,
		proposer,
		borChainID,
		uint64(time.Now().UTC().Unix()))

	return headerBlock
}
