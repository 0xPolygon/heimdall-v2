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

// SortCheckpoints sorts the array of checkpoints on the basis for timestamps
func SortCheckpoints(checkpoints []Checkpoint) []Checkpoint {
	sort.Slice(checkpoints, func(i, j int) bool {
		return checkpoints[i].Timestamp < checkpoints[j].Timestamp
	})

	return checkpoints
}

// ValidateCheckpointLength bounds a proposer-supplied [start, end] window to the same
// shape the bridge derives from the module params: no more than maxLength blocks, and a
// whole number of avgLength blocks unless the window sits exactly at maxLength (the cap
// the bridge applies when more blocks than that are available).
func ValidateCheckpointLength(start, end, avgLength, maxLength uint64) error {
	if end < start {
		return ErrInvalidCheckpointLength.Wrapf("end block %d precedes start block %d", end, start)
	}

	// Bound the span before turning it into a length: end-start+1 wraps to 0 for the
	// full uint64 range, which would then pass every check below.
	span := end - start
	if span >= maxLength {
		return ErrInvalidCheckpointLength.Wrapf("window %d-%d is longer than the maximum %d blocks", start, end, maxLength)
	}

	// Params validation keeps avgLength non-zero; guard anyway, a modulo by zero here
	// would panic every validator.
	if avgLength == 0 {
		return ErrInvalidCheckpointLength.Wrap("avg length is zero")
	}

	length := span + 1
	if length != maxLength && length%avgLength != 0 {
		return ErrInvalidCheckpointLength.Wrapf("length %d is not a multiple of %d", length, avgLength)
	}

	return nil
}
