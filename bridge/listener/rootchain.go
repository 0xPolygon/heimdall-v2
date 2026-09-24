package listener

import (
	"context"
	"errors"
	"math/big"
	"strconv"
	"time"

	"github.com/RichardKnop/machinery/v1/tasks"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/0xPolygon/heimdall-v2/bridge/util"
	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/metrics"
	chainmanagerTypes "github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
)

// RootChainListenerContext - Root chain listener context
type RootChainListenerContext struct {
	ChainmanagerParams *chainmanagerTypes.Params
}

// RootChainListener - Listens to and processes events from RootChain
type RootChainListener struct {
	BaseListener

	stakingInfoAbi *abi.ABI
	stateSenderAbi *abi.ABI

	// Pre-built topic→event lookup (avoids per-log linear scan across ABIs)
	eventMap map[ethCommon.Hash]*abi.Event

	// Pre-built topic→emitting-contract lookup, so a log's address can be
	// checked against the specific contract its topic says it came from
	// (not just any of the three watched contracts).
	eventContract map[ethCommon.Hash]rootChainContract

	// Poll-cycle failure count per rejected log (keyed the same way as
	// rootChainRejectionState.countedThisCycle), so a log that keeps failing
	// validation across many polls can be quarantined instead of withholding
	// the cursor forever. Cycles counted toward a log's total need not be
	// consecutive: a cycle that aborts before content validation runs at all
	// (a transient RPC error, or the FilterLogs call budget) doesn't touch
	// this map, and a count set by a since-aborted cycle survives until
	// pruneStaleLogFailureCounts drops it — which only happens once a full
	// sweep completes successfully without needing that log again, not on
	// every cycle. Only ever touched from ProcessHeader, which
	// BaseListener.StartHeaderProcess drives from a single goroutine, so no
	// lock is needed.
	logFailureCounts map[string]uint64

	// For self-healing, it will be only initialized if sub_graph_url is provided
	subGraphClient *subGraphClient

	// onLogDispatched, when set, is called right after handleLog dispatches a
	// validated log. Every handleLog variant gates its real side effect
	// behind a validator-set check that needs REST/queue plumbing to
	// exercise directly, so this is the seam tests use to observe that
	// dispatch happened at all, and for which log. Left nil in production.
	onLogDispatched func(vLog types.Log, selectedEvent *abi.Event)
}

// rootChainContract identifies which of the three watched L1 contracts an
// event belongs to.
type rootChainContract int

const (
	rootChainContractRootChain rootChainContract = iota
	rootChainContractStateSender
	rootChainContractStakingInfo
)

const (
	lastRootBlockKey       = "rootchain-last-block" // Storage key
	maxRootChainBlockRange = 5000                   // Maximum number of blocks to fetch logs for in a single FilterLogs call
)

// maxRootChainLogRejections bounds how many poll cycles a single log can
// fail validation before it's quarantined: the cursor advances past the
// block containing it instead of withholding it forever. Cycles need not be
// strictly back-to-back — a log's count survives a cycle that aborts before
// content validation even runs (see the abort path in
// processRootChainBlockRangeInChunks) — so this bounds total occurrences,
// not a run of adjacent ones. See rejectRootChainLog and
// quarantineRootChainLog.
const maxRootChainLogRejections = 100

// maxTrackedLogFailures bounds logFailureCounts itself, independent of any
// single log's own count: an endpoint that returns a different bad
// txHash:index on every poll never lets any one entry reach
// maxRootChainLogRejections, so nothing is ever quarantined or pruned. Once
// this many distinct logs are tracked, rejectRootChainLog evicts the entry
// with the lowest count to make room for a new one — never refuses the new
// key outright, or a log that keeps recurring after the map has filled with
// one-off entries could never accumulate enough count to be quarantined,
// stalling the cursor on it forever.
//
// A single, un-bisectable block whose response contains more distinct
// persistently-malformed logs than this cap is a residual gap this doesn't
// close: every one of them needs to be quarantine-eligible in the very same
// poll cycle for that block to ever resolve, and with more distinct keys
// than available slots, at least one is always mid-eviction rather than
// eligible, so that one block never advances. The cap is set high enough
// that reaching it requires a single fabricated response with tens of
// thousands of distinct fake logs — far beyond anything a real L1 block can
// contain — so this bounds the threat to an endpoint willing to serve an
// implausibly large adversarial response, not a realistic one.
const maxTrackedLogFailures = 100_000

