package types_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/x/bor/types"
	chainmanagertypes "github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
	staketypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

func TestNewGenesisState(t *testing.T) {
	t.Parallel()

	t.Run("creates new genesis state with params and spans", func(t *testing.T) {
		t.Parallel()

		params := types.DefaultParams()
		spans := []types.Span{
			{Id: 1, StartBlock: 0, EndBlock: 1000},
		}

		gs := types.NewGenesisState(params, spans)

		require.NotNil(t, gs)
		require.Equal(t, params, gs.Params)
		require.Equal(t, spans, gs.Spans)
	})

	t.Run("creates genesis state with empty spans", func(t *testing.T) {
		t.Parallel()

		params := types.DefaultParams()
		gs := types.NewGenesisState(params, nil)

		require.NotNil(t, gs)
		require.Equal(t, params, gs.Params)
		require.Nil(t, gs.Spans)
	})

	t.Run("creates genesis state with custom params", func(t *testing.T) {
		t.Parallel()

		params := types.Params{
			SprintDuration: 32,
			SpanDuration:   12800,
			ProducerCount:  8,
		}
		var spans []types.Span

		gs := types.NewGenesisState(params, spans)

		require.NotNil(t, gs)
		require.Equal(t, uint64(32), gs.Params.SprintDuration)
		require.Equal(t, uint64(12800), gs.Params.SpanDuration)
		require.Equal(t, uint64(8), gs.Params.ProducerCount)
	})
}

func TestDefaultGenesisState(t *testing.T) {
	t.Parallel()

	t.Run("returns default genesis state", func(t *testing.T) {
		t.Parallel()

		gs := types.DefaultGenesisState()

		require.NotNil(t, gs)
		require.Equal(t, types.DefaultParams(), gs.Params)
		require.Nil(t, gs.Spans)
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
			Spans:  nil,
		}

		err := gs.Validate()
		require.NoError(t, err)
	})

	t.Run("rejects genesis state with invalid params", func(t *testing.T) {
		t.Parallel()

		gs := types.GenesisState{
			Params: types.Params{
				SprintDuration: 0, // Invalid
				SpanDuration:   6400,
				ProducerCount:  4,
			},
			Spans: nil,
		}

		err := gs.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "sprint duration")
	})

	t.Run("validates genesis state with spans", func(t *testing.T) {
		t.Parallel()

		gs := types.GenesisState{
			Params: types.DefaultParams(),
			Spans: []types.Span{
				{Id: 1, StartBlock: 0, EndBlock: 1000, BorChainId: "137"},
				{Id: 2, StartBlock: 1001, EndBlock: 2000, BorChainId: "137"},
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

func TestGenFirstSpan(t *testing.T) {
	t.Parallel()

	newValSet := func(n int) staketypes.ValidatorSet {
		validators := make([]*staketypes.Validator, n)
		for i := range validators {
			validators[i] = &staketypes.Validator{ValId: uint64(i + 1), Signer: fmt.Sprintf("0x%040x", i+1)}
		}
		return staketypes.ValidatorSet{Validators: validators}
	}

	cases := []struct {
		name          string
		numValidators int
		wantProducers []uint64
	}{
		{name: "many validators", numValidators: 5, wantProducers: []uint64{1}},
		{name: "two validators", numValidators: 2, wantProducers: []uint64{1}},
		{name: "single validator", numValidators: 1, wantProducers: []uint64{1}},
		{name: "no validators", numValidators: 0, wantProducers: nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			interfaceRegistry := codectypes.NewInterfaceRegistry()
			types.RegisterInterfaces(interfaceRegistry)
			cdc := codec.NewProtoCodec(interfaceRegistry)

			appState := map[string]json.RawMessage{
				types.ModuleName:             cdc.MustMarshalJSON(types.DefaultGenesisState()),
				chainmanagertypes.ModuleName: cdc.MustMarshalJSON(chainmanagertypes.DefaultGenesisState()),
			}

			appState, err := types.SetGenesisStateToAppState(cdc, appState, newValSet(tc.numValidators))
			require.NoError(t, err)

			spans := types.GetGenesisStateFromAppState(cdc, appState).Spans
			require.Len(t, spans, 1)
			require.Equal(t, uint64(0), spans[0].Id)
			require.Equal(t, uint64(0), spans[0].StartBlock)
			require.Len(t, spans[0].ValidatorSet.Validators, tc.numValidators)

			var gotProducers []uint64
			for _, p := range spans[0].SelectedProducers {
				gotProducers = append(gotProducers, p.ValId)
			}
			require.Equal(t, tc.wantProducers, gotProducers)
		})
	}
}

func TestGenesisModuleName(t *testing.T) {
	t.Parallel()

	t.Run("validates module name is set", func(t *testing.T) {
		t.Parallel()

		require.NotEmpty(t, types.ModuleName)
		require.Equal(t, "bor", types.ModuleName)
	})
}
