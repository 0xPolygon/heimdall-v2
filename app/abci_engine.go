package app

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/cenkalti/backoff/v4"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gethEngine "github.com/ethereum/go-ethereum/beacon/engine"
	gethTypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum/go-ethereum/common"
)

func (app *HeimdallApp) ProduceELPayload(ctx context.Context) {
	logger := app.Logger()
	var blockCtx nextELBlockCtx
	for {
		select {
		case blockCtx = <-app.nextBlockChan:
			res, err := app.retryBuildNextPayload(blockCtx.ForkchoiceStateV1, blockCtx.context)
			if err != nil {
				logger.Error("error building next payload", "error", err)
				res = nil
			}

			app.nextExecPayload = res

		case blockCtx = <-app.currBlockChan:
			res, err := app.retryBuildLatestPayload(blockCtx.ForkchoiceStateV1, ctx, blockCtx.height)
			if err != nil {
				logger.Error("error building latest payload", "error", err)
				res = nil
			}

			app.latestExecPayload = res

		case <-ctx.Done():
			return
		}
	}
}

func (app *HeimdallApp) retryBuildLatestPayload(state gethEngine.ForkchoiceStateV1, ctx context.Context, height int64) (response *gethEngine.ExecutionPayloadEnvelope, err error) {
	forever := backoff.NewExponentialBackOff()

	ctxTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if state == (gethEngine.ForkchoiceStateV1{}) {
		latestBlock, err := app.caller.BorChainClient.BlockByNumber(ctxTimeout, big.NewInt(height)) // change this to a keeper
		if err != nil {
			return nil, err
		}
		state = gethEngine.ForkchoiceStateV1{
			HeadBlockHash:      latestBlock.Hash(),
			SafeBlockHash:      latestBlock.Hash(),
			FinalizedBlockHash: common.Hash{},
		}
	}

	// The engine complains when the withdrawals are empty
	withdrawals := []*gethTypes.Withdrawal{ // need to undestand
		{
			Index:     0,
			Validator: 0,
			Address:   common.Address{},
			Amount:    0,
		},
	}

	addr := common.BytesToAddress(helper.GetPrivKey().PubKey().Address().Bytes())
	attrs := gethEngine.PayloadAttributes{
		Timestamp:             uint64(time.Now().Unix()),
		Random:                common.Hash{}, // do we need to generate a randao for the EVM?
		SuggestedFeeRecipient: addr,
		Withdrawals:           withdrawals,
	}

	choice, err := app.caller.BorEngineClient.ForkchoiceUpdatedV2(ctxTimeout, &state, &attrs)
	if err != nil {
		return nil, err
	}

	payloadId := choice.PayloadID
	status := choice.PayloadStatus

	if status.Status != "VALID" {
		// logger.Error("validation err: %v, critical err: %v", status.ValidationError, status.CriticalError)
		return nil, errors.New(*status.ValidationError)
	}

	err = backoff.Retry(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		response, err = app.caller.BorEngineClient.GetPayloadV2(ctx, payloadId.String())
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

func (app *HeimdallApp) retryBuildNextPayload(state gethEngine.ForkchoiceStateV1, ctx sdk.Context) (response *gethEngine.ExecutionPayloadEnvelope, err error) {
	forever := backoff.NewExponentialBackOff()

	// The engine complains when the withdrawals are empty
	withdrawals := []*gethTypes.Withdrawal{ // need to undestand
		{
			Index:     0,
			Validator: 0,
			Address:   common.Address{},
			Amount:    0,
		},
	}

	addr := common.BytesToAddress(helper.GetPrivKey().PubKey().Address().Bytes())
	attrs := gethEngine.PayloadAttributes{
		Timestamp:             uint64(time.Now().UnixMilli()),
		Random:                common.Hash{}, // do we need to generate a randao for the EVM?
		SuggestedFeeRecipient: addr,
		Withdrawals:           withdrawals,
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	choice, err := app.caller.BorEngineClient.ForkchoiceUpdatedV2(ctxTimeout, &state, &attrs)
	if err != nil {
		return nil, err
	}

	payloadId := choice.PayloadID
	status := choice.PayloadStatus

	if status.Status != "VALID" {
		// logger.Error("validation err: %v, critical err: %v", status.ValidationError, status.CriticalError)
		return nil, errors.New(*status.ValidationError)
	}

	err = backoff.Retry(func() error {
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		response, err = app.caller.BorEngineClient.GetPayloadV2(ctx, payloadId.String())
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
