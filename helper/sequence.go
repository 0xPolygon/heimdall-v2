package helper

import (
	"context"
	"fmt"
	"math/big"

	"cosmossdk.io/log"

	"github.com/0xPolygon/heimdall-v2/types"
)

// SequenceChecker defines the interface for checking if a sequence exists in storage.
// Each keeper that uses sequence checking should implement this interface.
type SequenceChecker interface {
	// HasSequence checks if a sequence string exists in the keeper's storage.
	HasSequence(ctx context.Context, sequence string) bool
}

// CalculateSequence computes the unique sequence ID from block number and log index.
// Formula: sequence = (blockNumber × DefaultLogIndexUnit) + logIndex.
//
// That positional formula is only injective while logIndex < DefaultLogIndexUnit;
// a log index at or above the unit aliases the next block's low indices (e.g.
// (B, unit) == (B+1, 0)), so two distinct L1 events can map to the same key. Real L1
// blocks never reach that many logs, so historical keys are unchanged. At/after the Kyoto
// hardfork, an out-of-range log index gets a delimited, injective key instead;
// the height gate keeps pre-fork keys (and replay) byte-identical.
func CalculateSequence(height int64, blockNumber, logIndex uint64) string {
	if IsKyoto(height) && logIndex >= uint64(types.DefaultLogIndexUnit) {
		return fmt.Sprintf("%d_%d", blockNumber, logIndex)
	}
	blockNum := new(big.Int).SetUint64(blockNumber)
	sequence := new(big.Int).Mul(blockNum, big.NewInt(types.DefaultLogIndexUnit))
	sequence.Add(sequence, new(big.Int).SetUint64(logIndex))
	return sequence.String()
}

// CheckEventAlreadyProcessed checks if an event has already been processed based on its sequence.
// It calculates the sequence from blockNumber and logIndex, then checks if it exists in storage.
// Returns true if the event was already processed (caller should vote NO).
// Returns false if the event is new and can be processed.
func CheckEventAlreadyProcessed(
	ctx context.Context,
	checker SequenceChecker,
	height int64,
	blockNumber, logIndex uint64,
	logger log.Logger,
	moduleName string,
) bool {
	sequence := CalculateSequence(height, blockNumber, logIndex)

	if checker.HasSequence(ctx, sequence) {
		logger.Error("Event already processed",
			"module", moduleName,
			"sequence", sequence,
			"blockNumber", blockNumber,
			"logIndex", logIndex)
		return true
	}

	return false
}