// maxRootChainFilterLogsCallsPerCycle bounds how many FilterLogs round trips
// a single ProcessHeader poll cycle can make. Resolving a maxRootChainBlockRange
// chunk that's quarantine-excludable at every leaf needs up to 2*N-1 calls in
// the worst case; capping well below that keeps one poll cycle from turning
// into thousands of sequential round trips to an endpoint already known to be
// misbehaving. What doesn't get resolved this cycle is retried next cycle —
// logFailureCounts and the persisted cursor both carry real progress forward,
// so this trades one big cycle for several bounded ones, not lost progress.
const maxRootChainFilterLogsCallsPerCycle = 500

var (
	errMainChainClientUnavailable  = errors.New("main chain client is nil")
	errNoSupportedRootChainTopics  = errors.New("no supported rootChain event topics configured")
	errUnexpectedRootChainLog      = errors.New("rootchain log query returned a log that doesn't match the query")
	errFilterLogsCallBudgetReached = errors.New("rootchain FilterLogs call budget reached for this poll cycle")

	rootChainEvents = map[string]struct{}{
		helper.NewHeaderBlockEvent: {},
		helper.StakedEvent:         {},
		helper.StakeUpdateEvent:    {},
		helper.SignerChangeEvent:   {},
		helper.UnstakeInitEvent:    {},
		helper.StateSyncedEvent:    {},
		helper.TopUpFeeEvent:       {},
		helper.SlashedEvent:        {},
		helper.UnJailedEvent:       {},
	}
)

// NewRootChainListener - constructor func
func NewRootChainListener() *RootChainListener {
	contractCaller, err := helper.NewContractCaller()
	if err != nil {
		panic(err)
	}

	abiSources := []struct {
		abi      *abi.ABI
		contract rootChainContract
	}{
		{&contractCaller.RootChainABI, rootChainContractRootChain},
		{&contractCaller.StateSenderABI, rootChainContractStateSender},
		{&contractCaller.StakingInfoABI, rootChainContractStakingInfo},
	}

	eventMap := make(map[ethCommon.Hash]*abi.Event)
	eventContract := make(map[ethCommon.Hash]rootChainContract)
	for _, src := range abiSources {
		for _, event := range src.abi.Events {
			e := event
			eventMap[e.ID] = &e
			eventContract[e.ID] = src.contract
		}
	}

	return &RootChainListener{
		stakingInfoAbi:   &contractCaller.StakingInfoABI,
		stateSenderAbi:   &contractCaller.StateSenderABI,
		eventMap:         eventMap,
		eventContract:    eventContract,
		logFailureCounts: make(map[string]uint64),
	}
}

// Start starts new block subscription
func (rl *RootChainListener) Start() error {
	rl.Logger.Info("RootChainListener: starting")

	// create cancellable context
	ctx, cancelSubscription := context.WithCancel(context.Background())
	rl.cancelSubscription = cancelSubscription

	// create cancellable context
	headerCtx, cancelHeaderProcess := context.WithCancel(context.Background())
	rl.cancelHeaderProcess = cancelHeaderProcess

	// start the header process
	go rl.StartHeaderProcess(headerCtx)

	// start go routine to poll for the new header using the client object
	rl.Logger.Info("RootChainListener: starting polling for root chain header blocks", "pollInterval", helper.GetConfig().SyncerPollInterval)

	// start polling for the finalized block in the main L1 chain (available post-merge)
	go rl.StartPolling(ctx, helper.GetConfig().SyncerPollInterval, big.NewInt(int64(rpc.FinalizedBlockNumber)))

	// Start the self-healing process
	go rl.startSelfHealing(ctx)

	return nil
}

