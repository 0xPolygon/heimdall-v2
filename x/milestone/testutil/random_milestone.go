package testutil

import (
	"time"

	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
	"github.com/ethereum/go-ethereum/common"
)

// GenRandMilestone return headers
func GenRandMilestone(start uint64, sprintLength uint64) (milestone types.Milestone, err error) {
	end := start + sprintLength - 1
	borChainID := "1234"
	hash := hmTypes.HexToHeimdallHash("123")
	proposer := common.Address{}.String()

	milestoneID := "00000"
	milestone = types.CreateMilestone(
		start,
		end,
		hash,
		proposer,
		borChainID,
		milestoneID,
		uint64(time.Now().UTC().Unix()))

	return milestone, nil
}
