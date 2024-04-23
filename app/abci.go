package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	mod "github.com/0xPolygon/heimdall-v2/module"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtTypes "github.com/cometbft/cometbft/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	jsoniter "github.com/json-iterator/go"
)

// VoteExtensionProcessor handles Vote Extension processing for Heimdall app
type VoteExtensionProcessor struct {
	app       *HeimdallApp
	sideTxCfg mod.SideTxConfigurator
}

func NewVoteExtensionProcessor(cfg mod.SideTxConfigurator) *VoteExtensionProcessor {
	return &VoteExtensionProcessor{
		sideTxCfg: cfg,
	}
}

func (v *VoteExtensionProcessor) SetSideTxConfigurator(cfg mod.SideTxConfigurator) {
	v.sideTxCfg = cfg
}

// NewPrepareProposalHandler check for 2/3+ V.E. sigs and reject the proposal in case we don't have a majority.
func (app *HeimdallApp) NewPrepareProposalHandler() sdk.PrepareProposalHandler {
	return func(ctx sdk.Context, req *abci.RequestPrepareProposal) (*abci.ResponsePrepareProposal, error) {
		logger := app.Logger()

		// start including ExtendedVoteInfo a block after vote extensions are enabled
		if !canAddVE(ctx, req.Height) {
			return &abci.ResponsePrepareProposal{Txs: req.Txs}, nil
		}
		// Validate VE sigs and check whether they have 2/3+ majority
		hasTwoThirdsSigs := true
		// TODO HV2: uncomment when implemented
		// if err := ValidateVoteExtensions(ctx, ctx.BlockHeight(), ctx.ChainID(), req.LocalLastCommit.Votes, req.LocalLastCommit.Round, app.AccountKeeper); err != nil {
		// 	logger.Error("PrepareProposal: Error occurred while validating VEs: ", err)
		// 	hasTwoThirdsSigs = false
		// }

		var txs [][]byte
		rawTxs := req.Txs

		// Only include VE if there's majority
		if !hasTwoThirdsSigs {
			return nil, errors.New("can't prepare the block without more than 2/3 majority")
		}

		bz, err := jsoniter.ConfigFastest.Marshal(req.LocalLastCommit.Votes)
		if err != nil {
			// TODO CHECK HV2: we throw an error since encoding ExtendedVoteInfo should not fail ideally
			// clarify with Informal
			logger.Error("Error occurred while marshaling extVoteInfo", "error", err)
			return nil, err
		}
		txs = append(txs, bz)

		// encode the txs
		totalTxBytes := len(txs)
		for _, rtx := range rawTxs {
			totalTxBytes += len(rtx)
			if totalTxBytes > int(req.MaxTxBytes) {
				break
			}
			txs = append(txs, rtx)
		}
		return &abci.ResponsePrepareProposal{Txs: txs}, nil
	}
}

