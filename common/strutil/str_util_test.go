package strutil_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/common/strutil"
)

func TestHashesToString(t *testing.T) {
	tests := []struct {
		name     string
		input    [][]byte
		expected string
	}{
		{
			name:     "empty slice",
			input:    [][]byte{},
			expected: "",
		},
		{
			name:     "single hash",
			input:    [][]byte{{0xde, 0xad, 0xbe, 0xef}},
			expected: "0xdeadbeef",
		},
		{
			name: "multiple hashes",
			input: [][]byte{
				{0xde, 0xad, 0xbe, 0xef},
				{0xca, 0xfe, 0xba, 0xbe},
			},
			expected: "0xdeadbeef 0xcafebabe",
		},
		{
			name: "three hashes",
			input: [][]byte{
				{0x01, 0x02},
				{0x03, 0x04},
				{0x05, 0x06},
			},
			expected: "0x0102 0x0304 0x0506",
		},
		{
			name: "32-byte hashes",
			input: [][]byte{
				{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20},
				{0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29, 0x2a, 0x2b, 0x2c, 0x2d, 0x2e, 0x2f, 0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38, 0x39, 0x3a, 0x3b, 0x3c, 0x3d, 0x3e, 0x3f, 0x40},
			},
			expected: "0x0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20 0x2122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f40",
		},
		{
			name: "empty hash in slice",
			input: [][]byte{
				{},
			},
			expected: "0x",
		},
		{
			name: "multiple empty hashes",
			input: [][]byte{
				{},
				{},
			},
			expected: "0x 0x",
		},
		{
			name: "mixed empty and non-empty hashes",
			input: [][]byte{
				{0xaa},
				{},
				{0xbb},
			},
			expected: "0xaa 0x 0xbb",
		},
		{
			name: "single byte hashes",
			input: [][]byte{
				{0x00},
				{0x01},
				{0xff},
			},
			expected: "0x00 0x01 0xff",
		},
		{
			name: "all zeros",
			input: [][]byte{
				{0x00, 0x00},
				{0x00, 0x00, 0x00},
			},
			expected: "0x0000 0x000000",
		},
		{
			name: "all 0xff",
			input: [][]byte{
				{0xff, 0xff},
				{0xff, 0xff, 0xff},
			},
			expected: "0xffff 0xffffff",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := strutil.HashesToString(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestHashesToStringNilInput(t *testing.T) {
	// Test with nil slice - should not panic
	result := strutil.HashesToString(nil)
	require.Equal(t, "", result)
}

func TestHashesToStringLargeInput(t *testing.T) {
	// Test with a large number of hashes
	hashes := make([][]byte, 100)
	for i := 0; i < 100; i++ {
		hashes[i] = []byte{byte(i)}
	}
	result := strutil.HashesToString(hashes)

	// Should contain 100 hashes separated by 99 spaces
	// Each hash is "0x" + 2 hex chars
	require.NotEmpty(t, result)
	require.Contains(t, result, "0x00")
	require.Contains(t, result, "0x63") // byte 99
}
