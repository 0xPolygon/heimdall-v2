package processor

import (
	"cmp"
	"context"
	"crypto/sha256"
	"fmt"
	"sync"
	"time"

	"github.com/RichardKnop/machinery/v1/tasks"

	"github.com/0xPolygon/heimdall-v2/bridge/util"
	"github.com/0xPolygon/heimdall-v2/helper"
)

const (
	// eventOwnerTTL is how long an event's task may go unseen before another task for the event takes
	// over. A live task runs at least every 16*RetryStateSyncTaskDelay.
	eventOwnerTTL = 2 * 16 * util.RetryStateSyncTaskDelay
)

// rebroadcastDelay spaces broadcasts of an event that is still unapplied. The bridge can't tell a
// committed tx whose side tx failed from a dropped one without the tx indexer, which the shipped
// configs disable, so either is resent once per pass of the self-heal that used to resend it.
func rebroadcastDelay(taskName string) time.Duration {
	conf := helper.GetConfig()
	switch taskName {
	case "sendCheckpointAckToHeimdall":
		return cmp.Or(conf.SHCheckpointAckInterval, helper.DefaultSHCheckpointAckInterval)
	case "sendStateSyncedToHeimdall":
		return cmp.Or(conf.SHStateSyncedInterval, helper.DefaultSHStateSyncedInterval)
	default:
		return cmp.Or(conf.SHStakeUpdateInterval, helper.DefaultSHStakeUpdateInterval)
	}
}

type eventTask struct {
	owner       string
	seen        time.Time
	broadcastAt time.Time
}

// eventTasks keeps one live task per L1 event in this bridge. Self-heal re-enqueues a still-missing
// event with a new signature on every pass, and a task only ends once its event is applied.
type eventTasks struct {
	mu       sync.Mutex
	tasks    map[string]*eventTask
	prunedAt time.Time
}

var liveEvents = &eventTasks{tasks: map[string]*eventTask{}}

// eventKey identifies an event by its task name and arguments, which self-heal reproduces exactly.
func eventKey(signature *tasks.Signature) string {
	h := sha256.New()
	h.Write([]byte(signature.Name))
	for _, arg := range signature.Args {
		_, _ = fmt.Fprintf(h, "\x00%v", arg.Value)
	}
	return string(h.Sum(nil))
}

// claim reports whether the task in ctx owns its event, taking over an event whose owner went unseen.
func (e *eventTasks) claim(ctx context.Context, now time.Time) bool {
	signature := tasks.SignatureFromContext(ctx)
	if signature == nil {
		return true
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.prune(now)
	key := eventKey(signature)
	task, ok := e.tasks[key]
	if !ok {
		task = &eventTask{}
		e.tasks[key] = task
	}
	if task.owner != "" && task.owner != signature.UUID && now.Sub(task.seen) < eventOwnerTTL {
		return false
	}
	task.owner, task.seen = signature.UUID, now
	return true
}

// prune drops events whose owner went unseen, at most once per eventOwnerTTL. A live owner refreshes
// its entry on every run, so its last broadcast is kept.
func (e *eventTasks) prune(now time.Time) {
	if now.Sub(e.prunedAt) < eventOwnerTTL {
		return
	}
	e.prunedAt = now
	for key, task := range e.tasks {
		if now.Sub(task.seen) > eventOwnerTTL {
			delete(e.tasks, key)
		}
	}
}

// broadcastDue reports whether the event in ctx has no broadcast within rebroadcastDelay.
func (e *eventTasks) broadcastDue(ctx context.Context, now time.Time) bool {
	signature := tasks.SignatureFromContext(ctx)
	if signature == nil {
		return true
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	task, ok := e.tasks[eventKey(signature)]
	return !ok || task.broadcastAt.IsZero() || now.Sub(task.broadcastAt) >= rebroadcastDelay(signature.Name)
}

func (e *eventTasks) recordBroadcast(ctx context.Context, now time.Time) {
	signature := tasks.SignatureFromContext(ctx)
	if signature == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if task, ok := e.tasks[eventKey(signature)]; ok {
		task.broadcastAt = now
	}
}
