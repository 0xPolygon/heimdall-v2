package testutil

import (
	"crypto/rand"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/google/uuid"

	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
)

// GenRandMilestone creates and returns a random milestone
func GenRandMilestone(start uint64, milestoneLength uint64) (milestone types.Milestone) {
	end := start + milestoneLength - 1
	borChainID := "1234"
	hash := hmTypes.HeimdallHash{Hash: RandomBytes()}
	proposer := secp256k1.GenPrivKey().PubKey().Address().String()
	randN, _ := uuid.NewRandom()

	milestoneID := fmt.Sprintf("%s - %s", randN.String(), common.BytesToAddress(hash.GetHash()).String())
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
	_, _ = rand.Read(b)
	return b
}
