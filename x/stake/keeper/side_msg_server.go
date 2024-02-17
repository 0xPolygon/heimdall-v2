package keeper

// import (
// 	"bytes"
// 	"fmt"
// 	"math/big"
// 	"strconv"s
// 	"strings"

// 	"github.com/0xPolygon/heimdall-v2/x/stake/types"

// 	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
// 	sdk "github.com/cosmos/cosmos-sdk/types"

// 	"github.com/0xPolygon/heimdall-v2/helper"
// 	hmTypes "github.com/0xPolygon/heimdall-v2/x/types"
// 	voteTypes "github.com/0xPolygon/heimdall-v2/x/types"
// )

// type sideMsgServer struct {
// 	*Keeper
// }

// // NewMsgServerImpl returns an implementation of the staking MsgServer interface
// // for the provided Keeper.
// func NewSideMsgServerImpl(keeper *Keeper) types.SideMsgServer {
// 	return &sideMsgServer{Keeper: keeper}
// }

// // NewSideTxHandler returns a side handler for "staking" type messages.
// func (srv *sideMsgServer) SideTxHandler(methodName string) hmTypes.SideTxHandler {

// 	switch methodName {
// 	case types.JoinValidatorMethod:
// 		return srv.SideHandleMsgValidatorJoin
// 	case types.StakeUpdateMethod:
// 		return srv.SideHandleMsgStakeUpdate
// 	case types.SignerUpdateMethod:
// 		return srv.SideHandleMsgSignerUpdate
// 	case types.ValidatorExitMethod:
// 		return srv.SideHandleMsgValidatorExit
// 	default:
// 		return nil
// 	}
// }

// // NewSideTxHandler returns a side handler for "staking" type messages.
// func (srv *sideMsgServer) PostTxHandler(methodName string) hmTypes.PostTxHandler {

// 	switch methodName {
// 	case types.JoinValidatorMethod:
// 		return srv.PostHandleMsgValidatorJoin
// 	case types.StakeUpdateMethod:
// 		return srv.PostHandleMsgStakeUpdate
// 	case types.SignerUpdateMethod:
// 		return srv.PostHandleMsgSignerUpdate
// 	case types.ValidatorExitMethod:
// 		return srv.PostHandleMsgValidatorExit
// 	default:
// 		return nil
// 	}
// }

// // SideHandleMsgValidatorJoin side msg validator join
// func (k *sideMsgServer) SideHandleMsgValidatorJoin(ctx sdk.Context, _msg sdk.Msg) (result voteTypes.Vote) {
// 	msg, ok := _msg.(*types.MsgValidatorJoin)
// 	if !ok {
// 		k.Logger(ctx).Error("msg type mismatched")
// 		return voteTypes.Vote_VOTE_NO
// 	}

// 	k.Logger(ctx).Debug("✅ Validating External call for validator join msg",
// 		"txHash", hmTypes.BytesToHeimdallHash(msg.TxHash.Bytes()),
// 		"logIndex", msg.LogIndex,
// 		"blockNumber", msg.BlockNumber,
// 	)

// 	contractCaller := k.IContractCaller

// 	// chainManager params
// 	params := k.chainKeeper.GetParams(ctx)
// 	chainParams := params.ChainParams

// 	// get main tx receipt
// 	receipt, err := contractCaller.GetConfirmedTxReceipt(msg.TxHash.EthHash(), params.MainchainTxConfirmations)
// 	if err != nil || receipt == nil {
// 		k.Logger(ctx).Error("Need for more ethereum blocks to fetch the confirmed tx receipt")
// 		return voteTypes.Vote_VOTE_NO
// 	}

// 	// decode validator join event
// 	eventLog, err := contractCaller.DecodeValidatorJoinEvent(chainParams.StakingInfoAddress.EthAddress(), receipt, msg.LogIndex)
// 	if err != nil || eventLog == nil {
// 		k.Logger(ctx).Error("Error while decoding the receipt")
// 		return voteTypes.Vote_VOTE_NO
// 	}

// 	// Generate PubKey from Pubkey in message and signer
// 	anyPk := msg.SignerPubKey
// 	pubKey, ok := anyPk.GetCachedValue().(cryptotypes.PubKey)
// 	if !ok {
// 		k.Logger(ctx).Error("Error in interfacing out pub key")
// 		return voteTypes.Vote_VOTE_NO
// 	}

