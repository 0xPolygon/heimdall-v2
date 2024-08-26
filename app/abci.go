package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	cmtTypes "github.com/cometbft/cometbft/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	mod "github.com/0xPolygon/heimdall-v2/module"
)

// TODO HV2: implement abci_test.go

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

// NewPrepareProposalHandler checks for 2/3+ V.E. sigs and reject the proposal in case we don't have a majority.
func (app *HeimdallApp) NewPrepareProposalHandler() sdk.PrepareProposalHandler {
	return func(ctx sdk.Context, req *abci.RequestPrepareProposal) (*abci.ResponsePrepareProposal, error) {
		logger := app.Logger()

		// start including ExtendedVoteInfo a block after vote extensions are enabled
		if !mustAddSpecialTransaction(ctx, req.Height+1) {
			return &abci.ResponsePrepareProposal{Txs: req.Txs}, nil
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

		sideTxRes := make([]*mod.SideTxResponse, 0)

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

		var canonicalSideTxResponse mod.CanonicalSideTxResponse
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

		// TODO HV2: Ensure the side txs included in V.E.s are actually present in the block.
		// This will be possible once the block is available to be consumed in RequestVerifyVoteExtension from Comet
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

	// store side tx data
	txs := req.Txs
	if mustAddSpecialTransaction(ctx, req.Height+1) {
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
			postHandler := app.VoteExtensionProcessor.sideTxCfg.GetPostHandler(msg)
			if postHandler != nil {
				// TODO HV2: uncomment when implemented
				// app.VoteExtensionKeeper.storeTxData(ctx, txBytes.Hash(), tx)
			}
		}

	}

	if mustAddSpecialTransaction(ctx, req.Height+1) {
		// Extract ExtendedVoteInfo encoded at the beginning of txs bytes
		var extVoteInfo []abci.ExtendedVoteInfo

		bz := req.Txs[0]
		if err := json.Unmarshal(bz, &extVoteInfo); err != nil {
			logger.Error("Error occurred while unmarshalling ExtendedVoteInfo", "error", err)
			return nil, err
		}

		// Fetch validators from previous block
		// TODO HV2: Heimdall as of now uses validator set from currentHeight. But should we be taking into account the validator set from currentHeight - 1/ currentHeight - 2 ?
		// TODO HV2 (in response to the previous comment): could not find the function where currentHeight is used
		validators, err := app.StakeKeeper.GetValidatorSet(ctx)
		if err != nil {
			return nil, err
		}
		if len(validators.Validators) == 0 {
			return nil, errors.New("no validators found")
		}

		// tally votes
		approvedTxs, rejectedTxs, skippedTxs, err := tallyVotes(extVoteInfo, logger, validators.Validators)
		if err != nil {
			logger.Error("Error occurred while tallying votes", "error", err)
			return nil, err
		}

		// execute side txs
		for _, txHash := range approvedTxs {
			// check whether tx exists
			if !app.VoteExtensionKeeper.HasTx(ctx, txHash) {
				logger.Error("side tx not found in keeper", "tx", txHash)
				continue
			}

			// fetch side tx from keeper
			tx, err := app.VoteExtensionKeeper.GetTxData(ctx, txHash)
			if err != nil {
				logger.Error("Error occurred while fetching side tx from keeper", "error", err)
				return nil, err
			}

			// execute with YES vote
			msgs := tx.GetMsgs()
			for _, msg := range msgs {
				fn, ok := app.VoteExtensionHandler.modPostHandler[sdk.MsgTypeURL(msg)]
				if !ok {
					return nil, errors.New("could not fetch post handler for the tx msg")
				}

				// TODO HV2: how do we process the events ?
				err := fn(ctx, msg, mod.Vote_VOTE_YES)
				if err != nil {
					logger.Error("Error occurred while executing post handler", "error", err, "tx", tx)
					continue

				}
			}

			// remove tx from keeper to prevent re-execution
			if err := app.VoteExtensionKeeper.removeTx(ctx, txHash); err != nil {
				logger.Error("Error occurred while deleting side tx from keeper", "error", err)
				return nil, err
			}

		}

		// delete the rejected and skipped txs
		for _, txHash := range rejectedTxs {
			if err := app.VoteExtensionKeeper.removeTx(ctx, txHash); err != nil {
				logger.Error("Error occurred while deleting side tx from keeper", "error", err)
				return nil, err
			}
		}

		for _, txHash := range skippedTxs {
			if err := app.VoteExtensionKeeper.removeTx(ctx, txHash); err != nil {
				logger.Error("Error occurred while deleting side tx from keeper", "error", err)
				return nil, err
			}
		}
	}

	return app.mm.PreBlock(ctx)
}
