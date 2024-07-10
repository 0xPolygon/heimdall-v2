package testutil

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/ethereum/go-ethereum/common"
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
		Timestamp:   timestamp,
	}
}

func RandomBytes() []byte {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return b
}

func IsEqual(a, b *types.Milestone) bool {
	if a == nil || b == nil {
		return false
	}

	if a.StartBlock != b.StartBlock {
		return false
	}

	if a.EndBlock != b.EndBlock {
		return false
	}

	if !bytes.Equal(a.Hash.Hash, b.Hash.Hash) {
		fmt.Print("here")
		return false
	}

	if a.Proposer != b.Proposer {
		return false
	}

	return true
}
