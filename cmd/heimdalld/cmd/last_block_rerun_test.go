package heimdalld

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	cmtcfg "github.com/cometbft/cometbft/config"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/server"
	gogotypes "github.com/cosmos/gogoproto/types"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/app/rerun"
)

type rollbackCall struct {
	calls     int
	hard      bool
	retHeight int64
	retErr    error
}

func stubRollback(t *testing.T, height int64, err error) *rollbackCall {
	t.Helper()
	rc := &rollbackCall{retHeight: height, retErr: err}
	prev := rollbackCometState
	rollbackCometState = func(_ *cmtcfg.Config, removeBlock bool) (int64, []byte, error) {
		rc.calls++
		rc.hard = removeBlock
		return rc.retHeight, nil, rc.retErr
	}
	t.Cleanup(func() { rollbackCometState = prev })
	return rc
}

func newRerunServerCtx(t *testing.T) *server.Context {
	t.Helper()
	ctx := server.NewDefaultContext()
	ctx.Config.SetRoot(t.TempDir())
	return ctx
}

// seedAppDB writes the app height, committing binary and pending re-run, then closes the db.
func seedAppDB(t *testing.T, ctx *server.Context, appHeight int64, committedBy string, pending *rerun.Pending) {
	t.Helper()
	db, err := dbm.NewDB("application", dbm.GoLevelDBBackend, filepath.Join(ctx.Config.RootDir, "data"))
	require.NoError(t, err)
	defer func() { require.NoError(t, db.Close()) }()

	if appHeight > 0 {
		bz, err := gogotypes.StdInt64Marshal(appHeight)
		require.NoError(t, err)
		require.NoError(t, db.Set([]byte("s/latest"), bz))
	}
	if committedBy != "" {
		require.NoError(t, db.Set([]byte("heimdall/last-block-rerun/committed-by"), []byte(committedBy)))
	}
	if pending != nil {
		require.NoError(t, rerun.WritePending(db, *pending))
	}
}

func readPending(t *testing.T, ctx *server.Context) (rerun.Pending, bool) {
	t.Helper()
	db, err := dbm.NewDB("application", dbm.GoLevelDBBackend, filepath.Join(ctx.Config.RootDir, "data"))
	require.NoError(t, err)
	defer func() { require.NoError(t, db.Close()) }()

	p, ok, err := rerun.ReadPending(db)
	require.NoError(t, err)
	return p, ok
}

func TestPrepareLastBlockRerun(t *testing.T) {
	ctx := newRerunServerCtx(t)
	seedAppDB(t, ctx, 50, "", nil)
	rc := stubRollback(t, 49, nil)

	require.NoError(t, prepareLastBlockRerun(ctx))
	require.Equal(t, 1, rc.calls)
	require.False(t, rc.hard, "the re-run keeps the block in the block store")

	got, ok := readPending(t, ctx)
	require.True(t, ok)
	require.Equal(t, rerun.Pending{TargetHeight: 49}, got)
}

func TestWrapStartWithLastBlockRerun(t *testing.T) {
	ctx := newRerunServerCtx(t)
	seedAppDB(t, ctx, 50, "", nil)
	rc := stubRollback(t, 49, nil)

	prevErr := errors.New("previous pre-run failed")
	cmd := &cobra.Command{PreRunE: func(*cobra.Command, []string) error { return prevErr }}
	cmd.SetContext(context.WithValue(context.Background(), server.ServerContextKey, ctx))
	wrapStartWithLastBlockRerun(cmd)

	require.ErrorIs(t, cmd.PreRunE(cmd, nil), prevErr)
	require.Zero(t, rc.calls, "the re-run check must not run when the previous pre-run fails")

	require.Nil(t, ctx.Viper.Get(rerun.EnableOption), "the app option is set only when the start pre-run succeeds")

	cmd.PreRunE = nil
	wrapStartWithLastBlockRerun(cmd)
	require.NoError(t, cmd.PreRunE(cmd, nil))
	require.Equal(t, 1, rc.calls)
	require.Equal(t, true, ctx.Viper.Get(rerun.EnableOption), "the start app must act on the pending re-run")
}

func TestPrepareLastBlockRerunOpenFails(t *testing.T) {
	ctx := server.NewDefaultContext()
	home := filepath.Join(t.TempDir(), "home")
	require.NoError(t, os.WriteFile(home, nil, 0o600))
	ctx.Config.SetRoot(home)
	rc := stubRollback(t, 49, nil)

	require.ErrorContains(t, prepareLastBlockRerun(ctx), "open application db")
	require.Zero(t, rc.calls)
}