// Check for 2/3+ V.E. sigs and reject the proposal in case we don't have a majority.
func (app *HeimdallApp) NewProcessProposalHandler() sdk.ProcessProposalHandler {
	return func(ctx sdk.Context, req *abci.RequestProcessProposal) (*abci.ResponseProcessProposal, error) {
		logger := app.Logger()

		// check for ExtendedVoteInfo a block after vote extensions are enabled
		if !canAddVE(ctx, req.Height) {
			// TODO HV2: Clarify with Informal:
			// Should we execute CheckTx() as well if a malicious proposer includes an invalid tx
			return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_ACCEPT}, nil
		}

		var extVoteInfo []abci.ExtendedVoteInfo

		if len(req.Txs) < 1 {
			return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, nil
		}

		// Extract ExtendedVoteInfo from txs (encoded at the beginning)
		bz := req.Txs[0]

		if err := jsoniter.ConfigFastest.Unmarshal(bz, &extVoteInfo); err != nil {
			// returning an error here would cause consensus to panic. Reject the proposal instead if a proposer
			// deliberately does not include ExtendedVoteInfo at the beginning of txs slice
			logger.Error("Error occurred while decoding ExtendedVoteInfo", "error", err)
			return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, nil
		}

		// Validate VE sigs and check whether they have 2/3+ majority
		hasTwoThirdsSigs := true
		// TODO HV2: uncomment when implemented
		// if err := ValidateVoteExtensions(ctx, ctx.BlockHeight(), ctx.ChainID(), extVoteInfo, req.ProposedLastCommit.Round, app.AccountKeeper); err != nil {
		// 	hasTwoThirdsSigs = false
		// }

		if !hasTwoThirdsSigs {
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

		sideTxRes := make([]*mod.SideTxResponse, 0)

		if len(req.Txs) > 1 || (len(req.Txs) >= 1 && req.Height == ctx.ConsensusParams().Abci.VoteExtensionsEnableHeight) {
			var extVoteInfo []abci.ExtendedVoteInfo

			logger.Debug("Extending Vote!", "height", ctx.BlockHeight())

			txs := req.Txs

			if canAddVE(ctx, req.Height) {
				// check whether ExtendedVoteInfo is encoded at the beginning
				bz := req.Txs[0]
				if err := jsoniter.ConfigFastest.Unmarshal(bz, &extVoteInfo); err != nil {
					// abnormal behavior since the block got >2/3 prevotes
					panic(fmt.Errorf("%v occurred while decoding ExtendedVoteInfo; they should have be encoded in the beginning of txs slice",
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

				// TODO HV2: Clarify with Informal: Should we execute CheckTx() ?
				msgs := tx.GetMsgs()
				for _, msg := range msgs {
					sideHandler := v.sideTxCfg.GetSideHandler(msg)
					if sideHandler == nil {
						continue
					}

					res := sideHandler(ctx, msg)

					var txBytes cmtTypes.Tx = rawTx

					// add result to side tx response
					logger.Debug("Adding V.E", "txhash", txBytes.Hash(), "block height", req.Height, "block hash", req.Hash)
					ve := mod.SideTxResponse{
						TxHash: txBytes.Hash(),
						Result: res,
					}
					sideTxRes = append(sideTxRes, &ve)

				}

			}
		}

		canonicalSideTxRes := mod.CanonicalSideTxResponse{
			SideTxResponses: sideTxRes,
			Height:          req.Height,
			Hash:            req.Hash,
		}

		bz, err := jsoniter.ConfigFastest.Marshal(canonicalSideTxRes)
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

		var ve mod.CanonicalSideTxResponse
		if err := json.Unmarshal(req.VoteExtension, &ve); err != nil {
			logger.Error("ALERT, VOTE EXTENSION REJECTED. THIS SHOULD NOT HAPPEN; THE VALIDATOR COULD BE MALICIOUS! Error while unmarshalling VoteExtension: %v", err, "validator", req.ValidatorAddress)
			return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_REJECT}, nil
		}

		// ensure block height and hash match
		switch {
		case req.Height != ve.Height:
			logger.Error("ALERT, VOTE EXTENSION REJECTED. THIS SHOULD NOT HAPPEN; THE VALIDATOR COULD BE MALICIOUS!", "block height", req.Height, "ve height", ve.Height, "validator", req.ValidatorAddress)
			return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_REJECT}, nil

		case !bytes.Equal(req.Hash, ve.Hash):
			logger.Error("ALERT, VOTE EXTENSION REJECTED. THIS SHOULD NOT HAPPEN; THE VALIDATOR COULD BE MALICIOUS!", "block hash", req.Hash, "ve hash", ve.Hash, "validator", req.ValidatorAddress)
			return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_REJECT}, nil

		}

		// TODO HV2: Ensure the side txs included in V.E are actually present in the block.
		// This will be possible once the block is available to be consumed in RequestVerifyVoteExtension from Comet
		for _, v := range ve.SideTxResponses {
			// check whether the vote result is valid
			isValidVote := v.Result == mod.Vote_VOTE_YES || v.Result == mod.Vote_VOTE_NO || v.Result == mod.Vote_VOTE_SKIP
			if !isValidVote {
				logger.Error("ALERT, VOTE EXTENSION REJECTED. THIS SHOULD NOT HAPPEN; THE VALIDATOR COULD BE MALICIOUS! vote result type invalid", "vote result", v.Result, "validator", req.ValidatorAddress)
				return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_REJECT}, nil
			}
		}

		// check for duplicate votes
		hasDupVotes, txHash := checkDuplicateVotes(ve.SideTxResponses)
		if hasDupVotes {
			logger.Error("ALERT, VOTE EXTENSION REJECTED. THIS SHOULD NOT HAPPEN; THE VALIDATOR COULD BE MALICIOUS! Validator voted twice for a side transaction", "validator", req.ValidatorAddress, "tx hash", txHash)
			return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_REJECT}, nil
		}

		return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_ACCEPT}, nil
	}
}

