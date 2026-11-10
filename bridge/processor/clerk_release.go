package processor

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/RichardKnop/machinery/v1/tasks"

	"github.com/0xPolygon/heimdall-v2/bridge/util"
	"github.com/0xPolygon/heimdall-v2/metrics"
)

// awaitStored decides what a task whose record isn't stored yet does next,
// once the record entered the mempool (see settleAccepted): return a
// retry-later error to keep waiting, or nil to send the record through the
// busy and duplicate checks again. The check comes clerkMaxStoreWait after the
// record was accepted; a record heimdall still hasn't stored then is broadcast
// again, unless its ID is above the latest one on L1.
func (cp *ClerkProcessor) awaitStored(ctx context.Context, eventName string, id uint64, origin string, missing bool, now time.Time) error {
	signature := tasks.SignatureFromContext(ctx)
	if signature == nil {
		return nil
	}
	acceptedAt, ok := retryStart(signature.Headers, acceptedAtHeader)
	if !ok || acceptedAt.After(now) {
		return nil
	}
	if waited := now.Sub(acceptedAt); waited < clerkMaxStoreWait {
		return cp.waitForStore(eventName, id, acceptedAt, wholeSeconds(clerkMaxStoreWait-waited))
	}
	if err := cp.abandonUnstored(signature, eventName, id, origin, missing, acceptedAt, now); err != nil {
		return err
	}
	if err := cp.holdIfNotOnL1(eventName, id, now); err != nil {
		return err
	}
	if accepted := headerString(signature.Headers, acceptedAtHeader); headerString(signature.Headers, unstoredHeader) != accepted {
		signature.Headers[unstoredHeader] = accepted
		metrics.ClerkRecordsUnstored.Inc()
		cp.Logger.Error(errMsgClerkUnstored, "event", eventName, "id", id, "acceptedAt", acceptedAt)
	}
	return nil
}

// errRecordAbandoned ends a task whose record heimdall still hasn't stored
// clerkMaxReleasedAge after it first entered the mempool.
var errRecordAbandoned = errors.New("state sync record abandoned")

// abandonUnstored returns errRecordAbandoned once the record is
// clerkMaxReleasedAge past its first accept and this run's lookup found it
// missing; past that age with an unreadable lookup, the task asks again
// shortly instead. Tasks accepted before firstAcceptedAtHeader existed count
// from their last accept.
func (cp *ClerkProcessor) abandonUnstored(signature *tasks.Signature, eventName string, id uint64, origin string, missing bool, acceptedAt, now time.Time) error {
	first, ok := retryStart(signature.Headers, firstAcceptedAtHeader)
	if !ok {
		first = acceptedAt
	}
	if now.Sub(first) < clerkMaxReleasedAge {
		return nil
	}
	if !missing {
		cp.Logger.Warn(warnMsgClerkUnreadableAtCap, "event", eventName, "id", id)
		return tasks.NewErrRetryTaskLater(msgRecordUnreadable, abandonRetryDelay(id))
	}
	// Recorded before the lease goes: a standby that finds the lease gone
	// then also finds this (see admit).
	cp.released.abandoned.add(id, origin, now)
	cp.released.doneOrigin(id, origin)
	metrics.ClerkTasksExhausted.Inc()
	cp.Logger.Error(errMsgClerkAbandoned, "event", eventName, "id", id, "firstAcceptedAt", first)
	return errRecordAbandoned
}

// abandonRetryDelay is how soon a task past clerkMaxReleasedAge asks again
// after a lookup that didn't say whether its record is stored: on the
// released cadence, so a heimdall API outage doesn't cycle a burst's tasks
// through the queue.
func abandonRetryDelay(id uint64) time.Duration {
	return wholeSeconds(releasedRetryDelay + spread(id, releasedRetryDelay))
}

// settleAccepted ends the task's hold on the order window once its record is
// in the mempool, broadcast by this bridge or by another one: heimdall stores
// it from there, so the window moves at the pace records are accepted, not
// stored. The task checks back clerkMaxStoreWait later and broadcasts again a
// record heimdall dropped (see awaitStored).
// own is false when the record is in the mempool through another broadcast:
// only this bridge's own accepted tx starts the clerkMaxReleasedAge clock.
func (cp *ClerkProcessor) settleAccepted(ctx context.Context, id uint64, origin string, own bool, now time.Time) error {
	cp.leavePending(id, taskOwner(ctx))
	cp.released.touch(id, origin, now)
	if signature := tasks.SignatureFromContext(ctx); signature != nil {
		if signature.Headers == nil {
			signature.Headers = tasks.Headers{}
		}
		signature.Headers[windowReleasedHeader] = headerTrue
		signature.Headers[acceptedAtHeader] = strconv.FormatInt(now.Unix(), 10)
		if _, ok := retryStart(signature.Headers, firstAcceptedAtHeader); own && !ok {
			signature.Headers[firstAcceptedAtHeader] = signature.Headers[acceptedAtHeader]
		}
		signature.RetryCount = max(signature.RetryCount, clerkTaskRetries)
		// machinery grows RetryTimeout along Fibonacci on each counted
		// failure; left alone it would outgrow pendingRecordTTL.
		signature.RetryTimeout = 0
	}
	return tasks.NewErrRetryTaskLater(msgAwaitingStore, clerkMaxStoreWait)
}

