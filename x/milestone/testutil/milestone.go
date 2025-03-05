package testutil

import (
	"crypto/rand"
	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
)

// CreateMilestone generate new milestone
func CreateMilestone(
	start uint64,
	end uint64,
	hash []byte,
	proposer string,
	borChainID string,
	milestoneID string,
	timestamp uint64,
) types.Milestone {
	return types.Milestone{
		StartBlock:  start,
		EndBlock:    end,
		Hash:        hash,
		Proposer:    proposer,
		BorChainId:  borChainID,
		MilestoneId: milestoneID,
		Timestamp:   timestamp,
	}
}

func RandomBytes() []byte {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return b
}
