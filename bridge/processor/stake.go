package processor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"cosmossdk.io/math"
	"github.com/RichardKnop/machinery/v1/tasks"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authTx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/0xPolygon/heimdall-v2/bridge/util"
	"github.com/0xPolygon/heimdall-v2/contracts/stakinginfo"
	"github.com/0xPolygon/heimdall-v2/helper"
	stakingTypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

const (
	defaultDelayDuration = 15 * time.Second

	// Error messages
	errMsgUnmarshallingEvent  = "StakingProcessor: error while unmarshalling event from rootChain"
	errMsgParsingEvent        = "StakingProcessor: error while parsing event"
	errMsgValidatingNonce     = "StakingProcessor: error while validating nonce for the validator"
	errMsgCreatingMsg         = "StakingProcessor: error while creating new message"
	errMsgBroadcasting        = "StakingProcessor: error while broadcasting to heimdall"
	errMsgTxFailed            = "StakingProcessor: tx failed on heimdall"
	errMsgConvertingAddress   = "StakingProcessor: error converting address to string"
	errMsgCheckingOldTx       = "StakingProcessor: error while checking if tx is old"
	errMsgInvalidSignerPubkey = "StakingProcessor: invalid signer pubkey"
	errMsgFetchValidatorNonce = "StakingProcessor: failed to fetch validator nonce and height data from API"
	errMsgNonceNotInOrder     = "StakingProcessor: nonce for the given event not in order"
	errMsgQueryStakeTxs       = "StakingProcessor: failed to query stake txs by txQuery for the given validator"
	errMsgSearchTxs           = "StakingProcessor: failed to search for txs"

	// Info messages
	infoMsgIgnoringAlreadyProcessed = "StakingProcessor: ignoring task to send %s to heimdall as already processed"
	infoMsgIgnoringNonceOutOfOrder  = "StakingProcessor: ignoring task to send %s to heimdall as nonce is out of order"
	infoMsgAccountDoesNotExist      = "StakingProcessor: heimdall account doesn't exist. Retrying validator-join after 10 seconds"
	infoMsgRecentStakingTxnNotZero  = "StakingProcessor: recent staking txn count for the given validator is not zero"

	// Other messages
	msgNonceOutOfOrder     = "StakingProcessor: nonce out of order"
	msgAccountDoesNotExist = "StakingProcessor: account doesn't exist"
)

// StakingProcessor - process staking related events
type StakingProcessor struct {
	BaseProcessor
	stakingInfoAbi *abi.ABI
}

// NewStakingProcessor adds the abi to staking processor
func NewStakingProcessor(stakingInfoAbi *abi.ABI) *StakingProcessor {
	return &StakingProcessor{
		stakingInfoAbi: stakingInfoAbi,
	}
}

// Start starts new block subscription
func (sp *StakingProcessor) Start() error {
	sp.Logger.Info("StakingProcessor: starting")
	return nil
}

// RegisterTasks - Registers staking tasks with machinery
func (sp *StakingProcessor) RegisterTasks() {
	sp.Logger.Info("StakingProcessor: registering staking related tasks")

	if err := sp.queueConnector.Server.RegisterTask("sendValidatorJoinToHeimdall", sp.sendValidatorJoinToHeimdall); err != nil {
		sp.Logger.Error("StakingProcessor | RegisterTasks | sendValidatorJoinToHeimdall", "error", err)
	}

	if err := sp.queueConnector.Server.RegisterTask("sendUnstakeInitToHeimdall", sp.sendUnstakeInitToHeimdall); err != nil {
		sp.Logger.Error("StakingProcessor | RegisterTasks | sendUnstakeInitToHeimdall", "error", err)
	}

	if err := sp.queueConnector.Server.RegisterTask("sendStakeUpdateToHeimdall", sp.sendStakeUpdateToHeimdall); err != nil {
		sp.Logger.Error("StakingProcessor | RegisterTasks | sendStakeUpdateToHeimdall", "error", err)
	}

	if err := sp.queueConnector.Server.RegisterTask("sendSignerChangeToHeimdall", sp.sendSignerChangeToHeimdall); err != nil {
		sp.Logger.Error("StakingProcessor | RegisterTasks | sendSignerChangeToHeimdall", "error", err)
	}
}

