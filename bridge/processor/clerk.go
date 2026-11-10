package processor

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/RichardKnop/machinery/v1/tasks"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/0xPolygon/heimdall-v2/bridge/util"
	"github.com/0xPolygon/heimdall-v2/common/tracing"
	"github.com/0xPolygon/heimdall-v2/contracts/statesender"
	"github.com/0xPolygon/heimdall-v2/helper"
	chainmanagertypes "github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
	clerkTypes "github.com/0xPolygon/heimdall-v2/x/clerk/types"
)

const (
	// Error messages
	errMsgClerkUnmarshallingEvent         = "ClerkProcessor: error while unmarshalling event from rootChain"
	errMsgClerkParsingEvent               = "ClerkProcessor: error while parsing event"
	errMsgClerkConvertingAddress          = "ClerkProcessor: error converting address to string"
	errMsgClerkBroadcasting               = "ClerkProcessor: error while broadcasting clerk Record to heimdall"
	errMsgClerkFetchingChainManagerParams = "ClerkProcessor: error while fetching chain manager params"
	errMsgClerkInvalidID                  = "ClerkProcessor: dropping state sync with an ID heimdall can't store"

	// Info messages
	infoMsgClerkStarting                 = "ClerkProcessor: starting"
	infoMsgClerkRegisteringTasks         = "ClerkProcessor: registering clerk tasks"
	infoMsgClerkIgnoringAlreadyProcessed = "ClerkProcessor: ignoring task to send deposit to heimdall as already processed"
	infoMsgClerkDataTooLarge             = `ClerkProcessor: data is too large to process, Resetting to ""`
	infoMsgClerkTxInMempool              = "ClerkProcessor: similar transaction already in mempool, retrying in sometime"

	// Debug messages
	debugMsgClerkNewEventFound = "ClerkProcessor: new event found"
)

// ClerkContext for bridge
type ClerkContext struct {
	ChainmanagerParams *chainmanagertypes.Params
}

// ClerkProcessor - sync state/deposit events
type ClerkProcessor struct {
	BaseProcessor
	stateSenderAbi *abi.ABI
	pending        pendingRecords
	released       releasedRecords
	latestOnL1     cachedCounter
	lowestCheck    rateGate
	startedAt      time.Time
	seeded         atomic.Bool
	seeding        atomic.Bool
	// seedID is the last ID seeded; written only by the task holding seeding.
	seedID        uint64
	seedCheck     rateGate
	stateReceiver func(address string) (lastStateIDReader, error)
	mempool       mempoolSize
	chainParams   cachedChainParams
	recordLookups lookupSlots
}

type cachedChainParams struct {
	mu      sync.Mutex
	at      time.Time
	params  *chainmanagertypes.Params
	err     error
	failAt  time.Time
	reading bool
	now     func() time.Time
}

const (
	chainParamsCacheTTL = time.Minute
	// chainParamsFailTTL is how long a failed read is shared (with the last
	// params, if any), so an outage costs one request per period.
	chainParamsFailTTL = time.Second
)

var errChainParamsReadInProgress = errors.New("chain manager params read in progress")

// NewClerkProcessor - add state sender abi to clerk processor
func NewClerkProcessor(stateSenderAbi *abi.ABI) *ClerkProcessor {
	return &ClerkProcessor{
		stateSenderAbi: stateSenderAbi,
		startedAt:      time.Now(),
	}
}

// Start starts new block subscription
func (cp *ClerkProcessor) Start() error {
	cp.Logger.Info(infoMsgClerkStarting)
	return nil
}

// RegisterTasks registers the clerk-related tasks with machinery
func (cp *ClerkProcessor) RegisterTasks() {
	cp.Logger.Info(infoMsgClerkRegisteringTasks)

	if err := cp.queueConnector.Server.RegisterTask("sendStateSyncedToHeimdall", cp.sendStateSyncedToHeimdall); err != nil {
		cp.Logger.Error("ClerkProcessor | RegisterTasks | sendStateSyncedToHeimdall", "error", err)
	}
}

