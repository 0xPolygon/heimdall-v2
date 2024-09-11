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
)

// VoteExtensionProcessor handles Vote Extension processing for Heimdall app
type VoteExtensionProcessor struct {
	app *HeimdallApp
}

// NewVoteExtensionProcessor returns a new VoteExtensionProcessor with its sideTxConfigurator
func NewVoteExtensionProcessor() *VoteExtensionProcessor {
	return &VoteExtensionProcessor{}
}

// NewPrepareProposalHandler prepares the proposal after validating the vote extensions
func (app *HeimdallApp) NewPrepareProposalHandler() sdk.PrepareProposalHandler {
	return func(ctx sdk.Context, req *abci.RequestPrepareProposal) (*abci.ResponsePrepareProposal, error) {
		logger := app.Logger()

		if err := ValidateVoteExtensions(ctx, req.Height, req.ProposerAddress, req.LocalLastCommit.Votes, req.LocalLastCommit.Round, app.StakeKeeper); err != nil {
			logger.Error("Error occurred while validating VEs in PrepareProposal", err)
			panic("vote extension validation failed during PrepareProposal")
		}

		// prepare the proposal with the vote extensions and the validators set's votes
		var txs [][]byte
		bz, err := json.Marshal(req.LocalLastCommit.Votes)
		if err != nil {
			logger.Error("Error occurred while marshaling the ExtendedVoteInfo in prepare proposal", "error", err)
			return nil, err
		}
		txs = append(txs, bz)

		// once added the VEs, we append add the txs to the proposal
		totalTxBytes := len(txs)
		for i, proposedTx := range req.Txs {

			// check the txs via the ante handler
			// skip the first tx as it contains the ExtendedVoteInfo and may not have a handler
			if err := checkTx(app, proposedTx); err != nil && i > 0 {
				logger.Error("Error occurred while checking tx in prepare proposal", "error", err)
				return nil, err
			}

			totalTxBytes += len(proposedTx)
			// check if the total tx bytes exceed the max tx bytes of the request
			if totalTxBytes > int(req.MaxTxBytes) {
				break
			}
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
			logger.Error("Unexpected behaviour, no txs found in the proposal")
			return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, nil
		}

		// check the txs via the ante handler
		for i, tx := range req.Txs {
			// skip the first tx as it contains the ExtendedVoteInfo and may not have an ante handler
			if i == 0 {
				continue
			}
			if err := checkTx(app, tx); err != nil {
				logger.Error("Error occurred while checking the tx in process proposal", "error", err)
				return nil, err
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
		if err := ValidateVoteExtensions(ctx, req.Height, req.ProposerAddress, extVoteInfo, req.ProposedLastCommit.Round, app.StakeKeeper); err != nil {
			logger.Error("Vote extensions don't have 2/3rds majority signatures. Rejecting proposal")
			return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, nil
		}

		return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_ACCEPT}, nil
	}
}

// ExtendVoteHandler extends precommit vote
func (v *VoteExtensionProcessor) ExtendVoteHandler() sdk.ExtendVoteHandler {
	return func(ctx sdk.Context, req *abci.RequestExtendVote) (*abci.ResponseExtendVote, error) {
		logger := v.app.Logger()
		logger.Debug("Extending Vote!", "height", ctx.BlockHeight())

		// check if VEs are enabled
		mustAddSpecialTransaction(ctx, req.Height)

		// prepare the side tx responses
		sideTxRes := make([]*sidetxs.SideTxResponse, 0)

		// extract the ExtendedVoteInfo from the txs (it is encoded at the beginning, index 0)
		var extVoteInfos []abci.ExtendedVoteInfo

		// check whether ExtendedVoteInfo is encoded at the beginning
		bz := req.Txs[0]
		if err := json.Unmarshal(bz, &extVoteInfos); err != nil {
			// abnormal behavior since the block got >2/3 prevotes, so the special tx should have been added
			panic(fmt.Errorf("error occurred while decoding ExtendedVoteInfos; "+
				"they should have be encoded in the beginning of txs slice. Error: %v", err))
		}

		txs := req.Txs[1:]

		// decode txs and execute side txs
		for _, rawTx := range txs {
			// create a cache wrapped context for stateless execution
			ctx, _ = v.app.cacheTxContext(ctx, rawTx)
			tx, err := v.app.TxDecode(rawTx)
			if err != nil {
				// This tx comes from a block that has already been prevoted by >2/3 of the voting power, so this should never happen unless
				panic(fmt.Errorf("error occurred while decoding tx bytes in ExtendVoteHandler. Error: %v", err))
				return nil, err
			}

			// messages represent the side txs (operations performed by modules using the VEs mechanism)
			// e.g. bor, checkpoint, clerk, milestone, stake and topup
			messages := tx.GetMsgs()
			for _, msg := range messages {
				// get the right module's side handler for the message
				sideHandler := v.app.sideTxCfg.GetSideHandler(msg)
				if sideHandler == nil {
					logger.Debug("No side handler found for the message", "msg", msg)
					continue
				}

				// execute the side handler to collect the votes from the validators
				res := sideHandler(ctx, msg)

				// add the side handler results (YES/NO/UNSPECIFIED votes) to the side tx response
				txHash := cmtTypes.Tx(rawTx).Hash()
				logger.Debug("Adding V.E.", "txHash", txHash, "blockHeight", req.Height, "blockHash", req.Hash)
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
			return &abci.ResponseExtendVote{VoteExtension: []byte{}}, nil
		}

		return &abci.ResponseExtendVote{VoteExtension: bz}, nil
	}
}

// VerifyVoteExtensionHandler performs some sanity checks on the VE received from other validators
func (v *VoteExtensionProcessor) VerifyVoteExtensionHandler() sdk.VerifyVoteExtensionHandler {
	return func(ctx sdk.Context, req *abci.RequestVerifyVoteExtension) (*abci.ResponseVerifyVoteExtension, error) {
		logger := v.app.Logger()
		logger.Debug("Verifying vote extension", "height", ctx.BlockHeight())

		var canonicalSideTxResponse sidetxs.ConsolidatedSideTxResponse
		if err := proto.Unmarshal(req.VoteExtension, &canonicalSideTxResponse); err != nil {
			logger.Error("ALERT, VOTE EXTENSION REJECTED. THIS SHOULD NOT HAPPEN; THE VALIDATOR COULD BE MALICIOUS! Error while unmarshalling VoteExtension", "error", err, "validator", string(req.ValidatorAddress))
			return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_REJECT}, nil
		}

		// ensure block height and hash match
		if req.Height != canonicalSideTxResponse.Height {
			logger.Error("ALERT, VOTE EXTENSION REJECTED. THIS SHOULD NOT HAPPEN; THE VALIDATOR COULD BE MALICIOUS!", "block height", req.Height, "canonicalSideTxResponse height", canonicalSideTxResponse.Height, "validator", string(req.ValidatorAddress))
			return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_REJECT}, nil
		}

		if !bytes.Equal(req.Hash, canonicalSideTxResponse.BlockHash) {
			logger.Error("ALERT, VOTE EXTENSION REJECTED. THIS SHOULD NOT HAPPEN; THE VALIDATOR COULD BE MALICIOUS!", "block hash", req.Hash, "canonicalSideTxResponse hash", canonicalSideTxResponse.BlockHash, "validator", string(req.ValidatorAddress))
			return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_REJECT}, nil
		}

		// TODO HV2: Ensure the side txs included in V.E.s are actually present in the block. This would require the block being available in RequestVerifyVoteExtension
		//  As this issue https://github.com/informalsystems/heimdall-migration/issues/46 will be closed, this most probably won't be possible
		for _, v := range canonicalSideTxResponse.SideTxResponses {
			// check whether the vote result is valid
			if !isVoteValid(v.Result) {
				logger.Error("ALERT, VOTE EXTENSION REJECTED. THIS SHOULD NOT HAPPEN; THE VALIDATOR COULD BE MALICIOUS! Invalid vote result type", "vote result", v.Result, "validator", string(req.ValidatorAddress))
				return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_REJECT}, nil
			}
		}

		// check for duplicate votes
		responsesValid, txHash := areSideTxResponsesValid(canonicalSideTxResponse.SideTxResponses)
		if !responsesValid {
			logger.Error("ALERT, VOTE EXTENSION REJECTED. THIS SHOULD NOT HAPPEN; THE VALIDATOR COULD BE MALICIOUS!", "validator", string(req.ValidatorAddress), "tx hash", string(txHash))
			return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_REJECT}, nil
		}

		return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_ACCEPT}, nil
	}
}

// PreBlocker application updates every pre block
func (app *HeimdallApp) PreBlocker(ctx sdk.Context, req *abci.RequestFinalizeBlock) (*sdk.ResponsePreBlock, error) {
	logger := app.Logger()

	mustAddSpecialTransaction(ctx, req.Height+1)

	// Extract ExtendedVoteInfo encoded at the beginning of txs bytes
	var extVoteInfo []abci.ExtendedVoteInfo

	if len(req.Txs) > 0 {
		bz := req.Txs[0]
		if err := json.Unmarshal(bz, &extVoteInfo); err != nil {
			logger.Error("Error occurred while unmarshalling ExtendedVoteInfo", "error", err)
			return nil, err
		}
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

// checkTx invokes the abci method to check the tx by using the ante handler
func checkTx(app *HeimdallApp, tx []byte) error {
	res, err := app.CheckTx(&abci.RequestCheckTx{
		Tx:   tx,
		Type: abci.CheckTxType_New,
	})
	if err != nil || res.IsErr() {
		return err
	}
	return nil
}
