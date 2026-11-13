package types

import (
	"fmt"

	borgrpc "github.com/0xPolygon/heimdall-v2/x/bor/grpc"
)

// MilestonePropositionLengthCap bounds the number of blocks a validator puts in the proposition
// carried by its vote extension, whatever MaxMilestonePropositionLength is set to. Each block costs
// up to ~44 bytes, so at this cap the largest honest vote extension (50 side-tx responses plus the
// proposition) stays near 5KB: under the 10KB vote extension ceiling peers reject above from Karwar,
// and under the per-validator budget the proposer filter applies at the current block size and
// validator count. Above ~170 blocks honest extensions would be filtered out or rejected.
const MilestonePropositionLengthCap = 64

// ProposedLength returns how many blocks a validator may put in its milestone proposition.
func (p Params) ProposedLength() uint64 {
	return min(p.MaxMilestonePropositionLength, MilestonePropositionLengthCap)
}

// ValidateBasic checks that the milestone proposition's parameters have valid values.
func (p Params) ValidateBasic() error {
	if p.MaxMilestonePropositionLength == 0 {
		return fmt.Errorf("max milestone proposition length should not be zero")
	}
	// MaxMilestonePropositionLength feeds GetBorChainBlockInfoInBatch via
	// GenMilestoneProposition; raising it past the bor batch cap would make
	// every validator's proposition generation fail chain-wide.
	if p.MaxMilestonePropositionLength > uint64(borgrpc.MaxBlockInfoBatchSize) {
		return fmt.Errorf("max milestone proposition length %d exceeds bor batch size cap %d",
			p.MaxMilestonePropositionLength, borgrpc.MaxBlockInfoBatchSize)
	}
	if p.FfMilestoneThreshold == 0 {
		return fmt.Errorf("ff milestone threshold should not be zero")
	}
	if p.FfMilestoneBlockInterval == 0 {
		return fmt.Errorf("ff milestone block interval should not be zero")
	}
	if p.FfMilestoneBlockInterval >= p.FfMilestoneThreshold {
		return fmt.Errorf("ff milestone block interval should be less than ff milestone threshold")
	}
	if p.FfMilestoneThreshold%p.FfMilestoneBlockInterval != 0 {
		return fmt.Errorf("ff milestone threshold should be divisible by ff milestone block interval")
	}
	return nil
}