// cachedCounter shares a read of the L1 state sync counter: a success for
// latestOnL1TTL, a failure for latestOnL1FailTTL.
type cachedCounter struct {
	mu    sync.Mutex
	at    time.Time
	value uint64
	err   error
	read  func() (uint64, error)
}

var errCounterReadInProgress = errors.New("L1 state sync counter read in progress")

// latestRecordIDOnL1 reads the StateSender counter through heimdall. A task
// that finds another read in progress doesn't wait for it: the counter counts
// as unreadable, which never holds a rebroadcast.
func (cp *ClerkProcessor) latestRecordIDOnL1(now time.Time) (uint64, error) {
	c := &cp.latestOnL1
	if !c.mu.TryLock() {
		return 0, errCounterReadInProgress
	}
	defer c.mu.Unlock()

	ttl := latestOnL1TTL
	if c.err != nil {
		ttl = latestOnL1FailTTL
	}
	if !c.at.IsZero() && now.Sub(c.at) < ttl {
		return c.value, c.err
	}

	read := c.read
	if read == nil {
		read = func() (uint64, error) { return util.GetLatestRecordID(cp.cliCtx.Codec) }
	}
	started := time.Now()
	c.value, c.err = read()
	// Stamped when the read returns, so a slow failed read is still shared.
	c.at = now.Add(time.Since(started))
	return c.value, c.err
}

// holdIfNotOnL1 skips a released task's rebroadcast for clerkMaxStoreWait
// while its ID is above the latest one on L1: an event L1 doesn't have yet
// can't keep a task broadcasting. The task stays alive, since one counter read from a lagging
// L1 node is no proof the event doesn't exist; a failed read doesn't hold it.
func (cp *ClerkProcessor) holdIfNotOnL1(eventName string, id uint64, now time.Time) error {
	latest, err := cp.latestRecordIDOnL1(now)
	if err == nil && id > latest {
		metrics.ClerkRecordsNotOnL1.Inc()
		cp.Logger.Warn(warnMsgClerkNotOnL1, "event", eventName, "id", id, "latestOnL1", latest)
		return tasks.NewErrRetryTaskLater(msgNotOnL1, clerkMaxStoreWait)
	}
	return nil
}

func (cp *ClerkProcessor) waitForStore(eventName string, id uint64, acceptedAt time.Time, delay time.Duration) error {
	cp.Logger.Debug(debugMsgClerkAwaitingStore, "event", eventName, "id", id, "acceptedAt", acceptedAt)
	return tasks.NewErrRetryTaskLater(msgAwaitingStore, delay)
}

// storeCheckFailed reports whether the task's record entered the mempool and a
// store check found it missing since (see awaitStored, which lets only such a
// task through).
func storeCheckFailed(signature *tasks.Signature) bool {
	if signature == nil {
		return false
	}
	_, ok := retryStart(signature.Headers, acceptedAtHeader)
	return ok
}

// windowReleased reports whether the task stopped holding the order window.
func windowReleased(signature *tasks.Signature) bool {
	return signature != nil && headerString(signature.Headers, windowReleasedHeader) == headerTrue
}

func setHeader(signature *tasks.Signature, key string) {
	if signature == nil {
		return
	}
	if signature.Headers == nil {
		signature.Headers = tasks.Headers{}
	}
	signature.Headers[key] = headerTrue
}

func headerString(headers tasks.Headers, key string) string {
	v, _ := headers[key].(string)
	return v
}

// waitOnMempoolCopy settles a task whose record another broadcast already put
// in the mempool: an identical copy is the same event, stored when included.
func (cp *ClerkProcessor) waitOnMempoolCopy(ctx context.Context, eventName string, id uint64, origin string, now time.Time) error {
	cp.Logger.Info(infoMsgClerkTxInMempool, "event", eventName, "id", id, "retry delay", clerkMaxStoreWait)
	return cp.settleAccepted(ctx, id, origin, false, now)
}