// 	signer := pubKey.Address()

// 	// check signer pubkey in message corresponds
// 	if !bytes.Equal(pubKey.Bytes(), eventLog.SignerPubkey) {
// 		k.Logger(ctx).Error(
// 			"Signer Pubkey does not match",
// 			"msgValidator", pubKey.String(),
// 			"mainchainValidator", hmTypes.BytesToHexBytes(eventLog.SignerPubkey),
// 		)

// 		return voteTypes.Vote_VOTE_NO
// 	}

// 	// check signer corresponding to pubkey matches signer from event
// 	if !bytes.Equal(signer.Bytes(), eventLog.Signer.Bytes()) {
// 		k.Logger(ctx).Error(
// 			"Signer Address from Pubkey does not match",
// 			"Validator", signer.String(),
// 			"mainchainValidator", eventLog.Signer.Hex(),
// 		)

// 		return voteTypes.Vote_VOTE_NO
// 	}

// 	// check msg id
// 	if eventLog.ValidatorId.Uint64() != msg.ValId {
// 		k.Logger(ctx).Error("ID in message doesn't match with id in log", "msgId", msg.ValId, "validatorIdFromTx", eventLog.ValidatorId)
// 		return voteTypes.Vote_VOTE_NO
// 	}

// 	// check ActivationEpoch
// 	if eventLog.ActivationEpoch.Uint64() != msg.ActivationEpoch {
// 		k.Logger(ctx).Error("ActivationEpoch in message doesn't match with ActivationEpoch in log", "msgActivationEpoch", msg.ActivationEpoch, "activationEpochFromTx", eventLog.ActivationEpoch.Uint64)
// 		return voteTypes.Vote_VOTE_NO
// 	}

// 	// check Amount
// 	if eventLog.Amount.Cmp(msg.Amount.BigInt()) != 0 {
// 		k.Logger(ctx).Error("Amount in message doesn't match Amount in event logs", "MsgAmount", msg.Amount, "AmountFromEvent", eventLog.Amount)
// 		return voteTypes.Vote_VOTE_NO
// 	}

// 	// check Blocknumber
// 	if receipt.BlockNumber.Uint64() != msg.BlockNumber {
// 		k.Logger(ctx).Error("BlockNumber in message doesn't match blocknumber in receipt", "MsgBlockNumber", msg.BlockNumber, "ReceiptBlockNumber", receipt.BlockNumber.Uint64)
// 		return voteTypes.Vote_VOTE_NO
// 	}

// 	// check nonce
// 	if eventLog.Nonce.Uint64() != msg.Nonce {
// 		k.Logger(ctx).Error("Nonce in message doesn't match with nonce in log", "msgNonce", msg.Nonce, "nonceFromTx", eventLog.Nonce)
// 		return voteTypes.Vote_VOTE_NO
// 	}

// 	k.Logger(ctx).Debug("✅ Successfully validated External call for validator join msg")

// 	return voteTypes.Vote_VOTE_YES
// }

// // SideHandleMsgStakeUpdate handles stake update message
// func (k *sideMsgServer) SideHandleMsgStakeUpdate(ctx sdk.Context, _msg sdk.Msg) (result voteTypes.Vote) {
// 	msg, ok := _msg.(*types.MsgStakeUpdate)
// 	if !ok {
// 		k.Logger(ctx).Error("msg type mismatched")
// 		return voteTypes.Vote_VOTE_NO
// 	}

// 	k.Logger(ctx).Debug("✅ Validating External call for stake update msg",
// 		"txHash", hmTypes.BytesToHeimdallHash(msg.TxHash.Bytes()),
// 		"logIndex", msg.LogIndex,
// 		"blockNumber", msg.BlockNumber,
// 	)

// 	contractCaller := k.IContractCaller

// 	// chainManager params
// 	params := k.chainKeeper.GetParams(ctx)
// 	chainParams := params.ChainParams

// 	// get main tx receipt
// 	receipt, err := contractCaller.GetConfirmedTxReceipt(msg.TxHash.EthHash(), params.MainchainTxConfirmations)
// 	if err != nil || receipt == nil {
// 		return voteTypes.Vote_VOTE_NO
// 	}

