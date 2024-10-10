package keeper

import (
	"bytes"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	addrCodec "github.com/cosmos/cosmos-sdk/codec/address"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authTypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/sidetxs"
	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/stake/types"
)

var (
	joinValidatorMethod = sdk.MsgTypeURL(&types.MsgValidatorJoin{})
	stakeUpdateMethod   = sdk.MsgTypeURL(&types.MsgStakeUpdate{})
	signerUpdateMethod  = sdk.MsgTypeURL(&types.MsgSignerUpdate{})
	validatorExitMethod = sdk.MsgTypeURL(&types.MsgValidatorExit{})
)

type sideMsgServer struct {
	k *Keeper
}

// NewSideMsgServerImpl returns an implementation of the staking MsgServer interface
// for the provided Keeper.
func NewSideMsgServerImpl(keeper *Keeper) sidetxs.SideMsgServer {
	return &sideMsgServer{k: keeper}
}

// SideTxHandler returns a side handler for "staking" type messages.
func (s *sideMsgServer) SideTxHandler(methodName string) sidetxs.SideTxHandler {

	switch methodName {
	case joinValidatorMethod:
		return s.SideHandleMsgValidatorJoin
	case stakeUpdateMethod:
		return s.SideHandleMsgStakeUpdate
	case signerUpdateMethod:
		return s.SideHandleMsgSignerUpdate
	case validatorExitMethod:
		return s.SideHandleMsgValidatorExit
	default:
		return nil
	}
}

// PostTxHandler redirects to the right sideMsgServer post_handler based on methodName
func (s *sideMsgServer) PostTxHandler(methodName string) sidetxs.PostTxHandler {

	switch methodName {
	case joinValidatorMethod:
		return s.PostHandleMsgValidatorJoin
	case stakeUpdateMethod:
		return s.PostHandleMsgStakeUpdate
	case signerUpdateMethod:
		return s.PostHandleMsgSignerUpdate
	case validatorExitMethod:
		return s.PostHandleMsgValidatorExit
	default:
		return nil
	}
}

