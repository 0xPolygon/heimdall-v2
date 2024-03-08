package types

import (
	"time"

	"github.com/0xPolygon/heimdall-v2/types"
)

// NewEventRecord creates new record
func NewEventRecord(
	txHash types.HeimdallHash,
	logIndex uint64,
	id uint64,
	contract string,
	data types.HexBytes,
	chainID string,
	recordTime time.Time,
) EventRecord {
	return EventRecord{
		ID:         id,
		Contract:   contract,
		Data:       data,
		TxHash:     txHash,
		LogIndex:   logIndex,
		BorChainID: chainID,
		RecordTime: recordTime,
	}
}
