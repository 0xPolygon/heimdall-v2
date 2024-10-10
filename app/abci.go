package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	cmtTypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/codec/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/ethereum/go-ethereum/common"

	"github.com/0xPolygon/heimdall-v2/sidetxs"
)

// execModePrepareProposal and execModeProcessProposal match the ones defined on cosmos' baseapp
const (
	execModePrepareProposal = 3 // Prepare a block proposal
	execModeProcessProposal = 4 // Process a block proposal
)

// Note: returning any error in ABCI functions will cause cometBFT to panic

// NewPrepareProposalHandler prepares the proposal after validating the vote extensions
func (app *HeimdallApp) NewPrepareProposalHandler() sdk.PrepareProposalHandler {
	return func(ctx sdk.Context, req *abci.RequestPrepareProposal) (*abci.ResponsePrepareProposal, error) {
		logger := app.Logger()

		if err := ValidateVoteExtensions(ctx, req.Height, req.ProposerAddress, req.LocalLastCommit.Votes, req.LocalLastCommit.Round, app.StakeKeeper); err != nil {
			logger.Error("Error occurred while validating VEs in PrepareProposal", err)
			return nil, err
		}

		// prepare the proposal with the vote extensions and the validators set's votes
		var txs [][]byte
		// TODO HV2: shouldn't we be using proto.Marshal instead of json.Marshal?
		bz, err := json.Marshal(req.LocalLastCommit.Votes)
		if err != nil {
			logger.Error("Error occurred while marshaling the ExtendedVoteInfo in prepare proposal", "error", err)
			return nil, err
		}
		txs = append(txs, bz)

		// init totalTxBytes with the actual size of the marshaled vote info in bytes
		totalTxBytes := len(bz)
		for _, proposedTx := range req.Txs {

			// check if the total tx bytes exceed the max tx bytes of the request
			if totalTxBytes+len(proposedTx) > int(req.MaxTxBytes) {
				continue
			}

			// Enforce only one messageType per sideTx. Otherwise, a single comet tx would contain more than one sideTx,
			// allowing for more than one vote for the same tx hash
			tx, err := app.TxDecode(proposedTx)
			if err != nil {
				return nil, fmt.Errorf("error occurred while decoding tx bytes in PrepareProposalHandler. Error: %w", err)
			}
			sideHandlerCount := 0
			for _, msg := range tx.GetMsgs() {
				if sideHandler := app.sideTxCfg.GetSideHandler(msg); sideHandler != nil {
					sideHandlerCount++
				}
			}
			if sideHandlerCount > 1 {
				continue
			}

			// run the tx by executing the msg_server handler on the tx msgs and the ante handler
			_, _, _, err = app.RunTx(execModePrepareProposal, proposedTx)
			if err != nil {
				logger.Error("RunTx returned an error in PrepareProposal", "error", err)
				return nil, err
			}

			totalTxBytes += len(proposedTx)
			txs = append(txs, proposedTx)

			// check if there are more than 2 txs in the request
			if len(txs) > 2 {
				logger.Error(fmt.Sprintf("unexpected behaviour, more than 2 txs proposed by %s", req.ProposerAddress))
				return nil, fmt.Errorf("unexpected behaviour, more than 2 txs proposed by %s", req.ProposerAddress)
			}
			// check if there are less than 1 txs in the request
			if len(txs) < 1 {
				logger.Error(fmt.Sprintf("unexpected behaviour, less than 1 txs proposed by %s", req.ProposerAddress))
				return nil, fmt.Errorf("unexpected behaviour, less than 1 txs proposed by %s", req.ProposerAddress)
			}
		}
		return &abci.ResponsePrepareProposal{Txs: txs}, nil
	}
}

