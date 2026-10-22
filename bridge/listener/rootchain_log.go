package listener

import (
	"bytes"
	"encoding/json"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/0xPolygon/heimdall-v2/bridge/util"
	"github.com/0xPolygon/heimdall-v2/contracts/stakinginfo"
	"github.com/0xPolygon/heimdall-v2/contracts/statesender"
	"github.com/0xPolygon/heimdall-v2/helper"
)

const (
	failedToMarshalLog    = "RootChainListener: failed to marshal log"
	failedToParseEventLog = "RootChainListener: error while parsing event"
)

// handleLog handles the given log
func (rl *RootChainListener) handleLog(vLog types.Log, selectedEvent *abi.Event) {
	rl.Logger.Debug("RootChainListener: receivedEvent", "eventName", selectedEvent.Name)

	switch selectedEvent.Name {
	case helper.NewHeaderBlockEvent:
		rl.handleNewHeaderBlockLog(vLog, selectedEvent)
	case helper.StakedEvent:
		rl.handleStakedLog(vLog, selectedEvent)
	case helper.StakeUpdateEvent:
		rl.handleStakeUpdateLog(vLog, selectedEvent)
	case helper.SignerChangeEvent:
		rl.handleSignerChangeLog(vLog, selectedEvent)
	case helper.UnstakeInitEvent:
		rl.handleUnstakeInitLog(vLog, selectedEvent)
	case helper.StateSyncedEvent:
		rl.handleStateSyncedLog(vLog, selectedEvent)
	case helper.TopUpFeeEvent:
		rl.handleTopUpFeeLog(vLog, selectedEvent)
	}
}

func (rl *RootChainListener) handleNewHeaderBlockLog(vLog types.Log, selectedEvent *abi.Event) {
	logBytes, err := json.Marshal(vLog)
	if err != nil {
		rl.Logger.Error(failedToMarshalLog, "error", err)
		return
	}

	if isCurrentValidator, delay := util.CalculateTaskDelay(selectedEvent, rl.cliCtx.Codec); isCurrentValidator {
		_ = rl.SendTaskWithDelay("sendCheckpointAckToHeimdall", selectedEvent.Name, logBytes, delay, selectedEvent)
	}
}

func (rl *RootChainListener) handleStakedLog(vLog types.Log, selectedEvent *abi.Event) {
	logBytes, err := json.Marshal(vLog)
	if err != nil {
		rl.Logger.Error(failedToMarshalLog, "error", err)
		return
	}

	pubKey := helper.GetPubKey()

	event := new(stakinginfo.StakinginfoStaked)
	if err = helper.UnpackLog(rl.stakingInfoAbi, event, selectedEvent.Name, &vLog); err != nil {
		rl.Logger.Error(failedToParseEventLog, "name", selectedEvent.Name, "error", err)
		return
	}

	if len(pubKey) < 1 {
		rl.Logger.Error("RootChainListener: public key length is invalid", "length", len(pubKey))
		return
	}

	if !helper.IsPubKeyFirstByteValid(pubKey[0:1]) {
		rl.Logger.Error("RootChainListener: public key first byte mismatch", "expected", "0x04", "received", pubKey[0:1])
	}

	if bytes.Equal(event.SignerPubkey, pubKey[1:]) {
		delay := util.TaskDelayBetweenEachVal
		_ = rl.SendTaskWithDelay("sendValidatorJoinToHeimdall", selectedEvent.Name, logBytes, delay, event)
	} else if isCurrentValidator, delay := util.CalculateTaskDelay(event, rl.cliCtx.Codec); isCurrentValidator {
		delay = delay + util.TaskDelayBetweenEachVal
		_ = rl.SendTaskWithDelay("sendValidatorJoinToHeimdall", selectedEvent.Name, logBytes, delay, event)
	}
}

func (rl *RootChainListener) handleStakeUpdateLog(vLog types.Log, selectedEvent *abi.Event) {
	logBytes, err := json.Marshal(vLog)
	if err != nil {
		rl.Logger.Error(failedToMarshalLog, "error", err)
		return
	}

	event := new(stakinginfo.StakinginfoStakeUpdate)
	if err = helper.UnpackLog(rl.stakingInfoAbi, event, selectedEvent.Name, &vLog); err != nil {
		rl.Logger.Error(failedToParseEventLog, "name", selectedEvent.Name, "error", err)
		return
	}

	if util.IsEventSender(event.ValidatorId.Uint64(), rl.cliCtx.Codec) {
		_ = rl.SendTaskWithDelay("sendStakeUpdateToHeimdall", selectedEvent.Name, logBytes, 0, event)
	} else if isCurrentValidator, delay := util.CalculateTaskDelay(event, rl.cliCtx.Codec); isCurrentValidator {
		_ = rl.SendTaskWithDelay("sendStakeUpdateToHeimdall", selectedEvent.Name, logBytes, delay, event)
	}
}

