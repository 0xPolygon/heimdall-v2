package processor

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"time"

	// TODO HV2 - uncomment when clerk is merged
	// clerktypes "github.com/0xPolygon/heimdall-v2/x/clerk/types"

	"cosmossdk.io/log"
	"github.com/0xPolygon/heimdall-v2/bridge/setu/broadcaster"
	"github.com/0xPolygon/heimdall-v2/bridge/setu/queue"
	"github.com/0xPolygon/heimdall-v2/bridge/setu/util"
	"github.com/0xPolygon/heimdall-v2/helper"
	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authlegacytx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	jsoniter "github.com/json-iterator/go"
	"github.com/spf13/viper"
	"github.com/syndtr/goleveldb/leveldb"
)

// Processor defines a block header listener for Rootchain, Maticchain, Heimdall
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
	queueConnector *queue.QueueConnector

	// tx broadcaster
	txBroadcaster *broadcaster.TxBroadcaster

	// The "subclass" of BaseProcessor
	impl Processor

	// cli context
	cliCtx client.Context

	// contract caller
	contractConnector helper.ContractCaller

	// http client to subscribe to
	httpClient *rpchttp.HTTP

	// storage client
	storageClient *leveldb.DB
}

// Logger returns logger singleton instance
func Logger(ctx context.Context, name string) log.Logger {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.Logger().With("service", "processor", "module", name)
}

// NewBaseProcessor creates a new BaseProcessor.
func NewBaseProcessor(cdc codec.Codec, queueConnector *queue.QueueConnector, httpClient *rpchttp.HTTP, txBroadcaster *broadcaster.TxBroadcaster, name string, impl Processor) *BaseProcessor {
	logger := Logger(context.Background(), name)

	cliCtx := client.Context{}.WithCodec(cdc)
	cliCtx.BroadcastMode = flags.BroadcastSync

	contractCaller, err := helper.NewContractCaller()
	if err != nil {
		logger.Error("Error while getting root chain instance", "error", err)
		panic(err)
	}

	if logger == nil {
		logger = log.NewNopLogger()
	}

	// creating syncer object
	return &BaseProcessor{
		Logger: logger,
		name:   name,
		quit:   make(chan struct{}),
		impl:   impl,

		cliCtx:            cliCtx,
		queueConnector:    queueConnector,
		contractConnector: contractCaller,
		txBroadcaster:     txBroadcaster,
		httpClient:        httpClient,
		storageClient:     util.GetBridgeDBInstance(viper.GetString(util.BridgeDBFlag)),
	}
}

// String implements Service by returning a string representation of the service.
func (bp *BaseProcessor) String() string {
	return bp.name
}

// OnStop stops all necessary go routines
func (bp *BaseProcessor) Stop() {
	// override to stop any go-routines in individual processors
}

// isOldTx checks if the transaction already exists in the chain or not
// It is a generic function, which is consumed in all processors
func (bp *BaseProcessor) isOldTx(_ client.Context, txHash string, logIndex uint64, eventType util.BridgeEvent, event interface{}) (bool, error) {
	defer util.LogElapsedTimeForStateSyncedEvent(event, "isOldTx", time.Now())

	queryParam := map[string]interface{}{
		"txhash":   txHash,
		"logindex": logIndex,
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
	case util.SlashingEvent:
		endpoint = helper.GetHeimdallServerEndpoint(util.SlashingTxStatusURL)
	}

	// TODO HV2 - uncomment when we uncomment the below `helper.FetchFromAPI` call
	// url, err := util.CreateURLWithQuery(endpoint, queryParam)
	_, err := util.CreateURLWithQuery(endpoint, queryParam)
	if err != nil {
		bp.Logger.Error("Error in creating url", "endpoint", endpoint, "error", err)
		return false, err
	}

	// TODO HV2 Please uncomment the following fn once it is uncommented in helper.
	/*
		res, err := helper.FetchFromAPI(bp.cliCtx, url)
		if err != nil {
			bp.Logger.Error("Error fetching tx status", "url", url, "error", err)
			return false, err
		}
	*/

	// TODO HV2 - This is a place holder, remove when the above function is uncommented.
	var res struct{ Result []byte }

	var status bool
	if err := jsoniter.ConfigFastest.Unmarshal(res.Result, &status); err != nil {
		bp.Logger.Error("Error unmarshalling tx status received from Heimdall Server", "error", err)
		return false, err
	}

	return status, nil
}

// checkTxAgainstMempool checks if the transaction is already in the mempool or not
// It is consumed only for `clerk` processor
func (bp *BaseProcessor) checkTxAgainstMempool(msg types.Msg, event interface{}) (bool, error) {
	defer util.LogElapsedTimeForStateSyncedEvent(event, "checkTxAgainstMempool", time.Now())

	endpoint := helper.GetConfig().CometBFTRPCUrl + util.CometBFTUnconfirmedTxsURL

	resp, err := helper.Client.Get(endpoint)
	if err != nil || resp.StatusCode != http.StatusOK {
		bp.Logger.Error("Error fetching mempool tx", "url", endpoint, "error", err)
		return false, err
	}

	body, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()

	if err != nil {
		bp.Logger.Error("Error fetching mempool tx", "error", err)
		return false, err
	}

	// a minimal response of the unconfirmed txs
	var response util.CometBFTUnconfirmedTxs

	err = jsoniter.ConfigFastest.Unmarshal(body, &response)
	if err != nil {
		bp.Logger.Error("Error unmarshalling response received from Heimdall Server", "error", err)
		return false, err
	}

	// Iterate over txs present in the mempool
	// We can verify if the message we're about to send is present by
	// checking the type of transaction, the transaction hash and log index
	// present in the data of transaction

	status := false
Loop:
	for _, txn := range response.Result.Txs {
		// CometBFT encodes the transactions with base64 encoding. Decode it first.
		txBytes, err := base64.StdEncoding.DecodeString(txn)
		if err != nil {
			bp.Logger.Error("Error decoding tx (base64 decoder) while checking against mempool", "error", err)
			continue
		}

		// Unmarshal the transaction from bytes
		decodedTx, err := authlegacytx.DefaultTxDecoder(bp.cliCtx.Codec)(txBytes)
		if err != nil {
			bp.Logger.Error("Error decoding tx (tx decoder) while checking against mempool", "error", err)
			continue
		}
		txMsg := decodedTx.GetMsgs()[0]

		// We only need to check for `event-record` type transactions.
		// If required, add case for others here.
		switch txMsg.String() {
		case "event-record":

			// TODO HV2 - uncomment when clerk is merged
			/*
				// typecast the txs for clerk type message
				mempoolTxMsg, ok := txMsg.(clerkTypes.MsgEventRecord)
				if !ok {
					bp.Logger.Error("Unable to typecast message to clerk event record while checking against mempool")
					continue Loop
				}

				// typecast the msg for clerk type message
				clerkMsg, ok := msg.(clerkTypes.MsgEventRecord)
				if !ok {
					bp.Logger.Error("Unable to typecast message to clerk event record while checking against mempool")
					continue Loop
				}

				// check the transaction hash in message
				if clerkMsg.GetTxHash() != mempoolTxMsg.GetTxHash() {
					continue Loop
				}

				// check the log index in the message
				if clerkMsg.GetLogIndex() != mempoolTxMsg.GetLogIndex() {
					continue Loop
				}
			*/

			// If we reach here, there's already a same transaction in the mempool
			status = true
			break Loop
		default:
			// ignore
		}
	}

	return status, nil
}