// NewProcessProposalHandler processes the proposal, validates the vote extensions, and reject the proposal in case
// there's no majority. It is implemented by all the validators.
func (app *HeimdallApp) NewProcessProposalHandler() sdk.ProcessProposalHandler {
	return func(ctx sdk.Context, req *abci.RequestProcessProposal) (*abci.ResponseProcessProposal, error) {
		logger := app.Logger()

		// extract the ExtendedVoteInfo from the txs (it is encoded at the beginning, index 0)
		var extVoteInfo []abci.ExtendedVoteInfo
		extendedVoteTx := req.Txs[0]
		// TODO HV2: shouldn't we be using proto.Unmarshal instead of json.Unmarshal?
		if err := json.Unmarshal(extendedVoteTx, &extVoteInfo); err != nil {
			// returning an error here would cause consensus to panic. Reject the proposal instead if a proposer
			// deliberately does not include ExtendedVoteInfo at the beginning of the txs slice
			logger.Error("Error occurred while decoding ExtendedVoteInfo", "error", err)
			return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, nil
		}

		// validate the vote extensions
		if err := ValidateVoteExtensions(ctx, req.Height, req.ProposerAddress, extVoteInfo, req.ProposedLastCommit.Round, app.StakeKeeper); err != nil {
			logger.Error("Invalid vote extension, rejecting proposal", "error", err)
			return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, nil
		}

		// check if there are more than 2 txs in the request
		if len(req.Txs) > 2 {
			logger.Error(fmt.Sprintf("unexpected behaviour, more than 2 txs found in the proposal submitted by %s", req.ProposerAddress))
			return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, nil
		}
		// check if there are any txs in the request
		if len(req.Txs) < 1 {
			logger.Error("unexpected behaviour, no txs found in the proposal")
			return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, nil
		}

		// check if there is only one tx in the request
		if len(req.Txs) == 1 {
			return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_ACCEPT}, nil
		}

		for _, tx := range req.Txs[1:] {

			// Enforce only one messageType per sideTx. Otherwise, a single comet tx would contain more than one sideTx,
			// allowing for more than one vote for the same tx hash
			txn, err := app.TxDecode(tx)
			if err != nil {
				return nil, fmt.Errorf("error occurred while decoding tx bytes in PrepareProposalHandler. Error: %w", err)
			}
			sideHandlerCount := 0
			for _, msg := range txn.GetMsgs() {
				if sideHandler := app.sideTxCfg.GetSideHandler(msg); sideHandler != nil {
					sideHandlerCount++
				}
			}
			if sideHandlerCount > 1 {
				continue
			}

			// run the tx by executing the msg_server handler on the tx msgs and the ante handler
			_, _, _, err = app.RunTx(execModeProcessProposal, tx)
			if err != nil {
				// this should never happen, as the txs have already been checked in PrepareProposal
				logger.Error("RunTx returned an error in ProcessProposal", "error", err)
				return nil, err
			}
		}

		return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_ACCEPT}, nil
	}
}

// ExtendVoteHandler extends pre-commit vote
func (app *HeimdallApp) ExtendVoteHandler() sdk.ExtendVoteHandler {
	return func(ctx sdk.Context, req *abci.RequestExtendVote) (*abci.ResponseExtendVote, error) {
		logger := app.Logger()
		logger.Debug("Extending Vote!", "height", ctx.BlockHeight())

		// check if VEs are enabled
		panicOnVoteExtensionsDisabled(ctx, req.Height)

		// prepare the side tx responses
		sideTxRes := make([]*sidetxs.SideTxResponse, 0)

		// extract the ExtendedVoteInfo from the txs (it is encoded at the beginning, index 0)
		var extVoteInfos []abci.ExtendedVoteInfo

		// check whether ExtendedVoteInfo is encoded at the beginning
		bz := req.Txs[0]
		// TODO HV2: shouldn't we be using proto.Unmarshal instead of json.Unmarshal?
		if err := json.Unmarshal(bz, &extVoteInfos); err != nil {
			logger.Error("Error occurred while decoding ExtendedVoteInfo", "error", err)
			// abnormal behavior since the block got >2/3 pre-votes, so the special tx should have been added
			panic("error occurred while decoding ExtendedVoteInfos, they should have be encoded in the beginning of txs slice")
		}

		txs := req.Txs[1:]

		// decode txs and execute side txs
		for _, rawTx := range txs {
			// create a cache wrapped context for stateless execution
			ctx, _ = app.cacheTxContext(ctx, rawTx)
			tx, err := app.TxDecode(rawTx)
			if err != nil {
				// This tx comes from a block that has already been pre-voted by >2/3 of the voting power, so this should never happen
				panic(fmt.Errorf("error occurred while decoding tx bytes in ExtendVoteHandler. Error: %w", err))
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

				// add the side handler results (YES/NO/UNSPECIFIED votes) to the side tx response
				txHash := cmtTypes.Tx(rawTx).Hash()
				logger.Debug("Adding vote extension", "txHash", txHash, "blockHeight", req.Height, "blockHash", req.Hash, "vote", res)
				ve := sidetxs.SideTxResponse{
					TxHash: txHash,
					Result: res,
				}
				sideTxRes = append(sideTxRes, &ve)
			}

		}

		// prepare the response with votes, block height and block hash
		consolidatedSideTxRes := sidetxs.ConsolidatedSideTxResponse{
			SideTxResponses: sideTxRes,
			Height:          req.Height,
			BlockHash:       req.Hash,
		}

		bz, err := proto.Marshal(&consolidatedSideTxRes)
		if err != nil {
			logger.Error("Error occurred while marshalling the VoteExtension in ExtendVoteHandler", "error", err)
			return nil, err
		}

		return &abci.ResponseExtendVote{VoteExtension: bz}, nil
	}
}

