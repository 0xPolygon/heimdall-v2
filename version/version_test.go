package version_test

import (
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/version"
)

func TestNewInfo(t *testing.T) {
	// Get the current version info
	info := version.NewInfo()

	// Verify all fields are set
	require.NotNil(t, info)
	require.Equal(t, version.Name, info.Name)
	require.Equal(t, version.ServerName, info.ServerName)
	require.Equal(t, version.ClientName, info.ClientName)
	require.Equal(t, version.Version, info.Version)
	require.Equal(t, version.Commit, info.GitCommit)

	// Verify GoVersion contains expected runtime information
	require.Contains(t, info.GoVersion, "go version")
	require.Contains(t, info.GoVersion, runtime.Version())
	require.Contains(t, info.GoVersion, runtime.GOOS)
	require.Contains(t, info.GoVersion, runtime.GOARCH)
}

func TestInfoString(t *testing.T) {
	tests := []struct {
		name          string
		info          version.Info
		shouldContain []string
	}{
		{
			name: "full info",
			info: version.Info{
				Name:       "heimdall",
				ServerName: "heimdalld",
				ClientName: "heimdalld",
				Version:    "v1.0.0",
				GitCommit:  "abc123def456",
				GoVersion:  "go version go1.21.0 linux/amd64",
			},
			shouldContain: []string{
				"heimdall",
				"v1.0.0",
				"git commit: abc123def456",
				"go version go1.21.0 linux/amd64",
			},
		},
		{
			name: "empty version",
			info: version.Info{
				Name:       "heimdall",
				ServerName: "heimdalld",
				ClientName: "heimdalld",
				Version:    "",
				GitCommit:  "",
				GoVersion:  "go version go1.21.0 linux/amd64",
			},
			shouldContain: []string{
				"heimdall",
				"git commit:",
			},
		},
		{
			name: "with special characters",
			info: version.Info{
				Name:       "heimdall-test",
				ServerName: "heimdalld-test",
				ClientName: "heimdalld-test",
				Version:    "v2.0.0-alpha.1+build.123",
				GitCommit:  "1234567890abcdef",
				GoVersion:  "go version go1.22.0 darwin/arm64",
			},
			shouldContain: []string{
				"heimdall-test",
				"v2.0.0-alpha.1+build.123",
				"git commit: 1234567890abcdef",
				"go version go1.22.0 darwin/arm64",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.info.String()
			require.NotEmpty(t, result)

			for _, expected := range tt.shouldContain {
				require.Contains(t, result, expected)
			}
		})
	}
}

func TestInfoStringFormat(t *testing.T) {
	// Verify the string format matches the expected structure
	info := version.Info{
		Name:       "testapp",
		ServerName: "testappd",
		ClientName: "testappd",
		Version:    "v1.2.3",
		GitCommit:  "abcdef123456",
		GoVersion:  "go version go1.21.0 linux/amd64",
	}

	result := info.String()

	// Check format structure: name: version\ngit commit: hash\ngo version
	lines := strings.Split(result, "\n")
	require.GreaterOrEqual(t, len(lines), 3, "Should have at least 3 lines")

	// The first line should be "name: version"
	require.Contains(t, lines[0], "testapp:")
	require.Contains(t, lines[0], "v1.2.3")

	// The second line should be "git commit: hash"
	require.Contains(t, lines[1], "git commit:")
	require.Contains(t, lines[1], "abcdef123456")

	// The third line should be the go version
	require.Contains(t, lines[2], "go version")
}

func TestDefaultVersionVariables(t *testing.T) {
	// Test that default version variables are accessible
	// (they may be empty or set by build flags)
	t.Run("Name", func(t *testing.T) {
		// Name can be empty or set
		_ = version.Name
	})

	t.Run("ServerName", func(t *testing.T) {
		// ServerName should have a default value
		require.Equal(t, "heimdalld", version.ServerName)
	})

	t.Run("ClientName", func(t *testing.T) {
		// ClientName should have a default value
		require.Equal(t, "heimdalld", version.ClientName)
	})

	t.Run("Version", func(t *testing.T) {
		// Version can be empty or set by build flags
		_ = version.Version
	})

	t.Run("Commit", func(t *testing.T) {
		// Commit can be empty or set by build flags
		_ = version.Commit
	})
}

func TestInfoGoVersionFormat(t *testing.T) {
	info := version.NewInfo()

	// Verify GoVersion has the expected format
	require.NotEmpty(t, info.GoVersion)
	require.True(t, strings.HasPrefix(info.GoVersion, "go version "))

	// Should contain the version number
	require.Contains(t, info.GoVersion, runtime.Version())

	// Should contain OS
	require.Contains(t, info.GoVersion, runtime.GOOS)

	// Should contain architecture
	require.Contains(t, info.GoVersion, runtime.GOARCH)

	// Should have the format "go version <version> <os>/<arch>"
	parts := strings.Split(info.GoVersion, " ")
	require.GreaterOrEqual(t, len(parts), 4)
	require.Equal(t, "go", parts[0])
	require.Equal(t, "version", parts[1])
}

func TestInfoEmptyValues(t *testing.T) {
	// Test with all empty values
	info := version.Info{
		Name:       "",
		ServerName: "",
		ClientName: "",
		Version:    "",
		GitCommit:  "",
		GoVersion:  "",
	}

	// String() should still work without panicking
	result := info.String()
	require.NotNil(t, result)

	// Should contain the "git commit" label even if empty
	require.Contains(t, result, "git commit:")
}

func TestMultipleNewInfoCalls(t *testing.T) {
	// Test that multiple calls return consistent results
	info1 := version.NewInfo()
	info2 := version.NewInfo()

	require.Equal(t, info1.Name, info2.Name)
	require.Equal(t, info1.ServerName, info2.ServerName)
	require.Equal(t, info1.ClientName, info2.ClientName)
	require.Equal(t, info1.Version, info2.Version)
	require.Equal(t, info1.GitCommit, info2.GitCommit)
	require.Equal(t, info1.GoVersion, info2.GoVersion)
}
