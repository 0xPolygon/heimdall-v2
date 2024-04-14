package keeper

import (
	"bytes"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/0xPolygon/heimdall-v2/x/stake/types"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/0xPolygon/heimdall-v2/helper"
	hmModule "github.com/0xPolygon/heimdall-v2/module"
	hmTypes "github.com/0xPolygon/heimdall-v2/types"
)

var (
	joinValidatorMethod = sdk.MsgTypeURL(&types.MsgValidatorJoin{})
	stakeUpdateMethod   = sdk.MsgTypeURL(&types.MsgStakeUpdate{})
	signerUpdateMethod  = sdk.MsgTypeURL(&types.MsgSignerUpdate{})
	validatorExitMethod = sdk.MsgTypeURL(&types.MsgValidatorExit{})
)

type sideMsgServer struct {
	*Keeper
}

// NewSideMsgServerImpl returns an implementation of the staking MsgServer interface
// for the provided Keeper.
func NewSideMsgServerImpl(keeper *Keeper) types.SideMsgServer {
	return &sideMsgServer{Keeper: keeper}
}

// SideTxHandler returns a side handler for "staking" type messages.
func (srv *sideMsgServer) SideTxHandler(methodName string) hmModule.SideTxHandler {

	switch methodName {
	case joinValidatorMethod:
		return srv.SideHandleMsgValidatorJoin
	case stakeUpdateMethod:
		return srv.SideHandleMsgStakeUpdate
	case signerUpdateMethod:
		return srv.SideHandleMsgSignerUpdate
	case validatorExitMethod:
		return srv.SideHandleMsgValidatorExit
	default:
		return nil
	}
}

// PostTxHandler returns a post handler for "staking" type messages.
func (srv *sideMsgServer) PostTxHandler(methodName string) hmModule.PostTxHandler {

	switch methodName {
	case joinValidatorMethod:
		return srv.PostHandleMsgValidatorJoin
	case stakeUpdateMethod:
		return srv.PostHandleMsgStakeUpdate
	case signerUpdateMethod:
		return srv.PostHandleMsgSignerUpdate
	case validatorExitMethod:
		return srv.PostHandleMsgValidatorExit
	default:
		return nil
	}
}

