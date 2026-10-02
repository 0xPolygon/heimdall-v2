package app

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"sync"

	"cosmossdk.io/log"
	dbm "github.com/cosmos/cosmos-db"

	hversion "github.com/0xPolygon/heimdall-v2/version"
)

// A node that runs an old binary past a hardfork commits the fork block with the old
// rules and halts one height later (CometBFT checks a block's results only in the next
// header). Restarting with the new binary does not re-execute the committed block, so
// the node stays stuck. Each binary therefore records itself as the committer; when a
// different binary starts, it rolls CometBFT back one height (keeping the block) and
// loads the app at H-1 without deleting H, so the handshake re-executes H. If the
// AppHash is unchanged, IAVL treats saving H again as a no-op, so this costs one block.
// One height is enough: every earlier height was checked by the header after it.
//
// The keys live in application.db outside any IAVL prefix: local, non-consensus data.

var (
	// lastCommitBinaryKey holds the identity of the binary that committed the latest height.
	lastCommitBinaryKey = []byte("heimdall/last-block-rerun/committed-by")
	// pendingRerunKey holds the target height (H-1) and attempt count of an unfinished re-run.
	pendingRerunKey = []byte("heimdall/last-block-rerun/pending")
)

// fullRollbackAfterAttempts is the number of fast-path attempts after which the app falls
// back to a full rollback of its stores to the target height.
const fullRollbackAfterAttempts = 2

var (
	binaryIdentityOnce sync.Once
	binaryIdentity     string
)

// BinaryIdentity identifies this binary for the re-run check: version@commit for release
// builds, or the sha256 of the executable when the build carries no version information
// (for example a plain docker build without .git), so that two different dev builds never
// look the same.
func BinaryIdentity() string {
	binaryIdentityOnce.Do(func() {
		if hversion.Version != "" && hversion.Commit != "" {
			binaryIdentity = hversion.Version + "@" + hversion.Commit
			return
		}
		binaryIdentity = "unversioned"
		exe, err := os.Executable()
		if err != nil {
			return
		}
		f, err := os.Open(exe)
		if err != nil {
			return
		}
		defer f.Close()
		h := sha256.New()
		if _, err := io.Copy(h, f); err != nil {
			return
		}
		binaryIdentity = "sha256:" + hex.EncodeToString(h.Sum(nil))
	})
	return binaryIdentity
}

// PendingRerun is an unfinished last-block re-run.
type PendingRerun struct {
	TargetHeight int64
	Attempts     uint32
}

func encodePendingRerun(p PendingRerun) []byte {
	bz := make([]byte, 12)
	binary.BigEndian.PutUint64(bz[:8], uint64(p.TargetHeight))
	binary.BigEndian.PutUint32(bz[8:], p.Attempts)
	return bz
}

func decodePendingRerun(bz []byte) (PendingRerun, error) {
	if len(bz) != 12 {
		return PendingRerun{}, fmt.Errorf("invalid pending re-run record length %d", len(bz))
	}
	return PendingRerun{
		TargetHeight: int64(binary.BigEndian.Uint64(bz[:8])),
		Attempts:     binary.BigEndian.Uint32(bz[8:]),
	}, nil
}

// ReadLastCommitBinary returns the identity of the binary that committed the latest height,
// or "" when no binary with this feature has committed yet.
func ReadLastCommitBinary(db dbm.DB) (string, error) {
	bz, err := db.Get(lastCommitBinaryKey)
	if err != nil {
		return "", err
	}
	return string(bz), nil
}

// ReadPendingRerun returns the unfinished re-run, if any.
func ReadPendingRerun(db dbm.DB) (PendingRerun, bool, error) {
	bz, err := db.Get(pendingRerunKey)
	if err != nil {
		return PendingRerun{}, false, err
	}
	if bz == nil {
		return PendingRerun{}, false, nil
	}
	p, err := decodePendingRerun(bz)
	if err != nil {
		return PendingRerun{}, false, err
	}
	return p, true, nil
}

// WritePendingRerun records an unfinished re-run. It is synced to disk, so that a node killed
// right after the CometBFT state rollback resumes the re-run on its next start.
func WritePendingRerun(db dbm.DB, p PendingRerun) error {
	return db.SetSync(pendingRerunKey, encodePendingRerun(p))
}

