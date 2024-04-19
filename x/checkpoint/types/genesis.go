package types

import (
	"encoding/json"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
)

// Default parameter values
const (
	DefaultCheckpointBufferTime time.Duration = 1000 * time.Second // Time checkpoint is allowed to stay in buffer (1000 seconds ~ 17 mins)
	DefaultAvgCheckpointLength  uint64        = 256
	DefaultMaxCheckpointLength  uint64        = 1024
	DefaultChildBlockInterval   uint64        = 10000
)

// NewGenesisState creates a new genesis state.
func NewGenesisState(
	params Params,
	bufferedCheckpoint *Checkpoint,
	lastNoACK uint64,
	ackCount uint64,
	checkpoints []Checkpoint,
) GenesisState {
	return GenesisState{
		Params:             params,
		BufferedCheckpoint: bufferedCheckpoint,
		LastNoACK:          lastNoACK,
		AckCount:           ackCount,
		Checkpoints:        checkpoints,
	}
}

// DefaultGenesisState gets the raw genesis raw message for testing
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params: DefaultParams(),
	}
}

// GetGenesisStateFromAppState returns x/Checkpoint GenesisState given raw application
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
		CheckpointBufferTime: DefaultCheckpointBufferTime,
		AvgCheckpointLength:  DefaultAvgCheckpointLength,
		MaxCheckpointLength:  DefaultMaxCheckpointLength,
		ChildBlockInterval:   DefaultChildBlockInterval,
	}
}