// SideHandleMsgValidatorJoin side msg validator join
func (k *sideMsgServer) SideHandleMsgValidatorJoin(ctx sdk.Context, _msg sdk.Msg) (result hmModule.Vote) {
	msg, ok := _msg.(*types.MsgValidatorJoin)
	if !ok {
		k.Logger(ctx).Error("msg type mismatched")
		return hmModule.Vote_VOTE_NO
	}

	k.Logger(ctx).Debug("✅ validating external call for validator join msg",
		"txHash", hmTypes.BytesToHeimdallHash(msg.TxHash.Bytes()),
		"logIndex", msg.LogIndex,
		"blockNumber", msg.BlockNumber,
	)

	contractCaller := k.IContractCaller

	// chainManager params
	params, err := k.cmKeeper.GetParams(ctx)
	if err != nil {
		k.Logger(ctx).Error("error in fetching chain manager params")
		return hmModule.Vote_VOTE_NO
	}

	chainParams := params.ChainParams

	// get main tx receipt
	receipt, err := contractCaller.GetConfirmedTxReceipt(msg.TxHash.EthHash(), params.MainChainTxConfirmations)
	if err != nil || receipt == nil {
		k.Logger(ctx).Error("need for more ethereum blocks to fetch the confirmed tx receipt")
		return hmModule.Vote_VOTE_NO
	}

	// decode validator join event
	eventLog, err := contractCaller.DecodeValidatorJoinEvent(chainParams.StakingInfoAddress, receipt, msg.LogIndex)
	if err != nil || eventLog == nil {
		k.Logger(ctx).Error("error while decoding the validator join event receipt receipt")
		return hmModule.Vote_VOTE_NO
	}

	// Generate PubKey from Pubkey in message and signer
	anyPk := msg.SignerPubKey
	pubKey, ok := anyPk.GetCachedValue().(cryptotypes.PubKey)
	if !ok {
		k.Logger(ctx).Error("error in interfacing out pub key")
		return hmModule.Vote_VOTE_NO
	}

	signer := pubKey.Address()

	// check signer pubkey in message corresponds
	if !bytes.Equal(pubKey.Bytes()[1:], eventLog.SignerPubkey) {
		k.Logger(ctx).Error(
			"Signer Pubkey does not match",
			"msgValidator", pubKey.String(),
			"mainchainValidator", hmTypes.BytesToHexBytes(eventLog.SignerPubkey),
		)

		return hmModule.Vote_VOTE_NO
	}

	// check signer corresponding to pubkey matches signer from event
	if !bytes.Equal(signer.Bytes(), eventLog.Signer.Bytes()) {
		k.Logger(ctx).Error(
			"Signer Address from Pubkey does not match",
			"Validator", signer.String(),
			"mainchainValidator", eventLog.Signer.Hex(),
		)

		return hmModule.Vote_VOTE_NO
	}

	// check msg id
	if eventLog.ValidatorId.Uint64() != msg.ValId {
		k.Logger(ctx).Error("id in message doesn't match with id in log", "msgId", msg.ValId, "validatorIdFromTx", eventLog.ValidatorId)
		return hmModule.Vote_VOTE_NO
	}

	// check ActivationEpoch
	if eventLog.ActivationEpoch.Uint64() != msg.ActivationEpoch {
		k.Logger(ctx).Error("activationEpoch in message doesn't match with activationEpoch in log", "msgActivationEpoch", msg.ActivationEpoch, "activationEpochFromTx", eventLog.ActivationEpoch.Uint64)
		return hmModule.Vote_VOTE_NO
	}

	// check Amount
	if eventLog.Amount.Cmp(msg.Amount.BigInt()) != 0 {
		k.Logger(ctx).Error("amount in message doesn't match Amount in event logs", "msgAmount", msg.Amount, "amountFromEvent", eventLog.Amount)
		return hmModule.Vote_VOTE_NO
	}

	// check Blocknumber
	if receipt.BlockNumber.Uint64() != msg.BlockNumber {
		k.Logger(ctx).Error("blockNumber in message doesn't match blocknumber in receipt", "msgBlockNumber", msg.BlockNumber, "receiptBlockNumber", receipt.BlockNumber.Uint64)
		return hmModule.Vote_VOTE_NO
	}

	// check nonce
	if eventLog.Nonce.Uint64() != msg.Nonce {
		k.Logger(ctx).Error("nonce in message doesn't match with nonce in log", "msgNonce", msg.Nonce, "nonceFromTx", eventLog.Nonce)
		return hmModule.Vote_VOTE_NO
	}

	k.Logger(ctx).Debug("✅ successfully validated external call for validator join msg")

	return hmModule.Vote_VOTE_YES
}

