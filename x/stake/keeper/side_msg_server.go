package keeper

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"time"

	addrCodec "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authTypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/common"

	util "github.com/0xPolygon/heimdall-v2/common/hex"
	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/metrics/api"
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
	var err error
	startTime := time.Now()
	defer recordStakeMetric(api.SideHandleMsgValidatorJoinMethod, api.SideType, startTime, &err)

	msg, ok := msgI.(*types.MsgValidatorJoin)
	if !ok {
		s.k.Logger(ctx).Error(helper.ErrTypeMismatch("MsgValidatorJoin"))
		return sidetxs.Vote_VOTE_NO
	}

	s.k.Logger(ctx).Debug(helper.LogValidatingExternalCall("ValidatorJoin"),
		"txHash", common.Bytes2Hex(msg.TxHash),
		"logIndex", msg.LogIndex,
		"blockNumber", msg.BlockNumber,
	)

	err = msg.ValidateBasic()
	if err != nil {
		s.k.Logger(ctx).Error("Failed to validate MsgValidatorJoin", hmTypes.LogKeyError, err)
		return sidetxs.Vote_VOTE_NO
	}

	// check if the validator has been validator before
	if ok, err := s.k.DoesValIdExist(ctx, msg.ValId); ok {
		s.k.Logger(ctx).Error(hmTypes.ErrMsgValidatorAlreadyExists, hmTypes.LogKeyValidatorID, msg.ValId, hmTypes.LogKeyError, err)
		return sidetxs.Vote_VOTE_NO
	}

	// validate voting power
	if !helper.ValidateVotingPower(msg.Amount.BigInt(), s.k.Logger(ctx), msg.ValId, "stake") {
		return sidetxs.Vote_VOTE_NO
	}

	// add the sequence
	sequence := helper.CalculateSequence(msg.BlockNumber, msg.LogIndex)

	// check if the event has already been processed
	if s.k.HasStakingSequence(ctx, sequence) {
		s.k.Logger(ctx).Error(helper.LogEventAlreadyProcessedIn("stake"), hmTypes.LogKeySequence, sequence)
		return sidetxs.Vote_VOTE_NO
	}

	contractCaller := s.k.contractCaller

	// chainManager params
	params, err := s.k.cmKeeper.GetParams(ctx)
	if err != nil {
		s.k.Logger(ctx).Error(hmTypes.ErrMsgErrorInGettingChainManagerParams, hmTypes.LogKeyError, err)
		return sidetxs.Vote_VOTE_NO
	}

	chainParams := params.ChainParams

	// get and validate the main tx receipt
	receipt := helper.FetchAndValidateReceipt(
		contractCaller,
		helper.ReceiptValidationParams{
			TxHash:         msg.TxHash,
			MsgBlockNumber: msg.BlockNumber,
			Confirmations:  params.MainChainTxConfirmations,
			ModuleName:     "stake",
		},
		s.k.Logger(ctx),
	)
	if receipt == nil {
		return sidetxs.Vote_VOTE_NO
	}

	// decode validator join event
	eventLog, err := contractCaller.DecodeValidatorJoinEvent(chainParams.StakingInfoAddress, receipt, msg.LogIndex)
	if err != nil || eventLog == nil {
		s.k.Logger(ctx).Error("Error while decoding the validator join event receipt receipt")
		return sidetxs.Vote_VOTE_NO
	}

	// validate msg.SignerPubKey length
	if len(msg.SignerPubKey) != secp256k1.PubKeySize {
		s.k.Logger(ctx).Error("Invalid signer pubkey length in MsgValidatorJoin: " + strconv.Itoa(len(msg.SignerPubKey)) + ", expected length: " + fmt.Sprint(secp256k1.PubKeySize))
		return sidetxs.Vote_VOTE_NO
	}

	// Generate PubKey from PubKey in the message and signer
	pk := msg.SignerPubKey
	pubKey := secp256k1.PubKey{Key: pk}

	signer := pubKey.Address()
	ac := addrCodec.NewHexCodec()
	signerBytes, err := ac.StringToBytes(signer.String())
	if err != nil {
		s.k.Logger(ctx).Error(hmTypes.ErrMsgConvertSignerToBytes, hmTypes.LogKeyError, err)
		return sidetxs.Vote_VOTE_NO
	}
	eventLogSignerBytes, err := ac.StringToBytes(eventLog.Signer.String())
	if err != nil {
		s.k.Logger(ctx).Error(hmTypes.ErrMsgConvertEventLogSignerToBytes, hmTypes.LogKeyError, err)
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

	// check the public key first byte
	if !helper.IsPubKeyFirstByteValid(pubKey.Bytes()[0:1]) {
		s.k.Logger(ctx).Error(
			hmTypes.ErrMsgPubKeyFirstByteMismatch,
			"expected", "0x04",
			"received", pubKey.Bytes()[0:1])
		return sidetxs.Vote_VOTE_NO
	}

	// check the correspondence of signer pubKey in the message
	if !bytes.Equal(pubKey.Bytes()[1:], eventLog.SignerPubkey) {
		s.k.Logger(ctx).Error(
			"Signer PubKey does not match",
			"msgValidator", pubKey.String(),
			"mainChainValidator", common.Bytes2Hex(eventLog.SignerPubkey),
		)
		return sidetxs.Vote_VOTE_NO
	}

	// check msg id
	if eventLog.ValidatorId.Uint64() != msg.ValId {
		s.k.Logger(ctx).Error(
			hmTypes.ErrMsgIDMismatch,
			hmTypes.LogKeyMsgID, msg.ValId,
			hmTypes.LogKeyValidatorIdFromTx, eventLog.ValidatorId)
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
			hmTypes.ErrMsgAmountMismatch,
			hmTypes.LogKeyMsgAmount, msg.Amount,
			hmTypes.LogKeyAmountFromEvent, eventLog.Amount)
		return sidetxs.Vote_VOTE_NO
	}

	// check BlockNumber
	if receipt.BlockNumber.Uint64() != msg.BlockNumber {
		s.k.Logger(ctx).Error(
			hmTypes.ErrMsgBlockNumberMismatch,
			hmTypes.LogKeyMsgBlockNumber, msg.BlockNumber,
			hmTypes.LogKeyReceiptBlockNumber, receipt.BlockNumber.Uint64)
		return sidetxs.Vote_VOTE_NO
	}

	// check nonce
	if eventLog.Nonce.Uint64() != msg.Nonce {
		s.k.Logger(ctx).Error(
			hmTypes.ErrMsgNonceMismatch,
			hmTypes.LogKeyMsgNonce, msg.Nonce,
			hmTypes.LogKeyNonceFromTx, eventLog.Nonce)
		return sidetxs.Vote_VOTE_NO
	}

	s.k.Logger(ctx).Debug(helper.LogSuccessfullyValidated("ValidatorJoin"))

	return sidetxs.Vote_VOTE_YES
}

