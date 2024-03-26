package types

import fmt "fmt"

func (p Params) Validate() error {
	if p.MaxCheckpointLength == 0 || p.AvgCheckpointLength == 0 {
		return fmt.Errorf("MaxCheckpointLength, AvgCheckpointLength should be non-zero")
	}

	if p.MaxCheckpointLength < p.AvgCheckpointLength {
		return fmt.Errorf("AvgCheckpointLength should not be greater than MaxCheckpointLength")
	}

	if p.ChildBlockInterval == 0 {
		return fmt.Errorf("ChildBlockInterval should be greater than zero")
	}

	return nil
}
