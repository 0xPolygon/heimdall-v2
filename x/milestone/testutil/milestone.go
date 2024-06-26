package testutil

import (
	"crypto/rand"
	"fmt"
	"time"

	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/google/uuid"
)

// GenRandMilestone creates and returns a random milestone
func GenRandMilestone(start uint64, milestoneLength uint64) (milestone types.Milestone) {
	end := start + milestoneLength - 1
	borChainID := "1234"
	hash := hmTypes.HeimdallHash{Hash: RandomBytes()}
	proposer := secp256k1.GenPrivKey().PubKey().Address().String()

	milestoneID := fmt.Sprintf("%s - %s", uuid.NewRandom().String(), hmTypes.BytesToHeimdallAddress(hash[:]).String())
	milestone = CreateMilestone(
		start,
		end,
		hash,
		proposer,
		borChainID,
		milestoneID,
		uint64(time.Now().UTC().Unix()))

	return milestone
}

// CreateMilestone generate new milestone
func CreateMilestone(
	start uint64,
	end uint64,
	hash hmTypes.HeimdallHash,
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
		BorChainID:  borChainID,
		MilestoneID: milestoneID,
		TimeStamp:   timestamp,
	}
}

func RandomBytes() []byte {
	b := make([]byte, 32)
	rand.Read(b)
	return b
}
