package processor

import (
	"encoding/json"
	"fmt"
	"time"

	"cosmossdk.io/math"
	"github.com/0xPolygon/heimdall-v2/bridge/util"
	"github.com/0xPolygon/heimdall-v2/contracts/stakinginfo"
	"github.com/0xPolygon/heimdall-v2/helper"
	stakingTypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
	"github.com/RichardKnop/machinery/v1/tasks"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authTx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/types"
)

const (
	defaultDelayDuration = 15 * time.Second
)

// StakingProcessor - process staking related events
type StakingProcessor struct {
	BaseProcessor
	stakingInfoAbi *abi.ABI
}

// NewStakingProcessor - add  abi to staking processor
func NewStakingProcessor(stakingInfoAbi *abi.ABI) *StakingProcessor {
	return &StakingProcessor{
		stakingInfoAbi: stakingInfoAbi,
	}
}

// Start starts new block subscription
func (sp *StakingProcessor) Start() error {
	sp.Logger.Info("Starting")
	return nil
}

// RegisterTasks - Registers staking tasks with machinery
func (sp *StakingProcessor) RegisterTasks() {
	sp.Logger.Info("Registering staking related tasks")

	if err := sp.queueConnector.Server.RegisterTask("sendValidatorJoinToHeimdall", sp.sendValidatorJoinToHeimdall); err != nil {
		sp.Logger.Error("RegisterTasks | sendValidatorJoinToHeimdall", "error", err)
	}

	if err := sp.queueConnector.Server.RegisterTask("sendUnstakeInitToHeimdall", sp.sendUnstakeInitToHeimdall); err != nil {
		sp.Logger.Error("RegisterTasks | sendUnstakeInitToHeimdall", "error", err)
	}

	if err := sp.queueConnector.Server.RegisterTask("sendStakeUpdateToHeimdall", sp.sendStakeUpdateToHeimdall); err != nil {
		sp.Logger.Error("RegisterTasks | sendStakeUpdateToHeimdall", "error", err)
	}

	if err := sp.queueConnector.Server.RegisterTask("sendSignerChangeToHeimdall", sp.sendSignerChangeToHeimdall); err != nil {
		sp.Logger.Error("RegisterTasks | sendSignerChangeToHeimdall", "error", err)
	}
}

func (sp *StakingProcessor) sendValidatorJoinToHeimdall(eventName string, logBytes string) error {
	vLog := types.Log{}
	if err := json.Unmarshal([]byte(logBytes), &vLog); err != nil {
		sp.Logger.Error("Error while unmarshalling event from rootchain", "error", err)
		return err
	}

	event := new(stakinginfo.StakinginfoStaked)
	if err := helper.UnpackLog(sp.stakingInfoAbi, event, eventName, &vLog); err != nil {
		sp.Logger.Error("Error while parsing event", "name", eventName, "error", err)
	} else {
		signerPubKey := event.SignerPubkey
		if len(signerPubKey) == 64 {
			signerPubKey = util.AppendPrefix(signerPubKey)
		}
		if isOld, _ := sp.isOldTx(sp.cliCtx, vLog.TxHash.String(), uint64(vLog.Index), util.StakingEvent, event); isOld {
			sp.Logger.Info("Ignoring task to send validatorjoin to heimdall as already processed",
				"event", eventName,
				"validatorID", event.ValidatorId,
				"activationEpoch", event.ActivationEpoch,
				"nonce", event.Nonce,
				"amount", event.Amount,
				"totalAmount", event.Total,
				"SignerPubkey", string(signerPubKey[:]),
				"txHash", vLog.TxHash.String(),
				"logIndex", uint64(vLog.Index),
				"blockNumber", vLog.BlockNumber,
			)
			return nil
		}

		// if account doesn't exist Retry with delay for topup to process first.
		if _, err := util.GetAccount(sp.cliCtx, event.Signer.String()); err != nil {
			sp.Logger.Info(
				"Heimdall Account doesn't exist. Retrying validator-join after 10 seconds",
				"event", eventName,
				"signer", event.Signer,
			)
			return tasks.NewErrRetryTaskLater("account doesn't exist", util.RetryTaskDelay)
		}

		sp.Logger.Info(
			"✅ Received task to send validatorjoin to heimdall",
			"event", eventName,
			"validatorID", event.ValidatorId,
			"activationEpoch", event.ActivationEpoch,
			"nonce", event.Nonce,
			"amount", event.Amount,
			"totalAmount", event.Total,
			"SignerPubkey", string(signerPubKey[:]),
			"txHash", vLog.TxHash.String(),
			"logIndex", uint64(vLog.Index),
			"blockNumber", vLog.BlockNumber,
		)

		// msg validator join
		msg, err := stakingTypes.NewMsgValidatorJoin(
			string(helper.GetAddress()[:]),
			event.ValidatorId.Uint64(),
			event.ActivationEpoch.Uint64(),
			math.NewIntFromBigInt(event.Amount),
			&secp256k1.PubKey{Key: signerPubKey},
			vLog.TxHash.Bytes(),
			uint64(vLog.Index),
			vLog.BlockNumber,
			event.Nonce.Uint64(),
		)
		if err != nil {
			sp.Logger.Error("Error while creating msg for validator join", "error", err)
			return err
		}

		// return broadcast to heimdall
		txRes, err := sp.txBroadcaster.BroadcastToHeimdall(msg, event)
		if err != nil {
			sp.Logger.Error("Error while broadcasting unstakeInit to heimdall", "validatorId", event.ValidatorId.Uint64(), "error", err)
			return err
		}

		if txRes.Code != abci.CodeTypeOK {
			sp.Logger.Error("validator-join tx failed on heimdall", "txHash", txRes.TxHash, "code", txRes.Code)
			return fmt.Errorf("validator-join tx failed, tx response code: %v", txRes.Code)
		}

	}

	return nil
}

