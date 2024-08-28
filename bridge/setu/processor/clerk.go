package processor

import (
	"context"
	"encoding/hex"
	"time"

	"github.com/0xPolygon/heimdall-v2/bridge/setu/util"
	"github.com/0xPolygon/heimdall-v2/common/tracing"
	"github.com/0xPolygon/heimdall-v2/contracts/statesender"
	"github.com/0xPolygon/heimdall-v2/helper"
	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	chainmanagertypes "github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
	clerkTypes "github.com/0xPolygon/heimdall-v2/x/clerk/types"
	"github.com/RichardKnop/machinery/v1/tasks"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	jsoniter "github.com/json-iterator/go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

// ClerkContext for bridge
type ClerkContext struct {
	ChainmanagerParams *chainmanagertypes.Params
}

// ClerkProcessor - sync state/deposit events
type ClerkProcessor struct {
	BaseProcessor
	stateSenderAbi *abi.ABI
}

// NewClerkProcessor - add statesender abi to clerk processor
func NewClerkProcessor(stateSenderAbi *abi.ABI) *ClerkProcessor {
	return &ClerkProcessor{
		stateSenderAbi: stateSenderAbi,
	}
}

// Start starts new block subscription
func (cp *ClerkProcessor) Start() error {
	cp.Logger.Info("Starting")
	return nil
}

// RegisterTasks - Registers clerk related tasks with machinery
func (cp *ClerkProcessor) RegisterTasks() {
	cp.Logger.Info("Registering clerk tasks")

	if err := cp.queueConnector.Server.RegisterTask("sendStateSyncedToHeimdall", cp.sendStateSyncedToHeimdall); err != nil {
		cp.Logger.Error("RegisterTasks | sendStateSyncedToHeimdall", "error", err)
	}
}

// HandleStateSyncEvent - handle state sync event from rootchain
// 1. check if this deposit event has to be broadcasted to heimdall
// 2. create and broadcast  record transaction to heimdall
func (cp *ClerkProcessor) sendStateSyncedToHeimdall(eventName string, logBytes string) error {
	otelCtx := tracing.WithTracer(context.Background(), otel.Tracer("State-Sync"))
	// work begins
	sendStateSyncedToHeimdallCtx, sendStateSyncedToHeimdallSpan := tracing.StartSpan(otelCtx, "sendStateSyncedToHeimdall")
	defer tracing.EndSpan(sendStateSyncedToHeimdallSpan)

	start := time.Now()

	var vLog = types.Log{}
	if err := jsoniter.ConfigFastest.Unmarshal([]byte(logBytes), &vLog); err != nil {
		cp.Logger.Error("Error while unmarshalling event from rootchain", "error", err)
		return err
	}

	clerkContext, err := cp.getClerkContext()
	if err != nil {
		return err
	}

	chainParams := clerkContext.ChainmanagerParams.ChainParams

	event := new(statesender.StatesenderStateSynced)
	if err = helper.UnpackLog(cp.stateSenderAbi, event, eventName, &vLog); err != nil {
		cp.Logger.Error("Error while parsing event", "name", eventName, "error", err)
	} else {
		defer util.LogElapsedTimeForStateSyncedEvent(event, "sendStateSyncedToHeimdall", start)

		tracing.SetAttributes(sendStateSyncedToHeimdallSpan, []attribute.KeyValue{
			attribute.String("event", eventName),
			attribute.Int64("id", event.Id.Int64()),
			attribute.String("contract", event.ContractAddress.String()),
		}...)

		_, isOldTxSpan := tracing.StartSpan(sendStateSyncedToHeimdallCtx, "isOldTx")
		isOld, _ := cp.isOldTx(cp.cliCtx, vLog.TxHash.String(), uint64(vLog.Index), util.ClerkEvent, event)
		tracing.EndSpan(isOldTxSpan)

		if isOld {
			cp.Logger.Info("Ignoring task to send deposit to heimdall as already processed",
				"event", eventName,
				"id", event.Id,
				"contract", event.ContractAddress,
				"data", hex.EncodeToString(event.Data),
				"borChainId", chainParams.BorChainId,
				"txHash", vLog.TxHash.String(),
				"logIndex", uint64(vLog.Index),
				"blockNumber", vLog.BlockNumber,
			)

			return nil
		}

		cp.Logger.Debug(
			"⬜ New event found",
			"event", eventName,
			"id", event.Id,
			"contract", event.ContractAddress,
			"data", hex.EncodeToString(event.Data),
			"borChainId", chainParams.BorChainId,
			"txHash", vLog.TxHash.String(),
			"logIndex", uint64(vLog.Index),
			"blockNumber", vLog.BlockNumber,
		)

		_, maxStateSyncSizeCheckSpan := tracing.StartSpan(sendStateSyncedToHeimdallCtx, "maxStateSyncSizeCheck")
		if len(event.Data) > helper.MaxStateSyncSize {
			cp.Logger.Info(`Data is too large to process, Resetting to ""`, "data", hex.EncodeToString(event.Data))
			event.Data = common.FromHex("")
		}
		tracing.EndSpan(maxStateSyncSizeCheckSpan)

		msg := clerkTypes.NewMsgEventRecord(
			helper.GetAddress(),
			vLog.TxHash.String(),
			uint64(vLog.Index),
			vLog.BlockNumber,
			event.Id.Uint64(),
			event.ContractAddress.Bytes(),
			hmTypes.HexBytes{HexBytes: event.Data},
			chainParams.BorChainId,
		)

		_, checkTxAgainstMempoolSpan := tracing.StartSpan(sendStateSyncedToHeimdallCtx, "checkTxAgainstMempool")
		// Check if we have the same transaction in mempool or not
		// Don't drop the transaction. Keep retrying after `util.RetryStateSyncTaskDelay = 24 seconds`,
		// until the transaction in mempool is processed or cancelled.
		inMempool, _ := cp.checkTxAgainstMempool(&msg, event)
		tracing.EndSpan(checkTxAgainstMempoolSpan)

		if inMempool {
			cp.Logger.Info("Similar transaction already in mempool, retrying in sometime", "event", eventName, "retry delay", util.RetryStateSyncTaskDelay)
			return tasks.NewErrRetryTaskLater("transaction already in mempool", util.RetryStateSyncTaskDelay)
		}

		_, BroadcastToHeimdallSpan := tracing.StartSpan(sendStateSyncedToHeimdallCtx, "BroadcastToHeimdall")
		// return broadcast to heimdall
		err = cp.txBroadcaster.BroadcastToHeimdall(&msg, event)
		tracing.EndSpan(BroadcastToHeimdallSpan)

		if err != nil {
			cp.Logger.Error("Error while broadcasting clerk Record to heimdall", "error", err)
			return err
		}
	}

	return nil
}

//
// utils
//

func (cp *ClerkProcessor) getClerkContext() (*ClerkContext, error) {
	chainmanagerParams, err := util.GetChainmanagerParams(cp.cliCtx)
	if err != nil {
		cp.Logger.Error("Error while fetching chain manager params", "error", err)
		return nil, err
	}

	return &ClerkContext{
		ChainmanagerParams: chainmanagerParams,
	}, nil
}
