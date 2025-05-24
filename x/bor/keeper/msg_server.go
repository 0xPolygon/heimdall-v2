package keeper

import (
	"context"
	"strconv"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/0xPolygon/heimdall-v2/x/bor/types"
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

func (m msgServer) ProposeSpan(ctx context.Context, msg *types.MsgProposeSpan) (*types.MsgProposeSpanResponse, error) {
	logger := m.Logger(ctx)

	logger.Debug("✅ validating proposed span msg",
		"proposer", msg.Proposer,
		"spanId", msg.SpanId,
		"startBlock", msg.StartBlock,
		"endBlock", msg.EndBlock,
		"seed", msg.Seed,
	)

	_, err := sdk.ValAddressFromHex(msg.Proposer)
	if err != nil {
		logger.Error("invalid proposer address", "error", err)
		return nil, errors.Wrapf(err, "invalid proposer address")
	}

	// verify chain id
	chainParams, err := m.ck.GetParams(ctx)
	if err != nil {
		logger.Error("failed to get chain params", "error", err)
		return nil, errors.Wrapf(err, "failed to get chain params")
	}

	if chainParams.ChainParams.BorChainId != msg.ChainId {
		logger.Error("invalid bor chain id", "expected", chainParams.ChainParams.BorChainId, "got", msg.ChainId)
		return nil, types.ErrInvalidChainID
	}

	// verify seed length
	if len(msg.Seed) != common.HashLength {
		logger.Error("invalid seed length", "expected", common.HashLength, "got", len(msg.Seed))
		return nil, types.ErrInvalidSeedLength
	}

	lastSpan, err := m.GetLastSpan(ctx)
	if err != nil {
		logger.Error("unable to fetch last span", "Error", err)
		return nil, errors.Wrapf(err, "unable to fetch last span")
	}

	// Validate span continuity
	if lastSpan.Id+1 != msg.SpanId || msg.StartBlock != lastSpan.EndBlock+1 || msg.EndBlock <= msg.StartBlock {
		logger.Error("blocks not in continuity",
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

// UpdateParams defines a method to update the params in x/bor module.
func (m msgServer) UpdateParams(ctx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if m.authority != msg.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", m.authority, msg.Authority)
	}

	if err := msg.Params.ValidateBasic(); err != nil {
		return nil, err
	}

	if err := m.SetParams(ctx, msg.Params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}
