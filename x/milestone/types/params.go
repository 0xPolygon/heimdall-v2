package types

import (
	"fmt"

	"github.com/0xPolygon/heimdall-v2/helper"
)

func (p Params) Validate() error {
	if p.MinMilestoneLength == 0 {
		return fmt.Errorf("min milestone length should not be zero")
	}

	if p.MilestoneBufferLength == 0 {
		return fmt.Errorf("milestone buffer time should not be zero")
	}

	if p.MilestoneTxConfirmations == 0 {
		return fmt.Errorf("milestone tx confirmations should not be zero")
	}

	if p.MilestoneBufferTime.Microseconds() == 0 {
		return fmt.Errorf("milestone buffer time should not be zero")
	}

	return nil
}

// TODO HV2: remove this function if not needed

func GetDefaultParams() Params {
	return Params{
		MinMilestoneLength:       helper.MilestoneLength,
		MilestoneBufferTime:      helper.MilestoneBufferTime,
		MilestoneBufferLength:    helper.MilestoneBufferLength,
		MilestoneTxConfirmations: helper.BorChainMilestoneConfirmation,
	}
}