// 	eventLog, err := contractCaller.DecodeValidatorStakeUpdateEvent(chainParams.StakingInfoAddress.EthAddress(), receipt, msg.LogIndex)
// 	if err != nil || eventLog == nil {
// 		k.Logger(ctx).Error("Error fetching log from txhash")
// 		return voteTypes.Vote_VOTE_NO
// 	}

// 	if receipt.BlockNumber.Uint64() != msg.BlockNumber {
// 		k.Logger(ctx).Error("BlockNumber in message doesn't match blocknumber in receipt", "MsgBlockNumber", msg.BlockNumber, "ReceiptBlockNumber", receipt.BlockNumber.Uint64)
// 		return voteTypes.Vote_VOTE_NO
// 	}

// 	if eventLog.ValidatorId.Uint64() != msg.ValId {
// 		k.Logger(ctx).Error("ID in message doesn't match with id in log", "msgId", msg.ValId, "validatorIdFromTx", eventLog.ValidatorId)
// 		return voteTypes.Vote_VOTE_NO
// 	}

// 	// check Amount
// 	if eventLog.NewAmount.Cmp(msg.NewAmount.BigInt()) != 0 {
// 		k.Logger(ctx).Error("NewAmount in message doesn't match NewAmount in event logs", "MsgNewAmount", msg.NewAmount, "NewAmountFromEvent", eventLog.NewAmount)
// 		return voteTypes.Vote_VOTE_NO
// 	}

// 	// check nonce
// 	if eventLog.Nonce.Uint64() != msg.Nonce {
// 		k.Logger(ctx).Error("Nonce in message doesn't match with nonce in log", "msgNonce", msg.Nonce, "nonceFromTx", eventLog.Nonce)
// 		return voteTypes.Vote_VOTE_NO
// 	}

// 	k.Logger(ctx).Debug("✅ Successfully validated External call for stake update msg")

// 	return voteTypes.Vote_VOTE_YES
// }

// // SideHandleMsgSignerUpdate handles signer update message
// func (k *sideMsgServer) SideHandleMsgSignerUpdate(ctx sdk.Context, _msg sdk.Msg) (result voteTypes.Vote) {
// 	msg, ok := _msg.(*types.MsgSignerUpdate)
// 	if !ok {
// 		k.Logger(ctx).Error("msg type mismatched")
// 		return voteTypes.Vote_VOTE_NO
// 	}

// 	k.Logger(ctx).Debug("✅ Validating External call for signer update msg",
// 		"txHash", hmTypes.BytesToHeimdallHash(msg.TxHash.Bytes()),
// 		"logIndex", msg.LogIndex,
// 		"blockNumber", msg.BlockNumber,
// 	)

// 	contractCaller := k.IContractCaller

// 	// chainManager params
// 	params := k.chainKeeper.GetParams(ctx)
// 	chainParams := params.ChainParams

// 	// get main tx receipt
// 	receipt, err := contractCaller.GetConfirmedTxReceipt(msg.TxHash.EthHash(), params.MainchainTxConfirmations)
// 	if err != nil || receipt == nil {
// 		return voteTypes.Vote_VOTE_NO
// 	}

// 	// Generate PubKey from Pubkey in message and signer
// 	anyPk := msg.NewSignerPubKey
// 	newPubKey, ok := anyPk.GetCachedValue().(cryptotypes.PubKey)
// 	if !ok {
// 		k.Logger(ctx).Error("Error in interfacing out pub key")
// 		return voteTypes.Vote_VOTE_NO
// 	}

// 	newSigner := newPubKey.Address()

// 	eventLog, err := contractCaller.DecodeSignerUpdateEvent(chainParams.StakingInfoAddress.EthAddress(), receipt, msg.LogIndex)
// 	if err != nil || eventLog == nil {
// 		k.Logger(ctx).Error("Error fetching log from txhash")
// 		return voteTypes.Vote_VOTE_NO
// 	}

// 	if receipt.BlockNumber.Uint64() != msg.BlockNumber {
// 		k.Logger(ctx).Error("BlockNumber in message doesn't match blocknumber in receipt", "MsgBlockNumber", msg.BlockNumber, "ReceiptBlockNumber", receipt.BlockNumber.Uint64)
// 		return voteTypes.Vote_VOTE_NO
// 	}

// 	if eventLog.ValidatorId.Uint64() != msg.ValId {
// 		k.Logger(ctx).Error("ID in message doesn't match with id in log", "msgId", msg.ValId, "validatorIdFromTx", eventLog.ValidatorId)
// 		return voteTypes.Vote_VOTE_NO
// 	}

