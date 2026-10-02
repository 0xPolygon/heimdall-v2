package heimdalld

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	cmtcfg "github.com/cometbft/cometbft/config"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/server"
	gogotypes "github.com/cosmos/gogoproto/types"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/app"
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
func seedAppDB(t *testing.T, ctx *server.Context, appHeight int64, committedBy string, pending *app.PendingRerun) {
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
		require.NoError(t, app.WritePendingRerun(db, *pending))
	}
}

func readPending(t *testing.T, ctx *server.Context) (app.PendingRerun, bool) {
	t.Helper()
	db, err := dbm.NewDB("application", dbm.GoLevelDBBackend, filepath.Join(ctx.Config.RootDir, "data"))
	require.NoError(t, err)
	defer func() { require.NoError(t, db.Close()) }()

	p, ok, err := app.ReadPendingRerun(db)
	require.NoError(t, err)
	return p, ok
}

func TestPrepareLastBlockRerun(t *testing.T) {
	tests := []struct {
		name          string
		appHeight     int64
		committedBy   string
		pending       *app.PendingRerun
		rollbackTo    int64
		rollbackErr   error
		wantRollbacks int
		wantPending   *app.PendingRerun
	}{
		{name: "fresh node", wantRollbacks: 0},
		{name: "same binary", appHeight: 50, committedBy: app.BinaryIdentity(), wantRollbacks: 0},
		{
			name: "binary without marker", appHeight: 50, rollbackTo: 49,
			wantRollbacks: 1, wantPending: &app.PendingRerun{TargetHeight: 49, Attempts: 1},
		},
		{
			name: "different binary", appHeight: 50, committedBy: "v0@old", rollbackTo: 49,
			wantRollbacks: 1, wantPending: &app.PendingRerun{TargetHeight: 49, Attempts: 1},
		},
		{
			name: "resume counts the attempt", appHeight: 50, committedBy: "v0@old", rollbackTo: 49,
			pending:       &app.PendingRerun{TargetHeight: 49, Attempts: 1},
			wantRollbacks: 1, wantPending: &app.PendingRerun{TargetHeight: 49, Attempts: 2},
		},
		{
			name: "cometbft rollback fails", appHeight: 50, rollbackErr: errors.New("no blockstore"),
			wantRollbacks: 1,
		},
		{
			name: "cometbft height does not match", appHeight: 50, rollbackTo: 47,
			wantRollbacks: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := newRerunServerCtx(t)
			seedAppDB(t, ctx, tc.appHeight, tc.committedBy, tc.pending)
			rc := stubRollback(t, tc.rollbackTo, tc.rollbackErr)

			require.NoError(t, prepareLastBlockRerun(ctx))
			require.Equal(t, tc.wantRollbacks, rc.calls)
			require.False(t, rc.hard, "the re-run must keep the block in the block store")

			got, ok := readPending(t, ctx)
			if tc.wantPending == nil {
				if tc.pending == nil {
					require.False(t, ok)
				}
				return
			}
			require.True(t, ok)
			require.Equal(t, *tc.wantPending, got)
		})
	}
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

	cmd.PreRunE = nil
	wrapStartWithLastBlockRerun(cmd)
	require.NoError(t, cmd.PreRunE(cmd, nil))
	require.Equal(t, 1, rc.calls)
}
