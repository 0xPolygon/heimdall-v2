package processor

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/RichardKnop/machinery/v1/tasks"

	"github.com/0xPolygon/heimdall-v2/bridge/broadcaster"
	"github.com/0xPolygon/heimdall-v2/bridge/util"
	"github.com/0xPolygon/heimdall-v2/metrics"
)

const (
	// clerkMempoolBusyTxs is the mempool size at which clerk tasks stop
	// broadcasting and wait. The marks are soft: the size is read through a
	// cache shared for mempoolCountTTL, and for as long as a new read takes, so
	// each bridge's concurrent tasks can overshoot a mark by a few txs per read.
	// Every validator submits every state sync, so
	// without it a burst fills the mempool with copies of the same records,
	// recheck of the full mempool slows heimdall consensus, and a retried
	// record lands behind thousands of newer ones. Heimdall commits about ten
	// clerk txs per block, so this still keeps the next blocks full.
	clerkMempoolBusyTxs = 300

	// clerkOrderWindow bounds how far above the lowest pending state sync ID a
	// task may broadcast. Bor only commits contiguous IDs, so a record stored
	// ahead of a missing lower one waits anyway; holding it back leaves room
	// for the lower one. Heimdall stores about ten clerk records per block, so
	// this is roughly ten blocks of records.
	clerkOrderWindow = 100

	// clerkFrontierIDs is how many of the lowest pending IDs are held to
	// clerkFrontierBusyTxs instead of clerkMempoolBusyTxs, so the record bor is
	// waiting for doesn't queue behind deferred ones.
	clerkFrontierIDs = 25

	// clerkFrontierBusyTxs bounds the mempool for the clerkFrontierIDs lowest
	// pending IDs, which bor needs next, so they rarely queue behind deferred
	// ones while the mempool stays well below its capacity.
	clerkFrontierBusyTxs = 1000

	// clerkLowestIDs is how many of the lowest pending IDs, starting with the
	// one bor waits on, are held only to clerkLowestBusyTxs: about one block of
	// clerk records, so state sync keeps a block's pace under other load.
	clerkLowestIDs = 10

	// clerkLowestBusyTxs bounds the mempool for the clerkLowestIDs lowest
	// pending IDs. It sits well below the mempool's capacity so other traffic
	// must fill most of it to hold them back, and these few IDs alone can't
	// push the mempool to the cap.
	clerkLowestBusyTxs = 2500

	// clerkMaxStoreWait is how long after its record entered the mempool a task
	// checks that heimdall stored it, and broadcasts it again if not. Heimdall
	// stores an accepted record within a few blocks, so one still missing then
	// was dropped; the check is late enough that a burst's tasks come back
	// after it, a few at a time, and cost one retry each.
	clerkMaxStoreWait = 10 * time.Minute

	// clerkTaskRetries is the RetryCount the listener gives a clerk task. An
	// accepted broadcast restores it: a task now lives until its record is
	// stored, and transient errors over that life must not use up its budget.
	clerkTaskRetries = 5

	// windowReleasedHeader marks a task that stopped holding the order window.
	windowReleasedHeader = "clerkWindowReleased"
	headerTrue           = "true"

	// standbyHeader marks a task waiting to take over a released one for the
	// same event; taking over, it continues as released.
	standbyHeader = "clerkReleasedStandby"

	// clerkStoreCheckDelay is the shortest interval between the heimdall reads
	// clerk tasks share: whether an idle lowest pending ID is stored, and the
	// seed after a start.
	clerkStoreCheckDelay = 5 * time.Second

	// acceptedAtHeader holds the unix time the task's record was last seen
	// entering the mempool. machinery republishes the same signature on
	// retry, so it persists across retries and restarts.
	acceptedAtHeader = "clerkAcceptedAt"

	// clerkDeferRetryDelay is how soon a deferred clerk task runs again. An ID
	// that is n windows ahead of the lowest pending one waits n times this, up
	// to clerkMaxOrderDelay, so IDs about to enter the window are back quickly
	// while the rest of a burst waits without churning the queue.
	clerkDeferRetryDelay = 3 * time.Second

	// clerkMaxOrderDelay caps the order wait. It's reached 20000 IDs ahead of
	// the lowest pending one, more than heimdall stores in that time; the far
	// end of a large burst waits without republishing every few minutes.
	clerkMaxOrderDelay = 10 * time.Minute

	// clerkSeedOrderDelay caps the order wait behind a lowest ID only a seed
	// holds: it may clear any moment, and a third of seedLive keeps the
	// burst's retries low while it holds.
	clerkSeedOrderDelay = 3 * time.Minute

	// clerkLowestIdle is how long the lowest pending ID can go without its task
	// running before another clerk task checks whether heimdall already stored
	// it. A task whose retry machinery failed to republish never runs again and
	// would otherwise hold the window for pendingRecordTTL.
	clerkLowestIdle = time.Minute

	// clerkWarmup is how long after the processor starts its tasks only record
	// their ID. The order window lives in memory, so after a restart it must
	// refill before any task can tell whether it is the lowest; every task
	// inside the window runs again within util.RetryStateSyncTaskDelay.
	clerkWarmup = 30 * time.Second

	// borStateTimeout bounds the read of bor's last state sync ID. It is part
	// of seedTimeout, so a slow bor still leaves the gap scan some time.
	borStateTimeout = 2 * time.Second

	// clerkSeedRetryWindow is how long after a start tasks keep asking bor
	// for the seed when it hasn't answered: a node that runs without bor
	// stops asking then.
	clerkSeedRetryWindow = 10 * time.Minute

	// seedTimeout bounds the seed computation, so a slow heimdall can hold
	// at most one worker for about this long. A cold chain params read (one
	// per process) is bounded only by the HTTP client's timeout.
	seedTimeout = 5 * time.Second

	// latestOnL1TTL is how long a read of the L1 state sync counter is shared
	// across released tasks; each read is an L1 call on the heimdall side.
	latestOnL1TTL = 30 * time.Second

	// latestOnL1FailTTL is how long a failed counter read is shared, so an L1
	// outage costs one timed-out read per period, not one per released task.
	latestOnL1FailTTL = 10 * time.Second

	// mempoolCountTTL is how long a read of the mempool size is shared across
	// clerk tasks, so a busy period costs at most one count query per second.
	mempoolCountTTL = time.Second

	infoMsgClerkMempoolBusy    = "ClerkProcessor: heimdall mempool is busy, deferring broadcast"
	infoMsgClerkAheadOfPending = "ClerkProcessor: state sync is too far ahead of the lowest pending one, deferring broadcast"
	infoMsgClerkMempoolFull    = "ClerkProcessor: heimdall mempool is full, retrying in sometime"
	warnMsgClerkMempoolCount   = "ClerkProcessor: can't read the heimdall mempool size, deferring all but the frontier state syncs"
	debugMsgClerkNoMempoolSize = "ClerkProcessor: no heimdall mempool size, deferring broadcast"
	debugMsgClerkBroadcast     = "ClerkProcessor: record broadcast, checking later that heimdall stored it"
	debugMsgClerkAwaitingStore = "ClerkProcessor: record already in the mempool, checking later that heimdall stored it"
	errMsgClerkUnstored        = "ClerkProcessor: state sync still not stored long after it entered the mempool, broadcasting it again"
	warnMsgClerkNotOnL1        = "ClerkProcessor: state sync ID is above the latest one on L1, holding its rebroadcast"
	errMsgClerkTaskExhausted   = "ClerkProcessor: state sync task ran out of retries before heimdall stored the record"
	debugMsgClerkDuplicateTask = "ClerkProcessor: another task already retries this released state sync, checking it again later"

	msgMempoolBusy    = "heimdall mempool is busy"
	msgAwaitingStore  = "record broadcast, waiting for heimdall to store it"
	msgAheadOfPending = "state sync ahead of the lowest pending one"
	msgNotOnL1        = "state sync ID above the latest one on L1"
	msgWarmingUp      = "clerk processor warming up its order window"
	msgCopyInMempool  = "transaction already in mempool"
	msgLookupsBusy    = "other clerk tasks are reading heimdall's records"
	msgReleasedOwned  = "another task retries this released state sync"
)

