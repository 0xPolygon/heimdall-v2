package processor

import (
	"bytes"
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

const (
	// Error messages
	errMsgSpanCheckingProposer                   = "SpanProcessor: error while checking if proposer"
	errMsgSpanFetchingLastSpan                   = "SpanProcessor: unable to fetch last span"
	errMsgSpanFetchingCurrentChildBlock          = "SpanProcessor: error fetching current child block"
	errMsgSpanFetchingNextSpanDetails            = "SpanProcessor: unable to fetch next span details"
	errMsgSpanRecoveredPanic                     = "SpanProcessor: recovered panic in propose goroutine"
	errMsgSpanPropose                            = "SpanProcessor: error in propose"
	errMsgSpanFetchingLastSpanForVotes           = "SpanProcessor: uable to fetch last span"
	errMsgSpanValidatorNotFound                  = "SpanProcessor: validator not found in last span"
	errMsgSpanFetchingProducerVotes              = "SpanProcessor: unable to fetch producer votes"
	errMsgSpanSendingProducerVotes               = "SpanProcessor: error while sending producer votes"
	errMsgSpanConvertingAddress                  = "SpanProcessor: error converting address to string"
	errMsgSpanBroadcastingToHeimdall             = "SpanProcessor: error while broadcasting span to heimdall"
	errMsgSpanProducerVotesTxFailed              = "SpanProcessor: producer votes tx failed on heimdall"
	errMsgSpanFetchingCurrentChildBlockInPropose = "SpanProcessor: error while fetching current child block"
	errMsgSpanConvertingAddressInPropose         = "SpanProcessor: error converting address to string"
	errMsgSpanBroadcastingSpanToHeimdall         = "SpanProcessor: error while broadcasting span to heimdall. spanId: %d, startBlock: %d, endBlock: %d, error: %w"
	errMsgSpanProposeSpanTxFailed                = "SpanProcessor: propose span tx failed on heimdall, txHash: %s, code: %d, spanId: %d, startBlock: %d, endBlock: %d"
	errMsgSpanFetchingLatestSpan                 = "SpanProcessor: error while fetching latest span"
	errMsgSpanUnmarshallingSpan                  = "SpanProcessor: error unmarshalling span"
	errMsgSpanCreatingRequest                    = "SpanProcessor: error creating a new request"
	errMsgSpanFetchingProducerVotesAPI           = "SpanProcessor: error fetching producer votes"
	errMsgSpanUnmarshallingProducerVotes         = "SpanProcessor: error unmarshalling producer votes"
	errMsgSpanFetchingChainmanagerParams         = "SpanProcessor: error while fetching chainmanager params"
	errMsgSpanFetchingProposers                  = "SpanProcessor: error fetching proposers"
	errMsgSpanUnmarshallingProposeTxMsg          = "SpanProcessor: error unmarshalling propose tx msg "
	errMsgSpanFetchingNextSpanSeed               = "SpanProcessor: error fetching next span seed from HeimdallServer "
	errMsgSpanUnmarshallingNextSpanSeed          = "SpanProcessor: error unmarshalling next span seed received from HeimdallServer"

	// Info messages
	infoMsgSpanStarting         = "SpanProcessor: starting process"
	infoMsgSpanStartPolling     = "SpanProcessor: start polling for span"
	infoMsgSpanPollingStopped   = "SpanProcessor: polling stopped"
	infoMsgSpanProposingNewSpan = "SpanProcessor: proposing new span"

	// Debug messages
	debugMsgSpanNotProposer                 = "SpanProcessor: not the proposer, skipping span proposal"
	debugMsgSpanLastSpanNotFound            = "SpanProcessor: last span not found"
	debugMsgSpanCurrentBorBlockLessThanLast = "SpanProcessor: current bor block is less than last span start block, skipping proposing span"
	debugMsgSpanFoundLastSpan               = "SpanProcessor: found last span"
	debugMsgSpanCurrentProducerVotes        = "SpanProcessor: current producer votes"
	debugMsgSpanLocalProducers              = "SpanProcessor: local producers"
	debugMsgSpanUpdatingProducerVotes       = "SpanProcessor: updating producer votes"
	debugMsgSpanDetails                     = "SpanProcessor: span details"
	debugMsgSpanGeneratedProposerSpanMsg    = "SpanProcessor: generated proposer span msg"
	debugMsgSpanSendingRestCallForSeed      = "SpanProcessor: sending REST call to get seed for the next span"
	debugMsgSpanNextSpanSeedFetched         = "SpanProcessor: next span seed fetched"
)

// SpanProcessor - process span related events
type SpanProcessor struct {
	BaseProcessor

	// header listener subscription
	cancelSpanService context.CancelFunc
}

// Start starts new block subscription
func (sp *SpanProcessor) Start() error {
	sp.Logger.Info(infoMsgSpanStarting)

	// create cancellable context
	spanCtx, cancelSpanService := context.WithCancel(context.Background())

	sp.cancelSpanService = cancelSpanService

	// start polling for span
	sp.Logger.Info(infoMsgSpanStartPolling, "pollInterval", helper.GetConfig().SpanPollInterval)

	go sp.startPolling(spanCtx, helper.GetConfig().SpanPollInterval)

	return nil
}

// RegisterTasks - nil
func (sp *SpanProcessor) RegisterTasks() {
}

// startPolling polls heimdall and checks if a new span needs to be proposed
func (sp *SpanProcessor) startPolling(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	// stop ticker when everything done
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sp.checkAndPropose(ctx)
			sp.checkAndVoteProducers(ctx)
		case <-ctx.Done():
			sp.Logger.Info(infoMsgSpanPollingStopped)
			ticker.Stop()

			return
		}
	}
}

