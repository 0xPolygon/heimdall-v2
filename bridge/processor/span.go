package processor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/ethereum/go-ethereum/common"

	"github.com/0xPolygon/heimdall-v2/bridge/util"
	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/x/bor/types"
	stakeTypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

// SpanProcessor - process span related events
type SpanProcessor struct {
	BaseProcessor

	// header listener subscription
	cancelSpanService context.CancelFunc
}

// Start starts new block subscription
func (sp *SpanProcessor) Start() error {
	sp.Logger.Info("Starting")

	// create cancellable context
	spanCtx, cancelSpanService := context.WithCancel(context.Background())

	sp.cancelSpanService = cancelSpanService

	// start polling for span
	sp.Logger.Info("Start polling for span", "pollInterval", helper.GetConfig().SpanPollInterval)

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
		return
	}

	sp.Logger.Debug("Found last span", "lastSpan", lastSpan.Id, "startBlock", lastSpan.StartBlock, "endBlock", lastSpan.EndBlock)

	nextSpanMsg, err := sp.fetchNextSpanDetails(lastSpan.Id+1, lastSpan.EndBlock+1)
	if err != nil {
		sp.Logger.Error("Unable to fetch next span details", "error", err, "lastSpanId", lastSpan.Id)
		return
	}

	// check if current user is among next span producers
	if sp.isSpanProposer(nextSpanMsg.SelectedProducers) {
		go sp.propose(ctx, lastSpan, nextSpanMsg)
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

		seed, err := sp.fetchNextSpanSeed(nextSpanMsg.Id)
		if err != nil {
			sp.Logger.Info("Error while fetching next span seed from HeimdallServer", "err", err)
			return
		}

		// broadcast to heimdall
		msg := types.MsgProposeSpan{
			SpanId:     nextSpanMsg.Id,
			Proposer:   string(helper.GetAddress()[:]),
			StartBlock: nextSpanMsg.StartBlock,
			EndBlock:   nextSpanMsg.EndBlock,
			ChainId:    nextSpanMsg.ChainId,
			Seed:       seed.Bytes(),
		}

		// return broadcast to heimdall
		txRes, err := sp.txBroadcaster.BroadcastToHeimdall(&msg, nil)
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

	var lastSpan types.Span
	if err = json.Unmarshal(result, &lastSpan); err != nil {
		sp.Logger.Error("Error unmarshalling span", "error", err)
		return nil, err
	}

	return &lastSpan, nil
}

// getCurrentChildBlock gets the current child block
func (sp *SpanProcessor) getCurrentChildBlock(ctx context.Context) (uint64, error) {
	childBlock, err := sp.contractCaller.GetBorChainBlock(ctx, nil)
	if err != nil {
		return 0, err
	}

	return childBlock.Number.Uint64(), nil
}

// isSpanProposer checks if current user is span proposer
func (sp *SpanProcessor) isSpanProposer(nextSpanProducers []stakeTypes.Validator) bool {
	ac := address.NewHexCodec()
	// anyone among next span producers can become next span proposer
	for _, val := range nextSpanProducers {
		signerBytes, err := ac.StringToBytes(val.Signer)
		if err != nil {
			return false
		}
		if bytes.Equal(signerBytes, helper.GetAddress()) {
			return true
		}
	}

	return false
}

// fetch next span details from heimdall.
func (sp *SpanProcessor) fetchNextSpanDetails(id uint64, start uint64) (*types.Span, error) {
	req, err := http.NewRequest("GET", helper.GetHeimdallServerEndpoint(util.NextSpanInfoURL), nil)
	if err != nil {
		sp.Logger.Error("Error creating a new request", "error", err)
		return nil, err
	}

	configParams, err := util.GetChainmanagerParams()
	if err != nil {
		sp.Logger.Error("Error while fetching chainmanager params", "error", err)
		return nil, err
	}

	q := req.URL.Query()
	q.Add("span_id", strconv.FormatUint(id, 10))
	q.Add("start_block", strconv.FormatUint(start, 10))
	q.Add("chain_id", configParams.ChainParams.BorChainId)
	q.Add("proposer", helper.GetFromAddress(sp.cliCtx))
	req.URL.RawQuery = q.Encode()

	// fetch next span details
	result, err := helper.FetchFromAPI(req.URL.String())
	if err != nil {
		sp.Logger.Error("Error fetching proposers", "error", err)
		return nil, err
	}

	var msg types.Span
	if err = json.Unmarshal(result, &msg); err != nil {
		sp.Logger.Error("Error unmarshalling propose tx msg ", "error", err)
		return nil, err
	}

	sp.Logger.Debug("◽ Generated proposer span msg", "msg", msg.String())

	return &msg, nil
}

// fetchNextSpanSeed - fetches seed for next span
func (sp *SpanProcessor) fetchNextSpanSeed(id uint64) (nextSpanSeed common.Hash, err error) {
	sp.Logger.Info("Sending Rest call to Get Seed for next span")

	response, err := helper.FetchFromAPI(fmt.Sprintf(helper.GetHeimdallServerEndpoint(util.NextSpanSeedURL), strconv.FormatUint(id, 10)))
	if err != nil {
		sp.Logger.Error("Error Fetching nextspanseed from HeimdallServer ", "error", err)
		return nextSpanSeed, err
	}

	sp.Logger.Info("Next span seed fetched")

	if err = json.Unmarshal(response, &nextSpanSeed); err != nil {
		sp.Logger.Error("Error unmarshalling nextSpanSeed received from Heimdall Server", "error", err)
		return nextSpanSeed, err
	}

	return nextSpanSeed, nil
}

// Stop stops all necessary go routines
func (sp *SpanProcessor) Stop() {
	// cancel span polling
	sp.cancelSpanService()
}