// SideHandleMsgStakeUpdate handles stake update message
func (k *sideMsgServer) SideHandleMsgStakeUpdate(ctx sdk.Context, _msg sdk.Msg) (result hmModule.Vote) {
	msg, ok := _msg.(*types.MsgStakeUpdate)
	if !ok {
		k.Logger(ctx).Error("msg type mismatched")
		return hmModule.Vote_VOTE_NO
	}

	k.Logger(ctx).Debug("✅ validating external call for stake update msg",
		"txHash", hmTypes.BytesToHeimdallHash(msg.TxHash.Bytes()),
		"logIndex", msg.LogIndex,
		"blockNumber", msg.BlockNumber,
	)

	contractCaller := k.IContractCaller

	// chainManager params
	params, err := k.cmKeeper.GetParams(ctx)
	if err != nil {
		return hmModule.Vote_VOTE_NO
	}

	chainParams := params.ChainParams

	// get main tx receipt
	receipt, err := contractCaller.GetConfirmedTxReceipt(msg.TxHash.EthHash(), params.MainChainTxConfirmations)
	if err != nil || receipt == nil {
		return hmModule.Vote_VOTE_NO
	}

	eventLog, err := contractCaller.DecodeValidatorStakeUpdateEvent(chainParams.StakingInfoAddress, receipt, msg.LogIndex)
	if err != nil || eventLog == nil {
		k.Logger(ctx).Error("error fetching log from txhash")
		return hmModule.Vote_VOTE_NO
	}

	if receipt.BlockNumber.Uint64() != msg.BlockNumber {
		k.Logger(ctx).Error("blockNumber in message doesn't match blocknumber in receipt", "msgBlockNumber", msg.BlockNumber, "receiptBlockNumber", receipt.BlockNumber.Uint64)
		return hmModule.Vote_VOTE_NO
	}

	if eventLog.ValidatorId.Uint64() != msg.ValId {
		k.Logger(ctx).Error("id in message doesn't match with id in log", "msgId", msg.ValId, "validatorIdFromTx", eventLog.ValidatorId)
		return hmModule.Vote_VOTE_NO
	}

	// check Amount
	if eventLog.NewAmount.Cmp(msg.NewAmount.BigInt()) != 0 {
		k.Logger(ctx).Error("newAmount in message doesn't match newAmount in event logs", "msgNewAmount", msg.NewAmount, "newAmountFromEvent", eventLog.NewAmount)
		return hmModule.Vote_VOTE_NO
	}

	// check nonce
	if eventLog.Nonce.Uint64() != msg.Nonce {
		k.Logger(ctx).Error("nonce in message doesn't match with nonce in log", "msgNonce", msg.Nonce, "nonceFromTx", eventLog.Nonce)
		return hmModule.Vote_VOTE_NO
	}

	k.Logger(ctx).Debug("✅ successfully validated external call for stake update msg")

	return hmModule.Vote_VOTE_YES
}

// SideHandleMsgSignerUpdate handles signer update message
func (k *sideMsgServer) SideHandleMsgSignerUpdate(ctx sdk.Context, _msg sdk.Msg) (result hmModule.Vote) {
	msg, ok := _msg.(*types.MsgSignerUpdate)
	if !ok {
		k.Logger(ctx).Error("msg type mismatched")
		return hmModule.Vote_VOTE_NO
	}

	k.Logger(ctx).Debug("✅ Validating External call for signer update msg",
		"txHash", hmTypes.BytesToHeimdallHash(msg.TxHash.Bytes()),
		"logIndex", msg.LogIndex,
		"blockNumber", msg.BlockNumber,
	)

	contractCaller := k.IContractCaller

	// chainManager params
	params, err := k.cmKeeper.GetParams(ctx)
	if err != nil {
		return hmModule.Vote_VOTE_NO
	}
	chainParams := params.ChainParams

	// get main tx receipt
	receipt, err := contractCaller.GetConfirmedTxReceipt(msg.TxHash.EthHash(), params.MainChainTxConfirmations)
	if err != nil || receipt == nil {
		return hmModule.Vote_VOTE_NO
	}

	// Generate PubKey from Pubkey in message and signer
	anyPk := msg.NewSignerPubKey
	newPubKey, ok := anyPk.GetCachedValue().(cryptotypes.PubKey)
	if !ok {
		k.Logger(ctx).Error("error in interfacing out pub key")
		return hmModule.Vote_VOTE_NO
	}

	newSigner := newPubKey.Address()

	eventLog, err := contractCaller.DecodeSignerUpdateEvent(chainParams.StakingInfoAddress, receipt, msg.LogIndex)
	if err != nil || eventLog == nil {
		k.Logger(ctx).Error("error fetching log from txhash")
		return hmModule.Vote_VOTE_NO
	}

	if receipt.BlockNumber.Uint64() != msg.BlockNumber {
		k.Logger(ctx).Error("blockNumber in message doesn't match blocknumber in receipt", "msgBlockNumber", msg.BlockNumber, "receiptBlockNumber", receipt.BlockNumber.Uint64)
		return hmModule.Vote_VOTE_NO
	}

	if eventLog.ValidatorId.Uint64() != msg.ValId {
		k.Logger(ctx).Error("id in message doesn't match with id in log", "msgId", msg.ValId, "validatorIdFromTx", eventLog.ValidatorId)
		return hmModule.Vote_VOTE_NO
	}

	if !bytes.Equal(eventLog.SignerPubkey, newPubKey.Bytes()[1:]) {
		k.Logger(ctx).Error("newsigner pubkey in txhash and msg dont match", "msgPubKey", newPubKey.String(), "pubkeyTx", hmTypes.NewPubKey(eventLog.SignerPubkey[:]).String())
		return hmModule.Vote_VOTE_NO
	}

	// check signer corresponding to pubkey matches signer from event
	if !bytes.Equal(newSigner.Bytes(), eventLog.NewSigner.Bytes()) {
		k.Logger(ctx).Error("signer address from pubkey does not match", "validator", newSigner.String(), "mainchainValidator", eventLog.NewSigner.Hex())
		return hmModule.Vote_VOTE_NO
	}

	// check nonce
	if eventLog.Nonce.Uint64() != msg.Nonce {
		k.Logger(ctx).Error("nonce in message doesn't match with nonce in log", "msgNonce", msg.Nonce, "nonceFromTx", eventLog.Nonce)
		return hmModule.Vote_VOTE_NO
	}

	k.Logger(ctx).Debug("✅ successfully validated external call for signer update msg")

	return hmModule.Vote_VOTE_YES
}

