package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
)

func TestNewGenesisState(t *testing.T) {
	t.Parallel()

	t.Run("creates genesis state with default params", func(t *testing.T) {
		t.Parallel()

		params := types.DefaultParams()
		gs := types.NewGenesisState(params)

		require.NotNil(t, gs)
		require.Equal(t, params, gs.Params)
	})

	t.Run("creates genesis state with custom params", func(t *testing.T) {
		t.Parallel()

		customParams := types.Params{
			MaxMilestonePropositionLength: 20,
			FfMilestoneThreshold:          2000,
			FfMilestoneBlockInterval:      200,
		}
		gs := types.NewGenesisState(customParams)

		require.NotNil(t, gs)
		require.Equal(t, customParams, gs.Params)
		require.Equal(t, uint64(20), gs.Params.MaxMilestonePropositionLength)
		require.Equal(t, uint64(2000), gs.Params.FfMilestoneThreshold)
		require.Equal(t, uint64(200), gs.Params.FfMilestoneBlockInterval)
	})
}

func TestDefaultGenesisState(t *testing.T) {
	t.Parallel()

	t.Run("returns default genesis state", func(t *testing.T) {
		t.Parallel()

		gs := types.DefaultGenesisState()

		require.NotNil(t, gs)
		require.Equal(t, types.DefaultParams(), gs.Params)
	})

	t.Run("default genesis state is valid", func(t *testing.T) {
		t.Parallel()

		gs := types.DefaultGenesisState()

		err := gs.Validate()
		require.NoError(t, err)
	})

	t.Run("default genesis state has expected values", func(t *testing.T) {
		t.Parallel()

		gs := types.DefaultGenesisState()

		require.Equal(t, uint64(10), gs.Params.MaxMilestonePropositionLength)
		require.Equal(t, uint64(1000), gs.Params.FfMilestoneThreshold)
		require.Equal(t, uint64(100), gs.Params.FfMilestoneBlockInterval)
	})
}

func TestGenesisState_Validate(t *testing.T) {
	t.Parallel()

	t.Run("validates correct genesis state", func(t *testing.T) {
		t.Parallel()

		gs := types.GenesisState{
			Params: types.DefaultParams(),
		}

		err := gs.Validate()
		require.NoError(t, err)
	})

	t.Run("rejects genesis with zero max proposition length", func(t *testing.T) {
		t.Parallel()

		gs := types.GenesisState{
			Params: types.Params{
				MaxMilestonePropositionLength: 0,
				FfMilestoneThreshold:          1000,
				FfMilestoneBlockInterval:      100,
			},
		}

		err := gs.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "max milestone proposition length")
	})

	t.Run("rejects genesis with zero threshold", func(t *testing.T) {
		t.Parallel()

		gs := types.GenesisState{
			Params: types.Params{
				MaxMilestonePropositionLength: 10,
				FfMilestoneThreshold:          0,
				FfMilestoneBlockInterval:      100,
			},
		}

		err := gs.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "ff milestone threshold")
	})

	t.Run("rejects genesis with zero block interval", func(t *testing.T) {
		t.Parallel()

		gs := types.GenesisState{
			Params: types.Params{
				MaxMilestonePropositionLength: 10,
				FfMilestoneThreshold:          1000,
				FfMilestoneBlockInterval:      0,
			},
		}

		err := gs.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "ff milestone block interval")
	})

	t.Run("rejects genesis with block interval >= threshold", func(t *testing.T) {
		t.Parallel()

		gs := types.GenesisState{
			Params: types.Params{
				MaxMilestonePropositionLength: 10,
				FfMilestoneThreshold:          100,
				FfMilestoneBlockInterval:      200,
			},
		}

		err := gs.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "block interval should be less than")
	})

	t.Run("rejects genesis with non-divisible threshold", func(t *testing.T) {
		t.Parallel()

		gs := types.GenesisState{
			Params: types.Params{
				MaxMilestonePropositionLength: 10,
				FfMilestoneThreshold:          1000,
				FfMilestoneBlockInterval:      99,
			},
		}

		err := gs.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "divisible")
	})
}
