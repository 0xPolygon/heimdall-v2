package keeper

import (
	"context"
	"strconv"
	"time"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/metrics/api"
	heimdallTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/clerk/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the clerk MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

func (srv msgServer) HandleMsgEventRecord(ctx context.Context, msg *types.MsgEventRecord) (*types.MsgEventRecordResponse, error) {
	var err error
	startTime := time.Now()
	defer recordClerkTransactionMetric(api.HandleMsgEventRecordMethod, startTime, &err)

	logger := srv.Logger(ctx)

	logger.Debug(helper.LogValidatingExternalCall("ClerkEventRecord"),
		"id", msg.Id,
		"contract", msg.ContractAddress,
		"data", string(msg.Data),
		"txHash", msg.TxHash,
		"logIndex", msg.LogIndex,
		"blockNumber", msg.BlockNumber,
	)

	// check if the event record exists
	if exists := srv.HasEventRecord(ctx, msg.Id); exists {
		return nil, types.ErrEventRecordAlreadySynced
	}

	// chainManager params
	params, err := srv.ChainKeeper.GetParams(ctx)
	if err != nil {
		logger.Error(heimdallTypes.ErrMsgFailedToGetChainManagerParams, "error", err)
		return nil, err
	}

	chainParams := params.ChainParams

	// check chain id
	if !helper.ValidateChainID(msg.ChainId, chainParams.BorChainId, logger, "clerk") {
		return nil, errors.Wrapf(sdkerrors.ErrInvalidRequest, "invalid bor chain id")
	}

	// sequence id
	sequence := helper.CalculateSequence(msg.BlockNumber, msg.LogIndex)

	// check if the event has already been processed
	if srv.HasRecordSequence(ctx, sequence) {
		logger.Error(heimdallTypes.ErrMsgEventAlreadyProcessed, heimdallTypes.LogKeySequence, sequence)
		return nil, errors.Wrapf(sdkerrors.ErrConflict, heimdallTypes.ErrMsgOldEventsNotAllowed)
	}

	// add events
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeRecord,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(types.AttributeKeyRecordID, strconv.FormatUint(msg.Id, 10)),
			sdk.NewAttribute(types.AttributeKeyRecordContract, msg.ContractAddress),
			sdk.NewAttribute(types.AttributeKeyRecordTxHash, msg.TxHash),
			sdk.NewAttribute(types.AttributeKeyRecordTxLogIndex, strconv.FormatUint(msg.LogIndex, 10)),
		),
	})

	return &types.MsgEventRecordResponse{}, nil
}

func recordClerkTransactionMetric(method string, start time.Time, err *error) {
	success := *err == nil
	api.RecordAPICallWithStart(api.ClerkSubsystem, method, api.TxType, success, start)
}
