package app

import (
	"bytes"
	"errors"
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	cmtTypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/codec/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/ethereum/go-ethereum/common"

	"github.com/0xPolygon/heimdall-v2/sidetxs"
	borTypes "github.com/0xPolygon/heimdall-v2/x/bor/types"
	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
	checkpointTypes "github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
	milestoneAbci "github.com/0xPolygon/heimdall-v2/x/milestone/abci"
	milestoneTypes "github.com/0xPolygon/heimdall-v2/x/milestone/types"
)

// NewPrepareProposalHandler prepares the proposal after validating the vote extensions
func (app *HeimdallApp) NewPrepareProposalHandler() sdk.PrepareProposalHandler {
	return func(ctx sdk.Context, req *abci.RequestPrepareProposal) (*abci.ResponsePrepareProposal, error) {
		logger := app.Logger()

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
		bz, err := req.LocalLastCommit.Marshal()
		if err != nil {
			logger.Error("Error occurred while marshaling the LocalLastCommit in prepare proposal", "error", err)
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

		// check if there are less than 1 txs in the request
		if len(txs) < 1 {
			logger.Error(fmt.Sprintf("unexpected behaviour, less than 1 txs proposed by %s", req.ProposerAddress))
			return nil, fmt.Errorf("unexpected behaviour, less than 1 txs proposed by %s", req.ProposerAddress)
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

		// extract the ExtendedCommitInfo from the txs (it is encoded at the beginning, index 0)
		extCommitInfo := new(abci.ExtendedCommitInfo)
		extendedCommitTx := req.Txs[0]
		if err := extCommitInfo.Unmarshal(extendedCommitTx); err != nil {
			logger.Error("Error occurred while decoding ExtendedCommitInfo", "height", req.Height, "error", err)
			return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, nil
		}

		if extCommitInfo.Round != req.ProposedLastCommit.Round {
			logger.Error("Received commit round does not match expected round", "expected", req.ProposedLastCommit.Round, "got", extCommitInfo.Round)
			return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, nil
		}

		// validate the vote extensions
		if err := ValidateVoteExtensions(ctx, req.Height, req.ProposerAddress, extCommitInfo.Votes, req.ProposedLastCommit.Round, app.StakeKeeper); err != nil {
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
			return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, nil
		}

		for _, tx := range req.Txs[1:] {

			txn, err := app.TxDecode(tx)
			if err != nil {
				logger.Error("error occurred while decoding tx bytes in ProcessProposalHandler", err)
				return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, nil
			}

			// ensure we allow transactions with only one side msg inside
			if sidetxs.CountSideHandlers(app.sideTxCfg, txn) > 1 {
				return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, nil
			}

			// run the tx by executing the msg_server handler on the tx msgs and the ante handler
			_, err = app.ProcessProposalVerifyTx(tx)
			if err != nil {
				// this should never happen, as the txs have already been checked in PrepareProposal
				logger.Error("RunTx returned an error in ProcessProposal", "error", err)
				return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, nil
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
		if err := checkIfVoteExtensionsDisabled(ctx, req.Height); err != nil {
			return nil, err
		}

		// prepare the side tx responses
		sideTxRes := make([]sidetxs.SideTxResponse, 0)

		// extract the ExtendedVoteInfo from the txs (it is encoded at the beginning, index 0)
		extCommitInfo := new(abci.ExtendedCommitInfo)

		// check whether ExtendedVoteInfo is encoded at the beginning
		bz := req.Txs[0]
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
		// TODO: Drop the ConsolidatedSideTxResponse and just have SideTxResponses in the VoteExtension
		consolidatedSideTxRes := sidetxs.ConsolidatedSideTxResponse{
			SideTxResponses: sideTxRes,
			// TODO: Move Height and BlockHash to be part of the sidetxs.VoteExtension
			Height:    req.Height,
			BlockHash: req.Hash,
		}

		vt := sidetxs.VoteExtension{
			ConsolidatedSideTxResponse: &consolidatedSideTxRes,
			MilestoneProposition:       nil,
		}

		milestoneProp, err := milestoneAbci.GenMilestoneProposition(ctx, &app.MilestoneKeeper, &app.caller)
		if err != nil {
			logger.Error("Error occurred while generating milestone proposition", "error", err)
			// We still want to participate in the consensus even if we fail to generate the milestone proposition
		} else if milestoneProp != nil {
			vt.MilestoneProposition = milestoneProp
		}

		bz, err = vt.Marshal()
		if err != nil {
			logger.Error("Error occurred while marshalling the VoteExtension in ExtendVoteHandler", "error", err)
			return nil, err
		}

		// TODO: Before returning the milestone proposition, we can have here same level of validation as in VerifyVoteExtension to make sure it will really be accepted

		if err := ValidateNonRpVoteExtension(ctx, req.Height, nonRpVoteExt, app.ChainManagerKeeper, app.CheckpointKeeper, &app.caller); err != nil {
			logger.Error("Error occurred while validating non-rp vote extension", "error", err)
			if errors.Is(err, borTypes.ErrFailedToQueryBor) {
				return &abci.ResponseExtendVote{VoteExtension: bz, NonRpExtension: dummyVoteExt}, nil
			}
			return nil, err
		}

		return &abci.ResponseExtendVote{VoteExtension: bz, NonRpExtension: nonRpVoteExt}, nil
	}
}

// VerifyVoteExtensionHandler performs some sanity checks on the VE received from other validators
func (app *HeimdallApp) VerifyVoteExtensionHandler() sdk.VerifyVoteExtensionHandler {
	return func(ctx sdk.Context, req *abci.RequestVerifyVoteExtension) (*abci.ResponseVerifyVoteExtension, error) {
		logger := app.Logger()
		logger.Debug("Verifying vote extension", "height", ctx.BlockHeight())

		// check if VEs are enabled
		if err := checkIfVoteExtensionsDisabled(ctx, req.Height); err != nil {
			return nil, err
		}

		ac := address.NewHexCodec()
		valAddr, err := ac.BytesToString(req.ValidatorAddress)
		if err != nil {
			return nil, err
		}

		var voteExtension sidetxs.VoteExtension
		if err := proto.Unmarshal(req.VoteExtension, &voteExtension); err != nil {
			logger.Error("ALERT, VOTE EXTENSION REJECTED. THIS SHOULD NOT HAPPEN; THE VALIDATOR COULD BE MALICIOUS! Error while unmarshalling VoteExtension", "validator", valAddr, "error", err)
			return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_REJECT}, nil
		}

		consolidatedSideTxResponse := voteExtension.GetConsolidatedSideTxResponse()

		// TODO: Add here stronger or same level of verification as in PrepareProposal for the voteExtension.MilestoneProposition

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

		return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_ACCEPT}, nil
	}
}

// PreBlocker application updates every pre block
func (app *HeimdallApp) PreBlocker(ctx sdk.Context, req *abci.RequestFinalizeBlock) (*sdk.ResponsePreBlock, error) {
	logger := app.Logger()

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

	bz := req.Txs[0]
	if err := extCommitInfo.Unmarshal(bz); err != nil {
		logger.Error("Error occurred while unmarshalling ExtendedCommitInfo", "error", err)
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

	// TODO: we already have getPreviousBlockValidatorSet function in vote_ext_utils.go. Maybe drop it and just make
	// stakeKeeper.GetPreviousBlockValidatorSet to return error on empty set
	validators, err := app.StakeKeeper.GetPreviousBlockValidatorSet(ctx)
	if err != nil {
		return nil, err
	}
	if len(validators.Validators) == 0 {
		return nil, errors.New("no validators found")
	}

	majorityMilestone, err := milestoneAbci.GetMajorityMilestoneProposition(ctx, validators, extVoteInfo, logger)
	if err != nil {
		logger.Error("Error occurred while getting majority milestone proposition", "error", err)
		return nil, err
	}

	if majorityMilestone != nil {
		params, err := app.ChainManagerKeeper.GetParams(ctx)
		if err != nil {
			logger.Error("Error occurred while getting chain manager params", "error", err)
			return nil, err
		}

		addMilestoneCtx, msCache := app.cacheTxContext(ctx)
		if err := app.MilestoneKeeper.AddMilestone(addMilestoneCtx, milestoneTypes.Milestone{
			Proposer:    "0x0000000000000000000000000000000000000000", // TODO: Here we maybe put aggregated hash of all addresses that proposed the milestone
			Hash:        majorityMilestone.BlockHash,
			StartBlock:  majorityMilestone.BlockNumber,
			EndBlock:    majorityMilestone.BlockNumber,
			BorChainId:  params.ChainParams.BorChainId,
			MilestoneId: "0x0000000000000000000000000000000000000000", // TODO: This should be also deterministically generated, maybe the same like proposer
			Timestamp:   uint64(ctx.BlockHeader().Time.Unix()),
		}); err != nil {
			logger.Error("Error occurred while adding milestone", "error", err)
			return nil, err
		}

		msCache.Write()
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

	return app.ModuleManager.PreBlock(ctx)
}
