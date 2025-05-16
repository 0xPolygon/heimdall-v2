package types

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/ethereum/go-ethereum/common"
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

// ValidateGenesis validates the provided checkpoint data
func (gs GenesisState) ValidateGenesis() error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}

	if len(gs.Checkpoints) != 0 {
		if int(gs.AckCount) != len(gs.Checkpoints) {
			return errors.New("incorrect state in state-dump , please Check")
		}

		ac := address.HexCodec{}
		for i, checkpoint := range gs.Checkpoints {
			// create checkpoint message for the purpose of validation
			msg := NewMsgCheckpointBlock(
				checkpoint.Proposer,
				checkpoint.StartBlock,
				checkpoint.EndBlock,
				checkpoint.RootHash,
				nil, // account root hash is not used to validate checkpoint
				checkpoint.BorChainId,
			)
			if err := msg.ValidateBasic(ac); err != nil {
				return err
			}
			checkpointIndex := uint64(i) + 1
			if checkpoint.Id != checkpointIndex {
				return errors.New("checkpoint id mismatch")
			}
		}
	}

	if len(gs.CheckpointSignatures.Signatures) > 0 {
		for _, s := range gs.CheckpointSignatures.Signatures {
			if err := address.VerifyAddressFormat(s.ValidatorAddress); err != nil {
				return err
			}

			if len(s.Signature) == 0 {
				return errors.New("checkpoint signature is empty")
			}
		}
	}

	if gs.CheckpointSignaturesTxhash != "" {
		if len(gs.CheckpointSignaturesTxhash) != common.HashLength*2 {
			return errors.New("checkpoint signatures txhash is not valid")
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
