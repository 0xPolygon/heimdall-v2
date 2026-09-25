package listener

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/0xPolygon/heimdall-v2/bridge/util"
	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/metrics"
)

// resolveRecoverableStateSyncRange finds the portion of [gapStart, gapEnd]
// that's actually worth recovering and logs the outcome. ok is false when
// there's nothing to do this cycle (an error, already logged, or none of the
// gap is old enough yet) — the caller should return without looping.
func (rl *RootChainListener) resolveRecoverableStateSyncRange(ctx context.Context, gapStart, gapEnd int64) (start, end int64, ok bool) {
	// The gap between heimdall's stateId and the subgraph's latest is not
	// necessarily a real miss — it's what normal L1 finality lag looks like
	// every cycle too. Find the highest stateId actually old enough to be
	// worth recovering before touching anything else in the gap: a stateId
	// whose L1 event isn't past SHMaxDepthDuration yet would just be skipped
	// by processEvent anyway, but discovering that costs a subgraph lookup
	// plus an L1 receipt and block-time fetch per ID. Binary-searching the
	// boundary turns the common case (the whole gap is only finality lag,
	// nothing recoverable yet) into a single check instead of a pass over
	// every ID, and a genuine backlog into O(log gap) lookups instead of
	// O(gap).
	recoverableEnd, err := rl.findRecoverableStateSyncBoundary(ctx, gapStart, gapEnd)
	if err != nil {
		rl.Logger.Error("Self-healing: failed to determine recoverable state-sync boundary", "gapStart", gapStart, "gapEnd", gapEnd, "error", err)
		return 0, 0, false
	}
	if recoverableEnd < gapStart {
		rl.Logger.Info("Self-healing: entire state-sync gap is within the finality depth window; nothing to recover yet", "gapStart", gapStart, "gapEnd", gapEnd)
		return 0, 0, false
	}
	if recoverableEnd < gapEnd {
		rl.Logger.Info("Self-healing: part of the state-sync gap is still within the finality depth window; will retry next cycle", "recoverableEnd", recoverableEnd, "gapEnd", gapEnd)
	}
	return gapStart, recoverableEnd, true
}

// recoverStateSyncId attempts to recover a single missing stateId. Returns
// false only when the context was canceled mid-recovery, signaling the
// caller to stop the whole cycle instead of moving to the next ID.
func (rl *RootChainListener) recoverStateSyncId(ctx context.Context, stateId int64, sleepTimer *time.Timer) bool {
	if _, err := util.GetClerkEventRecord(stateId, rl.cliCtx.Codec); err == nil {
		rl.Logger.Info("Self-healing: state ID already synced on Heimdall; skipping", "stateId", stateId)
		return true
	}

	rl.Logger.Info("Self-healing: missing state detected; processing StateSynced event", "stateId", stateId)

	// getStateSynced retries its subgraph lookup and L1 receipt fetch
	// internally, each on its own backoff; no outer retry here, so a
	// deterministic content mismatch fails once instead of retrying a
	// result that can't change.
	stateSynced, err := rl.getStateSynced(ctx, stateId)
	if err != nil {
		rl.Logger.Error("Self-healing: failed to retrieve StateSynced event for missing state", "stateId", stateId, "error", err)
		return true
	}

	metrics.SelfHealStateSyncsProcessed.Inc()

	const maxRetriesPerState = 3
	for attempt := 0; attempt < maxRetriesPerState; attempt++ {
		synced, keepGoing := rl.attemptStateSyncRecovery(ctx, stateId, stateSynced, attempt, sleepTimer)
		if !keepGoing {
			return false
		}
		if synced {
			return true
		}
	}

	rl.Logger.Error("Self-healing: giving up on stateId after max retries; moving to next", "stateId", stateId, "maxRetries", maxRetriesPerState)
	return true
}

