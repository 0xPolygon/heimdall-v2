package keeper

import (
	"context"
	"github.com/ethereum/go-ethereum/common"
	"math/big"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/0xPolygon/heimdall-v2/x/topup/types"
)

type msgServer struct {
	k *Keeper
}

// NewMsgServerImpl returns an implementation of the x/topup MsgServer interface for the provided Keeper.
func NewMsgServerImpl(keeper *Keeper) types.MsgServer {
	return &msgServer{k: keeper}
}

// CreateTopupTx handles the creation of topup tx events for the x/topup module
func (m msgServer) CreateTopupTx(ctx context.Context, msg *types.MsgTopupTx) (*types.MsgTopupTxResponse, error) {
	logger := m.k.Logger(ctx)

	// TODO HV2: replace common.BytesToHash with hmTypes.BytesToHeimdallHash when implemented?
	txHash := common.BytesToHash(msg.TxHash.Hash)

	logger.Debug("CreateTopupTx msg received",
		"proposer", msg.Proposer,
		"user", msg.User,
		"fee", msg.Fee.String(),
		"txHash", txHash,
		"logIndex", msg.LogIndex,
		"blockNumber", msg.BlockNumber,
	)

	/* TODO HV2: is this the proper cosmos-sdk replacement for what's being done in heimdall-v1?
	   There, the BankKeeper.GetSendEnabled method is used but it's no longer available in cosmos-sdk.
	   So the approach here is to invoke BankKeeper.IsSendEnabledDenom on matic denom
	   I believe this is correct.
	*/

	// check if send is enabled for default denom
	if !m.k.BankKeeper.IsSendEnabledDenom(ctx, types.DefaultDenom) {
		logger.Error("send not enabled")
		return nil, errors.Wrapf(sdkerrors.ErrInvalidRequest,
			"send for denom %s is not enabled in bank keeper", types.DefaultDenom)
	}

	// calculate sequence
	blockNumber := new(big.Int).SetUint64(msg.BlockNumber)
	sequence := new(big.Int).Mul(blockNumber, big.NewInt(types.DefaultLogIndexUnit))
	sequence.Add(sequence, new(big.Int).SetUint64(msg.LogIndex))

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// check if incoming tx already exists
	exists, err := m.k.HasTopupSequence(sdkCtx, sequence.String())
	if err != nil {
		return nil, errors.Wrapf(sdkerrors.ErrLogic, err.Error())
	}
	if exists {
		logger.Error("older tx found")
		return nil, errors.Wrapf(sdkerrors.ErrInvalidRequest,
			"tx with hash %s already exists", txHash.String())
	}

	// emit event if tx is valid, then return
	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeTopup,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(types.AttributeKeySender, msg.Proposer),
			sdk.NewAttribute(types.AttributeKeyRecipient, msg.User),
			sdk.NewAttribute(types.AttributeKeyTopupAmount, msg.Fee.String()),
		),
	})

	logger.Debug("event created for CreateTopupTx")

	return &types.MsgTopupTxResponse{}, nil
}

// WithdrawFeeTx handles withdraw fee tx events for the x/topup module
func (m msgServer) WithdrawFeeTx(ctx context.Context, msg *types.MsgWithdrawFeeTx) (*types.MsgWithdrawFeeTxResponse, error) {
	logger := m.k.Logger(ctx)

	logger.Debug("WithdrawFeeTx msg received",
		"proposer", msg.Proposer,
		"amount", msg.Amount.String(),
	)

	// partial withdraw
	amount := msg.Amount

	/* TODO HV2: is this the proper cosmos-sdk replacement for what's being done in heimdall-v1?
	   There, the BankKeeper.GetCoins method is used to check the balance given an address.
	   This method is no longer available in cosmos-sdk. So the approach here is to invoke BankKeeper.GetBalance.
	   I believe this is correct.
	*/

	// full withdraw
	if msg.Amount.IsZero() {
		coins := m.k.BankKeeper.GetBalance(ctx, sdk.AccAddress(msg.Proposer), types.DefaultDenom)
		amount = coins.Amount
	}

	logger.Debug("fee amount", "fromAddress", msg.Proposer, "balance", amount.BigInt().String())

	// check if there is no balance to withdraw
	if amount.IsZero() {
		logger.Error("no balance to withdraw")
		return nil, errors.Wrapf(sdkerrors.ErrInsufficientFunds,
			"account %s has no balance", msg.Proposer)
	}

	// create coins object
	coins := sdk.Coins{sdk.Coin{Denom: types.DefaultDenom, Amount: amount}}

	/* TODO HV2: is this the proper cosmos-sdk replacement for what's being done in heimdall-v1?
	   There, the BankKeeper.SubtractCoins method is used to withdraw coins from the validator.
	   This method is no longer available in cosmos-sdk.
	   So the approach here is to invoke BankKeeper.SendCoinsFromAccountToModule + BankKeeper.BurnCoins
	   Not sure if this is the correct approach.
	*/

	// send coins from account to module
	err := m.k.BankKeeper.SendCoinsFromAccountToModule(ctx, sdk.AccAddress(msg.Proposer), types.ModuleName, coins)
	if err != nil {
		logger.Error("error while sending coins from account to module",
			"fromAddress", msg.Proposer,
			"module", types.ModuleName,
			"err", err)
		return nil, errors.Wrapf(sdkerrors.ErrLogic, err.Error())
	}
	// burn coins from module
	err = m.k.BankKeeper.BurnCoins(ctx, types.ModuleName, coins)
	if err != nil {
		logger.Error("error while burning coins",
			"module", types.ModuleName,
			"coinsAmount", coins.String(),
			"err", err)
		return nil, errors.Wrapf(sdkerrors.ErrLogic, err.Error())
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// add Fee to dividendAccount
	feeAmount := amount.BigInt()
	if err := m.k.AddFeeToDividendAccount(sdkCtx, msg.Proposer, feeAmount); err != nil {
		logger.Error("error while adding fee to dividend account",
			"fromAddress", msg.Proposer,
			"feeAmount", feeAmount,
			"err", err)
		return nil, errors.Wrapf(sdkerrors.ErrLogic, err.Error())
	}

	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeFeeWithdraw,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(types.AttributeKeyUser, msg.Proposer),
			sdk.NewAttribute(types.AttributeKeyFeeWithdrawAmount, feeAmount.String()),
		),
	})

	logger.Debug("event created for WithdrawFeeTx")

	return &types.MsgWithdrawFeeTxResponse{}, nil
}
