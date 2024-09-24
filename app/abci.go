package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/0xPolygon/heimdall-v2/sidetxs"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtTypes "github.com/cometbft/cometbft/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/ethereum/go-ethereum/common"
)

// Note: returning any error in ABCI functions will cause cometBFT to panic

// NewPrepareProposalHandler prepares the proposal after validating the vote extensions
func (app *HeimdallApp) NewPrepareProposalHandler() sdk.PrepareProposalHandler {
	return func(ctx sdk.Context, req *abci.RequestPrepareProposal) (*abci.ResponsePrepareProposal, error) {
		logger := app.Logger()

		if err := ValidateVoteExtensions(ctx, req.Height-1, req.ProposerAddress, req.LocalLastCommit.Votes, req.LocalLastCommit.Round, app.StakeKeeper); err != nil {
			logger.Error("Error occurred while validating VEs in PrepareProposal", err)
			return nil, err
		}

		// prepare the proposal with the vote extensions and the validators set's votes
		var txs [][]byte
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
			if totalTxBytes > int(req.MaxTxBytes) {
				break
			}

			// check the txs via the AnteHandler
			res, err := checkTx(app, proposedTx)
			if err != nil {
				// log the error and skip the tx, it won't be included in the proposal, and will stay in the mempool
				logger.Error("checkTx returned an error in PrepareProposal, skipping this tx", "error", err)
				continue
			}
			if res.IsErr() {
				// log the response and skip the tx, it won't be included in the proposal, and will stay in the mempool
				logger.Error("checkTx response is not ok in PrepareProposal, skipping this tx", "responseLog", res.Log)
			}

			totalTxBytes += len(proposedTx)
			txs = append(txs, proposedTx)
		}
		return &abci.ResponsePrepareProposal{Txs: txs}, nil
	}
}

