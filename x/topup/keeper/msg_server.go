package keeper

import (
	"context"
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

// CreateTopupTx handles topup tx events
func (m msgServer) CreateTopupTx(ctx context.Context, msg *types.MsgTopupTx) (*types.MsgTopupTxResponse, error) {

	// TODO HV2: enable when this is merged and remove the nil byte slice declaration
	// txHash := hTypes.BytesToHeimdallHash(msg.TxHash.Hash)
	var txHash []byte

	m.k.Logger(ctx).Debug("CreateTopupTx msg received",
		"proposer", msg.Proposer,
		"user", msg.User,
		"fee", msg.Fee.String(),
		"txHash", txHash,
		"logIndex", msg.LogIndex,
		"blockNumber", msg.BlockNumber,
	)

	// check if send is enabled for default denom
	if !m.k.BankKeeper.IsSendEnabledDenom(ctx, types.DefaultDenom) {
		m.k.Logger(ctx).Error("send not enabled")
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
		return nil, err
	}
	if exists {
		m.k.Logger(ctx).Error("older tx found")
		return nil, errors.Wrapf(sdkerrors.ErrInvalidRequest,
			"tx with hash %s already exists", txHash)
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

	m.k.Logger(ctx).Debug("event created for CreateTopupTx")

	return &types.MsgTopupTxResponse{}, nil
}

// WithdrawFeeTx handles withdraw fee tx events
func (m msgServer) WithdrawFeeTx(ctx context.Context, msg *types.MsgWithdrawFeeTx) (*types.MsgWithdrawFeeTxResponse, error) {
	m.k.Logger(ctx).Debug("WithdrawFeeTx msg received",
		"proposer", msg.Proposer,
		"amount", msg.Amount.String(),
	)

	// partial withdraw
	amount := msg.Amount

	// full withdraw
	if msg.Amount.String() == big.NewInt(0).String() {
		coins := m.k.BankKeeper.GetBalance(ctx, sdk.AccAddress(msg.Proposer), types.DefaultDenom)
		amount = coins.Amount
	}

	m.k.Logger(ctx).Debug("fee amount", "fromAddress", msg.Proposer, "balance", amount.BigInt().String())

	// check if there is no balance to withdraw
	if amount.IsZero() {
		m.k.Logger(ctx).Error("no balance to withdraw")
		return nil, errors.Wrapf(sdkerrors.ErrInsufficientFunds,
			"account %s has no balance", msg.Proposer)
	}

	// create coins object
	coins := sdk.Coins{sdk.Coin{Denom: types.DefaultDenom, Amount: amount}}

	// send coins from account to module
	err := m.k.BankKeeper.SendCoinsFromAccountToModule(ctx, sdk.AccAddress(msg.Proposer), types.ModuleName, coins)
	if err != nil {
		m.k.Logger(ctx).Error("error while sending coins from account to module",
			"fromAddress", msg.Proposer,
			"module", types.ModuleName,
			"err", err)
		return nil, err

	}
	// burn coins from module
	err = m.k.BankKeeper.BurnCoins(ctx, types.ModuleName, coins)
	if err != nil {
		m.k.Logger(ctx).Error("error while burning coins",
			"module", types.ModuleName,
			"coinsAmount", coins.String(),
			"err", err)
		return nil, err

	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// add Fee to dividendAccount
	feeAmount := amount.BigInt()
	if err := m.k.AddFeeToDividendAccount(sdkCtx, msg.Proposer, feeAmount); err != nil {
		m.k.Logger(ctx).Error("error while adding fee to dividend account",
			"fromAddress", msg.Proposer,
			"feeAmount", feeAmount,
			"err", err)
		return nil, err
	}

	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeFeeWithdraw,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(types.AttributeKeyUser, msg.Proposer),
			sdk.NewAttribute(types.AttributeKeyFeeWithdrawAmount, feeAmount.String()),
		),
	})

	m.k.Logger(ctx).Debug("event created for WithdrawFeeTx")

	return &types.MsgWithdrawFeeTxResponse{}, nil
}