// SideHandleMsgValidatorJoin is a side handler for validator join msg
func (s *sideMsgServer) SideHandleMsgValidatorJoin(ctx sdk.Context, msgI sdk.Msg) (result sidetxs.Vote) {
	msg, ok := msgI.(*types.MsgValidatorJoin)
	if !ok {
		s.k.Logger(ctx).Error("type mismatch for MsgValidatorJoin")
		return sidetxs.Vote_VOTE_NO
	}

	s.k.Logger(ctx).Debug("✅ validating external call for validator join msg",
		"txHash", common.Bytes2Hex(msg.TxHash),
		"logIndex", msg.LogIndex,
		"blockNumber", msg.BlockNumber,
	)

	contractCaller := s.k.contractCaller

	// chainManager params
	params, err := s.k.cmKeeper.GetParams(ctx)
	if err != nil {
		s.k.Logger(ctx).Error("error in fetching chain manager params", "err", err)
		return sidetxs.Vote_VOTE_NO
	}

	chainParams := params.ChainParams

	// get main tx receipt
	receipt, err := contractCaller.GetConfirmedTxReceipt(common.BytesToHash(msg.TxHash), params.MainChainTxConfirmations)
	if err != nil || receipt == nil {
		s.k.Logger(ctx).Error("can't get confirmed tx receipt", "err", err)
		return sidetxs.Vote_VOTE_NO
	}

	// decode validator join event
	eventLog, err := contractCaller.DecodeValidatorJoinEvent(chainParams.StakingInfoAddress, receipt, msg.LogIndex)
	if err != nil || eventLog == nil {
		s.k.Logger(ctx).Error("error while decoding the validator join event receipt receipt")
		return sidetxs.Vote_VOTE_NO
	}

	// Generate PubKey from PubKey in message and signer
	anyPk := msg.SignerPubKey
	pubKey, ok := anyPk.GetCachedValue().(cryptotypes.PubKey)
	if !ok {
		s.k.Logger(ctx).Error("error in interfacing out pub key")
		return sidetxs.Vote_VOTE_NO
	}

	signer := pubKey.Address()
	ac := addrCodec.NewHexCodec()
	signerBytes, err := ac.StringToBytes(signer.String())
	if err != nil {
		s.k.Logger(ctx).Error("error in converting signer address to bytes", "err", err)
		return sidetxs.Vote_VOTE_NO
	}
	eventLogSignerBytes, err := ac.StringToBytes(eventLog.Signer.String())
	if err != nil {
		s.k.Logger(ctx).Error("error in converting event log signer address to bytes", "err", err)
		return sidetxs.Vote_VOTE_NO
	}

	// check public key first byte
	if !helper.IsPubKeyFirstByteValid(pubKey.Bytes()[0:1]) {
		s.k.Logger(ctx).Error(
			"public key first byte mismatch",
			"expected", "0x04",
			"received", pubKey.Bytes()[0:1])
		return sidetxs.Vote_VOTE_NO
	}

	// check signer pubKey in message corresponds
	if !bytes.Equal(pubKey.Bytes()[1:], eventLog.SignerPubkey) {
		s.k.Logger(ctx).Error(
			"Signer PubKey does not match",
			"msgValidator", pubKey.String(),
			"mainChainValidator", common.Bytes2Hex(eventLog.SignerPubkey),
		)
		return sidetxs.Vote_VOTE_NO
	}

	// check signer corresponding to pubKey matches signer from event
	if !bytes.Equal(signerBytes, eventLogSignerBytes) {
		s.k.Logger(ctx).Error(
			"Signer address does not match event log signer address",
			"Validator", signer.String(),
			"mainChainValidator", eventLog.Signer.Hex(),
		)
		return sidetxs.Vote_VOTE_NO
	}

	// check msg id
	if eventLog.ValidatorId.Uint64() != msg.ValId {
		s.k.Logger(ctx).Error(
			"id in message doesn't match with id in log",
			"msgId", msg.ValId,
			"validatorIdFromTx", eventLog.ValidatorId)
		return sidetxs.Vote_VOTE_NO
	}

	// check ActivationEpoch
	if eventLog.ActivationEpoch.Uint64() != msg.ActivationEpoch {
		s.k.Logger(ctx).Error(
			"activationEpoch in message doesn't match with activationEpoch in log",
			"msgActivationEpoch", msg.ActivationEpoch,
			"activationEpochFromTx", eventLog.ActivationEpoch.Uint64)
		return sidetxs.Vote_VOTE_NO
	}

	// check Amount
	if eventLog.Amount.Cmp(msg.Amount.BigInt()) != 0 {
		s.k.Logger(ctx).Error(
			"amount in message doesn't match Amount in event logs",
			"msgAmount", msg.Amount,
			"amountFromEvent", eventLog.Amount)
		return sidetxs.Vote_VOTE_NO
	}

	// TODO HV2: these checks are repeated in all handlers, can be moved to a common function
	//  See https://polygon.atlassian.net/browse/POS-2615

	// check BlockNumber
	if receipt.BlockNumber.Uint64() != msg.BlockNumber {
		s.k.Logger(ctx).Error(
			"blockNumber in message doesn't match blockNumber in receipt",
			"msgBlockNumber", msg.BlockNumber,
			"receiptBlockNumber", receipt.BlockNumber.Uint64)
		return sidetxs.Vote_VOTE_NO
	}

	// check nonce
	if eventLog.Nonce.Uint64() != msg.Nonce {
		s.k.Logger(ctx).Error(
			"nonce in message doesn't match with nonce in log",
			"msgNonce", msg.Nonce,
			"nonceFromTx", eventLog.Nonce)
		return sidetxs.Vote_VOTE_NO
	}

	s.k.Logger(ctx).Debug("✅ successfully validated external call for validator join msg")

	return sidetxs.Vote_VOTE_YES
}

