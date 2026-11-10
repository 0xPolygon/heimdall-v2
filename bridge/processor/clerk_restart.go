package processor

import (
	"context"
	"errors"
	"math"
	"math/big"
	"time"

	"github.com/RichardKnop/machinery/v1/tasks"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"

	"github.com/0xPolygon/heimdall-v2/bridge/util"
)

// warmingUp reports whether the processor started less than clerkWarmup ago.
func (cp *ClerkProcessor) warmingUp(now time.Time) bool {
	return !cp.startedAt.IsZero() && now.Sub(cp.startedAt) < clerkWarmup
}

// warmUp records id and has its task run again when the warm-up ends: one
// pass over the queued tasks is enough, since a recorded ID stays for
// pendingRecordTTL.
func (cp *ClerkProcessor) warmUp(ctx context.Context, id uint64, origin string, isReleased bool, now time.Time) error {
	if isReleased {
		cp.leavePending(id, taskOwner(ctx))
		cp.released.touch(id, origin, now)
	} else {
		cp.pending.touch(id, taskOwner(ctx), now)
	}
	cp.seedFrontier(ctx, now)
	delay := max(clerkDeferRetryDelay, cp.startedAt.Add(clerkWarmup).Sub(now))
	return tasks.NewErrRetryTaskLater(msgWarmingUp, wholeSeconds(delay+spread(id, clerkDeferRetryDelay)))
}

// seedFrontier adds the state sync ID bor waits on to the pending set after a
// start; tasks try it until it succeeds. The window then rests on
// chain state, not only on the tasks seen during the warm-up: a lower task
// delayed past it can't be overtaken.
// Without a task here the seed is dropped once heimdall stores it (see
// dropStoredLowest). It is retried at most once per clerkStoreCheckDelay. A
// seed taken without bor is provisional: tasks keep asking bor for
// clerkSeedRetryWindow after the start, so a gap below heimdall's bound is
// still found once bor answers. That seed then replaces the provisional one.
func (cp *ClerkProcessor) seedFrontier(ctx context.Context, now time.Time) {
	// One seeding at a time: the scan can outlast the retry gate.
	if cp.seeded.Load() || !cp.seeding.CompareAndSwap(false, true) {
		return
	}
	defer cp.seeding.Store(false)
	if !cp.seedCheck.due(now) {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, seedTimeout)
	defer cancel()
	next, ok, final := cp.nextUnstoredID(ctx)
	if !ok {
		return
	}
	cp.placeSeed(next, final, now)
	if final || now.Sub(cp.startedAt) >= clerkSeedRetryWindow {
		cp.seeded.Store(true)
		return
	}
	// A bor or list read that failed may well fail again; ask less often,
	// but keep asking while the warm-up still holds tasks back for the seed.
	if !cp.warmingUp(now) {
		cp.seedCheck.holdUntil(now.Add(clerkSeedRetryBackoff))
	}
}

// placeSeed seeds next. A provisional seed is placed once; later attempts only
// replace it with one bor backs, so neither a rising bound nor a given-up ID
// piles up. A provisional ID 1 came from an empty store read, which records
// since stored disprove.
func (cp *ClerkProcessor) placeSeed(next uint64, final bool, now time.Time) {
	if !final && cp.seedID != 0 && (cp.seedID != 1 || next <= 1) {
		return
	}
	if prev := cp.seedID; prev != 0 && prev != next {
		cp.pending.release(prev, seedOrigin)
	}
	cp.seedID = next
	cp.pending.seed(next, now)
}

// nextUnstoredID returns the lowest state sync ID heimdall may still miss,
// whether there is one (ok) and whether bor's answer backs it (final).
// Heimdall can store records out of order, so its highest stored ID only
// bounds it; the first ID after bor's last one that heimdall lacks is the gap
// bor will block on. Without bor's answer the bound is a provisional seed.
func (cp *ClerkProcessor) nextUnstoredID(ctx context.Context) (next uint64, ok, final bool) {
	// The count is heimdall's highest stored ID; 0 also stands for a failed
	// store read, so only bor confirms an empty chain.
	count, err := util.GetRecordCount(ctx, cp.cliCtx.Codec)
	// IDs above int64 never enter the pending set (heimdall can't store them).
	if err != nil || count >= math.MaxInt64 {
		return 0, false, false
	}
	borLast, err := cp.borLastStateID(ctx)
	switch {
	case errors.Is(err, errChainParamsReadInProgress):
		// Another task is reading the params bor's address comes from; a
		// later task seeds with them.
		return 0, false, false
	case count == 0:
		// Without bor, an empty store still seeds ID 1, provisionally: on a
		// live chain a stored ID 1 is dropped by the reconcile. A bor ahead
		// says the store read failed.
		confirmed := err == nil && borLast == 0
		return 1, confirmed || err != nil, confirmed
	case err != nil:
		return count + 1, true, false
	case borLast >= count:
		return count + 1, true, true
	}
	// The first gap after bor, looked for within one order window. Past the
	// scan nothing is known, so a clean scan short of the bound seeds the
	// first ID it didn't read; the seed is dropped once heimdall stores it. A
	// failed scan proves nothing, so its bound stays provisional and a later
	// task scans again.
	to := min(count, borLast+clerkOrderWindow)
	gap, found, scanned := cp.firstGap(ctx, borLast+1, to)
	switch {
	case found:
		return gap, true, true
	case scanned && to < count:
		return to + 1, true, true
	}
	return count + 1, true, scanned
}

