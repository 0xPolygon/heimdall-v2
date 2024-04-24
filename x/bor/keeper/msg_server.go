package keeper

import (
	"context"
	"strconv"

	"github.com/0xPolygon/heimdall-v2/x/bor/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type msgServer struct {
	Keeper
}

var _ types.MsgServer = msgServer{}

// NewMsgServerImpl returns an implementation of the bor MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

func (k msgServer) ProposeSpan(ctx context.Context, msg *types.MsgProposeSpanRequest) (*types.MsgProposeSpanResponse, error) {
	logger := k.Logger(ctx)

	logger.Debug("✅ Validating proposed span msg",
		"proposer", msg.Proposer,
		"spanId", msg.SpanId,
		"startBlock", msg.StartBlock,
		"endBlock", msg.EndBlock,
		"seed", msg.Seed,
	)

	_, err := sdk.ValAddressFromHex(msg.Proposer)
	if err != nil {
		logger.Error("invalid proposer address", "error", err)
		return nil, err
	}

	// verify chain id
	chainParams, err := k.ck.GetParams(ctx)
	if err != nil {
		logger.Error("failed to get chain params", "error", err)
		return nil, err
	}

	if chainParams.ChainParams.BorChainId != msg.ChainId {
		logger.Error("invalid bor chain id", "expected", chainParams.ChainParams.BorChainId, "got", msg.ChainId)
		return nil, types.ErrInvalidChainID
	}

	lastSpan, err := k.GetLastSpan(ctx)
	if err != nil {
		logger.Error("Unable to fetch last span", "Error", err)
		return nil, err
	}

	// Validate span continuity
	if lastSpan.Id+1 != msg.SpanId || msg.StartBlock != lastSpan.EndBlock+1 || msg.EndBlock < msg.StartBlock {
		logger.Error("Blocks not in continuity",
			"lastSpanId", lastSpan.Id,
			"spanId", msg.SpanId,
			"lastSpanStartBlock", lastSpan.StartBlock,
			"lastSpanEndBlock", lastSpan.EndBlock,
			"spanStartBlock", msg.StartBlock,
			"spanEndBlock", msg.EndBlock,
		)

		return nil, types.ErrInvalidSpan
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// add events
	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeProposeSpan,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(types.AttributeKeySpanID, strconv.FormatUint(msg.SpanId, 10)),
			sdk.NewAttribute(types.AttributeKeySpanStartBlock, strconv.FormatUint(msg.StartBlock, 10)),
			sdk.NewAttribute(types.AttributeKeySpanEndBlock, strconv.FormatUint(msg.EndBlock, 10)),
		),
	})

	logger.Debug("Emitted propose-span event")
	return &types.MsgProposeSpanResponse{}, nil
}