// ProcessHeader - process header block from rootChain
func (rl *RootChainListener) ProcessHeader(newHeader *blockHeader) {
	rl.Logger.Debug("RootChainListener: new block detected", "blockNumber", newHeader.header.Number)

	// fetch context
	rootChainContext, err := rl.getRootChainContext()
	if err != nil {
		return
	}

	from, to, ok := rl.rootChainBlockRangeToProcess(rootChainContext, newHeader)
	if !ok {
		return
	}

	rl.processRootChainBlockRangeInChunks(rootChainContext, from, to)
}

// rootChainRejectionState tracks per-log rejection bookkeeping shared across
// one ProcessHeader poll cycle's whole call tree — every chunk and every
// bisection level processRootChainBlockRange visits for that cycle.
type rootChainRejectionState struct {
	// countedThisCycle ensures rootchain_listener_log_rejected_total, and the
	// persistent per-log failure count in RootChainListener.logFailureCounts,
	// only advance once per distinct log per cycle — not once per bisection
	// level that re-encounters it. Bounded at maxTrackedLogFailures entries
	// (see rejectRootChainLog) for the same reason logFailureCounts is
	// capped: an endpoint that fabricates more distinct malformed logs than
	// that in a single response would otherwise grow this map without limit
	// for the rest of the cycle. Past the cap, further distinct logs still
	// get rejected (validateAndHandleLogs falls back to failing the whole
	// batch for anything not already tracked), they just stop accumulating
	// their own log line, metric, and persistent failure count until the
	// next cycle.
	countedThisCycle map[string]struct{}

	// quarantine holds one entry per logKey whose persistent failure count
	// has crossed maxRootChainLogRejections in this cycle, keyed by the same
	// logKey used everywhere else — never by block number alone, so a
	// transient failure on an unrelated log can never be mistaken for this
	// one's quarantine. Once added, an entry stays for the rest of the cycle
	// (never deleted) so every remaining single-block leaf that re-encounters
	// the same persistently-misbehaving log can keep excluding it too —
	// otherwise a chunk-wide sweep would only ever clear one block per cycle
	// for a log an endpoint returns on every request regardless of range.
	quarantine map[string]*rootChainLogDetail

	// quarantinedThisCycle dedupes the one-time side effects of actually
	// quarantining a log (the log line and rootchain_listener_log_quarantined_total)
	// to once per logKey per cycle, independent of how many single-block
	// leaves reuse the still-live quarantine entry. It does not gate
	// logFailureCounts: quarantineRootChainLog deliberately leaves that entry
	// in place (see its doc comment) for pruneStaleLogFailureCounts to drop.
	quarantinedThisCycle map[string]struct{}

	// filterLogsCalls counts how many FilterLogs round trips this cycle has
	// made. An endpoint that keeps returning the same quarantine-excludable
	// log for every query forces bisection to visit every leaf to resolve a
	// wide chunk (see the quarantine field above) — up to 2*N-1 calls for an
	// N-block chunk. queryAndBroadcastEvents stops making new calls once this
	// crosses maxRootChainFilterLogsCallsPerCycle, so one cycle can't turn
	// into thousands of synchronous round trips to an endpoint already known
	// to be misbehaving; whatever didn't get resolved resumes next cycle.
	filterLogsCalls int
}

func newRootChainRejectionState() *rootChainRejectionState {
	return &rootChainRejectionState{
		countedThisCycle:     make(map[string]struct{}),
		quarantine:           make(map[string]*rootChainLogDetail),
		quarantinedThisCycle: make(map[string]struct{}),
	}
}

// rootChainLogKey identifies a log uniquely across every bisection level and
// poll cycle that re-encounters it, for both the per-cycle rejection dedup
// and the persistent per-log failure count.
func rootChainLogKey(vLog types.Log) string {
	return vLog.TxHash.Hex() + ":" + strconv.FormatUint(uint64(vLog.Index), 10)
}

// rootChainLogDetail carries everything an operator needs to investigate and
// manually recover a quarantined log.
type rootChainLogDetail struct {
	logKey      string
	reason      string
	txHash      ethCommon.Hash
	logIndex    uint64
	blockNumber uint64
	address     ethCommon.Address
	topic       ethCommon.Hash
}

