package types

import (
	"time"

	"github.com/0xPolygon/heimdall-v2/types"
)

// NewEventRecord creates new record
func NewEventRecord(
	txHash string,
	logIndex uint64,
	id uint64,
	contract string,
	data types.HexBytes,
	chainID string,
	recordTime time.Time,
) EventRecord {
	return EventRecord{
		Id:         id,
		Contract:   contract,
		Data:       data,
		TxHash:     txHash,
		LogIndex:   logIndex,
		BorChainId: chainID,
		RecordTime: recordTime,
	}
}