// SideHandleMsgStakeUpdate handles stake update message
func (s *sideMsgServer) SideHandleMsgStakeUpdate(ctx sdk.Context, msgI sdk.Msg) (result sidetxs.Vote) {
	msg, ok := msgI.(*types.MsgStakeUpdate)
	if !ok {
		s.k.Logger(ctx).Error("type mismatch for MsgStakeUpdate")
		return sidetxs.Vote_VOTE_NO
	}

	s.k.Logger(ctx).Debug("✅ validating external call for stake update msg",
		"txHash", common.Bytes2Hex(msg.TxHash),
		"logIndex", msg.LogIndex,
		"blockNumber", msg.BlockNumber,
	)

	params, err := s.k.cmKeeper.GetParams(ctx)
	if err != nil {
		s.k.Logger(ctx).Error("error in fetching params from store", "err", err)
		return sidetxs.Vote_VOTE_NO
	}

	// get main tx receipt
	contractCaller := s.k.contractCaller
	receipt, err := contractCaller.GetConfirmedTxReceipt(common.BytesToHash(msg.TxHash), params.MainChainTxConfirmations)
	if err != nil || receipt == nil {
		s.k.Logger(ctx).Error("error in getting event receipt from ethereum ", "err", err)
		return sidetxs.Vote_VOTE_NO
	}

	chainParams := params.ChainParams
	eventLog, err := contractCaller.DecodeValidatorStakeUpdateEvent(chainParams.StakingInfoAddress, receipt, msg.LogIndex)
	if err != nil || eventLog == nil {
		s.k.Logger(ctx).Error("error fetching log from txHash", "err", err)
		return sidetxs.Vote_VOTE_NO
	}

	if receipt.BlockNumber.Uint64() != msg.BlockNumber {
		s.k.Logger(ctx).Error(
			"blockNumber in message doesn't match blockNumber in receipt",
			"msgBlockNumber", msg.BlockNumber,
			"receiptBlockNumber", receipt.BlockNumber.Uint64)
		return sidetxs.Vote_VOTE_NO
	}

	if eventLog.ValidatorId.Uint64() != msg.ValId {
		s.k.Logger(ctx).Error(
			"id in message doesn't match with id in log",
			"msgId", msg.ValId,
			"validatorIdFromTx", eventLog.ValidatorId)
		return sidetxs.Vote_VOTE_NO
	}

	// check amount
	if eventLog.NewAmount.Cmp(msg.NewAmount.BigInt()) != 0 {
		s.k.Logger(ctx).Error("newAmount in message doesn't match newAmount in event logs",
			"msgNewAmount", msg.NewAmount,
			"newAmountFromEvent", eventLog.NewAmount)
		return sidetxs.Vote_VOTE_NO
	}

	// check nonce
	if eventLog.Nonce.Uint64() != msg.Nonce {
		s.k.Logger(ctx).Error("nonce in message doesn't match with nonce in log",
			"msgNonce", msg.Nonce,
			"nonceFromTx", eventLog.Nonce)
		return sidetxs.Vote_VOTE_NO
	}

	s.k.Logger(ctx).Debug("✅ successfully validated external call for stake update msg")

	return sidetxs.Vote_VOTE_YES
}