// SideHandleMsgValidatorExit  handle  side msg validator exit
func (k *sideMsgServer) SideHandleMsgValidatorExit(ctx sdk.Context, _msg sdk.Msg) (result hmModule.Vote) {
	msg, ok := _msg.(*types.MsgValidatorExit)
	if !ok {
		k.Logger(ctx).Error("msg type mismatched")
		return hmModule.Vote_VOTE_NO
	}

	k.Logger(ctx).Debug("✅ validating external call for validator exit msg",
		"txHash", hmTypes.BytesToHeimdallHash(msg.TxHash.Bytes()),
		"logIndex", msg.LogIndex,
		"blockNumber", msg.BlockNumber,
	)

	contractCaller := k.IContractCaller

	// chainManager params
	params, err := k.cmKeeper.GetParams(ctx)
	if err != nil {
		return hmModule.Vote_VOTE_NO
	}

	chainParams := params.ChainParams

	// get main tx receipt
	receipt, err := contractCaller.GetConfirmedTxReceipt(msg.TxHash.EthHash(), params.MainChainTxConfirmations)
	if err != nil || receipt == nil {
		return hmModule.Vote_VOTE_NO
	}

	// decode validator exit
	eventLog, err := contractCaller.DecodeValidatorExitEvent(chainParams.StakingInfoAddress, receipt, msg.LogIndex)
	if err != nil || eventLog == nil {
		k.Logger(ctx).Error("error fetching log from txhash")
		return hmModule.Vote_VOTE_NO
	}

	if receipt.BlockNumber.Uint64() != msg.BlockNumber {
		k.Logger(ctx).Error("blockNumber in message doesn't match blocknumber in receipt", "msgBlockNumber", msg.BlockNumber, "receiptBlockNumber", receipt.BlockNumber.Uint64)
		return hmModule.Vote_VOTE_NO
	}

	if eventLog.ValidatorId.Uint64() != msg.ValId {
		k.Logger(ctx).Error("id in message doesn't match with id in log", "msgId", msg.ValId, "validatorIdFromTx", eventLog.ValidatorId)
		return hmModule.Vote_VOTE_NO
	}

	if eventLog.DeactivationEpoch.Uint64() != msg.DeactivationEpoch {
		k.Logger(ctx).Error("deactivationEpoch in message doesn't match with deactivationEpoch in log", "msgDeactivationEpoch", msg.DeactivationEpoch, "deactivationEpochFromTx", eventLog.DeactivationEpoch.Uint64)
		return hmModule.Vote_VOTE_NO
	}

	// check nonce
	if eventLog.Nonce.Uint64() != msg.Nonce {
		k.Logger(ctx).Error("nonce in message doesn't match with nonce in log", "msgNonce", msg.Nonce, "nonceFromTx", eventLog.Nonce)
		return hmModule.Vote_VOTE_NO
	}

	k.Logger(ctx).Debug("✅ successfully validated external call for validator exit msg")

	return hmModule.Vote_VOTE_YES
}

