package helper

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
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
	origSpan := GetSpanRotationOnStallHeight()
	t.Cleanup(func() {
		SetRioHeight(origRio)
		SetPhuketHardforkHeight(origPhuket)
		SetSpanRotationOnStallHeight(origSpan)
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