// SideHandleMsgSignerUpdate handles signer update message
func (s *sideMsgServer) SideHandleMsgSignerUpdate(ctx sdk.Context, msgI sdk.Msg) (result sidetxs.Vote) {
	msg, ok := msgI.(*types.MsgSignerUpdate)
	if !ok {
		s.k.Logger(ctx).Error("type mismatch for MsgSignerUpdate")
		return sidetxs.Vote_VOTE_NO
	}

	s.k.Logger(ctx).Debug("✅ Validating External call for signer update msg",
		"txHash", common.Bytes2Hex(msg.TxHash),
		"logIndex", msg.LogIndex,
		"blockNumber", msg.BlockNumber,
	)

	// chainManager params
	params, err := s.k.cmKeeper.GetParams(ctx)
	if err != nil {
		s.k.Logger(ctx).Error("error in fetching params from store", "err", err)
		return sidetxs.Vote_VOTE_NO
	}

	// get main tx receipt
	contractCaller := s.k.contractCaller
	receipt, err := contractCaller.GetConfirmedTxReceipt(common.BytesToHash(msg.TxHash), params.MainChainTxConfirmations)
	if err != nil || receipt == nil {
		s.k.Logger(ctx).Error("error in getting event receipt from ethereum ", "err", err)
		return sidetxs.Vote_VOTE_NO
	}

	// Generate PubKey from PubKey in message and signer
	anyPk := msg.NewSignerPubKey
	newPubKey, ok := anyPk.GetCachedValue().(cryptotypes.PubKey)
	if !ok {
		s.k.Logger(ctx).Error("error in interfacing out pub key")
		return sidetxs.Vote_VOTE_NO
	}

	chainParams := params.ChainParams
	eventLog, err := contractCaller.DecodeSignerUpdateEvent(chainParams.StakingInfoAddress, receipt, msg.LogIndex)
	if err != nil || eventLog == nil {
		s.k.Logger(ctx).Error("error fetching log from txHash", "err", err)
		return sidetxs.Vote_VOTE_NO
	}

	if receipt.BlockNumber.Uint64() != msg.BlockNumber {
		s.k.Logger(ctx).Error("blockNumber in message doesn't match blockNumber in receipt", "msgBlockNumber", msg.BlockNumber, "receiptBlockNumber", receipt.BlockNumber.Uint64)
		return sidetxs.Vote_VOTE_NO
	}

	if eventLog.ValidatorId.Uint64() != msg.ValId {
		s.k.Logger(ctx).Error("id in message doesn't match with id in log", "msgId", msg.ValId, "validatorIdFromTx", eventLog.ValidatorId)
		return sidetxs.Vote_VOTE_NO
	}

	if !bytes.Equal(eventLog.SignerPubkey, newPubKey.Bytes()[1:]) {
		s.k.Logger(ctx).Error("newSigner pubKey in txHash and msg dont match", "msgPubKey", newPubKey.String(), "pubKeyTx", eventLog.SignerPubkey[:])
		return sidetxs.Vote_VOTE_NO
	}

	newSigner := newPubKey.Address()

	ac := addrCodec.NewHexCodec()
	signerBytes, err := ac.StringToBytes(newSigner.String())
	if err != nil {
		s.k.Logger(ctx).Error("error in converting signer address to bytes", "err", err)
		return sidetxs.Vote_VOTE_NO
	}
	eventLogSignerBytes, err := ac.StringToBytes(eventLog.NewSigner.String())
	if err != nil {
		s.k.Logger(ctx).Error("error in converting event log signer address to bytes", "err", err)
		return sidetxs.Vote_VOTE_NO
	}

	// check signer corresponding to pubKey matches signer from event
	if !bytes.Equal(signerBytes, eventLogSignerBytes) {
		s.k.Logger(ctx).Error("signer address does not match event log signer address", "validator", newSigner.String(), "mainChainValidator", eventLog.NewSigner.Hex())
		return sidetxs.Vote_VOTE_NO
	}

	// check nonce
	if eventLog.Nonce.Uint64() != msg.Nonce {
		s.k.Logger(ctx).Error("nonce in message doesn't match with nonce in log", "msgNonce", msg.Nonce, "nonceFromTx", eventLog.Nonce)
		return sidetxs.Vote_VOTE_NO
	}

	s.k.Logger(ctx).Debug("✅ successfully validated external call for signer update msg")

	return sidetxs.Vote_VOTE_YES
}

