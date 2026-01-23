package types_test

import (
	"encoding/json"
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
)

func TestNewGenesisState(t *testing.T) {
	t.Parallel()

	t.Run("creates genesis state with all parameters", func(t *testing.T) {
		t.Parallel()

		params := types.DefaultParams()
		gs := types.NewGenesisState(params, nil, 0, 0, nil)

		require.Equal(t, params, gs.Params)
		require.Nil(t, gs.BufferedCheckpoint)
		require.Equal(t, uint64(0), gs.LastNoAck)
		require.Equal(t, uint64(0), gs.AckCount)
		require.Nil(t, gs.Checkpoints)
	})

	t.Run("creates genesis state with checkpoints", func(t *testing.T) {
		t.Parallel()

		params := types.DefaultParams()
		checkpoints := []types.Checkpoint{
			{Id: 1, StartBlock: 0, EndBlock: 1000},
		}
		gs := types.NewGenesisState(params, nil, 0, 1, checkpoints)

		require.Equal(t, params, gs.Params)
		require.Equal(t, uint64(1), gs.AckCount)
		require.Len(t, gs.Checkpoints, 1)
	})
}

func TestDefaultGenesisState(t *testing.T) {
	t.Parallel()

	t.Run("returns default genesis state", func(t *testing.T) {
		t.Parallel()

		gs := types.DefaultGenesisState()

		require.NotNil(t, gs)
		require.Equal(t, types.DefaultParams(), gs.Params)
	})

	t.Run("default genesis state is valid", func(t *testing.T) {
		t.Parallel()

		gs := types.DefaultGenesisState()

		err := gs.Validate()
		require.NoError(t, err)
	})
}

