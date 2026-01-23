package file_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/file"
)

func TestRootify(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		root     string
		expected func(path, root string) string
	}{
		{
			name: "relative path with root",
			path: "config.toml",
			root: "/home/user/.heimdall",
			expected: func(path, root string) string {
				return filepath.Join(root, path)
			},
		},
		{
			name: "relative path with nested dirs",
			path: "config/app.toml",
			root: "/home/user/.heimdall",
			expected: func(path, root string) string {
				return filepath.Join(root, path)
			},
		},
		{
			name: "absolute path - should return unchanged (Unix)",
			path: "/etc/heimdall/config.toml",
			root: "/home/user/.heimdall",
			expected: func(path, root string) string {
				if runtime.GOOS == "windows" {
					// On Windows, this might not be considered absolute
					return filepath.Join(root, path)
				}
				return path
			},
		},
		{
			name: "empty path",
			path: "",
			root: "/home/user/.heimdall",
			expected: func(path, root string) string {
				return filepath.Join(root, path)
			},
		},
		{
			name: "dot path",
			path: ".",
			root: "/home/user/.heimdall",
			expected: func(path, root string) string {
				return filepath.Join(root, path)
			},
		},
		{
			name: "double dot path",
			path: "..",
			root: "/home/user/.heimdall",
			expected: func(path, root string) string {
				return filepath.Join(root, path)
			},
		},
		{
			name: "path with leading dot slash",
			path: "./config.toml",
			root: "/home/user/.heimdall",
			expected: func(path, root string) string {
				return filepath.Join(root, path)
			},
		},
		{
			name: "path with parent directory reference",
			path: "../other/config.toml",
			root: "/home/user/.heimdall",
			expected: func(path, root string) string {
				return filepath.Join(root, path)
			},
		},
		{
			name: "empty root",
			path: "config.toml",
			root: "",
			expected: func(path, root string) string {
				return filepath.Join(root, path)
			},
		},
		{
			name: "root with trailing slash",
			path: "config.toml",
			root: "/home/user/.heimdall/",
			expected: func(path, root string) string {
				return filepath.Join(root, path)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := file.Rootify(tt.path, tt.root)
			expected := tt.expected(tt.path, tt.root)
			require.Equal(t, expected, result)
		})
	}
}

func TestRootifyAbsolutePaths(t *testing.T) {
	// Test platform-specific absolute paths
	var tests []struct {
		name string
		path string
		root string
	}

	if runtime.GOOS == "windows" {
		tests = []struct {
			name string
			path string
			root string
		}{
			{
				name: "Windows absolute path C drive",
				path: "C:\\config\\app.toml",
				root: "D:\\heimdall",
			},
			{
				name: "Windows absolute path with forward slashes",
				path: "C:/config/app.toml",
				root: "D:/heimdall",
			},
		}
	} else {
		tests = []struct {
			name string
			path string
			root string
		}{
			{
				name: "Unix absolute path",
				path: "/etc/heimdall/config.toml",
				root: "/home/user/.heimdall",
			},
			{
				name: "Unix absolute path in home",
				path: "/home/user/config.toml",
				root: "/var/heimdall",
			},
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := file.Rootify(tt.path, tt.root)
			// Absolute paths should be returned unchanged
			require.Equal(t, tt.path, result)
		})
	}
}

func TestRootifyRelativePathsAlwaysJoined(t *testing.T) {
	// Verify that relative paths are always joined with the root
	tests := []struct {
		name string
		path string
		root string
	}{
		{
			name: "simple relative path",
			path: "file.txt",
			root: "/root",
		},
		{
			name: "nested relative path",
			path: "a/b/c/file.txt",
			root: "/root",
		},
		{
			name: "relative path with dots",
			path: "./a/../b/file.txt",
			root: "/root",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := file.Rootify(tt.path, tt.root)
			expected := filepath.Join(tt.root, tt.path)
			require.Equal(t, expected, result)

			// Verify the result contains the root
			require.Contains(t, result, filepath.FromSlash(tt.root))
		})
	}
}

func TestRootifyPreservesAbsoluteness(t *testing.T) {
	absolutePath := "/absolute/path/to/file"
	if runtime.GOOS == "windows" {
		absolutePath = "C:\\absolute\\path\\to\\file"
	}

	root := "/some/root"
	result := file.Rootify(absolutePath, root)

	// Result should be absolute
	require.True(t, filepath.IsAbs(result))

	// Result should equal the original absolute path
	require.Equal(t, absolutePath, result)
}
