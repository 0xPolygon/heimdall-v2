package types_test

import (
	"encoding/json"
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
)

func TestNewGenesisState(t *testing.T) {
	t.Parallel()

	t.Run("creates genesis state with params", func(t *testing.T) {
		t.Parallel()

		params := types.DefaultParams()
		gs := types.NewGenesisState(params)

		require.NotNil(t, gs)
		require.Equal(t, params, gs.Params)
	})

	t.Run("creates genesis state with custom params", func(t *testing.T) {
		t.Parallel()

		customParams := types.NewParams(10, 20, types.ChainParams{})
		gs := types.NewGenesisState(customParams)

		require.NotNil(t, gs)
		require.Equal(t, customParams, gs.Params)
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

	t.Run("validates empty genesis state", func(t *testing.T) {
		t.Parallel()

		gs := types.GenesisState{}

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
		require.Equal(t, "chainmanager", types.ModuleName)
	})
}
