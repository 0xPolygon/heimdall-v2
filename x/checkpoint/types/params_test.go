package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
)

func TestParams_ValidateBasic(t *testing.T) {
	t.Parallel()

	t.Run("validates default params", func(t *testing.T) {
		t.Parallel()

		params := types.DefaultParams()
		err := params.ValidateBasic()

		require.NoError(t, err)
	})

	t.Run("rejects zero max checkpoint length", func(t *testing.T) {
		t.Parallel()

		params := types.DefaultParams()
		params.MaxCheckpointLength = 0

		err := params.ValidateBasic()

		require.Error(t, err)
		require.Contains(t, err.Error(), "max checkpoint length should be non-zero")
	})

	t.Run("rejects zero avg checkpoint length", func(t *testing.T) {
		t.Parallel()

		params := types.DefaultParams()
		params.AvgCheckpointLength = 0

		err := params.ValidateBasic()

		require.Error(t, err)
		require.Contains(t, err.Error(), "avg checkpoint length should be non-zero")
	})

	t.Run("rejects avg greater than max", func(t *testing.T) {
		t.Parallel()

		params := types.DefaultParams()
		params.MaxCheckpointLength = 100
		params.AvgCheckpointLength = 200

		err := params.ValidateBasic()

		require.Error(t, err)
		require.Contains(t, err.Error(), "avg checkpoint length should not be greater than max checkpoint length")
	})

	t.Run("accepts avg equal to max", func(t *testing.T) {
		t.Parallel()

		params := types.DefaultParams()
		params.MaxCheckpointLength = 100
		params.AvgCheckpointLength = 100

		err := params.ValidateBasic()

		require.NoError(t, err)
	})

	t.Run("rejects zero child chain block interval", func(t *testing.T) {
		t.Parallel()

		params := types.DefaultParams()
		params.ChildChainBlockInterval = 0

		err := params.ValidateBasic()

		require.Error(t, err)
		require.Contains(t, err.Error(), "child chain block interval should be greater than zero")
	})

	t.Run("accepts valid custom params", func(t *testing.T) {
		t.Parallel()

		params := types.Params{
			MaxCheckpointLength:      1024,
			AvgCheckpointLength:      256,
			ChildChainBlockInterval:  10000,
		}

		err := params.ValidateBasic()

		require.NoError(t, err)
	})

	t.Run("accepts minimum valid values", func(t *testing.T) {
		t.Parallel()

		params := types.Params{
			MaxCheckpointLength:      1,
			AvgCheckpointLength:      1,
			ChildChainBlockInterval:  1,
		}

		err := params.ValidateBasic()

		require.NoError(t, err)
	})

	t.Run("accepts very large values", func(t *testing.T) {
		t.Parallel()

		params := types.Params{
			MaxCheckpointLength:      1000000,
			AvgCheckpointLength:      500000,
			ChildChainBlockInterval:  1000000,
		}

		err := params.ValidateBasic()

		require.NoError(t, err)
	})
}

func TestDefaultParams(t *testing.T) {
	t.Parallel()

	t.Run("returns valid default params", func(t *testing.T) {
		t.Parallel()

		params := types.DefaultParams()

		require.NotNil(t, params)
		require.NoError(t, params.ValidateBasic())
	})

	t.Run("default params have reasonable values", func(t *testing.T) {
		t.Parallel()

		params := types.DefaultParams()

		require.Greater(t, params.MaxCheckpointLength, uint64(0))
		require.Greater(t, params.AvgCheckpointLength, uint64(0))
		require.Greater(t, params.ChildChainBlockInterval, uint64(0))
		require.GreaterOrEqual(t, params.MaxCheckpointLength, params.AvgCheckpointLength)
	})
}
