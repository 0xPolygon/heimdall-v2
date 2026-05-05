package listener

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/0xPolygon/heimdall-v2/bridge/util"
	"github.com/0xPolygon/heimdall-v2/helper"
)

// maxStakeNoncesPerCycle bounds queued nonces per validator per self-heal tick.
// Polygon PoS runs one self-healing node per network, so total per-cycle calls
// cap at validatorSetSize * maxStakeNoncesPerCycle on that single node.
const maxStakeNoncesPerCycle = 10

var (
	stateSyncedCounter = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "self_healing",
		Subsystem: helper.GetConfig().Chain,
		Name:      "StateSynced",
		Help:      "The total number of missing StateSynced events processed",
	})

	stakeEventCounter = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "self_healing",
		Subsystem: helper.GetConfig().Chain,
		Name:      "StakeEvent",
		Help:      "The total number of missing nonce-gated stake events (StakeUpdate, SignerChange, UnstakeInit) processed",
	})

	checkpointAckCounter = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "self_healing",
		Subsystem: helper.GetConfig().Chain,
		Name:      "NewHeaderBlock",
		Help:      "The total number of acks sent for missing NewHeaderBlock events",
	})
)

type subGraphClient struct {
	graphUrl   string
	httpClient *http.Client
}

// startSelfHealing starts self-healing processes for all required events
func (rl *RootChainListener) startSelfHealing(ctx context.Context) {
	if !helper.GetConfig().EnableSH || helper.GetConfig().SubGraphUrl == "" {
		rl.Logger.Info("Self-healing is disabled")
		return
	}

	rl.subGraphClient = &subGraphClient{
		graphUrl:   helper.GetConfig().SubGraphUrl,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	stakeEventsTicker := time.NewTicker(helper.GetConfig().SHStakeUpdateInterval)
	stateSyncedTicker := time.NewTicker(helper.GetConfig().SHStateSyncedInterval)
	checkpointAckTicker := time.NewTicker(helper.GetConfig().SHCheckpointAckInterval)

	rl.Logger.Info("Self-healing: started")

	for {
		select {
		case <-stakeEventsTicker.C:
			rl.processStakeEvents(ctx)
		case <-stateSyncedTicker.C:
			rl.processStateSynced(ctx)
		case <-checkpointAckTicker.C:
			rl.processCheckpointAck(ctx)
		case <-ctx.Done():
			rl.Logger.Info("Self-healing: stopping")
			stakeEventsTicker.Stop()
			stateSyncedTicker.Stop()
			checkpointAckTicker.Stop()

			return
		}
	}
}

func (rl *RootChainListener) processCheckpointAck(ctx context.Context) {
	rl.Logger.Info("Self-healing: processing checkpoint")

	// Get the latest header block event from L1 using the subgraph.
	latestL1Checkpoint, err := rl.getLatestCheckpointFromL1(ctx)
	if err != nil {
		rl.Logger.Error("Self-healing: failed to fetch latest header block event from L1 subgraph", "error", err)
		return
	}

	l1HeaderBlockId, err := strconv.ParseUint(latestL1Checkpoint.HeaderBlockId, 10, 64)
	if err != nil {
		rl.Logger.Error("Self-healing: failed to parse L1 checkpoint header block ID", "error", err)
		return
	}

	// Get checkpoint parameters to get ChildChainBlockInterval.
	checkpointParams, err := util.GetCheckpointParams(rl.cliCtx.Codec)
	if err != nil {
		rl.Logger.Error("Self-healing: failed to get checkpoint params", "error", err)
		return
	}

	l1HeaderBlockId = l1HeaderBlockId / checkpointParams.ChildChainBlockInterval

	// Get the latest checkpoint id from Heimdall using the checkpoint ack count.
	// Using GetLatestCheckpoint returns an error if there is no checkpoint.
	// So we use GetCheckpointAckCount instead.
	ackCount, err := util.GetCheckpointAckCount(rl.cliCtx.Codec)
	if err != nil {
		rl.Logger.Error("Self-healing: failed to get checkpoint ack count", "error", err)
		return
	}

	if l1HeaderBlockId == ackCount {
		rl.Logger.Info("Self-healing: latest checkpoint is already synced on heimdall; skipping", "l1HeaderBlockId", l1HeaderBlockId, "heimdallAckCount", ackCount)
		return
	}

	// Check if we have a checkpoint in the buffer.
	bufferedCheckpoint, err := util.GetBufferedCheckpoint(rl.cliCtx.Codec)
	if err != nil {
		rl.Logger.Error("Self-healing: failed to get buffered checkpoint", "error", err)
		return
	}

	if bufferedCheckpoint == nil || bufferedCheckpoint.Id == 0 {
		rl.Logger.Warn("Self-healing: empty buffered checkpoint")
		return
	}

	// Check if the buffered checkpoint matches the L1 checkpoint.
	if l1HeaderBlockId != bufferedCheckpoint.Id {
		rl.Logger.Info("Self-healing: no matching buffered checkpoint found", "l1HeaderBlockId", l1HeaderBlockId, "bufferedCheckpointId", bufferedCheckpoint.Id)
		return
	}

	rl.Logger.Info("Self-healing: found matching buffered checkpoint, preparing to send ACK", "checkpointId", l1HeaderBlockId)

	// Get the transaction receipt to construct the ACK.
	receipt, err := rl.contractCaller.MainChainClient.TransactionReceipt(ctx, common.HexToHash(latestL1Checkpoint.TransactionHash))
	if err != nil {
		rl.Logger.Error("Self-healing: failed to get transaction receipt for L1 checkpoint", "txHash", latestL1Checkpoint.TransactionHash, "error", err)
		return
	}
	// Find the correct log within the transaction.
	var targetLog *types.Log
	for _, log := range receipt.Logs {
		if strconv.Itoa(int(log.Index)) == latestL1Checkpoint.LogIndex {
			targetLog = log
			rl.Logger.Info("Self-healing: retrieved log for NewHeaderBlock event", "headerBlockId", l1HeaderBlockId, "logIndex", latestL1Checkpoint.LogIndex, "txHash", latestL1Checkpoint.TransactionHash)
			break
		}
	}
	if targetLog == nil {
		rl.Logger.Error("Self-healing: failed to find matching log in transaction receipt", "txHash", latestL1Checkpoint.TransactionHash, "expectedLogIndex", latestL1Checkpoint.LogIndex)
		return
	}

	// Marshal the log to JSON for the task queue.
	logBytes, err := json.Marshal(*targetLog)
	if err != nil {
		rl.Logger.Error("Self-healing: failed to marshal log to JSON", "error", err)
		return
	}

	checkpointAckCounter.Inc()

	// Send the checkpoint ACK task.
	rl.SendTaskWithDelay("sendCheckpointAckToHeimdall", helper.NewHeaderBlockEvent, logBytes, 0, nil)
	rl.Logger.Info("Self-healing: successfully queued checkpoint ACK task", "headerBlockId", l1HeaderBlockId, "logIndex", latestL1Checkpoint.LogIndex, "txHash", targetLog.TxHash.Hex())
}

// processStakeEvents recovers any missing nonce-gated stake event for each validator.
// StakeUpdate, SignerChange, and UnstakeInit events share a single per-validator nonce counter.
// If Heimdall's nonce for a validator lags the L1 maximum across these three event types,
// the missing event is fetched from the subgraph and replayed.
func (rl *RootChainListener) processStakeEvents(ctx context.Context) {
	validatorSet, err := util.GetValidatorSet(rl.cliCtx.Codec)
	if err != nil {
		rl.Logger.Error("Self-healing: failed to fetch validator set from heimdall", "error", err)
		return
	}

	rl.Logger.Info("Self-healing: fetched validator list from heimdall", "validatorCount", len(validatorSet.Validators))

	var wg sync.WaitGroup
	for _, validator := range validatorSet.Validators {
		wg.Add(1)
		go func(id uint64) {
			defer wg.Done()
			rl.recoverStakeEventsForValidator(ctx, id)
		}(validator.ValId)
	}
	wg.Wait()
}

func (rl *RootChainListener) recoverStakeEventsForValidator(ctx context.Context, id uint64) {
	heimdallNonce, l1MaxNonce, ok := rl.fetchValidatorNonces(ctx, id)
	if !ok || l1MaxNonce <= heimdallNonce {
		return
	}

	// Successive iterations are paced by util.StakeNonceRetryDelay so the bridge
	// processor sees nonces in committed order rather than firing its
	// deferred-retry path with growing per-task multipliers.
	queued := uint64(0)
	for nonce := heimdallNonce + 1; nonce <= l1MaxNonce && queued < maxStakeNoncesPerCycle; nonce++ {
		if !rl.replayStakeEvent(ctx, id, l1MaxNonce, nonce) {
			break
		}
		queued++
		if nonce < l1MaxNonce && queued < maxStakeNoncesPerCycle && !pauseBetweenReplays(ctx) {
			return
		}
	}

	if remaining := l1MaxNonce - heimdallNonce - queued; remaining > 0 {
		rl.Logger.Warn("Self-healing: stake event backlog exceeds per-cycle limit; will continue next cycle", "validatorId", id, "queued", queued, "remaining", remaining, "perCycleLimit", maxStakeNoncesPerCycle)
	}
}

// fetchValidatorNonces returns the validator's heimdall and L1 nonces
func (rl *RootChainListener) fetchValidatorNonces(ctx context.Context, id uint64) (uint64, uint64, bool) {
	heimdallNonce, err := util.GetValidatorNonce(id, rl.cliCtx.Codec)
	if err != nil {
		rl.Logger.Error("Self-healing: failed to fetch nonce for validator from Heimdall", "validatorId", id, "error", err)
		return 0, 0, false
	}

	var l1MaxNonce uint64
	if err = helper.ExponentialBackoff(func() error {
		l1MaxNonce, err = rl.getMaxL1NonceForValidator(ctx, id)
		return err
	}, 3, time.Second); err != nil {
		rl.Logger.Error("Self-healing: failed to fetch latest nonce from Ethereum (L1) for validator", "validatorId", id, "error", err)
		return 0, 0, false
	}
	rl.Logger.Info("Self-healing: retrieved nonces for validator", "validatorId", id, "ethereumNonce", l1MaxNonce, "heimdallNonce", heimdallNonce)
	return heimdallNonce, l1MaxNonce, true
}

// pauseBetweenReplays returns false if context is canceled during the wait.
func pauseBetweenReplays(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(util.StakeNonceRetryDelay):
		return true
	}
}

