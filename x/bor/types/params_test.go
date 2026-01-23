package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/x/bor/types"
)

func TestDefaultParams(t *testing.T) {
	t.Parallel()

	t.Run("returns default parameters", func(t *testing.T) {
		t.Parallel()

		params := types.DefaultParams()

		require.Equal(t, types.DefaultSprintDuration, params.SprintDuration)
		require.Equal(t, types.DefaultSpanDuration, params.SpanDuration)
		require.Equal(t, types.DefaultProducerCount, params.ProducerCount)
	})

	t.Run("default parameters are valid", func(t *testing.T) {
		t.Parallel()

		params := types.DefaultParams()

		err := params.ValidateBasic()
		require.NoError(t, err)
	})

	t.Run("default sprint duration is 16", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, uint64(16), types.DefaultSprintDuration)
	})

	t.Run("default span duration is 400 * sprint duration", func(t *testing.T) {
		t.Parallel()

		expected := 400 * types.DefaultSprintDuration
		require.Equal(t, expected, types.DefaultSpanDuration)
		require.Equal(t, uint64(6400), types.DefaultSpanDuration)
	})

	t.Run("default producer count is 4", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, uint64(4), types.DefaultProducerCount)
	})
}

func TestParams_ValidateBasic(t *testing.T) {
	t.Parallel()

	t.Run("validates correct parameters", func(t *testing.T) {
		t.Parallel()

		params := types.Params{
			SprintDuration: 16,
			SpanDuration:   6400,
			ProducerCount:  4,
		}

		err := params.ValidateBasic()
		require.NoError(t, err)
	})

	t.Run("accepts custom valid parameters", func(t *testing.T) {
		t.Parallel()

		params := types.Params{
			SprintDuration: 32,
			SpanDuration:   12800,
			ProducerCount:  8,
		}

		err := params.ValidateBasic()
		require.NoError(t, err)
	})

	t.Run("rejects zero sprint duration", func(t *testing.T) {
		t.Parallel()

		params := types.Params{
			SprintDuration: 0,
			SpanDuration:   6400,
			ProducerCount:  4,
		}

		err := params.ValidateBasic()
		require.Error(t, err)
		require.Contains(t, err.Error(), "sprint duration")
		require.Contains(t, err.Error(), "invalid value")
	})

	t.Run("rejects zero span duration", func(t *testing.T) {
		t.Parallel()

		params := types.Params{
			SprintDuration: 16,
			SpanDuration:   0,
			ProducerCount:  4,
		}

		err := params.ValidateBasic()
		require.Error(t, err)
		require.Contains(t, err.Error(), "span duration")
		require.Contains(t, err.Error(), "invalid value")
	})

	t.Run("rejects zero producer count", func(t *testing.T) {
		t.Parallel()

		params := types.Params{
			SprintDuration: 16,
			SpanDuration:   6400,
			ProducerCount:  0,
		}

		err := params.ValidateBasic()
		require.Error(t, err)
		require.Contains(t, err.Error(), "producer count")
		require.Contains(t, err.Error(), "invalid value")
	})

	t.Run("accepts very large values", func(t *testing.T) {
		t.Parallel()

		params := types.Params{
			SprintDuration: 1000000,
			SpanDuration:   10000000,
			ProducerCount:  1000,
		}

		err := params.ValidateBasic()
		require.NoError(t, err)
	})

	t.Run("accepts minimum valid values (1)", func(t *testing.T) {
		t.Parallel()

		params := types.Params{
			SprintDuration: 1,
			SpanDuration:   1,
			ProducerCount:  1,
		}

		err := params.ValidateBasic()
		require.NoError(t, err)
	})
}

func TestValidatePositiveIntForParam(t *testing.T) {
	t.Parallel()

	// This tests the validatePositiveIntForParam function indirectly through ValidateBasic

	t.Run("validates all parameters are checked", func(t *testing.T) {
		t.Parallel()

		// Each parameter should be validated
		tests := []struct {
			name   string
			params types.Params
			errMsg string
		}{
			{
				name: "invalid sprint duration",
				params: types.Params{
					SprintDuration: 0,
					SpanDuration:   100,
					ProducerCount:  1,
				},
				errMsg: "sprint duration",
			},
			{
				name: "invalid span duration",
				params: types.Params{
					SprintDuration: 16,
					SpanDuration:   0,
					ProducerCount:  1,
				},
				errMsg: "span duration",
			},
			{
				name: "invalid producer count",
				params: types.Params{
					SprintDuration: 16,
					SpanDuration:   100,
					ProducerCount:  0,
				},
				errMsg: "producer count",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				err := tt.params.ValidateBasic()
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errMsg)
			})
		}
	})
}

func TestParamConstants(t *testing.T) {
	t.Parallel()

	t.Run("validates default first span duration", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, uint64(256), types.DefaultFirstSpanDuration)
		require.Less(t, types.DefaultFirstSpanDuration, types.DefaultSpanDuration)
	})

	t.Run("validates span duration calculation", func(t *testing.T) {
		t.Parallel()

		expected := 400 * types.DefaultSprintDuration
		require.Equal(t, expected, types.DefaultSpanDuration)
	})
}