func (sp *StakingProcessor) sendUnstakeInitToHeimdall(eventName string, logBytes string) error {
	vLog := types.Log{}
	if err := json.Unmarshal([]byte(logBytes), &vLog); err != nil {
		sp.Logger.Error("Error while unmarshalling event from rootchain", "error", err)
		return err
	}

	event := new(stakinginfo.StakinginfoUnstakeInit)
	if err := helper.UnpackLog(sp.stakingInfoAbi, event, eventName, &vLog); err != nil {
		sp.Logger.Error("Error while parsing event", "name", eventName, "error", err)
	} else {
		if isOld, _ := sp.isOldTx(sp.cliCtx, vLog.TxHash.String(), uint64(vLog.Index), util.StakingEvent, event); isOld {
			sp.Logger.Info("Ignoring task to send unstakeinit to heimdall as already processed",
				"event", eventName,
				"validator", event.User,
				"validatorID", event.ValidatorId,
				"nonce", event.Nonce,
				"deactivatonEpoch", event.DeactivationEpoch,
				"amount", event.Amount,
				"txHash", vLog.TxHash.String(),
				"logIndex", uint64(vLog.Index),
				"blockNumber", vLog.BlockNumber,
			)
			return nil
		}

		validNonce, nonceDelay, err := sp.checkValidNonce(event.ValidatorId.Uint64(), event.Nonce.Uint64())
		if err != nil {
			sp.Logger.Error("Error while validating nonce for the validator", "error", err)
			return err
		}

		if !validNonce {
			sp.Logger.Info("Ignoring task to send unstake-init to heimdall as nonce is out of order")
			return tasks.NewErrRetryTaskLater("Nonce out of order", defaultDelayDuration*time.Duration(nonceDelay))
		}

		sp.Logger.Info(
			"✅ Received task to send unstake-init to heimdall",
			"event", eventName,
			"validator", event.User,
			"validatorID", event.ValidatorId,
			"nonce", event.Nonce,
			"deactivatonEpoch", event.DeactivationEpoch,
			"amount", event.Amount,
			"txHash", vLog.TxHash.String(),
			"logIndex", uint64(vLog.Index),
			"blockNumber", vLog.BlockNumber,
		)

		// msg validator exit
		msg, err := stakingTypes.NewMsgValidatorExit(
			string(helper.GetAddress()[:]),
			event.ValidatorId.Uint64(),
			event.DeactivationEpoch.Uint64(),
			vLog.TxHash.Bytes(),
			uint64(vLog.Index),
			vLog.BlockNumber,
			event.Nonce.Uint64(),
		)
		if err != nil {
			sp.Logger.Error("Error while creating new MsgValidatorExit", "error", err)
			return err

		}

		// return broadcast to heimdall
		txRes, err := sp.txBroadcaster.BroadcastToHeimdall(msg, event)
		if err != nil {
			sp.Logger.Error("Error while broadcasting unstakeInit to heimdall", "validatorId", event.ValidatorId.Uint64(), "error", err)
			return err
		}

		if txRes.Code != abci.CodeTypeOK {
			sp.Logger.Error("unstakeInit tx failed on heimdall", "txHash", txRes.TxHash, "code", txRes.Code)
			return fmt.Errorf("unstakeInit tx failed, tx response code: %v", txRes.Code)
		}

	}

	return nil
}

