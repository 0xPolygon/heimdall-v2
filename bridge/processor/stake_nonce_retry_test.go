package processor

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"cosmossdk.io/log"
	"github.com/RichardKnop/machinery/v1/tasks"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/bridge/util"
)

func TestRetryOutOfOrderNonce(t *testing.T) {
	t.Parallel()

	unixAgo := func(d time.Duration) string { return strconv.FormatInt(time.Now().Add(-d).Unix(), 10) }

	tests := []struct {
		name        string
		signature   *tasks.Signature
		wantRetry   bool
		wantStamped bool
	}{
		{name: "no signature in context retries", signature: nil, wantRetry: true},
		{name: "first attempt stamps start and retries", signature: &tasks.Signature{}, wantRetry: true, wantStamped: true},
		{name: "malformed start is re-stamped", signature: &tasks.Signature{Headers: tasks.Headers{nonceRetryStartHeader: "bogus"}}, wantRetry: true, wantStamped: true},
		{name: "non-string start is re-stamped", signature: &tasks.Signature{Headers: tasks.Headers{nonceRetryStartHeader: 12.0}}, wantRetry: true, wantStamped: true},
		{name: "within age limit retries", signature: &tasks.Signature{Headers: tasks.Headers{nonceRetryStartHeader: unixAgo(time.Hour)}}, wantRetry: true},
		{name: "past age limit drops the task", signature: &tasks.Signature{Headers: tasks.Headers{nonceRetryStartHeader: unixAgo(maxStakeNonceRetryAge + time.Minute)}}, wantRetry: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sp := &StakingProcessor{BaseProcessor: BaseProcessor{Logger: log.NewNopLogger()}}
			ctx := context.Background()
			var before string
			if tc.signature != nil {
				task, err := tasks.NewWithSignature(func(context.Context) error { return nil }, tc.signature)
				require.NoError(t, err)
				ctx = task.Context
				before, _ = tc.signature.Headers[nonceRetryStartHeader].(string)
			}

			err := sp.retryOutOfOrderNonce(ctx, 3)

			if !tc.wantRetry {
				require.NoError(t, err)
				return
			}
			var retryLater tasks.ErrRetryTaskLater
			require.True(t, errors.As(err, &retryLater))
			require.Equal(t, 3*util.StakeNonceRetryDelay, retryLater.RetryIn())

			if tc.signature == nil {
				return
			}
			after, ok := tc.signature.Headers[nonceRetryStartHeader].(string)
			require.True(t, ok)
			if tc.wantStamped {
				start, ok := nonceRetryStart(tc.signature.Headers)
				require.True(t, ok)
				require.WithinDuration(t, time.Now(), start, 5*time.Second)
			} else {
				require.Equal(t, before, after)
			}
		})
	}
}
