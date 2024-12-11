package processor

import (
	"encoding/json"
	"fmt"

	"cosmossdk.io/math"
	"github.com/0xPolygon/heimdall-v2/bridge/util"
	"github.com/0xPolygon/heimdall-v2/contracts/stakinginfo"
	"github.com/0xPolygon/heimdall-v2/helper"
	topupTypes "github.com/0xPolygon/heimdall-v2/x/topup/types"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/types"
)

// FeeProcessor - process fee related events
type FeeProcessor struct {
	BaseProcessor
	stakingInfoAbi *abi.ABI
}

// NewFeeProcessor - add  abi to clerk processor
func NewFeeProcessor(stakingInfoAbi *abi.ABI) *FeeProcessor {
	return &FeeProcessor{
		stakingInfoAbi: stakingInfoAbi,
	}
}

// Start starts new block subscription
func (fp *FeeProcessor) Start() error {
	fp.Logger.Info("Starting")
	return nil
}

// RegisterTasks - Registers clerk related tasks with machinery
func (fp *FeeProcessor) RegisterTasks() {
	fp.Logger.Info("Registering fee related tasks")

	if err := fp.queueConnector.Server.RegisterTask("sendTopUpFeeToHeimdall", fp.sendTopUpFeeToHeimdall); err != nil {
		fp.Logger.Error("RegisterTasks | sendTopUpFeeToHeimdall", "error", err)
	}
}

// processTopupFeeEvent - processes topup fee event
func (fp *FeeProcessor) sendTopUpFeeToHeimdall(eventName string, logBytes string) error {
	vLog := types.Log{}
	if err := json.Unmarshal([]byte(logBytes), &vLog); err != nil {
		fp.Logger.Error("Error while unmarshalling event from rootchain", "error", err)
		return err
	}

	event := new(stakinginfo.StakinginfoTopUpFee)
	if err := helper.UnpackLog(fp.stakingInfoAbi, event, eventName, &vLog); err != nil {
		fp.Logger.Error("Error while parsing event", "name", eventName, "error", err)
	} else {
		if isOld, _ := fp.isOldTx(fp.cliCtx, vLog.TxHash.String(), uint64(vLog.Index), util.TopupEvent, event); isOld {
			fp.Logger.Info("Ignoring task to send topup to heimdall as already processed",
				"event", eventName,
				"user", event.User,
				"Fee", event.Fee,
				"txHash", vLog.TxHash.String(),
				"logIndex", uint64(vLog.Index),
				"blockNumber", vLog.BlockNumber,
			)
			return nil
		}

		fp.Logger.Info("✅ sending topup to heimdall",
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
		txRes, err := fp.txBroadcaster.BroadcastToHeimdall(msg, event)
		if err != nil {
			fp.Logger.Error("Error while broadcasting TopupFee msg to heimdall", "msg", msg, "error", err)
			return err
		}

		if txRes.Code != abci.CodeTypeOK {
			fp.Logger.Error("topup tx failed on heimdall", "txHash", txRes.TxHash, "code", txRes.Code)
			return fmt.Errorf("topup tx failed, tx response code: %v", txRes.Code)
		}

	}

	return nil
}