func (sp *StakingProcessor) sendStakeUpdateToHeimdall(eventName string, logBytes string) error {
	vLog := types.Log{}
	if err := json.Unmarshal([]byte(logBytes), &vLog); err != nil {
		sp.Logger.Error("Error while unmarshalling event from rootchain", "error", err)
		return err
	}

	event := new(stakinginfo.StakinginfoStakeUpdate)
	if err := helper.UnpackLog(sp.stakingInfoAbi, event, eventName, &vLog); err != nil {
		sp.Logger.Error("Error while parsing event", "name", eventName, "error", err)
	} else {
		if isOld, _ := sp.isOldTx(sp.cliCtx, vLog.TxHash.String(), uint64(vLog.Index), util.StakingEvent, event); isOld {
			sp.Logger.Info("Ignoring task to send unstakeinit to heimdall as already processed",
				"event", eventName,
				"validatorID", event.ValidatorId,
				"nonce", event.Nonce,
				"newAmount", event.NewAmount,
				"txHash", vLog.TxHash.String(),
				"logIndex", uint64(vLog.Index),
				"blockNumber", vLog.BlockNumber,
			)
			return nil
		}

		validNonce, nonceDelay, err := sp.checkValidNonce(event.ValidatorId.Uint64(), event.Nonce.Uint64())
		if err != nil {
			sp.Logger.Error("Error while validating nonce for the validator", "error", err)
			return err
		}

		if !validNonce {
			sp.Logger.Info("Ignoring task to send stake-update to heimdall as nonce is out of order")
			return tasks.NewErrRetryTaskLater("Nonce out of order", defaultDelayDuration*time.Duration(nonceDelay))
		}

		sp.Logger.Info(
			"✅ Received task to send stake-update to heimdall",
			"event", eventName,
			"validatorID", event.ValidatorId,
			"nonce", event.Nonce,
			"newAmount", event.NewAmount,
			"txHash", vLog.TxHash.String(),
			"logIndex", uint64(vLog.Index),
			"blockNumber", vLog.BlockNumber,
		)

		// msg validator update
		msg, err := stakingTypes.NewMsgStakeUpdate(
			string(helper.GetAddress()[:]),
			event.ValidatorId.Uint64(),
			math.NewIntFromBigInt(event.NewAmount),
			vLog.TxHash.Bytes(),
			uint64(vLog.Index),
			vLog.BlockNumber,
			event.Nonce.Uint64(),
		)
		if err != nil {
			sp.Logger.Error("Error while creating new MsgStakeUpdate", "error", err)
			return err
		}

		// return broadcast to heimdall
		txRes, err := sp.txBroadcaster.BroadcastToHeimdall(msg, event)
		if err != nil {
			sp.Logger.Error("Error while broadcasting stakeupdate to heimdall", "validatorId", event.ValidatorId.Uint64(), "error", err)
			return err
		}

		if txRes.Code != abci.CodeTypeOK {
			sp.Logger.Error("stakeupdate tx failed on heimdall", "txHash", txRes.TxHash, "code", txRes.Code)
			return fmt.Errorf("stakeupdate tx failed, tx response code: %d", txRes.Code)
		}

	}

	return nil
}

