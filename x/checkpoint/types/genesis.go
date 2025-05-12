package types

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
)

// Default parameter values
const (
	// DefaultCheckpointBufferTime represents the time a checkpoint is allowed to stay in the buffer (1000s ~ 17m)
	DefaultCheckpointBufferTime           = 1000 * time.Second
	DefaultAvgCheckpointLength     uint64 = 256
	DefaultMaxCheckpointLength     uint64 = 1024
	DefaultChildChainBlockInterval uint64 = 10000
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
		LastNoAck:          lastNoACK,
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

// Validate validates the provided checkpoint data
func (gs GenesisState) Validate() error {
	if err := gs.Params.ValidateBasic(); err != nil {
		return err
	}

	if len(gs.Checkpoints) != 0 {
		if int(gs.AckCount) != len(gs.Checkpoints) {
			return errors.New("incorrect state in state-dump , please Check")
		}
	}

	return nil
}

// GetGenesisStateFromAppState returns x/checkpoint GenesisState given raw application
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
		CheckpointBufferTime:    DefaultCheckpointBufferTime,
		AvgCheckpointLength:     DefaultAvgCheckpointLength,
		MaxCheckpointLength:     DefaultMaxCheckpointLength,
		ChildChainBlockInterval: DefaultChildChainBlockInterval,
	}
}