// deferOutOfOrder returns the lowest ID still pending here, and a retry-later
// error, which machinery does not count against RetryCount, while id is
// clerkOrderWindow or more above it. It reads heimdall state only when the
// lowest pending ID's task stopped running: a stored ID left there would also
// rank tasks inside the window against the wrong lowest in the busy gate.
func (cp *ClerkProcessor) deferOutOfOrder(eventName string, id uint64, owner string) (uint64, error) {
	now := time.Now()
	lowest := cp.pending.touch(id, owner, now)
	if id != lowest {
		lowest = cp.dropStoredLowest(lowest, now)
	}
	if id < lowest+clerkOrderWindow {
		return lowest, nil
	}

	delay := orderRetryDelay(id - lowest)
	if cp.pending.seedOnly(lowest) {
		// No task holds the seeded lowest ID, so it may clear any moment
		// (stored, or its seedLive over): don't wait a far task's full delay.
		delay = min(delay, clerkSeedOrderDelay)
	}
	delay = wholeSeconds(delay + spread(id, delay/5))
	metrics.ClerkOrderDeferrals.Inc()
	cp.Logger.Debug(infoMsgClerkAheadOfPending, "event", eventName, "id", id, "lowestPending", lowest, "retry delay", delay)
	return lowest, tasks.NewErrRetryTaskLater(msgAheadOfPending, delay)
}

