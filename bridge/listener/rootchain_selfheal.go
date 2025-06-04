package listener

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/0xPolygon/heimdall-v2/bridge/util"
	"github.com/0xPolygon/heimdall-v2/helper"
)

var (
	stateSyncedCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "self_healing",
		Subsystem: helper.GetConfig().Chain,
		Name:      "StateSynced",
		Help:      "The total number of missing StateSynced events",
	}, []string{"id", "contract_address", "block_number", "tx_hash"})

	stakeUpdateCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "self_healing",
		Subsystem: helper.GetConfig().Chain,
		Name:      "StakeUpdate",
		Help:      "The total number of missing StakeUpdate events",
	}, []string{"id", "nonce", "contract_address", "block_number", "tx_hash"})
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

	stakeUpdateTicker := time.NewTicker(helper.GetConfig().SHStakeUpdateInterval)
	stateSyncedTicker := time.NewTicker(helper.GetConfig().SHStateSyncedInterval)

	rl.Logger.Info("Started self-healing")

	for {
		select {
		case <-stakeUpdateTicker.C:
			rl.processStakeUpdate(ctx)
		case <-stateSyncedTicker.C:
			rl.processStateSynced(ctx)
		case <-ctx.Done():
			rl.Logger.Info("Stopping self-healing")
			stakeUpdateTicker.Stop()
			stateSyncedTicker.Stop()

			return
		}
	}
}

// processStakeUpdate checks if validators are in sync, otherwise syncs them by broadcasting missing events
func (rl *RootChainListener) processStakeUpdate(ctx context.Context) {
	// Fetch all heimdall validators
	validatorSet, err := util.GetValidatorSet(rl.cliCtx.Codec)
	if err != nil {
		rl.Logger.Error("Failed to fetch validator set from Heimdall", "error", err)
		return
	}

	rl.Logger.Info("Fetched validator list from Heimdall", "validatorCount", len(validatorSet.Validators))

	// Make sure each validator is in sync
	var wg sync.WaitGroup
	for _, validator := range validatorSet.Validators {
		wg.Add(1)

		go func(id uint64) {
			defer wg.Done()

			nonce, err := util.GetValidatorNonce(id, rl.cliCtx.Codec)
			if err != nil {
				rl.Logger.Error("Failed to fetch nonce for validator from Heimdall", "validatorId", id, "error", err)
				return
			}

			var ethereumNonce uint64

			if err = helper.ExponentialBackoff(func() error {
				ethereumNonce, err = rl.getLatestNonce(ctx, id)
				return err
			}, 3, time.Second); err != nil {
				rl.Logger.Error("Failed to fetch latest nonce from Ethereum (L1) for validator", "validatorId", id, "error", err)
				return
			}
			rl.Logger.Info("Retrieved nonces for validator", "validatorId", id, "ethereumNonce", ethereumNonce, "heimdallNonce", nonce)

			if ethereumNonce <= nonce {
				return
			}

			nonce++

			rl.Logger.Info("Validator is behind; processing missing stake update", "validatorId", id, "ethereumNonce", ethereumNonce, "nextExpectedNonce", nonce)

			var stakeUpdate *types.Log

			if err = helper.ExponentialBackoff(func() error {
				stakeUpdate, err = rl.getStakeUpdate(ctx, id, nonce)
				return err
			}, 3, time.Second); err != nil {
				rl.Logger.Error("Failed to retrieve StakeUpdate event from subgraph", "validatorId", id, "nonce", nonce, "error", err)
				return
			}
			rl.Logger.Info("Fetched StakeUpdate event from Ethereum", "validatorId", id, "nonce", nonce, "blockNumber", stakeUpdate.BlockNumber, "txHash", stakeUpdate.TxHash.Hex())

			stakeUpdateCounter.WithLabelValues(
				fmt.Sprintf("%d", id),
				fmt.Sprintf("%d", nonce),
				stakeUpdate.Address.Hex(),
				fmt.Sprintf("%d", stakeUpdate.BlockNumber),
				stakeUpdate.TxHash.Hex(),
			).Add(1)

			if _, err = rl.processEvent(ctx, stakeUpdate); err != nil {
				rl.Logger.Error("Failed to process StakeUpdate event", "validatorId", id, "nonce", nonce, "error", err)
			} else {
				rl.Logger.Info("Successfully processed StakeUpdate event", "validatorId", id, "nonce", nonce)
			}
		}(validator.ValId)
	}

	wg.Wait()
}

// processStateSynced checks if chains are in sync, otherwise syncs them by broadcasting missing events
func (rl *RootChainListener) processStateSynced(ctx context.Context) {
	latestPolygonStateId, err := rl.getCurrentStateID(ctx)
	if err != nil {
		rl.Logger.Error("Failed to fetch current Polygon stateId from StateReceiver contract", "error", err)
		return
	}

	latestEthereumStateId, err := rl.getLatestStateID(ctx)
	if err != nil {
		rl.Logger.Error("Failed to fetch latest Ethereum stateId from StateSender contract", "error", err)
		return
	}
	rl.Logger.Info("Retrieved latest state IDs", "polygonStateId", latestPolygonStateId, "ethereumStateId", latestEthereumStateId)

	if latestEthereumStateId.Cmp(latestPolygonStateId) != 1 {
		return
	}

	for i := latestPolygonStateId.Int64() + 1; i <= latestEthereumStateId.Int64(); i++ {
		if _, err = util.GetClerkEventRecord(i, rl.cliCtx.Codec); err == nil {
			rl.Logger.Info("State ID already synced on Heimdall; skipping", "stateId", i)
			continue
		}

		rl.Logger.Info("Missing state detected; processing StateSynced event", "stateId", i)

		var stateSynced *types.Log

		if err = helper.ExponentialBackoff(func() error {
			stateSynced, err = rl.getStateSynced(ctx, i)
			return err
		}, 3, time.Second); err != nil {
			rl.Logger.Error("Failed to retrieve StateSynced event for missing state", "stateId", i, "error", err)
			continue
		}

		stateSyncedCounter.WithLabelValues(
			fmt.Sprintf("%d", i),
			stateSynced.Address.Hex(),
			fmt.Sprintf("%d", stateSynced.BlockNumber),
			stateSynced.TxHash.Hex(),
		).Add(1)

		ignore, err := rl.processEvent(ctx, stateSynced)
		if err != nil {
			rl.Logger.Error("Failed to process StateSynced event and update Heimdall", "stateId", i, "error", err)
			i--
			continue
		}

		if !ignore {
			time.Sleep(1 * time.Second)

			var statusCheck int
			for statusCheck = 0; statusCheck < 15; statusCheck++ {
				if _, err = util.GetClerkEventRecord(i, rl.cliCtx.Codec); err == nil {
					rl.Logger.Info("StateId found on Heimdall after processing", "stateId", i)
					break
				}
				rl.Logger.Info("StateId not yet found on Heimdall; retrying", "stateId", i)
				time.Sleep(1 * time.Second)
			}

			if statusCheck >= 15 {
				i--
				continue
			}
		}
	}
}

func (rl *RootChainListener) processEvent(ctx context.Context, vLog *types.Log) (bool, error) {
	blockTime, err := rl.contractCaller.GetMainChainBlockTime(ctx, vLog.BlockNumber)
	if err != nil {
		rl.Logger.Error("Failed to get block time", "error", err)
		return false, err
	}

	if time.Since(blockTime) < helper.GetConfig().SHMaxDepthDuration {
		rl.Logger.Info("Block time is less than the max time depth; skipping event")
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
