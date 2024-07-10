package types

import (
	"bytes"
	"errors"

	"github.com/0xPolygon/heimdall-v2/helper"
	hmTypes "github.com/0xPolygon/heimdall-v2/types"
)

// IsValidCheckpoint validates if checkpoint rootHash matches or not
func IsValidCheckpoint(start uint64, end uint64, rootHash hmTypes.HeimdallHash, checkpointLength uint64, contractCaller helper.IContractCaller, confirmations uint64) (bool, error) {
	// Check if blocks exist locally
	if !contractCaller.CheckIfBlocksExist(end + confirmations) {
		return false, errors.New("blocks not found locally")
	}

	// Compare RootHash
	root, err := contractCaller.GetRootHash(start, end, checkpointLength)
	if err != nil {
		return false, err
	}

	if bytes.Equal(root, rootHash.GetHash()) {
		return true, nil
	}

	return false, nil
}
