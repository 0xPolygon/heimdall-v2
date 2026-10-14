// Package rerun re-executes the last committed block after a binary change.
//
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
package rerun

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

var (
	// lastCommitBinaryKey holds the identity of the binary that committed the latest height.
	lastCommitBinaryKey = []byte("heimdall/last-block-rerun/committed-by")
	// pendingKey holds the target height (H-1) and attempt count of an unfinished re-run.
	pendingKey = []byte("heimdall/last-block-rerun/pending")
)

// EnableOption is the app option that lets the app act on a pending re-run. Only the start
// command sets it (next to the CometBFT-side check), so commands that build the app for other
// work, such as export or rollback, never change the stores because of a pending re-run.
const EnableOption = "heimdall.last-block-rerun"

// FullRollbackAfterAttempts is the number of executed fast-path attempts after which the stores
// are rolled back to the target height (the fast path cannot finish when the AppHash changed).
const FullRollbackAfterAttempts = 1

// unversioned is the identity of a build without version information whose executable cannot be read.
const unversioned = "unversioned"

var (
	identityOnce sync.Once
	identity     string
)

// BinaryIdentity is version@commit for release builds, or the sha256 of the executable when
// the build carries no version information, so two different dev builds never look the same.
func BinaryIdentity() string {
	identityOnce.Do(func() { identity = computeIdentity(hversion.Version, hversion.Commit, os.Executable) })
	return identity
}

