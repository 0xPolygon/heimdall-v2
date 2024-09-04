package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	cmtTypes "github.com/cometbft/cometbft/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/0xPolygon/heimdall-v2/sidetxs"
)

// TODO HV2: check the correct usage and flow of the VEs

// VoteExtensionProcessor handles Vote Extension processing for Heimdall app
type VoteExtensionProcessor struct {
	app       *HeimdallApp
	sideTxCfg sidetxs.SideTxConfigurator
}

func NewVoteExtensionProcessor(cfg sidetxs.SideTxConfigurator) *VoteExtensionProcessor {
	return &VoteExtensionProcessor{
		sideTxCfg: cfg,
	}
}

func (v *VoteExtensionProcessor) SetSideTxConfigurator(cfg sidetxs.SideTxConfigurator) {
	v.sideTxCfg = cfg
}

// NewPrepareProposalHandler checks for 2/3+ V.E. sigs and reject the proposal in case we don't have a majority.
func (app *HeimdallApp) NewPrepareProposalHandler() sdk.PrepareProposalHandler {
	return func(ctx sdk.Context, req *abci.RequestPrepareProposal) (*abci.ResponsePrepareProposal, error) {
		logger := app.Logger()

		// start including ExtendedVoteInfo a block after vote extensions are enabled
		if !mustAddSpecialTransaction(ctx, req.Height+1) {
			return &abci.ResponsePrepareProposal{Txs: req.Txs}, nil
		}

		for _, vote := range req.LocalLastCommit.Votes {
			var consolidatedSideTxResponse sidetxs.ConsolidatedSideTxResponse
			if err := json.Unmarshal(vote.VoteExtension, &consolidatedSideTxResponse); err != nil {
				logger.Error("Error while unmarshalling VoteExtension during PrepareProposal", "error", err, "validator", string(req.ProposerAddress))
				return nil, errors.New("can't prepare the proposal because the vote extension is not valid")
			}
			// check for duplicate votes
			hasDupVotes, txHash := checkDuplicateVotes(consolidatedSideTxResponse.SideTxResponses)
			if hasDupVotes {
				logger.Error("Proposer voted more than once for a side transaction", "validator", string(req.ProposerAddress), "tx hash", string(txHash))
				panic("can't prepare the proposal because of duplicated votes")
			}
		}

		// Validate VE sigs and check whether they have 2/3+ majority
		if err := ValidateVoteExtensions(ctx, ctx.BlockHeight(), ctx.ChainID(), req.LocalLastCommit.Votes, req.LocalLastCommit.Round, app.StakeKeeper); err != nil {
			logger.Error("PrepareProposal: Error occurred while validating VEs: ", err)
			return nil, errors.New("can't prepare the block without more than 2/3 majority")
		}

		var txs [][]byte

		bz, err := json.Marshal(req.LocalLastCommit.Votes)
		if err != nil {
			logger.Error("Error occurred while marshaling extVoteInfo", "error", err)
			return nil, err
		}
		txs = append(txs, bz)

		// encode the txs
		totalTxBytes := len(txs)
		for _, rtx := range req.Txs {
			totalTxBytes += len(rtx)
			if totalTxBytes > int(req.MaxTxBytes) {
				break
			}
			txs = append(txs, rtx)
		}
		return &abci.ResponsePrepareProposal{Txs: txs}, nil
	}
}

