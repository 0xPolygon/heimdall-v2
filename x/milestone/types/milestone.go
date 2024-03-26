package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
)

// MarshallMilestone marshal the milestone using binaryCodec
func MarshallCheckpoint(cdc codec.BinaryCodec, milestone Milestone) (bz []byte, err error) {
	bz, err = cdc.Marshal(&milestone)
	if err != nil {
		return bz, err
	}

	return bz, nil
}

// UnMarshallMilestone unmarshal the milestone using binaryCodec
func UnMarshallCheckpoint(cdc codec.BinaryCodec, value []byte) (Milestone, error) {
	var milestone Milestone
	if err := cdc.Unmarshal(value, &milestone); err != nil {
		return milestone, err
	}

	return milestone, nil
}
