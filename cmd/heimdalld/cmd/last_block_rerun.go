package heimdalld

import (
	"fmt"
	"path/filepath"

	"cosmossdk.io/store/rootmulti"
	cmtcmd "github.com/cometbft/cometbft/cmd/cometbft/commands"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/spf13/cobra"

	"github.com/0xPolygon/heimdall-v2/app/rerun"
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

// prepareLastBlockRerun opens application.db before the node does and runs the re-run check.
func prepareLastBlockRerun(svrCtx *server.Context) error {
	db, err := dbm.NewDB("application", server.GetAppDBBackend(svrCtx.Viper), filepath.Join(svrCtx.Config.RootDir, "data"))
	if err != nil {
		return fmt.Errorf("last-block re-run: open application db: %w", err)
	}
	defer db.Close()

	// Soft rollback: the block stays in the block store, so the handshake replays it against the
	// real app. Idempotent: when state is already one height below the store it changes nothing.
	rollback := func() (int64, error) {
		height, _, err := rollbackCometState(svrCtx.Config, false)
		return height, err
	}
	return rerun.Prepare(db, rootmulti.GetLatestVersion(db), svrCtx.Logger.With("module", "last-block-rerun"), rollback)
}