// SideHandleMsgStakeUpdate handles stake update message
func (s *sideMsgServer) SideHandleMsgStakeUpdate(ctx sdk.Context, msgI sdk.Msg) (result sidetxs.Vote) {
	var err error
	startTime := time.Now()
	defer recordStakeMetric(api.SideHandleMsgStakeUpdateMethod, api.SideType, startTime, &err)

	msg, ok := msgI.(*types.MsgStakeUpdate)
	if !ok {
		s.k.Logger(ctx).Error(helper.ErrTypeMismatch("MsgStakeUpdate"))
		return sidetxs.Vote_VOTE_NO
	}

	s.k.Logger(ctx).Debug(helper.LogValidatingExternalCall("MsgStakeUpdate"),
		"txHash", common.Bytes2Hex(msg.TxHash),
		"logIndex", msg.LogIndex,
		"blockNumber", msg.BlockNumber,
	)

	err = msg.ValidateBasic()
	if err != nil {
		s.k.Logger(ctx).Error("Failed to validate MsgStakeUpdate", hmTypes.LogKeyError, err)
		return sidetxs.Vote_VOTE_NO
	}

	// pull validator from store
	_, err = s.k.GetValidatorFromValID(ctx, msg.ValId)
	if err != nil {
		s.k.Logger(ctx).Error(hmTypes.ErrMsgFailedToFetchValidator, hmTypes.LogKeyValidatorID, msg.ValId, hmTypes.LogKeyError, err)
		return sidetxs.Vote_VOTE_NO
	}

	// add the sequence
	sequence := helper.CalculateSequence(msg.BlockNumber, msg.LogIndex)

	// check if the event has already been processed
	if s.k.HasStakingSequence(ctx, sequence) {
		s.k.Logger(ctx).Error(helper.LogEventAlreadyProcessedIn("stake"), hmTypes.LogKeySequence, sequence)
		return sidetxs.Vote_VOTE_NO
	}

	// set validator amount
	if !helper.ValidateVotingPower(msg.NewAmount.BigInt(), s.k.Logger(ctx), msg.ValId, "stake") {
		return sidetxs.Vote_VOTE_NO
	}

	params, err := s.k.cmKeeper.GetParams(ctx)
	if err != nil {
		s.k.Logger(ctx).Error(hmTypes.ErrMsgFailedToFetchParams, hmTypes.LogKeyError, err)
		return sidetxs.Vote_VOTE_NO
	}

	// get and validate the main tx receipt
	contractCaller := s.k.contractCaller
	receipt := helper.FetchAndValidateReceipt(
		contractCaller,
		helper.ReceiptValidationParams{
			TxHash:         msg.TxHash,
			MsgBlockNumber: msg.BlockNumber,
			Confirmations:  params.MainChainTxConfirmations,
			ModuleName:     "stake",
		},
		s.k.Logger(ctx),
	)
	if receipt == nil {
		return sidetxs.Vote_VOTE_NO
	}

	chainParams := params.ChainParams
	eventLog, err := contractCaller.DecodeValidatorStakeUpdateEvent(chainParams.StakingInfoAddress, receipt, msg.LogIndex)
	if err != nil || eventLog == nil {
		s.k.Logger(ctx).Error(hmTypes.ErrMsgErrorFetchingLog, hmTypes.LogKeyError, err)
		return sidetxs.Vote_VOTE_NO
	}

	if eventLog.ValidatorId.Uint64() != msg.ValId {
		s.k.Logger(ctx).Error(
			hmTypes.ErrMsgIDMismatch,
			hmTypes.LogKeyMsgID, msg.ValId,
			hmTypes.LogKeyValidatorIdFromTx, eventLog.ValidatorId)
		return sidetxs.Vote_VOTE_NO
	}

	// check amount
	if eventLog.NewAmount.Cmp(msg.NewAmount.BigInt()) != 0 {
		s.k.Logger(ctx).Error(hmTypes.ErrMsgAmountMismatch,
			hmTypes.LogKeyNewAmount, msg.NewAmount,
			hmTypes.LogKeyAmountFromEvent, eventLog.NewAmount)
		return sidetxs.Vote_VOTE_NO
	}

	// check nonce
	if eventLog.Nonce.Uint64() != msg.Nonce {
		s.k.Logger(ctx).Error(hmTypes.ErrMsgNonceMismatch,
			hmTypes.LogKeyMsgNonce, msg.Nonce,
			hmTypes.LogKeyNonceFromTx, eventLog.Nonce)
		return sidetxs.Vote_VOTE_NO
	}

	s.k.Logger(ctx).Debug(helper.LogSuccessfullyValidated("MsgStakeUpdate"))

	return sidetxs.Vote_VOTE_YES
}