/*
	Post Handlers - update the state of the tx
**/

// PostHandleMsgValidatorJoin msg validator join
func (k *sideMsgServer) PostHandleMsgValidatorJoin(ctx sdk.Context, _msg sdk.Msg, sideTxResult hmModule.Vote) {
	msg, ok := _msg.(*types.MsgValidatorJoin)
	if !ok {
		k.Logger(ctx).Error("msg type mismatched")
		return
	}

	// Skip handler if validator join is not approved
	if sideTxResult != hmModule.Vote_VOTE_YES {
		k.Logger(ctx).Debug("skipping new validator-join since side-tx didn't get yes votes")
		return
	}

	// Check for replay attack
	blockNumber := new(big.Int).SetUint64(msg.BlockNumber)
	sequence := new(big.Int).Mul(blockNumber, big.NewInt(hmTypes.DefaultLogIndexUnit))
	sequence.Add(sequence, new(big.Int).SetUint64(msg.LogIndex))

	// check if incoming tx is older
	if k.HasStakingSequence(ctx, sequence.String()) {
		k.Logger(ctx).Error("older invalid tx found")
		return
	}

	k.Logger(ctx).Debug("adding validator to state", "sideTxResult", sideTxResult)

	// Generate PubKey from Pubkey in message and signer
	anyPk := msg.SignerPubKey
	pubKey, ok := anyPk.GetCachedValue().(cryptotypes.PubKey)
	if !ok {
		k.Logger(ctx).Error("error in interfacing out pub key")
		return
	}

	signer := pubKey.Address().String()

	// get voting power from amount
	votingPower, err := helper.GetPowerFromAmount(msg.Amount.BigInt())
	if err != nil {
		k.Logger(ctx).Error(fmt.Sprintf("invalid amount %v for validator %v", msg.Amount, msg.ValId))
		return
	}

	// create new validator
	newValidator := types.Validator{
		ValId:       msg.ValId,
		StartEpoch:  msg.ActivationEpoch,
		EndEpoch:    0,
		Nonce:       msg.Nonce,
		VotingPower: votingPower.Int64(),
		PubKey:      anyPk,
		Signer:      strings.ToLower(signer),
		LastUpdated: "",
	}

	// update last updated
	newValidator.LastUpdated = sequence.String()

	// add validator to store
	k.Logger(ctx).Debug("adding new validator to state", "validator", newValidator.String())

	if err = k.AddValidator(ctx, newValidator); err != nil {
		k.Logger(ctx).Error("unable to add validator to state", "validator", newValidator.String(), "error", err)
		return
	}

	// Add Validator signing info. It is required for slashing module
	k.Logger(ctx).Debug("adding signing info for new validator")

	//TODO HV2 PLease check whether we need the following code or not
	//as this code belongs to slashing
	// valSigningInfo := hmTypes.NewValidatorSigningInfo(newValidator.ID, ctx.BlockHeight(), int64(0), int64(0))
	// if err = k.AddValidatorSigningInfo(ctx, newValidator.ID, valSigningInfo); err != nil {
	// 	k.Logger(ctx).Error("Unable to add validator signing info to state", "valSigningInfo", valSigningInfo.String(), "error", err)
	// 	return hmCommon.ErrValidatorSigningInfoSave(k.Codespace()).Result()
	// }

	// save staking sequence
	err = k.SetStakingSequence(ctx, sequence.String())
	if err != nil {
		k.Logger(ctx).Error("unable to set the sequence", "error", err)
		return
	}

	k.Logger(ctx).Debug("✅ new validator successfully joined", "validator", strconv.FormatUint(newValidator.ValId, 10))

	// TX bytes
	txBytes := ctx.TxBytes()
	hash := hmTypes.TxHash{Hash: txBytes}.Bytes()

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeValidatorJoin,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(hmTypes.AttributeKeyTxHash, hmTypes.BytesToHeimdallHash(hash).Hex()),
			sdk.NewAttribute(hmTypes.AttributeKeyTxLogIndex, strconv.FormatUint(msg.LogIndex, 10)),
			sdk.NewAttribute(hmTypes.AttributeKeySideTxResult, sideTxResult.String()),
			sdk.NewAttribute(types.AttributeKeyValidatorID, strconv.FormatUint(newValidator.ValId, 10)),
			sdk.NewAttribute(types.AttributeKeySigner, newValidator.Signer),
			sdk.NewAttribute(types.AttributeKeyValidatorNonce, strconv.FormatUint(msg.Nonce, 10)),
		),
	})

	return
}

