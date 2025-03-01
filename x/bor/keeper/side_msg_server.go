package keeper

import (
	"bytes"
	"errors"
	"strconv"

	cmttypes "github.com/cometbft/cometbft/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	util "github.com/0xPolygon/heimdall-v2/common/address"
	"github.com/0xPolygon/heimdall-v2/sidetxs"
	heimdallTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/bor/types"
)

var SpanProposeMsgTypeURL = sdk.MsgTypeURL(&types.MsgProposeSpan{})

type sideMsgServer struct {
	k *Keeper
}

var _ sidetxs.SideMsgServer = sideMsgServer{}

// NewSideMsgServerImpl returns an implementation of the x/bor SideMsgServer interface for the provided Keeper.
func NewSideMsgServerImpl(keeper *Keeper) sidetxs.SideMsgServer {
	return &sideMsgServer{
		k: keeper,
	}
}

// SideTxHandler returns a side handler for span type messages.
func (s sideMsgServer) SideTxHandler(methodName string) sidetxs.SideTxHandler {
	switch methodName {
	case SpanProposeMsgTypeURL:
		return s.SideHandleMsgSpan
	default:
		return nil
	}
}

// SideHandleMsgSpan validates external calls required for processing proposed span
func (s sideMsgServer) SideHandleMsgSpan(ctx sdk.Context, msgI sdk.Msg) sidetxs.Vote {
	logger := s.k.Logger(ctx)

	msg, ok := msgI.(*types.MsgProposeSpan)
	if !ok {
		logger.Error("MsgProposeSpan type mismatch", "msg type received", msgI)
		return sidetxs.Vote_UNSPECIFIED

	}

	logger.Debug("✅ validating external call for span msg",
		"proposer", msg.Proposer,
		"spanId", msg.SpanId,
		"startBlock", msg.StartBlock,
		"endBlock", msg.EndBlock,
		"seed", msg.Seed,
	)

	// calculate next span seed locally
	nextSpanSeed, nextSpanSeedAuthor, err := s.k.FetchNextSpanSeed(ctx, msg.SpanId)
	if err != nil {
		logger.Error("error fetching next span seed from mainChain", "error", err)
		return sidetxs.Vote_UNSPECIFIED
	}

	// check if span seed matches or not
	if !bytes.Equal(msg.Seed, nextSpanSeed.Bytes()) {
		logger.Error(
			"span seed does not match",
			"msgSeed", msg.Seed,
			"mainChainSeed", nextSpanSeed.String(),
		)

		return sidetxs.Vote_VOTE_NO
	}

	// check if span seed author matches or not.
	if util.FormatAddress(msg.SeedAuthor) != util.FormatAddress(nextSpanSeedAuthor.Hex()) {
		logger.Error(
			"Span Seed Author does not match",
			"proposer", msg.Proposer,
			"chainID", msg.ChainId,
			"msgSeed", msg.Seed,
			"msgSeedAuthor", msg.SeedAuthor,
			"mainchainSeedAuthor", nextSpanSeedAuthor.Hex(),
			"mainchainSeed", nextSpanSeed,
		)

		return sidetxs.Vote_VOTE_NO
	}

	// fetch current child block
	childBlock, err := s.k.contractCaller.GetBorChainBlock(ctx, nil)
	if err != nil {
		logger.Error("error fetching current child block", "error", err)
		return sidetxs.Vote_UNSPECIFIED
	}

	lastSpan, err := s.k.GetLastSpan(ctx)
	if err != nil {
		logger.Error("error fetching last span", "error", err)
		return sidetxs.Vote_UNSPECIFIED
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

		return sidetxs.Vote_VOTE_NO
	}

	logger.Debug("✅ successfully validated external call for span msg")

	return sidetxs.Vote_VOTE_YES
}

// PostTxHandler returns a side handler for span type messages.
func (s sideMsgServer) PostTxHandler(methodName string) sidetxs.PostTxHandler {
	switch methodName {
	case SpanProposeMsgTypeURL:
		return s.PostHandleMsgSpan
	default:
		return nil
	}
}

// PostHandleMsgSpan handles state persisting span msg
func (s sideMsgServer) PostHandleMsgSpan(ctx sdk.Context, msgI sdk.Msg, sideTxResult sidetxs.Vote) error {
	logger := s.k.Logger(ctx)

	msg, ok := msgI.(*types.MsgProposeSpan)
	if !ok {
		err := errors.New("MsgProposeSpan type mismatch")
		logger.Error(err.Error(), "msg type received", msg)
		return err
	}

	// Skip handler if span is not approved
	if sideTxResult != sidetxs.Vote_VOTE_YES {
		logger.Debug("skipping new span since side-tx didn't get yes votes")
		return errors.New("side-tx didn't get yes votes")
	}

	// check for replay
	ok, err := s.k.HasSpan(ctx, msg.SpanId)
	if err != nil {
		logger.Error("error occurred while checking for span", "span id", msg.SpanId, "error", err)
		return err
	}
	if ok {
		logger.Debug("skipping new span as it's already processed", "span id", msg.SpanId)
		return errors.New("span already processed")
	}

	logger.Debug("persisting span state", "span id", msg.SpanId, "sideTxResult", sideTxResult)

	seedAuthor := common.HexToAddress(msg.SeedAuthor)
	if err = s.k.StoreSeedProducer(ctx, msg.SpanId, &seedAuthor); err != nil {
		logger.Error("Unable to store seed producer", "error", err)
		return err
	}

	// freeze for new span
	err = s.k.FreezeSet(ctx, msg.SpanId, msg.StartBlock, msg.EndBlock, msg.ChainId, common.Hash(msg.Seed))
	if err != nil {
		logger.Error("unable to freeze validator set for span", "span id", msg.SpanId, "error", err)
		return err
	}

	txBytes := ctx.TxBytes()
	hash := cmttypes.Tx(txBytes).Hash()

	// add events
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeProposeSpan,
			sdk.NewAttribute(sdk.AttributeKeyAction, msg.Type()),                               // action
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),             // module name
			sdk.NewAttribute(heimdallTypes.AttributeKeyTxHash, common.BytesToHash(hash).Hex()), // tx hash
			sdk.NewAttribute(heimdallTypes.AttributeKeySideTxResult, sideTxResult.String()),    // result
			sdk.NewAttribute(types.AttributeKeySpanID, strconv.FormatUint(msg.SpanId, 10)),
			sdk.NewAttribute(types.AttributeKeySpanStartBlock, strconv.FormatUint(msg.StartBlock, 10)),
			sdk.NewAttribute(types.AttributeKeySpanEndBlock, strconv.FormatUint(msg.EndBlock, 10)),
		),
	})

	return nil
}
