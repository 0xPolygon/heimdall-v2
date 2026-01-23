package types_test

import (
	"encoding/json"
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/x/stake/types"
)

func TestNewGenesisState(t *testing.T) {
	t.Parallel()

	t.Run("creates genesis state with all parameters", func(t *testing.T) {
		t.Parallel()

		var validators []*types.Validator
		valSet := types.ValidatorSet{}
		stakingSeqs := []string{"seq1", "seq2"}

		gs := types.NewGenesisState(validators, valSet, stakingSeqs)

		require.NotNil(t, gs)
		require.Equal(t, validators, gs.Validators)
		require.Equal(t, valSet, gs.CurrentValidatorSet)
		require.Equal(t, stakingSeqs, gs.StakingSequences)
	})

	t.Run("creates genesis state with nil validators", func(t *testing.T) {
		t.Parallel()

		gs := types.NewGenesisState(nil, types.ValidatorSet{}, nil)

		require.NotNil(t, gs)
		require.Nil(t, gs.Validators)
		require.Nil(t, gs.StakingSequences)
	})
}

func TestDefaultGenesisState(t *testing.T) {
	t.Parallel()

	t.Run("returns default genesis state", func(t *testing.T) {
		t.Parallel()

		gs := types.DefaultGenesisState()

		require.NotNil(t, gs)
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

		gs := types.GenesisState{}

		err := gs.Validate()
		require.NoError(t, err)
	})

	t.Run("rejects empty staking sequence", func(t *testing.T) {
		t.Parallel()

		gs := types.GenesisState{
			StakingSequences: []string{""},
		}

		err := gs.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid sequence")
	})

	t.Run("rejects sequence with empty string in middle", func(t *testing.T) {
		t.Parallel()

		gs := types.GenesisState{
			StakingSequences: []string{"seq1", "", "seq3"},
		}

		err := gs.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid sequence")
	})

	t.Run("validates genesis with valid sequences", func(t *testing.T) {
		t.Parallel()

		gs := types.GenesisState{
			StakingSequences: []string{"seq1", "seq2", "seq3"},
		}

		err := gs.Validate()
		require.NoError(t, err)
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

func TestSetGenesisStateToAppState(t *testing.T) {
	t.Parallel()

	t.Run("sets genesis state to app state", func(t *testing.T) {
		t.Parallel()

		interfaceRegistry := codectypes.NewInterfaceRegistry()
		types.RegisterInterfaces(interfaceRegistry)
		cdc := codec.NewProtoCodec(interfaceRegistry)

		appState := make(map[string]json.RawMessage)
		var validators []*types.Validator
		valSet := types.ValidatorSet{}

		result, err := types.SetGenesisStateToAppState(cdc, appState, validators, valSet)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Contains(t, result, types.ModuleName)
	})

	t.Run("updates existing app state", func(t *testing.T) {
		t.Parallel()

		interfaceRegistry := codectypes.NewInterfaceRegistry()
		types.RegisterInterfaces(interfaceRegistry)
		cdc := codec.NewProtoCodec(interfaceRegistry)

		// Initialize with the default state
		gs := types.DefaultGenesisState()
		appState := make(map[string]json.RawMessage)
		appState[types.ModuleName] = cdc.MustMarshalJSON(gs)

		// Update with new validators
		var validators []*types.Validator
		valSet := types.ValidatorSet{}

		result, err := types.SetGenesisStateToAppState(cdc, appState, validators, valSet)

		require.NoError(t, err)
		require.NotNil(t, result)
	})
}