// RerunDecision says whether the last height must be re-executed on this start.
type RerunDecision struct {
	Rerun        bool
	TargetHeight int64
	Reason       string
}

// DecideLastBlockRerun is the pure decision behind the start-up check.
//   - appHeight: latest committed app version.
//   - committedBy: identity stored by the binary that committed appHeight ("" if none).
//   - self: identity of the running binary.
//   - pending: an unfinished re-run, if any.
func DecideLastBlockRerun(appHeight int64, committedBy, self string, pending *PendingRerun) RerunDecision {
	if pending != nil {
		return RerunDecision{Rerun: true, TargetHeight: pending.TargetHeight, Reason: "resuming unfinished re-run"}
	}
	if appHeight <= 1 {
		return RerunDecision{Reason: "no committed height to re-run"}
	}
	if committedBy == self {
		return RerunDecision{Reason: "last height committed by this binary"}
	}
	reason := fmt.Sprintf("last height committed by a different binary (%q)", committedBy)
	if committedBy == "" {
		reason = "last height committed by a binary without the re-run marker"
	}
	return RerunDecision{Rerun: true, TargetHeight: appHeight - 1, Reason: reason}
}

// loadVersionForRerun loads the app for a pending re-run instead of the latest version.
// It returns false when there is no pending re-run, so the caller loads the latest version.
func (app *HeimdallApp) loadVersionForRerun(db dbm.DB, logger log.Logger) (bool, error) {
	p, ok, err := ReadPendingRerun(db)
	if err != nil || !ok {
		return false, err
	}

	if p.Attempts >= fullRollbackAfterAttempts {
		// The fast path did not finish twice (most likely the re-executed AppHash differs from
		// the stored one, so IAVL refused to save the version again). Roll every store back.
		logger.Warn("last-block re-run: fast path did not finish, rolling the app back to the target height",
			"target", p.TargetHeight, "attempts", p.Attempts)
		// Roll back at the multistore level first: BaseApp.LoadLatestVersion seals the app and
		// may run only once, after the stores are at the target height.
		cms := app.CommitMultiStore()
		if err := cms.LoadLatestVersion(); err != nil {
			return true, err
		}
		if err := cms.RollbackToVersion(p.TargetHeight); err != nil {
			return true, fmt.Errorf("last-block re-run: full rollback to %d failed: %w", p.TargetHeight, err)
		}
		return true, app.LoadLatestVersion()
	}

	logger.Info("last-block re-run: loading app at the previous height; block will be re-executed",
		"target", p.TargetHeight, "attempt", p.Attempts)
	if err := app.LoadVersion(p.TargetHeight); err != nil {
		// For example, the version was pruned. Start from the latest version as before; the
		// handshake then restores the stored results of the last block, which is no worse than
		// running without this feature.
		logger.Error("last-block re-run: cannot load the previous height, starting from the latest version",
			"target", p.TargetHeight, "error", err)
		if delErr := db.DeleteSync(pendingRerunKey); delErr != nil {
			return false, delErr
		}
		return false, nil
	}
	return true, nil
}

// recordCommitByThisBinary runs after a successful commit. It stores this binary's identity and
// clears a finished re-run. It writes only once per process.
func (app *HeimdallApp) recordCommitByThisBinary() {
	if app.rerunMarkerWritten || app.rerunDB == nil {
		return
	}
	app.recordCommitByThisBinaryWith(app.Logger(), app.LastBlockHeight())
}

func (app *HeimdallApp) recordCommitByThisBinaryWith(logger log.Logger, height int64) {
	batch := app.rerunDB.NewBatch()
	defer batch.Close()

	if err := batch.Set(lastCommitBinaryKey, []byte(BinaryIdentity())); err != nil {
		logger.Error("last-block re-run: failed to record committing binary", "error", err)
		return
	}
	if err := batch.Delete(pendingRerunKey); err != nil {
		logger.Error("last-block re-run: failed to clear pending re-run", "error", err)
		return
	}
	if err := batch.WriteSync(); err != nil {
		logger.Error("last-block re-run: failed to write marker", "error", err)
		return
	}

	if app.rerunInProgress {
		logger.Info("last-block re-run: finished, block re-executed with this binary",
			"height", height, "binary", BinaryIdentity())
	}
	app.rerunMarkerWritten = true
	app.rerunInProgress = false
}
