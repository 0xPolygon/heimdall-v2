package testutil

import (
	"fmt"
	"time"

	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
)

// GenRandMilestone creates and returns a random milestone
func GenRandMilestone(start uint64, sprintLength uint64) (milestone types.Milestone) {
	end := start + sprintLength - 1
	borChainID := "1234"
	hash := common.Hash{}
	proposer := common.Address{}.String()

	milestoneID := fmt.Sprintf("%s - %s", uuid.NewRandom().String(), hmTypes.BytesToHeimdallAddress(hash[:]).String())
	milestone = types.CreateMilestone(
		start,
		end,
		hash,
		proposer,
		borChainID,
		milestoneID,
		uint64(time.Now().UTC().Unix()))

	return milestone
}
