package app

import (
	"fmt"

	"github.com/0xPolygon/heimdall-v2/app/rerun"
)

// rollbackStoresAndLoad rolls every store back to target and loads the app there. The rollback
// runs on the multistore first because BaseApp.LoadLatestVersion seals the app and may run once.
// A failure to load after the rollback is fatal: the stores have already changed.
func (app *HeimdallApp) rollbackStoresAndLoad(target int64) error {
	cms := app.CommitMultiStore()
	if err := cms.LoadLatestVersion(); err != nil {
		return err
	}
	if err := cms.RollbackToVersion(target); err != nil {
		return err
	}
	if err := app.LoadLatestVersion(); err != nil {
		panic(fmt.Errorf("last-block re-run: loading the app after the rollback to %d: %w", target, err))
	}
	return nil
}

// recordCommitByThisBinary runs after a successful commit, once per process.
func (app *HeimdallApp) recordCommitByThisBinary() {
	if app.rerunMarkerWritten || app.rerunDB == nil {
		return
	}
	if err := rerun.RecordCommit(app.rerunDB, rerun.BinaryIdentity()); err != nil {
		app.Logger().Error("last-block re-run: failed to record the committing binary", "error", err)
		return
	}
	if app.rerunInProgress {
		app.Logger().Info("last-block re-run: finished, block re-executed with this binary",
			"height", app.LastBlockHeight(), "binary", rerun.BinaryIdentity())
	}
	app.rerunMarkerWritten = true
	app.rerunInProgress = false
}

func (app *HeimdallApp) loadLatestOrRerunVersion() error {
	inProgress, err := rerun.Load(app.rerunDB, app.Logger(), app.LoadVersion, app.rollbackStoresAndLoad)
	if err != nil {
		return err
	}
	app.rerunInProgress = inProgress
	if inProgress {
		return nil
	}
	if err := app.LoadLatestVersion(); err != nil {
		return fmt.Errorf("error loading last version: %w", err)
	}
	return nil
}