// SideHandleMsgSignerUpdate handles signer update message
func (s *sideMsgServer) SideHandleMsgSignerUpdate(ctx sdk.Context, msgI sdk.Msg) (result sidetxs.Vote) {
	var err error
	startTime := time.Now()
	defer recordStakeMetric(api.SideHandleMsgSignerUpdateMethod, api.SideType, startTime, &err)

	msg, ok := msgI.(*types.MsgSignerUpdate)
	if !ok {
		s.k.Logger(ctx).Error(helper.ErrTypeMismatch("MsgSignerUpdate"))
		return sidetxs.Vote_VOTE_NO
	}

	s.k.Logger(ctx).Debug(helper.LogValidatingMsg("MsgSignerUpdate"),
		"txHash", common.Bytes2Hex(msg.TxHash),
		"logIndex", msg.LogIndex,
		"blockNumber", msg.BlockNumber,
	)

	err = msg.ValidateBasic()
	if err != nil {
		s.k.Logger(ctx).Error("Failed to validate MsgSignerUpdate", hmTypes.LogKeyError, err)
		return sidetxs.Vote_VOTE_NO
	}

	// chainManager params
	params, err := s.k.cmKeeper.GetParams(ctx)
	if err != nil {
		s.k.Logger(ctx).Error(hmTypes.ErrMsgFailedToFetchParams, hmTypes.LogKeyError, err)
		return sidetxs.Vote_VOTE_NO
	}

	// pull validator from store
	validator, err := s.k.GetValidatorFromValID(ctx, msg.ValId)
	if err != nil {
		s.k.Logger(ctx).Error(hmTypes.ErrMsgFailedToFetchValidator, hmTypes.LogKeyValidatorID, msg.ValId, hmTypes.LogKeyError, err)
		return sidetxs.Vote_VOTE_NO
	}
	oldSigner := util.FormatAddress(validator.Signer)

	// add the sequence
	sequence := helper.CalculateSequence(msg.BlockNumber, msg.LogIndex)

	// check if the event has already been processed
	if s.k.HasStakingSequence(ctx, sequence) {
		s.k.Logger(ctx).Error(helper.LogEventAlreadyProcessedIn("stake"), hmTypes.LogKeySequence, sequence)
		return sidetxs.Vote_VOTE_NO
	}

	// get and validate the main tx receipt
	contractCaller := s.k.contractCaller
	receipt := helper.FetchAndValidateReceipt(
		contractCaller,
		helper.ReceiptValidationParams{
			TxHash:         msg.TxHash,
			MsgBlockNumber: msg.BlockNumber,
			Confirmations:  params.MainChainTxConfirmations,
			ModuleName:     "stake",
		},
		s.k.Logger(ctx),
	)
	if receipt == nil {
		return sidetxs.Vote_VOTE_NO
	}

	chainParams := params.ChainParams
	eventLog, err := contractCaller.DecodeSignerUpdateEvent(chainParams.StakingInfoAddress, receipt, msg.LogIndex)
	if err != nil || eventLog == nil {
		s.k.Logger(ctx).Error(hmTypes.ErrMsgErrorFetchingLog, hmTypes.LogKeyError, err)
		return sidetxs.Vote_VOTE_NO
	}

	if eventLog.ValidatorId.Uint64() != msg.ValId {
		s.k.Logger(ctx).Error(hmTypes.ErrMsgIDMismatch, hmTypes.LogKeyMsgID, msg.ValId, hmTypes.LogKeyValidatorIdFromTx, eventLog.ValidatorId)
		return sidetxs.Vote_VOTE_NO
	}

	// validate msg.NewSignerPubKey length
	if len(msg.NewSignerPubKey) != secp256k1.PubKeySize {
		s.k.Logger(ctx).Error("Invalid signer pubkey length in MsgSignerUpdate: " + strconv.Itoa(len(msg.NewSignerPubKey)) + ", expected length: " + fmt.Sprint(secp256k1.PubKeySize))
		return sidetxs.Vote_VOTE_NO
	}

	if !helper.IsPubKeyFirstByteValid(msg.NewSignerPubKey[0:1]) {
		s.k.Logger(ctx).Error(hmTypes.ErrMsgPubKeyFirstByteMismatch, "expected", "0x04", "received", msg.NewSignerPubKey[0:1])
		return sidetxs.Vote_VOTE_NO
	}

	if !bytes.Equal(eventLog.SignerPubkey, msg.NewSignerPubKey[1:]) {
		s.k.Logger(ctx).Error("NewSigner pubKey in txHash and msg don't match", "msgPubKey", common.Bytes2Hex(msg.NewSignerPubKey), "pubKeyTx", eventLog.SignerPubkey[:])
		return sidetxs.Vote_VOTE_NO
	}

	newPubKey := secp256k1.PubKey{Key: msg.NewSignerPubKey}
	newSigner := newPubKey.Address()

	ac := addrCodec.NewHexCodec()
	signerBytes, err := ac.StringToBytes(newSigner.String())
	if err != nil {
		s.k.Logger(ctx).Error(hmTypes.ErrMsgConvertSignerToBytes, hmTypes.LogKeyError, err)
		return sidetxs.Vote_VOTE_NO
	}

	// check if the new signer address is the same as the existing signer
	if newSigner.String() == oldSigner {
		// No signer change
		s.k.Logger(ctx).Error("NewSigner is the same as old signer in stake side handler")
		return sidetxs.Vote_VOTE_NO
	}

	eventLogSignerBytes, err := ac.StringToBytes(eventLog.NewSigner.String())
	if err != nil {
		s.k.Logger(ctx).Error(hmTypes.ErrMsgConvertEventLogSignerToBytes, hmTypes.LogKeyError, err)
		return sidetxs.Vote_VOTE_NO
	}

	// check signer corresponding to pubKey matches signer from event
	if !bytes.Equal(signerBytes, eventLogSignerBytes) {
		s.k.Logger(ctx).Error("Signer address does not match event log signer address", hmTypes.LogKeySigner, newSigner.String(), "mainChainValidator", eventLog.NewSigner.Hex())
		return sidetxs.Vote_VOTE_NO
	}

	// check nonce
	if eventLog.Nonce.Uint64() != msg.Nonce {
		s.k.Logger(ctx).Error(hmTypes.ErrMsgNonceMismatch, hmTypes.LogKeyMsgNonce, msg.Nonce, hmTypes.LogKeyNonceFromTx, eventLog.Nonce)
		return sidetxs.Vote_VOTE_NO
	}

	s.k.Logger(ctx).Debug(helper.LogSuccessfullyValidated("MsgSignerUpdate"))

	return sidetxs.Vote_VOTE_YES
}

