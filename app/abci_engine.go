package app

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/0xPolygon/heimdall-v2/helper"
	engineclient "github.com/0xPolygon/heimdall-v2/x/engine/client"
	enginetypes "github.com/0xPolygon/heimdall-v2/x/engine/types"
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
			res, err := app.retryBuildNextPayload(blockCtx.ForkChoiceState, blockCtx.context)
			if err != nil {
				logger.Error("error building next payload", "error", err)
				res = nil
			}

			app.nextExecPayload = res

		case blockCtx = <-app.currBlockChan:
			res, err := app.retryBuildLatestPayload(blockCtx.ForkChoiceState, ctx, blockCtx.height)
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

func (app *HeimdallApp) retryBuildLatestPayload(state engineclient.ForkChoiceState, ctx context.Context, height uint64) (response *engineclient.Payload, err error) {
	forever := backoff.NewExponentialBackOff()

	ctxTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if state == (engineclient.ForkChoiceState{}) {
		latestBlock, err := app.caller.BorChainClient.BlockByNumber(ctxTimeout, big.NewInt(int64(height))) // change this to a keeper
		if err != nil {
			return nil, err
		}
		state = engineclient.ForkChoiceState{
			HeadHash:           latestBlock.Hash(),
			SafeBlockHash:      latestBlock.Hash(),
			FinalizedBlockHash: common.Hash{},
		}
	}

	// The engine complains when the withdrawals are empty
	withdrawals := []*engineclient.Withdrawal{ // need to understand
		{
			Index:     "0x0",
			Validator: "0x0",
			Address:   common.Address{}.Hex(),
			Amount:    "0x0",
		},
	}

	addr := common.BytesToAddress(helper.GetPrivKey().PubKey().Address().Bytes())
	attrs := engineclient.PayloadAttributes{
		Timestamp:             hexutil.Uint64(time.Now().UnixMilli()),
		PrevRandao:            common.Hash{}, // do we need to generate a randao for the EVM?
		SuggestedFeeRecipient: addr,
		Withdrawals:           withdrawals,
	}

	choice, err := app.ExecutionEngineClient.ForkchoiceUpdatedV2(ctxTimeout, &state, &attrs)
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
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		response, err = app.ExecutionEngineClient.GetPayloadV2(ctx, payloadId)
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

func (app *HeimdallApp) retryBuildNextPayload(state engineclient.ForkChoiceState, ctx sdk.Context) (response *engineclient.Payload, err error) {
	forever := backoff.NewExponentialBackOff()

	// The engine complains when the withdrawals are empty
	withdrawals := []*engineclient.Withdrawal{ // need to understand
		{
			Index:     "0x0",
			Validator: "0x0",
			Address:   common.Address{}.Hex(),
			Amount:    "0x0",
		},
	}

	addr := common.BytesToAddress(helper.GetPrivKey().PubKey().Address().Bytes())
	attrs := engineclient.PayloadAttributes{
		Timestamp:             hexutil.Uint64(time.Now().UnixMilli()),
		PrevRandao:            common.Hash{}, // do we need to generate a randao for the EVM?
		SuggestedFeeRecipient: addr,
		Withdrawals:           withdrawals,
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	choice, err := app.ExecutionEngineClient.ForkchoiceUpdatedV2(ctxTimeout, &state, &attrs)
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
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		response, err = app.ExecutionEngineClient.GetPayloadV2(ctx, payloadId)
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

func (app *HeimdallApp) getExecutionStateMetadata(ctx sdk.Context) (enginetypes.ExecutionStateMetadata, error) {
	logger := app.Logger()
	executionState, err := app.EngineKeeper.GetExecutionStateMetadata(ctx)
	if err != nil {
		logger.Warn("execution state not found in the keeper, this should not happen. Fetching from bor chain", "error", err)
		blockNum, err := app.caller.BorChainClient.BlockNumber(ctx)
		if err != nil {
			return enginetypes.ExecutionStateMetadata{}, err
		}

		lastHeader, err := app.caller.BorChainClient.BlockByNumber(ctx, big.NewInt(int64(blockNum)))
		if err != nil {
			return enginetypes.ExecutionStateMetadata{}, err
		}

		executionState = enginetypes.ExecutionStateMetadata{
			FinalBlockHash:    lastHeader.Hash().Bytes(),
			LatestBlockNumber: blockNum,
		}

	}

	return executionState, nil

}