// SideHandleMsgValidatorExit handles side msg validator exit
func (s *sideMsgServer) SideHandleMsgValidatorExit(ctx sdk.Context, msgI sdk.Msg) (result sidetxs.Vote) {
	msg, ok := msgI.(*types.MsgValidatorExit)
	if !ok {
		s.k.Logger(ctx).Error("type mismatch for MsgValidatorExit")
		return sidetxs.Vote_VOTE_NO
	}

	s.k.Logger(ctx).Debug("✅ validating external call for validator exit msg",
		"txHash", common.Bytes2Hex(msg.TxHash),
		"logIndex", msg.LogIndex,
		"blockNumber", msg.BlockNumber,
	)

	contractCaller := s.k.contractCaller

	// chainManager params
	params, err := s.k.cmKeeper.GetParams(ctx)
	if err != nil {
		s.k.Logger(ctx).Error("error in fetching params from store", "err", err)
		return sidetxs.Vote_VOTE_NO
	}

	chainParams := params.ChainParams

	// get main tx receipt
	receipt, err := contractCaller.GetConfirmedTxReceipt(common.BytesToHash(msg.TxHash), params.MainChainTxConfirmations)
	if err != nil || receipt == nil {
		s.k.Logger(ctx).Error("error in getting event receipt from ethereum ", "err", err)
		return sidetxs.Vote_VOTE_NO
	}

	// decode validator exit
	eventLog, err := contractCaller.DecodeValidatorExitEvent(chainParams.StakingInfoAddress, receipt, msg.LogIndex)
	if err != nil || eventLog == nil {
		s.k.Logger(ctx).Error("error fetching log from txHash", "err", err)
		return sidetxs.Vote_VOTE_NO
	}

	if receipt.BlockNumber.Uint64() != msg.BlockNumber {
		s.k.Logger(ctx).Error("blockNumber in message doesn't match blockNumber in receipt", "msgBlockNumber", msg.BlockNumber, "receiptBlockNumber", receipt.BlockNumber.Uint64)
		return sidetxs.Vote_VOTE_NO
	}

	if eventLog.ValidatorId.Uint64() != msg.ValId {
		s.k.Logger(ctx).Error("id in message doesn't match with id in log", "msgId", msg.ValId, "validatorIdFromTx", eventLog.ValidatorId)
		return sidetxs.Vote_VOTE_NO
	}

	if eventLog.DeactivationEpoch.Uint64() != msg.DeactivationEpoch {
		s.k.Logger(ctx).Error("deactivationEpoch in message doesn't match with deactivationEpoch in log", "msgDeactivationEpoch", msg.DeactivationEpoch, "deactivationEpochFromTx", eventLog.DeactivationEpoch.Uint64)
		return sidetxs.Vote_VOTE_NO
	}

	// check nonce
	if eventLog.Nonce.Uint64() != msg.Nonce {
		s.k.Logger(ctx).Error("nonce in message doesn't match with nonce in log", "msgNonce", msg.Nonce, "nonceFromTx", eventLog.Nonce)
		return sidetxs.Vote_VOTE_NO
	}

	s.k.Logger(ctx).Debug("✅ successfully validated external call for validator exit msg")

	return sidetxs.Vote_VOTE_YES
}

// PostHandleMsgValidatorJoin handles validator join message
func (s *sideMsgServer) PostHandleMsgValidatorJoin(ctx sdk.Context, msgI sdk.Msg, sideTxResult sidetxs.Vote) {
	msg, ok := msgI.(*types.MsgValidatorJoin)
	if !ok {
		s.k.Logger(ctx).Error("type mismatch for MsgValidatorJoin")
		return
	}

	// Skip handler if validator join is not approved
	if sideTxResult != sidetxs.Vote_VOTE_YES {
		s.k.Logger(ctx).Debug("skipping new validator-join since side-tx didn't get yes votes")
		return
	}

	// Check for replay attack
	blockNumber := new(big.Int).SetUint64(msg.BlockNumber)
	sequence := new(big.Int).Mul(blockNumber, big.NewInt(types.DefaultLogIndexUnit))
	sequence.Add(sequence, new(big.Int).SetUint64(msg.LogIndex))

	// check if incoming tx is older
	if s.k.HasStakingSequence(ctx, sequence.String()) {
		s.k.Logger(ctx).Error("older invalid tx found", "sequence", sequence.String())
		return
	}

	s.k.Logger(ctx).Debug("adding validator to state", "sideTxResult", sideTxResult)

	// Generate PubKey from PubKey in message and signer
	anyPk := msg.SignerPubKey
	pubKey, ok := anyPk.GetCachedValue().(cryptotypes.PubKey)
	if !ok {
		s.k.Logger(ctx).Error("error in interfacing out pub key")
		return
	}

	if pubKey.Type() != types.Secp256k1Type {
		s.k.Logger(ctx).Error("public key is invalid")
		return
	}

	signer := strings.ToLower(pubKey.Address().String())

	// get voting power from amount
	votingPower, err := helper.GetPowerFromAmount(msg.Amount.BigInt())
	if err != nil {
		s.k.Logger(ctx).Error(fmt.Sprintf("invalid amount %v for validator %v", msg.Amount, msg.ValId))
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
		Signer:      signer,
		LastUpdated: sequence.String(),
	}

	// add validator to store
	s.k.Logger(ctx).Debug("adding new validator to state", "validator", newValidator.String())

	if err = s.k.AddValidator(ctx, newValidator); err != nil {
		s.k.Logger(ctx).Error("unable to add validator to state", "validator", newValidator.String(), "error", err)
		return
	}

	// Add Validator signing info. It is required for slashing module
	s.k.Logger(ctx).Debug("adding signing info for new validator")

	// save staking sequence
	err = s.k.SetStakingSequence(ctx, sequence.String())
	if err != nil {
		s.k.Logger(ctx).Error("unable to set the sequence", "error", err)
		return
	}

	s.k.Logger(ctx).Debug("✅ new validator successfully joined", "validator", strconv.FormatUint(newValidator.ValId, 10))

	txBytes := ctx.TxBytes()

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeValidatorJoin,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(hmTypes.AttributeKeyTxHash, common.Bytes2Hex(txBytes)),
			sdk.NewAttribute(hmTypes.AttributeKeyTxLogIndex, strconv.FormatUint(msg.LogIndex, 10)),
			sdk.NewAttribute(hmTypes.AttributeKeySideTxResult, sideTxResult.String()),
			sdk.NewAttribute(types.AttributeKeyValidatorID, strconv.FormatUint(newValidator.ValId, 10)),
			sdk.NewAttribute(types.AttributeKeySigner, newValidator.Signer),
			sdk.NewAttribute(types.AttributeKeyValidatorNonce, strconv.FormatUint(msg.Nonce, 10)),
		),
	})
}

