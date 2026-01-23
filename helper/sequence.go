package helper

import (
	"context"
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
// Formula: sequence = (blockNumber × DefaultLogIndexUnit) + logIndex
// This creates a unique identifier for each event based on its position in the blockchain.
func CalculateSequence(blockNumber, logIndex uint64) string {
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
	blockNumber, logIndex uint64,
	logger log.Logger,
	moduleName string,
) bool {
	sequence := CalculateSequence(blockNumber, logIndex)

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
