package processor

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/RichardKnop/machinery/v1/tasks"
	promtestutil "github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/bridge/util"
	"github.com/0xPolygon/heimdall-v2/metrics"
)

func TestEventRetriesRemainPending(t *testing.T) {
	for _, age := range []time.Duration{0, time.Hour, 48 * time.Hour} {
		t.Run(age.String(), func(t *testing.T) {
			signature := &tasks.Signature{RetryCount: 0, Headers: tasks.Headers{
				eventRetryStartHeader: strconv.FormatInt(time.Now().Add(-age).Unix(), 10),
			}}
			task, err := tasks.NewWithSignature(func(context.Context) error { return nil }, signature)
			require.NoError(t, err)
			failure := errors.New("temporary API outage")
			retryEventFailure(task.Context, &failure)
			var later tasks.ErrRetryTaskLater
			require.ErrorAs(t, failure, &later)
			require.GreaterOrEqual(t, later.RetryIn(), util.RetryStateSyncTaskDelay)
			require.LessOrEqual(t, later.RetryIn(), 16*util.RetryStateSyncTaskDelay)
			require.Equal(t, 0, signature.RetryCount)
		})
	}
}

func TestEventRetryHeaders(t *testing.T) {
	for _, start := range []interface{}{nil, "bad", 42, strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10)} {
		signature := &tasks.Signature{Headers: tasks.Headers{eventRetryStartHeader: start}}
		task, err := tasks.NewWithSignature(func(context.Context) error { return nil }, signature)
		require.NoError(t, err)
		var later tasks.ErrRetryTaskLater
		require.ErrorAs(t, retryEvent(task.Context, "pending", time.Second), &later)
		stamped, err := strconv.ParseInt(signature.Headers[eventRetryStartHeader].(string), 10, 64)
		require.NoError(t, err)
		require.WithinDuration(t, time.Now(), time.Unix(stamped, 0), time.Second)
	}
}

func TestNonceRetryWithoutSignature(t *testing.T) {
	sp := new(StakingProcessor)
	require.NoError(t, sp.retryOutOfOrderNonce(context.Background(), 0))
	var later tasks.ErrRetryTaskLater
	for _, count := range []uint64{1, 3, 10, 11, 1000} {
		require.ErrorAs(t, sp.retryOutOfOrderNonce(context.Background(), count), &later)
		require.Equal(t, time.Duration(min(count, 10))*util.StakeNonceRetryDelay, later.RetryIn())
	}
}

// An out-of-order stake task polls on the nonce gap however long it has waited, keeps its start stamp
// and still counts toward the self-heal depth.
func TestNonceRetryIgnoresTimeWaited(t *testing.T) {
	start := strconv.FormatInt(time.Now().Add(-48*time.Hour).Unix(), 10)
	for _, gap := range []uint64{1, 20} {
		signature := &tasks.Signature{Headers: tasks.Headers{eventRetryStartHeader: start}}
		task, err := tasks.NewWithSignature(func(context.Context) error { return nil }, signature)
		require.NoError(t, err)
		before := promtestutil.ToFloat64(metrics.EventRetriesPastSelfHealDepth)
		var later tasks.ErrRetryTaskLater
		require.ErrorAs(t, new(StakingProcessor).retryOutOfOrderNonce(task.Context, gap), &later)
		require.Equal(t, time.Duration(min(gap, 10))*util.StakeNonceRetryDelay, later.RetryIn())
		require.Equal(t, start, signature.Headers[eventRetryStartHeader])
		require.Equal(t, before+1, promtestutil.ToFloat64(metrics.EventRetriesPastSelfHealDepth))
	}
}
