package keeper

import (
	"context"
	"time"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/metrics/api"
	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
)

type msgServer struct {
	*Keeper
}

// NewMsgServerImpl returns an implementation of the milestone MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper *Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

// UpdateParams defines a method to update the params in x/milestone module.
func (srv msgServer) UpdateParams(ctx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	var err error
	startTime := time.Now()
	defer recordMilestoneTransactionMetric(api.MilestoneUpdateParamsMethod, startTime, &err)

	if srv.authority != msg.Authority {
		return nil, errorsmod.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", srv.authority, msg.Authority)
	}

	if err := msg.Params.ValidateBasic(); err != nil {
		return nil, err
	}

	// Validators never propose more than the cap, so a larger value would only advertise a length
	// that is never used. Gated so the accept/reject of a params update changes at one height.
	if helper.IsKarwar(sdk.UnwrapSDKContext(ctx).BlockHeight()) &&
		msg.Params.MaxMilestonePropositionLength > types.MilestonePropositionLengthCap {
		return nil, errorsmod.Wrapf(govtypes.ErrInvalidProposalMsg, "max milestone proposition length %d exceeds cap %d",
			msg.Params.MaxMilestonePropositionLength, types.MilestonePropositionLengthCap)
	}

	if err := srv.SetParams(ctx, msg.Params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}

func recordMilestoneTransactionMetric(method string, start time.Time, err *error) {
	success := *err == nil
	api.RecordAPICallWithStart(api.MilestoneSubsystem, method, api.TxType, success, start)
}
