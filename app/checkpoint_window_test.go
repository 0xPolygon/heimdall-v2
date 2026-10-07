package app

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/helper"
	borTypes "github.com/0xPolygon/heimdall-v2/x/bor/types"
	checkpointTypes "github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
)

// A non-RP checkpoint vote extension carries a proposer-supplied [start,end] window.
// Below Lugano a window whose length doesn't match the params reaches the Bor RPC inside
// IsValidCheckpoint (here returning ErrBorBlockNotFound); at/after Lugano it is rejected
// before that call.
func TestValidateCheckpointMsgData_LengthGate(t *testing.T) {
	const (
		luganoHeight = int64(100)
		start        = uint64(1_000_000)
	)

	_, app, ctx, _ := SetupAppWithABCICtx(t)
	origKyoto, origLugano := helper.GetKyotoHeight(), helper.GetLuganoHeight()
	t.Cleanup(func() {
		helper.SetKyotoHeight(origKyoto)
		helper.SetLuganoHeight(origLugano)
	})
	helper.SetKyotoHeight(1)
	helper.SetLuganoHeight(luganoHeight)

	params, err := app.CheckpointKeeper.GetParams(ctx)
	require.NoError(t, err)

	// Seed a last checkpoint so the continuity guard accepts a window starting at start.
	require.NoError(t, app.CheckpointKeeper.AddCheckpoint(ctx, checkpointTypes.Checkpoint{
		Id:         1,
		StartBlock: 1,
		EndBlock:   start - 1,
		RootHash:   make([]byte, 32),
		Proposer:   "0x0000000000000000000000000000000000000001",
		BorChainId: "test",
	}))
	require.NoError(t, app.CheckpointKeeper.UpdateAckCountWithValue(ctx, 1))

	validators := app.StakeKeeper.GetAllValidators(ctx)
	caller := setupBorBlockNotFoundCaller()

	chainParams, err := app.ChainManagerKeeper.GetParams(ctx)
	require.NoError(t, err)

	// Distinct start/end values keep the package-level root-hash and block-existence caches
	// in x/checkpoint/types from returning another test's result here.
	packWindow := func(end uint64) []byte {
		msg := &checkpointTypes.MsgCheckpoint{
			Proposer:        validators[0].Signer,
			StartBlock:      start,
			EndBlock:        end,
			RootHash:        common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000dead"),
			AccountRootHash: common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000003dead"),
			BorChainId:      chainParams.ChainParams.BorChainId,
		}
		return packExtensionWithVote(msg.GetSideSignBytes())
	}

	// One block past an avg-length window, so the length check is the only thing wrong.
	misaligned := packWindow(start + params.AvgCheckpointLength)

	validate := func(height int64, ext []byte) error {
		return validateNonRpVoteExtensionData(ctx, height, ext, app.ChainManagerKeeper, app.CheckpointKeeper, caller)
	}

	t.Run("one height below activation the window reaches the bor RPC", func(t *testing.T) {
		require.ErrorIs(t, validate(luganoHeight-1, misaligned), borTypes.ErrBorBlockNotFound)
	})

	t.Run("at activation the window is rejected before the bor RPC", func(t *testing.T) {
		require.ErrorIs(t, validate(luganoHeight, misaligned), checkpointTypes.ErrInvalidCheckpointLength)
	})

	t.Run("one height above activation the window is still rejected", func(t *testing.T) {
		require.ErrorIs(t, validate(luganoHeight+1, misaligned), checkpointTypes.ErrInvalidCheckpointLength)
	})

	t.Run("with the fork disabled the window reaches the bor RPC", func(t *testing.T) {
		helper.SetLuganoHeight(0)
		t.Cleanup(func() { helper.SetLuganoHeight(luganoHeight) })

		require.ErrorIs(t, validate(luganoHeight+1, misaligned), borTypes.ErrBorBlockNotFound)
	})

	t.Run("an avg-length window still reaches the bor RPC at activation", func(t *testing.T) {
		require.ErrorIs(t, validate(luganoHeight, packWindow(start+params.AvgCheckpointLength-1)),
			borTypes.ErrBorBlockNotFound)
	})
}