// replayStakeEvent fetches and replays the stake event at (id, nonce). Returns
// false on any condition that makes further iteration pointless this cycle:
// subgraph fetch error, processEvent error, or event too recent for
// SHMaxDepthDuration (since later nonces have strictly later block times).
func (rl *RootChainListener) replayStakeEvent(ctx context.Context, id, l1MaxNonce, nonce uint64) bool {
	rl.Logger.Info("Self-healing: validator is behind; processing missing stake event", "validatorId", id, "ethereumNonce", l1MaxNonce, "nextExpectedNonce", nonce)

	var stakeEventLog *types.Log
	var err error
	if err = helper.ExponentialBackoff(func() error {
		stakeEventLog, err = rl.getStakeEventLogByNonce(ctx, id, nonce)
		return err
	}, 3, time.Second); err != nil {
		rl.Logger.Error("Self-healing: failed to retrieve stake event from subgraph", "validatorId", id, "nonce", nonce, "error", err)
		return false
	}
	rl.Logger.Info("Self-healing: fetched stake event from Ethereum", "validatorId", id, "nonce", nonce, "blockNumber", stakeEventLog.BlockNumber, "txHash", stakeEventLog.TxHash.Hex())

	skipped, err := rl.processEvent(ctx, stakeEventLog)
	if err != nil {
		rl.Logger.Error("Self-healing: failed to process stake event", "validatorId", id, "nonce", nonce, "error", err)
		return false
	}
	if skipped {
		rl.Logger.Info("Self-healing: stake event too recent for self-heal; will retry next cycle", "validatorId", id, "nonce", nonce)
		return false
	}

	stakeEventCounter.Inc()
	rl.Logger.Info("Self-healing: successfully processed stake event", "validatorId", id, "nonce", nonce)
	return true
}