func computeIdentity(version, commit string, executable func() (string, error)) string {
	if version != "" && commit != "" {
		return version + "@" + commit
	}
	exe, err := executable()
	if err != nil {
		return unversioned
	}
	f, err := os.Open(exe)
	if err != nil {
		return unversioned
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return unversioned
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

// Pending is an unfinished re-run.
type Pending struct {
	TargetHeight int64
	Attempts     uint32
}

func encodePending(p Pending) []byte {
	bz := make([]byte, 12)
	binary.BigEndian.PutUint64(bz[:8], uint64(p.TargetHeight))
	binary.BigEndian.PutUint32(bz[8:], p.Attempts)
	return bz
}

func decodePending(bz []byte) (Pending, error) {
	if len(bz) != 12 {
		return Pending{}, fmt.Errorf("invalid pending re-run record length %d", len(bz))
	}
	return Pending{
		TargetHeight: int64(binary.BigEndian.Uint64(bz[:8])),
		Attempts:     binary.BigEndian.Uint32(bz[8:]),
	}, nil
}

// ReadLastCommitBinary returns the identity of the binary that committed the latest height,
// or "" when no binary with this feature has committed yet.
func ReadLastCommitBinary(db dbm.DB) (string, error) {
	bz, err := db.Get(lastCommitBinaryKey)
	return string(bz), err
}

// ReadPending returns the unfinished re-run, if any.
func ReadPending(db dbm.DB) (Pending, bool, error) {
	bz, err := db.Get(pendingKey)
	if err != nil || bz == nil {
		return Pending{}, false, err
	}
	p, err := decodePending(bz)
	return p, err == nil, err
}

// WritePending records an unfinished re-run. It is synced, so a node killed right after the
// CometBFT rollback resumes the re-run on its next start.
func WritePending(db dbm.DB, p Pending) error {
	return db.SetSync(pendingKey, encodePending(p))
}

// RecordCommit stores the committing binary and clears a finished re-run.
func RecordCommit(db dbm.DB, binaryIdentity string) error {
	batch := db.NewBatch()
	defer batch.Close()
	if err := batch.Set(lastCommitBinaryKey, []byte(binaryIdentity)); err != nil {
		return err
	}
	if err := batch.Delete(pendingKey); err != nil {
		return err
	}
	return batch.WriteSync()
}

// Decision says whether the last height must be re-executed on this start.
type Decision struct {
	Rerun        bool
	TargetHeight int64
	Reason       string
}

// Decide is the pure decision behind the start-up check.
func Decide(appHeight int64, committedBy, self string, pending *Pending) Decision {
	if pending != nil {
		return Decision{Rerun: true, TargetHeight: pending.TargetHeight, Reason: "resuming unfinished re-run"}
	}
	if appHeight <= 1 {
		return Decision{Reason: "no committed height to re-run"}
	}
	if committedBy == self {
		return Decision{Reason: "last height committed by this binary"}
	}
	reason := fmt.Sprintf("last height committed by a different binary (%q)", committedBy)
	if committedBy == "" {
		reason = "last height committed by a binary without the re-run marker"
	}
	return Decision{Rerun: true, TargetHeight: appHeight - 1, Reason: reason}
}

// Prepare runs before the node opens its databases. When the last height must be re-executed,
// it calls rollbackComet, which must roll CometBFT state back one height without removing the
// block and return the new state height, and records the pending re-run. Attempts are counted
// by Load, where the fast path really runs, so a start that aborts before loading the app does
// not count. Prepare never blocks the start: when the re-run is not possible, it clears any
// pending re-run, logs why and the node starts as before.
func Prepare(db dbm.DB, appHeight int64, logger log.Logger, rollbackComet func() (int64, error)) error {
	p, hasPending, err := ReadPending(db)
	if err != nil {
		return fmt.Errorf("last-block re-run: read pending re-run: %w", err)
	}
	var pending *Pending
	if hasPending {
		pending = &p
	}
	committedBy, err := ReadLastCommitBinary(db)
	if err != nil {
		return fmt.Errorf("last-block re-run: read marker: %w", err)
	}

	d := Decide(appHeight, committedBy, BinaryIdentity(), pending)
	if !d.Rerun {
		logger.Debug("no re-run needed", "reason", d.Reason, "app_height", appHeight)
		return nil
	}
	logger.Info("re-running the last block with this binary",
		"reason", d.Reason, "app_height", appHeight, "target", d.TargetHeight, "binary", BinaryIdentity())

	height, err := rollbackComet()
	if err != nil {
		logger.Error("cannot re-run the last block, starting without it", "error", err)
		return db.DeleteSync(pendingKey)
	}
	if height != d.TargetHeight {
		logger.Error("cannot re-run the last block: CometBFT state does not match the app height, starting without it",
			"cometbft_height", height, "target", d.TargetHeight)
		return db.DeleteSync(pendingKey)
	}

	if pending == nil {
		if err := WritePending(db, Pending{TargetHeight: d.TargetHeight}); err != nil {
			return fmt.Errorf("last-block re-run: write pending re-run: %w", err)
		}
	}
	logger.Info("CometBFT state rolled back one height; block will be re-executed during the handshake",
		"target", d.TargetHeight)
	return nil
}

// Load loads the app for a pending re-run. loadVersion must load the app at a height without
// deleting later versions; rollbackStores must roll every store back to a height and load it.
// It returns false when there is no re-run, so the caller loads the latest version. When the
// re-run cannot be done, it clears the pending re-run and returns false, so the node starts as
// it would without this feature.
func Load(db dbm.DB, logger log.Logger, loadVersion, rollbackStores func(int64) error) (bool, error) {
	p, ok, err := ReadPending(db)
	if err != nil || !ok {
		return false, err
	}

	if p.Attempts >= FullRollbackAfterAttempts {
		logger.Warn("last-block re-run: fast path did not finish, rolling the app back to the target height",
			"target", p.TargetHeight, "attempts", p.Attempts)
		if err := rollbackStores(p.TargetHeight); err != nil {
			logger.Error("last-block re-run: cannot roll the app back, starting from the latest version",
				"target", p.TargetHeight, "error", err)
			return false, db.DeleteSync(pendingKey)
		}
		return true, nil
	}

	p.Attempts++
	if err := WritePending(db, p); err != nil {
		return false, fmt.Errorf("last-block re-run: write pending re-run: %w", err)
	}
	logger.Info("last-block re-run: loading app at the previous height; block will be re-executed",
		"target", p.TargetHeight, "attempt", p.Attempts)
	if err := loadVersion(p.TargetHeight); err != nil {
		// For example, the version was pruned. The handshake then restores the stored results of
		// the last block, which is no worse than running without this feature.
		logger.Error("last-block re-run: cannot load the previous height, starting from the latest version",
			"target", p.TargetHeight, "error", err)
		return false, db.DeleteSync(pendingKey)
	}
	return true, nil
}
