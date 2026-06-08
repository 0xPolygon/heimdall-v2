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
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/0xPolygon/heimdall-v2/bridge/util"
	"github.com/0xPolygon/heimdall-v2/helper"
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

	// For self-healing, it will be only initialized if sub_graph_url is provided
	subGraphClient *subGraphClient
}

const (
	lastRootBlockKey       = "rootchain-last-block" // Storage key
	maxRootChainBlockRange = 5000                   // Maximum number of blocks to fetch logs for in a single FilterLogs call
)

var (
	errMainChainClientUnavailable = errors.New("main chain client is nil")
	errNoSupportedRootChainTopics = errors.New("no supported rootChain event topics configured")

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

	abis := []*abi.ABI{
		&contractCaller.RootChainABI,
		&contractCaller.StateSenderABI,
		&contractCaller.StakingInfoABI,
	}

	eventMap := make(map[ethCommon.Hash]*abi.Event)
	for _, abiObj := range abis {
		for _, event := range abiObj.Events {
			e := event
			eventMap[e.ID] = &e
		}
	}

	return &RootChainListener{
		stakingInfoAbi: &contractCaller.StakingInfoABI,
		stateSenderAbi: &contractCaller.StateSenderABI,
		eventMap:       eventMap,
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

	// Start the per-node checkpoint-lag backstop monitor
	go rl.startCheckpointLagMonitor(ctx)

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

	query := ethereum.FilterQuery{
		FromBlock: fromBlock,
		ToBlock:   toBlock,
		Addresses: []ethCommon.Address{
			ethCommon.HexToAddress(chainParams.RootChainAddress),
			ethCommon.HexToAddress(chainParams.StakingInfoAddress),
			ethCommon.HexToAddress(chainParams.StateSenderAddress),
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

	for _, vLog := range logs {
		if len(vLog.Topics) == 0 {
			continue
		}

		selectedEvent, ok := rl.eventMap[vLog.Topics[0]]
		if !ok {
			continue
		}

		rl.handleLog(vLog, selectedEvent)
	}

	return nil
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