// NewProcessProposalHandler processes the proposal, validates the vote extensions, and reject the proposal in case
// there's no majority. It is implemented by all the validators.
func (app *HeimdallApp) NewProcessProposalHandler() sdk.ProcessProposalHandler {
	return func(ctx sdk.Context, req *abci.RequestProcessProposal) (*abci.ResponseProcessProposal, error) {
		logger := app.Logger()

		// check if there are any txs in the request
		if len(req.Txs) < 1 {
			logger.Error("unexpected behaviour, no txs found in the proposal")
			return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, nil
		}

		for _, tx := range req.Txs[1:] {
			// check the txs via the AnteHandler
			// skip the first tx as it contains the ExtendedVoteInfo and may not have an AnteHandler
			res, err := checkTx(app, tx)
			if err != nil {
				// this should never happen, as the txs have already been checked in PrepareProposal
				logger.Error("checkTx returned an error in ProcessProposal, skipping this tx", "error", err)
				return nil, err
			}
			if res.IsErr() {
				logger.Error("checkTx response is not ok in ProcessProposal, rejecting this tx", "responseLog", res.Log)
				return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, nil
			}
		}

		// extract the ExtendedVoteInfo from the txs (it is encoded at the beginning, index 0)
		var extVoteInfo []abci.ExtendedVoteInfo
		extendedVoteTx := req.Txs[0]
		if err := json.Unmarshal(extendedVoteTx, &extVoteInfo); err != nil {
			// returning an error here would cause consensus to panic. Reject the proposal instead if a proposer
			// deliberately does not include ExtendedVoteInfo at the beginning of the txs slice
			logger.Error("Error occurred while decoding ExtendedVoteInfo", "error", err)
			return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, nil
		}

		// validate the vote extensions
		if err := ValidateVoteExtensions(ctx, req.Height-1, req.ProposerAddress, extVoteInfo, req.ProposedLastCommit.Round, app.StakeKeeper); err != nil {
			logger.Error("Vote extensions don't have 2/3rds majority signatures. Rejecting proposal")
			return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, nil
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
				// This tx comes from a block that has already been pre-voted by >2/3 of the voting power, so this should never happen unless
				panic(fmt.Errorf("error occurred while decoding tx bytes in ExtendVoteHandler. Error: %w", err))
				return nil, err
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

		var canonicalSideTxResponse sidetxs.ConsolidatedSideTxResponse
		if err := proto.Unmarshal(req.VoteExtension, &canonicalSideTxResponse); err != nil {
			logger.Error("ALERT, VOTE EXTENSION REJECTED. THIS SHOULD NOT HAPPEN; THE VALIDATOR COULD BE MALICIOUS! Error while unmarshalling VoteExtension", "validator", common.Bytes2Hex(req.ValidatorAddress), "error", err)
			return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_REJECT}, nil
		}

		// ensure block height and hash match
		if req.Height != canonicalSideTxResponse.Height {
			logger.Error("ALERT, VOTE EXTENSION REJECTED. THIS SHOULD NOT HAPPEN; THE VALIDATOR COULD BE MALICIOUS!", "block height", req.Height, "canonicalSideTxResponse height", canonicalSideTxResponse.Height, "validator", common.Bytes2Hex(req.ValidatorAddress))
			return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_REJECT}, nil
		}

		if !bytes.Equal(req.Hash, canonicalSideTxResponse.BlockHash) {
			logger.Error("ALERT, VOTE EXTENSION REJECTED. THIS SHOULD NOT HAPPEN; THE VALIDATOR COULD BE MALICIOUS!", "block hash", common.Bytes2Hex(req.Hash), "canonicalSideTxResponse blockHash", common.Bytes2Hex(canonicalSideTxResponse.BlockHash), "validator", common.Bytes2Hex(req.ValidatorAddress))
			return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_REJECT}, nil
		}

		// TODO HV2: Ensure the side txs included in V.E.s are actually present in the block. This would require the block being available in RequestVerifyVoteExtension
		//  As this issue https://github.com/informalsystems/heimdall-migration/issues/46 will be closed, this most probably won't be possible
		for _, v := range canonicalSideTxResponse.SideTxResponses {
			// check whether the vote result is valid
			if !isVoteValid(v.Result) {
				logger.Error("ALERT, VOTE EXTENSION REJECTED. THIS SHOULD NOT HAPPEN; THE VALIDATOR COULD BE MALICIOUS! Invalid vote result type", "vote result", v.Result, "validator", common.Bytes2Hex(req.ValidatorAddress))
				return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_REJECT}, nil
			}
		}

		// check for duplicate votes
		txHash, err := validateSideTxResponses(canonicalSideTxResponse.SideTxResponses)
		if err != nil {
			logger.Error("ALERT, VOTE EXTENSION REJECTED. THIS SHOULD NOT HAPPEN; THE VALIDATOR COULD BE MALICIOUS!", "validator", common.Bytes2Hex(req.ValidatorAddress), "tx hash", common.Bytes2Hex(txHash), "error", err)
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
	bz := req.Txs[0]
	if err := json.Unmarshal(bz, &extVoteInfo); err != nil {
		logger.Error("Error occurred while unmarshalling ExtendedVoteInfo", "error", err)
		return nil, err
	}

	if len(req.Txs) > 1 {
		txs := req.Txs[1:]

		// Fetch validators from previous block
		// TODO HV2: Heimdall as of now uses validator set from current height.
		//  Should we be taking into account the validator set from currentHeight-1 or currentHeight-2? Discuss with PoS team
		validators, err := app.StakeKeeper.GetValidatorSet(ctx)
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
		for _, rawTx := range txs {
			// create a cache wrapped context for stateless execution
			ctx, _ = app.cacheTxContext(ctx, rawTx)
			decodedTx, err := app.TxDecode(rawTx)
			if err != nil {
				logger.Error("Error occurred while decoding tx bytes", "error", err)
				return nil, err
			}

			var txBytes cmtTypes.Tx = rawTx

			for _, approvedTx := range approvedTxs {

				if bytes.Equal(approvedTx, txBytes.Hash()) {

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
