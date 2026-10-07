package abci

import (
	"bytes"
	"fmt"
	"math"

	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
)

// An honest parent hash always equals the stored last milestone hash, so a prefix only has to match
// that known value.
const compactParentHashLength = 8

// compactProposition drops bytes every validator can rebuild: the parent hash beyond its prefix, and
// the latest head hash when the head is the proposition tail. A head at block 0 keeps its hash, since
// head number 0 without a hash means no head.
func compactProposition(height int64, prop *types.MilestoneProposition) *types.MilestoneProposition {
	if !helper.IsCompactVoteExt(height) {
		return prop
	}
	if len(prop.ParentHash) > compactParentHashLength {
		prop.ParentHash = prop.ParentHash[:compactParentHashLength]
	}
	if end, ok := propositionEnd(prop); ok && prop.LatestBlockNumber != 0 && prop.LatestBlockNumber == end {
		prop.LatestBlockHash = nil
	}
	return prop
}

// expandParentHash restores a compact parent prefix to the stored hash it was cut from, so the tally
// keeps comparing full hashes. Any other prefix stays short and never matches.
func expandParentHash(veHeight int64, parent, lastEndBlockHash []byte) []byte {
	if helper.IsCompactVoteExt(veHeight) && len(parent) == compactParentHashLength && bytes.HasPrefix(lastEndBlockHash, parent) {
		return lastEndBlockHash
	}
	return parent
}

// propositionEnd returns the last block of a non-empty proposition; !ok on overflow of the
// attacker-supplied start.
func propositionEnd(prop *types.MilestoneProposition) (uint64, bool) {
	if len(prop.BlockHashes) == 0 {
		return 0, false
	}
	offset := uint64(len(prop.BlockHashes) - 1)
	if prop.StartBlockNumber > math.MaxUint64-offset {
		return 0, false
	}
	return prop.StartBlockNumber + offset, true
}

// impliedLatestHead restores a latest head hash omitted because the head is the proposition tail.
func impliedLatestHead(prop *types.MilestoneProposition) *types.MilestoneProposition {
	if len(prop.LatestBlockHash) != 0 || prop.LatestBlockNumber == 0 {
		return prop
	}
	if end, ok := propositionEnd(prop); !ok || prop.LatestBlockNumber != end {
		return prop
	}
	expanded := *prop
	expanded.LatestBlockHash = prop.BlockHashes[len(prop.BlockHashes)-1]
	return &expanded
}

// validateCompactLatestHead allows one encoding per head: the hash is present only beyond the tail, or
// for a head at block 0.
func validateCompactLatestHead(prop *types.MilestoneProposition) error {
	if end, ok := propositionEnd(prop); ok && len(prop.LatestBlockHash) != 0 && prop.LatestBlockNumber != 0 && prop.LatestBlockNumber == end {
		return fmt.Errorf("latest block hash must be omitted at proposition end %d", end)
	}
	return validateLatestHead(impliedLatestHead(prop))
}