// 	if !bytes.Equal(eventLog.SignerPubkey, newPubKey.Bytes()[1:]) {
// 		k.Logger(ctx).Error("Newsigner pubkey in txhash and msg dont match", "msgPubKey", newPubKey.String(), "pubkeyTx", hmTypes.NewPubKey(eventLog.SignerPubkey[:]).String())
// 		return voteTypes.Vote_VOTE_NO
// 	}

// 	// check signer corresponding to pubkey matches signer from event
// 	if !bytes.Equal(newSigner.Bytes(), eventLog.NewSigner.Bytes()) {
// 		k.Logger(ctx).Error("Signer Address from Pubkey does not match", "Validator", newSigner.String(), "mainchainValidator", eventLog.NewSigner.Hex())
// 		return voteTypes.Vote_VOTE_NO
// 	}

// 	// check nonce
// 	if eventLog.Nonce.Uint64() != msg.Nonce {
// 		k.Logger(ctx).Error("Nonce in message doesn't match with nonce in log", "msgNonce", msg.Nonce, "nonceFromTx", eventLog.Nonce)
// 		return voteTypes.Vote_VOTE_NO
// 	}

// 	k.Logger(ctx).Debug("✅ Successfully validated External call for signer update msg")

// 	return voteTypes.Vote_VOTE_YES
// }

// // SideHandleMsgValidatorExit  handle  side msg validator exit
// func (k *sideMsgServer) SideHandleMsgValidatorExit(ctx sdk.Context, _msg sdk.Msg) (result voteTypes.Vote) {
// 	msg, ok := _msg.(*types.MsgValidatorExit)
// 	if !ok {
// 		k.Logger(ctx).Error("msg type mismatched")
// 		return voteTypes.Vote_VOTE_NO
// 	}

// 	k.Logger(ctx).Debug("✅ Validating External call for validator exit msg",
// 		"txHash", hmTypes.BytesToHeimdallHash(msg.TxHash.Bytes()),
// 		"logIndex", msg.LogIndex,
// 		"blockNumber", msg.BlockNumber,
// 	)

// 	contractCaller := k.IContractCaller

// 	// chainManager params
// 	params := k.chainKeeper.GetParams(ctx)
// 	chainParams := params.ChainParams

// 	// get main tx receipt
// 	receipt, err := contractCaller.GetConfirmedTxReceipt(msg.TxHash.EthHash(), params.MainchainTxConfirmations)
// 	if err != nil || receipt == nil {
// 		return voteTypes.Vote_VOTE_NO
// 	}

// 	// decode validator exit
// 	eventLog, err := contractCaller.DecodeValidatorExitEvent(chainParams.StakingInfoAddress.EthAddress(), receipt, msg.LogIndex)
// 	if err != nil || eventLog == nil {
// 		k.Logger(ctx).Error("Error fetching log from txhash")
// 		return voteTypes.Vote_VOTE_NO
// 	}

// 	if receipt.BlockNumber.Uint64() != msg.BlockNumber {
// 		k.Logger(ctx).Error("BlockNumber in message doesn't match blocknumber in receipt", "MsgBlockNumber", msg.BlockNumber, "ReceiptBlockNumber", receipt.BlockNumber.Uint64)
// 		return voteTypes.Vote_VOTE_NO
// 	}

// 	if eventLog.ValidatorId.Uint64() != msg.ValId {
// 		k.Logger(ctx).Error("ID in message doesn't match with id in log", "msgId", msg.ValId, "validatorIdFromTx", eventLog.ValidatorId)
// 		return voteTypes.Vote_VOTE_NO
// 	}

// 	if eventLog.DeactivationEpoch.Uint64() != msg.DeactivationEpoch {
// 		k.Logger(ctx).Error("DeactivationEpoch in message doesn't match with deactivationEpoch in log", "msgDeactivationEpoch", msg.DeactivationEpoch, "deactivationEpochFromTx", eventLog.DeactivationEpoch.Uint64)
// 		return voteTypes.Vote_VOTE_NO
// 	}

// 	// check nonce
// 	if eventLog.Nonce.Uint64() != msg.Nonce {
// 		k.Logger(ctx).Error("Nonce in message doesn't match with nonce in log", "msgNonce", msg.Nonce, "nonceFromTx", eventLog.Nonce)
// 		return voteTypes.Vote_VOTE_NO
// 	}