// processStateSynced checks if chains are in sync, otherwise syncs them by broadcasting missing events
func (rl *RootChainListener) processStateSynced(ctx context.Context) {
	latestPolygonStateId, err := rl.getCurrentStateID(ctx)
	if err != nil {
		rl.Logger.Error("Self-healing: failed to fetch current Polygon stateId from StateReceiver contract", "error", err)
		return
	}

	latestEthereumStateId, err := rl.getLatestStateID(ctx)
	if err != nil {
		rl.Logger.Error("Self-healing: failed to fetch latest Ethereum stateId from StateSender contract", "error", err)
		return
	}
	rl.Logger.Info("Self-healing: retrieved latest state IDs", "polygonStateId", latestPolygonStateId, "ethereumStateId", latestEthereumStateId)

	if latestEthereumStateId.Cmp(latestPolygonStateId) != 1 {
		return
	}

	const maxRetriesPerState = 3

	sleepTimer := time.NewTimer(0)
	if !sleepTimer.Stop() {
		<-sleepTimer.C
	}

	defer sleepTimer.Stop()

	for i := latestPolygonStateId.Int64() + 1; i <= latestEthereumStateId.Int64(); i++ {
		if _, err = util.GetClerkEventRecord(i, rl.cliCtx.Codec); err == nil {
			rl.Logger.Info("Self-healing: state ID already synced on Heimdall; skipping", "stateId", i)
			continue
		}

		rl.Logger.Info("Self-healing: missing state detected; processing StateSynced event", "stateId", i)

		var stateSynced *types.Log

		if err = helper.ExponentialBackoff(func() error {
			stateSynced, err = rl.getStateSynced(ctx, i)
			return err
		}, 3, time.Second); err != nil {
			rl.Logger.Error("Self-healing: failed to retrieve StateSynced event for missing state", "stateId", i, "error", err)
			continue
		}

		stateSyncedCounter.Inc()

		var synced bool

		for attempt := 0; attempt < maxRetriesPerState; attempt++ {
			ignore, err := rl.processEvent(ctx, stateSynced)
			if err != nil {
				rl.Logger.Error("Self-healing: failed to process StateSynced event and update Heimdall", "stateId", i, "attempt", attempt+1, "error", err)
				continue
			}

			if ignore {
				synced = true
				break
			}

			sleepTimer.Reset(1 * time.Second)

			select {
			case <-sleepTimer.C:
			case <-ctx.Done():
				return
			}

			var confirmed bool

			for statusCheck := 0; statusCheck < 15; statusCheck++ {
				if _, err = util.GetClerkEventRecord(i, rl.cliCtx.Codec); err == nil {
					rl.Logger.Info("Self-healing: stateId found on Heimdall after processing", "stateId", i)
					confirmed = true
					break
				}
				rl.Logger.Info("Self-healing: stateId not yet found on Heimdall; retrying", "stateId", i)
				sleepTimer.Reset(1 * time.Second)

				select {
				case <-sleepTimer.C:
				case <-ctx.Done():
					return
				}
			}

			if confirmed {
				synced = true
				break
			}

			rl.Logger.Warn("Self-healing: stateId not confirmed after polling; will retry", "stateId", i, "attempt", attempt+1)
		}

		if !synced {
			rl.Logger.Error("Self-healing: giving up on stateId after max retries; moving to next", "stateId", i, "maxRetries", maxRetriesPerState)
		}
	}
}

func (rl *RootChainListener) processEvent(ctx context.Context, vLog *types.Log) (bool, error) {
	blockTime, err := rl.contractCaller.GetMainChainBlockTime(ctx, vLog.BlockNumber)
	if err != nil {
		rl.Logger.Error("Self-healing: failed to get block time", "error", err)
		return false, err
	}

	if time.Since(blockTime) < helper.GetConfig().SHMaxDepthDuration {
		rl.Logger.Info("Self-healing: block time is less than the max time depth; skipping event")
		return true, err
	}

	topic := vLog.Topics[0].Bytes()
	for _, abiObject := range rl.abis {
		selectedEvent := helper.EventByID(abiObject, topic)
		if selectedEvent == nil {
			continue
		}

		rl.handleLog(*vLog, selectedEvent)
	}

	return false, nil
}