// PostHandleMsgStakeUpdate handles stake update message
func (s *sideMsgServer) PostHandleMsgStakeUpdate(ctx sdk.Context, msgI sdk.Msg, sideTxResult sidetxs.Vote) {
	msg, ok := msgI.(*types.MsgStakeUpdate)
	if !ok {
		s.k.Logger(ctx).Error("type mismatch for MsgStakeUpdate")
		return
	}

	// skip handler if stakeUpdate is not approved
	if sideTxResult != sidetxs.Vote_VOTE_YES {
		s.k.Logger(ctx).Debug("skipping stake update since side-tx didn't get yes votes")
		return
	}

	// check for replay attack
	blockNumber := new(big.Int).SetUint64(msg.BlockNumber)
	sequence := new(big.Int).Mul(blockNumber, big.NewInt(types.DefaultLogIndexUnit))
	sequence.Add(sequence, new(big.Int).SetUint64(msg.LogIndex))

	// check if incoming tx is older
	if s.k.HasStakingSequence(ctx, sequence.String()) {
		s.k.Logger(ctx).Error("older invalid tx found", "sequence", sequence.String())
		return
	}

	s.k.Logger(ctx).Debug("updating validator stake", "sideTxResult", sideTxResult)

	// pull validator from store
	validator, err := s.k.GetValidatorFromValID(ctx, msg.ValId)
	if err != nil {
		s.k.Logger(ctx).Error("failed to fetch validator from store", "validatorId", msg.ValId)
		return
	}

	validator.LastUpdated = sequence.String()
	validator.Nonce = msg.Nonce

	// set validator amount
	p, err := helper.GetPowerFromAmount(msg.NewAmount.BigInt())
	if err != nil {
		s.k.Logger(ctx).Error("error in calculating power value from amount", "err", err)
		return
	}

	validator.VotingPower = p.Int64()

	err = s.k.AddValidator(ctx, validator)
	if err != nil {
		s.k.Logger(ctx).Error("unable to update signer", "validatorID", validator.ValId, "error", err)
		return
	}

	// save staking sequence
	err = s.k.SetStakingSequence(ctx, sequence.String())
	if err != nil {
		s.k.Logger(ctx).Error("unable to set the sequence", "error", err)
		return
	}

	txBytes := ctx.TxBytes()

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeStakeUpdate,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(hmTypes.AttributeKeyTxHash, common.Bytes2Hex(txBytes)),   // tx hash
			sdk.NewAttribute(hmTypes.AttributeKeySideTxResult, sideTxResult.String()), // result
			sdk.NewAttribute(types.AttributeKeyValidatorID, strconv.FormatUint(validator.ValId, 10)),
			sdk.NewAttribute(types.AttributeKeyValidatorNonce, strconv.FormatUint(msg.Nonce, 10)),
		),
	})
}