// 	k.Logger(ctx).Debug("✅ Successfully validated External call for validator exit msg")

// 	return voteTypes.Vote_VOTE_YES
// }

// /*
// 	Post Handlers - update the state of the tx
// **/

// // PostHandleMsgValidatorJoin msg validator join
// func (k *sideMsgServer) PostHandleMsgValidatorJoin(ctx sdk.Context, _msg sdk.Msg, sideTxResult voteTypes.Vote) {
// 	msg, ok := _msg.(*types.MsgValidatorJoin)
// 	if !ok {
// 		k.Logger(ctx).Error("msg type mismatched")
// 		return
// 	}

// 	// Skip handler if validator join is not approved
// 	if sideTxResult != voteTypes.Vote_VOTE_YES {
// 		k.Logger(ctx).Debug("Skipping new validator-join since side-tx didn't get yes votes")
// 		return
// 	}

// 	// Check for replay attack
// 	blockNumber := new(big.Int).SetUint64(msg.BlockNumber)
// 	sequence := new(big.Int).Mul(blockNumber, big.NewInt(hmTypes.DefaultLogIndexUnit))
// 	sequence.Add(sequence, new(big.Int).SetUint64(msg.LogIndex))

// 	// check if incoming tx is older
// 	if k.HasStakingSequence(ctx, sequence.String()) {
// 		k.Logger(ctx).Error("Older invalid tx found")
// 		return
// 	}

// 	k.Logger(ctx).Debug("Adding validator to state", "sideTxResult", sideTxResult)

// 	// Generate PubKey from Pubkey in message and signer
// 	anyPk := msg.SignerPubKey
// 	pubKey, ok := anyPk.GetCachedValue().(cryptotypes.PubKey)
// 	if !ok {
// 		k.Logger(ctx).Error("Error in interfacing out pub key")
// 		return
// 	}

// 	signer := pubKey.Address().String()

// 	// get voting power from amount
// 	votingPower, err := helper.GetPowerFromAmount(msg.Amount.BigInt())
// 	if err != nil {
// 		k.Logger(ctx).Error(fmt.Sprintf("Invalid amount %v for validator %v", msg.Amount, msg.ValId))
// 		return
// 	}

// 	// create new validator
// 	newValidator := hmTypes.Validator{
// 		ValId:       msg.ValId,
// 		StartEpoch:  msg.ActivationEpoch,
// 		EndEpoch:    0,
// 		Nonce:       msg.Nonce,
// 		VotingPower: votingPower.Int64(),
// 		PubKey:      anyPk,
// 		Signer:      strings.ToLower(signer),
// 		LastUpdated: "",
// 	}

// 	// update last updated
// 	newValidator.LastUpdated = sequence.String()

// 	// add validator to store
// 	k.Logger(ctx).Debug("Adding new validator to state", "validator", newValidator.String())

// 	if err = k.AddValidator(ctx, newValidator); err != nil {
// 		k.Logger(ctx).Error("Unable to add validator to state", "validator", newValidator.String(), "error", err)
// 		return
// 	}

// 	// Add Validator signing info. It is required for slashing module
// 	k.Logger(ctx).Debug("Adding signing info for new validator")

// 	//TODO H2 PLease check whether we need the following code or not
// 	//as this code belongs to slashing
// 	// valSigningInfo := hmTypes.NewValidatorSigningInfo(newValidator.ID, ctx.BlockHeight(), int64(0), int64(0))
// 	// if err = k.AddValidatorSigningInfo(ctx, newValidator.ID, valSigningInfo); err != nil {
// 	// 	k.Logger(ctx).Error("Unable to add validator signing info to state", "valSigningInfo", valSigningInfo.String(), "error", err)
// 	// 	return hmCommon.ErrValidatorSigningInfoSave(k.Codespace()).Result()
// 	// }

// 	// save staking sequence
// 	k.SetStakingSequence(ctx, sequence.String())
// 	k.Logger(ctx).Debug("✅ New validator successfully joined", "validator", strconv.FormatUint(newValidator.ValId, 10))

// 	// TX bytes
// 	txBytes := ctx.TxBytes()
// 	hash := hmTypes.TxHash{txBytes}.Bytes()

