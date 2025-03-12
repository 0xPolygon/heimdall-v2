package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	cmtTypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/codec/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/0xPolygon/heimdall-v2/engine"
	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/sidetxs"
	borTypes "github.com/0xPolygon/heimdall-v2/x/bor/types"
	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
	checkpointTypes "github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
)

type HeimdallMetadata struct {
	MarshaledLocalLastCommit  []byte `json:"marshaledLocalLastCommit"`
	MarshaledExecutionPayload []byte `json:"marshaledExecutionPayload"`
}

// NewPrepareProposalHandler prepares the proposal after validating the vote extensions
func (app *HeimdallApp) NewPrepareProposalHandler() sdk.PrepareProposalHandler {
	return func(ctx sdk.Context, req *abci.RequestPrepareProposal) (*abci.ResponsePrepareProposal, error) {
		logger := app.Logger()
		start := time.Now()
		logger.Info("🕒 Start PrepareProposal:", "height", req.Height, "momentTime", time.Now().Format("04:05.000000"))

		if err := ValidateVoteExtensions(ctx, req.Height, req.ProposerAddress, req.LocalLastCommit.Votes, req.LocalLastCommit.Round, app.StakeKeeper); err != nil {
			logger.Error("Error occurred while validating VEs in PrepareProposal", err)
			return nil, err
		}

		if err := ValidateNonRpVoteExtensions(ctx, req.Height, req.LocalLastCommit.Votes, app.StakeKeeper, app.ChainManagerKeeper, app.CheckpointKeeper, &app.caller, logger); err != nil {
			logger.Error("Error occurred while validating non-rp VEs in PrepareProposal", err)
			if !errors.Is(err, borTypes.ErrFailedToQueryBor) {
				return nil, err
			}
		}

		// prepare the proposal with the vote extensions and the validators set's votes
		var txs [][]byte
		marshaledLocalLastCommit, err := req.LocalLastCommit.Marshal()
		if err != nil {
			logger.Error("Error occurred while marshaling the LocalLastCommit in prepare proposal", "error", err)
			return nil, err
		}

		// Engine API
		var payload *engine.Payload

		// TODO: store bor block height in a keeper
		if app.latestExecPayload != nil && app.latestExecPayload.ExecutionPayload.BlockNumber == hexutil.EncodeUint64(uint64(req.Height)) {
			payload = app.latestExecPayload
		} else if app.nextExecPayload != nil && app.nextExecPayload.ExecutionPayload.BlockNumber == hexutil.EncodeUint64(uint64(req.Height)) {
			payload = app.nextExecPayload
		} else {
			logger.Debug("latest payload not found, fetching from CheckpointKeeper")

			executionState, err := app.CheckpointKeeper.GetExecutionStateMetadata(ctx)
			if err != nil {
				logger.Error("Error occurred while fetching latest execution metadata", "error", err)
				return nil, err
			}
			blockHash := common.BytesToHash(executionState.FinalBlockHash)

			state := engine.ForkChoiceState{
				HeadHash:           blockHash,
				SafeBlockHash:      blockHash,
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

			ctx, cancelFunc := context.WithTimeout(ctx, 10*time.Second)
			defer cancelFunc()
			choice, err := app.caller.BorEngineClient.ForkchoiceUpdatedV2(ctx, &state, &attrs)
			if err != nil {
				return nil, err
			}

			payloadId := choice.PayloadId
			status := choice.PayloadStatus

			if status.Status != "VALID" {
				logger.Error("validation err: %v, critical err: %v", status.ValidationError, status.CriticalError)
				return nil, errors.New(status.ValidationError)
			}

			payload, err = app.caller.BorEngineClient.GetPayloadV2(ctx, payloadId)
			if err != nil {
				return nil, err
			}
		}

		// this is where we could filter/reorder transactions, or mark them for filtering so consensus could be checked

		marshaledExecutionPayload, err := json.Marshal(payload.ExecutionPayload)
		if err != nil {
			return nil, err
		}

		metadata := HeimdallMetadata{
			MarshaledLocalLastCommit:  marshaledLocalLastCommit,
			MarshaledExecutionPayload: marshaledExecutionPayload,
		}
		marshaledMetadata, err := json.Marshal(metadata)
		if err != nil {
			return nil, err
		}

		txs = append(txs, marshaledMetadata)

		// init totalTxBytes with the actual size of the marshaled vote info in bytes
		totalTxBytes := len(marshaledLocalLastCommit)
		for _, proposedTx := range req.Txs {

			// check if the total tx bytes exceed the max tx bytes of the request
			if totalTxBytes+len(proposedTx) > int(req.MaxTxBytes) {
				continue
			}

			tx, err := app.TxDecode(proposedTx)
			if err != nil {
				return nil, fmt.Errorf("error occurred while decoding tx bytes in PrepareProposalHandler. Error: %w", err)
			}

			// ensure we allow transactions with only one side msg inside
			if sidetxs.CountSideHandlers(app.sideTxCfg, tx) > 1 {
				continue
			}

			// run the tx by executing the msg_server handler on the tx msgs and the ante handler
			_, err = app.PrepareProposalVerifyTx(tx)
			if err != nil {
				logger.Error("RunTx returned an error in PrepareProposal", "error", err)
				continue
			}

			totalTxBytes += len(proposedTx)
			txs = append(txs, proposedTx)
		}

		duration := time.Since(start)
		formatted := fmt.Sprintf("%.6fms", float64(duration)/float64(time.Millisecond))
		logger.Info("🕒 End PrepareProposal:", "duration", formatted, "height", req.Height, "payloadSize", len(txs[0]), "momentTime", time.Now().Format("04:05.000000"))

		return &abci.ResponsePrepareProposal{Txs: txs}, nil
	}
}

// NewProcessProposalHandler processes the proposal, validates the vote extensions, and reject the proposal in case
// there's no majority. It is implemented by all the validators.
func (app *HeimdallApp) NewProcessProposalHandler() sdk.ProcessProposalHandler {
	return func(ctx sdk.Context, req *abci.RequestProcessProposal) (*abci.ResponseProcessProposal, error) {
		logger := app.Logger()

		start := time.Now()
		logger.Info("🕒 Start ProcessProposal:", "height", req.Height, "momentTime", time.Now().Format("04:05.000000"))

		// check if there are any txs in the request
		if len(req.Txs) < 1 {
			logger.Error("unexpected behaviour, no txs found in the proposal")
			app.currBlockChan <- nextELBlockCtx{height: req.Height, context: ctx}
			return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, nil
		}

		var metadata HeimdallMetadata
		err := json.Unmarshal(req.Txs[0], &metadata)
		if err != nil {
			logger.Error("failed to decode metadata, cannot proceed", "error", err)
			return nil, err
		}

		// extract the ExtendedCommitInfo from the txs (it is encoded at the beginning, index 0)
		extCommitInfo := new(abci.ExtendedCommitInfo)
		extendedCommitTx := metadata.MarshaledLocalLastCommit
		if err := extCommitInfo.Unmarshal(extendedCommitTx); err != nil {
			app.currBlockChan <- nextELBlockCtx{height: req.Height, context: ctx}
			logger.Error("Error occurred while decoding ExtendedCommitInfo", "height", req.Height, "error", err)
			return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, nil
		}

		if extCommitInfo.Round != req.ProposedLastCommit.Round {
			app.currBlockChan <- nextELBlockCtx{height: req.Height, context: ctx}
			logger.Error("Received commit round does not match expected round", "expected", req.ProposedLastCommit.Round, "got", extCommitInfo.Round)
			return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, nil
		}

		// validate the vote extensions
		if err := ValidateVoteExtensions(ctx, req.Height, req.ProposerAddress, extCommitInfo.Votes, req.ProposedLastCommit.Round, app.StakeKeeper); err != nil {
			app.currBlockChan <- nextELBlockCtx{height: req.Height, context: ctx}
			logger.Error("Invalid vote extension, rejecting proposal", "error", err)
			return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, nil
		}

		if err := ValidateNonRpVoteExtensions(ctx, req.Height, extCommitInfo.Votes, app.StakeKeeper, app.ChainManagerKeeper, app.CheckpointKeeper, &app.caller, logger); err != nil {
			// We could reject proposal if we fail to query bor, we follow RFC 105 (https://github.com/cometbft/cometbft/blob/main/docs/references/rfc/rfc-105-non-det-process-proposal.md)
			if errors.Is(err, borTypes.ErrFailedToQueryBor) {
				logger.Error("Failed to query bor, rejecting proposal", "error", err)
			} else {
				logger.Error("Invalid non-rp vote extension, rejecting proposal", "error", err)
			}

			app.currBlockChan <- nextELBlockCtx{height: req.Height, context: ctx}
			return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, nil
		}

		for _, tx := range req.Txs[1:] {

			txn, err := app.TxDecode(tx)
			if err != nil {
				app.currBlockChan <- nextELBlockCtx{height: req.Height, context: ctx}
				logger.Error("error occurred while decoding tx bytes in ProcessProposalHandler", err)
				return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, nil
			}

			// ensure we allow transactions with only one side msg inside
			if sidetxs.CountSideHandlers(app.sideTxCfg, txn) > 1 {
				app.currBlockChan <- nextELBlockCtx{height: req.Height, context: ctx}
				return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, nil
			}

			// run the tx by executing the msg_server handler on the tx msgs and the ante handler
			_, err = app.ProcessProposalVerifyTx(tx)
			if err != nil {
				app.currBlockChan <- nextELBlockCtx{height: req.Height, context: ctx}
				// this should never happen, as the txs have already been checked in PrepareProposal
				logger.Error("RunTx returned an error in ProcessProposal", "error", err)
				return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, nil
			}
		}

		// Engine API - Validate block
		var executionPayload engine.ExecutionPayload
		err = json.Unmarshal(metadata.MarshaledExecutionPayload, &executionPayload)
		if err != nil {
			// TODO: use forkchoice state from the latest block stored in the keeper
			app.currBlockChan <- nextELBlockCtx{height: req.Height, context: ctx}
			logger.Error("failed to decode execution payload, cannot proceed", "error", err)
			return nil, err
		}
		payload, err := app.caller.BorEngineClient.NewPayloadV2(ctx, executionPayload)
		if err != nil {
			// TODO: use forkchoice state from the latest block stored in the keeper
			app.currBlockChan <- nextELBlockCtx{height: req.Height, context: ctx}
			logger.Error("failed to validate execution payload on execution client, cannot proceed", "error", err)
			return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, nil
		}

		if payload.Status != "VALID" {
			// TODO: use forkchoice state from the latest block stored in the keeper
			app.currBlockChan <- nextELBlockCtx{height: req.Height, context: ctx}
			logger.Error("execution payload is not valid, cannot proceed", "error", err)
			return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, nil
		}

		// app.execPayloadCache.Add(payload.LatestValidHash, payload)
		duration := time.Since(start)
		formatted := fmt.Sprintf("%.6fms", float64(duration)/float64(time.Millisecond))
		logger.Info("🕒 ProcessProposal:" + formatted)

		app.nextBlockChan <- nextELBlockCtx{height: req.Height + 1,
			context: ctx,
			ForkChoiceState: engine.ForkChoiceState{
				HeadHash:           common.HexToHash(executionPayload.BlockHash),
				SafeBlockHash:      common.HexToHash(executionPayload.BlockHash),
				FinalizedBlockHash: common.Hash{},
			},
		}
		return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_ACCEPT}, nil
	}
}

