package processor

import (
	"context"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/0xPolygon/heimdall-v2/bridge/util"
)

// Stake events apply one validator nonce at a time, so they poll on the nonce gap rather than on time
// waited: a long backlog then keeps draining as fast as heimdall applies it. A zero gap means heimdall
// already used the event's nonce, so the side handler can never apply it (e.g. after an L1 reorg moved
// the log) and the task ends instead of polling forever.
func (sp *StakingProcessor) retryOutOfOrderNonce(ctx context.Context, nonceGap uint64) error {
	if nonceGap == 0 {
		return nil
	}
	delay := util.StakeNonceRetryDelay * time.Duration(min(nonceGap, 10))
	return retryEventEvery(ctx, msgNonceOutOfOrder, delay)
}

func (sp *StakingProcessor) broadcastStakeEvent(ctx context.Context, msg sdk.Msg, event interface{}) error {
	if err := sp.submitEvent(ctx, msg, event); err != nil {
		return err
	}
	return retryEventEvery(ctx, "waiting for staking event", util.StakeNonceRetryDelay)
}
