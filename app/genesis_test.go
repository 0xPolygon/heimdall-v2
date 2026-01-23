package app_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/app"
)

func TestGenesisStateType(t *testing.T) {
	// Test that GenesisState can be initialized and is a map type
	var gs app.GenesisState
	require.Nil(t, gs) // Uninitialized map is nil

	gs = make(app.GenesisState)
	require.NotNil(t, gs) // After make(), map is not nil
	require.Len(t, gs, 0)
}

func TestGenesisStateBasicOperations(t *testing.T) {
	gs := make(app.GenesisState)

	// Test adding entries
	testData := json.RawMessage(`{"test": "data"}`)
	gs["module1"] = testData

	require.Len(t, gs, 1)
	require.Contains(t, gs, "module1")
	require.Equal(t, testData, gs["module1"])
}

func TestGenesisStateMultipleModules(t *testing.T) {
	gs := make(app.GenesisState)

	// Add multiple module genesis states
	gs["bank"] = json.RawMessage(`{"balances": []}`)
	gs["staking"] = json.RawMessage(`{"validators": []}`)
	gs["gov"] = json.RawMessage(`{"proposals": []}`)

	require.Len(t, gs, 3)
	require.Contains(t, gs, "bank")
	require.Contains(t, gs, "staking")
	require.Contains(t, gs, "gov")
}

func TestGenesisStateJSONMarshaling(t *testing.T) {
	gs := make(app.GenesisState)
	gs["test_module"] = json.RawMessage(`{"key": "value"}`)

	// Marshal to JSON
	data, err := json.Marshal(gs)
	require.NoError(t, err)
	require.NotEmpty(t, data)

	// Unmarshal from JSON
	var decoded app.GenesisState
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	require.Len(t, decoded, 1)
	require.Contains(t, decoded, "test_module")
	require.JSONEq(t, string(gs["test_module"]), string(decoded["test_module"]))
}

func TestGenesisStateEmptyState(t *testing.T) {
	gs := make(app.GenesisState)

	// Marshal empty state
	data, err := json.Marshal(gs)
	require.NoError(t, err)
	require.Equal(t, "{}", string(data))

	// Unmarshal empty state
	var decoded app.GenesisState
	err = json.Unmarshal([]byte("{}"), &decoded)
	require.NoError(t, err)
	require.Len(t, decoded, 0)
}

func TestGenesisStateNilValue(t *testing.T) {
	gs := make(app.GenesisState)
	gs["module"] = nil

	require.Len(t, gs, 1)
	require.Contains(t, gs, "module")
	require.Nil(t, gs["module"])
}

func TestGenesisStateOverwrite(t *testing.T) {
	gs := make(app.GenesisState)

	// Add initial value
	gs["module"] = json.RawMessage(`{"version": 1}`)
	require.JSONEq(t, `{"version": 1}`, string(gs["module"]))

	// Overwrite with new value
	gs["module"] = json.RawMessage(`{"version": 2}`)
	require.JSONEq(t, `{"version": 2}`, string(gs["module"]))
}

func TestGenesisStateDelete(t *testing.T) {
	gs := make(app.GenesisState)
	gs["module1"] = json.RawMessage(`{}`)
	gs["module2"] = json.RawMessage(`{}`)

	require.Len(t, gs, 2)

	// Delete a module
	delete(gs, "module1")
	require.Len(t, gs, 1)
	require.NotContains(t, gs, "module1")
	require.Contains(t, gs, "module2")
}

func TestGenesisStateComplexJSON(t *testing.T) {
	gs := make(app.GenesisState)

	complexJSON := json.RawMessage(`{
		"params": {
			"key1": "value1",
			"key2": 42
		},
		"data": [1, 2, 3],
		"nested": {
			"deep": {
				"value": true
			}
		}
	}`)

	gs["complex_module"] = complexJSON

	// Marshal and unmarshal
	data, err := json.Marshal(gs)
	require.NoError(t, err)

	var decoded app.GenesisState
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	require.JSONEq(t, string(complexJSON), string(decoded["complex_module"]))
}

func TestGenesisStateIteration(t *testing.T) {
	gs := make(app.GenesisState)
	gs["module1"] = json.RawMessage(`{}`)
	gs["module2"] = json.RawMessage(`{}`)
	gs["module3"] = json.RawMessage(`{}`)

	count := 0
	for key := range gs {
		require.NotEmpty(t, key)
		count++
	}

	require.Equal(t, 3, count)
}