// dropStoredLowest removes the lowest pending ID when its task hasn't run for
// clerkLowestIdle and heimdall already stored it, and returns the lowest
// pending ID after that. The check runs at most once per clerkStoreCheckDelay.
func (cp *ClerkProcessor) dropStoredLowest(lowest uint64, now time.Time) uint64 {
	idle, lastRun, ok := cp.pending.lowest(now)
	if !ok || now.Sub(lastRun) < clerkLowestIdle || !cp.lowestCheck.due(now) {
		return lowest
	}

	// IDs above int64 never enter the pending set.
	record, err := util.GetClerkEventRecord(int64(idle), cp.cliCtx.Codec)
	if err != nil || record.Id == 0 {
		return idle
	}
	cp.pending.done(idle)
	metrics.ClerkPendingReconciled.Inc()
	next, _, _ := cp.pending.lowest(now)
	return next
}

// deferWhileMempoolBusy returns a retry-later error while the local mempool
// holds clerkMempoolBusyTxs or more txs. The clerkFrontierIDs lowest pending
// IDs wait only at clerkFrontierBusyTxs, and the clerkLowestIDs lowest only
// at clerkLowestBusyTxs. A released task waits releasedRetryDelay unless it
// ranks as the lowest. The mempool is read when the task runs, since the listener sizes its delay
// before a burst is submitted. When the size can't be read, the frontier IDs
// go ahead and the rest wait.
func (cp *ClerkProcessor) deferWhileMempoolBusy(eventName string, event interface{}, id, lowest uint64, released bool) error {
	delay := deferDelay(id, lowest, released)
	busyTxs := clerkMempoolBusyTxs
	switch {
	case id < lowest+clerkLowestIDs:
		busyTxs = clerkLowestBusyTxs
	case id < lowest+clerkFrontierIDs:
		busyTxs = clerkFrontierBusyTxs
	}

	mempoolTxs, fresh, err := cp.mempool.size(event)
	if err != nil {
		if fresh {
			cp.Logger.Warn(warnMsgClerkMempoolCount, "error", err)
		}
		if id < lowest+clerkFrontierIDs {
			return nil
		}
		metrics.ClerkMempoolBusyDeferrals.Inc()
		cp.Logger.Debug(debugMsgClerkNoMempoolSize, "event", eventName, "id", id, "retry delay", delay)
		return tasks.NewErrRetryTaskLater(msgMempoolBusy, delay)
	}
	if mempoolTxs < busyTxs {
		return nil
	}

	metrics.ClerkMempoolBusyDeferrals.Inc()
	cp.Logger.Info(infoMsgClerkMempoolBusy, "event", eventName, "id", id, "pending", mempoolTxs, "retry delay", delay)
	return tasks.NewErrRetryTaskLater(msgMempoolBusy, delay)
}

// mempoolSize caches the last mempool size read for mempoolCountTTL.
type mempoolSize struct {
	mu      sync.Mutex
	at      time.Time
	txs     int
	err     error
	reading bool
	// readOnce reports whether a read ever completed; forget doesn't reset it.
	readOnce bool
	read     func(event interface{}) (int, error)
	now      func() time.Time
}

var errMempoolReadInProgress = errors.New("mempool size read in progress")

// size returns the cached mempool size, reading it again once the cache is
// mempoolCountTTL old. fresh reports whether this call did the read. Only one
// task reads at a time and the lock isn't held across the read: the workers
// are shared with every other bridge task, so a slow RPC must hold at most
// one of them. The others get the last size read, or an error before the
// first read returns.
func (m *mempoolSize) size(event interface{}) (int, bool, error) {
	clock := m.now
	if clock == nil {
		clock = time.Now
	}

	m.mu.Lock()
	switch {
	case !m.at.IsZero() && clock().Sub(m.at) < mempoolCountTTL:
		defer m.mu.Unlock()
		return m.txs, false, m.err
	case m.reading && !m.readOnce:
		m.mu.Unlock()
		return 0, false, errMempoolReadInProgress
	case m.reading:
		defer m.mu.Unlock()
		return m.txs, false, m.err
	}
	m.reading = true
	m.mu.Unlock()
	// machinery recovers a panicking task; the next read must still happen.
	defer func() {
		m.mu.Lock()
		m.reading = false
		m.mu.Unlock()
	}()

	read := m.read
	if read == nil {
		read = util.UnconfirmedTxnCount
	}
	txs, err := read(event)

	m.mu.Lock()
	defer m.mu.Unlock()
	m.txs, m.err, m.at, m.readOnce = txs, err, clock(), true
	return txs, true, err
}

// forget drops the cached size, so the next task reads it again.
func (m *mempoolSize) forget() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.at = time.Time{}
}

