package app

import (
	"bytes"
	"math"

	abciTypes "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/0xPolygon/heimdall-v2/common/strutil"
	"github.com/0xPolygon/heimdall-v2/helper"
	borTypes "github.com/0xPolygon/heimdall-v2/x/bor/types"
	milestoneAbci "github.com/0xPolygon/heimdall-v2/x/milestone/abci"
	milestoneTypes "github.com/0xPolygon/heimdall-v2/x/milestone/types"
	stakeTypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

// handlePendingMilestone is the PreBlocker action for a pending (1/3<=PM<2/3) milestone. Before the
// Ithaca hardfork it simply logs and lets the pending gate suppress rotation; after
// it, a stalled pending head can force a rotation.
func (app *HeimdallApp) handlePendingMilestone(ctx sdk.Context, pendingMilestone *milestoneTypes.MilestoneProposition, validatorSet *stakeTypes.ValidatorSet, extVoteInfo []abciTypes.ExtendedVoteInfo, minMajorityVP int64) error {
	logger := app.Logger()

	if helper.IsIthaca(ctx.BlockHeight()) {
		// Key the stall/rotation on the >1/3-agreed actual bor head, not the capped milestone
		// proposition tail, so blocks the producer made beyond the proposition window are preserved
		// rather than reorged.
		//
		// Bound the agreed head by the last span's end — the furthest scheduled runway. An honest
		// producer cannot advance past the scheduled spans, so a head beyond it is fabricated. Bounding
		// by the scheduled runway (not the active span alone) keeps recovery working when bor honestly
		// crosses into an already-scheduled future span while milestones lag behind. GetMajorityActualHead
		// then selects the head with the greatest voting power, so a >1/3 byzantine minority cannot
		// outvote the converged honest majority to install a fabricated head.
		//
		// Residual (accepted, >1/3-trust model of the pending band): a byzantine slice holding strictly
		// more voting power than any honest agreement could still steer the agreed head within the
		// scheduled runway. That is only reachable while honest votes are fragmented across heads, which
		// a genuine stall removes by converging them on the real tip. Even then it cannot point past the
		// runway, reorg finalized blocks, or choose the new producer — selection uses the 2/3-fed active
		// producer set (rotateSpanFromPendingHead), not the pending milestone's supporters.
		lastSpan, err := app.BorKeeper.GetLastSpan(ctx)
		if err != nil {
			logger.Error("Error occurred while getting last span", "error", err)
			return err
		}
		agreedHead, agreedHash, found, err := milestoneAbci.GetMajorityActualHead(ctx, validatorSet, extVoteInfo, minMajorityVP, lastSpan.EndBlock)
		if err != nil {
			logger.Error("Error occurred while tallying actual bor head", "error", err)
			return err
		}
		if !found {
			// No >1/3-agreed actual bor head this block (heads not yet converged, or pre-fork VEs at the
			// activation boundary). Skip rotation without resetting any running stall clock; never rotate
			// from the truncated proposition tail.
			logger.Debug("Pending stall: no >1/3-agreed actual bor head this block, skipping rotation")
			return nil
		}
		return app.checkAndRotateOnPendingStall(ctx, agreedHead, agreedHash)
	}

	logger.Info("1/3rd voting power found on milestone proposition, skipping span rotation",
		"startBlock", pendingMilestone.StartBlockNumber,
		"endBlock", pendingMilestone.StartBlockNumber+uint64(len(pendingMilestone.BlockHashes)-1),
		"blockHashes", strutil.HashesToString(pendingMilestone.BlockHashes),
	)
	return nil
}

// checkAndRotateOnPendingStall forces a span rotation when the >1/3-agreed actual bor head
// stops advancing for longer than the stall threshold, even under a pending milestone.
// The head is the validators' agreed actual latest block (resolved by the caller via
// GetMajorityActualHead), not the capped milestone proposition tail, so blocks the producer made
// beyond the proposition window are preserved rather than reorged. It tracks the head and its hash
// identity across blocks; the stall clock restarts whenever either changes, so continued healthy
// production (an advancing head) never ages it.
func (app *HeimdallApp) checkAndRotateOnPendingStall(ctx sdk.Context, agreedHead uint64, agreedHash []byte) error {
	logger := app.Logger()

	trackedBlock, trackedID, trackedHeight, err := app.MilestoneKeeper.GetPendingBorBlockTracking(ctx)
	if err != nil {
		logger.Error("Error occurred while getting pending bor block tracking", "error", err)
		return err
	}

	// (Re)start the stall clock on first observation or whenever the agreed head or its identity
	// changes — so continued healthy bor production, which advances the head, never ages the clock.
	// A fresh observation is not yet a stall, so we never rotate on it.
	if trackedHeight == 0 || agreedHead != trackedBlock || !bytes.Equal(agreedHash, trackedID) {
		if err := app.MilestoneKeeper.SetPendingBorBlockTracking(ctx, agreedHead, agreedHash, uint64(ctx.BlockHeight())); err != nil {
			logger.Error("Error occurred while setting pending bor block tracking", "error", err)
			return err
		}
		return nil
	}

	// Head and identity unchanged since trackedHeight: measure the stall with signed arithmetic
	// (trackedHeight can be a future, debounced height set after a recent rotation).
	borStallDiff := ctx.BlockHeight() - int64(trackedHeight)
	if borStallDiff <= helper.GetBorStallThreshold(ctx) {
		return nil
	}

	return app.rotateSpanFromPendingHead(ctx, agreedHead, agreedHash)
}

// rotateSpanFromPendingHead mints a new veblop span starting at pendingHead+1, where pendingHead is
// the >1/3-agreed actual bor head (not the capped proposition tail), so every block up to that head
// is preserved — no reorg of the blocks the producer made beyond the proposition window. It anchors
// the new span's end to lastSpan.EndBlock so it fully supersedes the old span's runway, draws the
// candidate set from the 2/3-fed latest active producers (not the pending milestone's 1/3 supporters,
// which a byzantine slice can control), and excludes the stalled and previously-failed producers.
func (app *HeimdallApp) rotateSpanFromPendingHead(ctx sdk.Context, pendingHead uint64, pendingHeadID []byte) error {
	logger := app.Logger()

	addSpanCtx, spanCache := app.cacheTxContext(ctx)

	lastSpan, err := app.BorKeeper.GetLastSpan(ctx)
	if err != nil {
		logger.Error("Error occurred while getting last span", "error", err)
		return err
	}

	// pendingHead comes from an aggregated, unfinalized milestone proposition whose StartBlockNumber is
	// not bounded against chain state, so a >=1/3 byzantine slice could push it arbitrarily far past
	// lastSpan.EndBlock. An honest producer can never advance beyond its span's end, so a head past it
	// is not a real stall to rotate on. Bail before the runway loop (which would otherwise spin for huge
	// values, or overflow into an infinite loop near MaxUint64) and the producer lookup (which would
	// error for an out-of-range block and halt the PreBlocker).
	if pendingHead > lastSpan.EndBlock {
		// An honest producer cannot advance beyond its span end, so a >1/3-agreed head past it is
		// invalid/unexpected vote data rather than a real stall — warn, don't silently info-log.
		logger.Warn("Pending bor head is beyond the last span end, skipping rotation",
			"pendingHead", pendingHead,
			"lastSpanEndBlock", lastSpan.EndBlock,
		)
		return nil
	}
	if pendingHead == math.MaxUint64 {
		logger.Warn("Pending bor head is MaxUint64, skipping rotation")
		return nil
	}

	params, err := app.BorKeeper.GetParams(ctx)
	if err != nil {
		logger.Error("Error occurred while getting bor params", "error", err)
		return err
	}

	// Cover at least the old span's committed runway so producer lookup (newest span first) can
	// never fall back to the stalled producer for blocks the old span still owns.
	endBlock := lastSpan.EndBlock
	for endBlock <= pendingHead {
		if params.SpanDuration == 0 || endBlock > math.MaxUint64-params.SpanDuration {
			logger.Warn("Pending-stall span end would overflow, skipping rotation",
				"endBlock", endBlock,
				"spanDuration", params.SpanDuration,
				"pendingHead", pendingHead,
			)
			return nil
		}
		endBlock += params.SpanDuration
	}

	// Resolve the producer of the next block to produce (pendingHead+1), not the last produced block.
	// FindCurrentProducerID scans newest span first, then older spans, so this also handles a future
	// scheduled lastSpan whose StartBlock is beyond pendingHead+1: the lookup falls through to the span
	// that actually owns the next block. At span exhaustion (pendingHead == lastSpan.EndBlock),
	// pendingHead+1 lies beyond every span, so fall back to pendingHead.
	producerLookupBlock := pendingHead
	if pendingHead < math.MaxUint64 {
		nextBlock := pendingHead + 1
		if nextBlock <= lastSpan.EndBlock {
			producerLookupBlock = nextBlock
		}
	}

	currentProducer, err := app.BorKeeper.FindCurrentProducerID(ctx, producerLookupBlock)
	if err != nil && producerLookupBlock < lastSpan.StartBlock {
		// Defensive fallback for non-contiguous/corrupt span state. Normal future-scheduled spans are
		// contiguous, so the next-block lookup above resolves through an older span; only fall back to
		// the last span's anchor if lookup cannot resolve any owner for pendingHead+1.
		producerLookupBlock = lastSpan.StartBlock
		currentProducer, err = app.BorKeeper.FindCurrentProducerID(ctx, producerLookupBlock)
	}
	if err != nil {
		logger.Error("Error occurred while finding current producer", "error", err)
		return err
	}

	excludedProducers, err := app.pendingStallExcludedProducers(ctx, currentProducer)
	if err != nil {
		return err
	}

	// Draw candidates from the 2/3-fed active producer set, mirroring checkAndRotateCurrentSpan. The
	// pending milestone's supporters are only a 1/3 set and a byzantine slice can make them
	// byzantine-only via the per-block hash tie-break, so they must not gate producer selection.
	latestActiveProducer, err := app.BorKeeper.GetLatestActiveProducer(ctx)
	if err != nil {
		logger.Error("Error occurred while getting latest active producer", "error", err)
		return err
	}

	if err := app.BorKeeper.AddNewVeBlopSpan(addSpanCtx, currentProducer, pendingHead+1, endBlock, lastSpan.BorChainId, latestActiveProducer, uint64(ctx.BlockHeight()), borTypes.RoundRobinDefault, excludedProducers, false); err != nil {
		// Don't halt consensus if no producer can be selected this block; retry next block.
		logger.Warn("Error occurred while adding new veblop span on pending stall", "error", err)
		return nil
	}

	if err := app.recordPendingStallRotation(addSpanCtx, pendingHead, pendingHeadID, endBlock, currentProducer); err != nil {
		return err
	}

	spanCache.Write()
	return nil
}

// pendingStallExcludedProducers is the set kept out of selection on a pending-stall rotation: the
// stalled producer plus the latest failed set (honored in both the active and fallback paths).
func (app *HeimdallApp) pendingStallExcludedProducers(ctx sdk.Context, currentProducer uint64) (map[uint64]struct{}, error) {
	failedProducers, err := app.BorKeeper.GetLatestFailedProducer(ctx)
	if err != nil {
		app.Logger().Error("Error occurred while getting latest failed producer", "error", err)
		return nil, err
	}

	excluded := map[uint64]struct{}{currentProducer: {}}
	for id := range failedProducers {
		excluded[id] = struct{}{}
	}
	return excluded, nil
}

// recordPendingStallRotation debounces both rotation clocks past the buffer (so we don't immediately
// re-rotate before the new producer can extend the head) and records the stalled producer as failed.
func (app *HeimdallApp) recordPendingStallRotation(ctx sdk.Context, pendingHead uint64, pendingHeadID []byte, endBlock, currentProducer uint64) error {
	logger := app.Logger()
	debounceHeight := uint64(ctx.BlockHeight()) + helper.GetSpanRotationBuffer(ctx)

	if err := app.MilestoneKeeper.SetLastMilestoneBlock(ctx, debounceHeight); err != nil {
		logger.Error("Error occurred while setting last milestone block", "error", err)
		return err
	}
	if err := app.MilestoneKeeper.SetPendingBorBlockTracking(ctx, pendingHead, pendingHeadID, debounceHeight); err != nil {
		logger.Error("Error occurred while setting pending bor block tracking", "error", err)
		return err
	}
	if err := app.BorKeeper.AddLatestFailedProducer(ctx, currentProducer); err != nil {
		logger.Error("Error occurred while adding latest failed producer", "error", err)
		return err
	}

	logger.Info("Span rotated due to a stalled pending bor head",
		"pendingHead", pendingHead,
		"newSpanStartBlock", pendingHead+1,
		"newSpanEndBlock", endBlock,
		"currentProducerID", currentProducer,
	)
	return nil
}

// debouncePendingStallClock advances the pending-stall clock to debounceHeight (preserving the tracked
// head and identity) after checkAndRotateCurrentSpan rotates the producer for the stalled current
// range. Without it, a pending milestone reappearing at the same head — before the new producer has
// extended it — would measure the stall from the pre-rotation baseline and immediately re-rotate the
// producer just installed for that range.
//
// The sibling checkAndAddFutureSpan path is deliberately not debounced here: it only schedules a span
// beyond lastSpan.EndBlock, so any still-pending head it leaves behind belongs to the current range,
// and pending-stall must stay free to rotate the producer responsible for that head. An unconditional
// debounce there would instead hand an already-stalled producer a fresh buffer window — a liveness
// regression. No-op before the fork or when no stall clock is running, so it adds no pre-fork state write.
func (app *HeimdallApp) debouncePendingStallClock(ctx sdk.Context, debounceHeight uint64) error {
	if !helper.IsIthaca(ctx.BlockHeight()) {
		return nil
	}

	trackedBlock, trackedID, trackedHeight, err := app.MilestoneKeeper.GetPendingBorBlockTracking(ctx)
	if err != nil {
		app.Logger().Error("Error occurred while getting pending bor block tracking", "error", err)
		return err
	}
	if trackedHeight == 0 {
		return nil
	}

	if err := app.MilestoneKeeper.SetPendingBorBlockTracking(ctx, trackedBlock, trackedID, debounceHeight); err != nil {
		app.Logger().Error("Error occurred while debouncing pending bor block tracking", "error", err)
		return err
	}
	return nil
}
