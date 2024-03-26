package testutil

import (
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
	hmTypes "github.com/0xPolygon/heimdall-v2/x/types"
)

// GenRandCheckpoint return headers
func GenRandCheckpoint(start uint64, headerSize uint64, maxCheckpointLenght uint64) (headerBlock types.Checkpoint, err error) {
	end := start + headerSize
	borChainID := "1234"
	rootHash := hmTypes.HexToHeimdallHash("123")
	proposer := common.Address{}.String()

	headerBlock = types.CreateBlock(
		start,
		end,
		rootHash,
		proposer,
		borChainID,
		uint64(time.Now().UTC().Unix()))

	return headerBlock, nil
}