// queryAndBroadcastEvents fetches supported events from the rootChain and handles all of them
func (rl *RootChainListener) queryAndBroadcastEvents(rootChainContext *RootChainListenerContext, fromBlock *big.Int, toBlock *big.Int, state *rootChainRejectionState) error {
	rl.Logger.Debug("RootChainListener: querying rootChain event logs", "fromBlock", fromBlock, "toBlock", toBlock)

	if rl.contractCaller.MainChainClient == nil {
		// don't advance the cursor if the client isn't ready.
		return errMainChainClientUnavailable
	}

	if state.filterLogsCalls >= maxRootChainFilterLogsCallsPerCycle {
		rl.Logger.Warn("RootChainListener: FilterLogs call budget reached for this poll cycle, resuming the rest next cycle",
			"fromBlock", fromBlock, "toBlock", toBlock, "budget", maxRootChainFilterLogsCallsPerCycle)
		return errFilterLogsCallBudgetReached
	}
	state.filterLogsCalls++

	ctx, cancel := context.WithTimeout(context.Background(), rl.contractCaller.MainChainTimeout)
	defer cancel()

	// get chain params
	chainParams := rootChainContext.ChainmanagerParams.ChainParams

	contractAddresses := map[rootChainContract]ethCommon.Address{
		rootChainContractRootChain:   ethCommon.HexToAddress(chainParams.RootChainAddress),
		rootChainContractStakingInfo: ethCommon.HexToAddress(chainParams.StakingInfoAddress),
		rootChainContractStateSender: ethCommon.HexToAddress(chainParams.StateSenderAddress),
	}

	query := ethereum.FilterQuery{
		FromBlock: fromBlock,
		ToBlock:   toBlock,
		Addresses: []ethCommon.Address{
			contractAddresses[rootChainContractRootChain],
			contractAddresses[rootChainContractStakingInfo],
			contractAddresses[rootChainContractStateSender],
		},
	}

	eventTopics := rootChainEventTopics(rl.eventMap)
	if len(eventTopics) == 0 {
		// Fail closed. Querying without topics can fetch every log from the
		// bridge contracts and then advance the cursor without handling them.
		rl.Logger.Error("RootChainListener: no supported rootChain event topics configured")
		return errNoSupportedRootChainTopics
	}
	query.Topics = [][]ethCommon.Hash{eventTopics}

	// Fetch events from the rootChain
	logs, err := rl.contractCaller.MainChainClient.FilterLogs(ctx, query)
	if err != nil {
		rl.Logger.Error("RootChainListener: error while filtering logs", "error", err)
		return err
	}

	if len(logs) > 0 {
		rl.Logger.Debug("RootChainListener: new logs found", "numberOfLogs", len(logs))
	}

	return rl.validateAndHandleLogs(logs, contractAddresses, fromBlock, toBlock, state)
}

// validateAndHandleLogs validates every log in the batch before handling any
// of it: a batch with one bad log among several good ones must not partially
// dispatch, or a retry of the same range (see processRootChainBlockRange)
// would re-dispatch the already-handled ones. The one exception is a log
// that has crossed maxRootChainLogRejections this cycle (state.quarantine) —
// it's excluded rather than failing the batch, so a real event sharing its
// block isn't permanently lost along with it. Quarantine is only actually
// committed (logged, metriced, and cleared) once the rest of the batch is
// otherwise clean: if some other log in the same batch is still failing and
// below its own threshold, the whole batch still withholds exactly as
// before, and the quarantine-eligible log stays eligible (not reset) for
// the next attempt — quarantine is per log, not a blanket give-up on the
// block. See isQuarantineExcludable for the conditions that gate exclusion.
func (rl *RootChainListener) validateAndHandleLogs(logs []types.Log, contractAddresses map[rootChainContract]ethCommon.Address, fromBlock, toBlock *big.Int, state *rootChainRejectionState) error {
	selectedEvents, dispatch, toQuarantine, unquarantinedFailure := rl.classifyRootChainLogs(logs, contractAddresses, fromBlock, toBlock, state)
	if unquarantinedFailure {
		return errUnexpectedRootChainLog
	}

	for _, logKey := range toQuarantine {
		// state.quarantine keeps the entry for the rest of the cycle (see its
		// field doc), so this same logKey can show up here again from a later
		// leaf; quarantinedThisCycle keeps the log line and metric to exactly
		// once despite that (logFailureCounts is left alone either way).
		if detail, ok := state.quarantine[logKey]; ok {
			if _, already := state.quarantinedThisCycle[logKey]; !already {
				rl.quarantineRootChainLog(detail)
				state.quarantinedThisCycle[logKey] = struct{}{}
			}
		}
	}

	for i, vLog := range logs {
		if dispatch[i] {
			rl.handleLog(vLog, selectedEvents[i])
			if rl.onLogDispatched != nil {
				rl.onLogDispatched(vLog, selectedEvents[i])
			}
		}
	}

	return nil
}

