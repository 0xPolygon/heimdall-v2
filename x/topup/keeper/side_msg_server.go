package keeper

import (
	"bytes"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authTypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	hModule "github.com/0xPolygon/heimdall-v2/module"
	hTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/topup/types"
)

var (
	topupMsgTypeURL = sdk.MsgTypeURL(&types.MsgTopupTx{})
)

type sideMsgServer struct {
	k *Keeper
}

// NewSideMsgServerImpl returns an implementation of the x/topup SideMsgServer interface for the provided Keeper.
func NewSideMsgServerImpl(keeper *Keeper) types.SideMsgServer {
	return &sideMsgServer{k: keeper}
}

// SideTxHandler redirects to the right sideMsgServer side_handler based on methodName
func (s sideMsgServer) SideTxHandler(methodName string) hModule.SideTxHandler {
	switch methodName {
	case topupMsgTypeURL:
		return s.SideHandleTopupTx
	default:
		return nil
	}
}

// PostTxHandler redirects to the right sideMsgServer post_handler based on methodName
func (s sideMsgServer) PostTxHandler(methodName string) hModule.PostTxHandler {
	switch methodName {
	case topupMsgTypeURL:
		return s.PostHandleTopupTx
	default:
		return nil
	}
}

// SideHandleTopupTx handles the side tx for a validator's topup
func (s sideMsgServer) SideHandleTopupTx(ctx sdk.Context, msgI sdk.Msg) hModule.Vote {
	logger := s.k.Logger(ctx)

	msg, ok := msgI.(*types.MsgTopupTx)
	if !ok {
		logger.Error("MsgTopupTx type mismatch")
		return hModule.Vote_VOTE_NO
	}

	logger.Debug("validating external call for topup msg",
		"txHash", msg.TxHash.GetHash(),
		"logIndex", msg.LogIndex,
		"blockNumber", msg.BlockNumber,
	)

	// get chain params
	params := s.k.chainKeeper.GetParams(ctx)
	chainParams := params.ChainParams

	// get main tx receipt
	receipt, err := contractCaller.GetConfirmedTxReceipt(msg.TxHash.EthHash(), params.MainchainTxConfirmations)
	if err != nil || receipt == nil {
		return hModule.Vote_VOTE_NO
	}

	// get event log for topup
	eventLog, err := contractCaller.DecodeValidatorTopupFeesEvent(chainParams.StakingInfoAddress.EthAddress(), receipt, msg.LogIndex)
	if err != nil || eventLog == nil {
		logger.Error("error fetching log from txhash for DecodeValidatorTopupFeesEvent")
		return hModule.Vote_VOTE_NO
	}

	if receipt.BlockNumber.Uint64() != msg.BlockNumber {
		logger.Error("blockNumber in message doesn't match blockNumber in receipt", "msgBlockNumber", msg.BlockNumber, "receiptBlockNumber", receipt.BlockNumber.Uint64)
		return hModule.Vote_VOTE_NO
	}

	if !bytes.Equal(eventLog.User.Bytes(), []byte(msg.User)) {
		logger.Error(
			"user address from contract event log does not match with user from topup message",
			"eventUser", eventLog.User.String(),
			"msgUser", msg.User,
		)

		return hModule.Vote_VOTE_NO
	}

	if eventLog.Fee.Cmp(msg.Fee.BigInt()) != 0 {
		logger.Error("fee in message doesn't match fee in event logs", "msgFee", msg.Fee, "eventFee", eventLog.Fee)
		return hModule.Vote_VOTE_NO
	}

	logger.Debug("Successfully validated external call for topup msg")

	return hModule.Vote_VOTE_NO
}

// PostHandleTopupTx handles the post side tx for a validator's topup
func (s sideMsgServer) PostHandleTopupTx(ctx sdk.Context, msgI sdk.Msg, sideTxResult hModule.Vote) {
	logger := s.k.Logger(ctx)

	msg, ok := msgI.(*types.MsgTopupTx)
	if !ok {
		logger.Error("MsgTopupTx type mismatch")
		return
	}

	// skip handler if topup is not approved
	if sideTxResult != hModule.Vote_VOTE_YES {
		logger.Debug("skipping new topup tx since side-tx didn't get yes votes")
		return
	}

	// check if incoming tx is older
	blockNumber := new(big.Int).SetUint64(msg.BlockNumber)
	sequence := new(big.Int).Mul(blockNumber, big.NewInt(types.DefaultLogIndexUnit))
	sequence.Add(sequence, new(big.Int).SetUint64(msg.LogIndex))

	exists, err := s.k.HasTopupSequence(ctx, sequence.String())
	if err != nil {
		logger.Error("error while fetching older topup sequence", "error", err)
		return
	}
	if exists {
		logger.Error("older tx found")
		return
	}

	logger.Debug("persisting topup state", "sideTxResult", sideTxResult)

	// create topup event
	user := msg.User
	topupAmount := sdk.Coins{sdk.Coin{Denom: authTypes.FeeToken, Amount: msg.Fee}}

	// TODO HV2: is the following a proper replacement for AddCoins? Transfer from module to user, then from user to proposer

	err = s.k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, sdk.AccAddress(user), topupAmount)
	if err != nil {
		logger.Error("error while adding coins to user", "user", user, "topupAmount", topupAmount, "error", err)
		return
	}

	err = s.k.bankKeeper.SendCoins(ctx, sdk.AccAddress(user), sdk.AccAddress(msg.Proposer), ante.DefaultFeeWantedPerTx)
	if err != nil {
		logger.Error("error while sending coins from user to proposer", "user", user, "proposer", msg.Proposer, "topupAmount", topupAmount, "error", err)
		return
	}

	logger.Debug("persisted topup state for", "user", user, "topupAmount", topupAmount.String())

	// save topup
	err = s.k.SetTopupSequence(ctx, sequence.String())
	if err != nil {
		logger.Error("error while saving topup sequence", "sequence", sequence.String(), "error", err)
		return
	}

	txBytes := ctx.TxBytes()
	hash := hTypes.TxHash{Hash: txBytes}.Bytes()

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeTopup,
			sdk.NewAttribute(sdk.AttributeKeyAction, msg.Type()),
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(types.AttributeKeyTxHash, hTypes.BytesToHeimdallHash(hash).Hex()),
			sdk.NewAttribute(types.AttributeKeySideTxResult, sideTxResult.String()),
			sdk.NewAttribute(types.AttributeKeySender, msg.Proposer),
			sdk.NewAttribute(types.AttributeKeyRecipient, msg.User),
			sdk.NewAttribute(types.AttributeKeyTopupAmount, msg.Fee.String()),
		),
	})

	return
}
