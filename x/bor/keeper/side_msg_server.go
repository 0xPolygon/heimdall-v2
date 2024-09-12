package keeper

import (
	"bytes"
	"context"
	"strconv"

	cmttypes "github.com/cometbft/cometbft/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	hModule "github.com/0xPolygon/heimdall-v2/module"
	"github.com/0xPolygon/heimdall-v2/x/bor/types"
)

var (
	SpanProposeMsgTypeURL = sdk.MsgTypeURL(&types.MsgProposeSpanRequest{})
)

type sideMsgServer struct {
	k *Keeper
}

var _ types.SideMsgServer = sideMsgServer{}

// NewSideMsgServerImpl returns an implementation of the x/bor SideMsgServer interface for the provided Keeper.
func NewSideMsgServerImpl(keeper *Keeper) types.SideMsgServer {
	return &sideMsgServer{
		k: keeper,
	}
}

// SideTxHandler returns a side handler for span type messages.
func (s sideMsgServer) SideTxHandler(methodName string) hModule.SideTxHandler {
	switch methodName {
	case SpanProposeMsgTypeURL:
		return s.SideHandleMsgSpan
	default:
		s.k.Logger(context.Background()).Error("unrecognized side message type", "method", methodName)
		return nil
	}
}

// SideHandleMsgSpan validates external calls required for processing proposed span
func (s sideMsgServer) SideHandleMsgSpan(ctx sdk.Context, msgI sdk.Msg) hModule.Vote {
	logger := s.k.Logger(ctx)

	msg, ok := msgI.(*types.MsgProposeSpanRequest)
	if !ok {
		logger.Error("MsgProposeSpan type mismatch", "msg type received", msgI)
		return hModule.Vote_VOTE_SKIP

	}

	logger.Debug("✅ validating external call for span msg",
		"proposer", msg.Proposer,
		"spanId", msg.SpanId,
		"startBlock", msg.StartBlock,
		"endBlock", msg.EndBlock,
		"seed", msg.Seed,
	)

	// calculate next span seed locally
	nextSpanSeed, err := s.k.FetchNextSpanSeed(ctx)
	if err != nil {
		logger.Error("error fetching next span seed from mainchain", "error", err)
		return hModule.Vote_VOTE_SKIP
	}

	// check if span seed matches or not
	if !bytes.Equal(msg.Seed, nextSpanSeed.Bytes()) {
		logger.Error(
			"span seed does not match",
			"msgSeed", msg.Seed,
			"mainchainSeed", nextSpanSeed.String(),
		)

		return hModule.Vote_VOTE_NO
	}

	// fetch current child block
	childBlock, err := s.k.contractCaller.GetBorChainBlock(nil)
	if err != nil {
		logger.Error("error fetching current child block", "error", err)
		return hModule.Vote_VOTE_SKIP
	}

	lastSpan, err := s.k.GetLastSpan(ctx)
	if err != nil {
		logger.Error("error fetching last span", "error", err)
		return hModule.Vote_VOTE_SKIP
	}

	currentBlock := childBlock.Number.Uint64()
	// check if span proposed is in-turn or not
	if !(lastSpan.StartBlock <= currentBlock && currentBlock <= lastSpan.EndBlock) {
		logger.Error(
			"span proposed is not in-turn",
			"currentChildBlock", currentBlock,
			"msgStartBlock", msg.StartBlock,
			"msgEndBlock", msg.EndBlock,
		)

		return hModule.Vote_VOTE_NO
	}

	logger.Debug("✅ successfully validated external call for span msg")

	return hModule.Vote_VOTE_YES
}

// PostTxHandler returns a side handler for span type messages.
func (s sideMsgServer) PostTxHandler(methodName string) hModule.PostTxHandler {
	switch methodName {
	case SpanProposeMsgTypeURL:
		return s.PostHandleMsgSpan
	default:
		s.k.Logger(context.Background()).Error("unrecognized side message type", "method", methodName)
		return nil
	}
}

// PostHandleMsgSpan handles state persisting span msg
func (s sideMsgServer) PostHandleMsgSpan(ctx sdk.Context, msgI sdk.Msg, sideTxResult hModule.Vote) {
	logger := s.k.Logger(ctx)

	msg, ok := msgI.(*types.MsgProposeSpanRequest)
	if !ok {
		logger.Error("MsgProposeSpan type mismatch", "msg type received", msg)
		return
	}

	// Skip handler if span is not approved
	if sideTxResult != hModule.Vote_VOTE_YES {
		logger.Debug("skipping new span since side-tx didn't get yes votes")
		return
	}

	// check for replay
	ok, err := s.k.HasSpan(ctx, msg.SpanId)
	if err != nil {
		logger.Error("error occurred while checking for span", "span id", msg.SpanId, "error", err)
		return
	}
	if ok {
		logger.Debug("skipping new span as it's already processed", "span id", msg.SpanId)
		return
	}

	logger.Debug("persisting span state", "span id", msg.SpanId, "sideTxResult", sideTxResult)

	// freeze for new span
	err = s.k.FreezeSet(ctx, msg.SpanId, msg.StartBlock, msg.EndBlock, msg.ChainId, common.Hash(msg.Seed))
	if err != nil {
		logger.Error("unable to freeze validator set for span", "span id", msg.SpanId, "error", err)
		return

	}

	txBytes := ctx.TxBytes()
	hash := cmttypes.Tx(txBytes).Hash()

	// add events
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeProposeSpan,
			sdk.NewAttribute(sdk.AttributeKeyAction, msg.Type()),                       // action
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),     // module name
			sdk.NewAttribute(types.AttributeKeyTxHash, common.BytesToHash(hash).Hex()), // tx hash
			sdk.NewAttribute(types.AttributeKeySideTxResult, sideTxResult.String()),    // result
			sdk.NewAttribute(types.AttributeKeySpanID, strconv.FormatUint(msg.SpanId, 10)),
			sdk.NewAttribute(types.AttributeKeySpanStartBlock, strconv.FormatUint(msg.StartBlock, 10)),
			sdk.NewAttribute(types.AttributeKeySpanEndBlock, strconv.FormatUint(msg.EndBlock, 10)),
		),
	})

}