// PostHandleMsgStakeUpdate handles stake update message
func (k *sideMsgServer) PostHandleMsgStakeUpdate(ctx sdk.Context, _msg sdk.Msg, sideTxResult hmModule.Vote) {
	msg, ok := _msg.(*types.MsgStakeUpdate)
	if !ok {
		k.Logger(ctx).Error("msg type mismatched")
		return
	}

	// Skip handler if stakeUpdate is not approved
	if sideTxResult != hmModule.Vote_VOTE_YES {
		k.Logger(ctx).Debug("skipping stake update since side-tx didn't get yes votes")
		return
	}

	// Check for replay attack
	blockNumber := new(big.Int).SetUint64(msg.BlockNumber)
	sequence := new(big.Int).Mul(blockNumber, big.NewInt(hmTypes.DefaultLogIndexUnit))
	sequence.Add(sequence, new(big.Int).SetUint64(msg.LogIndex))

	// check if incoming tx is older
	if k.HasStakingSequence(ctx, sequence.String()) {
		k.Logger(ctx).Error("Older invalid tx found")
		return
	}

	k.Logger(ctx).Debug("updating validator stake", "sideTxResult", sideTxResult)

	// pull validator from store
	validator, ok := k.GetValidatorFromValID(ctx, msg.ValId)
	if !ok {
		k.Logger(ctx).Error("fetching of validator from store failed", "validatorId", msg.ValId)
		return
	}

	// update last updated
	validator.LastUpdated = sequence.String()

	// update nonce
	validator.Nonce = msg.Nonce

	// set validator amount
	p, err := helper.GetPowerFromAmount(msg.NewAmount.BigInt())
	if err != nil {
		return
	}

	validator.VotingPower = p.Int64()

	// save validator
	err = k.AddValidator(ctx, validator)
	if err != nil {
		k.Logger(ctx).Error("unable to update signer", "validatorID", validator.ValId, "error", err)
		return
	}

	// save staking sequence
	err = k.SetStakingSequence(ctx, sequence.String())
	if err != nil {
		k.Logger(ctx).Error("unable to set the sequence", "error", err)
		return
	}

	// TX bytes
	txBytes := ctx.TxBytes()
	hash := hmTypes.TxHash{Hash: txBytes}.Bytes()

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeStakeUpdate,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(hmTypes.AttributeKeyTxHash, hmTypes.BytesToHeimdallHash(hash).Hex()), // tx hash
			sdk.NewAttribute(hmTypes.AttributeKeySideTxResult, sideTxResult.String()),             // result
			sdk.NewAttribute(types.AttributeKeyValidatorID, strconv.FormatUint(validator.ValId, 10)),
			sdk.NewAttribute(types.AttributeKeyValidatorNonce, strconv.FormatUint(msg.Nonce, 10)),
		),
	})

	return
}