func (rl *RootChainListener) handleSignerChangeLog(vLog types.Log, selectedEvent *abi.Event) {
	logBytes, err := json.Marshal(vLog)
	if err != nil {
		rl.Logger.Error(failedToMarshalLog, "error", err)
		return
	}

	pubKey := helper.GetPubKey()

	event := new(stakinginfo.StakinginfoSignerChange)
	if err = helper.UnpackLog(rl.stakingInfoAbi, event, selectedEvent.Name, &vLog); err != nil {
		rl.Logger.Error(failedToParseEventLog, "name", selectedEvent.Name, "error", err)
		return
	}

	if bytes.Equal(event.SignerPubkey, pubKey[1:]) && helper.IsPubKeyFirstByteValid(pubKey[0:1]) {
		_ = rl.SendTaskWithDelay("sendSignerChangeToHeimdall", selectedEvent.Name, logBytes, 0, event)
	} else if isCurrentValidator, delay := util.CalculateTaskDelay(event, rl.cliCtx.Codec); isCurrentValidator {
		_ = rl.SendTaskWithDelay("sendSignerChangeToHeimdall", selectedEvent.Name, logBytes, delay, event)
	}
}

func (rl *RootChainListener) handleUnstakeInitLog(vLog types.Log, selectedEvent *abi.Event) {
	logBytes, err := json.Marshal(vLog)
	if err != nil {
		rl.Logger.Error(failedToMarshalLog, "error", err)
		return
	}

	event := new(stakinginfo.StakinginfoUnstakeInit)
	if err = helper.UnpackLog(rl.stakingInfoAbi, event, selectedEvent.Name, &vLog); err != nil {
		rl.Logger.Error(failedToParseEventLog, "name", selectedEvent.Name, "error", err)
		return
	}

	if util.IsEventSender(event.ValidatorId.Uint64(), rl.cliCtx.Codec) {
		_ = rl.SendTaskWithDelay("sendUnstakeInitToHeimdall", selectedEvent.Name, logBytes, 0, event)
	} else if isCurrentValidator, delay := util.CalculateTaskDelay(event, rl.cliCtx.Codec); isCurrentValidator {
		_ = rl.SendTaskWithDelay("sendUnstakeInitToHeimdall", selectedEvent.Name, logBytes, delay, event)
	}
}

func (rl *RootChainListener) handleStateSyncedLog(vLog types.Log, selectedEvent *abi.Event) {
	logBytes, err := json.Marshal(vLog)
	if err != nil {
		rl.Logger.Error(failedToMarshalLog, "error", err)
		return
	}

	event := new(statesender.StatesenderStateSynced)
	if err = helper.UnpackLog(rl.stateSenderAbi, event, selectedEvent.Name, &vLog); err != nil {
		rl.Logger.Error(failedToParseEventLog, "name", selectedEvent.Name, "error", err)
		return
	}

	rl.Logger.Info("RootChainListener: StateSyncedEvent detected", "stateSyncId", event.Id)

	if isCurrentValidator, delay := util.CalculateTaskDelay(event, rl.cliCtx.Codec); isCurrentValidator {
		_ = rl.SendTaskWithDelay("sendStateSyncedToHeimdall", selectedEvent.Name, logBytes, delay, event)
	}
}

func (rl *RootChainListener) handleTopUpFeeLog(vLog types.Log, selectedEvent *abi.Event) {
	logBytes, err := json.Marshal(vLog)
	if err != nil {
		rl.Logger.Error(failedToMarshalLog, "error", err)
		return
	}

	event := new(stakinginfo.StakinginfoTopUpFee)
	if err = helper.UnpackLog(rl.stakingInfoAbi, event, selectedEvent.Name, &vLog); err != nil {
		rl.Logger.Error(failedToParseEventLog, "name", selectedEvent.Name, "error", err)
		return
	}

	if bytes.Equal(event.User.Bytes(), helper.GetAddress()) {
		_ = rl.SendTaskWithDelay("sendTopUpFeeToHeimdall", selectedEvent.Name, logBytes, 0, event)
	} else if isCurrentValidator, delay := util.CalculateTaskDelay(event, rl.cliCtx.Codec); isCurrentValidator {
		_ = rl.SendTaskWithDelay("sendTopUpFeeToHeimdall", selectedEvent.Name, logBytes, delay, event)
	}
}
