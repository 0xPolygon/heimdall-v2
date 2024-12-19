package types

import (
	"sort"
)

// CreateCheckpoint generate new checkpoint
func CreateCheckpoint(
	id uint64,
	start uint64,
	end uint64,
	rootHash []byte,
	proposer string,
	borChainID string,
	timestamp uint64,
) Checkpoint {
	return Checkpoint{
		Id:         id,
		StartBlock: start,
		EndBlock:   end,
		RootHash:   rootHash,
		Proposer:   proposer,
		BorChainId: borChainID,
		Timestamp:  timestamp,
	}
}

// SortCheckpoints sorts array of checkpoints on the basis for timestamps
func SortCheckpoints(headers []Checkpoint) []Checkpoint {
	sort.Slice(headers, func(i, j int) bool {
		return headers[i].Timestamp < headers[j].Timestamp
	})

	return headers
}
