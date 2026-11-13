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
// Past maxStakeNonceRetryAge it returns nil, which ends the task.
func (sp *StakingProcessor) retryOutOfOrderNonce(ctx context.Context, nonceDelay uint64) error {
	waited, expired := retryAgeExpired(ctx, nonceRetryStartHeader, maxStakeNonceRetryAge, time.Now())
	if expired {
		signature := tasks.SignatureFromContext(ctx)
		metrics.StakeNonceRetriesDropped.Inc()
		sp.Logger.Error(infoMsgDroppingNonceRetry, "task", signature.Name, "taskUUID", signature.UUID, "waited", waited.String())
		return nil
	}

	return tasks.NewErrRetryTaskLater(msgNonceOutOfOrder, util.StakeNonceRetryDelay*time.Duration(nonceDelay))
}

// retryAgeExpired reports whether a task has been deferred for longer than
// maxAge. machinery does not count ErrRetryTaskLater against RetryCount, so
// without this bound a deferred task would retry forever. The first call stamps
// the start time under header; a context without a signature never expires.
func retryAgeExpired(ctx context.Context, header string, maxAge time.Duration, now time.Time) (time.Duration, bool) {
	signature := tasks.SignatureFromContext(ctx)
	if signature == nil {
		return 0, false
	}

	start, ok := retryStart(signature.Headers, header)
	if !ok {
		if signature.Headers == nil {
			signature.Headers = tasks.Headers{}
		}
		signature.Headers[header] = strconv.FormatInt(now.Unix(), 10)
		return 0, false
	}

	waited := now.Sub(start)
	return waited, waited > maxAge
}

func retryStart(headers tasks.Headers, header string) (time.Time, bool) {
	raw, _ := headers[header].(string)
	unix, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return time.Time{}, false
	}
	return time.Unix(unix, 0), true
}
