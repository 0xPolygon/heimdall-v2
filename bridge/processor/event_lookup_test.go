package processor

import (
	"context"
	"net/http"
	"testing"

	"github.com/RichardKnop/machinery/v1/tasks"
	"github.com/stretchr/testify/require"
)

// A failed "already applied?" lookup retries without broadcasting; only an authoritative "not applied"
// broadcasts, and an applied event completes the task.
func TestEventTasksBroadcastOnlyOnAuthoritativeLookup(t *testing.T) {
	const unavailable, internal = http.StatusServiceUnavailable, http.StatusInternalServerError
	tests := []struct {
		name      string
		event     string
		status    map[string]int
		ackCount  uint64
		applied   bool
		broadcast bool
		done      bool
	}{
		{name: "topup status fails", event: "TopUpFee", status: map[string]int{"/topup/is-old-tx": unavailable}},
		{name: "topup not applied", event: "TopUpFee", broadcast: true},
		{name: "ack latest and count fail", event: "NewHeaderBlock", status: map[string]int{"/checkpoints/latest": internal, "/checkpoints/count": unavailable}},
		{name: "ack latest fails with acks", event: "NewHeaderBlock", status: map[string]int{"/checkpoints/latest": internal}, ackCount: 3},
		{name: "ack before first checkpoint", event: "NewHeaderBlock", status: map[string]int{"/checkpoints/latest": internal}, broadcast: true},
		{name: "record and status fail", event: "StateSynced", status: map[string]int{"/clerk/": internal}},
		{name: "record missing, not applied", event: "StateSynced", status: map[string]int{"/clerk/event-records/": internal}, broadcast: true},
		{name: "record missing, applied", event: "StateSynced", status: map[string]int{"/clerk/event-records/": internal}, applied: true, done: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := newEventTaskFixture(t)
			f.status, f.ackCount = tc.status, tc.ackCount
			f.applied.Store(tc.applied)
			run, payload := f.lookupTask(t, tc.event)
			err := run(context.Background(), tc.event, payload)
			if tc.done {
				require.NoError(t, err)
			} else {
				var later tasks.ErrRetryTaskLater
				require.ErrorAs(t, err, &later)
			}
			require.Equal(t, map[bool]int{true: 1}[tc.broadcast], f.broadcast.calls)
		})
	}
}

func (f *eventTaskFixture) lookupTask(t *testing.T, event string) (func(context.Context, string, string) error, string) {
	t.Helper()
	switch event {
	case "NewHeaderBlock":
		return f.chp.sendCheckpointAckToHeimdall, checkpointAckLog(t, f.chp.rootChainAbi)
	case "StateSynced":
		payload, err := prepareDummyLogBytes()
		require.NoError(t, err)
		return f.cp.sendStateSyncedToHeimdall, payload.String()
	default:
		return f.fp.sendTopUpFeeToHeimdall, stakeEventLog(t, f.fp.stakingInfoAbi, event, 10, 5)
	}
}
