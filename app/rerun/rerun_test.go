package rerun

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"cosmossdk.io/log"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/stretchr/testify/require"
)

func TestDecide(t *testing.T) {
	const self = "v1@abc"

	tests := []struct {
		name        string
		appHeight   int64
		committedBy string
		pending     *Pending
		wantRerun   bool
		wantTarget  int64
		wantReason  string
	}{
		{name: "fresh node", appHeight: 0, wantReason: "no committed height to re-run"},
		{name: "only the first height", appHeight: 1, wantReason: "no committed height to re-run"},
		{name: "same binary", appHeight: 100, committedBy: self, wantReason: "last height committed by this binary"},
		{
			name: "binary without marker", appHeight: 100, wantRerun: true, wantTarget: 99,
			wantReason: "last height committed by a binary without the re-run marker",
		},
		{
			name: "different binary", appHeight: 100, committedBy: "v0@old", wantRerun: true, wantTarget: 99,
			wantReason: `last height committed by a different binary ("v0@old")`,
		},
		{
			name: "pending re-run wins over a same-binary marker", appHeight: 100, committedBy: self,
			pending: &Pending{TargetHeight: 42, Attempts: 1}, wantRerun: true, wantTarget: 42,
			wantReason: "resuming unfinished re-run",
		},
		{
			name: "pending re-run on a fresh-looking app", appHeight: 0,
			pending: &Pending{TargetHeight: 7, Attempts: 1}, wantRerun: true, wantTarget: 7,
			wantReason: "resuming unfinished re-run",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := Decide(tc.appHeight, tc.committedBy, self, tc.pending)
			require.Equal(t, Decision{Rerun: tc.wantRerun, TargetHeight: tc.wantTarget, Reason: tc.wantReason}, d)
		})
	}
}

func TestPendingRoundTrip(t *testing.T) {
	db := dbm.NewMemDB()

	_, ok, err := ReadPending(db)
	require.NoError(t, err)
	require.False(t, ok)

	want := Pending{TargetHeight: 54627194, Attempts: 2}
	require.NoError(t, WritePending(db, want))

	got, ok, err := ReadPending(db)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, want, got)

	require.NoError(t, db.Set(pendingKey, []byte{1, 2, 3}))
	_, ok, err = ReadPending(db)
	require.Error(t, err)
	require.False(t, ok)
}

func TestRecordCommit(t *testing.T) {
	db := dbm.NewMemDB()
	require.NoError(t, WritePending(db, Pending{TargetHeight: 9, Attempts: 1}))

	require.NoError(t, RecordCommit(db, "v2@new"))

	committedBy, err := ReadLastCommitBinary(db)
	require.NoError(t, err)
	require.Equal(t, "v2@new", committedBy)

	_, ok, err := ReadPending(db)
	require.NoError(t, err)
	require.False(t, ok, "a successful commit clears the pending re-run")
}

func TestComputeIdentity(t *testing.T) {
	require.Equal(t, "v1.2.3@abc", computeIdentity("v1.2.3", "abc", nil))

	exe := filepath.Join(t.TempDir(), "heimdalld")
	require.NoError(t, os.WriteFile(exe, []byte("binary"), 0o600))
	sum := sha256.Sum256([]byte("binary"))
	want := "sha256:" + hex.EncodeToString(sum[:])

	exeFn := func() (string, error) { return exe, nil }
	require.Equal(t, want, computeIdentity("", "abc", exeFn), "a missing version uses the executable hash")
	require.Equal(t, want, computeIdentity("v1", "", exeFn), "a missing commit uses the executable hash")

	require.Equal(t, "unversioned", computeIdentity("", "", func() (string, error) { return "", errors.New("no exe") }))
	require.Equal(t, "unversioned", computeIdentity("", "", func() (string, error) { return exe + ".missing", nil }))

	require.Equal(t, BinaryIdentity(), BinaryIdentity())
	require.NotEmpty(t, BinaryIdentity())
}

