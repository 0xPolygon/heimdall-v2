package types

import (
	"encoding/json"

	"github.com/cosmos/cosmos-sdk/codec"

	"github.com/0xPolygon/heimdall-v2/helper"
)

// NewGenesisState creates a new genesis state.
func NewGenesisState() GenesisState {
	return GenesisState{}
}

// DefaultGenesisState gets the raw genesis raw message for testing
func DefaultGenesisState() *GenesisState {
	return &GenesisState{}
}

// GetGenesisStateFromAppState returns x/Milestone GenesisState given raw application
// genesis state.
func GetGenesisStateFromAppState(cdc codec.JSONCodec, appState map[string]json.RawMessage) *GenesisState {
	var genesisState GenesisState

	if appState[ModuleName] != nil {
		cdc.MustUnmarshalJSON(appState[ModuleName], &genesisState)
	}

	return &genesisState
}

// DefaultParams returns a default set of parameters.
func DefaultParams() Params {
	return Params{
		MinMilestoneLength:       helper.MilestoneLength,
		MilestoneBufferTime:      helper.MilestoneBufferTime,
		MilestoneBufferLength:    helper.MilestoneBufferLength,
		MilestoneTxConfirmations: helper.MaticChainMilestoneConfirmation,
	}
}
