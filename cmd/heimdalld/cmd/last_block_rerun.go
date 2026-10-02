package heimdalld

import (
	"fmt"
	"path/filepath"

	"cosmossdk.io/log"
	"cosmossdk.io/store/rootmulti"
	cmtcmd "github.com/cometbft/cometbft/cmd/cometbft/commands"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/spf13/cobra"

	"github.com/0xPolygon/heimdall-v2/app"
)

// rollbackCometState is replaced in tests.
var rollbackCometState = cmtcmd.RollbackState

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
// if so, rolls CometBFT state back one height and records a pending re-run for the app. It never
// blocks the start: when the re-run is not possible, it logs why and the node starts as before.
func prepareLastBlockRerun(svrCtx *server.Context) error {
	logger := svrCtx.Logger.With("module", "last-block-rerun")

	db, err := dbm.NewDB("application", server.GetAppDBBackend(svrCtx.Viper), filepath.Join(svrCtx.Config.RootDir, "data"))
	if err != nil {
		return fmt.Errorf("last-block re-run: open application db: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error("failed to close application db", "error", err)
		}
	}()

	appHeight := rootmulti.GetLatestVersion(db)
	pendingRerun, hasPending, err := app.ReadPendingRerun(db)
	if err != nil {
		return fmt.Errorf("last-block re-run: read pending re-run: %w", err)
	}
	var pending *app.PendingRerun
	if hasPending {
		pending = &pendingRerun
	}
	committedBy, err := app.ReadLastCommitBinary(db)
	if err != nil {
		return fmt.Errorf("last-block re-run: read marker: %w", err)
	}

	decision := app.DecideLastBlockRerun(appHeight, committedBy, app.BinaryIdentity(), pending)
	if !decision.Rerun {
		logger.Debug("no re-run needed", "reason", decision.Reason, "app_height", appHeight)
		return nil
	}
	logger.Info("re-running the last block with this binary",
		"reason", decision.Reason, "app_height", appHeight, "target", decision.TargetHeight, "binary", app.BinaryIdentity())

	return rollBackOneHeight(svrCtx, logger, db, decision.TargetHeight, pending)
}

// rollBackOneHeight rolls CometBFT state back to target (soft: block target+1 stays in the block
// store, so the handshake replays it against the real app) and records the attempt. The CometBFT
// rollback is idempotent: when state is already at target it returns target without changes.
func rollBackOneHeight(svrCtx *server.Context, logger log.Logger, db dbm.DB, target int64, pending *app.PendingRerun) error {
	height, _, err := rollbackCometState(svrCtx.Config, false)
	if err != nil {
		logger.Error("cannot re-run the last block, starting without it", "error", err)
		return nil
	}
	if height != target {
		logger.Error("cannot re-run the last block: CometBFT state does not match the app height, starting without it",
			"cometbft_height", height, "target", target)
		return nil
	}

	next := app.PendingRerun{TargetHeight: target, Attempts: 1}
	if pending != nil {
		next.Attempts = pending.Attempts + 1
	}
	if err := app.WritePendingRerun(db, next); err != nil {
		return fmt.Errorf("last-block re-run: write pending re-run: %w", err)
	}

	logger.Info("CometBFT state rolled back one height; block will be re-executed during the handshake",
		"target", next.TargetHeight, "attempt", next.Attempts)
	return nil
}