// SideHandleMsgValidatorExit handles side msg validator exit
func (s *sideMsgServer) SideHandleMsgValidatorExit(ctx sdk.Context, msgI sdk.Msg) (result sidetxs.Vote) {
	var err error
	startTime := time.Now()
	defer recordStakeMetric(api.SideHandleMsgValidatorExitMethod, api.SideType, startTime, &err)

	msg, ok := msgI.(*types.MsgValidatorExit)
	if !ok {
		s.k.Logger(ctx).Error(helper.ErrTypeMismatch("MsgValidatorExit"))
		return sidetxs.Vote_VOTE_NO
	}

	s.k.Logger(ctx).Debug(helper.LogValidatingExternalCall("MsgValidatorExit"),
		"txHash", common.Bytes2Hex(msg.TxHash),
		"logIndex", msg.LogIndex,
		"blockNumber", msg.BlockNumber,
	)

	err = msg.ValidateBasic()
	if err != nil {
		s.k.Logger(ctx).Error("Failed to validate MsgValidatorExit", hmTypes.LogKeyError, err)
		return sidetxs.Vote_VOTE_NO
	}

	validator, err := s.k.GetValidatorFromValID(ctx, msg.ValId)
	if err != nil {
		s.k.Logger(ctx).Error(hmTypes.ErrMsgFailedToFetchValidator, hmTypes.LogKeyValidatorID, msg.ValId, hmTypes.LogKeyError, err)
		return sidetxs.Vote_VOTE_NO
	}

	s.k.Logger(ctx).Debug("Validator in store", "validator", validator)
	// check if the validator deactivation period is set
	if validator.EndEpoch != 0 {
		s.k.Logger(ctx).Error("Validator already unBonded in stake side handler")
		return sidetxs.Vote_VOTE_NO
	}

	// add the sequence
	sequence := helper.CalculateSequence(msg.BlockNumber, msg.LogIndex)

	// check if the event has already been processed
	if s.k.HasStakingSequence(ctx, sequence) {
		s.k.Logger(ctx).Error(helper.LogEventAlreadyProcessedIn("stake"), hmTypes.LogKeySequence, sequence)
		return sidetxs.Vote_VOTE_NO
	}

	contractCaller := s.k.contractCaller

	// chainManager params
	params, err := s.k.cmKeeper.GetParams(ctx)
	if err != nil {
		s.k.Logger(ctx).Error(hmTypes.ErrMsgFailedToFetchParams, hmTypes.LogKeyError, err)
		return sidetxs.Vote_VOTE_NO
	}

	chainParams := params.ChainParams

	// get and validate the main tx receipt
	receipt := helper.FetchAndValidateReceipt(
		contractCaller,
		helper.ReceiptValidationParams{
			TxHash:         msg.TxHash,
			MsgBlockNumber: msg.BlockNumber,
			Confirmations:  params.MainChainTxConfirmations,
			ModuleName:     "stake",
		},
		s.k.Logger(ctx),
	)
	if receipt == nil {
		return sidetxs.Vote_VOTE_NO
	}

	// decode validator exit
	eventLog, err := contractCaller.DecodeValidatorExitEvent(chainParams.StakingInfoAddress, receipt, msg.LogIndex)
	if err != nil || eventLog == nil {
		s.k.Logger(ctx).Error(hmTypes.ErrMsgErrorFetchingLog, hmTypes.LogKeyError, err)
		return sidetxs.Vote_VOTE_NO
	}

	if eventLog.ValidatorId.Uint64() != msg.ValId {
		s.k.Logger(ctx).Error(hmTypes.ErrMsgIDMismatch, hmTypes.LogKeyMsgID, msg.ValId, hmTypes.LogKeyValidatorIdFromTx, eventLog.ValidatorId)
		return sidetxs.Vote_VOTE_NO
	}

	if eventLog.DeactivationEpoch.Uint64() != msg.DeactivationEpoch {
		s.k.Logger(ctx).Error("DeactivationEpoch in message doesn't match with deactivationEpoch in log", "msgDeactivationEpoch", msg.DeactivationEpoch, "deactivationEpochFromTx", eventLog.DeactivationEpoch.Uint64)
		return sidetxs.Vote_VOTE_NO
	}

	// check nonce
	if eventLog.Nonce.Uint64() != msg.Nonce {
		s.k.Logger(ctx).Error(hmTypes.ErrMsgNonceMismatch, hmTypes.LogKeyMsgNonce, msg.Nonce, hmTypes.LogKeyNonceFromTx, eventLog.Nonce)
		return sidetxs.Vote_VOTE_NO
	}

	s.k.Logger(ctx).Debug(helper.LogSuccessfullyValidated("ValidatorExit"))

	return sidetxs.Vote_VOTE_YES
}

