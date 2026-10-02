package app

import (
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store/rootmulti"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/client/flags"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	"github.com/stretchr/testify/require"
)

func TestDecideLastBlockRerun(t *testing.T) {
	const self = "v1@abc"

	tests := []struct {
		name        string
		appHeight   int64
		committedBy string
		pending     *PendingRerun
		wantRerun   bool
		wantTarget  int64
	}{
		{name: "fresh node", appHeight: 0, committedBy: "", wantRerun: false},
		{name: "only genesis height", appHeight: 1, committedBy: "", wantRerun: false},
		{name: "same binary", appHeight: 100, committedBy: self, wantRerun: false},
		{name: "binary without marker", appHeight: 100, committedBy: "", wantRerun: true, wantTarget: 99},
		{name: "different binary", appHeight: 100, committedBy: "v0@old", wantRerun: true, wantTarget: 99},
		{
			name: "pending re-run wins over a same-binary marker", appHeight: 100, committedBy: self,
			pending: &PendingRerun{TargetHeight: 99, Attempts: 1}, wantRerun: true, wantTarget: 99,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := DecideLastBlockRerun(tc.appHeight, tc.committedBy, self, tc.pending)
			require.Equal(t, tc.wantRerun, d.Rerun, d.Reason)
			if tc.wantRerun {
				require.Equal(t, tc.wantTarget, d.TargetHeight)
			}
		})
	}
}

func TestPendingRerunRoundTrip(t *testing.T) {
	db := dbm.NewMemDB()

	_, ok, err := ReadPendingRerun(db)
	require.NoError(t, err)
	require.False(t, ok)

	want := PendingRerun{TargetHeight: 54627194, Attempts: 2}
	require.NoError(t, WritePendingRerun(db, want))

	got, ok, err := ReadPendingRerun(db)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, want, got)

	_, err = decodePendingRerun([]byte{1, 2, 3})
	require.Error(t, err)
}

func TestRecordCommitByThisBinary(t *testing.T) {
	db := dbm.NewMemDB()
	require.NoError(t, WritePendingRerun(db, PendingRerun{TargetHeight: 9, Attempts: 1}))

	hApp := &HeimdallApp{rerunDB: db}
	hApp.recordCommitByThisBinaryWith(log.NewNopLogger(), 10)

	committedBy, err := ReadLastCommitBinary(db)
	require.NoError(t, err)
	require.Equal(t, BinaryIdentity(), committedBy)

	_, ok, err := ReadPendingRerun(db)
	require.NoError(t, err)
	require.False(t, ok, "a successful commit must clear the pending re-run")
	require.True(t, hApp.rerunMarkerWritten)
}

// committedApp returns an app and its db with three committed heights.
func committedApp(t *testing.T) (*HeimdallApp, *dbm.MemDB) {
	t.Helper()
	res := SetupApp(t, 1)
	for i := 0; i < 2; i++ {
		RequestFinalizeBlock(t, res.App, res.App.LastBlockHeight()+1)
		_, err := res.App.Commit()
		require.NoError(t, err)
	}
	return res.App, res.DB
}

func reopenApp(t *testing.T, db dbm.DB) *HeimdallApp {
	t.Helper()
	appOptions := make(simtestutil.AppOptionsMap)
	appOptions[flags.FlagHome] = DefaultNodeHome
	return NewHeimdallApp(log.NewTestLogger(t), db, nil, true, appOptions)
}

func TestCommitRecordsThisBinary(t *testing.T) {
	_, db := committedApp(t)

	committedBy, err := ReadLastCommitBinary(db)
	require.NoError(t, err)
	require.Equal(t, BinaryIdentity(), committedBy)
}

func TestLoadVersionForRerun(t *testing.T) {
	t.Run("no pending re-run loads the latest version", func(t *testing.T) {
		old, db := committedApp(t)
		latest := old.LastBlockHeight()

		hApp := reopenApp(t, db)
		require.Equal(t, latest, hApp.LastBlockHeight())
		require.False(t, hApp.rerunInProgress)
	})

	t.Run("fast path loads H-1 and keeps version H", func(t *testing.T) {
		old, db := committedApp(t)
		latest := old.LastBlockHeight()
		require.NoError(t, WritePendingRerun(db, PendingRerun{TargetHeight: latest - 1, Attempts: 1}))

		hApp := reopenApp(t, db)
		require.Equal(t, latest-1, hApp.LastBlockHeight())
		require.True(t, hApp.rerunInProgress)
		require.Equal(t, latest, rootmulti.GetLatestVersion(db), "version H must stay on disk")
	})

	t.Run("after repeated attempts the stores are rolled back to H-1", func(t *testing.T) {
		old, db := committedApp(t)
		latest := old.LastBlockHeight()
		require.NoError(t, WritePendingRerun(db, PendingRerun{TargetHeight: latest - 1, Attempts: fullRollbackAfterAttempts}))

		hApp := reopenApp(t, db)
		require.Equal(t, latest-1, hApp.LastBlockHeight())
		require.True(t, hApp.rerunInProgress)
		require.Equal(t, latest-1, rootmulti.GetLatestVersion(db))
	})

	t.Run("one attempt below the threshold keeps the fast path", func(t *testing.T) {
		old, db := committedApp(t)
		latest := old.LastBlockHeight()
		require.NoError(t, WritePendingRerun(db, PendingRerun{TargetHeight: latest - 1, Attempts: fullRollbackAfterAttempts - 1}))

		reopenApp(t, db)
		require.Equal(t, latest, rootmulti.GetLatestVersion(db))
	})

	t.Run("unloadable target falls back to the latest version", func(t *testing.T) {
		old, db := committedApp(t)
		latest := old.LastBlockHeight()
		require.NoError(t, WritePendingRerun(db, PendingRerun{TargetHeight: latest + 10, Attempts: 1}))

		hApp := reopenApp(t, db)
		require.Equal(t, latest, hApp.LastBlockHeight())
		require.False(t, hApp.rerunInProgress)

		_, ok, err := ReadPendingRerun(db)
		require.NoError(t, err)
		require.False(t, ok)
	})
}

func TestBinaryIdentityIsStable(t *testing.T) {
	id := BinaryIdentity()
	require.NotEmpty(t, id)
	require.Equal(t, id, BinaryIdentity())
}

func TestRecordCommitByThisBinaryOncePerProcess(t *testing.T) {
	db := dbm.NewMemDB()
	hApp := &HeimdallApp{rerunDB: db, rerunInProgress: true}
	hApp.recordCommitByThisBinaryWith(log.NewNopLogger(), 10)
	require.False(t, hApp.rerunInProgress)

	require.NoError(t, db.Set(lastCommitBinaryKey, []byte("other")))
	hApp.recordCommitByThisBinary()

	committedBy, err := ReadLastCommitBinary(db)
	require.NoError(t, err)
	require.Equal(t, "other", committedBy, "the marker is written once per process")
}