// classifyRootChainLogs validates every log against the query and sorts
// each one into exactly one bucket: dispatch (selectedEvents[i]/dispatch[i],
// safe to hand to handleLog), toQuarantine (excludable per
// isQuarantineExcludable), or neither — in which case unquarantinedFailure
// is set, telling the caller the whole batch must fail regardless of what
// else this call found.
func (rl *RootChainListener) classifyRootChainLogs(logs []types.Log, contractAddresses map[rootChainContract]ethCommon.Address, fromBlock, toBlock *big.Int, state *rootChainRejectionState) (selectedEvents []*abi.Event, dispatch []bool, toQuarantine []string, unquarantinedFailure bool) {
	selectedEvents = make([]*abi.Event, len(logs))
	dispatch = make([]bool, len(logs))

	for i, vLog := range logs {
		selectedEvent, ok := rl.validateLogAgainstQuery(vLog, contractAddresses, fromBlock, toBlock, state)
		if ok {
			selectedEvents[i] = selectedEvent
			dispatch[i] = true
			continue
		}

		logKey := rootChainLogKey(vLog)
		if isQuarantineExcludable(state, logKey, fromBlock, toBlock) {
			toQuarantine = append(toQuarantine, logKey)
			continue
		}
		unquarantinedFailure = true
	}

	return selectedEvents, dispatch, toQuarantine, unquarantinedFailure
}

// isQuarantineExcludable reports whether logKey, having already crossed the
// quarantine threshold, may be excluded from the current batch rather than
// failing it. fromBlock == toBlock is the only condition that matters:
// bisection must have narrowed to the exact single block, not some
// still-being-bisected wider range, or excluding the log would let a
// multi-block range report success — and persistLastRootBlock advance the
// cursor — while never actually re-querying every block in it. Nothing
// further is needed on the log's own claimed block: a logKey only reaches
// state.quarantine after failing content validation on its own merits
// (rejectRootChainLog, reachable only once FilterLogs has already
// succeeded — see queryAndBroadcastEvents), so it was never going to be
// dispatched regardless of which single block it's excluded at. Excluding
// it here costs nothing beyond that block's batch trusting the rest of its
// own content, exactly as an in-range quarantined log already does.
func isQuarantineExcludable(state *rootChainRejectionState, logKey string, fromBlock, toBlock *big.Int) bool {
	_, quarantined := state.quarantine[logKey]
	return quarantined && fromBlock.Cmp(toBlock) == 0
}

