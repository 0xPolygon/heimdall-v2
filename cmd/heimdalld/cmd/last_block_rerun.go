package heimdalld

import (
	"fmt"
	"path/filepath"

	"cosmossdk.io/store/rootmulti"
	cmtcmd "github.com/cometbft/cometbft/cmd/cometbft/commands"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/spf13/cobra"

	"github.com/0xPolygon/heimdall-v2/app"
)

// wrapStartWithLastBlockRerun runs the last-block re-run check before the start command opens
// any database. See app/last_block_rerun.go for the mechanism.
func wrapStartWithLastBlockRerun(startCmd *cobra.Command) {
	prev := startCmd.PreRunE
	startCmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if prev != nil {
			if err := prev(cmd, args); err != nil {
				return err
			}
		}
		return prepareLastBlockRerun(server.GetServerContextFromCmd(cmd))
	}
}

// prepareLastBlockRerun decides whether the last height must be re-executed by this binary and,
// if so, rolls CometBFT state back one height (soft: the block stays in the block store) and
// records a pending re-run for the app. It never blocks the start: when the re-run is not
// possible, it logs why and the node starts as before.
func prepareLastBlockRerun(svrCtx *server.Context) error {
	logger := svrCtx.Logger.With("module", "last-block-rerun")
	cfg := svrCtx.Config

	db, err := dbm.NewDB("application", server.GetAppDBBackend(svrCtx.Viper), filepath.Join(cfg.RootDir, "data"))
	if err != nil {
		return fmt.Errorf("last-block re-run: open application db: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error("failed to close application db", "error", err)
		}
	}()

	appHeight := rootmulti.GetLatestVersion(db)

	committedBy, err := app.ReadLastCommitBinary(db)
	if err != nil {
		return fmt.Errorf("last-block re-run: read marker: %w", err)
	}

	pending, hasPending, err := app.ReadPendingRerun(db)
	if err != nil {
		return fmt.Errorf("last-block re-run: read pending re-run: %w", err)
	}

	var pendingPtr *app.PendingRerun
	if hasPending {
		pendingPtr = &pending
	}

	decision := app.DecideLastBlockRerun(appHeight, committedBy, app.BinaryIdentity(), pendingPtr)
	if !decision.Rerun {
		logger.Debug("no re-run needed", "reason", decision.Reason, "app_height", appHeight)
		return nil
	}

	logger.Info("re-running the last block with this binary",
		"reason", decision.Reason,
		"app_height", appHeight,
		"target", decision.TargetHeight,
		"binary", app.BinaryIdentity(),
	)

	// Soft rollback: CometBFT state goes to H-1 and block H stays in the block store, so the
	// handshake replays H against the real app. Idempotent: when state is already at H-1 it
	// returns H-1 without changes.
	height, _, err := cmtcmd.RollbackState(cfg, false)
	if err != nil {
		logger.Error("cannot re-run the last block, starting without it", "error", err)
		return nil
	}
	if height != decision.TargetHeight {
		logger.Error("cannot re-run the last block: CometBFT state does not match the app height, starting without it",
			"cometbft_height", height, "target", decision.TargetHeight)
		return nil
	}

	next := app.PendingRerun{TargetHeight: decision.TargetHeight, Attempts: 1}
	if hasPending {
		next.Attempts = pending.Attempts + 1
	}
	if err := app.WritePendingRerun(db, next); err != nil {
		return fmt.Errorf("last-block re-run: write pending re-run: %w", err)
	}

	logger.Info("CometBFT state rolled back one height; block will be re-executed during the handshake",
		"target", next.TargetHeight, "attempt", next.Attempts)

	return nil
}