func (sp *StakingProcessor) sendSignerChangeToHeimdall(eventName string, logBytes string) error {
	vLog := types.Log{}
	if err := json.Unmarshal([]byte(logBytes), &vLog); err != nil {
		sp.Logger.Error("Error while unmarshalling event from rootchain", "error", err)
		return err
	}

	event := new(stakinginfo.StakinginfoSignerChange)
	if err := helper.UnpackLog(sp.stakingInfoAbi, event, eventName, &vLog); err != nil {
		sp.Logger.Error("Error while parsing event", "name", eventName, "error", err)
	} else {
		newSignerPubKey := event.SignerPubkey
		if len(newSignerPubKey) == 64 {
			newSignerPubKey = util.AppendPrefix(newSignerPubKey)
		}

		if !util.IsPubKeyFirstByteValid(newSignerPubKey) {
			sp.Logger.Error("Invalid signer pubkey", "event", eventName, "newSignerPubKey", newSignerPubKey)
			return fmt.Errorf("invalid signer pubkey")
		}

		if isOld, err := sp.isOldTx(sp.cliCtx, vLog.TxHash.String(), uint64(vLog.Index), util.StakingEvent, event); isOld {
			sp.Logger.Info("Ignoring task to send unstakeinit to heimdall as already processed",
				"event", eventName,
				"validatorID", event.ValidatorId,
				"nonce", event.Nonce,
				"NewSignerPubkey", string(newSignerPubKey[:]),
				"oldSigner", event.OldSigner.Hex(),
				"newSigner", event.NewSigner.Hex(),
				"txHash", vLog.TxHash.String(),
				"logIndex", uint64(vLog.Index),
				"blockNumber", vLog.BlockNumber,
			)
			return nil
		} else if err != nil {
			sp.Logger.Error("Error while checking if tx is old", "error", err)
			return err
		}

		validNonce, nonceDelay, err := sp.checkValidNonce(event.ValidatorId.Uint64(), event.Nonce.Uint64())
		if err != nil {
			sp.Logger.Error("Error while validating nonce for the validator", "error", err)
			return err
		}

		if !validNonce {
			sp.Logger.Info("Ignoring task to send signer-change to heimdall as nonce is out of order")
			return tasks.NewErrRetryTaskLater("Nonce out of order", defaultDelayDuration*time.Duration(nonceDelay))
		}

		sp.Logger.Info(
			"✅ Received task to send signer-change to heimdall",
			"event", eventName,
			"validatorID", event.ValidatorId,
			"nonce", event.Nonce,
			"NewSignerPubkey", string(newSignerPubKey[:]),
			"oldSigner", event.OldSigner.Hex(),
			"newSigner", event.NewSigner.Hex(),
			"txHash", vLog.TxHash.String(),
			"logIndex", uint64(vLog.Index),
			"blockNumber", vLog.BlockNumber,
		)

		// signer change
		msg, err := stakingTypes.NewMsgSignerUpdate(
			string(helper.GetAddress()[:]),
			event.ValidatorId.Uint64(),
			newSignerPubKey,
			vLog.TxHash.Bytes(),
			uint64(vLog.Index),
			vLog.BlockNumber,
			event.Nonce.Uint64(),
		)
		if err != nil {
			sp.Logger.Error("Error while creating new MsgSignerUpdate", "error", err)
			return err
		}

		// return broadcast to heimdall
		txRes, err := sp.txBroadcaster.BroadcastToHeimdall(msg, event)
		if err != nil {
			sp.Logger.Error("Error while broadcasting signerChainge to heimdall", "msg", msg, "validatorId", event.ValidatorId.Uint64(), "error", err)
			return err
		}

		if txRes.Code != abci.CodeTypeOK {
			sp.Logger.Error("signerChange tx failed on heimdall", "txHash", txRes.TxHash, "code", txRes.Code)
			return fmt.Errorf("signerChange tx failed, tx response code: %v", txRes.Code)
		}

	}

	return nil
}

func (sp *StakingProcessor) checkValidNonce(validatorID uint64, txnNonce uint64) (bool, uint64, error) {
	currentNonce, err := util.GetValidatorNonce(validatorID)
	if err != nil {
		sp.Logger.Error("Failed to fetch validator nonce and height data from API", "validatorID", validatorID)
		return false, 0, err
	}

	if currentNonce+1 != txnNonce {
		diff := txnNonce - currentNonce
		if diff > 10 {
			diff = 10
		}

		sp.Logger.Error("Nonce for the given event not in order", "validatorID", validatorID, "currentNonce", currentNonce, "txnNonce", txnNonce, "delay", diff*uint64(defaultDelayDuration))

		return false, diff, nil
	}

	stakingTxnCount, err := queryTxCount(sp.cliCtx, validatorID)
	if err != nil {
		sp.Logger.Error("Failed to query stake txns by txquery for the given validator", "validatorID", validatorID)
		return false, 0, err
	}

	if stakingTxnCount != 0 {
		sp.Logger.Info("Recent staking txn count for the given validator is not zero", "validatorID", validatorID, "currentNonce", currentNonce, "txnNonce", txnNonce)
		return false, 1, nil
	}

	return true, 0, nil
}

// TODO HV2 - this function was modified a bit, please review carefully
func queryTxCount(cliCtx client.Context, validatorID uint64) (int, error) {
	const (
		defaultPage  = 1
		defaultLimit = 100
	)

	stakingTxnMsgMap := map[string]string{
		"validator-stake-update": "stake-update",
		"validator-join":         "validator-join",
		"signer-update":          "signer-update",
		"validator-exit":         "validator-exit",
	}

	for msg, action := range stakingTxnMsgMap {
		event1 := fmt.Sprintf("%s.%s='%s'", sdk.EventTypeMessage, sdk.AttributeKeyAction, msg)

		event2 := fmt.Sprintf("%s.%s=%d", action, "validator-id", validatorID)

		searchResult1, err1 := authTx.QueryTxsByEvents(cliCtx, defaultPage, defaultLimit, event1, "")
		searchResult2, err2 := authTx.QueryTxsByEvents(cliCtx, defaultPage, defaultLimit, event2, "")
		if err1 != nil && err2 != nil {
			return 0, fmt.Errorf("%w %w", err1, err2)
		}

		var totalCount uint64

		// Check if searchResult1 is not nil before accessing TotalCount
		if searchResult1 != nil {
			totalCount += searchResult1.TotalCount
		}

		// Check if searchResult2 is not nil before accessing TotalCount
		if searchResult2 != nil {
			totalCount += searchResult2.TotalCount
		}

		// Only return if totalCount is non-zero
		if totalCount != 0 {
			return int(totalCount), nil
		}

	}

	return 0, nil
}
