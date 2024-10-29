package types

import (
	"encoding/json"

	"github.com/cosmos/cosmos-sdk/codec"

	"github.com/0xPolygon/heimdall-v2/helper"
)

// NewGenesisState creates a new genesis state.
func NewGenesisState(params Params) GenesisState {
	return GenesisState{Params: params}
}

// DefaultGenesisState gets the raw genesis raw message for testing
func DefaultGenesisState() *GenesisState {
	params := DefaultParams()
	return &GenesisState{Params: params}
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

// ValidateGenesis validates the provided checkpoint data
func (gs GenesisState) ValidateGenesis() error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}

	return nil
}

// DefaultParams returns a default set of parameters.
func DefaultParams() Params {
	return Params{
		MinMilestoneLength:       helper.MilestoneLength,
		MilestoneBufferTime:      helper.MilestoneBufferTime,
		MilestoneBufferLength:    helper.MilestoneBufferLength,
		MilestoneTxConfirmations: helper.PolygonPosChainMilestoneConfirmation,
	}
}
