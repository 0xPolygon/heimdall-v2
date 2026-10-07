package hex_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/common/hex"
)

func TestFormatHex(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "empty bytes",
			input:    []byte{},
			expected: "0x",
		},
		{
			name:     "single byte",
			input:    []byte{0x0a},
			expected: "0x0a",
		},
		{
			name:     "multiple bytes",
			input:    []byte{0xde, 0xad, 0xbe, 0xef},
			expected: "0xdeadbeef",
		},
		{
			name:     "32-byte hash",
			input:    []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20},
			expected: "0x0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20",
		},
		{
			name:     "zeros",
			input:    []byte{0x00, 0x00, 0x00, 0x00},
			expected: "0x00000000",
		},
		{
			name:     "all 0xff",
			input:    []byte{0xff, 0xff, 0xff, 0xff},
			expected: "0xffffffff",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hex.FormatHex(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatAddress(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "address with 0x prefix",
			input:    "0x1234567890abcdef1234567890abcdef12345678",
			expected: "0x1234567890abcdef1234567890abcdef12345678",
		},
		{
			name:     "address without 0x prefix",
			input:    "1234567890abcdef1234567890abcdef12345678",
			expected: "0x1234567890abcdef1234567890abcdef12345678",
		},
		{
			name:     "uppercase address with 0x",
			input:    "0x1234567890ABCDEF1234567890ABCDEF12345678",
			expected: "0x1234567890abcdef1234567890abcdef12345678",
		},
		{
			name:     "uppercase address without 0x",
			input:    "1234567890ABCDEF1234567890ABCDEF12345678",
			expected: "0x1234567890abcdef1234567890abcdef12345678",
		},
		{
			name:     "mixed case address",
			input:    "0x1234567890AbCdEf1234567890aBcDeF12345678",
			expected: "0x1234567890abcdef1234567890abcdef12345678",
		},
		{
			name:     "address with leading spaces",
			input:    "  0x1234567890abcdef1234567890abcdef12345678",
			expected: "0x1234567890abcdef1234567890abcdef12345678",
		},
		{
			name:     "address with trailing spaces",
			input:    "0x1234567890abcdef1234567890abcdef12345678  ",
			expected: "0x1234567890abcdef1234567890abcdef12345678",
		},
		{
			name:     "address with leading and trailing spaces",
			input:    "  0x1234567890abcdef1234567890abcdef12345678  ",
			expected: "0x1234567890abcdef1234567890abcdef12345678",
		},
		{
			name:     "address with tabs",
			input:    "\t0x1234567890abcdef1234567890abcdef12345678\t",
			expected: "0x1234567890abcdef1234567890abcdef12345678",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "0x",
		},
		{
			name:     "just 0x",
			input:    "0x",
			expected: "0x",
		},
		{
			name:     "short address",
			input:    "0x123",
			expected: "0x123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hex.FormatAddress(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestIsTxHashNonEmpty(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "non-empty tx hash",
			input:    "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			expected: true,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "whitespace only",
			input:    "   ",
			expected: false,
		},
		{
			name:     "tab only",
			input:    "\t",
			expected: false,
		},
		{
			name:     "newline only",
			input:    "\n",
			expected: false,
		},
		{
			name:     "mixed whitespace",
			input:    " \t\n ",
			expected: false,
		},
		{
			name:     "single character",
			input:    "a",
			expected: true,
		},
		{
			name:     "string with leading whitespace",
			input:    "  0x123",
			expected: true,
		},
		{
			name:     "string with trailing whitespace",
			input:    "0x123  ",
			expected: true,
		},
		{
			name:     "string with spaces in middle",
			input:    "0x 123",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hex.IsTxHashNonEmpty(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateProof(t *testing.T) {
	tests := []struct {
		name        string
		proof       string
		expectError bool
		errContains string
	}{
		{
			name:        "valid proof - single 32-byte chunk",
			proof:       "0x" + strings.Repeat("a", 64), // 32 bytes
			expectError: false,
		},
		{
			name:        "valid proof - two 32-byte chunks",
			proof:       "0x" + strings.Repeat("a", 128), // 64 bytes
			expectError: false,
		},
		{
			name:        "valid proof - max size (1024 bytes)",
			proof:       "0x" + strings.Repeat("a", 2048), // 1024 bytes
			expectError: false,
		},
		{
			name:        "valid proof without 0x prefix",
			proof:       strings.Repeat("a", 64), // 32 bytes
			expectError: false,
		},
		{
			name:        "empty proof",
			proof:       "",
			expectError: true,
			errContains: "proof is empty or invalid hex",
		},
		{
			name:        "empty proof with 0x",
			proof:       "0x",
			expectError: true,
			errContains: "proof is empty or invalid hex",
		},
		{
			name:        "invalid hex characters",
			proof:       "0xzzzz",
			expectError: true,
			errContains: "proof is empty or invalid hex",
		},
		{
			name:        "proof not multiple of 32 bytes - 1 byte",
			proof:       "0xaa",
			expectError: true,
			errContains: "proof length must be a multiple of 32 bytes",
		},
		{
			name:        "proof not multiple of 32 bytes - 31 bytes",
			proof:       "0x" + strings.Repeat("a", 62),
			expectError: true,
			errContains: "proof length must be a multiple of 32 bytes",
		},
		{
			name:        "proof not multiple of 32 bytes - 33 bytes",
			proof:       "0x" + strings.Repeat("a", 66),
			expectError: true,
			errContains: "proof length must be a multiple of 32 bytes",
		},
		{
			name:        "proof exceeds max size - 1056 bytes (33*32)",
			proof:       "0x" + strings.Repeat("a", 2112), // 1056 bytes (33 * 32)
			expectError: true,
			errContains: "proof exceeds maximum allowed size",
		},
		{
			name:        "proof exceeds max size - 2048 bytes",
			proof:       "0x" + strings.Repeat("a", 4096), // 2048 bytes
			expectError: true,
			errContains: "proof exceeds maximum allowed size",
		},
		{
			name:        "valid proof - three 32-byte chunks",
			proof:       "0x" + strings.Repeat("deadbeef", 24), // 96 bytes (3*32)
			expectError: false,
		},
		{
			name:        "valid proof - uppercase hex",
			proof:       "0x" + strings.Repeat("AAAA", 16), // 32 bytes
			expectError: false,
		},
		{
			name:        "valid proof - mixed case hex",
			proof:       "0x" + strings.Repeat("AaBb", 16), // 32 bytes
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := hex.ValidateProof(tt.proof)
			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMaxProofLength(t *testing.T) {
	// Verify the MaxProofLength constant is set correctly
	require.Equal(t, 1024, hex.MaxProofLength)
}
