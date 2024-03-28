package types

import (
	"encoding/json"
	"errors"

	"github.com/cosmos/cosmos-sdk/codec"

	"github.com/0xPolygon/heimdall-v2/types"
)

// NewGenesisState creates a new genesis state for topup module
func NewGenesisState(sequences []string, accounts []types.DividendAccount) *GenesisState {
	return &GenesisState{
		TopupSequences:   sequences,
		DividendAccounts: accounts,
	}
}

// DefaultGenesisState returns a default genesis state for topup module
func DefaultGenesisState() *GenesisState {
	return NewGenesisState(nil, nil)
}

// Validate performs basic validation of topup genesis data
func (gs GenesisState) Validate() error {
	for _, sequence := range gs.TopupSequences {
		if sequence == "" {
			return errors.New("invalid sequence detected while validating genesis state")
		}
	}

	return nil
}

// GetGenesisStateFromAppState returns the topup GenesisState given a raw application genesis state
func GetGenesisStateFromAppState(cdc codec.JSONCodec, appState map[string]json.RawMessage) *GenesisState {
	var genesisState GenesisState
	if appState[ModuleName] != nil {
		cdc.MustUnmarshalJSON(appState[ModuleName], &genesisState)
	}

	return &genesisState
}