// checkAndPropose checks if the current user is the span proposer and proposes the span
func (sp *SpanProcessor) checkAndPropose(ctx context.Context) {
	isProposer, err := util.IsProposer(sp.cliCtx.Codec)
	if err != nil {
		sp.Logger.Error(errMsgSpanCheckingProposer, "error", err)
		return
	}

	if !isProposer {
		sp.Logger.Debug(debugMsgSpanNotProposer)
		return
	}

	lastSpan, err := sp.getLastSpan()
	if err != nil {
		sp.Logger.Error(errMsgSpanFetchingLastSpan, "error", err)
		return
	}

	if lastSpan == nil {
		sp.Logger.Debug(debugMsgSpanLastSpanNotFound)
		return
	}

	latestBlock, err := sp.contractCaller.GetBorChainBlock(ctx, nil)
	if err != nil {
		sp.Logger.Error(errMsgSpanFetchingCurrentChildBlock, "error", err)
		return
	}
	latestBorBlockNumber := latestBlock.Number.Uint64()

	if latestBorBlockNumber < lastSpan.StartBlock {
		sp.Logger.Debug(debugMsgSpanCurrentBorBlockLessThanLast,
			"lastBlock", latestBorBlockNumber,
			"lastSpanStartBlock", lastSpan.StartBlock,
		)
		return
	}

	sp.Logger.Debug(debugMsgSpanFoundLastSpan,
		"lastSpan", lastSpan.Id,
		"startBlock", lastSpan.StartBlock,
		"endBlock", lastSpan.EndBlock,
	)

	nextSpanMsg, err := sp.fetchNextSpanDetails(lastSpan.Id+1, lastSpan.EndBlock+1)
	if err != nil {
		sp.Logger.Error(errMsgSpanFetchingNextSpanDetails, "error", err, "lastSpanId", lastSpan.Id)
		return
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				sp.Logger.Error(errMsgSpanRecoveredPanic, "panic", r)
			}
		}()

		if err := sp.propose(ctx, lastSpan, nextSpanMsg); err != nil {
			sp.Logger.Error(errMsgSpanPropose, "error", err)
		}
	}()
}

func (sp *SpanProcessor) checkAndVoteProducers(ctx context.Context) {
	validatorPubKey := helper.GetPubKey()

	lastSpan, err := sp.getLastSpan()
	if err != nil {
		sp.Logger.Error(errMsgSpanFetchingLastSpanForVotes, "error", err)
		return
	}

	found := false
	validatorId := uint64(0)

	for _, validator := range lastSpan.ValidatorSet.Validators {
		if bytes.Equal(validator.PubKey, validatorPubKey) {
			validatorId = validator.ValId
			found = true
			break
		}
	}

	if !found {
		sp.Logger.Error(errMsgSpanValidatorNotFound, "validatorPubKey", validatorPubKey)
		return
	}

	producerVotes, err := sp.getProducerVotesByValidatorId(validatorId)
	if err != nil {
		sp.Logger.Error(errMsgSpanFetchingProducerVotes, "error", err)
		return
	}

	sp.Logger.Debug(debugMsgSpanCurrentProducerVotes, "votes", producerVotes)

	localProducers := helper.GetProducerVotes()

	sp.Logger.Debug(debugMsgSpanLocalProducers, "producers", localProducers)

	needToUpdateVotes := false
	if len(localProducers) != len(producerVotes.Votes) {
		needToUpdateVotes = true
	} else {
		for i, producer := range localProducers {
			if producer != producerVotes.Votes[i] {
				needToUpdateVotes = true
				break
			}
		}
	}

	if needToUpdateVotes {
		err := sp.sendProducerVotes(ctx, validatorId, localProducers)
		if err != nil {
			sp.Logger.Error(errMsgSpanSendingProducerVotes, "error", err)
		}
	}
}

