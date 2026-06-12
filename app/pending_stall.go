package app

import (
	"bytes"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/0xPolygon/heimdall-v2/common/strutil"
	"github.com/0xPolygon/heimdall-v2/helper"
	borTypes "github.com/0xPolygon/heimdall-v2/x/bor/types"
	milestoneAbci "github.com/0xPolygon/heimdall-v2/x/milestone/abci"
	milestoneTypes "github.com/0xPolygon/heimdall-v2/x/milestone/types"
)

// handlePendingMilestone is the PreBlocker action for a pending (1/3<=PM<2/3) milestone. Before the
// span-rotation-on-stall hardfork it simply logs and lets the pending gate suppress rotation; after
// it, a stalled pending head can force a rotation (POS-3629).
func (app *HeimdallApp) handlePendingMilestone(ctx sdk.Context, pendingMilestone *milestoneTypes.MilestoneProposition, supportingValidatorIDs map[uint64]struct{}) error {
	if helper.IsSpanRotationOnStall(ctx.BlockHeight()) {
		return app.checkAndRotateOnPendingStall(ctx, pendingMilestone, supportingValidatorIDs)
	}

	app.Logger().Info("1/3rd voting power found on milestone proposition, skipping span rotation",
		"startBlock", pendingMilestone.StartBlockNumber,
		"endBlock", pendingMilestone.StartBlockNumber+uint64(len(pendingMilestone.BlockHashes)-1),
		"blockHashes", strutil.HashesToString(pendingMilestone.BlockHashes),
	)
	return nil
}

// checkAndRotateOnPendingStall forces a span rotation when the >1/3-agreed pending bor head
// (POS-3629) stops advancing for longer than the stall threshold, even under a pending milestone.
// It tracks the head and its hash+td identity across blocks; the stall clock restarts whenever
// either changes, so a flapping or contested tip never ages it.
func (app *HeimdallApp) checkAndRotateOnPendingStall(ctx sdk.Context, pendingMilestone *milestoneTypes.MilestoneProposition, supportingValidatorIDs map[uint64]struct{}) error {
	logger := app.Logger()

	if len(pendingMilestone.BlockHashes) == 0 {
		return nil
	}

	// Pending propositions are always computed for blocks after the last finalized milestone,
	// so the head is necessarily beyond finality (the backlog is non-empty).
	pendingHead := pendingMilestone.StartBlockNumber + uint64(len(pendingMilestone.BlockHashes)-1)
	pendingHeadID := milestoneAbci.MilestonePropositionHeadID(pendingMilestone)

	trackedBlock, trackedID, trackedHeight, err := app.MilestoneKeeper.GetPendingBorBlockTracking(ctx)
	if err != nil {
		logger.Error("Error occurred while getting pending bor block tracking", "error", err)
		return err
	}

	// (Re)start the stall clock on first observation or whenever the agreed head or its identity
	// changes. A fresh observation is not yet a stall, so we never rotate on it.
	if trackedHeight == 0 || pendingHead != trackedBlock || !bytes.Equal(pendingHeadID, trackedID) {
		if err := app.MilestoneKeeper.SetPendingBorBlockTracking(ctx, pendingHead, pendingHeadID, uint64(ctx.BlockHeight())); err != nil {
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

	return app.rotateSpanFromPendingHead(ctx, pendingHead, pendingHeadID, supportingValidatorIDs)
}

// rotateSpanFromPendingHead mints a new veblop span starting at pendingHead+1, preserving the
// pending blocks (no reorg — >1/3 agreement vouches for them). It anchors the new span's end to
// lastSpan.EndBlock so it fully supersedes the old span's runway, and excludes the stalled and
// previously-failed producers from selection.
func (app *HeimdallApp) rotateSpanFromPendingHead(ctx sdk.Context, pendingHead uint64, pendingHeadID []byte, supportingValidatorIDs map[uint64]struct{}) error {
	logger := app.Logger()

	addSpanCtx, spanCache := app.cacheTxContext(ctx)

	lastSpan, err := app.BorKeeper.GetLastSpan(ctx)
	if err != nil {
		logger.Error("Error occurred while getting last span", "error", err)
		return err
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
		endBlock += params.SpanDuration
	}

	// Resolve the producer responsible for the block that should come next (pendingHead+1), not the
	// last produced block. On the first rotation both resolve to the same span. On a re-rotation under
	// a persistent stall the prior rotation already installed a span starting at pendingHead+1, so
	// looking up pendingHead would fall back (newest span first) to the old, already-rotated-out
	// producer and never mark the actually-stalled one failed.
	currentProducer, err := app.BorKeeper.FindCurrentProducerID(ctx, pendingHead+1)
	if err != nil {
		logger.Error("Error occurred while finding current producer", "error", err)
		return err
	}

	excludedProducers, err := app.pendingStallExcludedProducers(ctx, currentProducer)
	if err != nil {
		return err
	}

	if err := app.BorKeeper.AddNewVeBlopSpan(addSpanCtx, currentProducer, pendingHead+1, endBlock, lastSpan.BorChainId, supportingValidatorIDs, uint64(ctx.BlockHeight()), borTypes.RoundRobinDefault, excludedProducers); err != nil {
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
