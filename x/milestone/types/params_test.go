package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
)

func TestDefaultParams(t *testing.T) {
	t.Parallel()

	t.Run("returns non-nil default params", func(t *testing.T) {
		t.Parallel()

		params := types.DefaultParams()

		require.NotNil(t, params)
	})

	t.Run("default params have correct values", func(t *testing.T) {
		t.Parallel()

		params := types.DefaultParams()

		require.Equal(t, uint64(10), params.MaxMilestonePropositionLength)
		require.Equal(t, uint64(1000), params.FfMilestoneThreshold)
		require.Equal(t, uint64(100), params.FfMilestoneBlockInterval)
	})

	t.Run("default params are valid", func(t *testing.T) {
		t.Parallel()

		params := types.DefaultParams()

		err := params.ValidateBasic()
		require.NoError(t, err)
	})
}

func TestParams_ValidateBasic(t *testing.T) {
	t.Parallel()

	t.Run("validates correct params", func(t *testing.T) {
		t.Parallel()

		params := types.Params{
			MaxMilestonePropositionLength: 10,
			FfMilestoneThreshold:          1000,
			FfMilestoneBlockInterval:      100,
		}

		err := params.ValidateBasic()
		require.NoError(t, err)
	})

	t.Run("rejects zero max milestone proposition length", func(t *testing.T) {
		t.Parallel()

		params := types.Params{
			MaxMilestonePropositionLength: 0,
			FfMilestoneThreshold:          1000,
			FfMilestoneBlockInterval:      100,
		}

		err := params.ValidateBasic()
		require.Error(t, err)
		require.Contains(t, err.Error(), "max milestone proposition length should not be zero")
	})

	t.Run("rejects zero ff milestone threshold", func(t *testing.T) {
		t.Parallel()

		params := types.Params{
			MaxMilestonePropositionLength: 10,
			FfMilestoneThreshold:          0,
			FfMilestoneBlockInterval:      100,
		}

		err := params.ValidateBasic()
		require.Error(t, err)
		require.Contains(t, err.Error(), "ff milestone threshold should not be zero")
	})

	t.Run("rejects zero ff milestone block interval", func(t *testing.T) {
		t.Parallel()

		params := types.Params{
			MaxMilestonePropositionLength: 10,
			FfMilestoneThreshold:          1000,
			FfMilestoneBlockInterval:      0,
		}

		err := params.ValidateBasic()
		require.Error(t, err)
		require.Contains(t, err.Error(), "ff milestone block interval should not be zero")
	})

	t.Run("rejects block interval >= threshold", func(t *testing.T) {
		t.Parallel()

		params := types.Params{
			MaxMilestonePropositionLength: 10,
			FfMilestoneThreshold:          100,
			FfMilestoneBlockInterval:      100,
		}

		err := params.ValidateBasic()
		require.Error(t, err)
		require.Contains(t, err.Error(), "ff milestone block interval should be less than ff milestone threshold")
	})

	t.Run("rejects block interval > threshold", func(t *testing.T) {
		t.Parallel()

		params := types.Params{
			MaxMilestonePropositionLength: 10,
			FfMilestoneThreshold:          100,
			FfMilestoneBlockInterval:      200,
		}

		err := params.ValidateBasic()
		require.Error(t, err)
		require.Contains(t, err.Error(), "ff milestone block interval should be less than ff milestone threshold")
	})

	t.Run("rejects non-divisible threshold", func(t *testing.T) {
		t.Parallel()

		params := types.Params{
			MaxMilestonePropositionLength: 10,
			FfMilestoneThreshold:          1000,
			FfMilestoneBlockInterval:      99,
		}

		err := params.ValidateBasic()
		require.Error(t, err)
		require.Contains(t, err.Error(), "ff milestone threshold should be divisible by ff milestone block interval")
	})

	t.Run("accepts valid divisible values", func(t *testing.T) {
		t.Parallel()

		params := types.Params{
			MaxMilestonePropositionLength: 5,
			FfMilestoneThreshold:          500,
			FfMilestoneBlockInterval:      50,
		}

		err := params.ValidateBasic()
		require.NoError(t, err)
	})

	t.Run("accepts threshold exactly divisible by interval", func(t *testing.T) {
		t.Parallel()

		params := types.Params{
			MaxMilestonePropositionLength: 20,
			FfMilestoneThreshold:          2000,
			FfMilestoneBlockInterval:      200,
		}

		err := params.ValidateBasic()
		require.NoError(t, err)
	})
}
