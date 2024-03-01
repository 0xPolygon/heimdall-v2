package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
)

// MarshallDividendAccount - amino Marshall DividendAccount
func MarshallCheckpoint(cdc codec.BinaryCodec, checkpoint Checkpoint) (bz []byte, err error) {
	bz, err = cdc.Marshal(&checkpoint)
	if err != nil {
		return bz, err
	}

	return bz, nil
}

// UnMarshallDividendAccount - amino Unmarshall DividendAccount
func UnMarshallCheckpoint(cdc codec.BinaryCodec, value []byte) (Checkpoint, error) {
	var checkpoint Checkpoint
	if err := cdc.Unmarshal(value, &checkpoint); err != nil {
		return checkpoint, err
	}

	return checkpoint, nil
}
