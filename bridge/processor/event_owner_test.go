package processor

import (
	"context"
	"testing"
	"time"

	"cosmossdk.io/log"
	"github.com/RichardKnop/machinery/v1/tasks"
	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/bridge/util"
	"github.com/0xPolygon/heimdall-v2/helper"
	clerktypes "github.com/0xPolygon/heimdall-v2/x/clerk/types"
)

func useLiveEvents(t *testing.T) *eventTasks {
	t.Helper()
	previous := liveEvents
	liveEvents = &eventTasks{tasks: map[string]*eventTask{}}
	t.Cleanup(func() { liveEvents = previous })
	return liveEvents
}

func eventTaskContext(t *testing.T, uuid, logBytes string) context.Context {
	t.Helper()
	signature := &tasks.Signature{UUID: uuid, Name: "sendStateSyncedToHeimdall", Args: []tasks.Arg{
		{Type: "string", Value: "StateSynced"}, {Type: "string", Value: logBytes},
	}}
	task, err := tasks.NewWithSignature(func(context.Context) error { return nil }, signature)
	require.NoError(t, err)
	return task.Context
}

// Self-heal re-enqueues a missing event under a new UUID; only the task that owns the event keeps retrying.
func TestRetryKeepsOneTaskPerEvent(t *testing.T) {
	events := useLiveEvents(t)
	owner, duplicate := eventTaskContext(t, "a", "log-1"), eventTaskContext(t, "b", "log-1")
	other := eventTaskContext(t, "c", "log-2")
	var later tasks.ErrRetryTaskLater

	require.ErrorAs(t, retryEvent(owner, "waiting", util.RetryStateSyncTaskDelay), &later)
	require.NoError(t, retryEvent(duplicate, "waiting", util.RetryStateSyncTaskDelay))
	require.NoError(t, retryEventEvery(duplicate, "waiting", util.StakeNonceRetryDelay))
	require.ErrorAs(t, retryEventEvery(owner, "waiting", util.StakeNonceRetryDelay), &later)
	require.ErrorAs(t, retryEvent(other, "waiting", util.RetryStateSyncTaskDelay), &later)

	require.False(t, events.claim(duplicate, time.Now().Add(eventOwnerTTL-time.Second)))
	require.True(t, events.claim(duplicate, time.Now().Add(eventOwnerTTL+time.Second)))
	require.ErrorAs(t, retryEvent(context.Background(), "waiting", util.RetryStateSyncTaskDelay), &later)
}

// An unapplied event is resent once per rebroadcastDelay, never by a task that doesn't own it.
func TestSubmitEventResendsOncePerDelay(t *testing.T) {
	events := useLiveEvents(t)
	bp := BaseProcessor{Logger: log.NewNopLogger(), cliCtx: client.Context{}.WithCodec(eventTestCodec())}
	broadcaster := &broadcastResult{response: &sdk.TxResponse{}}
	bp.txBroadcaster = broadcaster
	owner, duplicate := eventTaskContext(t, "a", "log-1"), eventTaskContext(t, "b", "log-1")
	msg := &clerktypes.MsgEventRecord{Id: 1}

	require.NoError(t, bp.submitEvent(owner, msg, nil))
	require.Equal(t, 1, broadcaster.calls)
	require.NoError(t, bp.submitEvent(duplicate, msg, nil))
	require.NoError(t, bp.submitEvent(owner, msg, nil))
	require.Equal(t, 1, broadcaster.calls)

	events.tasks[eventKey(tasks.SignatureFromContext(owner))].broadcastAt = time.Now().Add(-rebroadcastDelay("sendStateSyncedToHeimdall"))
	require.NoError(t, bp.submitEvent(owner, msg, nil))
	require.Equal(t, 2, broadcaster.calls)

	broadcaster.response.Code = 5
	events.tasks[eventKey(tasks.SignatureFromContext(owner))].broadcastAt = time.Now().Add(-rebroadcastDelay("sendStateSyncedToHeimdall"))
	require.Error(t, bp.submitEvent(owner, msg, nil))
	broadcaster.response.Code = 0
	require.NoError(t, bp.submitEvent(owner, msg, nil))
	require.Equal(t, 4, broadcaster.calls, "a rejected broadcast is retried at once")
}

// Each event is resent at the cadence of the self-heal pass that used to resend it.
func TestRebroadcastDelayFollowsSelfHealInterval(t *testing.T) {
	previous := helper.GetConfig()
	t.Cleanup(func() { helper.SetTestConfig(helper.CustomAppConfig{Custom: previous}) })

	helper.SetTestConfig(helper.CustomAppConfig{})
	require.Equal(t, helper.DefaultSHCheckpointAckInterval, rebroadcastDelay("sendCheckpointAckToHeimdall"))
	require.Equal(t, helper.DefaultSHStateSyncedInterval, rebroadcastDelay("sendStateSyncedToHeimdall"))
	require.Equal(t, helper.DefaultSHStakeUpdateInterval, rebroadcastDelay("sendTopUpFeeToHeimdall"))

	helper.SetTestConfig(helper.CustomAppConfig{Custom: helper.CustomConfig{
		SHCheckpointAckInterval: time.Minute, SHStateSyncedInterval: 2 * time.Minute, SHStakeUpdateInterval: 3 * time.Minute,
	}})
	require.Equal(t, time.Minute, rebroadcastDelay("sendCheckpointAckToHeimdall"))
	require.Equal(t, 2*time.Minute, rebroadcastDelay("sendStateSyncedToHeimdall"))
	require.Equal(t, 3*time.Minute, rebroadcastDelay("sendStakeUpdateToHeimdall"))
}