// PostHandleMsgSignerUpdate handles signer update message
func (k *sideMsgServer) PostHandleMsgSignerUpdate(ctx sdk.Context, _msg sdk.Msg, sideTxResult hmModule.Vote) {
	msg, ok := _msg.(*types.MsgSignerUpdate)
	if !ok {
		k.Logger(ctx).Error("msg type mismatched")
		return
	}

	// Skip handler if signer update is not approved
	if sideTxResult != hmModule.Vote_VOTE_YES {
		k.Logger(ctx).Debug("skipping signer update since side-tx didn't get yes votes")
		return
	}

	// Check for replay attack
	blockNumber := new(big.Int).SetUint64(msg.BlockNumber)
	sequence := new(big.Int).Mul(blockNumber, big.NewInt(hmTypes.DefaultLogIndexUnit))
	sequence.Add(sequence, new(big.Int).SetUint64(msg.LogIndex))
	// check if incoming tx is older
	if k.HasStakingSequence(ctx, sequence.String()) {
		k.Logger(ctx).Error("older invalid tx found")
		return
	}

	k.Logger(ctx).Debug("persisting signer update", "sideTxResult", sideTxResult)

	// Generate PubKey from Pubkey in message and signer
	anyPk := msg.NewSignerPubKey
	newPubKey, ok := anyPk.GetCachedValue().(cryptotypes.PubKey)
	if !ok {
		k.Logger(ctx).Error("error in interfacing out pub key")
		return
	}

	newSigner := strings.ToLower(newPubKey.Address().String())

	// pull validator from store
	validator, ok := k.GetValidatorFromValID(ctx, msg.ValId)
	if !ok {
		k.Logger(ctx).Error("fetching of validator from store failed", "validatorId", msg.ValId)
		return
	}

	oldValidator := validator.Copy()

	// update last updated
	validator.LastUpdated = sequence.String()

	// update nonce
	validator.Nonce = msg.Nonce

	// check if we are actually updating signer
	if !(newSigner == validator.Signer) {
		// Update signer in prev Validator
		validator.Signer = newSigner
		validator.PubKey = anyPk

		k.Logger(ctx).Debug("updating new signer", "newSigner", newSigner, "oldSigner", oldValidator.Signer, "validatorID", msg.ValId)
	} else {
		k.Logger(ctx).Error("no signer change", "newSigner", newSigner, "oldSigner", oldValidator.Signer, "validatorID", msg.ValId)
		return
	}

	k.Logger(ctx).Debug("removing old validator", "validator", oldValidator.String())

	// remove old validator from HM
	oldValidator.EndEpoch = k.moduleCommunicator.GetACKCount(ctx)

	// remove old validator from TM
	oldValidator.VotingPower = 0
	// updated last
	oldValidator.LastUpdated = sequence.String()

	// updated nonce
	oldValidator.Nonce = msg.Nonce

	// save old validator
	if err := k.AddValidator(ctx, *oldValidator); err != nil {
		k.Logger(ctx).Error("unable to update signer", "validatorId", validator.ValId, "error", err)
		return
	}

	// adding new validator
	k.Logger(ctx).Debug("adding new validator", "validator", validator.String())

	// save validator
	err := k.AddValidator(ctx, validator)
	if err != nil {
		k.Logger(ctx).Error("unable to update signer", "validatorID", validator.ValId, "error", err)
		return
	}

	// save staking sequence
	err = k.SetStakingSequence(ctx, sequence.String())
	if err != nil {
		k.Logger(ctx).Error("unable to set the sequence", "error", err)
		return
	}

	// TX bytes
	txBytes := ctx.TxBytes()
	hash := hmTypes.TxHash{Hash: txBytes}.Bytes()

	//
	// Move heimdall fee to new signer
	//

	//TODO HV2 Please check this code once module communicatator is defined properlu
	// // check if fee is already withdrawn
	// coins := k.moduleCommunicator.GetCoins(ctx, oldValidator.Signer)

	// maticBalance := coins.AmountOf(authTypes.FeeToken)
	// if !maticBalance.IsZero() {
	// 	k.Logger(ctx).Info("Transferring fee", "from", oldValidator.Signer.String(), "to", validator.Signer.String(), "balance", maticBalance.String())

	// 	maticCoins := sdk.Coins{sdk.Coin{Denom: authTypes.FeeToken, Amount: maticBalance}}
	// 	if err := k.moduleCommunicator.SendCoins(ctx, oldValidator.Signer, validator.Signer, maticCoins); err != nil {
	// 		k.Logger(ctx).Info("Error while transferring fee", "from", oldValidator.Signer.String(), "to", validator.Signer.String(), "balance", maticBalance.String())
	// 		return err.Result()
	// 	}
	// }

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeSignerUpdate,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(hmTypes.AttributeKeyTxHash, hmTypes.BytesToHeimdallHash(hash).Hex()),
			sdk.NewAttribute(hmTypes.AttributeKeySideTxResult, sideTxResult.String()),
			sdk.NewAttribute(types.AttributeKeyValidatorID, strconv.FormatUint(validator.ValId, 10)),
			sdk.NewAttribute(types.AttributeKeyValidatorNonce, strconv.FormatUint(msg.Nonce, 10)),
		),
	})

	return
}

