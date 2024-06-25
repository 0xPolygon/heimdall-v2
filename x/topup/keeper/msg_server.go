package keeper

import (
	"context"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
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

// HandleTopupTx handles the topup tx events for the x/topup module
func (m msgServer) HandleTopupTx(ctx context.Context, msg *types.MsgTopupTx) (*types.MsgTopupTxResponse, error) {
	logger := m.k.Logger(ctx)

	// TODO HV2: replace common.BytesToHash with hmTypes.BytesToHeimdallHash when implemented?
	txHash := common.BytesToHash(msg.TxHash.Hash)

	logger.Debug("HandleTopupTx msg received",
		"proposer", msg.Proposer,
		"user", msg.User,
		"fee", msg.Fee.String(),
		"txHash", txHash,
		"logIndex", msg.LogIndex,
		"blockNumber", msg.BlockNumber,
	)

	/* HV2: v1's BankKeeper.GetSendEnabled is no longer available in cosmos-sdk.
	   We use BankKeeper.IsSendEnabledDenom instead
	*/

	// check if send is enabled for default denom
	if !m.k.BankKeeper.IsSendEnabledDenom(ctx, sdk.DefaultBondDenom) {
		logger.Error("send not enabled")
		return nil, errors.Wrapf(sdkerrors.ErrInvalidRequest,
			"send for denom %s is not enabled in bank keeper", sdk.DefaultBondDenom)
	}

	// check feasibility of topup tx based on msg fee
	if msg.Fee.LT(ante.DefaultFeeWantedPerTx[0].Amount) {
		logger.Error("default fee exceeds amount to topup", "user", msg.User,
			"amount", msg.Fee, "defaultFeeWantedPerTx", ante.DefaultFeeWantedPerTx[0])
		return nil, errors.Wrapf(sdkerrors.ErrInsufficientFunds, "default fee exceeds amount to topup")
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
		logger.Error("older tx found",
			"sequence", sequence.String(),
			"logIndex", msg.LogIndex,
			"blockNumber", msg.BlockNumber,
			"txHash", txHash)
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

	logger.Debug("event created for HandleTopupTx")

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

	/* HV2: v1's BankKeeper.GetCoins is no longer available in cosmos-sdk.
	Hence, we could use BankKeeper.GetBalance,
	but - just to be compatible with vesting, if ever enabled - we use BankKeeper.SpendableCoin.
	*/

	// full withdraw
	if msg.Amount.IsZero() {
		coins := m.k.BankKeeper.SpendableCoin(ctx, sdk.AccAddress(msg.Proposer), sdk.DefaultBondDenom)
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
	coins := sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: amount}}

	/* HV2: v1's BankKeeper.SubtractCoins is no longer available in cosmos-sdk.
	   We use BankKeeper.SendCoinsFromAccountToModule + BankKeeper.BurnCoins instead
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