// sendStateSyncedToHeimdall - handle state sync event from rootChain
// 1. Check if this deposit event has to be broadcast to heimdall
// 2. Create and broadcast record transaction to heimdall
func (cp *ClerkProcessor) sendStateSyncedToHeimdall(ctx context.Context, eventName string, logBytes string) (err error) {
	tracingCtx := tracing.WithTracer(context.Background(), otel.Tracer("State-Sync"))
	// work begins
	sendStateSyncedToHeimdallCtx, sendStateSyncedToHeimdallSpan := tracing.StartSpan(tracingCtx, "sendStateSyncedToHeimdall")
	defer tracing.EndSpan(sendStateSyncedToHeimdallSpan)

	start := time.Now()

	vLog := types.Log{}
	if err = json.Unmarshal([]byte(logBytes), &vLog); err != nil {
		cp.Logger.Error(errMsgClerkUnmarshallingEvent, "error", err)
		return err
	}

	event := new(statesender.StatesenderStateSynced)
	if err = helper.UnpackLog(cp.stateSenderAbi, event, eventName, &vLog); err != nil {
		cp.Logger.Error(errMsgClerkParsingEvent, "name", eventName, "error", err)
		return nil
	}
	defer util.LogElapsedTimeForStateSyncedEvent(event, "sendStateSyncedToHeimdall", start)

	// Heimdall never stores a record with a zero or oversized ID, so a task
	// for one would only hold the order window.
	if event.Id.Sign() <= 0 || !event.Id.IsInt64() {
		cp.Logger.Error(errMsgClerkInvalidID, "event", eventName, "id", event.Id)
		return nil
	}
	id := event.Id.Uint64()
	origin := fmt.Sprintf("%s:%d", vLog.TxHash.Hex(), vLog.Index)
	defer func() {
		// machinery turns a panic into a failed attempt; the last one must
		// still let go of the order window.
		if r := recover(); r != nil {
			cp.releaseOnLastAttempt(ctx, id, origin, fmt.Errorf("task panicked: %v", r))
			panic(r)
		}
		cp.releaseOnLastAttempt(ctx, id, origin, err)
	}()

	tracing.SetAttributes(sendStateSyncedToHeimdallSpan, []attribute.KeyValue{
		attribute.String("event", eventName),
		attribute.Int64("id", event.Id.Int64()),
		attribute.String("contract", event.ContractAddress.String()),
	}...)

	// Ordering needs only the ID, so it runs before any heimdall query: during a
	// burst most tasks wait here, and they retry often.
	lowest, err := cp.admit(ctx, eventName, id, origin)
	if err != nil {
		return taskResult(err)
	}

	stored, err := cp.checkStored(ctx, eventName, event, &vLog, id, lowest)
	if err != nil || stored {
		return err
	}

	if err = cp.awaitStored(ctx, eventName, id, time.Now()); err != nil {
		return taskResult(err)
	}

	clerkContext, err := cp.getClerkContext()
	if errors.Is(err, errChainParamsReadInProgress) {
		// Not a failure: another task is reading them; don't spend a retry.
		return tasks.NewErrRetryTaskLater(errChainParamsReadInProgress.Error(), clerkDeferRetryDelay)
	}
	if err != nil {
		return err
	}

	chainParams := clerkContext.ChainmanagerParams.ChainParams

	cp.Logger.Debug(
		debugMsgClerkNewEventFound,
		"event", eventName,
		"id", event.Id,
		"contract", event.ContractAddress,
		"data", hex.EncodeToString(event.Data),
		"borChainId", chainParams.BorChainId,
		"txHash", vLog.TxHash.String(),
		"logIndex", uint64(vLog.Index),
		"blockNumber", vLog.BlockNumber,
	)

	return cp.submitEventRecord(ctx, sendStateSyncedToHeimdallCtx, eventName, event, &vLog, chainParams.BorChainId, lowest)
}

// recordStored reports whether heimdall already stored the event's record, and
// if so stops tracking it. A failed lookup counts as not stored.
// checkStored runs recordStored within the bound on concurrent record lookups.
// The lowest ID, the one bor waits on, never waits for a slot.
func (cp *ClerkProcessor) checkStored(ctx context.Context, eventName string, event *statesender.StatesenderStateSynced, vLog *types.Log, id, lowest uint64) (bool, error) {
	if id != lowest {
		if !cp.recordLookups.tryAcquire() {
			delay := wholeSeconds(clerkDeferRetryDelay + spread(id, clerkDeferRetryDelay))
			if windowReleased(tasks.SignatureFromContext(ctx)) {
				delay = deferDelay(id, lowest, true)
			}
			return false, tasks.NewErrRetryTaskLater(msgLookupsBusy, delay)
		}
		defer cp.recordLookups.release()
	}
	return cp.recordStored(eventName, event, vLog), nil
}

func (cp *ClerkProcessor) recordStored(eventName string, event *statesender.StatesenderStateSynced, vLog *types.Log) bool {
	existingRecord, err := util.GetClerkEventRecord(event.Id.Int64(), cp.cliCtx.Codec)
	if err != nil || existingRecord.Id == 0 {
		return false
	}

	cp.Logger.Info(infoMsgClerkIgnoringAlreadyProcessed,
		"event", eventName,
		"id", event.Id,
		"contract", event.ContractAddress,
		"txHash", vLog.TxHash.String(),
		"logIndex", uint64(vLog.Index),
		"blockNumber", vLog.BlockNumber,
	)
	cp.pending.done(event.Id.Uint64())
	cp.released.done(event.Id.Uint64())
	return true
}

