package processor

import (
	"context"
	"strconv"
	"time"

	"github.com/RichardKnop/machinery/v1/tasks"

	"github.com/0xPolygon/heimdall-v2/bridge/util"
	"github.com/0xPolygon/heimdall-v2/metrics"
)

const (
	// nonceRetryStartHeader holds the unix time of a task's first out-of-order
	// attempt. machinery republishes the same signature on retry, so it persists.
	nonceRetryStartHeader = "stakeNonceRetryStart"

	// maxStakeNonceRetryAge bounds how long a stake task waits for heimdall to
	// reach its nonce. A normal gap closes within minutes. It is set well above
	// the default SHStakeUpdateInterval, so self-heal re-queues any event that
	// is still missing once heimdall catches up.
	maxStakeNonceRetryAge = 24 * time.Hour

	infoMsgDroppingNonceRetry = "StakingProcessor: dropping task after retrying out-of-order nonce past the age limit"
)

// retryOutOfOrderNonce defers a stake task whose nonce is ahead of heimdall's.
// machinery does not count ErrRetryTaskLater against RetryCount, so without an
// age limit a task for a nonce heimdall never reaches would retry forever.
// Past the limit it returns nil, which ends the task.
func (sp *StakingProcessor) retryOutOfOrderNonce(ctx context.Context, nonceDelay uint64) error {
	retryLater := tasks.NewErrRetryTaskLater(msgNonceOutOfOrder, util.StakeNonceRetryDelay*time.Duration(nonceDelay))

	signature := tasks.SignatureFromContext(ctx)
	if signature == nil {
		return retryLater
	}

	now := time.Now()
	start, ok := nonceRetryStart(signature.Headers)
	if !ok {
		if signature.Headers == nil {
			signature.Headers = tasks.Headers{}
		}
		signature.Headers[nonceRetryStartHeader] = strconv.FormatInt(now.Unix(), 10)
		return retryLater
	}

	if waited := now.Sub(start); waited > maxStakeNonceRetryAge {
		metrics.StakeNonceRetriesDropped.Inc()
		sp.Logger.Error(infoMsgDroppingNonceRetry, "task", signature.Name, "taskUUID", signature.UUID, "waited", waited.String())
		return nil
	}

	return retryLater
}

func nonceRetryStart(headers tasks.Headers) (time.Time, bool) {
	raw, _ := headers[nonceRetryStartHeader].(string)
	unix, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return time.Time{}, false
	}
	return time.Unix(unix, 0), true
}
