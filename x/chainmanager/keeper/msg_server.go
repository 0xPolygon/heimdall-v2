package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
)

var _ types.MsgServer = msgServer{}

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the chainmanager MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

func (srv msgServer) UpdateParams(ctx context.Context, req *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if srv.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", srv.GetAuthority(), req.Authority)
	}

	if err := req.Params.Validate(); err != nil {
		return nil, errorsmod.Wrapf(types.ErrInvalidParams, "invalid chainmanager params; %s", err)
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if err := srv.SetParams(sdkCtx, req.Params); err != nil {
		return nil, errorsmod.Wrapf(sdkerrors.ErrLogic, "failed to update chainmanager params; %s", err)
	}

	return &types.MsgUpdateParamsResponse{}, nil
}
