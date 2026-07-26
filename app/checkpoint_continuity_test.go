package app

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/helper"
	borTypes "github.com/0xPolygon/heimdall-v2/x/bor/types"
	checkpointTypes "github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
)

// A non-RP checkpoint vote extension carries a proposer-supplied [start,end]
// window. Below Kyoto the window reaches the Bor RPC inside IsValidCheckpoint
// (here returning ErrBorBlockNotFound); at/after Kyoto a window that breaks
// checkpoint continuity is rejected by the cheap pre-RPC guard, so the expensive
// GetRootHash call is never made.
func TestValidateCheckpointMsgData_ContinuityGate(t *testing.T) {
	const height = int64(3)

	_, app, ctx, _ := SetupAppWithABCICtx(t)
	validators := app.StakeKeeper.GetAllValidators(ctx)
	caller := setupBorBlockNotFoundCaller()

	// endBlock-100 is not the continuation of any existing checkpoint in the fresh app.
	packedVE, _ := buildCheckpointNonRpVoteExt(t, app, ctx, validators[0].Signer, 7_654_321)

	t.Run("below kyoto reaches the bor RPC", func(t *testing.T) {
		helper.SetKyotoHeight(0)
		err := validateNonRpVoteExtensionData(ctx, height, packedVE, app.ChainManagerKeeper, app.CheckpointKeeper, caller)
		require.Error(t, err)
		require.True(t, errors.Is(err, borTypes.ErrBorBlockNotFound),
			"below the fork the window should reach IsValidCheckpoint's RPC")
	})

	t.Run("at/after kyoto is rejected before the bor RPC", func(t *testing.T) {
		helper.SetKyotoHeight(1)
		defer helper.SetKyotoHeight(0)
		err := validateNonRpVoteExtensionData(ctx, height, packedVE, app.ChainManagerKeeper, app.CheckpointKeeper, caller)
		require.Error(t, err)
		require.False(t, errors.Is(err, borTypes.ErrBorBlockNotFound),
			"the continuity guard must short-circuit before the RPC")
	})
}

// Direct unit test of the gated continuity guard across the no-checkpoint and
// seeded-checkpoint paths and the exact end-block+1 boundary.
func TestCheckpointWindowContinuity(t *testing.T) {
	_, app, ctx, _ := SetupAppWithABCICtx(t)
	const height = int64(10)

	t.Run("kyoto off lets any window pass", func(t *testing.T) {
		helper.SetKyotoHeight(0)
		require.NoError(t, checkpointWindowContinuity(ctx, height, app.CheckpointKeeper, 999))
	})

	helper.SetKyotoHeight(1)
	defer helper.SetKyotoHeight(0)

	t.Run("no prior checkpoint: start 0 is valid", func(t *testing.T) {
		require.NoError(t, checkpointWindowContinuity(ctx, height, app.CheckpointKeeper, 0))
	})
	t.Run("no prior checkpoint: non-zero start rejected", func(t *testing.T) {
		require.Error(t, checkpointWindowContinuity(ctx, height, app.CheckpointKeeper, 5))
	})

	// Seed a last checkpoint ending at block 100.
	require.NoError(t, app.CheckpointKeeper.AddCheckpoint(ctx, checkpointTypes.Checkpoint{
		Id:         1,
		StartBlock: 1,
		EndBlock:   100,
		RootHash:   make([]byte, 32),
		Proposer:   "0x0000000000000000000000000000000000000001",
		BorChainId: "test",
	}))
	require.NoError(t, app.CheckpointKeeper.UpdateAckCountWithValue(ctx, 1))

	t.Run("continuous window (end+1) passes", func(t *testing.T) {
		require.NoError(t, checkpointWindowContinuity(ctx, height, app.CheckpointKeeper, 101))
	})
	t.Run("non-continuous window rejected", func(t *testing.T) {
		require.Error(t, checkpointWindowContinuity(ctx, height, app.CheckpointKeeper, 200))
	})
}
