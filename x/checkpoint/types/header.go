package types

import (
	"sort"

	hmTypes "github.com/0xPolygon/heimdall-v2/types"
)

// CreateBlock generate new block
func CreateBlock(
	start uint64,
	end uint64,
	rootHash hmTypes.HeimdallHash,
	proposer string,
	borChainID string,
	timestamp uint64,
) Checkpoint {
	return Checkpoint{
		StartBlock: start,
		EndBlock:   end,
		RootHash:   rootHash,
		Proposer:   proposer,
		BorChainID: borChainID,
		TimeStamp:  timestamp,
	}
}

// SortHeaders sorts array of headers on the basis for timestamps
func SortHeaders(headers []Checkpoint) []Checkpoint {
	sort.Slice(headers, func(i, j int) bool {
		return headers[i].TimeStamp < headers[j].TimeStamp
	})

	return headers
}
