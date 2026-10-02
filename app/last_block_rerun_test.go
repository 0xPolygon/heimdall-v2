package app

import (
	"errors"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store/rootmulti"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/client/flags"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/app/rerun"
)

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

	committedBy, err := rerun.ReadLastCommitBinary(db)
	require.NoError(t, err)
	require.Equal(t, rerun.BinaryIdentity(), committedBy)
}

func TestRecordCommitByThisBinaryOncePerProcess(t *testing.T) {
	hApp, db := committedApp(t)
	require.True(t, hApp.rerunMarkerWritten)

	require.NoError(t, db.Set([]byte("heimdall/last-block-rerun/committed-by"), []byte("other")))
	hApp.recordCommitByThisBinary()

	committedBy, err := rerun.ReadLastCommitBinary(db)
	require.NoError(t, err)
	require.Equal(t, "other", committedBy, "the marker is written once per process")
}

func TestLoadLatestOrRerunVersion(t *testing.T) {
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
		require.NoError(t, rerun.WritePending(db, rerun.Pending{TargetHeight: latest - 1}))

		hApp := reopenApp(t, db)
		require.Equal(t, latest-1, hApp.LastBlockHeight())
		require.True(t, hApp.rerunInProgress)
		require.Equal(t, latest, rootmulti.GetLatestVersion(db), "version H stays on disk")
	})

	t.Run("full rollback moves the stores to H-1", func(t *testing.T) {
		old, db := committedApp(t)
		latest := old.LastBlockHeight()
		require.NoError(t, rerun.WritePending(db, rerun.Pending{TargetHeight: latest - 1, Attempts: rerun.FullRollbackAfterAttempts}))

		hApp := reopenApp(t, db)
		require.Equal(t, latest-1, hApp.LastBlockHeight())
		require.True(t, hApp.rerunInProgress)
		require.Equal(t, latest-1, rootmulti.GetLatestVersion(db))
	})

	t.Run("unloadable target falls back to the latest version", func(t *testing.T) {
		old, db := committedApp(t)
		latest := old.LastBlockHeight()
		require.NoError(t, rerun.WritePending(db, rerun.Pending{TargetHeight: latest + 10}))

		hApp := reopenApp(t, db)
		require.Equal(t, latest, hApp.LastBlockHeight())
		require.False(t, hApp.rerunInProgress)
	})

	t.Run("failed full rollback starts from the latest version", func(t *testing.T) {
		old, db := committedApp(t)
		latest := old.LastBlockHeight()
		require.NoError(t, rerun.WritePending(db, rerun.Pending{TargetHeight: latest + 10, Attempts: rerun.FullRollbackAfterAttempts}))

		hApp := reopenApp(t, db)
		require.Equal(t, latest, hApp.LastBlockHeight())
		require.False(t, hApp.rerunInProgress)
		_, ok, err := rerun.ReadPending(db)
		require.NoError(t, err)
		require.False(t, ok)
	})

	t.Run("finished re-run clears the in-progress flag on commit", func(t *testing.T) {
		old, db := committedApp(t)
		latest := old.LastBlockHeight()
		require.NoError(t, rerun.WritePending(db, rerun.Pending{TargetHeight: latest - 1}))

		hApp := reopenApp(t, db)
		require.True(t, hApp.rerunInProgress)
		RequestFinalizeBlock(t, hApp, latest)
		_, err := hApp.Commit()
		require.NoError(t, err)

		require.False(t, hApp.rerunInProgress)
		require.Equal(t, latest, hApp.LastBlockHeight())
		_, ok, err := rerun.ReadPending(db)
		require.NoError(t, err)
		require.False(t, ok)
	})
}

type failingBatchDB struct{ dbm.DB }

func (d failingBatchDB) NewBatch() dbm.Batch { return failingBatch{d.DB.NewBatch()} }

type failingBatch struct{ dbm.Batch }

func (failingBatch) WriteSync() error { return errors.New("disk full") }

func TestRecordCommitByThisBinaryWriteFails(t *testing.T) {
	hApp, db := committedApp(t)
	require.NoError(t, db.Delete([]byte("heimdall/last-block-rerun/committed-by")))
	hApp.rerunDB = failingBatchDB{DB: db}
	hApp.rerunMarkerWritten = false
	hApp.rerunInProgress = true

	hApp.recordCommitByThisBinary()

	require.False(t, hApp.rerunMarkerWritten, "a failed write is retried on the next commit")
	require.True(t, hApp.rerunInProgress)
	committedBy, err := rerun.ReadLastCommitBinary(db)
	require.NoError(t, err)
	require.Empty(t, committedBy)
}

func TestCorruptPendingRerunStopsTheStart(t *testing.T) {
	_, db := committedApp(t)
	require.NoError(t, db.Set([]byte("heimdall/last-block-rerun/pending"), []byte{1}))

	require.Panics(t, func() { reopenApp(t, db) })
}
