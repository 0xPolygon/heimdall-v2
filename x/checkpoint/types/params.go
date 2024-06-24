package types

import fmt "fmt"

func (p Params) Validate() error {
	if p.MaxCheckpointLength == 0 {
		return fmt.Errorf("max checkpoint length should be non-zero")
	}

	if p.AvgCheckpointLength == 0 {
		return fmt.Errorf("value of avg checkpoint length should be non-zero")
	}

	if p.MaxCheckpointLength < p.AvgCheckpointLength {
		return fmt.Errorf("avg checkpoint length should not be greater than max checkpoint length")
	}

	if p.ChildBlockInterval == 0 {
		return fmt.Errorf("child block interval should be greater than zero")
	}

	return nil
}