// PostHandleMsgSignerUpdate handles signer update message
func (s *sideMsgServer) PostHandleMsgSignerUpdate(ctx sdk.Context, msgI sdk.Msg, sideTxResult sidetxs.Vote) {
	msg, ok := msgI.(*types.MsgSignerUpdate)
	if !ok {
		s.k.Logger(ctx).Error("type mismatch for MsgSignerUpdate")
		return
	}

	// Skip handler if signer update is not approved
	if sideTxResult != sidetxs.Vote_VOTE_YES {
		s.k.Logger(ctx).Debug("skipping signer update since side-tx didn't get yes votes")
		return
	}

	// Check for replay attack
	blockNumber := new(big.Int).SetUint64(msg.BlockNumber)
	sequence := new(big.Int).Mul(blockNumber, big.NewInt(types.DefaultLogIndexUnit))
	sequence.Add(sequence, new(big.Int).SetUint64(msg.LogIndex))
	// check if incoming tx is older
	if s.k.HasStakingSequence(ctx, sequence.String()) {
		s.k.Logger(ctx).Error("Older invalid tx found", "sequence", sequence.String())
		return
	}

	s.k.Logger(ctx).Debug("persisting signer update", "sideTxResult", sideTxResult)

	// Generate PubKey from PubKey in message and signer
	anyPk := msg.NewSignerPubKey
	newPubKey, ok := anyPk.GetCachedValue().(cryptotypes.PubKey)
	if !ok {
		s.k.Logger(ctx).Error("error in interfacing out pub key")
		return
	}

	if newPubKey.Type() != types.Secp256k1Type {
		s.k.Logger(ctx).Error("public key is invalid")
		return
	}

	newSigner := strings.ToLower(newPubKey.Address().String())

	// pull validator from store
	validator, err := s.k.GetValidatorFromValID(ctx, msg.ValId)
	if err != nil {
		s.k.Logger(ctx).Error("fetching of validator from store failed", "validatorId", msg.ValId)
		return
	}

	oldValidator := validator.Copy()

	validator.LastUpdated = sequence.String()
	validator.Nonce = msg.Nonce

	// check if we are actually updating signer
	if newSigner != validator.Signer {
		validator.Signer = newSigner
		validator.PubKey = anyPk

		s.k.Logger(ctx).Debug("updating new signer", "newSigner", newSigner, "oldSigner", oldValidator.Signer, "validatorID", msg.ValId)

	} else {
		s.k.Logger(ctx).Error("no signer change", "newSigner", newSigner, "oldSigner", oldValidator.Signer, "validatorID", msg.ValId)
		return
	}

	s.k.Logger(ctx).Debug("removing old validator", "validator", oldValidator.String())

	// remove the old validator from validator set
	oldValidator.EndEpoch, err = s.k.checkpointKeeper.GetAckCount(ctx)
	if err != nil {
		s.k.Logger(ctx).Error("unable to get ack count", "error", err)
		return
	}

	oldValidator.VotingPower = 0
	oldValidator.LastUpdated = sequence.String()

	oldValidator.Nonce = msg.Nonce

	// save old validator
	if err := s.k.AddValidator(ctx, *oldValidator); err != nil {
		s.k.Logger(ctx).Error("unable to update signer", "validatorId", validator.ValId, "error", err)
		return
	}

	// adding new validator
	s.k.Logger(ctx).Debug("adding new validator", "validator", validator.String())
	err = s.k.AddValidator(ctx, validator)
	if err != nil {
		s.k.Logger(ctx).Error("unable to update signer", "validatorID", validator.ValId, "error", err)
		return
	}

	// save staking sequence
	err = s.k.SetStakingSequence(ctx, sequence.String())
	if err != nil {
		s.k.Logger(ctx).Error("unable to set the sequence", "error", err)
		return
	}

	// Move heimdall fee to new signer
	oldAccAddress, err := addrCodec.NewHexCodec().StringToBytes(oldValidator.Signer)
	if err != nil {
		s.k.Logger(ctx).Error("error in coverting hex address to bytes", "error", err)
		return
	}

	newAccAddress, err := addrCodec.NewHexCodec().StringToBytes(validator.Signer)
	if err != nil {
		s.k.Logger(ctx).Error("error in coverting hex address to bytes", "error", err)
		return
	}

	coins := s.k.bankKeeper.GetBalance(ctx, oldAccAddress, authTypes.FeeToken)

	polygonPosTokensBalance := coins.Amount.Abs()
	if !polygonPosTokensBalance.IsZero() {
		s.k.Logger(ctx).Info("Transferring fee", "from", oldValidator.Signer, "to", validator.Signer, "balance", polygonPosTokensBalance.String())

		polygonPosCoins := sdk.Coins{coins}
		if err := s.k.bankKeeper.SendCoins(ctx, oldAccAddress, newAccAddress, polygonPosCoins); err != nil {
			s.k.Logger(ctx).Info("Error while transferring fee", "from", oldValidator.Signer, "to", validator.Signer, "balance", polygonPosTokensBalance.String())
			return
		}
	}

	txBytes := ctx.TxBytes()

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeSignerUpdate,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(hmTypes.AttributeKeyTxHash, common.Bytes2Hex(txBytes)),
			sdk.NewAttribute(hmTypes.AttributeKeySideTxResult, sideTxResult.String()),
			sdk.NewAttribute(types.AttributeKeyValidatorID, strconv.FormatUint(validator.ValId, 10)),
			sdk.NewAttribute(types.AttributeKeyValidatorNonce, strconv.FormatUint(msg.Nonce, 10)),
		),
	})
}