// validateLogAgainstQuery checks that a log returned by FilterLogs actually
// matches what was queried: a block number inside the requested range, a
// topic in the exact set of events this query asked for (not just any event
// these three contracts can ever emit), and an address matching the specific
// contract that topic's event belongs to (not just any of the three watched
// contracts — a StakingInfo-family topic must come from StakingInfoAddress,
// not merely from one of the three). FilterLogs is expected to only ever
// return matching logs; this exists so an endpoint that doesn't honor the
// filter is caught here rather than treated as a legitimate empty match and
// silently passed over.
//
// This only catches a response shaped wrong. It can't catch one that's
// merely incomplete but otherwise well-formed — the query has no independent
// way to know what should have come back. Self-heal (rootchain_selfheal.go)
// covers that gap for checkpoint acks, the three nonce-gated stake events,
// and state syncs; validator joins, top-ups, slashes, and unjails have no
// self-heal path today.
func (rl *RootChainListener) validateLogAgainstQuery(vLog types.Log, contractAddresses map[rootChainContract]ethCommon.Address, fromBlock, toBlock *big.Int, state *rootChainRejectionState) (*abi.Event, bool) {
	if vLog.Removed {
		return rl.rejectRootChainLog(state, vLog, "a removed (reorg'd) log", "txHash", vLog.TxHash)
	}

	if vLog.BlockNumber < fromBlock.Uint64() || vLog.BlockNumber > toBlock.Uint64() {
		return rl.rejectRootChainLog(state, vLog, "a log outside the requested block range",
			"blockNumber", vLog.BlockNumber, "fromBlock", fromBlock, "toBlock", toBlock, "txHash", vLog.TxHash)
	}

	if len(vLog.Topics) == 0 {
		return rl.rejectRootChainLog(state, vLog, "a log with no topics", "txHash", vLog.TxHash)
	}

	selectedEvent, ok := rl.eventMap[vLog.Topics[0]]
	if !ok {
		return rl.rejectRootChainLog(state, vLog, "a log with an unrecognized topic", "topic", vLog.Topics[0], "txHash", vLog.TxHash)
	}

	// eventMap spans every event the three ABIs define; rootChainEvents is the
	// curated subset this listener actually queried for (see
	// rootChainEventTopics). A topic can resolve in eventMap while still being
	// outside that subset, which would otherwise defeat this exact check.
	if _, ok := rootChainEvents[selectedEvent.Name]; !ok {
		return rl.rejectRootChainLog(state, vLog, "a log for an event outside the query's topic set", "event", selectedEvent.Name, "txHash", vLog.TxHash)
	}

	contract, ok := rl.eventContract[vLog.Topics[0]]
	if !ok {
		return rl.rejectRootChainLog(state, vLog, "a log for an event missing from the contract-binding map", "event", selectedEvent.Name, "txHash", vLog.TxHash)
	}

	expectedAddress := contractAddresses[contract]
	if vLog.Address != expectedAddress {
		return rl.rejectRootChainLog(state, vLog, "a log from an address that doesn't match its event's contract",
			"address", vLog.Address, "expectedAddress", expectedAddress, "event", selectedEvent.Name, "txHash", vLog.TxHash)
	}

	if !eventShapeMatches(vLog, selectedEvent) {
		return rl.rejectRootChainLog(state, vLog, "a log whose topic count or data doesn't match its event's ABI shape",
			"event", selectedEvent.Name, "numTopics", len(vLog.Topics), "dataLen", len(vLog.Data), "txHash", vLog.TxHash)
	}

	return selectedEvent, true
}

// eventShapeMatches reports whether vLog actually decodes against
// selectedEvent's ABI: the right number of indexed topics, and non-indexed
// data that unpacks cleanly. A log can pass every other check here (right
// topic0, right contract, right block range) while still being malformed in
// a way that only surfaces once handleLog's UnpackLog call fails — and that
// failure is a bare log line with no rejection, so the batch still succeeds
// and the cursor advances past an event that was never actually queued.
// Checking the shape here routes that failure through the same
// reject/bisect/quarantine path as any other malformed log instead.
func eventShapeMatches(vLog types.Log, selectedEvent *abi.Event) bool {
	indexedCount := 0
	for _, arg := range selectedEvent.Inputs {
		if arg.Indexed {
			indexedCount++
		}
	}
	if len(vLog.Topics)-1 != indexedCount {
		return false
	}
	_, err := selectedEvent.Inputs.NonIndexed().UnpackValues(vLog.Data)
	return err == nil
}

