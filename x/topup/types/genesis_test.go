package types_test

import (
	"encoding/json"
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/stretchr/testify/require"

	heimdalltypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/topup/types"
)

func TestNewGenesisState(t *testing.T) {
	t.Parallel()

	t.Run("creates genesis state with sequences and accounts", func(t *testing.T) {
		t.Parallel()

		sequences := []string{"seq1", "seq2", "seq3"}
		accounts := []heimdalltypes.DividendAccount{
			{User: "0x1234567890123456789012345678901234567890"},
			{User: "0x1234567890123456789012345678901234567891"},
		}

		gs := types.NewGenesisState(sequences, accounts)

		require.NotNil(t, gs)
		require.Equal(t, sequences, gs.TopupSequences)
		require.Equal(t, accounts, gs.DividendAccounts)
	})

	t.Run("creates genesis state with nil values", func(t *testing.T) {
		t.Parallel()

		gs := types.NewGenesisState(nil, nil)

		require.NotNil(t, gs)
		require.Nil(t, gs.TopupSequences)
		require.Nil(t, gs.DividendAccounts)
	})
}

func TestDefaultGenesisState(t *testing.T) {
	t.Parallel()

	t.Run("returns default genesis state", func(t *testing.T) {
		t.Parallel()

		gs := types.DefaultGenesisState()

		require.NotNil(t, gs)
		require.Nil(t, gs.TopupSequences)
		require.Nil(t, gs.DividendAccounts)
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
			TopupSequences: []string{"seq1", "seq2"},
			DividendAccounts: []heimdalltypes.DividendAccount{
				{User: "0x1234567890123456789012345678901234567890"},
			},
		}

		err := gs.Validate()
		require.NoError(t, err)
	})

	t.Run("validates empty genesis state", func(t *testing.T) {
		t.Parallel()

		gs := types.GenesisState{}

		err := gs.Validate()
		require.NoError(t, err)
	})

	t.Run("rejects genesis with empty sequence", func(t *testing.T) {
		t.Parallel()

		gs := types.GenesisState{
			TopupSequences: []string{"seq1", "", "seq3"},
		}

		err := gs.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid sequence")
	})

	t.Run("rejects genesis with invalid dividend account", func(t *testing.T) {
		t.Parallel()

		gs := types.GenesisState{
			DividendAccounts: []heimdalltypes.DividendAccount{
				{User: "0x1234567890123456789012345678901234567890"},
				{User: ""},
			},
		}

		err := gs.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid dividend account")
	})

	t.Run("accepts genesis with only sequences", func(t *testing.T) {
		t.Parallel()

		gs := types.GenesisState{
			TopupSequences: []string{"seq1", "seq2"},
		}

		err := gs.Validate()
		require.NoError(t, err)
	})

	t.Run("accepts genesis with only accounts", func(t *testing.T) {
		t.Parallel()

		gs := types.GenesisState{
			DividendAccounts: []heimdalltypes.DividendAccount{
				{User: "0x1234567890123456789012345678901234567890"},
			},
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

		result, err := types.GetGenesisStateFromAppState(cdc, appState)

		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("returns empty genesis state when module not in app state", func(t *testing.T) {
		t.Parallel()

		interfaceRegistry := codectypes.NewInterfaceRegistry()
		cdc := codec.NewProtoCodec(interfaceRegistry)

		appState := make(map[string]json.RawMessage)

		result, err := types.GetGenesisStateFromAppState(cdc, appState)

		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("rejects invalid genesis state", func(t *testing.T) {
		t.Parallel()

		interfaceRegistry := codectypes.NewInterfaceRegistry()
		types.RegisterInterfaces(interfaceRegistry)
		cdc := codec.NewProtoCodec(interfaceRegistry)

		invalidGs := &types.GenesisState{
			TopupSequences: []string{""},
		}
		appState := make(map[string]json.RawMessage)
		appState[types.ModuleName] = cdc.MustMarshalJSON(invalidGs)

		result, err := types.GetGenesisStateFromAppState(cdc, appState)

		require.Error(t, err)
		require.Nil(t, result)
	})
}

func TestSetGenesisStateToAppState(t *testing.T) {
	t.Parallel()

	t.Run("sets genesis state to app state", func(t *testing.T) {
		t.Parallel()

		interfaceRegistry := codectypes.NewInterfaceRegistry()
		types.RegisterInterfaces(interfaceRegistry)
		cdc := codec.NewProtoCodec(interfaceRegistry)

		gs := types.DefaultGenesisState()
		appState := make(map[string]json.RawMessage)
		appState[types.ModuleName] = cdc.MustMarshalJSON(gs)

		dividendAccounts := []heimdalltypes.DividendAccount{
			{User: "0x1234567890123456789012345678901234567890"},
		}

		result, err := types.SetGenesisStateToAppState(cdc, appState, dividendAccounts)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Contains(t, result, types.ModuleName)
	})

	t.Run("updates existing genesis state", func(t *testing.T) {
		t.Parallel()

		interfaceRegistry := codectypes.NewInterfaceRegistry()
		types.RegisterInterfaces(interfaceRegistry)
		cdc := codec.NewProtoCodec(interfaceRegistry)

		originalGs := types.NewGenesisState([]string{"seq1"}, nil)
		appState := make(map[string]json.RawMessage)
		appState[types.ModuleName] = cdc.MustMarshalJSON(originalGs)

		newAccounts := []heimdalltypes.DividendAccount{
			{User: "0x1234567890123456789012345678901234567890"},
			{User: "0x1234567890123456789012345678901234567891"},
		}

		result, err := types.SetGenesisStateToAppState(cdc, appState, newAccounts)

		require.NoError(t, err)
		require.NotNil(t, result)

		var updatedGs types.GenesisState
		err = cdc.UnmarshalJSON(result[types.ModuleName], &updatedGs)
		require.NoError(t, err)
		require.Len(t, updatedGs.DividendAccounts, 2)
	})
}