// releaseOnLastAttempt stops holding the order window for id once machinery
// won't run its task again: it runs the last attempt with RetryCount 0, and a
// retry-later error doesn't count against it.
func (cp *ClerkProcessor) releaseOnLastAttempt(ctx context.Context, id uint64, origin string, err error) {
	var retryLater tasks.ErrRetryTaskLater
	if err == nil || errors.As(err, &retryLater) {
		return
	}
	if signature := tasks.SignatureFromContext(ctx); signature != nil && signature.RetryCount == 0 {
		cp.leavePending(id, signature.UUID)
		cp.released.doneOrigin(id, origin)
		metrics.ClerkTasksExhausted.Inc()
		cp.Logger.Error(errMsgClerkTaskExhausted, "id", id, "error", err)
	}
}

// errTaskDone ends a task without an error: its work is done or another task
// carries it.
var errTaskDone = errors.New("clerk task done")

// taskResult maps errTaskDone to a finished task.
func taskResult(err error) error {
	if errors.Is(err, errTaskDone) {
		return nil
	}
	return err
}

// admit runs the order gate and returns the lowest ID to rank id against in
// the busy gate. A task that released the window skips ordering and ranks
// against the lowest ID still unstored here, released ones included; a second
// task for the same event as a live released task waits, and takes over once
// that task stops (stored, given up or gone); one waits per event, others end.
func (cp *ClerkProcessor) admit(ctx context.Context, eventName string, id uint64, origin string) (uint64, error) {
	now := time.Now()
	signature := tasks.SignatureFromContext(ctx)
	isReleased := windowReleased(signature)
	if !isReleased && cp.released.has(id, origin, now) {
		// After a restart this task may have reached the pending set first.
		cp.leavePending(id, taskOwner(ctx))
		if !cp.released.standby.claim(id, origin, taskOwner(ctx), now) {
			return 0, errTaskDone
		}
		setHeader(signature, standbyHeader)
		cp.Logger.Debug(debugMsgClerkDuplicateTask, "event", eventName, "id", id)
		return 0, tasks.NewErrRetryTaskLater(msgReleasedOwned, clerkMaxStoreWait)
	}
	if !isReleased && signature != nil && headerString(signature.Headers, standbyHeader) == headerTrue {
		// Its record already reached the mempool through the released task,
		// so it doesn't hold the window again.
		setHeader(signature, windowReleasedHeader)
		isReleased = true
	}
	if cp.warmingUp(now) {
		return 0, cp.warmUp(ctx, id, origin, isReleased, now)
	}
	// A bridge quiet through the warm-up, or one whose heimdall was
	// unreadable then, seeds on its first tasks after it. Tasks don't wait
	// for the seed: an unreadable heimdall must not stall state sync.
	cp.seedFrontier(ctx, now)
	if isReleased {
		// A released ID never holds the window, also when seeded after a restart.
		cp.leavePending(id, taskOwner(ctx))
		lowest := cp.released.touch(id, origin, now)
		if pending, _, ok := cp.pending.lowest(now); ok {
			lowest = min(lowest, pending)
		}
		return lowest, nil
	}
	return cp.deferOutOfOrder(eventName, id, taskOwner(ctx))
}

// leavePending drops the pending lease of the task owner for id and the seed
// standing in for that task; a task for id from another publish keeps its own.
func (cp *ClerkProcessor) leavePending(id uint64, owner string) {
	cp.pending.release(id, owner)
	cp.pending.release(id, seedOrigin)
}

// taskOwner identifies the task holding a pending lease: each separately
// published task for an ID (a self-heal replay, a re-send) holds its own, so
// one giving up doesn't free the ID while another still runs. A broker
// redelivery keeps its task's UUID and shares its lease.
func taskOwner(ctx context.Context) string {
	if signature := tasks.SignatureFromContext(ctx); signature != nil {
		return signature.UUID
	}
	return ""
}

// handleBroadcastErr maps a broadcast result to the task's outcome; an
// accepted broadcast settles the task (see settleAccepted).
func (cp *ClerkProcessor) handleBroadcastErr(ctx context.Context, eventName string, id uint64, origin string, err error) error {
	if errors.Is(err, broadcaster.ErrMempoolFull) {
		cp.Logger.Info(infoMsgClerkMempoolFull, "event", eventName, "id", id, "retry delay", util.RetryStateSyncTaskDelay)
		return tasks.NewErrRetryTaskLater(broadcaster.ErrMempoolFull.Error(), util.RetryStateSyncTaskDelay)
	}

	if err != nil {
		cp.Logger.Error(errMsgClerkBroadcasting, "error", err)
		return err
	}

	// Our tx just entered the mempool: let the next task read the size again.
	cp.mempool.forget()
	cp.Logger.Debug(debugMsgClerkBroadcast, "event", eventName, "id", id)
	return cp.settleAccepted(ctx, id, origin, time.Now())
}
