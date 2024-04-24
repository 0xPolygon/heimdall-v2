package types

import (
	"encoding/json"

	chainmanagertypes "github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
	"github.com/cosmos/cosmos-sdk/codec"
)

// NewGenesisState creates a new genesis state for bor.
func NewGenesisState(params Params, spans []*Span) *GenesisState {
	return &GenesisState{
		Params: params,
		Spans:  spans,
	}
}

// DefaultGenesisState returns a default genesis state for bor
func DefaultGenesisState() *GenesisState {
	return NewGenesisState(DefaultParams(), nil)
}

// Validate performs basic validation of bor genesis data returning an
// error for any failed validation criteria.
func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}

	return nil
}

// GetGenesisStateFromAppState returns x/bor GenesisState given raw application
// genesis state.
func GetGenesisStateFromAppState(cdc codec.JSONCodec, appState map[string]json.RawMessage) *GenesisState {
	var genesisState GenesisState

	if appState[ModuleName] != nil {
		cdc.MustUnmarshalJSON(appState[ModuleName], &genesisState)
	}

	return &genesisState
}

// SetGenesisStateToAppState sets x/bor GenesisState into raw application
// genesis state.
func SetGenesisStateToAppState(cdc codec.JSONCodec, appState map[string]json.RawMessage, currentValSet ValidatorSet) (map[string]json.RawMessage, error) {
	// set state to bor state
	borState := GetGenesisStateFromAppState(cdc, appState)
	chainState := chainmanagertypes.GetGenesisStateFromAppState(cdc, appState)
	borState.Spans = genFirstSpan(currentValSet, chainState.Params.ChainParams.BorChainId)

	appState[ModuleName] = cdc.MustMarshalJSON(borState)

	return appState, nil
}

// genFirstSpan generates default first validator producer set
func genFirstSpan(valset ValidatorSet, chainId string) []*Span {
	var (
		firstSpan         []*Span
		selectedProducers []Validator
	)

	if len(valset.Validators) > int(DefaultProducerCount) {
		// pop top validators and select
		for i := 0; uint64(i) < DefaultProducerCount; i++ {
			selectedProducers = append(selectedProducers, *valset.Validators[i])
		}
	} else {
		for _, val := range valset.Validators {
			selectedProducers = append(selectedProducers, *val)
		}
	}

	newSpan := Span{
		Id:                0,
		StartBlock:        0,
		EndBlock:          0 + DefaultFirstSpanDuration - 1,
		ValidatorSet:      valset,
		SelectedProducers: selectedProducers,
		ChainId:           chainId,
	}

	firstSpan = append(firstSpan, &newSpan)

	return firstSpan
}