// ExtendVoteHandler extends pre-commit vote
func (app *HeimdallApp) ExtendVoteHandler() sdk.ExtendVoteHandler {
	return func(ctx sdk.Context, req *abci.RequestExtendVote) (*abci.ResponseExtendVote, error) {
		logger := app.Logger()
		logger.Debug("Extending Vote!", "height", ctx.BlockHeight())
		start := time.Now()
		logger.Info("🕒 Start ExtendVote:", "height", req.Height, "momentTime", time.Now().Format("04:05.000000"))

		// check if VEs are enabled
		if err := checkIfVoteExtensionsDisabled(ctx, req.Height); err != nil {
			return nil, err
		}

		var metadata HeimdallMetadata
		err := json.Unmarshal(req.Txs[0], &metadata)
		if err != nil {
			logger.Error("failed to decode metadata, cannot proceed", "error", err)
			return nil, err
		}

		// prepare the side tx responses
		sideTxRes := make([]sidetxs.SideTxResponse, 0)

		// extract the ExtendedVoteInfo from the txs (it is encoded at the beginning, index 0)
		extCommitInfo := new(abci.ExtendedCommitInfo)

		// check whether ExtendedVoteInfo is encoded at the beginning
		bz := metadata.MarshaledLocalLastCommit
		if err := extCommitInfo.Unmarshal(bz); err != nil {
			logger.Error("Error occurred while decoding ExtendedCommitInfo", "error", err)
			// abnormal behavior since the block got >2/3 pre-votes, so the special tx should have been added
			return nil, errors.New("error occurred while decoding ExtendedCommitInfo, they should have be encoded in the beginning of txs slice")
		}

		dummyVoteExt, err := getDummyNonRpVoteExtension(req.Height, ctx.ChainID())
		if err != nil {
			logger.Error("Error occurred while getting dummy vote extension", "error", err)
			return nil, err
		}

		nonRpVoteExt := dummyVoteExt

		txs := req.Txs[1:]

		// decode txs and execute side txs
		for _, rawTx := range txs {
			// create a cache wrapped context for stateless execution
			ctx, _ = app.cacheTxContext(ctx)
			tx, err := app.TxDecode(rawTx)
			if err != nil {
				// This tx comes from a block that has already been pre-voted by >2/3 of the voting power, so this should never happen
				return nil, fmt.Errorf("error occurred while decoding tx bytes in ExtendVoteHandler. Error: %w", err)
			}

			// messages represent the side txs (operations performed by modules using the VEs mechanism)
			// e.g. bor, checkpoint, clerk, milestone, stake and topup
			messages := tx.GetMsgs()
			for _, msg := range messages {
				// get the right module's side handler for the message
				sideHandler := app.sideTxCfg.GetSideHandler(msg)
				if sideHandler == nil {
					logger.Debug("No side handler found for the message", "msg", msg)
					continue
				}

				// execute the side handler to collect the votes from the validators
				res := sideHandler(ctx, msg)

				if res == sidetxs.Vote_VOTE_YES && checkpointTypes.IsCheckpointMsg(msg) {
					checkpointMsg, ok := msg.(*types.MsgCheckpoint)
					if !ok {
						logger.Error("ExtendVoteHandler: type mismatch for MsgCheckpoint")
						continue
					}

					nonRpVoteExt = packExtensionWithVote(checkpointMsg.GetSideSignBytes())
				}

				// add the side handler results (YES/NO/UNSPECIFIED votes) to the side tx response
				txHash := cmtTypes.Tx(rawTx).Hash()
				logger.Debug("Adding vote extension", "txHash", txHash, "blockHeight", req.Height, "blockHash", req.Hash, "vote", res)
				ve := sidetxs.SideTxResponse{
					TxHash: txHash,
					Result: res,
				}
				sideTxRes = append(sideTxRes, ve)
			}
		}

		// prepare the response with votes, block height and block hash
		consolidatedSideTxRes := sidetxs.ConsolidatedSideTxResponse{
			SideTxResponses: sideTxRes,
			Height:          req.Height,
			BlockHash:       req.Hash,
		}

		bz, err = consolidatedSideTxRes.Marshal()
		if err != nil {
			logger.Error("Error occurred while marshalling the ConsolidatedSideTxResponse in ExtendVoteHandler", "error", err)
			return nil, err
		}

		if err := ValidateNonRpVoteExtension(ctx, req.Height, nonRpVoteExt, app.ChainManagerKeeper, app.CheckpointKeeper, &app.caller); err != nil {
			logger.Error("Error occurred while validating non-rp vote extension", "error", err)
			if errors.Is(err, borTypes.ErrFailedToQueryBor) {
				return &abci.ResponseExtendVote{VoteExtension: bz, NonRpExtension: dummyVoteExt}, nil
			}
			return nil, err
		}

		duration := time.Since(start)
		formatted := fmt.Sprintf("%.6fms", float64(duration)/float64(time.Millisecond))
		logger.Info("🕒 End ExtendVote:", "duration", formatted, "height", req.Height, "payloadSize", len(req.Txs[0]), "momentTime", time.Now().Format("04:05.000000"))
		return &abci.ResponseExtendVote{VoteExtension: bz, NonRpExtension: nonRpVoteExt}, nil
	}
}