// VerifyVoteExtensionHandler performs some sanity checks on the VE received from other validators
func (app *HeimdallApp) VerifyVoteExtensionHandler() sdk.VerifyVoteExtensionHandler {
	return func(ctx sdk.Context, req *abci.RequestVerifyVoteExtension) (*abci.ResponseVerifyVoteExtension, error) {
		logger := app.Logger()
		logger.Debug("Verifying vote extension", "height", ctx.BlockHeight())

		// check if VEs are enabled
		panicOnVoteExtensionsDisabled(ctx, req.Height)

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

		return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_ACCEPT}, nil
	}
}

// PreBlocker application updates every pre block
func (app *HeimdallApp) PreBlocker(ctx sdk.Context, req *abci.RequestFinalizeBlock) (*sdk.ResponsePreBlock, error) {
	logger := app.Logger()

	panicOnVoteExtensionsDisabled(ctx, req.Height+1)

	// Extract ExtendedVoteInfo encoded at the beginning of txs bytes
	var extVoteInfo []abci.ExtendedVoteInfo

	// req.Txs must have non-zero length
	if len(req.Txs) == 0 {
		logger.Error("Unexpected behavior, no txs found in the pre-blocker", "height", req.Height)
		panic(fmt.Sprintf("no txs found in the pre-blocker at height %d", req.Height))
	}

	bz := req.Txs[0]
	// TODO HV2: shouldn't we be using proto.Unmarshal instead of json.Unmarshal?
	if err := json.Unmarshal(bz, &extVoteInfo); err != nil {
		logger.Error("Error occurred while unmarshalling ExtendedVoteInfo", "error", err)
		return nil, err
	}

	if req.Height == retrieveVoteExtensionsEnableHeight(ctx) {
		if len(extVoteInfo) != 0 {
			logger.Error("Unexpected behavior, non-empty VEs found in the initial height's pre-blocker", "height", req.Height)
			return nil, errors.New("non-empty VEs found in the initial height's pre-blocker")
		}
		return app.ModuleManager.PreBlock(ctx)
	}

	// Fetch txs from block n-1 so that we can match them with the approved txs in block n to execute sideTxs
	lastBlockTxs, err := app.StakeKeeper.GetLastBlockTxs(ctx, req.Height)
	if err != nil {
		logger.Error("Error occurred while fetching last block txs", "error", err)
		return nil, err
	}
	// update last block txs
	err = app.StakeKeeper.SetLastBlockTxs(ctx, req.Height, req.Txs[1:])
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

	// execute side txs
	txs := lastBlockTxs.Txs

	if len(req.Txs) > 1 {

		for _, rawTx := range txs {
			decodedTx, err := app.TxDecode(rawTx)
			if err != nil {
				logger.Error("Error occurred while decoding tx bytes", "error", err)
				return nil, err
			}

			var txBytes cmtTypes.Tx = rawTx

			for _, approvedTx := range approvedTxs {

				if bytes.Equal(approvedTx, txBytes.Hash()) {

					// execute post handlers for the approved side txs' msgs
					msgs := decodedTx.GetMsgs()
					for _, msg := range msgs {
						postHandler := app.sideTxCfg.GetPostHandler(msg)
						if postHandler != nil {
							postHandler(ctx, msg, sidetxs.Vote_VOTE_YES)
						}
					}

				}
			}
		}
	}

	return app.ModuleManager.PreBlock(ctx)
}
