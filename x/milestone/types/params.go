package types

import (
	fmt "fmt"

	"github.com/0xPolygon/heimdall-v2/helper"
)

func (p Params) Validate() error {
	if p.MinMilestoneLength == 0 || p.MilestoneBufferLength == 0 || p.MilestoneTxConfirmations == 0 {
		return fmt.Errorf("MinMilestoneLength, MilestoneBufferLength,MilestoneTxConfirmations should be non-zero")
	}

	if p.MilestoneBufferTime.Microseconds() == 0 {
		return fmt.Errorf("MilestoneBufferTime should not be zero")
	}

	return nil
}

func GetDefaultParams() Params {
	return Params{
		MinMilestoneLength:       helper.MilestoneLength,
		MilestoneBufferTime:      helper.MilestoneBufferTime,
		MilestoneBufferLength:    helper.MilestoneBufferLength,
		MilestoneTxConfirmations: helper.MaticChainMilestoneConfirmation,
	}
}