// PostHandleMsgValidatorExit handles msg validator exit
func (s *sideMsgServer) PostHandleMsgValidatorExit(ctx sdk.Context, msgI sdk.Msg, sideTxResult sidetxs.Vote) {
	msg, ok := msgI.(*types.MsgValidatorExit)
	if !ok {
		s.k.Logger(ctx).Error("type mismatch for MsgValidatorExit")
		return
	}

	// skip handler if validator exit is not approved
	if sideTxResult != sidetxs.Vote_VOTE_YES {
		s.k.Logger(ctx).Debug("skipping validator exit since side-tx didn't get yes votes")
		return
	}

	// check for replay attack
	blockNumber := new(big.Int).SetUint64(msg.BlockNumber)
	sequence := new(big.Int).Mul(blockNumber, big.NewInt(types.DefaultLogIndexUnit))
	sequence.Add(sequence, new(big.Int).SetUint64(msg.LogIndex))

	// check if incoming tx is older
	if s.k.HasStakingSequence(ctx, sequence.String()) {
		s.k.Logger(ctx).Error("Older invalid tx found", "sequence", sequence.String())
		return
	}

	s.k.Logger(ctx).Debug("persisting validator exit", "sideTxResult", sideTxResult)

	validator, err := s.k.GetValidatorFromValID(ctx, msg.ValId)
	if err != nil {
		s.k.Logger(ctx).Error("fetching of validator from store failed", "validatorID", msg.ValId)
		return
	}

	validator.EndEpoch = msg.DeactivationEpoch
	validator.LastUpdated = sequence.String()
	validator.Nonce = msg.Nonce

	// add deactivation time for validator
	if err := s.k.AddValidator(ctx, validator); err != nil {
		s.k.Logger(ctx).Error("error while setting deactivation epoch to validator", "error", err, "validatorID", validator.ValId)
		return
	}

	// save staking sequence
	err = s.k.SetStakingSequence(ctx, sequence.String())
	if err != nil {
		s.k.Logger(ctx).Error("unable to set the sequence", "error", err)
		return
	}

	txBytes := ctx.TxBytes()

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeValidatorExit,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(hmTypes.AttributeKeyTxHash, common.Bytes2Hex(txBytes)),
			sdk.NewAttribute(hmTypes.AttributeKeySideTxResult, sideTxResult.String()),
			sdk.NewAttribute(types.AttributeKeyValidatorID, strconv.FormatUint(validator.ValId, 10)),
			sdk.NewAttribute(types.AttributeKeyValidatorNonce, strconv.FormatUint(msg.Nonce, 10)),
		),
	})
}
