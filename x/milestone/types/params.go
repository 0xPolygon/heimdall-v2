package types

import (
	"fmt"

	borgrpc "github.com/0xPolygon/heimdall-v2/x/bor/grpc"
)

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
