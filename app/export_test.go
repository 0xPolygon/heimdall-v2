package app_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/app"
)

func TestExportAppStateAndValidators_BasicExport(t *testing.T) {
	setupResult := app.SetupApp(t, 1)
	hApp := setupResult.App

	exported, err := hApp.ExportAppStateAndValidators(false, []string{}, []string{})

	require.NoError(t, err)
	require.NotNil(t, exported)
	require.NotEmpty(t, exported.AppState)
	require.Greater(t, exported.Height, int64(0))
	require.NotNil(t, exported.ConsensusParams)
	require.Equal(t, int64(1), exported.ConsensusParams.Abci.VoteExtensionsEnableHeight)
}

func TestExportAppStateAndValidators_WithModules(t *testing.T) {
	setupResult := app.SetupApp(t, 1)
	hApp := setupResult.App

	// Export specific modules
	modulesToExport := []string{"auth", "bank"}
	exported, err := hApp.ExportAppStateAndValidators(false, []string{}, modulesToExport)

	require.NoError(t, err)
	require.NotNil(t, exported)
	require.NotEmpty(t, exported.AppState)
	require.Greater(t, exported.Height, int64(0))
}

func TestExportAppStateAndValidators_EmptyModulesList(t *testing.T) {
	setupResult := app.SetupApp(t, 1)
	hApp := setupResult.App

	// Export with an empty modules list (should export all)
	exported, err := hApp.ExportAppStateAndValidators(false, []string{}, []string{})

	require.NoError(t, err)
	require.NotNil(t, exported)
	require.NotEmpty(t, exported.AppState)
}

func TestExportAppStateAndValidators_HeightIncrement(t *testing.T) {
	setupResult := app.SetupApp(t, 1)
	hApp := setupResult.App

	lastBlockHeight := hApp.LastBlockHeight()
	exported, err := hApp.ExportAppStateAndValidators(false, []string{}, []string{})

	require.NoError(t, err)
	// Exported height should be last block height + 1
	require.Equal(t, lastBlockHeight+1, exported.Height)
}

func TestExportAppStateAndValidators_ValidatorsNotNil(t *testing.T) {
	setupResult := app.SetupApp(t, 1)
	hApp := setupResult.App

	exported, err := hApp.ExportAppStateAndValidators(false, []string{}, []string{})

	require.NoError(t, err)
	require.NotNil(t, exported.Validators)
}

func TestExportAppStateAndValidators_AppStateIsJSON(t *testing.T) {
	setupResult := app.SetupApp(t, 1)
	hApp := setupResult.App

	exported, err := hApp.ExportAppStateAndValidators(false, []string{}, []string{})

	require.NoError(t, err)
	require.NotEmpty(t, exported.AppState)

	// Verify it's a valid JSON by checking first and last characters
	require.Equal(t, byte('{'), exported.AppState[0])
	// Find the last non-whitespace character
	lastChar := exported.AppState[len(exported.AppState)-1]
	for lastChar == '\n' || lastChar == ' ' || lastChar == '\t' {
		exported.AppState = exported.AppState[:len(exported.AppState)-1]
		lastChar = exported.AppState[len(exported.AppState)-1]
	}
	require.Equal(t, byte('}'), lastChar)
}

func TestExportAppStateAndValidators_ConsensusParamsNotNil(t *testing.T) {
	setupResult := app.SetupApp(t, 1)
	hApp := setupResult.App

	exported, err := hApp.ExportAppStateAndValidators(false, []string{}, []string{})

	require.NoError(t, err)
	require.NotNil(t, exported.ConsensusParams)
}

func TestExportAppStateAndValidators_MultipleValidators(t *testing.T) {
	setupResult := app.SetupApp(t, 3)
	hApp := setupResult.App

	exported, err := hApp.ExportAppStateAndValidators(false, []string{}, []string{})

	require.NoError(t, err)
	require.NotNil(t, exported.Validators)
	require.Equal(t, len(exported.Validators), 3)
}

func TestExportAppStateAndValidators_ConsecutiveExports(t *testing.T) {
	setupResult := app.SetupApp(t, 1)
	hApp := setupResult.App

	// First export
	exported1, err1 := hApp.ExportAppStateAndValidators(false, []string{}, []string{})
	require.NoError(t, err1)

	// Second export
	exported2, err2 := hApp.ExportAppStateAndValidators(false, []string{}, []string{})
	require.NoError(t, err2)

	// Both exports should succeed
	require.NotNil(t, exported1)
	require.NotNil(t, exported2)

	// Heights should be the same
	require.Equal(t, exported1.Height, exported2.Height)
}

func TestExportAppStateAndValidators_AfterBlockCommit(t *testing.T) {
	setupResult := app.SetupApp(t, 1)
	hApp := setupResult.App

	initialHeight := hApp.LastBlockHeight()

	exported, err := hApp.ExportAppStateAndValidators(false, []string{}, []string{})

	require.NoError(t, err)
	require.Equal(t, initialHeight+1, exported.Height)
}

func TestExportAppStateAndValidators_CheckExportedFields(t *testing.T) {
	setupResult := app.SetupApp(t, 1)
	hApp := setupResult.App

	exported, err := hApp.ExportAppStateAndValidators(false, []string{}, []string{})

	require.NoError(t, err)

	// Verify all fields are populated
	require.NotNil(t, exported.AppState, "AppState should not be nil")
	require.Greater(t, exported.Height, int64(0), "Height should be greater than 0")
	require.NotNil(t, exported.Validators, "Validators should not be nil")
	require.NotNil(t, exported.ConsensusParams, "ConsensusParams should not be nil")
}

func TestExportAppStateAndValidators_SpecificModuleExport(t *testing.T) {
	setupResult := app.SetupApp(t, 1)
	hApp := setupResult.App

	// Test exporting specific modules
	modules := [][]string{
		{"stake"},
		{"checkpoint"},
		{"bor"},
		{"auth", "bank"},
	}

	for _, modList := range modules {
		t.Run("export_"+modList[0], func(t *testing.T) {
			exported, err := hApp.ExportAppStateAndValidators(false, []string{}, modList)

			// Should succeed even if the module doesn't exist
			require.NoError(t, err)
			require.NotNil(t, exported)
		})
	}
}

func TestExportAppStateAndValidators_ValidJSONStructure(t *testing.T) {
	setupResult := app.SetupApp(t, 1)
	hApp := setupResult.App

	exported, err := hApp.ExportAppStateAndValidators(false, []string{}, []string{})

	require.NoError(t, err)
	require.NotEmpty(t, exported.AppState)

	// Basic JSON structure validation
	appStateStr := string(exported.AppState)
	require.Contains(t, appStateStr, "{")
	require.Contains(t, appStateStr, "}")
}
