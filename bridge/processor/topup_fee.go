package processor

import (
	"context"
	"encoding/json"
	"fmt"

	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/0xPolygon/heimdall-v2/bridge/util"
	"github.com/0xPolygon/heimdall-v2/contracts/stakinginfo"
	"github.com/0xPolygon/heimdall-v2/helper"
	topupTypes "github.com/0xPolygon/heimdall-v2/x/topup/types"
)

const (
	// Error messages
	errMsgTopupUnmarshallingEvent = "FeeProcessor: error while unmarshalling event from rootChain"
	errMsgTopupParsingEvent       = "FeeProcessor: error while parsing event"
	errMsgTopupBroadcasting       = "FeeProcessor: error while broadcasting TopUpFee msg to heimdall"
	errMsgTopupTxFailed           = "FeeProcessor: topUp tx failed on heimdall"
	errMsgTopupRegisteringTask    = "FeeProcessor | RegisterTasks | sendTopUpFeeToHeimdall"

	// Info messages
	infoMsgTopupStarting         = "FeeProcessor: starting"
	infoMsgTopupRegisteringTasks = "FeeProcessor: registering fee related tasks"
	infoMsgTopupSending          = "FeeProcessor: sending top up to heimdall"

	// Debug messages
	debugMsgTopupIgnoringProcessed = "FeeProcessor: ignoring task to send top up to heimdall as already processed"
)

// FeeProcessor processes the fee-related events
type FeeProcessor struct {
	BaseProcessor
	stakingInfoAbi *abi.ABI
}

// NewFeeProcessor adds the abi to the clerk processor
func NewFeeProcessor(stakingInfoAbi *abi.ABI) *FeeProcessor {
	return &FeeProcessor{
		stakingInfoAbi: stakingInfoAbi,
	}
}

// Start starts new block subscription
func (fp *FeeProcessor) Start() error {
	fp.Logger.Info(infoMsgTopupStarting)
	return nil
}

// RegisterTasks registers the clerk-related tasks with machinery
func (fp *FeeProcessor) RegisterTasks() {
	fp.Logger.Info(infoMsgTopupRegisteringTasks)

	if err := fp.queueConnector.Server.RegisterTask("sendTopUpFeeToHeimdall", fp.sendTopUpFeeToHeimdall); err != nil {
		fp.Logger.Error(errMsgTopupRegisteringTask, "error", err)
	}
}

// sendTopUpFeeToHeimdall - processes top up fee event
func (fp *FeeProcessor) sendTopUpFeeToHeimdall(eventName string, logBytes string) error {
	vLog := types.Log{}
	if err := json.Unmarshal([]byte(logBytes), &vLog); err != nil {
		fp.Logger.Error(errMsgTopupUnmarshallingEvent, "error", err)
		return err
	}

	event := new(stakinginfo.StakinginfoTopUpFee)
	if err := helper.UnpackLog(fp.stakingInfoAbi, event, eventName, &vLog); err != nil {
		fp.Logger.Error(errMsgTopupParsingEvent, "name", eventName, "error", err)
	} else {
		if isOld, _ := fp.isOldTx(fp.cliCtx, vLog.TxHash.String(), uint64(vLog.Index), util.TopupEvent, event); isOld {
			fp.Logger.Debug(debugMsgTopupIgnoringProcessed,
				"event", eventName,
				"user", event.User,
				"Fee", event.Fee,
				"txHash", vLog.TxHash.String(),
				"logIndex", uint64(vLog.Index),
				"blockNumber", vLog.BlockNumber,
			)
			return nil
		}

		fp.Logger.Info(infoMsgTopupSending,
			"event", eventName,
			"user", event.User,
			"Fee", event.Fee,
			"txHash", vLog.TxHash.String(),
			"logIndex", uint64(vLog.Index),
			"blockNumber", vLog.BlockNumber,
		)

		// create msg checkpoint ack message
		msg := topupTypes.NewMsgTopupTx(helper.GetFromAddress(fp.cliCtx), event.User.String(), math.NewIntFromBigInt(event.Fee), vLog.TxHash.Bytes(), uint64(vLog.Index), vLog.BlockNumber)

		// return broadcast to heimdall
		txRes, err := fp.txBroadcaster.BroadcastToHeimdall(context.Background(), msg, event)
		if err != nil {
			fp.Logger.Error(errMsgTopupBroadcasting, "msg", msg, "error", err)
			return err
		}

		if txRes.Code != abci.CodeTypeOK {
			fp.Logger.Error(errMsgTopupTxFailed, "txHash", txRes.TxHash, "code", txRes.Code)
			return fmt.Errorf(errMsgTopupTxFailed+", tx response code: %v", txRes.Code)
		}

	}

	return nil
}
