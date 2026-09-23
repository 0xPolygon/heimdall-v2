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

	// For self-healing, it will be only initialized if sub_graph_url is provided
	subGraphClient *subGraphClient
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

var (
	errMainChainClientUnavailable = errors.New("main chain client is nil")
	errNoSupportedRootChainTopics = errors.New("no supported rootChain event topics configured")
	errUnexpectedRootChainLog     = errors.New("rootchain log query returned a log that doesn't match the query")

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
		stakingInfoAbi: &contractCaller.StakingInfoABI,
		stateSenderAbi: &contractCaller.StateSenderABI,
		eventMap:       eventMap,
		eventContract:  eventContract,
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

	requiredConfirmations := rootChainContext.ChainmanagerParams.MainChainTxConfirmations
	headerNumber := newHeader.header.Number
	from := headerNumber

	// If the incoming header is a `finalized` header, it can directly be considered as
	// the upper cap (i.e., the `to` value)
	//
	// If the incoming header is a `latest` header, rely on `requiredConfirmations` to get
	// finalized block range.
	if !newHeader.isFinalized {
		// This check is only useful when the L1 blocks received are < requiredConfirmations
		// just for the below headerNumber -= requiredConfirmations math operation
		confirmationBlocks := big.NewInt(0).SetUint64(requiredConfirmations)
		if headerNumber.Cmp(confirmationBlocks) <= 0 {
			rl.Logger.Error("RootChainListener: block number less than confirmations required", "blockNumber", headerNumber.Uint64, "confirmationsRequired", confirmationBlocks.Uint64)
			return
		}

		// subtract the `confirmationBlocks` to only consider blocks before that
		headerNumber = headerNumber.Sub(headerNumber, confirmationBlocks)

		// update the `from` value
		from = headerNumber
	}

	// get the last block from storage
	hasLastBlock, _ := rl.storageClient.Has([]byte(lastRootBlockKey), nil)
	if hasLastBlock {
		lastBlockBytes, err := rl.storageClient.Get([]byte(lastRootBlockKey), nil)
		if err != nil {
			rl.Logger.Error("RootChainListener: error while fetching last block bytes from storage", "error", err)
			return
		}

		rl.Logger.Debug("RootChainListener: got last block from bridge storage", "lastBlock", string(lastBlockBytes))

		if result, err := strconv.ParseUint(string(lastBlockBytes), 10, 64); err == nil {
			if result >= headerNumber.Uint64() {
				return
			}

			from = big.NewInt(0).SetUint64(result + 1)
		}
	}

	to := headerNumber

	// Prepare block range
	if to.Cmp(from) == -1 {
		from = to
	}

	// process logs in chunks to avoid oversized FilterLogs responses
	for chunkFrom := new(big.Int).Set(from); chunkFrom.Cmp(to) <= 0; {
		chunkTo := new(big.Int).Add(chunkFrom, big.NewInt(maxRootChainBlockRange-1))
		if chunkTo.Cmp(to) > 0 {
			chunkTo = to
		}

		if err := rl.processRootChainBlockRange(rootChainContext, chunkFrom, chunkTo); err != nil {
			rl.Logger.Error(
				"queryAndBroadcastEvents failed",
				"error", err,
				"from", chunkFrom,
				"to", chunkTo,
			)
			// do not advance the cursor, as we want to retry this range on the next header
			return
		}

		chunkFrom = new(big.Int).Add(chunkTo, big.NewInt(1))
	}
}

// processRootChainBlockRange queries and handles logs for a block range. If the
// range fails, it is split into smaller ranges until either processing succeeds
// or a single-block query fails. The root block cursor is advanced only after the
// current range has been fully processed.
func (rl *RootChainListener) processRootChainBlockRange(rootChainContext *RootChainListenerContext, fromBlock *big.Int, toBlock *big.Int) error {
	if err := rl.queryAndBroadcastEvents(rootChainContext, fromBlock, toBlock); err != nil {
		// A single-block failure cannot be split further. Return the error so
		// the caller keeps the cursor unchanged and retries this block later.
		if fromBlock.Cmp(toBlock) >= 0 {
			return err
		}

		// Split the failed range and retry smaller ranges. If the left half
		// also fails, it will be split again by the recursive call below.
		midBlock := splitBlockRange(fromBlock, toBlock)
		rl.Logger.Warn(
			"RootChainListener: splitting rootChain event log query after RPC failure",
			"error", err,
			"fromBlock", fromBlock,
			"toBlock", toBlock,
			"leftToBlock", midBlock,
		)

		// Process the earlier half first to preserve root-chain block order.
		if err := rl.processRootChainBlockRange(rootChainContext, fromBlock, midBlock); err != nil {
			return err
		}

		// Process the later half only after the earlier half has succeeded.
		nextBlock := new(big.Int).Add(midBlock, big.NewInt(1))
		return rl.processRootChainBlockRange(rootChainContext, nextBlock, toBlock)
	}

	// Persist only after the full range has been handled successfully.
	return rl.persistLastRootBlock(toBlock)
}

