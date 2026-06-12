package helper

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestIsSpanRotationOnStall verifies the POS-3629 hardfork gate: disabled when the height is
// zero (the default on every network), and active only at/after a configured height.
func TestIsSpanRotationOnStall(t *testing.T) {
	orig := GetSpanRotationOnStallHeight()
	t.Cleanup(func() { SetSpanRotationOnStallHeight(orig) })

	// Disabled by default: never active, regardless of height.
	SetSpanRotationOnStallHeight(0)
	require.False(t, IsSpanRotationOnStall(0))
	require.False(t, IsSpanRotationOnStall(1))
	require.False(t, IsSpanRotationOnStall(1_000_000))

	// Once set, active at and after the height, inactive before it.
	SetSpanRotationOnStallHeight(100)
	require.Equal(t, int64(100), GetSpanRotationOnStallHeight())
	require.False(t, IsSpanRotationOnStall(99))
	require.True(t, IsSpanRotationOnStall(100))
	require.True(t, IsSpanRotationOnStall(101))
}
