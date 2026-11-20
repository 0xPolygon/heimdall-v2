package processor

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/RichardKnop/machinery/v1/tasks"

	"github.com/0xPolygon/heimdall-v2/bridge/util"
	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/metrics"
)

const eventRetryStartHeader = "eventRetryStart"

// retryEventFailure keeps API and broadcast failures on the same durable path as
// confirmation polling. Machinery otherwise exhausts RetryCount and deletes the task.
func retryEventFailure(ctx context.Context, result *error) {
	if *result == nil {
		return
	}
	var later tasks.ErrRetryTaskLater
	if errors.As(*result, &later) {
		*result = later
		return
	}
	*result = retryEvent(ctx, (*result).Error(), util.RetryStateSyncTaskDelay)
}

// retryEvent ends a task whose event another live task owns, so self-heal re-enqueues don't pile up.
func retryEvent(ctx context.Context, reason string, minimum time.Duration) error {
	if !liveEvents.claim(ctx, time.Now()) {
		return nil
	}
	// The broker keeps one delay queue per distinct delay, so the backoff grows in whole steps.
	delay := max(minimum, trackEventRetry(ctx).Truncate(util.RetryStateSyncTaskDelay))
	return tasks.NewErrRetryTaskLater(reason, min(delay, 16*util.RetryStateSyncTaskDelay))
}

// retryEventEvery retries after a fixed delay, for waits whose progress doesn't depend on time waited.
func retryEventEvery(ctx context.Context, reason string, delay time.Duration) error {
	if !liveEvents.claim(ctx, time.Now()) {
		return nil
	}
	trackEventRetry(ctx)
	return tasks.NewErrRetryTaskLater(reason, delay)
}

// trackEventRetry returns how long the task has been retrying, stamping its first retry and counting
// retries past the self-heal depth. The serialized header survives worker restarts, without an age cutoff.
func trackEventRetry(ctx context.Context) time.Duration {
	signature := tasks.SignatureFromContext(ctx)
	if signature == nil {
		return 0
	}
	now := time.Now()
	raw, _ := signature.Headers[eventRetryStartHeader].(string)
	unix, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || unix <= 0 || unix > now.Unix() {
		if signature.Headers == nil {
			signature.Headers = tasks.Headers{}
		}
		signature.Headers[eventRetryStartHeader] = strconv.FormatInt(now.Unix(), 10)
		return 0
	}
	waited := now.Sub(time.Unix(unix, 0))
	if waited > helper.DefaultSHMaxDepthDuration {
		metrics.EventRetriesPastSelfHealDepth.Inc()
	}
	return waited
}

// retryStart reads the unix time a task stamped under header on an earlier run.
func retryStart(headers tasks.Headers, header string) (time.Time, bool) {
	raw, _ := headers[header].(string)
	unix, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return time.Time{}, false
	}
	return time.Unix(unix, 0), true
}
