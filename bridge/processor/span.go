package processor

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/0xPolygon/heimdall-v2/bridge/util"
	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/x/bor/types"
)

// SpanProcessor - process span related events
type SpanProcessor struct {
	BaseProcessor

	// header listener subscription
	cancelSpanService context.CancelFunc
}

// Start starts new block subscription
func (sp *SpanProcessor) Start() error {
	sp.Logger.Info("starting bor process")

	// create cancellable context
	spanCtx, cancelSpanService := context.WithCancel(context.Background())

	sp.cancelSpanService = cancelSpanService

	// start polling for span
	sp.Logger.Info("start polling for span", "pollInterval", helper.GetConfig().SpanPollInterval)

	go sp.startPolling(spanCtx, helper.GetConfig().SpanPollInterval)

	return nil
}

// RegisterTasks - nil
func (sp *SpanProcessor) RegisterTasks() {
}

// startPolling - polls heimdall and checks if new span needs to be proposed
func (sp *SpanProcessor) startPolling(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	// stop ticker when everything done
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sp.checkAndPropose(ctx)
		case <-ctx.Done():
			sp.Logger.Info("Polling stopped")
			ticker.Stop()

			return
		}
	}
}

// checkAndPropose - will check if current user is span proposer and proposes the span
func (sp *SpanProcessor) checkAndPropose(ctx context.Context) {
	lastSpan, err := sp.getLastSpan()
	if err != nil {
		sp.Logger.Error("Unable to fetch last span", "error", err)
		return
	}

	if lastSpan == nil {
		sp.Logger.Debug("Last span not found")
		return
	}

	latestBlock, err := sp.contractCaller.GetBorChainBlock(ctx, nil)
	if err != nil {
		sp.Logger.Error("Error fetching current child block", "error", err)
		return
	}
	latestBorBlockNumber := latestBlock.Number.Uint64()

	if latestBorBlockNumber < lastSpan.StartBlock {
		sp.Logger.Debug("Current bor block is less than last span start block, skipping proposing span", "currentBlock", latestBlock.Number.Uint64(), "lastSpanStartBlock", lastSpan.StartBlock)
		return
	}

	var latestMilestoneEndBlock uint64
	latestMilestone, err := util.GetLatestMilestone(sp.cliCtx.Codec)
	if err == nil {
		latestMilestoneEndBlock = latestMilestone.EndBlock
	}

	// Max of latest milestone end block and latest bor block number
	// Handle cases where bor is syncing and latest milestone end block is greater than latest bor block number
	maxBlockNumber := max(latestMilestoneEndBlock, latestBorBlockNumber)

	sp.Logger.Debug("Found last span", "lastSpan", lastSpan.Id, "startBlock", lastSpan.StartBlock, "endBlock", lastSpan.EndBlock)

	isProposer, err := util.IsProposer(sp.cliCtx.Codec)
	if err != nil {
		sp.Logger.Error("Error while checking if proposer", "error", err)
		return
	}

	if !isProposer {
		return
	}

	if maxBlockNumber > lastSpan.EndBlock {
		if latestMilestoneEndBlock > lastSpan.EndBlock {
			sp.Logger.Debug("Bor self commmitted spans, backfill heimdall to fill missing spans", "currentBlock", latestBlock.Number.Uint64(), "lastSpanEndBlock", lastSpan.EndBlock)
			go sp.backfillSpans(ctx, latestBorBlockNumber, lastSpan)
			return
		} else {
			sp.Logger.Debug("Will not backfill heimdall spans, as latest milestone end block is less than last span end block", "currentBlock", latestBlock.Number.Uint64(), "lastSpanEndBlock", lastSpan.EndBlock)
			return
		}
	} else {
		if types.IsBlockCloseToSpanEnd(maxBlockNumber, lastSpan.EndBlock) {
			sp.Logger.Debug("Current bor block is close to last span end block, skipping proposing span", "currentBlock", latestBlock.Number.Uint64(), "lastSpanEndBlock", lastSpan.EndBlock)
			return
		}

		nextSpanMsg, err := sp.fetchNextSpanDetails(lastSpan.Id+1, lastSpan.EndBlock+1)
		if err != nil {
			sp.Logger.Error("Unable to fetch next span details", "error", err, "lastSpanId", lastSpan.Id)
			return
		}

		go sp.propose(ctx, lastSpan, nextSpanMsg)
	}

}

