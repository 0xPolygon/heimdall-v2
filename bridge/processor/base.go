package processor

import (
	"context"
	"time"

	"cosmossdk.io/log"
	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types"
	authlegacytx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/spf13/viper"
	"github.com/syndtr/goleveldb/leveldb"

	"github.com/0xPolygon/heimdall-v2/bridge/broadcaster"
	"github.com/0xPolygon/heimdall-v2/bridge/queue"
	"github.com/0xPolygon/heimdall-v2/bridge/util"
	"github.com/0xPolygon/heimdall-v2/helper"
	clerkTypes "github.com/0xPolygon/heimdall-v2/x/clerk/types"
	staketypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
	topupTypes "github.com/0xPolygon/heimdall-v2/x/topup/types"
)

const errorUnmarshallingTxStatus = "BaseProcessor: error unmarshalling tx status received from heimdall server"

// Processor defines a block header listener for RootChain, BorChain, Heimdall
type Processor interface {
	Start() error

	RegisterTasks()

	String() string

	Stop()
}

type BaseProcessor struct {
	Logger log.Logger
	name   string
	quit   chan struct{}

	// queue connector
	queueConnector *queue.Connector

	// tx broadcaster
	txBroadcaster *broadcaster.TxBroadcaster

	// The "subclass" of BaseProcessor
	impl Processor

	// cli context
	cliCtx client.Context

	// contract caller
	contractCaller helper.ContractCaller

	// http client to subscribe to
	httpClient *rpchttp.HTTP

	// storage client
	storageClient *leveldb.DB
}

// Logger returns logger singleton instance
func Logger(name string) log.Logger {
	return util.Logger().With("service", "processor", "module", name)
}

// NewBaseProcessor creates a new BaseProcessor.
func NewBaseProcessor(cdc codec.Codec, queueConnector *queue.Connector, httpClient *rpchttp.HTTP, txBroadcaster *broadcaster.TxBroadcaster, name string, impl Processor) *BaseProcessor {
	logger := Logger(name)

	cliCtx := client.Context{}.WithCodec(cdc)
	cliCtx.BroadcastMode = flags.BroadcastSync
	cmt := helper.GetConfig().CometBFTRPCUrl
	rpc, err := client.NewClientFromNode(cmt)
	if err != nil {
		logger.Error("BaseProcessor: error while creating rpc client", "error", err)
		panic(err)
	}
	cliCtx = cliCtx.WithClient(rpc)

	contractCaller, err := helper.NewContractCaller()
	if err != nil {
		logger.Error("BaseProcessor: error while getting root chain instance", "error", err)
		panic(err)
	}

	if logger == nil {
		logger = helper.Logger.With("module", "bridge/processor")
	}

	// creating the syncer object
	return &BaseProcessor{
		Logger: logger,
		name:   name,
		quit:   make(chan struct{}),
		impl:   impl,

		cliCtx:         cliCtx,
		queueConnector: queueConnector,
		contractCaller: contractCaller,
		txBroadcaster:  txBroadcaster,
		httpClient:     httpClient,
		storageClient:  util.GetBridgeDBInstance(viper.GetString(util.BridgeDBFlag)),
	}
}

// String implements Service by returning a string representation of the service.
func (bp *BaseProcessor) String() string {
	return bp.name
}

// Stop stops all necessary go routines
func (bp *BaseProcessor) Stop() {
	// override to stop any go-routines in individual processors
}

// isOldTx checks if the transaction already exists in the chain or not
// It is a generic function, which is consumed in all processors
func (bp *BaseProcessor) isOldTx(_ client.Context, txHash string, logIndex uint64, eventType util.BridgeEvent, event interface{}) (bool, error) {
	defer util.LogElapsedTimeForStateSyncedEvent(event, "isOldTx", time.Now())

	queryParam := map[string]interface{}{
		"tx_hash":   txHash,
		"log_index": logIndex,
	}

	// define the endpoint based on the type of event
	var endpoint string

	switch eventType {
	case util.StakingEvent:
		endpoint = helper.GetHeimdallServerEndpoint(util.StakingTxStatusURL)
	case util.TopupEvent:
		endpoint = helper.GetHeimdallServerEndpoint(util.TopupTxStatusURL)
	case util.ClerkEvent:
		endpoint = helper.GetHeimdallServerEndpoint(util.ClerkTxStatusURL)
	default:
		bp.Logger.Error("BaseProcessor: invalid event type", "event", eventType)
	}

	url, err := util.CreateURLWithQuery(endpoint, queryParam)
	if err != nil {
		bp.Logger.Error("BaseProcessor: error in creating url", "endpoint", endpoint, "error", err)
		return false, err
	}

	res, err := helper.FetchFromAPI(url)
	if err != nil {
		bp.Logger.Error("BaseProcessor: error fetching tx status", "url", url, "error", err)
		return false, err
	}

	switch eventType {
	case util.StakingEvent:
		var response staketypes.QueryStakeIsOldTxResponse
		if err := bp.cliCtx.Codec.UnmarshalJSON(res, &response); err != nil {
			bp.Logger.Error(errorUnmarshallingTxStatus, "error", err)
			return false, err
		}
		return response.IsOld, nil
	case util.TopupEvent:
		var response topupTypes.QueryIsTopupTxOldResponse
		if err := bp.cliCtx.Codec.UnmarshalJSON(res, &response); err != nil {
			bp.Logger.Error(errorUnmarshallingTxStatus, "error", err)
			return false, err
		}
		return response.IsOld, nil
	case util.ClerkEvent:
		var response clerkTypes.IsClerkTxOldResponse
		if err := bp.cliCtx.Codec.UnmarshalJSON(res, &response); err != nil {
			bp.Logger.Error(errorUnmarshallingTxStatus, "error", err)
			return false, err
		}
		return response.IsOld, nil
	default:
		bp.Logger.Error("BaseProcessor: invalid event type", "event", eventType)
	}

	return false, nil
}

// checkTxAgainstMempool checks if the transaction is already in the mempool or not.
// It is consumed only for `clerk` processor.
func (bp *BaseProcessor) checkTxAgainstMempool(msg types.Msg, event interface{}) (bool, error) {
	defer util.LogElapsedTimeForStateSyncedEvent(event, "checkTxAgainstMempool", time.Now())

	clerkMsg, ok := msg.(*clerkTypes.MsgEventRecord)
	if !ok {
		return false, nil
	}

	targetTxHash := clerkMsg.GetTxHash()
	targetLogIndex := clerkMsg.GetLogIndex()

	// Use the typed CometBFT RPC client instead of raw HTTP + manual JSON/base64 decoding.
	// The client returns txs as already-decoded []byte slices.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := bp.httpClient.UnconfirmedTxs(ctx, nil)
	if err != nil {
		bp.Logger.Error("BaseProcessor: error fetching unconfirmed txs", "error", err)
		return false, err
	}

	for _, txBytes := range result.Txs {
		decodedTx, err := authlegacytx.DefaultTxDecoder(bp.cliCtx.Codec)(txBytes)
		if err != nil {
			continue
		}

		msgs := decodedTx.GetMsgs()
		if len(msgs) == 0 {
			continue
		}

		mempoolClerkMsg, ok := msgs[0].(*clerkTypes.MsgEventRecord)
		if !ok {
			continue
		}

		if mempoolClerkMsg.TxHash == targetTxHash && mempoolClerkMsg.LogIndex == targetLogIndex {
			bp.Logger.Debug("Duplicate clerk tx found in mempool", "txHash", targetTxHash, "logIndex", targetLogIndex)
			return true, nil
		}
	}

	return false, nil
}