// VerifyVoteExtensionHandler performs some sanity checks on the VE received from other validators
func (app *HeimdallApp) VerifyVoteExtensionHandler() sdk.VerifyVoteExtensionHandler {
	return func(ctx sdk.Context, req *abci.RequestVerifyVoteExtension) (*abci.ResponseVerifyVoteExtension, error) {
		logger := app.Logger()
		logger.Debug("Verifying vote extension", "height", ctx.BlockHeight())
		start := time.Now()
		logger.Info("🕒 Start VerifyVote:", "height", req.Height, "momentTime", time.Now().Format("04:05.000000"))

		// check if VEs are enabled
		if err := checkIfVoteExtensionsDisabled(ctx, req.Height); err != nil {
			return nil, err
		}

		ac := address.NewHexCodec()
		valAddr, err := ac.BytesToString(req.ValidatorAddress)
		if err != nil {
			return nil, err
		}

		var consolidatedSideTxResponse sidetxs.ConsolidatedSideTxResponse
		if err := proto.Unmarshal(req.VoteExtension, &consolidatedSideTxResponse); err != nil {
			logger.Error("ALERT, VOTE EXTENSION REJECTED. THIS SHOULD NOT HAPPEN; THE VALIDATOR COULD BE MALICIOUS! Error while unmarshalling VoteExtension", "validator", valAddr, "error", err)
			return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_REJECT}, nil
		}

		// ensure block height and hash match
		if req.Height != consolidatedSideTxResponse.Height {
			logger.Error("ALERT, VOTE EXTENSION REJECTED. THIS SHOULD NOT HAPPEN; THE VALIDATOR COULD BE MALICIOUS!", "block height", req.Height, "consolidatedSideTxResponse height", consolidatedSideTxResponse.Height, "validator", valAddr)
			return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_REJECT}, nil
		}

		if !bytes.Equal(req.Hash, consolidatedSideTxResponse.BlockHash) {
			logger.Error("ALERT, VOTE EXTENSION REJECTED. THIS SHOULD NOT HAPPEN; THE VALIDATOR COULD BE MALICIOUS!", "block hash", common.Bytes2Hex(req.Hash), "consolidatedSideTxResponse blockHash", common.Bytes2Hex(consolidatedSideTxResponse.BlockHash), "validator", valAddr)
			return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_REJECT}, nil
		}

		// check for duplicate votes
		txHash, err := validateSideTxResponses(consolidatedSideTxResponse.SideTxResponses)
		if err != nil {
			logger.Error("ALERT, VOTE EXTENSION REJECTED. THIS SHOULD NOT HAPPEN; THE VALIDATOR COULD BE MALICIOUS!", "validator", valAddr, "tx hash", common.Bytes2Hex(txHash), "error", err)
			return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_REJECT}, nil
		}

		if err := ValidateNonRpVoteExtension(ctx, req.Height, req.NonRpVoteExtension, app.ChainManagerKeeper, app.CheckpointKeeper, &app.caller); err != nil {
			logger.Error("ALERT, VOTE EXTENSION REJECTED. THIS SHOULD NOT HAPPEN; THE VALIDATOR COULD BE MALICIOUS!", "validator", valAddr, "error", err)
			if !errors.Is(err, borTypes.ErrFailedToQueryBor) {
				return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_REJECT}, nil
			}
		}

		duration := time.Since(start)
		formatted := fmt.Sprintf("%.6fms", float64(duration)/float64(time.Millisecond))
		logger.Info("🕒 End VerifyVote:", "duration", formatted, "height", req.Height, "momentTime", time.Now().Format("04:05.000000"))
		return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_ACCEPT}, nil
	}
}