// NewProcessProposalHandler checks for 2/3+ V.E. sigs and reject the proposal in case we don't have a majority.
// It is implemented by all the validators
func (app *HeimdallApp) NewProcessProposalHandler() sdk.ProcessProposalHandler {
	return func(ctx sdk.Context, req *abci.RequestProcessProposal) (*abci.ResponseProcessProposal, error) {
		logger := app.Logger()

		// check for ExtendedVoteInfo a block after vote extensions are enabled
		if !mustAddSpecialTransaction(ctx, req.Height+1) {
			for _, tx := range req.Txs {
				checkTx, err := app.CheckTx(&abci.RequestCheckTx{Tx: tx})
				if err != nil || checkTx.IsErr() {
					logger.Error("Error occurred while checking tx", "error", err)
					return nil, err
				}
			}
			return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_ACCEPT}, nil
		}

		var extVoteInfo []abci.ExtendedVoteInfo

		if len(req.Txs) < 1 {
			logger.Error("Unexpected behaviour, no txs found in the proposal")
			return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, nil
		}

		// Extract ExtendedVoteInfo from txs (encoded at the beginning)
		bz := req.Txs[0]

		if err := json.Unmarshal(bz, &extVoteInfo); err != nil {
			// returning an error here would cause consensus to panic. Reject the proposal instead if a proposer
			// deliberately does not include ExtendedVoteInfo at the beginning of txs slice
			logger.Error("Error occurred while decoding ExtendedVoteInfo", "error", err)
			return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, nil
		}

		for _, vote := range extVoteInfo {
			var consolidatedSideTxResponse sidetxs.ConsolidatedSideTxResponse
			if err := json.Unmarshal(vote.VoteExtension, &consolidatedSideTxResponse); err != nil {
				logger.Error("Error while unmarshalling VoteExtension during ProcessProposal", "error", err, "proposer", string(req.ProposerAddress))
				return nil, errors.New("can't process the proposal because the vote extension is not valid")
			}
			// check for duplicate votes
			hasDupVotes, txHash := checkDuplicateVotes(consolidatedSideTxResponse.SideTxResponses)
			if hasDupVotes {
				logger.Error("Proposer voted more than once for a side transaction", "validator", string(req.ProposerAddress), "tx hash", string(txHash))
				panic("can't prepare the proposal because of duplicated votes")
			}
		}

		// Validate VE sigs and check whether they have 2/3+ majority
		if err := ValidateVoteExtensions(ctx, ctx.BlockHeight(), ctx.ChainID(), extVoteInfo, req.ProposedLastCommit.Round, app.StakeKeeper); err != nil {
			logger.Error("Vote extensions don't have 2/3rds majority signatures. Rejecting proposal")
			return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, nil
		}

		return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_ACCEPT}, nil

	}

}

// ExtendVote extends precommit vote
func (v *VoteExtensionProcessor) ExtendVote() sdk.ExtendVoteHandler {
	return func(ctx sdk.Context, req *abci.RequestExtendVote) (*abci.ResponseExtendVote, error) {
		logger := v.app.Logger()

		sideTxRes := make([]*sidetxs.SideTxResponse, 0)

		if len(req.Txs) > 1 || (len(req.Txs) == 1 && req.Height == ctx.ConsensusParams().Abci.VoteExtensionsEnableHeight) {
			var extVoteInfo []abci.ExtendedVoteInfo

			logger.Debug("Extending Vote!", "height", ctx.BlockHeight())

			txs := req.Txs

			if mustAddSpecialTransaction(ctx, req.Height) {
				// check whether ExtendedVoteInfo is encoded at the beginning
				bz := req.Txs[0]
				if err := json.Unmarshal(bz, &extVoteInfo); err != nil {
					// abnormal behavior since the block got >2/3 prevotes
					panic(fmt.Errorf("error occurred while decoding ExtendedVoteInfo; they should have be encoded in the beginning of txs slice. Error: %v",
						err))
				}

				txs = req.Txs[1:]
			}

			for _, rawTx := range txs {
				// create a cache wrapped context for stateless execution
				ctx, _ = v.app.cacheTxContext(ctx, rawTx)
				tx, err := v.app.TxDecode(rawTx)
				if err != nil {
					logger.Error("Error occurred while decoding tx bytes", "error", err)
					return nil, err
				}

				msgs := tx.GetMsgs()
				for _, msg := range msgs {
					sideHandler := v.sideTxCfg.GetSideHandler(msg)
					if sideHandler == nil {
						continue
					}

					res := sideHandler(ctx, msg)

					var txBytes cmtTypes.Tx = rawTx

					// add result to side tx response
					logger.Debug("Adding V.E.", "txHash", txBytes.Hash(), "blockHeight", req.Height, "blockHash", req.Hash)
					ve := sidetxs.SideTxResponse{
						TxHash: txBytes.Hash(),
						Result: res,
					}
					sideTxRes = append(sideTxRes, &ve)

				}

			}
		}

		canonicalSideTxRes := sidetxs.ConsolidatedSideTxResponse{
			SideTxResponses: sideTxRes,
			Height:          req.Height,
			Hash:            req.Hash,
		}

		bz, err := json.Marshal(canonicalSideTxRes)
		if err != nil {
			logger.Error("Error occurred while marshalling VoteExtension", "error", err)
			return &abci.ResponseExtendVote{VoteExtension: []byte{}}, nil
		}

		return &abci.ResponseExtendVote{VoteExtension: bz}, nil
	}
}