// rejectRootChainLog logs a rootchain log query rejection, always returning
// (nil, false) so validateLogAgainstQuery's reject sites read as a single
// line each. The log line, the rootchain_listener_log_rejected_total counter,
// and the log's persistent failure count all advance at most once per logKey
// per state.countedThisCycle set: processRootChainBlockRange bisects a
// failed range and re-queries every sub-range that still contains a
// persistently-bad log, so without this dedup a single bad log returned by
// every sub-query would inflate the metric — and reach the quarantine
// threshold — by roughly log2(range) within a single poll cycle, and (before
// the call budget) could otherwise print up to hundreds of duplicate error
// lines for one log in one cycle. state is shared across that whole cycle
// (see ProcessHeader) so the log, the metric, and the failure count still all
// advance once per cycle for a log that's still broken — that's honest
// signal, just not duplicated within it. Crossing maxRootChainLogRejections
// records an entry in state.quarantine keyed by this exact log, for
// validateAndHandleLogs to act on — never keyed by block number alone, so a
// different log or a transient failure on the same block can't consume it.
func (rl *RootChainListener) rejectRootChainLog(state *rootChainRejectionState, vLog types.Log, reason string, keyvals ...any) (*abi.Event, bool) {
	logKey := rootChainLogKey(vLog)
	if _, alreadyCounted := state.countedThisCycle[logKey]; alreadyCounted {
		return nil, false
	}
	if len(state.countedThisCycle) >= maxTrackedLogFailures {
		return nil, false
	}
	state.countedThisCycle[logKey] = struct{}{}

	rl.Logger.Error("RootChainListener: rootchain log query returned "+reason, keyvals...)
	metrics.RootChainListenerLogRejected.Inc()

	if rl.logFailureCounts == nil {
		rl.logFailureCounts = make(map[string]uint64)
	}
	if _, tracked := rl.logFailureCounts[logKey]; !tracked && len(rl.logFailureCounts) >= maxTrackedLogFailures {
		rl.evictLowestTrackedLogFailure()
	}
	rl.logFailureCounts[logKey]++
	if rl.logFailureCounts[logKey] >= maxRootChainLogRejections {
		var topic ethCommon.Hash
		if len(vLog.Topics) > 0 {
			topic = vLog.Topics[0]
		}
		state.quarantine[logKey] = &rootChainLogDetail{
			logKey:      logKey,
			reason:      reason,
			txHash:      vLog.TxHash,
			logIndex:    uint64(vLog.Index),
			blockNumber: vLog.BlockNumber,
			address:     vLog.Address,
			topic:       topic,
		}
	}

	return nil, false
}

// evictionSampleSize bounds how many entries evictLowestTrackedLogFailure
// inspects. Go randomizes map iteration order per call, so stopping after
// this many entries is an effective random sample without needing a
// separate shuffle — the same approach Redis uses for approximate-LRU
// eviction, for the same reason: scanning the whole keyspace to find the
// true minimum is too expensive to do on every admission.
const evictionSampleSize = 20

// evictLowestTrackedLogFailure drops the lowest-count entry among a bounded
// random sample, making room for a new one once logFailureCounts is at
// maxTrackedLogFailures. Sampling instead of scanning the whole map matters
// at this cap: a full scan costs O(maxTrackedLogFailures) per admission, so
// a single response introducing that many new logs in one poll would cost
// O(maxTrackedLogFailures²). The sample doesn't need to find the true
// global minimum either — a log that keeps recurring climbs past any
// one-off flood entry within a few cycles and earns a permanent slot
// regardless of which lower entry was evicted to make room for it, and a
// flood large enough to fill the map leaves the sample overwhelmingly
// likely to land on one of its many equally-evictable entries.
// Returns the number of entries it actually inspected, so tests can assert
// the bound directly instead of inferring it from wall-clock timing (which a
// contended or instrumented CI runner could blow through even though the
// underlying cost is still O(evictionSampleSize)).
func (rl *RootChainListener) evictLowestTrackedLogFailure() int {
	var lowestKey string
	var lowestCount uint64
	sampled := 0
	for logKey, count := range rl.logFailureCounts {
		if sampled == 0 || count < lowestCount {
			lowestKey, lowestCount = logKey, count
		}
		sampled++
		if sampled >= evictionSampleSize {
			break
		}
	}
	delete(rl.logFailureCounts, lowestKey)
	return sampled
}