// advanceSeed replaces a seed heimdall stored with the next ID it lacks up to
// its highest stored ID, found by the same bounded scan as the first seed, so
// the window keeps resting on chain state past the first scan. It reports
// false, keeping the stored seed for a later check, when another seeding runs
// or a read fails.
func (cp *ClerkProcessor) advanceSeed(ctx context.Context, stored uint64, now time.Time) bool {
	if !cp.seeding.CompareAndSwap(false, true) {
		return false
	}
	defer cp.seeding.Store(false)

	ctx, cancel := context.WithTimeout(ctx, seedTimeout)
	defer cancel()
	count, err := util.GetRecordCount(ctx, cp.cliCtx.Codec)
	if err != nil {
		return false
	}
	var next uint64
	if stored < count && count < math.MaxInt64 {
		to := min(count, stored+clerkOrderWindow)
		gap, found, scanned := cp.firstGap(ctx, stored+1, to)
		switch {
		case !scanned:
			return false
		case found:
			next = gap
		case to < count:
			next = to + 1
		}
	}
	cp.pending.done(stored)
	if next != 0 {
		cp.seedID = next
		cp.pending.seed(next, now)
	}
	return true
}

// lastStateIDReader is the StateReceiver read seedFrontier needs.
type lastStateIDReader interface {
	LastStateId(opts *bind.CallOpts) (*big.Int, error)
}

// borLastStateID reads the last state sync ID bor committed.
func (cp *ClerkProcessor) borLastStateID(ctx context.Context) (uint64, error) {
	clerkContext, err := cp.getClerkContext()
	if err != nil {
		return 0, err
	}
	// A cold params read isn't bounded by ctx; don't start bor's after it.
	if err = ctx.Err(); err != nil {
		return 0, err
	}
	stateReceiver := cp.stateReceiver
	if stateReceiver == nil {
		stateReceiver = func(address string) (lastStateIDReader, error) {
			return cp.contractCaller.GetStateReceiverInstance(address)
		}
	}
	receiver, err := stateReceiver(clerkContext.ChainmanagerParams.ChainParams.StateReceiverAddress)
	if err != nil {
		return 0, err
	}
	callCtx, cancel := context.WithTimeout(ctx, borStateTimeout)
	defer cancel()
	last, err := receiver.LastStateId(&bind.CallOpts{Context: callCtx})
	if err != nil {
		return 0, err
	}
	if !last.IsUint64() {
		return 0, errors.New("bor last state ID out of range")
	}
	return last.Uint64(), nil
}

// clerkSeedRetryBackoff spaces the attempts that replace a provisional seed.
const clerkSeedRetryBackoff = 30 * time.Second

// recordListPage is the page size of heimdall's record list (its maximum).
const recordListPage = 50

// firstGap returns the lowest ID in from..to heimdall hasn't stored, and
// whether every page was read. Only a successful list read counts: a failed
// page reports no gap, so a transient error can't pass for a missing record.
func (cp *ClerkProcessor) firstGap(ctx context.Context, from, to uint64) (gap uint64, found, scanned bool) {
	stored := make(map[uint64]struct{}, to-from+1)
	for page := (from-1)/recordListPage + 1; (page-1)*recordListPage+1 <= to; page++ {
		ids, err := util.GetRecordIDsPage(ctx, cp.cliCtx.Codec, page, recordListPage)
		if err != nil {
			return 0, false, false
		}
		for _, id := range ids {
			stored[id] = struct{}{}
		}
	}
	for id := from; id <= to; id++ {
		if _, ok := stored[id]; !ok {
			return id, true, true
		}
	}
	return 0, false, true
}