// PostHandleMsgValidatorJoin handles validator join message
func (s *sideMsgServer) PostHandleMsgValidatorJoin(ctx sdk.Context, msgI sdk.Msg, sideTxResult sidetxs.Vote) error {
	var err error
	startTime := time.Now()
	defer recordStakeMetric(api.PostHandleMsgValidatorJoinMethod, api.PostType, startTime, &err)

	msg, ok := msgI.(*types.MsgValidatorJoin)
	if !ok {
		err := errors.New(helper.ErrTypeMismatch("MsgValidatorJoin"))
		s.k.Logger(ctx).Error(err.Error())
		return err
	}

	// Skip handler if validator join is not approved
	if !helper.IsSideTxApproved(sideTxResult) {
		s.k.Logger(ctx).Debug(helper.ErrSkippingMsg("ValidatorJoin"))
		return errors.New(hmTypes.ErrMsgSideTxRejected)
	}

	err = msg.ValidateBasic()
	if err != nil {
		s.k.Logger(ctx).Error(hmTypes.ErrMsgValidationFailed, hmTypes.LogKeyError, err)
		return errors.New(helper.ErrInvalidSideTxMsg("ValidatorJoin"))
	}

	// Check for replay attack
	sequence := helper.CalculateSequence(msg.BlockNumber, msg.LogIndex)

	// check if the event has already been processed
	if s.k.HasStakingSequence(ctx, sequence) {
		s.k.Logger(ctx).Error(hmTypes.ErrMsgEventAlreadyProcessed, hmTypes.LogKeySequence, sequence)
		return errors.New(hmTypes.ErrMsgOldEventsNotAllowed)
	}

	s.k.Logger(ctx).Debug("Adding validator to state", "sideTxResult", sideTxResult)

	pubKey := secp256k1.PubKey{Key: msg.SignerPubKey}

	if pubKey.Type() != types.Secp256k1Type {
		s.k.Logger(ctx).Error(hmTypes.ErrMsgInvalidPubKey)
		return errors.New(hmTypes.ErrMsgInvalidPubKey)
	}

	signer := util.FormatAddress(pubKey.Address().String())

	// get voting power from amount
	votingPower, err := helper.GetPowerFromAmount(msg.Amount.BigInt())
	if err != nil {
		s.k.Logger(ctx).Error(fmt.Sprintf("%s %v for validator %v", hmTypes.ErrMsgInvalidAmount, msg.Amount, msg.ValId))
		return err
	}

	// create new validator
	newValidator := types.Validator{
		ValId:       msg.ValId,
		StartEpoch:  msg.ActivationEpoch,
		EndEpoch:    0,
		Nonce:       msg.Nonce,
		VotingPower: votingPower.Int64(),
		PubKey:      pubKey.Bytes(),
		Signer:      signer,
		LastUpdated: sequence,
	}

	// add validator to store
	s.k.Logger(ctx).Debug("Adding new validator to state", "validator", newValidator.String())

	if err = s.k.AddValidator(ctx, newValidator); err != nil {
		s.k.Logger(ctx).Error("Unable to add validator to state", "validator", newValidator.String(), hmTypes.LogKeyError, err)
		return err
	}

	// Add Validator signing info. It is required for slashing module
	s.k.Logger(ctx).Debug("Adding signing info for new validator")

	// save the staking sequence
	err = s.k.SetStakingSequence(ctx, sequence)
	if err != nil {
		s.k.Logger(ctx).Error(hmTypes.ErrMsgUnableToSetSequence, hmTypes.LogKeyError, err)
		return err
	}

	s.k.Logger(ctx).Debug("New validator successfully joined", "validator", strconv.FormatUint(newValidator.ValId, 10))

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

	return nil
}