func (sp *StakingProcessor) sendValidatorJoinToHeimdall(eventName string, logBytes string) error {
	vLog := types.Log{}
	if err := json.Unmarshal([]byte(logBytes), &vLog); err != nil {
		sp.Logger.Error(errMsgUnmarshallingEvent, "error", err)
		return err
	}

	event := new(stakinginfo.StakinginfoStaked)
	if err := helper.UnpackLog(sp.stakingInfoAbi, event, eventName, &vLog); err != nil {
		sp.Logger.Error(errMsgParsingEvent, "name", eventName, "error", err)
	} else {
		signerPubKey := event.SignerPubkey
		if len(signerPubKey) == 64 {
			signerPubKey = util.AppendPrefix(signerPubKey)
		}
		if isOld, _ := sp.isOldTx(sp.cliCtx, vLog.TxHash.String(), uint64(vLog.Index), util.StakingEvent, event); isOld {
			sp.Logger.Info(fmt.Sprintf(infoMsgIgnoringAlreadyProcessed, "validatorJoin"),
				"event", eventName,
				"validatorID", event.ValidatorId,
				"activationEpoch", event.ActivationEpoch,
				"nonce", event.Nonce,
				"amount", event.Amount,
				"totalAmount", event.Total,
				"SignerPubkey", common.Bytes2Hex(signerPubKey),
				"txHash", vLog.TxHash.String(),
				"logIndex", uint64(vLog.Index),
				"blockNumber", vLog.BlockNumber,
			)
			return nil
		}

		// if the account doesn't exist, retry with delay for top-up to process first.
		if _, err := util.GetAccount(context.Background(), sp.cliCtx, sdk.MustAccAddressFromHex(event.Signer.Hex()).String()); err != nil {
			sp.Logger.Info(
				infoMsgAccountDoesNotExist,
				"event", eventName,
				"signer", event.Signer,
			)
			return tasks.NewErrRetryTaskLater(msgAccountDoesNotExist, util.RetryTaskDelay)
		}

		sp.Logger.Info(
			helper.LogReceivedTaskToSend("validatorJoin"),
			"event", eventName,
			"validatorID", event.ValidatorId,
			"activationEpoch", event.ActivationEpoch,
			"nonce", event.Nonce,
			"amount", event.Amount,
			"totalAmount", event.Total,
			"SignerPubkey", common.Bytes2Hex(signerPubKey),
			"txHash", vLog.TxHash.String(),
			"logIndex", uint64(vLog.Index),
			"blockNumber", vLog.BlockNumber,
		)

		address, err := helper.GetAddressString()
		if err != nil {
			return fmt.Errorf("%s: %w", errMsgConvertingAddress, err)
		}

		// msg validator join
		msg, err := stakingTypes.NewMsgValidatorJoin(
			address,
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
			sp.Logger.Error(errMsgCreatingMsg, "error", err)
			return err
		}

		// return broadcast to heimdall
		txRes, err := sp.txBroadcaster.BroadcastToHeimdall(context.Background(), msg, event)
		if err != nil {
			sp.Logger.Error(errMsgBroadcasting, "validatorId", event.ValidatorId.Uint64(), "error", err)
			return err
		}

		if txRes.Code != abci.CodeTypeOK {
			sp.Logger.Error(errMsgTxFailed, "txHash", txRes.TxHash, "code", txRes.Code)
			return fmt.Errorf("validator-join %s, tx response code: %v", errMsgTxFailed, txRes.Code)
		}

	}

	return nil
}