// attemptStateSyncRecovery runs one recovery attempt for stateId: process the
// event, then poll Heimdall until the record is confirmed or the poll budget
// is exhausted. synced reports whether this attempt succeeded; keepGoing is
// false only when the context was canceled, telling the caller to abort
// immediately rather than retry.
func (rl *RootChainListener) attemptStateSyncRecovery(ctx context.Context, stateId int64, stateSynced *types.Log, attempt int, sleepTimer *time.Timer) (synced, keepGoing bool) {
	ignore, err := rl.processEvent(ctx, stateSynced)
	if err != nil {
		rl.Logger.Error("Self-healing: failed to process StateSynced event and update Heimdall", "stateId", stateId, "attempt", attempt+1, "error", err)
		return false, true
	}
	if ignore {
		return true, true
	}

	if !waitOnTimer(ctx, sleepTimer, time.Second) {
		return false, false
	}

	for statusCheck := 0; statusCheck < 15; statusCheck++ {
		if _, err = util.GetClerkEventRecord(stateId, rl.cliCtx.Codec); err == nil {
			rl.Logger.Info("Self-healing: stateId found on Heimdall after processing", "stateId", stateId)
			return true, true
		}
		rl.Logger.Info("Self-healing: stateId not yet found on Heimdall; retrying", "stateId", stateId)
		if !waitOnTimer(ctx, sleepTimer, time.Second) {
			return false, false
		}
	}

	rl.Logger.Warn("Self-healing: stateId not confirmed after polling; will retry", "stateId", stateId, "attempt", attempt+1)
	return false, true
}

// waitOnTimer resets sleepTimer to d and waits for it, returning false if ctx
// is canceled first.
func waitOnTimer(ctx context.Context, sleepTimer *time.Timer, d time.Duration) bool {
	sleepTimer.Reset(d)
	select {
	case <-sleepTimer.C:
		return true
	case <-ctx.Done():
		return false
	}
}

// findRecoverableStateSyncBoundary returns the highest stateId in [lo, hi]
// whose L1 event is old enough to be worth recovering (block time at least
// SHMaxDepthDuration in the past), or lo-1 if even the oldest stateId in the
// gap is too recent. StateSynced stateIds increase monotonically with L1
// block number, so "old enough" is monotonically non-increasing across the
// range (older stateId -> earlier block -> more time has passed): a standard
// binary search finds the boundary in O(log(hi-lo)) L1 lookups instead of
// checking every ID in the gap. The two ends are checked first so the common
// cases (the whole gap is too recent, or the whole gap is already old
// enough) resolve with a single lookup instead of a full search.
func (rl *RootChainListener) findRecoverableStateSyncBoundary(ctx context.Context, lo, hi int64) (int64, error) {
	loOldEnough, err := rl.stateSyncOldEnough(ctx, lo)
	if err != nil {
		return 0, err
	}
	if !loOldEnough {
		return lo - 1, nil
	}

	hiOldEnough, err := rl.stateSyncOldEnough(ctx, hi)
	if err != nil {
		return 0, err
	}
	if hiOldEnough {
		return hi, nil
	}

	for lo < hi {
		mid := lo + (hi-lo+1)/2

		midOldEnough, err := rl.stateSyncOldEnough(ctx, mid)
		if err != nil {
			return 0, err
		}

		if midOldEnough {
			lo = mid
		} else {
			hi = mid - 1
		}
	}

	return lo, nil
}

// stateSyncOldEnough resolves stateId's L1 event and reports whether its
// block time is past the self-heal recovery depth. It reuses getStateSynced's
// full validation path — the same strictness self-heal always applies to a
// resolved event — purely to read the event's block timestamp; a probe ID
// that ends up inside the recoverable range gets resolved again by the
// caller's processing loop, which is a small, bounded amount of repeated
// work (at most a handful of probe IDs per cycle) traded for skipping the
// entire too-recent remainder of the gap.
func (rl *RootChainListener) stateSyncOldEnough(ctx context.Context, stateId int64) (bool, error) {
	vLog, err := rl.getStateSynced(ctx, stateId)
	if err != nil {
		return false, fmt.Errorf("resolving stateId %d for age check: %w", stateId, err)
	}

	blockTime, err := rl.contractCaller.GetMainChainBlockTime(ctx, vLog.BlockNumber)
	if err != nil {
		return false, fmt.Errorf("fetching block time for stateId %d: %w", stateId, err)
	}

	return time.Since(blockTime) >= helper.GetConfig().SHMaxDepthDuration, nil
}