// submitEventRecord broadcasts the record for an event heimdall has not stored yet.
func (cp *ClerkProcessor) submitEventRecord(ctx, spanCtx context.Context, eventName string, event *statesender.StatesenderStateSynced, vLog *types.Log, borChainID string, lowest uint64) error {
	if err := cp.deferWhileMempoolBusy(eventName, event, event.Id.Uint64(), lowest, windowReleased(tasks.SignatureFromContext(ctx))); err != nil {
		return err
	}

	_, maxStateSyncSizeCheckSpan := tracing.StartSpan(spanCtx, "maxStateSyncSizeCheck")
	if len(event.Data) > helper.MaxStateSyncSize {
		cp.Logger.Info(infoMsgClerkDataTooLarge, "data", hex.EncodeToString(event.Data))
		event.Data = common.FromHex("")
	}
	tracing.EndSpan(maxStateSyncSizeCheckSpan)

	address, err := helper.GetAddressString()
	if err != nil {
		return fmt.Errorf("ClerkProcessor: error converting address to string: %w", err)
	}

	msg := clerkTypes.NewMsgEventRecord(
		address,
		vLog.TxHash.String(),
		uint64(vLog.Index),
		vLog.BlockNumber,
		event.Id.Uint64(),
		event.ContractAddress.Bytes(),
		event.Data,
		borChainID,
	)

	_, checkTxAgainstMempoolSpan := tracing.StartSpan(spanCtx, "checkTxAgainstMempool")
	// An identical copy of this record in the mempool is the same event.
	inMempool, _ := cp.checkTxAgainstMempool(ctx, &msg, event)
	tracing.EndSpan(checkTxAgainstMempoolSpan)

	origin := fmt.Sprintf("%s:%d", vLog.TxHash.Hex(), vLog.Index)
	if inMempool {
		return cp.waitOnMempoolCopy(ctx, eventName, event.Id.Uint64(), origin, time.Now())
	}

	_, BroadcastToHeimdallSpan := tracing.StartSpan(spanCtx, "BroadcastToHeimdall")
	// Broadcast the transaction to heimdall.
	_, err = cp.txBroadcaster.BroadcastToHeimdall(ctx, &msg, event)
	tracing.EndSpan(BroadcastToHeimdallSpan)

	return cp.handleBroadcastErr(ctx, eventName, event.Id.Uint64(), origin, err)
}

// getClerkContext returns the chain manager params, cached for
// chainParamsCacheTTL: clerk tasks retry often during a burst and only need
// the bor chain id, which changes only through governance. One task reads at
// a time, without holding the lock, so a slow heimdall holds one shared worker
// at most: the others get the last params read, or an error before any read
// succeeded. A failed refresh keeps serving the last params.
func (cp *ClerkProcessor) getClerkContext() (*ClerkContext, error) {
	c := &cp.chainParams
	clock := c.now
	if clock == nil {
		clock = time.Now
	}

	c.mu.Lock()
	if cached, ok := c.cachedLocked(clock()); ok {
		c.mu.Unlock()
		if cached.err != nil {
			return nil, cached.err
		}
		return &ClerkContext{ChainmanagerParams: cached.params}, nil
	}
	c.reading = true
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		c.reading = false
		c.mu.Unlock()
	}()

	fresh, err := util.GetChainmanagerParams(cp.cliCtx.Codec)

	c.mu.Lock()
	defer c.mu.Unlock()
	if err != nil {
		cp.Logger.Error(errMsgClerkFetchingChainManagerParams, "error", err)
		c.err, c.failAt = err, clock()
		if c.params == nil {
			return nil, err
		}
		return &ClerkContext{ChainmanagerParams: c.params}, nil
	}
	c.params, c.at, c.err = fresh, clock(), nil
	return &ClerkContext{ChainmanagerParams: fresh}, nil
}

type cachedAnswer struct {
	params *chainmanagertypes.Params
	err    error
}

// cachedLocked returns what a caller gets without reading; ok is false when
// this caller is to read the params.
func (c *cachedChainParams) cachedLocked(now time.Time) (cachedAnswer, bool) {
	switch {
	case c.params != nil && now.Sub(c.at) < chainParamsCacheTTL:
		return cachedAnswer{params: c.params}, true
	case c.reading && c.params != nil:
		return cachedAnswer{params: c.params}, true
	case c.reading:
		return cachedAnswer{err: errChainParamsReadInProgress}, true
	case c.err != nil && now.Sub(c.failAt) < chainParamsFailTTL && c.params != nil:
		return cachedAnswer{params: c.params}, true
	case c.err != nil && now.Sub(c.failAt) < chainParamsFailTTL:
		return cachedAnswer{err: c.err}, true
	}
	return cachedAnswer{}, false
}