func (sp *SpanProcessor) backfillSpans(ctx context.Context, latestBorBlockNumber uint64, lastHeimdallSpan *types.Span) {
	// Get what span bor used when heimdall was down and it tried to fetch the next span from heimdall
	// We need to know if it managed to fetch the latest span or it used the previous one
	// We will take it from the next start block after the last heimdall span end block
	borLastUsedSpanID, err := sp.contractCaller.GetStartBlockHeimdallSpanID(ctx, lastHeimdallSpan.EndBlock+1)
	if err != nil {
		sp.Logger.Error("Error while fetching last used span id", "error", err)
		return
	}

	if borLastUsedSpanID == 0 {
		sp.Logger.Error("No span id found for bor, skipping backfill")
		return
	}

	borLastUsedSpan, err := sp.getSpanById(borLastUsedSpanID)
	if err != nil {
		sp.Logger.Error("Error while fetching last used span", "error", err)
		return
	}

	if borLastUsedSpan == nil {
		sp.Logger.Error("No span found for bor, skipping backfill")
		return
	}

	params, err := util.GetChainmanagerParams(sp.cliCtx.Codec)
	if err != nil {
		sp.Logger.Error("Error while fetching chainmanager params", "error", err)
		return
	}

	addrString, err := helper.GetAddressString()
	if err != nil {
		sp.Logger.Info("error converting address to string", "err", err)
		return
	}

	borSpanId, err := types.CalcCurrentBorSpanId(latestBorBlockNumber, borLastUsedSpan)
	if err != nil {
		sp.Logger.Error("Error calculating current bor span id", "error", err)
		return
	}

	msg := types.MsgBackfillSpans{
		Proposer:        addrString,
		ChainId:         params.ChainParams.BorChainId,
		LatestSpanId:    borLastUsedSpan.Id,
		LatestBorSpanId: borSpanId,
		LatestBorBlock:  latestBorBlockNumber,
	}

	txRes, err := sp.txBroadcaster.BroadcastToHeimdall(&msg, nil) //nolint:contextcheck
	if err != nil {
		sp.Logger.Error("Error while broadcasting spans backfill to heimdall", "spanId", msg.LatestSpanId, "startBlock", msg.LatestBorSpanId, "endBlock", msg.LatestBorBlock, "error", err)
		return
	}

	if txRes.Code != abci.CodeTypeOK {
		sp.Logger.Error("spans backfill tx failed on heimdall", "txHash", txRes.TxHash, "code", txRes.Code)
		return
	}
}

// propose producers for next span if needed
func (sp *SpanProcessor) propose(ctx context.Context, lastSpan *types.Span, nextSpanMsg *types.Span) {
	// call with last span on record + new span duration and see if it has been proposed
	currentBlock, err := sp.getCurrentChildBlock(ctx)
	if err != nil {
		sp.Logger.Error("Unable to fetch current block", "error", err)
		return
	}

	if lastSpan.StartBlock <= currentBlock && currentBlock <= lastSpan.EndBlock {
		// log new span
		sp.Logger.Info("✅ Proposing new span", "spanId", nextSpanMsg.Id, "startBlock", nextSpanMsg.StartBlock, "endBlock", nextSpanMsg.EndBlock)

		seed, seedAuthor, err := sp.fetchNextSpanSeed(nextSpanMsg.Id)
		if err != nil {
			sp.Logger.Info("Error while fetching next span seed from HeimdallServer", "err", err)
			return
		}

		addrString, err := helper.GetAddressString()
		if err != nil {
			sp.Logger.Info("error converting address to string", "err", err)
			return
		}

		// broadcast to heimdall
		msg := types.MsgProposeSpan{
			SpanId:     nextSpanMsg.Id,
			Proposer:   addrString,
			StartBlock: nextSpanMsg.StartBlock,
			EndBlock:   nextSpanMsg.EndBlock,
			ChainId:    nextSpanMsg.BorChainId,
			Seed:       seed.Bytes(),
			SeedAuthor: seedAuthor,
		}

		// return broadcast to heimdall
		txRes, err := sp.txBroadcaster.BroadcastToHeimdall(&msg, nil) //nolint:contextcheck
		if err != nil {
			sp.Logger.Error("Error while broadcasting span to heimdall", "spanId", nextSpanMsg.Id, "startBlock", nextSpanMsg.StartBlock, "endBlock", nextSpanMsg.EndBlock, "error", err)
			return
		}

		if txRes.Code != abci.CodeTypeOK {
			sp.Logger.Error("span tx failed on heimdall", "txHash", txRes.TxHash, "code", txRes.Code)
			return
		}

	}
}

