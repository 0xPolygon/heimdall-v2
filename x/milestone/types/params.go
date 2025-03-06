package types

import (
	"fmt"
)

// Validate checks that the milestone proposition's parameters have valid values.
func (p Params) Validate() error {
	if p.MaxMilestonePropositionLength == 0 {
		return fmt.Errorf("max milestone proposition length should not be zero")
	}
	return nil
}