func (rl *RootChainListener) persistLastRootBlock(block *big.Int) error {
	if err := rl.storageClient.Put([]byte(lastRootBlockKey), []byte(block.String()), nil); err != nil {
		rl.Logger.Error("RootChainListener: error persisting last root block in storage", "error", err, "lastRootBlock", block.String())
		return err
	}

	return nil
}

// queryAndBroadcastEvents fetches supported events from the rootChain and handles all of them
func (rl *RootChainListener) queryAndBroadcastEvents(rootChainContext *RootChainListenerContext, fromBlock *big.Int, toBlock *big.Int) error {
	rl.Logger.Debug("RootChainListener: querying rootChain event logs", "fromBlock", fromBlock, "toBlock", toBlock)

	if rl.contractCaller.MainChainClient == nil {
		// don't advance the cursor if the client isn't ready.
		return errMainChainClientUnavailable
	}

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

	return rl.validateAndHandleLogs(logs, contractAddresses, fromBlock, toBlock)
}

// validateAndHandleLogs validates every log in the batch before handling any
// of it: a batch with one bad log among several good ones must not partially
// dispatch, or a retry of the same range (see processRootChainBlockRange)
// would re-dispatch the already-handled ones.
func (rl *RootChainListener) validateAndHandleLogs(logs []types.Log, contractAddresses map[rootChainContract]ethCommon.Address, fromBlock, toBlock *big.Int) error {
	selectedEvents := make([]*abi.Event, len(logs))

	for i, vLog := range logs {
		selectedEvent, ok := rl.validateLogAgainstQuery(vLog, contractAddresses, fromBlock, toBlock)
		if !ok {
			return errUnexpectedRootChainLog
		}

		selectedEvents[i] = selectedEvent
	}

	for i, vLog := range logs {
		rl.handleLog(vLog, selectedEvents[i])
	}

	return nil
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
func (rl *RootChainListener) validateLogAgainstQuery(vLog types.Log, contractAddresses map[rootChainContract]ethCommon.Address, fromBlock, toBlock *big.Int) (*abi.Event, bool) {
	if vLog.Removed {
		return rl.rejectRootChainLog("a removed (reorg'd) log", "txHash", vLog.TxHash)
	}

	if vLog.BlockNumber < fromBlock.Uint64() || vLog.BlockNumber > toBlock.Uint64() {
		return rl.rejectRootChainLog("a log outside the requested block range",
			"blockNumber", vLog.BlockNumber, "fromBlock", fromBlock, "toBlock", toBlock, "txHash", vLog.TxHash)
	}

	if len(vLog.Topics) == 0 {
		return rl.rejectRootChainLog("a log with no topics", "txHash", vLog.TxHash)
	}

	selectedEvent, ok := rl.eventMap[vLog.Topics[0]]
	if !ok {
		return rl.rejectRootChainLog("a log with an unrecognized topic", "topic", vLog.Topics[0], "txHash", vLog.TxHash)
	}

	// eventMap spans every event the three ABIs define; rootChainEvents is the
	// curated subset this listener actually queried for (see
	// rootChainEventTopics). A topic can resolve in eventMap while still being
	// outside that subset, which would otherwise defeat this exact check.
	if _, ok := rootChainEvents[selectedEvent.Name]; !ok {
		return rl.rejectRootChainLog("a log for an event outside the query's topic set", "event", selectedEvent.Name, "txHash", vLog.TxHash)
	}

	expectedAddress := contractAddresses[rl.eventContract[vLog.Topics[0]]]
	if vLog.Address != expectedAddress {
		return rl.rejectRootChainLog("a log from an address that doesn't match its event's contract",
			"address", vLog.Address, "expectedAddress", expectedAddress, "event", selectedEvent.Name, "txHash", vLog.TxHash)
	}

	return selectedEvent, true
}

// rejectRootChainLog logs and counts a rootchain log query rejection, always
// returning (nil, false) so validateLogAgainstQuery's reject sites read as a
// single line each.
func (rl *RootChainListener) rejectRootChainLog(reason string, keyvals ...any) (*abi.Event, bool) {
	rl.Logger.Error("RootChainListener: rootchain log query returned "+reason, keyvals...)
	metrics.RootChainListenerLogRejected.Inc()
	return nil, false
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

func splitBlockRange(fromBlock *big.Int, toBlock *big.Int) *big.Int {
	return new(big.Int).Add(
		fromBlock,
		new(big.Int).Div(new(big.Int).Sub(toBlock, fromBlock), big.NewInt(2)),
	)
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