// PreBlocker application updates every pre block
func (app *HeimdallApp) PreBlocker(ctx sdk.Context, req *abci.RequestFinalizeBlock) (*sdk.ResponsePreBlock, error) {
	logger := app.Logger()

	start := time.Now()
	logger.Info("🕒 Start PreBlocker:", "height", req.Height, "momentTime", time.Now().Format("04:05.000000"))

	if err := checkIfVoteExtensionsDisabled(ctx, req.Height+1); err != nil {
		return nil, err
	}

	// Extract ExtendedVoteInfo encoded at the beginning of txs bytes
	extCommitInfo := new(abci.ExtendedCommitInfo)

	// req.Txs must have non-zero length
	if len(req.Txs) == 0 {
		logger.Error("Unexpected behavior, no txs found in the pre-blocker", "height", req.Height)
		return nil, fmt.Errorf("no txs found in the pre-blocker at height %d", req.Height)
	}

	var metadata HeimdallMetadata
	err := json.Unmarshal(req.Txs[0], &metadata)
	if err != nil {
		logger.Error("failed to decode metadata, cannot proceed", "error", err)
		return nil, err
	}

	bz := metadata.MarshaledLocalLastCommit
	if err := extCommitInfo.Unmarshal(bz); err != nil {
		logger.Error("Error occurred while unmarshalling ExtendedCommitInfo", "error", err)
		return nil, err
	}

	// Engine API
	var executionPayload engine.ExecutionPayload
	err = json.Unmarshal(metadata.MarshaledExecutionPayload, &executionPayload)
	if err != nil {
		logger.Error("failed to decode execution payload, cannot proceed", "error", err)
		return nil, err
	}

	state := engine.ForkChoiceState{
		HeadHash:           common.HexToHash(executionPayload.BlockHash),
		SafeBlockHash:      common.HexToHash(executionPayload.BlockHash),
		FinalizedBlockHash: common.HexToHash(executionPayload.BlockHash), // latestHash from the Proposal stage
	}

	var choice *engine.ForkchoiceUpdatedResponse
	choice, err = app.caller.BorEngineClient.ForkchoiceUpdatedV2(ctx, &state, nil)
	if err != nil {
		infoLog := "fork choice failed, cannot proceed"
		logger.Error(infoLog, err.Error())
		if choice != nil && choice.PayloadStatus.Status != "VALID" {
			infoLog = fmt.Sprintf("%s: %s", infoLog, choice.PayloadStatus.ValidationError)
		}
		return nil, err
	}

	if err := app.CheckpointKeeper.SetExecutionStateMetadata(ctx, checkpointTypes.ExecutionStateMetadata{
		FinalBlockHash: common.Hex2Bytes(executionPayload.BlockHash),
	}); err != nil {
		logger.Error("Error occurred while setting execution state metadata", "error", err)
		return nil, err
	}

	extVoteInfo := extCommitInfo.Votes

	if req.Height <= retrieveVoteExtensionsEnableHeight(ctx) {
		if len(extVoteInfo) != 0 {
			logger.Error("Unexpected behavior, non-empty VEs found in the initial height's pre-blocker", "height", req.Height)
			return nil, errors.New("non-empty VEs found in the initial height's pre-blocker")
		}
		return app.ModuleManager.PreBlock(ctx)
	}

	// Fetch txs from block n-1 so that we can match them with the approved txs in block n to execute sideTxs
	lastBlockTxs, err := app.StakeKeeper.GetLastBlockTxs(ctx)
	if err != nil {
		logger.Error("Error occurred while fetching last block txs", "error", err)
		return nil, err
	}
	// update last block txs
	err = app.StakeKeeper.SetLastBlockTxs(ctx, req.Txs[1:])
	if err != nil {
		logger.Error("Error occurred while setting last block txs", "error", err)
		return nil, err
	}

	validators, err := app.StakeKeeper.GetPreviousBlockValidatorSet(ctx)
	if err != nil {
		return nil, err
	}
	if len(validators.Validators) == 0 {
		return nil, errors.New("no validators found")
	}

	// tally votes
	approvedTxs, _, _, err := tallyVotes(extVoteInfo, logger, validators.GetTotalVotingPower(), req.Height)
	if err != nil {
		logger.Error("Error occurred while tallying votes", "error", err)
		return nil, err
	}

	approvedTxsMap := make(map[string]bool)
	for _, tx := range approvedTxs {
		approvedTxsMap[common.Bytes2Hex(tx)] = true
	}

	txs := lastBlockTxs.Txs

	majorityExt, err := getMajorityNonRpVoteExtension(ctx, extVoteInfo, app.StakeKeeper, logger)
	if err != nil {
		logger.Error("Error occurred while getting majority non-rp vote extension", "error", err)
		return nil, err
	}

	checkpointTxHash := findCheckpointTx(txs, majorityExt[1:], app, logger) // skip first byte because its the vote
	if approvedTxsMap[checkpointTxHash] {
		signatures := getCheckpointSignatures(majorityExt, extVoteInfo)
		if err := app.CheckpointKeeper.SetCheckpointSignaturesTxHash(ctx, checkpointTxHash); err != nil {
			logger.Error("Error occurred while setting checkpoint signatures tx hash", "error", err)
			return nil, err
		}
		if err := app.CheckpointKeeper.SetCheckpointSignatures(ctx, signatures); err != nil {
			logger.Error("Error occurred while setting checkpoint signatures", "error", err)
			return nil, err
		}
	}

	// execute side txs
	for _, rawTx := range txs {
		decodedTx, err := app.TxDecode(rawTx)
		if err != nil {
			logger.Error("Error occurred while decoding tx bytes", "error", err)
			return nil, err
		}

		var txBytes cmtTypes.Tx = rawTx

		if approvedTxsMap[common.Bytes2Hex(txBytes.Hash())] {

			// execute post handler for the approved side tx
			msgs := decodedTx.GetMsgs()
			executedPostHandlers := 0
			for _, msg := range msgs {
				postHandler := app.sideTxCfg.GetPostHandler(msg)
				if postHandler != nil {
					// Create a new context based off of the existing context with a cache wrapped
					// multi-store in case message processing fails.
					postHandlerCtx, msCache := app.cacheTxContext(ctx)
					postHandlerCtx = postHandlerCtx.WithTxBytes(txBytes.Hash())
					if err := postHandler(postHandlerCtx, msg, sidetxs.Vote_VOTE_YES); err == nil {
						msCache.Write()
					}

					executedPostHandlers++
				}

				// make sure only one post handler is executed
				if executedPostHandlers > 0 {
					break
				}
			}

		}
	}
	duration := time.Since(start)
	formatted := fmt.Sprintf("%.6fms", float64(duration)/float64(time.Millisecond))
	logger.Info("🕒 End PreBlocker:", "duration", formatted, "height", req.Height, "payloadSize", len(req.Txs[0]), "momentTime", time.Now().Format("04:05.000000"))
	return app.ModuleManager.PreBlock(ctx)
}
