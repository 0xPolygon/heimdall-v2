package listener

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"time"

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

// validatorSlot is this node's turn among the validators that submit the same
// root chain event to heimdall, so they don't all broadcast it at once.
type validatorSlot struct {
	isValidator bool
	delay       time.Duration
}

func (rl *RootChainListener) currentValidatorSlot() validatorSlot {
	isValidator, delay := util.CalculateTaskDelay(nil, rl.cliCtx.Codec)
	return validatorSlot{isValidator: isValidator, delay: delay}
}

// handleLog enqueues the processor task for a validated root chain log. The
// validator that emitted the event on L1 submits it first when that's this
// node; otherwise current validators submit it in slot order.
func (rl *RootChainListener) handleLog(vLog types.Log, selectedEvent *abi.Event, slot validatorSlot) {
	spec, ok := rootChainEvents[selectedEvent.Name]
	if !ok {
		return
	}

	rl.Logger.Debug("RootChainListener: receivedEvent", "eventName", selectedEvent.Name)

	logBytes, err := json.Marshal(vLog)
	if err != nil {
		rl.Logger.Error(failedToMarshalLog, "error", err)
		return
	}

	event, isSender, err := rl.decodeEventSender(vLog, selectedEvent)
	if err != nil {
		rl.Logger.Error(failedToParseEventLog, "name", selectedEvent.Name, "error", err)
		return
	}

	var delay time.Duration
	switch {
	case isSender:
	case slot.isValidator:
		delay = slot.delay
	default:
		return
	}

	start := time.Now()
	_ = rl.sendTask(spec.task, rootChainTaskRetryCount, delay+spec.extraDelay, stringArg(selectedEvent.Name), stringArg(string(logBytes)))
	util.LogElapsedTimeForStateSyncedEvent(event, "sendTask", start)
}

// decodeEventSender decodes vLog and reports whether this node's validator
// emitted it on L1. The decoded event feeds the state sync timing logs.
func (rl *RootChainListener) decodeEventSender(vLog types.Log, selectedEvent *abi.Event) (interface{}, bool, error) {
	switch selectedEvent.Name {
	case helper.StakedEvent:
		event, err := unpackEvent[stakinginfo.StakinginfoStaked](rl.stakingInfoAbi, &vLog, selectedEvent.Name)
		if err != nil {
			return nil, false, err
		}

		pubKey := helper.GetPubKey()
		if len(pubKey) < 1 {
			return nil, false, errors.New("public key length is invalid")
		}
		if !helper.IsPubKeyFirstByteValid(pubKey[0:1]) {
			rl.Logger.Error("RootChainListener: public key first byte mismatch", "expected", "0x04", "received", pubKey[0:1])
		}

		return event, bytes.Equal(event.SignerPubkey, pubKey[1:]), nil

	case helper.StakeUpdateEvent:
		event, err := unpackEvent[stakinginfo.StakinginfoStakeUpdate](rl.stakingInfoAbi, &vLog, selectedEvent.Name)
		if err != nil {
			return nil, false, err
		}
		return event, util.IsEventSender(event.ValidatorId.Uint64(), rl.cliCtx.Codec), nil

	case helper.SignerChangeEvent:
		event, err := unpackEvent[stakinginfo.StakinginfoSignerChange](rl.stakingInfoAbi, &vLog, selectedEvent.Name)
		if err != nil {
			return nil, false, err
		}
		pubKey := helper.GetPubKey()
		return event, bytes.Equal(event.SignerPubkey, pubKey[1:]) && helper.IsPubKeyFirstByteValid(pubKey[0:1]), nil

	case helper.UnstakeInitEvent:
		event, err := unpackEvent[stakinginfo.StakinginfoUnstakeInit](rl.stakingInfoAbi, &vLog, selectedEvent.Name)
		if err != nil {
			return nil, false, err
		}
		return event, util.IsEventSender(event.ValidatorId.Uint64(), rl.cliCtx.Codec), nil

	case helper.StateSyncedEvent:
		event, err := unpackEvent[statesender.StatesenderStateSynced](rl.stateSenderAbi, &vLog, selectedEvent.Name)
		if err != nil {
			return nil, false, err
		}
		rl.Logger.Info("RootChainListener: StateSyncedEvent detected", "stateSyncId", event.Id)
		return event, false, nil

	case helper.TopUpFeeEvent:
		event, err := unpackEvent[stakinginfo.StakinginfoTopUpFee](rl.stakingInfoAbi, &vLog, selectedEvent.Name)
		if err != nil {
			return nil, false, err
		}
		return event, bytes.Equal(event.User.Bytes(), helper.GetAddress()), nil

	case helper.NewHeaderBlockEvent:
		return nil, false, nil
	}

	return nil, false, fmt.Errorf("no decoder for root chain event %s", selectedEvent.Name)
}

func unpackEvent[T any](contractABI *abi.ABI, vLog *types.Log, eventName string) (*T, error) {
	event := new(T)
	if err := helper.UnpackLog(contractABI, event, eventName, vLog); err != nil {
		return nil, err
	}
	return event, nil
}
