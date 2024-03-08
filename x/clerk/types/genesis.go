package types

import "errors"

// NewGenesisState creates a new genesis state.
func NewGenesisState(eventRecords []*EventRecord, recordSequences []string) GenesisState {
	return GenesisState{
		EventRecords:    eventRecords,
		RecordSequences: recordSequences,
	}
}

// DefaultGenesisState returns a default genesis state
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		EventRecords:    make([]*EventRecord, 0),
		RecordSequences: nil,
	}
}

// ValidateGenesis performs basic validation of bank genesis data returning an
// error for any failed validation criteria.
func ValidateGenesis(data GenesisState) error {
	for _, sq := range data.RecordSequences {
		if sq == "" {
			return errors.New("Invalid Sequence")
		}
	}

	return nil
}
