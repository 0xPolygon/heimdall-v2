package errors_test

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	typesErrors "github.com/0xPolygon/heimdall-v2/types/error"
)

func TestInvalidPermissionsErrorIsDetailed(t *testing.T) {
	tests := []struct {
		name     string
		err      typesErrors.InvalidPermissionsError
		expected bool
	}{
		{
			name: "detailed error with file and perm",
			err: typesErrors.InvalidPermissionsError{
				File: "/path/to/file",
				Perm: 0600,
			},
			expected: true,
		},
		{
			name: "not detailed - missing file",
			err: typesErrors.InvalidPermissionsError{
				File: "",
				Perm: 0600,
			},
			expected: false,
		},
		{
			name: "not detailed - missing perm",
			err: typesErrors.InvalidPermissionsError{
				File: "/path/to/file",
				Perm: 0,
			},
			expected: false,
		},
		{
			name: "not detailed - missing both",
			err: typesErrors.InvalidPermissionsError{
				File: "",
				Perm: 0,
			},
			expected: false,
		},
		{
			name: "detailed with 0644 perm",
			err: typesErrors.InvalidPermissionsError{
				File: "config.toml",
				Perm: 0644,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.IsDetailed()
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestInvalidPermissionsErrorError(t *testing.T) {
	tests := []struct {
		name             string
		err              typesErrors.InvalidPermissionsError
		shouldContain    []string
		shouldNotContain []string
	}{
		{
			name: "detailed error message",
			err: typesErrors.InvalidPermissionsError{
				File: "/path/to/file.txt",
				Perm: 0600,
			},
			shouldContain: []string{
				typesErrors.InvalidFilePermissionErrMsg,
				"for file",
				"/path/to/file.txt",
				"should be",
			},
		},
		{
			name: "error with wrapped error",
			err: typesErrors.InvalidPermissionsError{
				Err: errors.New("permission denied"),
			},
			shouldContain: []string{
				typesErrors.InvalidFilePermissionErrMsg,
				"err:",
				"permission denied",
			},
		},
		{
			name: "error with both details and wrapped error",
			err: typesErrors.InvalidPermissionsError{
				File: "/path/to/file.txt",
				Perm: 0600,
				Err:  errors.New("wrapped error"),
			},
			shouldContain: []string{
				typesErrors.InvalidFilePermissionErrMsg,
				"err:",
				"wrapped error",
			},
			shouldNotContain: []string{
				"for file",
			},
		},
		{
			name: "error with 0644 permission",
			err: typesErrors.InvalidPermissionsError{
				File: "config.yaml",
				Perm: 0644,
			},
			shouldContain: []string{
				"config.yaml",
				"should be",
			},
		},
		{
			name: "error with 0400 permission",
			err: typesErrors.InvalidPermissionsError{
				File: "private.key",
				Perm: 0400,
			},
			shouldContain: []string{
				"private.key",
				"should be",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errMsg := tt.err.Error()

			for _, expected := range tt.shouldContain {
				require.Contains(t, errMsg, expected)
			}

			for _, notExpected := range tt.shouldNotContain {
				require.NotContains(t, errMsg, notExpected)
			}
		})
	}
}

func TestInvalidPermissionsErrorConstant(t *testing.T) {
	require.Equal(t, "invalid file permission", typesErrors.InvalidFilePermissionErrMsg)
}

func TestInvalidPermissionsErrorFields(t *testing.T) {
	err := typesErrors.InvalidPermissionsError{
		File: "/test/file",
		Perm: 0755,
		Err:  errors.New("test error"),
	}

	require.Equal(t, "/test/file", err.File)
	require.Equal(t, os.FileMode(0755), err.Perm)
	require.NotNil(t, err.Err)
	require.Equal(t, "test error", err.Err.Error())
}

func TestInvalidPermissionsErrorEmptyMessage(t *testing.T) {
	// Test with no fields set
	err := typesErrors.InvalidPermissionsError{}
	errMsg := err.Error()

	// Should return an empty string or base message
	require.NotPanics(t, func() {
		_ = err.Error()
	})

	// IsDetailed should return false
	require.False(t, err.IsDetailed())

	// Empty error should not contain the wrapped error message
	require.NotContains(t, errMsg, "err:")
}

func TestInvalidPermissionsErrorPermissionFormats(t *testing.T) {
	tests := []struct {
		name string
		perm os.FileMode
		file string
	}{
		{
			name: "0600 permission",
			perm: 0600,
			file: "file1",
		},
		{
			name: "0644 permission",
			perm: 0644,
			file: "file2",
		},
		{
			name: "0755 permission",
			perm: 0755,
			file: "file3",
		},
		{
			name: "0400 permission",
			perm: 0400,
			file: "file4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := typesErrors.InvalidPermissionsError{
				File: tt.file,
				Perm: tt.perm,
			}

			errMsg := err.Error()
			require.Contains(t, errMsg, tt.file)
			require.Contains(t, errMsg, tt.perm.String())
		})
	}
}

func TestInvalidPermissionsErrorWithNilWrappedError(t *testing.T) {
	err := typesErrors.InvalidPermissionsError{
		File: "/path/to/file",
		Perm: 0600,
		Err:  nil,
	}

	errMsg := err.Error()
	require.Contains(t, errMsg, "/path/to/file")
	require.Contains(t, errMsg, "should be")
	require.NotContains(t, errMsg, "err:")
}

func TestInvalidPermissionsErrorPriority(t *testing.T) {
	// When both detailed and Err are set, Err takes priority in the message
	err := typesErrors.InvalidPermissionsError{
		File: "/path/to/file",
		Perm: 0600,
		Err:  errors.New("wrapped error"),
	}

	errMsg := err.Error()
	// Should contain a wrapped error message
	require.Contains(t, errMsg, "wrapped error")
	require.Contains(t, errMsg, "err:")

	// IsDetailed should still return true
	require.True(t, err.IsDetailed())
}
