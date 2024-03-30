package types

import (
	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/cosmos/cosmos-sdk/codec"
)

// MarshallMilestone marshal the milestone using binaryCodec
func MarshallMilestone(cdc codec.BinaryCodec, milestone Milestone) (bz []byte, err error) {
	bz, err = cdc.Marshal(&milestone)
	if err != nil {
		return bz, err
	}

	return bz, nil
}

// UnMarshallMilestone unmarshal the milestone using binaryCodec
func UnMarshallMilestone(cdc codec.BinaryCodec, value []byte) (Milestone, error) {
	var milestone Milestone
	if err := cdc.Unmarshal(value, &milestone); err != nil {
		return milestone, err
	}

	return milestone, nil
}

// CreateBlock generate new block
func CreateMilestone(
	start uint64,
	end uint64,
	hash hmTypes.HeimdallHash,
	proposer string,
	borChainID string,
	milestoneID string,
	timestamp uint64,
) Milestone {
	return Milestone{
		StartBlock:  start,
		EndBlock:    end,
		Hash:        hash,
		Proposer:    proposer,
		BorChainID:  borChainID,
		MilestoneID: milestoneID,
		TimeStamp:   timestamp,
	}
}
