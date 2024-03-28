package keeper

import (
	"context"
	"github.com/0xPolygon/heimdall-v2/x/topup/types"
)

type msgServer struct {
	*Keeper
}

// NewMsgServerImpl returns an implementation of the gov MsgServer interface for the provided Keeper.
func NewMsgServerImpl(keeper *Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

func (m msgServer) CreateTopupTx(ctx context.Context, tx *types.MsgTopupTx) (*types.MsgTopupTxResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (m msgServer) WithdrawFeeTx(ctx context.Context, tx *types.MsgWithdrawFeeTx) (*types.MsgWithdrawFeeTxResponse, error) {
	//TODO implement me
	panic("implement me")
}
