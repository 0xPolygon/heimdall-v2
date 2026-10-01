package testutil

import (
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
)

// GenRandCheckpoint returns a random checkpoint header
// headerSize is an offset, not an inclusive length: the generated window spans
// headerSize+1 blocks, so a caller wanting a window the checkpoint length bound accepts
// has to pass an offset one below the intended length.
func GenRandCheckpoint(start, headerSize, id uint64) (headerBlock types.Checkpoint) {
	end := start + headerSize
	borChainID := "1234"
	rootHash := RandomBytes()
	proposer := common.Address{}.String()

	headerBlock = types.CreateCheckpoint(
		id,
		start,
		end,
		rootHash,
		proposer,
		borChainID,
		uint64(time.Now().UTC().Unix()))

	return headerBlock
}