// VerifyVoteExtension performs some sanity checks on the V.E received from other validators
func (v *VoteExtensionProcessor) VerifyVoteExtension() sdk.VerifyVoteExtensionHandler {
	return func(ctx sdk.Context, req *abci.RequestVerifyVoteExtension) (*abci.ResponseVerifyVoteExtension, error) {
		logger := v.app.Logger()
		logger.Debug("Verifying vote extension", "height", ctx.BlockHeight())

		var canonicalSideTxResponse sidetxs.ConsolidatedSideTxResponse
		if err := json.Unmarshal(req.VoteExtension, &canonicalSideTxResponse); err != nil {
			logger.Error("ALERT, VOTE EXTENSION REJECTED. THIS SHOULD NOT HAPPEN; THE VALIDATOR COULD BE MALICIOUS! Error while unmarshalling VoteExtension", "error", err, "validator", string(req.ValidatorAddress))
			return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_REJECT}, nil
		}

		// ensure block height and hash match
		switch {
		case req.Height != canonicalSideTxResponse.Height:
			logger.Error("ALERT, VOTE EXTENSION REJECTED. THIS SHOULD NOT HAPPEN; THE VALIDATOR COULD BE MALICIOUS!", "block height", req.Height, "canonicalSideTxResponse height", canonicalSideTxResponse.Height, "validator", string(req.ValidatorAddress))
			return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_REJECT}, nil

		case !bytes.Equal(req.Hash, canonicalSideTxResponse.Hash):
			logger.Error("ALERT, VOTE EXTENSION REJECTED. THIS SHOULD NOT HAPPEN; THE VALIDATOR COULD BE MALICIOUS!", "block hash", req.Hash, "canonicalSideTxResponse hash", canonicalSideTxResponse.Hash, "validator", string(req.ValidatorAddress))
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
		hasDupVotes, txHash := checkDuplicateVotes(canonicalSideTxResponse.SideTxResponses)
		if hasDupVotes {
			logger.Error("ALERT, VOTE EXTENSION REJECTED. THIS SHOULD NOT HAPPEN; THE VALIDATOR COULD BE MALICIOUS! Validator voted more than once for a side transaction", "validator", string(req.ValidatorAddress), "tx hash", string(txHash))
			return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_REJECT}, nil
		}

		return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_ACCEPT}, nil
	}
}

// PreBlocker application updates every pre block
func (app *HeimdallApp) PreBlocker(ctx sdk.Context, req *abci.RequestFinalizeBlock) (*sdk.ResponsePreBlock, error) {
	logger := app.Logger()

	if mustAddSpecialTransaction(ctx, req.Height+1) {
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
			//  Should we be taking into account the validator set from currentHeight - 1/ currentHeight - 2 ?
			//  Discuss with PoS team
			validators, err := app.StakeKeeper.GetValidatorSet(ctx)
			if err != nil {
				return nil, err
			}
			if len(validators.Validators) == 0 {
				return nil, errors.New("no validators found")
			}

			// tally votes
			approvedTxs, _, _, err := tallyVotes(extVoteInfo, logger, validators.Validators)
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
							postHandler := app.VoteExtensionProcessor.sideTxCfg.GetPostHandler(msg)
							if postHandler != nil {
								postHandler(ctx, msg, sidetxs.Vote_VOTE_YES)
							}
						}

					}
				}
			}
		}
	}

	return app.ModuleManager.PreBlock(ctx)
}
