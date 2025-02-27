package app

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/0xPolygon/heimdall-v2/engine"
	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/cenkalti/backoff/v4"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func (app *HeimdallApp) ProduceELPayload(ctx context.Context) {
	logger := app.Logger()
	var blockCtx nextELBlockCtx
	for {
		select {
		case blockCtx = <-app.nextBlockChan:
			res, err := app.retryBuildNextPayload(blockCtx.context, blockCtx.height+1)
			if err != nil {
				logger.Error("error building next payload", "error", err)
				res = nil
			}

			app.nextExecPayload = res

		case blockCtx = <-app.currBlockChan:
			res, err := app.retryBuildLatestPayload(blockCtx.context, blockCtx.height)
			if err != nil {
				logger.Error("error building latest payload", "error", err)
				res = nil
			}

			app.latestExecPayload = res

		case <-ctx.Done():
			return

		default:
		}
	}
}

func (app *HeimdallApp) retryBuildLatestPayload(ctx sdk.Context, height int64) (response *engine.Payload, err error) {
	forever := backoff.NewExponentialBackOff()
	latestBlock, err := app.caller.BorChainClient.BlockByNumber(ctx, big.NewInt(height)) // change this to a keeper
	if err != nil {
		return nil, err
	}

	state := engine.ForkChoiceState{
		HeadHash:           latestBlock.Hash(),
		SafeBlockHash:      latestBlock.Hash(),
		FinalizedBlockHash: common.Hash{},
	}

	// The engine complains when the withdrawals are empty
	withdrawals := []*engine.Withdrawal{ // need to undestand
		{
			Index:     "0x0",
			Validator: "0x0",
			Address:   common.Address{}.Hex(),
			Amount:    "0x0",
		},
	}

	addr := common.BytesToAddress(helper.GetPrivKey().PubKey().Address().Bytes())
	attrs := engine.PayloadAttributes{
		Timestamp:             hexutil.Uint64(time.Now().UnixMilli()),
		PrevRandao:            common.Hash{}, // do we need to generate a randao for the EVM?
		SuggestedFeeRecipient: addr,
		Withdrawals:           withdrawals,
	}

	choice, err := app.caller.BorEngineClient.ForkchoiceUpdatedV2(&state, &attrs)
	if err != nil {
		return nil, err
	}

	payloadId := choice.PayloadId
	status := choice.PayloadStatus

	if status.Status != "VALID" {
		// logger.Error("validation err: %v, critical err: %v", status.ValidationError, status.CriticalError)
		return nil, errors.New(status.ValidationError)
	}

	err = backoff.Retry(func() error {
		response, err = app.caller.BorEngineClient.GetPayloadV2(payloadId)
		if forever.NextBackOff() > 1*time.Minute {
			forever.Reset()
		}
		if err != nil {
			return err
		}
		return nil
	}, forever)
	if err != nil {
		return nil, err // should not happen, retries forever
	}

	return response, nil
}

func (app *HeimdallApp) retryBuildNextPayload(ctx sdk.Context, height int64) (response *engine.Payload, err error) {
	forever := backoff.NewExponentialBackOff()
	latestBlock, err := app.caller.BorChainClient.BlockByNumber(ctx, big.NewInt(height)) // change this to a keeper
	if err != nil {
		return nil, err
	}

	state := engine.ForkChoiceState{
		HeadHash:           latestBlock.Hash(),
		SafeBlockHash:      latestBlock.Hash(),
		FinalizedBlockHash: common.Hash{},
	}

	// The engine complains when the withdrawals are empty
	withdrawals := []*engine.Withdrawal{ // need to undestand
		{
			Index:     "0x0",
			Validator: "0x0",
			Address:   common.Address{}.Hex(),
			Amount:    "0x0",
		},
	}

	addr := common.BytesToAddress(helper.GetPrivKey().PubKey().Address().Bytes())
	attrs := engine.PayloadAttributes{
		Timestamp:             hexutil.Uint64(time.Now().UnixMilli()),
		PrevRandao:            common.Hash{}, // do we need to generate a randao for the EVM?
		SuggestedFeeRecipient: addr,
		Withdrawals:           withdrawals,
	}

	choice, err := app.caller.BorEngineClient.ForkchoiceUpdatedV2(&state, &attrs)
	if err != nil {
		return nil, err
	}

	payloadId := choice.PayloadId
	status := choice.PayloadStatus

	if status.Status != "VALID" {
		// logger.Error("validation err: %v, critical err: %v", status.ValidationError, status.CriticalError)
		return nil, errors.New(status.ValidationError)
	}

	err = backoff.Retry(func() error {
		response, err = app.caller.BorEngineClient.GetPayloadV2(payloadId)
		if forever.NextBackOff() > 1*time.Minute {
			forever.Reset()
		}
		if err != nil {
			return err
		}
		return nil
	}, forever)
	if err != nil {
		return nil, err // should not happen, retries forever
	}

	return response, nil
}