func (sp *StakingProcessor) sendUnstakeInitToHeimdall(eventName string, logBytes string) error {
	vLog := types.Log{}
	if err := json.Unmarshal([]byte(logBytes), &vLog); err != nil {
		sp.Logger.Error(errMsgUnmarshallingEvent, "error", err)
		return err
	}

	event := new(stakinginfo.StakinginfoUnstakeInit)
	if err := helper.UnpackLog(sp.stakingInfoAbi, event, eventName, &vLog); err != nil {
		sp.Logger.Error(errMsgParsingEvent, "name", eventName, "error", err)
	} else {
		if isOld, _ := sp.isOldTx(sp.cliCtx, vLog.TxHash.String(), uint64(vLog.Index), util.StakingEvent, event); isOld {
			sp.Logger.Info(fmt.Sprintf(infoMsgIgnoringAlreadyProcessed, "unStakeInit"),
				"event", eventName,
				"validator", event.User,
				"validatorID", event.ValidatorId,
				"nonce", event.Nonce,
				"deactivationEpoch", event.DeactivationEpoch,
				"amount", event.Amount,
				"txHash", vLog.TxHash.String(),
				"logIndex", uint64(vLog.Index),
				"blockNumber", vLog.BlockNumber,
			)
			return nil
		}

		validNonce, nonceDelay, err := sp.checkValidNonce(event.ValidatorId.Uint64(), event.Nonce.Uint64())
		if err != nil {
			sp.Logger.Error(errMsgValidatingNonce, "error", err)
			return err
		}

		if !validNonce {
			sp.Logger.Info(fmt.Sprintf(infoMsgIgnoringNonceOutOfOrder, "unStakeInit"))
			return tasks.NewErrRetryTaskLater(msgNonceOutOfOrder, defaultDelayDuration*time.Duration(nonceDelay))
		}

		sp.Logger.Info(
			helper.LogReceivedTaskToSend("unStakeInit"),
			"event", eventName,
			"validator", event.User,
			"validatorID", event.ValidatorId,
			"nonce", event.Nonce,
			"deactivationEpoch", event.DeactivationEpoch,
			"amount", event.Amount,
			"txHash", vLog.TxHash.String(),
			"logIndex", uint64(vLog.Index),
			"blockNumber", vLog.BlockNumber,
		)

		address, err := helper.GetAddressString()
		if err != nil {
			return fmt.Errorf("%s: %w", errMsgConvertingAddress, err)
		}

		// msg validator exit
		msg, err := stakingTypes.NewMsgValidatorExit(
			address,
			event.ValidatorId.Uint64(),
			event.DeactivationEpoch.Uint64(),
			vLog.TxHash.Bytes(),
			uint64(vLog.Index),
			vLog.BlockNumber,
			event.Nonce.Uint64(),
		)
		if err != nil {
			sp.Logger.Error(errMsgCreatingMsg, "error", err)
			return err

		}

		// return broadcast to heimdall
		txRes, err := sp.txBroadcaster.BroadcastToHeimdall(context.Background(), msg, event)
		if err != nil {
			sp.Logger.Error(errMsgBroadcasting, "validatorId", event.ValidatorId.Uint64(), "error", err)
			return err
		}

		if txRes.Code != abci.CodeTypeOK {
			sp.Logger.Error(errMsgTxFailed, "txHash", txRes.TxHash, "code", txRes.Code)
			return fmt.Errorf("unStakeInit %s, tx response code: %v", errMsgTxFailed, txRes.Code)
		}

	}

	return nil
}