func TestPrepare(t *testing.T) {
	self := BinaryIdentity()

	tests := []struct {
		name          string
		appHeight     int64
		committedBy   string
		pending       *Pending
		rollbackTo    int64
		rollbackErr   error
		wantRollbacks int
		wantPending   *Pending
	}{
		{name: "fresh node", wantRollbacks: 0},
		{name: "same binary", appHeight: 50, committedBy: self, wantRollbacks: 0},
		{
			name: "binary without marker", appHeight: 50, rollbackTo: 49,
			wantRollbacks: 1, wantPending: &Pending{TargetHeight: 49, Attempts: 1},
		},
		{
			name: "different binary", appHeight: 50, committedBy: "v0@old", rollbackTo: 49,
			wantRollbacks: 1, wantPending: &Pending{TargetHeight: 49, Attempts: 1},
		},
		{
			name: "resume counts the attempt", appHeight: 50, committedBy: self, rollbackTo: 49,
			pending:       &Pending{TargetHeight: 49, Attempts: 1},
			wantRollbacks: 1, wantPending: &Pending{TargetHeight: 49, Attempts: 2},
		},
		{
			name: "cometbft rollback fails", appHeight: 50, rollbackErr: errors.New("no blockstore"),
			wantRollbacks: 1,
		},
		{name: "cometbft height does not match", appHeight: 50, rollbackTo: 47, wantRollbacks: 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db := dbm.NewMemDB()
			if tc.committedBy != "" {
				require.NoError(t, db.Set(lastCommitBinaryKey, []byte(tc.committedBy)))
			}
			if tc.pending != nil {
				require.NoError(t, WritePending(db, *tc.pending))
			}

			calls := 0
			rollback := func() (int64, error) {
				calls++
				return tc.rollbackTo, tc.rollbackErr
			}

			require.NoError(t, Prepare(db, tc.appHeight, log.NewNopLogger(), rollback))
			require.Equal(t, tc.wantRollbacks, calls)

			got, ok, err := ReadPending(db)
			require.NoError(t, err)
			if tc.wantPending == nil {
				require.Equal(t, tc.pending != nil, ok, "a failed re-run leaves the pending record as it was")
				return
			}
			require.True(t, ok)
			require.Equal(t, *tc.wantPending, got)
		})
	}
}

func TestPrepareReadErrors(t *testing.T) {
	db := dbm.NewMemDB()
	require.NoError(t, db.Set(pendingKey, []byte{1}))

	err := Prepare(db, 50, log.NewNopLogger(), func() (int64, error) { return 49, nil })
	require.ErrorContains(t, err, "read pending re-run")
}

func TestLoad(t *testing.T) {
	errLoad := errors.New("version pruned")
	errRollback := errors.New("rollback failed")

	tests := []struct {
		name            string
		pending         *Pending
		loadErr         error
		rollbackErr     error
		wantInProgress  bool
		wantErr         error
		wantLoaded      int64
		wantRolledBack  int64
		wantPendingLeft bool
	}{
		{name: "no pending re-run"},
		{
			name: "fast path", pending: &Pending{TargetHeight: 9, Attempts: 1},
			wantInProgress: true, wantLoaded: 9, wantPendingLeft: true,
		},
		{
			name: "full rollback after repeated attempts", pending: &Pending{TargetHeight: 9, Attempts: FullRollbackAfterAttempts},
			wantInProgress: true, wantRolledBack: 9, wantPendingLeft: true,
		},
		{
			name: "full rollback fails", pending: &Pending{TargetHeight: 9, Attempts: FullRollbackAfterAttempts + 1},
			rollbackErr: errRollback, wantInProgress: true, wantErr: errRollback, wantRolledBack: 9, wantPendingLeft: true,
		},
		{
			name: "unloadable target falls back to the latest version", pending: &Pending{TargetHeight: 9, Attempts: 1},
			loadErr: errLoad, wantLoaded: 9,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db := dbm.NewMemDB()
			if tc.pending != nil {
				require.NoError(t, WritePending(db, *tc.pending))
			}

			var loaded, rolledBack int64
			load := func(h int64) error { loaded = h; return tc.loadErr }
			rollback := func(h int64) error { rolledBack = h; return tc.rollbackErr }

			inProgress, err := Load(db, log.NewNopLogger(), load, rollback)
			require.ErrorIs(t, err, tc.wantErr)
			require.Equal(t, tc.wantInProgress, inProgress)
			require.Equal(t, tc.wantLoaded, loaded)
			require.Equal(t, tc.wantRolledBack, rolledBack)

			_, ok, err := ReadPending(db)
			require.NoError(t, err)
			require.Equal(t, tc.wantPendingLeft, ok)
		})
	}
}
