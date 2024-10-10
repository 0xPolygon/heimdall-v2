package keeper

import (
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common"

	"github.com/0xPolygon/heimdall-v2/helper"
)

// ValidateMilestone - Validates if milestone rootHash matches or not
func ValidateMilestone(start uint64, end uint64, hash []byte, milestoneID string, contractCaller helper.IContractCaller, minMilestoneLength uint64, confirmations uint64) (bool, error) {
	msgMilestoneLength := int64(end) - int64(start) + 1

	// Check for the minimum length of the milestone
	if msgMilestoneLength < int64(minMilestoneLength) {
		return false, errors.New(fmt.Sprint("invalid milestone, difference in start and end block is less than milestone length", "milestone Length:", minMilestoneLength))
	}

	// Check if blocks+confirmations  exist locally
	if !contractCaller.CheckIfBlocksExist(end + confirmations) {
		return false, errors.New(fmt.Sprint("end block number with confirmation is not available in bor chain", "EndBlock", end, "confirmation", confirmations))
	}

	// Get the vote on hash of milestone from Bor
	vote, err := contractCaller.GetVoteOnHash(start, end, common.Bytes2Hex(hash), milestoneID)
	if err != nil {
		return false, err
	}

	return vote, nil
}
