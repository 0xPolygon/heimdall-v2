package listener

import (
	"context"
	"strconv"
	"time"

	"github.com/0xPolygon/heimdall-v2/bridge/util"
	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/metrics"
	staketypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

// stakeNonceStuck reports whether heimdall already records the event for
// nextNonce as processed even though the validator's nonce never reached it.
// Heimdall then rejects that event as old and every later one as out of order,
// so replaying further nonces only queues bridge tasks that can never land.
// Lookup failures return false and leave error handling to replayStakeEvent.
func (rl *RootChainListener) stakeNonceStuck(ctx context.Context, id, nextNonce uint64) bool {
	var hit *txAndLogIndex
	var err error
	if err = helper.ExponentialBackoff(func() error {
		hit, err = rl.getStakeEventRefByNonce(ctx, id, nextNonce)
		return err
	}, 3, time.Second); err != nil {
		return false
	}

	logIndex, err := strconv.ParseUint(hit.LogIndex, 10, 64)
	if err != nil {
		rl.Logger.Error("Self-healing: invalid log index for stake event", "validatorId", id, "nonce", nextNonce, "logIndex", hit.LogIndex, "error", err)
		return false
	}

	processed, err := rl.isStakeEventProcessed(hit.TransactionHash, logIndex)
	if err != nil {
		rl.Logger.Error("Self-healing: failed to check whether stake event is processed", "validatorId", id, "nonce", nextNonce, "txHash", hit.TransactionHash, "error", err)
		return false
	}
	if !processed {
		return false
	}

	metrics.SelfHealStakeNonceStuck.Inc()
	rl.Logger.Error("Self-healing: validator nonce is stuck; the next stake event is already processed but the nonce did not advance; skipping validator",
		"validatorId", id, "heimdallNonce", nextNonce-1, "nextNonce", nextNonce, "txHash", hit.TransactionHash, "logIndex", logIndex)
	return true
}

func (rl *RootChainListener) isStakeEventProcessed(txHash string, logIndex uint64) (bool, error) {
	url, err := util.CreateURLWithQuery(helper.GetHeimdallServerEndpoint(util.StakingTxStatusURL), map[string]any{
		"tx_hash":   txHash,
		"log_index": logIndex,
	})
	if err != nil {
		return false, err
	}

	res, err := helper.FetchFromAPI(url)
	if err != nil {
		return false, err
	}

	var response staketypes.QueryStakeIsOldTxResponse
	if err := rl.cliCtx.Codec.UnmarshalJSON(res, &response); err != nil {
		return false, err
	}
	return response.IsOld, nil
}