// 	ctx.EventManager().EmitEvents(sdk.Events{
// 		sdk.NewEvent(
// 			types.EventTypeValidatorJoin,
// 			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),                // module name
// 			sdk.NewAttribute(hmTypes.AttributeKeyTxHash, hmTypes.BytesToHeimdallHash(hash).Hex()), // tx hash
// 			sdk.NewAttribute(hmTypes.AttributeKeyTxLogIndex, strconv.FormatUint(msg.LogIndex, 10)),
// 			sdk.NewAttribute(hmTypes.AttributeKeySideTxResult, sideTxResult.String()), // result
// 			sdk.NewAttribute(types.AttributeKeyValidatorID, strconv.FormatUint(newValidator.ValId, 10)),
// 			sdk.NewAttribute(types.AttributeKeySigner, newValidator.Signer),
// 			sdk.NewAttribute(types.AttributeKeyValidatorNonce, strconv.FormatUint(msg.Nonce, 10)),
// 		),
// 	})

// 	return
// }

// // PostHandleMsgStakeUpdate handles stake update message
// func (k *sideMsgServer) PostHandleMsgStakeUpdate(ctx sdk.Context, _msg sdk.Msg, sideTxResult voteTypes.Vote) {
// 	msg, ok := _msg.(*types.MsgStakeUpdate)
// 	if !ok {
// 		k.Logger(ctx).Error("msg type mismatched")
// 		return
// 	}

// 	// Skip handler if stakeUpdate is not approved
// 	if sideTxResult != voteTypes.Vote_VOTE_YES {
// 		k.Logger(ctx).Debug("Skipping stake update since side-tx didn't get yes votes")
// 		return
// 	}

// 	// Check for replay attack
// 	blockNumber := new(big.Int).SetUint64(msg.BlockNumber)
// 	sequence := new(big.Int).Mul(blockNumber, big.NewInt(hmTypes.DefaultLogIndexUnit))
// 	sequence.Add(sequence, new(big.Int).SetUint64(msg.LogIndex))

// 	// check if incoming tx is older
// 	if k.HasStakingSequence(ctx, sequence.String()) {
// 		k.Logger(ctx).Error("Older invalid tx found")
// 		return
// 	}

// 	k.Logger(ctx).Debug("Updating validator stake", "sideTxResult", sideTxResult)

// 	// pull validator from store
// 	validator, ok := k.GetValidatorFromValID(ctx, msg.ValId)
// 	if !ok {
// 		k.Logger(ctx).Error("Fetching of validator from store failed", "validatorId", msg.ValId)
// 		return
// 	}

// 	// update last updated
// 	validator.LastUpdated = sequence.String()

// 	// update nonce
// 	validator.Nonce = msg.Nonce

// 	// set validator amount
// 	p, err := helper.GetPowerFromAmount(msg.NewAmount.BigInt())
// 	if err != nil {
// 		return
// 	}

// 	validator.VotingPower = p.Int64()

// 	// save validator
// 	err = k.AddValidator(ctx, validator)
// 	if err != nil {
// 		k.Logger(ctx).Error("Unable to update signer", "ValidatorID", validator.ValId, "error", err)
// 		return
// 	}

// 	// save staking sequence
// 	k.SetStakingSequence(ctx, sequence.String())

// 	// TX bytes
// 	txBytes := ctx.TxBytes()
// 	hash := hmTypes.TxHash{txBytes}.Bytes()

// 	ctx.EventManager().EmitEvents(sdk.Events{
// 		sdk.NewEvent(
// 			types.EventTypeStakeUpdate,
// 			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
// 			sdk.NewAttribute(hmTypes.AttributeKeyTxHash, hmTypes.BytesToHeimdallHash(hash).Hex()), // tx hash
// 			sdk.NewAttribute(hmTypes.AttributeKeySideTxResult, sideTxResult.String()),             // result
// 			sdk.NewAttribute(types.AttributeKeyValidatorID, strconv.FormatUint(validator.ValId, 10)),
// 			sdk.NewAttribute(types.AttributeKeyValidatorNonce, strconv.FormatUint(msg.Nonce, 10)),
// 		),
// 	})

// 	return
// }

// // PostHandleMsgSignerUpdate handles signer update message
// func (k *sideMsgServer) PostHandleMsgSignerUpdate(ctx sdk.Context, _msg sdk.Msg, sideTxResult voteTypes.Vote) {
// 	msg, ok := _msg.(*types.MsgSignerUpdate)
// 	if !ok {
// 		k.Logger(ctx).Error("msg type mismatched")
// 		return
// 	}