// checks span status
func (sp *SpanProcessor) getLastSpan() (*types.Span, error) {
	// fetch latest start block from heimdall via rest query
	result, err := helper.FetchFromAPI(helper.GetHeimdallServerEndpoint(util.LatestSpanURL))
	if err != nil {
		sp.Logger.Error("Error while fetching latest span")
		return nil, err
	}

	var lastSpan types.QueryLatestSpanResponse
	if err = sp.cliCtx.Codec.UnmarshalJSON(result, &lastSpan); err != nil {
		sp.Logger.Error("Error unmarshalling span", "error", err)
	}
	return &lastSpan.Span, nil
}

// get span by id
func (sp *SpanProcessor) getSpanById(id uint64) (*types.Span, error) {
	// fetch latest start block from heimdall via rest query
	result, err := helper.FetchFromAPI(fmt.Sprintf(helper.GetHeimdallServerEndpoint(util.SpanByIdURL), strconv.FormatUint(id, 10)))
	if err != nil {
		sp.Logger.Error("Error while fetching latest span")
		return nil, err
	}

	var span types.QuerySpanByIdResponse
	if err = sp.cliCtx.Codec.UnmarshalJSON(result, &span); err != nil {
		sp.Logger.Error("Error unmarshalling span", "error", err)
		return nil, err
	}

	sp.Logger.Debug("Span details", "span", span.Span.String())
	return span.Span, nil
}

// getCurrentChildBlock gets the current child block
func (sp *SpanProcessor) getCurrentChildBlock(ctx context.Context) (uint64, error) {
	childBlock, err := sp.contractCaller.GetBorChainBlock(ctx, nil)
	if err != nil {
		return 0, err
	}

	return childBlock.Number.Uint64(), nil
}

// fetch next span details from heimdall.
func (sp *SpanProcessor) fetchNextSpanDetails(id uint64, start uint64) (*types.Span, error) {
	req, err := http.NewRequest("GET", helper.GetHeimdallServerEndpoint(util.NextSpanInfoURL), nil)
	if err != nil {
		sp.Logger.Error("Error creating a new request", "error", err)
		return nil, err
	}

	configParams, err := util.GetChainmanagerParams(sp.cliCtx.Codec)
	if err != nil {
		sp.Logger.Error("Error while fetching chainmanager params", "error", err)
		return nil, err
	}

	q := req.URL.Query()
	q.Add("span_id", strconv.FormatUint(id, 10))
	q.Add("bor_chain_id", configParams.ChainParams.BorChainId)
	q.Add("start_block", strconv.FormatUint(start, 10))
	req.URL.RawQuery = q.Encode()

	// fetch next span details
	result, err := helper.FetchFromAPI(req.URL.String())
	if err != nil {
		sp.Logger.Error("Error fetching proposers", "error", err)
		return nil, err
	}

	var res types.QueryNextSpanResponse
	if err = sp.cliCtx.Codec.UnmarshalJSON(result, &res); err != nil {
		sp.Logger.Error("Error unmarshalling propose tx msg ", "error", err)
		return nil, err
	}

	sp.Logger.Debug("◽ Generated proposer span msg", "msg", res.Span.String())

	return &res.Span, nil
}

// fetchNextSpanSeed - fetches seed for next span
func (sp *SpanProcessor) fetchNextSpanSeed(id uint64) (common.Hash, string, error) {
	sp.Logger.Info("Sending Rest call to Get Seed for next span")

	response, err := helper.FetchFromAPI(fmt.Sprintf(helper.GetHeimdallServerEndpoint(util.NextSpanSeedURL), strconv.FormatUint(id, 10)))
	if err != nil {
		sp.Logger.Error("Error Fetching nextspanseed from HeimdallServer ", "error", err)
		return common.Hash{}, "", err
	}

	sp.Logger.Info("Next span seed fetched")

	var nextSpanSeedRes types.QueryNextSpanSeedResponse
	if err := sp.cliCtx.Codec.UnmarshalJSON(response, &nextSpanSeedRes); err != nil {
		sp.Logger.Error("Error unmarshalling nextSpanSeed received from Heimdall Server", "error", err)
		return common.Hash{}, "", err
	}

	return common.HexToHash(nextSpanSeedRes.Seed), nextSpanSeedRes.SeedAuthor, nil
}

// Stop stops all necessary go routines
func (sp *SpanProcessor) Stop() {
	// cancel span polling
	sp.cancelSpanService()
}