// quarantineRootChainLog is called once a log has failed validation
// maxRootChainLogRejections times (not necessarily consecutively — see
// maxRootChainLogRejections's doc comment): it logs everything needed for
// manual investigation, counts it in rootchain_listener_log_quarantined_total
// (distinct from the rejection counter, so an operator can tell "still
// retrying" from "gave up, needs a human"). It deliberately does not touch
// logFailureCounts: excluding this log at one leaf doesn't mean the range
// containing it is fully resolved yet (the FilterLogs call budget can abort
// a cycle partway through a wide range), so clearing its count here could
// throw away real progress the next cycle would otherwise resume from.
// pruneStaleLogFailureCounts is the only place a count gets dropped, and
// only once a full cycle completes without needing this log again.
func (rl *RootChainListener) quarantineRootChainLog(detail *rootChainLogDetail) {
	rl.Logger.Error("RootChainListener: quarantining a rootchain log stuck failing validation",
		"reason", detail.reason,
		"txHash", detail.txHash,
		"logIndex", detail.logIndex,
		"blockNumber", detail.blockNumber,
		"address", detail.address,
		"topic", detail.topic,
		"totalFailures", maxRootChainLogRejections,
	)
	metrics.RootChainListenerLogQuarantined.Inc()
}

// pruneStaleLogFailureCounts drops any tracked failure count for a log that
// wasn't rejected again in the cycle that just finished. Because
// processRootChainBlockRangeInChunks's chunk loop and processRootChainBlockRange's
// bisection both give up immediately on the first still-failing sub-range,
// only the single earliest unresolved block can ever be under active
// investigation at a time — every count entry that isn't in countedThisCycle
// belongs to a position that has since resolved (succeeded or been
// quarantined), so this keeps logFailureCounts from growing without bound
// over the node's lifetime.
func (rl *RootChainListener) pruneStaleLogFailureCounts(countedThisCycle map[string]struct{}) {
	for logKey := range rl.logFailureCounts {
		if _, seen := countedThisCycle[logKey]; !seen {
			delete(rl.logFailureCounts, logKey)
		}
	}
}

func rootChainEventTopics(eventMap map[ethCommon.Hash]*abi.Event) []ethCommon.Hash {
	topics := make([]ethCommon.Hash, 0, len(rootChainEvents))

	for topic, event := range eventMap {
		if event == nil {
			continue
		}
		if _, ok := rootChainEvents[event.Name]; !ok {
			continue
		}
		topics = append(topics, topic)
	}

	return topics
}

func (rl *RootChainListener) SendTaskWithDelay(taskName string, eventName string, logBytes []byte, delay time.Duration, event interface{}) {
	defer util.LogElapsedTimeForStateSyncedEvent(event, "SendTaskWithDelay", time.Now())

	signature := &tasks.Signature{
		Name: taskName,
		Args: []tasks.Arg{
			{
				Type:  "string",
				Value: eventName,
			},
			{
				Type:  "string",
				Value: string(logBytes),
			},
		},
	}
	signature.RetryCount = 5

	eta := time.Now().Add(delay)
	signature.ETA = &eta
	rl.Logger.Info("RootChainListener: Sending task", "taskName", taskName, "currentTime", time.Now(), "delayTime", eta)

	_, err := rl.queueConnector.Server.SendTask(signature)
	if err != nil {
		rl.Logger.Error("RootChainListener: error sending task", "taskName", taskName, "error", err)
	}
}

// getRootChainContext returns the root chain context
func (rl *RootChainListener) getRootChainContext() (*RootChainListenerContext, error) {
	chainmanagerParams, err := util.GetChainmanagerParams(rl.cliCtx.Codec)
	if err != nil {
		rl.Logger.Error("RootChainListener: error while fetching chain manager params", "error", err)
		return nil, err
	}

	return &RootChainListenerContext{
		ChainmanagerParams: chainmanagerParams,
	}, nil
}