// 	// Skip handler if signer update is not approved
// 	if sideTxResult != voteTypes.Vote_VOTE_YES {
// 		k.Logger(ctx).Debug("Skipping signer update since side-tx didn't get yes votes")
// 		return
// 	}

// 	// Check for replay attack
// 	blockNumber := new(big.Int).SetUint64(msg.BlockNumber)
// 	sequence := new(big.Int).Mul(blockNumber, big.NewInt(hmTypes.DefaultLogIndexUnit))
// 	sequence.Add(sequence, new(big.Int).SetUint64(msg.LogIndex))
// 	// check if incoming tx is older
// 	if k.HasStakingSequence(ctx, sequence.String()) {
// 		k.Logger(ctx).Error("Older invalid tx found")
// 		return
// 	}

// 	k.Logger(ctx).Debug("Persisting signer update", "sideTxResult", sideTxResult)

// 	// Generate PubKey from Pubkey in message and signer
// 	anyPk := msg.NewSignerPubKey
// 	newPubKey, ok := anyPk.GetCachedValue().(cryptotypes.PubKey)
// 	if !ok {
// 		k.Logger(ctx).Error("Error in interfacing out pub key")
// 		return
// 	}

// 	newSigner := strings.ToLower(newPubKey.Address().String())

// 	// pull validator from store
// 	validator, ok := k.GetValidatorFromValID(ctx, msg.ValId)
// 	if !ok {
// 		k.Logger(ctx).Error("Fetching of validator from store failed", "validatorId", msg.ValId)
// 		return
// 	}

// 	oldValidator := validator.Copy()

// 	// update last updated
// 	validator.LastUpdated = sequence.String()

// 	// update nonce
// 	validator.Nonce = msg.Nonce

// 	// check if we are actually updating signer
// 	if !(newSigner == validator.Signer) {
// 		// Update signer in prev Validator
// 		validator.Signer = newSigner
// 		validator.PubKey = anyPk

// 		k.Logger(ctx).Debug("Updating new signer", "newSigner", newSigner, "oldSigner", oldValidator.Signer, "validatorID", msg.ValId)
// 	} else {
// 		k.Logger(ctx).Error("No signer change", "newSigner", newSigner, "oldSigner", oldValidator.Signer, "validatorID", msg.ValId)
// 		return
// 	}

// 	k.Logger(ctx).Debug("Removing old validator", "validator", oldValidator.String())

// 	// remove old validator from HM
// 	oldValidator.EndEpoch = k.moduleCommunicator.GetACKCount(ctx)

// 	// remove old validator from TM
// 	oldValidator.VotingPower = 0
// 	// updated last
// 	oldValidator.LastUpdated = sequence.String()

// 	// updated nonce
// 	oldValidator.Nonce = msg.Nonce

// 	// save old validator
// 	if err := k.AddValidator(ctx, *oldValidator); err != nil {
// 		k.Logger(ctx).Error("Unable to update signer", "validatorId", validator.ValId, "error", err)
// 		return
// 	}

// 	// adding new validator
// 	k.Logger(ctx).Debug("Adding new validator", "validator", validator.String())

// 	// save validator
// 	err := k.AddValidator(ctx, validator)
// 	if err != nil {
// 		k.Logger(ctx).Error("Unable to update signer", "ValidatorID", validator.ValId, "error", err)
// 		return
// 	}

// 	// save staking sequence
// 	k.SetStakingSequence(ctx, sequence.String())

// 	// TX bytes
// 	txBytes := ctx.TxBytes()
// 	hash := hmTypes.TxHash{txBytes}.Bytes()

// 	//
// 	// Move heimdall fee to new signer
// 	//

// 	//TODO H2 Please check this code once module communicatator is defined properlu
// 	// // check if fee is already withdrawn
// 	// coins := k.moduleCommunicator.GetCoins(ctx, oldValidator.Signer)

// 	// maticBalance := coins.AmountOf(authTypes.FeeToken)
// 	// if !maticBalance.IsZero() {
// 	// 	k.Logger(ctx).Info("Transferring fee", "from", oldValidator.Signer.String(), "to", validator.Signer.String(), "balance", maticBalance.String())

