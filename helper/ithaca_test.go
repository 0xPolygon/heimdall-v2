package helper

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

// TestIsIthaca verifies the POS-3629 hardfork gate: disabled when the height is
// zero (the default on every network), and active only at/after a configured height.
func TestIsIthaca(t *testing.T) {
	orig := GetIthacaHeight()
	t.Cleanup(func() { SetIthacaHeight(orig) })

	// Disabled by default: never active, regardless of height.
	SetIthacaHeight(0)
	require.False(t, IsIthaca(0))
	require.False(t, IsIthaca(1))
	require.False(t, IsIthaca(1_000_000))

	// Once set, active at and after the height, inactive before it.
	SetIthacaHeight(100)
	require.Equal(t, int64(100), GetIthacaHeight())
	require.False(t, IsIthaca(99))
	require.True(t, IsIthaca(100))
	require.True(t, IsIthaca(101))
}

// TestGetBorStallThreshold keeps the POS-3629 threshold alias honest: the stall
// threshold must exactly match the existing change-producer threshold helper.
func TestGetBorStallThreshold(t *testing.T) {
	ctx := sdk.Context{}.WithBlockHeight(1234)
	require.Equal(t, GetChangeProducerThreshold(ctx), GetBorStallThreshold(ctx))
}

// TestConfigAccessorsAndSetters covers the tiny helper accessors that were
// introduced alongside the POS-3629 gates so their branches stay covered even
// when the broader init path is not exercised.
func TestConfigAccessorsAndSetters(t *testing.T) {
	origRio := GetRioHeight()
	origPhuket := GetPhuketHardforkHeight()
	origSpan := GetIthacaHeight()
	t.Cleanup(func() {
		SetRioHeight(origRio)
		SetPhuketHardforkHeight(origPhuket)
		SetIthacaHeight(origSpan)
	})

	_ = GetTallyFixHeight()
	_ = GetDisableVPCheckHeight()
	_ = GetDisableValSetCheckHeight()
	_ = GetMilestoneDeletionHeight()
	_ = GetFaultyMilestoneNumber()
	_ = GetSetProducerDowntimeHeight()

	SetRioHeight(42)
	require.Equal(t, int64(42), GetRioHeight())
	require.True(t, IsRio(42))
	require.False(t, IsRio(41))

	SetPhuketHardforkHeight(77)
	require.Equal(t, int64(77), GetPhuketHardforkHeight())
	require.True(t, IsPhuketHardfork(77))
	require.False(t, IsPhuketHardfork(76))

	helperCtx := sdk.Context{}.WithBlockHeight(100)
	_ = GetProducerVotes()
	_ = GetFallbackProducerVotes()
	_ = GetProducerSetLimit(helperCtx)
	_ = GetChangeProducerThreshold(helperCtx)
	_ = GetSpanRotationBuffer(helperCtx)
	_ = GetBorStallThreshold(helperCtx)
}