func (sp *SpanProcessor) sendProducerVotes(ctx context.Context, validatorId uint64, producerVotes []uint64) error {
	sp.Logger.Debug(debugMsgSpanUpdatingProducerVotes, "producers", producerVotes)

	addrString, err := helper.GetAddressString()
	if err != nil {
		sp.Logger.Error(errMsgSpanConvertingAddress, "err", err)
		return err
	}

	msg := types.MsgVoteProducers{
		Voter:   addrString,
		VoterId: validatorId,
		Votes:   types.ProducerVotes{Votes: producerVotes},
	}

	txRes, err := sp.txBroadcaster.BroadcastToHeimdall(ctx, &msg, nil)
	if err != nil {
		sp.Logger.Error(errMsgSpanBroadcastingToHeimdall, "error", err)
		return err
	}

	if txRes.Code != abci.CodeTypeOK {
		sp.Logger.Error(errMsgSpanProducerVotesTxFailed, "txHash", txRes.TxHash, "code", txRes.Code)
		return fmt.Errorf(errMsgSpanProducerVotesTxFailed+", code: %d", txRes.Code)
	}

	return nil
}

// propose producers for the next span if needed
func (sp *SpanProcessor) propose(ctx context.Context, lastSpan *types.Span, nextSpanMsg *types.Span) error {
	// call with the last span on record plus new span duration and see if it has been proposed
	currentBlock, err := sp.getCurrentChildBlock(ctx)
	if err != nil {
		return fmt.Errorf(errMsgSpanFetchingCurrentChildBlockInPropose+": %w", err)
	}

	if lastSpan.StartBlock <= currentBlock && currentBlock <= lastSpan.EndBlock {
		// log new span
		sp.Logger.Info(infoMsgSpanProposingNewSpan, "spanId", nextSpanMsg.Id, "startBlock", nextSpanMsg.StartBlock, "endBlock", nextSpanMsg.EndBlock)

		seed, seedAuthor, err := sp.fetchNextSpanSeed(nextSpanMsg.Id)
		if err != nil {
			return fmt.Errorf(errMsgSpanFetchingNextSpanSeed+": %w", err)
		}

		addrString, err := helper.GetAddressString()
		if err != nil {
			return fmt.Errorf(errMsgSpanConvertingAddressInPropose+": %w", err)
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
		txRes, err := sp.txBroadcaster.BroadcastToHeimdall(ctx, &msg, nil)
		if err != nil {
			return fmt.Errorf(errMsgSpanBroadcastingSpanToHeimdall,
				nextSpanMsg.Id, nextSpanMsg.StartBlock, nextSpanMsg.EndBlock, err)
		}

		if txRes.Code != abci.CodeTypeOK {
			return fmt.Errorf(errMsgSpanProposeSpanTxFailed,
				txRes.TxHash, txRes.Code, nextSpanMsg.Id, nextSpanMsg.StartBlock, nextSpanMsg.EndBlock)
		}
	}

	return nil
}

// getLastSpan gets the last span from heimdall
func (sp *SpanProcessor) getLastSpan() (*types.Span, error) {
	// fetch the latest start block from heimdall using the rest query
	result, err := helper.FetchFromAPI(helper.GetHeimdallServerEndpoint(util.LatestSpanURL))
	if err != nil {
		sp.Logger.Error(errMsgSpanFetchingLatestSpan)
		return nil, err
	}

	var lastSpan types.QueryLatestSpanResponse
	if err = sp.cliCtx.Codec.UnmarshalJSON(result, &lastSpan); err != nil {
		sp.Logger.Error(errMsgSpanUnmarshallingSpan, "error", err)
	}
	return &lastSpan.Span, nil
}

// getProducerVotesByValidatorId gets the producer votes for a given voter id
func (sp *SpanProcessor) getProducerVotesByValidatorId(validatorId uint64) (*types.ProducerVotes, error) {
	req, err := http.NewRequest("GET", helper.GetHeimdallServerEndpoint(fmt.Sprintf(util.ProducerVotesURL, validatorId)), nil)
	if err != nil {
		sp.Logger.Error(errMsgSpanCreatingRequest, "error", err)
		return nil, err
	}

	result, err := helper.FetchFromAPI(req.URL.String())
	if err != nil {
		sp.Logger.Error(errMsgSpanFetchingProducerVotesAPI, "error", err)
		return nil, err
	}

	var res types.QueryProducerVotesByValidatorIdResponse
	if err = sp.cliCtx.Codec.UnmarshalJSON(result, &res); err != nil {
		sp.Logger.Error(errMsgSpanUnmarshallingProducerVotes, "error", err)
		return nil, err
	}

	return &types.ProducerVotes{Votes: res.Votes}, nil
}

// getSpanById gets span details by id
func (sp *SpanProcessor) getSpanById(id uint64) (*types.Span, error) {
	// fetch the latest span from heimdall using the rest query
	result, err := helper.FetchFromAPI(fmt.Sprintf(helper.GetHeimdallServerEndpoint(util.SpanByIdURL), strconv.FormatUint(id, 10)))
	if err != nil {
		sp.Logger.Error(errMsgSpanFetchingLatestSpan)
		return nil, err
	}

	var span types.QuerySpanByIdResponse
	if err = sp.cliCtx.Codec.UnmarshalJSON(result, &span); err != nil {
		sp.Logger.Error(errMsgSpanUnmarshallingSpan, "error", err)
		return nil, err
	}

	sp.Logger.Debug(debugMsgSpanDetails, "span", span.Span.String())
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

// fetchNextSpanDetails fetches next span details from heimdall
func (sp *SpanProcessor) fetchNextSpanDetails(id uint64, start uint64) (*types.Span, error) {
	req, err := http.NewRequest("GET", helper.GetHeimdallServerEndpoint(util.NextSpanInfoURL), nil)
	if err != nil {
		sp.Logger.Error(errMsgSpanCreatingRequest, "error", err)
		return nil, err
	}

	configParams, err := util.GetChainmanagerParams(sp.cliCtx.Codec)
	if err != nil {
		sp.Logger.Error(errMsgSpanFetchingChainmanagerParams, "error", err)
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
		sp.Logger.Error(errMsgSpanFetchingProposers, "error", err)
		return nil, err
	}

	var res types.QueryNextSpanResponse
	if err = sp.cliCtx.Codec.UnmarshalJSON(result, &res); err != nil {
		sp.Logger.Error(errMsgSpanUnmarshallingProposeTxMsg, "error", err)
		return nil, err
	}

	sp.Logger.Debug(debugMsgSpanGeneratedProposerSpanMsg, "msg", res.Span.String())

	return &res.Span, nil
}

// fetchNextSpanSeed fetches seed for next span
func (sp *SpanProcessor) fetchNextSpanSeed(id uint64) (common.Hash, string, error) {
	sp.Logger.Debug(debugMsgSpanSendingRestCallForSeed)

	response, err := helper.FetchFromAPI(fmt.Sprintf(helper.GetHeimdallServerEndpoint(util.NextSpanSeedURL), strconv.FormatUint(id, 10)))
	if err != nil {
		sp.Logger.Error(errMsgSpanFetchingNextSpanSeed, "error", err)
		return common.Hash{}, "", err
	}

	sp.Logger.Debug(debugMsgSpanNextSpanSeedFetched)

	var nextSpanSeedRes types.QueryNextSpanSeedResponse
	if err := sp.cliCtx.Codec.UnmarshalJSON(response, &nextSpanSeedRes); err != nil {
		sp.Logger.Error(errMsgSpanUnmarshallingNextSpanSeed, "error", err)
		return common.Hash{}, "", err
	}

	return common.HexToHash(nextSpanSeedRes.Seed), nextSpanSeedRes.SeedAuthor, nil
}

// Stop stops all necessary go routines
func (sp *SpanProcessor) Stop() {
	// cancel span polling
	sp.cancelSpanService()
}
