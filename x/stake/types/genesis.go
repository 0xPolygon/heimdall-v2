package types

import (
	"encoding/json"

	"github.com/cosmos/cosmos-sdk/codec"
)

// NewGenesisState creates a new GenesisState instance
func NewGenesisState(validators []*Validator,
	currentValSet ValidatorSet,
	stakingSequences []string) *GenesisState {
	return &GenesisState{
		Validators:          validators,
		CurrentValidatorSet: currentValSet,
		StakingSequences:    stakingSequences,
	}
}

// DefaultGenesisState gets the raw genesis raw message for testing
func DefaultGenesisState() *GenesisState {
	return &GenesisState{}
}

// GetGenesisStateFromAppState returns x/stake GenesisState given raw application
// genesis state.
func GetGenesisStateFromAppState(cdc codec.JSONCodec, appState map[string]json.RawMessage) *GenesisState {
	var genesisState GenesisState

	if appState[ModuleName] != nil {
		cdc.MustUnmarshalJSON(appState[ModuleName], &genesisState)
	}

	return &genesisState
}