func TestGenesisState_Validate(t *testing.T) {
	t.Parallel()

	t.Run("validates correct genesis state", func(t *testing.T) {
		t.Parallel()

		gs := types.GenesisState{
			Params: types.DefaultParams(),
		}

		err := gs.Validate()
		require.NoError(t, err)
	})

	t.Run("rejects genesis state with invalid params", func(t *testing.T) {
		t.Parallel()

		gs := types.GenesisState{
			Params: types.Params{
				MaxCheckpointLength:     0, // Invalid
				AvgCheckpointLength:     512,
				ChildChainBlockInterval: 10000,
			},
		}

		err := gs.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "max checkpoint length")
	})

	t.Run("validates genesis state with checkpoints", func(t *testing.T) {
		t.Parallel()

		gs := types.GenesisState{
			Params: types.DefaultParams(),
			Checkpoints: []types.Checkpoint{
				{
					Id:         1,
					Proposer:   "0x1234567890123456789012345678901234567890",
					StartBlock: 0,
					EndBlock:   1000,
					RootHash:   common.HexToHash("0xabcdef123456").Bytes(),
					BorChainId: "137",
				},
			},
			AckCount: 1,
		}

		err := gs.Validate()
		require.NoError(t, err)
	})

	t.Run("rejects checkpoints count mismatch with ack count", func(t *testing.T) {
		t.Parallel()

		gs := types.GenesisState{
			Params: types.DefaultParams(),
			Checkpoints: []types.Checkpoint{
				{
					Id:         1,
					Proposer:   "0x1234567890123456789012345678901234567890",
					StartBlock: 0,
					EndBlock:   1000,
					RootHash:   common.HexToHash("0xabcdef123456").Bytes(),
					BorChainId: "137",
				},
			},
			AckCount: 2, // Mismatch
		}

		err := gs.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "incorrect state")
	})

	t.Run("rejects checkpoint with invalid message", func(t *testing.T) {
		t.Parallel()

		gs := types.GenesisState{
			Params: types.DefaultParams(),
			Checkpoints: []types.Checkpoint{
				{
					Id:         1,
					Proposer:   "invalid-address",
					StartBlock: 0,
					EndBlock:   1000,
					RootHash:   common.HexToHash("0xabcdef123456").Bytes(),
					BorChainId: "137",
				},
			},
			AckCount: 1,
		}

		err := gs.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid proposer")
	})

	t.Run("rejects checkpoint ID mismatch", func(t *testing.T) {
		t.Parallel()

		gs := types.GenesisState{
			Params: types.DefaultParams(),
			Checkpoints: []types.Checkpoint{
				{
					Id:         5, // Should be 1
					Proposer:   "0x1234567890123456789012345678901234567890",
					StartBlock: 0,
					EndBlock:   1000,
					RootHash:   common.HexToHash("0xabcdef123456").Bytes(),
					BorChainId: "137",
				},
			},
			AckCount: 1,
		}

		err := gs.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "checkpoint id mismatch")
	})

	t.Run("validates checkpoint signatures", func(t *testing.T) {
		t.Parallel()

		validAddr := common.HexToAddress("0x1234567890123456789012345678901234567890")
		gs := types.GenesisState{
			Params: types.DefaultParams(),
			CheckpointSignatures: types.CheckpointSignatures{
				Signatures: []types.CheckpointSignature{
					{
						ValidatorAddress: validAddr.Bytes(),
						Signature:        []byte("valid_signature"),
					},
				},
			},
		}

		err := gs.Validate()
		require.NoError(t, err)
	})

	t.Run("rejects checkpoint signature with invalid address", func(t *testing.T) {
		t.Parallel()

		invalidAddr := []byte("invalid") // Not a proper address format
		gs := types.GenesisState{
			Params: types.DefaultParams(),
			CheckpointSignatures: types.CheckpointSignatures{
				Signatures: []types.CheckpointSignature{
					{
						ValidatorAddress: invalidAddr,
						Signature:        []byte("valid_signature"),
					},
				},
			},
		}

		err := gs.Validate()
		require.Error(t, err)
	})

	t.Run("rejects checkpoint signature with empty signature", func(t *testing.T) {
		t.Parallel()

		validAddr := common.HexToAddress("0x1234567890123456789012345678901234567890")
		gs := types.GenesisState{
			Params: types.DefaultParams(),
			CheckpointSignatures: types.CheckpointSignatures{
				Signatures: []types.CheckpointSignature{
					{
						ValidatorAddress: validAddr.Bytes(),
						Signature:        []byte{}, // Empty
					},
				},
			},
		}

		err := gs.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "checkpoint signature is empty")
	})

	t.Run("rejects non-empty checkpoint signatures tx hash", func(t *testing.T) {
		t.Parallel()

		gs := types.GenesisState{
			Params:                       types.DefaultParams(),
			CheckpointSignaturesTxhash: "some_txhash",
		}

		err := gs.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "checkpoint signatures tx hash is not valid")
	})
}

func TestGetGenesisStateFromAppState(t *testing.T) {
	t.Parallel()

	t.Run("retrieves genesis state from app state", func(t *testing.T) {
		t.Parallel()

		interfaceRegistry := codectypes.NewInterfaceRegistry()
		types.RegisterInterfaces(interfaceRegistry)
		cdc := codec.NewProtoCodec(interfaceRegistry)

		gs := types.DefaultGenesisState()
		appState := make(map[string]json.RawMessage)
		appState[types.ModuleName] = cdc.MustMarshalJSON(gs)

		result := types.GetGenesisStateFromAppState(cdc, appState)

		require.NotNil(t, result)
		require.Equal(t, gs.Params, result.Params)
	})

	t.Run("returns empty genesis state when module not in app state", func(t *testing.T) {
		t.Parallel()

		interfaceRegistry := codectypes.NewInterfaceRegistry()
		cdc := codec.NewProtoCodec(interfaceRegistry)

		appState := make(map[string]json.RawMessage)

		result := types.GetGenesisStateFromAppState(cdc, appState)

		require.NotNil(t, result)
	})
}

func TestGenesisModuleName(t *testing.T) {
	t.Parallel()

	t.Run("validates module name is set", func(t *testing.T) {
		t.Parallel()

		require.NotEmpty(t, types.ModuleName)
		require.Equal(t, "checkpoint", types.ModuleName)
	})
}