// PostHandleMsgValidatorExit handle msg validator exit
func (k *sideMsgServer) PostHandleMsgValidatorExit(ctx sdk.Context, _msg sdk.Msg, sideTxResult hmModule.Vote) {
	msg, ok := _msg.(*types.MsgValidatorExit)
	if !ok {
		k.Logger(ctx).Error("msg type mismatched")
		return
	}

	// Skip handler if validator exit is not approved
	if sideTxResult != hmModule.Vote_VOTE_YES {
		k.Logger(ctx).Debug("skipping validator exit since side-tx didn't get yes votes")
		return
	}

	// Check for replay attack
	blockNumber := new(big.Int).SetUint64(msg.BlockNumber)
	sequence := new(big.Int).Mul(blockNumber, big.NewInt(hmTypes.DefaultLogIndexUnit))
	sequence.Add(sequence, new(big.Int).SetUint64(msg.LogIndex))

	// check if incoming tx is older
	if k.HasStakingSequence(ctx, sequence.String()) {
		k.Logger(ctx).Error("older invalid tx found")
		return
	}

	k.Logger(ctx).Debug("persisting validator exit", "sideTxResult", sideTxResult)

	validator, ok := k.GetValidatorFromValID(ctx, msg.ValId)
	if !ok {
		k.Logger(ctx).Error("fetching of validator from store failed", "validatorID", msg.ValId)
		return
	}

	// set end epoch
	validator.EndEpoch = msg.DeactivationEpoch

	// update last updated
	validator.LastUpdated = sequence.String()

	// update nonce
	validator.Nonce = msg.Nonce

	// Add deactivation time for validator
	if err := k.AddValidator(ctx, validator); err != nil {
		k.Logger(ctx).Error("error while setting deactivation epoch to validator", "error", err, "validatorID", validator.ValId)
		return
	}

	// save staking sequence
	err := k.SetStakingSequence(ctx, sequence.String())
	if err != nil {
		k.Logger(ctx).Error("unable to set the sequence", "error", err)
		return
	}

	// TX bytes
	txBytes := ctx.TxBytes()
	hash := hmTypes.TxHash{Hash: txBytes}.Bytes()

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeValidatorExit,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(hmTypes.AttributeKeyTxHash, hmTypes.BytesToHeimdallHash(hash).Hex()),
			sdk.NewAttribute(hmTypes.AttributeKeySideTxResult, sideTxResult.String()),
			sdk.NewAttribute(types.AttributeKeyValidatorID, strconv.FormatUint(validator.ValId, 10)),
			sdk.NewAttribute(types.AttributeKeyValidatorNonce, strconv.FormatUint(msg.Nonce, 10)),
		),
	})

	return
}