// PreBlocker application updates every pre block
func (app *HeimdallApp) PreBlocker(ctx sdk.Context, req *abci.RequestFinalizeBlock) (*sdk.ResponsePreBlock, error) {
	logger := app.Logger()

	if req.Height >= ctx.ConsensusParams().Abci.VoteExtensionsEnableHeight {
		// store side tx data
		txs := req.Txs
		if canAddVE(ctx, req.Height) {
			txs = req.Txs[1:]
		}

		for _, rawTx := range txs {
			tx, err := app.TxDecode(rawTx)
			if err != nil {
				logger.Error("Error occurred while decoding tx bytes", "error", err)
				return nil, err
			}

			msgs := tx.GetMsgs()
			for _, msg := range msgs {
				postHandler := app.VoteExtensionProcessor.sideTxCfg.GetPostHandler(msg) //nolint:staticcheck
				//nolint:staticcheck
				if postHandler != nil {
					// TODO HV2: uncomment when implemented
					// app.VoteExtensionKeeper.storeTxData(ctx, txBytes.Hash(), tx)
				}
			}

		}

		if canAddVE(ctx, req.Height) {
			// Extract ExtendedVoteInfo encoded at the beginning of txs bytes
			var extVoteInfo []abci.ExtendedVoteInfo

			bz := req.Txs[0]
			// TODO HV2: Clarify with Informal: should we throw an error here ?
			if err := jsoniter.ConfigFastest.Unmarshal(bz, &extVoteInfo); err != nil {
				logger.Error("Error occurred while unmarshalling ExtendedVoteInfo", "error", err)
				return nil, err
			}

			// Fetch validators from previous block
			// TODO HV2: Heimdall as of now uses validator set from currentHeight. But should we be taking into account the validator set from currentHeight - 1/ currentHeight - 2 ?
			// validators := app.StakeKeeper.GetValidators(ctx, req.Height - 1)
			// if len(validators) == 0 {
			// 	return errors.New("No validators found")
			// }

			// TODO HV2: uncomment when implemented
			// tally votes
			// approvedTxs, rejectedTxs, skippedTxs, err := tallyVotes(extVoteInfo, logger, validators)
			// if err != nil {
			// 	logger.Error("Error occurred while tallying votes", "error", err)
			// 	return nil, err
			// }

			// // execute side txs
			// for _, txHash := range approvedTxs {
			// 	// check whether tx exists
			// 	if !app.VoteExtensionKeeper.HasTx(ctx, txHash) {
			// 		logger.Error("side tx not found in keeper", "tx", txHash)
			// 		continue
			// 	}

			// 	// fetch side tx from keeper
			// 	tx, err := app.VoteExtensionKeeper.GetTxData(ctx, txHash)
			// 	if err != nil {
			// 		logger.Error("Error occurred while fetching side tx from keeper", "error", err)
			// 		return nil, err
			// 	}

			// 	// execute with YES vote
			// 	msgs := tx.GetMsgs()
			// 	for _, msg := range msgs {
			// 		fn, ok := app.VoteExtensionHandler.modPostHandler[sdk.MsgTypeURL(msg)]
			// 		if !ok {
			// 			return nil, errors.New("Could not fetch Posthandler for the tx msg")
			// 		}

			// 		// TODO HV2: how do we process the events ?
			// 		err := fn(ctx, msg, types.Vote_VOTE_YES)
			// 		if err != nil {
			// 			logger.Error("Error occurred while executing post handler", "error", err, "tx", tx)
			// 			continue

			// 		}
			// 	}

			// 	// remove tx from keeper to prevent re-execution
			// 	if err := app.VoteExtensionKeeper.removeTx(ctx, txHash); err != nil {
			// 		logger.Error("Error occurred while deleting side tx from keeper", "error", err)
			// 		return nil, err
			// 	}

			// }

			// // delete the rejected and skipped txs
			// for _, txHash := range rejectedTxs {
			// 	if err := app.VoteExtensionKeeper.removeTx(ctx, txHash); err != nil {
			// 		logger.Error("Error occurred while deleting side tx from keeper", "error", err)
			// 		return nil, err
			// 	}
			// }

			// for _, txHash := range skippedTxs {
			// 	if err := app.VoteExtensionKeeper.removeTx(ctx, txHash); err != nil {
			// 		logger.Error("Error occurred while deleting side tx from keeper", "error", err)
			// 		return nil, err
			// 	}
			// }
		}

	}

	return app.mm.PreBlock(ctx)
}
