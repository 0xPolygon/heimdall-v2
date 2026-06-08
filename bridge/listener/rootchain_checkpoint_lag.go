package listener

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"

	"github.com/0xPolygon/heimdall-v2/bridge/util"
	"github.com/0xPolygon/heimdall-v2/metrics"
	checkpointTypes "github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
)

// checkpointLagInterval is how often the per-node checkpoint-lag backstop polls
// L1. Fixed rather than configurable: this is a low-rate backstop metric, and a
// new config field would need helper/config.go plus packaging-template wiring for
// no operational benefit.
const checkpointLagInterval = 30 * time.Second

// minCheckpointIdGap is the checkpoint id gap at or above which the node is
// considered genuinely behind. A gap of 1 is the normal one-in-flight window
// (L1 has finalized a checkpoint this node has not yet acked), so the threshold
// is 2.
const minCheckpointIdGap = 2

// startCheckpointLagMonitor periodically compares the latest L1-finalized
// checkpoint against this node's committed checkpoint and records the lag as a
// metric. It runs on every node, independent of self-healing: a node whose
// consensus P2P is partitioned keeps an intact L1 link, so this is the backstop
// that observes "behind finalized state" when the peer-height catching_up signal
// is momentarily fooled by not-yet-evicted stale peers.
func (rl *RootChainListener) startCheckpointLagMonitor(ctx context.Context) {
	ticker := time.NewTicker(checkpointLagInterval)
	defer ticker.Stop()

	rl.Logger.Info("Checkpoint-lag monitor: started", "interval", checkpointLagInterval)

	for {
		select {
		case <-ticker.C:
			rl.recordCheckpointLag()
		case <-ctx.Done():
			rl.Logger.Info("Checkpoint-lag monitor: stopping")
			return
		}
	}
}

// recordCheckpointLag reads the latest L1 checkpoint and the local committed
// checkpoint and updates the lag gauges. It fails open: if either read fails it
// returns without touching the gauges, so a transient L1/query failure reads as
// "unknown" (gauge holds its last value) rather than "healthy" (0).
func (rl *RootChainListener) recordCheckpointLag() {
	l1Id, l1End, ok := rl.fetchLatestL1Checkpoint()
	if !ok {
		return
	}

	committed, ok := rl.fetchCommittedCheckpoint()
	if !ok {
		return
	}

	raw, effective, behind := computeCheckpointLag(l1Id, l1End, committed.Id, committed.EndBlock)

	metrics.BridgeCheckpointLagBlocks.Set(raw)
	metrics.BridgeCheckpointEffectiveLagBlocks.Set(effective)

	if behind {
		rl.Logger.Warn("Checkpoint-lag monitor: node is behind L1-finalized checkpoints",
			"l1_end", l1End, "committed_end", committed.EndBlock,
			"l1_id", l1Id, "committed_id", committed.Id, "lag_blocks", raw)
	}
}

// fetchLatestL1Checkpoint reads the latest finalized checkpoint from the L1
// RootChain contract, returning its header-block id and end block. ok is false
// (fail-open) on any error.
func (rl *RootChainListener) fetchLatestL1Checkpoint() (l1Id, l1End uint64, ok bool) {
	chainmanagerParams, err := util.GetChainmanagerParams(rl.cliCtx.Codec)
	if err != nil {
		rl.Logger.Debug("Checkpoint-lag monitor: failed to get chainmanager params", "error", err)
		return 0, 0, false
	}

	checkpointParams, err := util.GetCheckpointParams(rl.cliCtx.Codec)
	if err != nil {
		rl.Logger.Debug("Checkpoint-lag monitor: failed to get checkpoint params", "error", err)
		return 0, 0, false
	}

	rootChainInstance, err := rl.contractCaller.GetRootChainInstance(chainmanagerParams.ChainParams.RootChainAddress)
	if err != nil {
		rl.Logger.Debug("Checkpoint-lag monitor: failed to create rootchain instance", "error", err)
		return 0, 0, false
	}

	// Bound the L1 reads: the contractCaller's CurrentHeaderBlock/GetHeaderInfo
	// helpers call the generated bindings with nil CallOpts (context.Background),
	// so without an explicit deadline a stuck L1 RPC would pin this goroutine and
	// stop all future samples. Call the bindings directly with a timeout instead.
	ctx, cancel := context.WithTimeout(context.Background(), rl.contractCaller.MainChainTimeout)
	defer cancel()
	opts := &bind.CallOpts{Context: ctx}
	interval := checkpointParams.ChildChainBlockInterval

	currentHeaderBlock, err := rootChainInstance.CurrentHeaderBlock(opts)
	if err != nil {
		rl.Logger.Debug("Checkpoint-lag monitor: failed to fetch current L1 header block", "error", err)
		return 0, 0, false
	}
	if currentHeaderBlock == nil {
		rl.Logger.Debug("Checkpoint-lag monitor: rootchain returned nil current header block")
		return 0, 0, false
	}
	l1Id = currentHeaderBlock.Uint64() / interval

	headerBlock, err := rootChainInstance.HeaderBlocks(opts, new(big.Int).SetUint64(l1Id*interval))
	if err != nil {
		rl.Logger.Debug("Checkpoint-lag monitor: failed to fetch L1 header info", "headerId", l1Id, "error", err)
		return 0, 0, false
	}

	return l1Id, headerBlock.End.Uint64(), true
}

// fetchCommittedCheckpoint reads this node's latest committed checkpoint. ok is
// false (fail-open) on error, which includes the "no checkpoint committed yet"
// case: util.GetLatestCheckpoint returns an error rather than a nil checkpoint
// when none exists, so the result is non-nil whenever ok is true.
func (rl *RootChainListener) fetchCommittedCheckpoint() (*checkpointTypes.Checkpoint, bool) {
	committed, err := util.GetLatestCheckpoint(rl.cliCtx.Codec)
	if err != nil {
		rl.Logger.Debug("Checkpoint-lag monitor: failed to get latest committed checkpoint", "error", err)
		return nil, false
	}
	return committed, true
}

// computeCheckpointLag derives the raw lag, the alert-safe effective lag, and the
// behind flag from the L1 and committed checkpoint positions. Pure: all of the
// monitor's decision logic lives here so it is exercised directly by unit tests.
func computeCheckpointLag(l1Id, l1End, committedId, committedEnd uint64) (raw, effective float64, behind bool) {
	raw = checkpointLagBlocks(l1End, committedEnd)
	behind = checkpointBehind(l1Id, committedId)
	return raw, effectiveLag(raw, behind), behind
}

// checkpointLagBlocks is the Bor-block extent by which the local committed
// checkpoint end trails the latest L1-finalized checkpoint end, clamped at 0.
func checkpointLagBlocks(l1End, committedEnd uint64) float64 {
	if l1End <= committedEnd {
		return 0
	}
	return float64(l1End - committedEnd)
}

// checkpointBehind reports whether the node is behind by more than the normal
// one-in-flight checkpoint. The subtraction form avoids unsigned overflow if the
// committed id is momentarily ahead of the observed L1 id.
func checkpointBehind(l1Id, committedId uint64) bool {
	return l1Id > committedId && l1Id-committedId >= minCheckpointIdGap
}

// effectiveLag returns the raw lag only when the node is genuinely behind, else 0.
func effectiveLag(lag float64, behind bool) float64 {
	if behind {
		return lag
	}
	return 0
}
