package version_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/version"
)

func TestCmdExists(t *testing.T) {
	// Verify the command is not nil
	require.NotNil(t, version.Cmd)
	require.Equal(t, "version", version.Cmd.Use)
	require.NotEmpty(t, version.Cmd.Short)
}

func TestCmdFlags(t *testing.T) {
	// Verify flags are registered
	require.NotNil(t, version.Cmd.Flags())

	longFlag := version.Cmd.Flags().Lookup("long")
	require.NotNil(t, longFlag)
	require.Equal(t, "l", longFlag.Shorthand)

	outputFlag := version.Cmd.Flags().Lookup("output")
	require.NotNil(t, outputFlag)
	require.Equal(t, "o", outputFlag.Shorthand)
	require.Equal(t, "text", outputFlag.DefValue)
}

func TestCmdShortFormat(t *testing.T) {
	// Just verify the command executes without error
	// Note: Output goes to stdout, not easily captured in tests
	cmd := version.Cmd
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	require.NoError(t, err)
}

func TestCmdShortFormatJSON(t *testing.T) {
	// Verify command executes successfully with JSON output
	cmd := version.Cmd
	cmd.SetArgs([]string{"--output", "json"})
	err := cmd.Execute()
	require.NoError(t, err)
}

func TestCmdLongFormatText(t *testing.T) {
	// Verify command executes successfully with the long format
	cmd := version.Cmd
	cmd.SetArgs([]string{"--long"})
	err := cmd.Execute()
	require.NoError(t, err)
}

func TestCmdLongFormatJSON(t *testing.T) {
	// Verify command executes successfully with long format and JSON
	cmd := version.Cmd
	cmd.SetArgs([]string{"--long", "--output", "json"})
	err := cmd.Execute()
	require.NoError(t, err)
}

func TestCmdShorthandFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "long flag shorthand",
			args: []string{"-l"},
		},
		{
			name: "output flag shorthand",
			args: []string{"-o", "json"},
		},
		{
			name: "both shorthands",
			args: []string{"-l", "-o", "json"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := version.Cmd
			cmd.SetArgs(tt.args)
			err := cmd.Execute()
			require.NoError(t, err)
		})
	}
}

func TestCmdInvalidOutputFormat(t *testing.T) {
	// Note: The command doesn't validate the output format,
	// invalid formats default to text/YAML
	cmd := version.Cmd
	cmd.SetArgs([]string{"--output", "invalid"})
	err := cmd.Execute()
	// Should not error, just defaults to YAML/text
	require.NoError(t, err)
}

func TestCmdJSONFormatting(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "short JSON",
			args: []string{"--output", "json"},
		},
		{
			name: "long JSON",
			args: []string{"--long", "--output", "json"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := version.Cmd
			cmd.SetArgs(tt.args)
			err := cmd.Execute()
			require.NoError(t, err)
		})
	}
}

func TestCmdYAMLFormatting(t *testing.T) {
	// Verify YAML format executes successfully
	cmd := version.Cmd
	cmd.SetArgs([]string{"--long"})
	err := cmd.Execute()
	require.NoError(t, err)
}

func TestCmdRunE(t *testing.T) {
	// Verify the command has a RunE function
	require.NotNil(t, version.Cmd.RunE)
}

func TestCmdNoArgs(t *testing.T) {
	// Verify the command works with no arguments
	cmd := version.Cmd
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	require.NoError(t, err)
}

func TestVersionInfoStructure(t *testing.T) {
	// Test that version.Info has all expected fields
	info := version.NewInfo()

	// Verify we can access all fields without panicking
	_ = info.Name
	_ = info.ServerName
	_ = info.ClientName
	_ = info.Version
	_ = info.GitCommit
	_ = info.GoVersion

	// Verify the go version is populated
	require.NotEmpty(t, info.GoVersion)
	require.Contains(t, info.GoVersion, "go version")
}