// 	// 	maticCoins := sdk.Coins{sdk.Coin{Denom: authTypes.FeeToken, Amount: maticBalance}}
// 	// 	if err := k.moduleCommunicator.SendCoins(ctx, oldValidator.Signer, validator.Signer, maticCoins); err != nil {
// 	// 		k.Logger(ctx).Info("Error while transferring fee", "from", oldValidator.Signer.String(), "to", validator.Signer.String(), "balance", maticBalance.String())
// 	// 		return err.Result()
// 	// 	}
// 	// }

// 	ctx.EventManager().EmitEvents(sdk.Events{
// 		sdk.NewEvent(
// 			types.EventTypeSignerUpdate,
// 			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),                // module name
// 			sdk.NewAttribute(hmTypes.AttributeKeyTxHash, hmTypes.BytesToHeimdallHash(hash).Hex()), // tx hash
// 			sdk.NewAttribute(hmTypes.AttributeKeySideTxResult, sideTxResult.String()),             // result
// 			sdk.NewAttribute(types.AttributeKeyValidatorID, strconv.FormatUint(validator.ValId, 10)),
// 			sdk.NewAttribute(types.AttributeKeyValidatorNonce, strconv.FormatUint(msg.Nonce, 10)),
// 		),
// 	})

// 	return
// }

// // PostHandleMsgValidatorExit handle msg validator exit
// func (k *sideMsgServer) PostHandleMsgValidatorExit(ctx sdk.Context, _msg sdk.Msg, sideTxResult voteTypes.Vote) {
// 	msg, ok := _msg.(*types.MsgValidatorExit)
// 	if !ok {
// 		k.Logger(ctx).Error("msg type mismatched")
// 		return
// 	}

// 	// Skip handler if validator exit is not approved
// 	if sideTxResult != voteTypes.Vote_VOTE_YES {
// 		k.Logger(ctx).Debug("Skipping validator exit since side-tx didn't get yes votes")
// 		return
// 	}

// 	// Check for replay attack
// 	blockNumber := new(big.Int).SetUint64(msg.BlockNumber)
// 	sequence := new(big.Int).Mul(blockNumber, big.NewInt(hmTypes.DefaultLogIndexUnit))
// 	sequence.Add(sequence, new(big.Int).SetUint64(msg.LogIndex))

// 	// check if incoming tx is older
// 	if k.HasStakingSequence(ctx, sequence.String()) {
// 		k.Logger(ctx).Error("Older invalid tx found")
// 		return
// 	}

// 	k.Logger(ctx).Debug("Persisting validator exit", "sideTxResult", sideTxResult)

// 	validator, ok := k.GetValidatorFromValID(ctx, msg.ValId)
// 	if !ok {
// 		k.Logger(ctx).Error("Fetching of validator from store failed", "validatorID", msg.ValId)
// 		return
// 	}

// 	// set end epoch
// 	validator.EndEpoch = msg.DeactivationEpoch

// 	// update last updated
// 	validator.LastUpdated = sequence.String()

// 	// update nonce
// 	validator.Nonce = msg.Nonce

// 	// Add deactivation time for validator
// 	if err := k.AddValidator(ctx, validator); err != nil {
// 		k.Logger(ctx).Error("Error while setting deactivation epoch to validator", "error", err, "validatorID", validator.ValId)
// 		return
// 	}

// 	// save staking sequence
// 	k.SetStakingSequence(ctx, sequence.String())

// 	// TX bytes
// 	txBytes := ctx.TxBytes()
// 	hash := hmTypes.TxHash{txBytes}.Bytes()

// 	ctx.EventManager().EmitEvents(sdk.Events{
// 		sdk.NewEvent(
// 			types.EventTypeValidatorExit,
// 			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),                // module name
// 			sdk.NewAttribute(hmTypes.AttributeKeyTxHash, hmTypes.BytesToHeimdallHash(hash).Hex()), // tx hash
// 			sdk.NewAttribute(hmTypes.AttributeKeySideTxResult, sideTxResult.String()),             // result
// 			sdk.NewAttribute(types.AttributeKeyValidatorID, strconv.FormatUint(validator.ValId, 10)),
// 			sdk.NewAttribute(types.AttributeKeyValidatorNonce, strconv.FormatUint(msg.Nonce, 10)),
// 		),
// 	})

// 	return
// }