func (sp *StakingProcessor) sendStakeUpdateToHeimdall(eventName string, logBytes string) error {
	vLog := types.Log{}
	if err := json.Unmarshal([]byte(logBytes), &vLog); err != nil {
		sp.Logger.Error(errMsgUnmarshallingEvent, "error", err)
		return err
	}

	event := new(stakinginfo.StakinginfoStakeUpdate)
	if err := helper.UnpackLog(sp.stakingInfoAbi, event, eventName, &vLog); err != nil {
		sp.Logger.Error(errMsgParsingEvent, "name", eventName, "error", err)
	} else {
		if isOld, _ := sp.isOldTx(sp.cliCtx, vLog.TxHash.String(), uint64(vLog.Index), util.StakingEvent, event); isOld {
			sp.Logger.Info(fmt.Sprintf(infoMsgIgnoringAlreadyProcessed, "stakeUpdate"),
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
			sp.Logger.Error(errMsgValidatingNonce, "error", err)
			return err
		}

		if !validNonce {
			sp.Logger.Info(fmt.Sprintf(infoMsgIgnoringNonceOutOfOrder, "stakeUpdate"))
			return tasks.NewErrRetryTaskLater(msgNonceOutOfOrder, defaultDelayDuration*time.Duration(nonceDelay))
		}

		sp.Logger.Info(
			helper.LogReceivedTaskToSend("stake-update"),
			"event", eventName,
			"validatorID", event.ValidatorId,
			"nonce", event.Nonce,
			"newAmount", event.NewAmount,
			"txHash", vLog.TxHash.String(),
			"logIndex", uint64(vLog.Index),
			"blockNumber", vLog.BlockNumber,
		)

		address, err := helper.GetAddressString()
		if err != nil {
			return fmt.Errorf("%s: %w", errMsgConvertingAddress, err)
		}

		// msg validator update
		msg, err := stakingTypes.NewMsgStakeUpdate(
			address,
			event.ValidatorId.Uint64(),
			math.NewIntFromBigInt(event.NewAmount),
			vLog.TxHash.Bytes(),
			uint64(vLog.Index),
			vLog.BlockNumber,
			event.Nonce.Uint64(),
		)
		if err != nil {
			sp.Logger.Error(errMsgCreatingMsg, "error", err)
			return err
		}

		// return broadcast to heimdall
		txRes, err := sp.txBroadcaster.BroadcastToHeimdall(context.Background(), msg, event)
		if err != nil {
			sp.Logger.Error(errMsgBroadcasting, "validatorId", event.ValidatorId.Uint64(), "error", err)
			return err
		}

		if txRes.Code != abci.CodeTypeOK {
			sp.Logger.Error(errMsgTxFailed, "txHash", txRes.TxHash, "code", txRes.Code)
			return fmt.Errorf("stakeupdate %s, tx response code: %d", errMsgTxFailed, txRes.Code)
		}

	}

	return nil
}

func (sp *StakingProcessor) sendSignerChangeToHeimdall(eventName string, logBytes string) error {
	vLog := types.Log{}
	if err := json.Unmarshal([]byte(logBytes), &vLog); err != nil {
		sp.Logger.Error(errMsgUnmarshallingEvent, "error", err)
		return err
	}

	event := new(stakinginfo.StakinginfoSignerChange)
	if err := helper.UnpackLog(sp.stakingInfoAbi, event, eventName, &vLog); err != nil {
		sp.Logger.Error(errMsgParsingEvent, "name", eventName, "error", err)
	} else {
		newSignerPubKey := event.SignerPubkey
		if len(newSignerPubKey) == 64 {
			newSignerPubKey = util.AppendPrefix(newSignerPubKey)
		}

		if !helper.IsPubKeyFirstByteValid(newSignerPubKey) {
			sp.Logger.Error(errMsgInvalidSignerPubkey, "event", eventName, "newSignerPubKey", newSignerPubKey)
			return errors.New(errMsgInvalidSignerPubkey)
		}

		if isOld, err := sp.isOldTx(sp.cliCtx, vLog.TxHash.String(), uint64(vLog.Index), util.StakingEvent, event); isOld {
			sp.Logger.Info(fmt.Sprintf(infoMsgIgnoringAlreadyProcessed, "signerChange"),
				"event", eventName,
				"validatorID", event.ValidatorId,
				"nonce", event.Nonce,
				"NewSignerPubkey", common.Bytes2Hex(newSignerPubKey),
				"oldSigner", event.OldSigner.Hex(),
				"newSigner", event.NewSigner.Hex(),
				"txHash", vLog.TxHash.String(),
				"logIndex", uint64(vLog.Index),
				"blockNumber", vLog.BlockNumber,
			)
			return nil
		} else if err != nil {
			sp.Logger.Error(errMsgCheckingOldTx, "error", err)
			return err
		}

		validNonce, nonceDelay, err := sp.checkValidNonce(event.ValidatorId.Uint64(), event.Nonce.Uint64())
		if err != nil {
			sp.Logger.Error(errMsgValidatingNonce, "error", err)
			return err
		}

		if !validNonce {
			sp.Logger.Info(fmt.Sprintf(infoMsgIgnoringNonceOutOfOrder, "signerChange"))
			return tasks.NewErrRetryTaskLater(msgNonceOutOfOrder, defaultDelayDuration*time.Duration(nonceDelay))
		}

		sp.Logger.Info(
			helper.LogReceivedTaskToSend("signer-change"),
			"event", eventName,
			"validatorID", event.ValidatorId,
			"nonce", event.Nonce,
			"NewSignerPubkey", common.Bytes2Hex(newSignerPubKey),
			"oldSigner", event.OldSigner.Hex(),
			"newSigner", event.NewSigner.Hex(),
			"txHash", vLog.TxHash.String(),
			"logIndex", uint64(vLog.Index),
			"blockNumber", vLog.BlockNumber,
		)

		address, err := helper.GetAddressString()
		if err != nil {
			return fmt.Errorf("%s: %w", errMsgConvertingAddress, err)
		}

		// signer change
		msg, err := stakingTypes.NewMsgSignerUpdate(
			address,
			event.ValidatorId.Uint64(),
			newSignerPubKey,
			vLog.TxHash.Bytes(),
			uint64(vLog.Index),
			vLog.BlockNumber,
			event.Nonce.Uint64(),
		)
		if err != nil {
			sp.Logger.Error(errMsgCreatingMsg, "error", err)
			return err
		}

		// return broadcast to heimdall
		txRes, err := sp.txBroadcaster.BroadcastToHeimdall(context.Background(), msg, event)
		if err != nil {
			sp.Logger.Error(errMsgBroadcasting, "msg", msg, "validatorId", event.ValidatorId.Uint64(), "error", err)
			return err
		}

		if txRes.Code != abci.CodeTypeOK {
			sp.Logger.Error(errMsgTxFailed, "txHash", txRes.TxHash, "code", txRes.Code)
			return fmt.Errorf("signerChange %s, tx response code: %v", errMsgTxFailed, txRes.Code)
		}

	}

	return nil
}

func (sp *StakingProcessor) checkValidNonce(validatorId uint64, txNonce uint64) (bool, uint64, error) {
	currentNonce, err := util.GetValidatorNonce(validatorId, sp.cliCtx.Codec)
	if err != nil {
		sp.Logger.Error(errMsgFetchValidatorNonce, "validatorId", validatorId)
		return false, 0, err
	}

	if currentNonce+1 != txNonce {
		diff := txNonce - currentNonce
		if diff > 10 {
			diff = 10
		}

		sp.Logger.Error(errMsgNonceNotInOrder, "validatorId", validatorId, "currentNonce", currentNonce, "txNonce", txNonce, "delay", diff*uint64(defaultDelayDuration))

		return false, diff, nil
	}

	stakingTxnCount, err := queryTxCount(sp.cliCtx, validatorId)
	if err != nil {
		sp.Logger.Error(errMsgQueryStakeTxs, "validatorId", validatorId)
		return false, 0, err
	}

	if stakingTxnCount != 0 {
		sp.Logger.Info(infoMsgRecentStakingTxnNotZero, "validatorId", validatorId, "currentNonce", currentNonce, "txNonce", txNonce)
		return false, 1, nil
	}

	return true, 0, nil
}

func queryTxCount(cliCtx client.Context, validatorId uint64) (int, error) {
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
		events := []string{
			fmt.Sprintf("%s.%s='%s'", sdk.EventTypeMessage, sdk.AttributeKeyAction, msg),
			fmt.Sprintf("%s.%s=%d", action, "validator-id", validatorId),
		}

		// XXX: implement ANY
		query := strings.Join(events, " AND ")

		searchTxResult, err := authTx.QueryTxsByEvents(cliCtx, defaultPage, defaultLimit, query, "")
		if err != nil {
			return 0, fmt.Errorf("%s: %w", errMsgSearchTxs, err)
		}

		if searchTxResult.TotalCount != 0 {
			return int(searchTxResult.TotalCount), nil
		}
	}

	return 0, nil
}