// PostHandleMsgStakeUpdate handles stake update message
func (s *sideMsgServer) PostHandleMsgStakeUpdate(ctx sdk.Context, msgI sdk.Msg, sideTxResult sidetxs.Vote) error {
	var err error
	startTime := time.Now()
	defer recordStakeMetric(api.PostHandleMsgStakeUpdateMethod, api.PostType, startTime, &err)

	msg, ok := msgI.(*types.MsgStakeUpdate)
	if !ok {
		err := errors.New(helper.ErrTypeMismatch("MsgStakeUpdate"))
		s.k.Logger(ctx).Error(err.Error())
		return err
	}

	// skip handler if stakeUpdate is not approved
	if !helper.IsSideTxApproved(sideTxResult) {
		s.k.Logger(ctx).Debug(helper.ErrSkippingMsg("MsgSignerUpdate"))
		return errors.New(hmTypes.ErrMsgSideTxRejected)
	}

	err = msg.ValidateBasic()
	if err != nil {
		s.k.Logger(ctx).Error(hmTypes.ErrMsgValidationFailed, hmTypes.LogKeyError, err)
		return errors.New(helper.ErrInvalidSideTxMsg("MsgStakeUpdate"))
	}

	// check for replay attack
	sequence := helper.CalculateSequence(msg.BlockNumber, msg.LogIndex)

	// check if the event has already been processed
	if s.k.HasStakingSequence(ctx, sequence) {
		s.k.Logger(ctx).Error(hmTypes.ErrMsgEventAlreadyProcessed, hmTypes.LogKeySequence, sequence)
		return errors.New(hmTypes.ErrMsgOldEventsNotAllowed)
	}

	// pull validator from store
	validator, err := s.k.GetValidatorFromValID(ctx, msg.ValId)
	if err != nil {
		s.k.Logger(ctx).Error(hmTypes.ErrMsgFailedToFetchValidator, hmTypes.LogKeyValidatorID, msg.ValId)
		return err
	}

	// Check nonce validity just before applying the state update
	if msg.Nonce != validator.Nonce+1 {
		s.k.Logger(ctx).Error(helper.ErrIncorrectNonceDuringPostHandle("StakeUpdate"), hmTypes.LogKeyValidatorNonce, validator.Nonce, hmTypes.LogKeyMsgNonce, msg.Nonce)
		return errors.New(helper.ErrIncorrectNonceDuringPostHandle("StakeUpdate"))
	}

	s.k.Logger(ctx).Debug("Updating validator stake", "sideTxResult", sideTxResult)

	validator.LastUpdated = sequence
	validator.Nonce = msg.Nonce

	// set validator amount
	p, err := helper.GetPowerFromAmount(msg.NewAmount.BigInt())
	if err != nil {
		s.k.Logger(ctx).Error("Error in calculating power value from amount", hmTypes.LogKeyError, err)
		return err
	}

	validator.VotingPower = p.Int64()

	err = s.k.AddValidator(ctx, validator)
	if err != nil {
		s.k.Logger(ctx).Error("Unable to update signer", hmTypes.LogKeyValidatorID, validator.ValId, hmTypes.LogKeyError, err)
		return err
	}

	// save the staking sequence
	err = s.k.SetStakingSequence(ctx, sequence)
	if err != nil {
		s.k.Logger(ctx).Error(hmTypes.ErrMsgUnableToSetSequence, hmTypes.LogKeyError, err)
		return err
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

	return nil
}

// PostHandleMsgSignerUpdate handles signer update message
func (s *sideMsgServer) PostHandleMsgSignerUpdate(ctx sdk.Context, msgI sdk.Msg, sideTxResult sidetxs.Vote) error {
	var err error
	startTime := time.Now()
	defer recordStakeMetric(api.PostHandleMsgSignerUpdateMethod, api.PostType, startTime, &err)

	msg, ok := msgI.(*types.MsgSignerUpdate)
	if !ok {
		err := errors.New(helper.ErrTypeMismatch("MsgSignerUpdate"))
		s.k.Logger(ctx).Error(err.Error())
		return err
	}

	// Skip handler if signer update is not approved
	if !helper.IsSideTxApproved(sideTxResult) {
		s.k.Logger(ctx).Debug(helper.ErrSkippingMsg("MsgSignerUpdate"))
		return errors.New(hmTypes.ErrMsgSideTxRejected)
	}

	err = msg.ValidateBasic()
	if err != nil {
		s.k.Logger(ctx).Error(hmTypes.ErrMsgValidationFailed, hmTypes.LogKeyError, err)
		return errors.New(helper.ErrInvalidSideTxMsg("MsgSignerUpdate"))
	}

	// Check for replay attack
	sequence := helper.CalculateSequence(msg.BlockNumber, msg.LogIndex)
	// check if the event has already been processed
	if s.k.HasStakingSequence(ctx, sequence) {
		s.k.Logger(ctx).Error(hmTypes.ErrMsgEventAlreadyProcessed, hmTypes.LogKeySequence, sequence)
		return errors.New(hmTypes.ErrMsgOldEventsNotAllowed)
	}

	// Generate PubKey from PubKey in the message and signer
	newPubKey := secp256k1.PubKey{Key: msg.NewSignerPubKey}

	if newPubKey.Type() != types.Secp256k1Type {
		s.k.Logger(ctx).Error(hmTypes.ErrMsgInvalidPubKey)
		return errors.New(hmTypes.ErrMsgInvalidPubKey)
	}

	newSigner := util.FormatAddress(newPubKey.Address().String())

	// pull validator from store
	validator, err := s.k.GetValidatorFromValID(ctx, msg.ValId)
	if err != nil {
		s.k.Logger(ctx).Error(hmTypes.ErrMsgFailedToFetchValidator, hmTypes.LogKeyValidatorID, msg.ValId)
		return err
	}

	// Check nonce validity just before applying the state update
	if msg.Nonce != validator.Nonce+1 {
		s.k.Logger(ctx).Error(helper.ErrIncorrectNonceDuringPostHandle("SignerUpdate"), hmTypes.LogKeyValidatorNonce, validator.Nonce, hmTypes.LogKeyMsgNonce, msg.Nonce)
		return errors.New(helper.ErrIncorrectNonceDuringPostHandle("SignerUpdate"))
	}

	s.k.Logger(ctx).Debug("Persisting signer update", "sideTxResult", sideTxResult)

	oldValidator := validator.Copy()

	validator.LastUpdated = sequence
	validator.Nonce = msg.Nonce

	// check if we are actually updating signer
	if newSigner != validator.Signer {
		validator.Signer = newSigner
		validator.PubKey = newPubKey.Bytes()

		s.k.Logger(ctx).Debug("Updating new signer", "newSigner", newSigner, "oldSigner", oldValidator.Signer, "validatorID", msg.ValId)

	} else {
		s.k.Logger(ctx).Error("No signer change", "newSigner", newSigner, "oldSigner", oldValidator.Signer, "validatorID", msg.ValId)
		return errors.New("no signer change")
	}

	s.k.Logger(ctx).Debug("Removing old validator", "validator", oldValidator.String())

	// remove the old validator from the validator set
	oldValidator.EndEpoch, err = s.k.checkpointKeeper.GetAckCount(ctx)
	if err != nil {
		s.k.Logger(ctx).Error("Unable to get ack count", "error", err)
		return err
	}

	oldValidator.VotingPower = 0
	oldValidator.LastUpdated = sequence

	oldValidator.Nonce = msg.Nonce

	// save old validator
	if err := s.k.AddValidator(ctx, *oldValidator); err != nil {
		s.k.Logger(ctx).Error("Unable to update signer", hmTypes.LogKeyValidatorID, validator.ValId, hmTypes.LogKeyError, err)
		return err
	}

	// adding new validator
	s.k.Logger(ctx).Debug("Adding new validator", "validator", validator.String())
	err = s.k.AddValidator(ctx, validator)
	if err != nil {
		s.k.Logger(ctx).Error("Unable to update signer", hmTypes.LogKeyValidatorID, validator.ValId, hmTypes.LogKeyError, err)
		return err
	}

	// save the staking sequence
	err = s.k.SetStakingSequence(ctx, sequence)
	if err != nil {
		s.k.Logger(ctx).Error(hmTypes.ErrMsgUnableToSetSequence, hmTypes.LogKeyError, err)
		return err
	}

	// Move heimdall fee to new signer
	oldAccAddress, err := addrCodec.NewHexCodec().StringToBytes(oldValidator.Signer)
	if err != nil {
		s.k.Logger(ctx).Error(hmTypes.ErrMsgConvertHexToBytes, hmTypes.LogKeyError, err)
		return err
	}

	newAccAddress, err := addrCodec.NewHexCodec().StringToBytes(validator.Signer)
	if err != nil {
		s.k.Logger(ctx).Error(hmTypes.ErrMsgConvertHexToBytes, hmTypes.LogKeyError, err)
		return err
	}

	coins := s.k.bankKeeper.GetBalance(ctx, oldAccAddress, authTypes.FeeToken)

	// validate balance
	if coins.IsNegative() {
		s.k.Logger(ctx).Error("Negative balance for fee token", "address", oldValidator.Signer, "balance", coins.String())
		return errors.New("negative balance for fee token")
	}

	polTokensBalance := coins.Amount.Abs()
	if !polTokensBalance.IsZero() {
		s.k.Logger(ctx).Info("Transferring fee", "from", oldValidator.Signer, "to", validator.Signer, "balance", polTokensBalance.String())

		polCoins := sdk.Coins{coins}
		if err := s.k.bankKeeper.SendCoins(ctx, oldAccAddress, newAccAddress, polCoins); err != nil {
			s.k.Logger(ctx).Info("Error while transferring fee", "from", oldValidator.Signer, "to", validator.Signer, "balance", polTokensBalance.String())
			return err
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

	return nil
}

// PostHandleMsgValidatorExit handles msg validator exit
func (s *sideMsgServer) PostHandleMsgValidatorExit(ctx sdk.Context, msgI sdk.Msg, sideTxResult sidetxs.Vote) error {
	var err error
	startTime := time.Now()
	defer recordStakeMetric(api.PostHandleMsgValidatorExitMethod, api.PostType, startTime, &err)

	msg, ok := msgI.(*types.MsgValidatorExit)
	if !ok {
		err := errors.New(helper.ErrTypeMismatch("MsgValidatorExit"))
		s.k.Logger(ctx).Error(err.Error())
		return err
	}

	// skip handler if validator exit is not approved
	if !helper.IsSideTxApproved(sideTxResult) {
		s.k.Logger(ctx).Debug(helper.ErrSkippingMsg("ValidatorExit"))
		return errors.New(hmTypes.ErrMsgSideTxRejected)
	}

	err = msg.ValidateBasic()
	if err != nil {
		s.k.Logger(ctx).Error(hmTypes.ErrMsgValidationFailed, hmTypes.LogKeyError, err)
		return errors.New(helper.ErrInvalidSideTxMsg("MsgValidatorExit"))
	}

	// check for replay attack
	sequence := helper.CalculateSequence(msg.BlockNumber, msg.LogIndex)

	// check if the event has already been processed
	if s.k.HasStakingSequence(ctx, sequence) {
		s.k.Logger(ctx).Error(hmTypes.ErrMsgEventAlreadyProcessed, hmTypes.LogKeySequence, sequence)
		return errors.New(hmTypes.ErrMsgOldEventsNotAllowed)
	}

	validator, err := s.k.GetValidatorFromValID(ctx, msg.ValId)
	if err != nil {
		s.k.Logger(ctx).Error(hmTypes.ErrMsgFailedToFetchValidator, hmTypes.LogKeyValidatorID, msg.ValId)
		return err
	}

	// Check nonce validity just before applying the state update
	if msg.Nonce != validator.Nonce+1 {
		s.k.Logger(ctx).Error(helper.ErrIncorrectNonceDuringPostHandle("ValidatorExit"), hmTypes.LogKeyValidatorNonce, validator.Nonce, hmTypes.LogKeyMsgNonce, msg.Nonce)
		return errors.New(helper.ErrIncorrectNonceDuringPostHandle("ValidatorExit"))
	}

	s.k.Logger(ctx).Debug("Persisting validator exit", "sideTxResult", sideTxResult)

	validator.EndEpoch = msg.DeactivationEpoch
	validator.LastUpdated = sequence
	validator.Nonce = msg.Nonce
	validator.VotingPower = 0

	// add deactivation time for validator
	if err := s.k.AddValidator(ctx, validator); err != nil {
		s.k.Logger(ctx).Error("Error while setting deactivation epoch to validator", hmTypes.LogKeyError, err, hmTypes.LogKeyValidatorID, validator.ValId)
		return err
	}

	// save the staking sequence
	err = s.k.SetStakingSequence(ctx, sequence)
	if err != nil {
		s.k.Logger(ctx).Error(hmTypes.ErrMsgUnableToSetSequence, hmTypes.LogKeyError, err)
		return err
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

	return nil
}

// recordStakeMetric records metrics for side and post-handlers.
func recordStakeMetric(method string, apiType string, start time.Time, err *error) {
	success := *err == nil
	api.RecordAPICallWithStart(api.StakeSubsystem, method, apiType, success, start)
}
