package app

import (
	"errors"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/0xPolygon/heimdall-v2/helper"
	borTypes "github.com/0xPolygon/heimdall-v2/x/bor/types"
	checkpointTypes "github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
	stakeTypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

func (app *HeimdallApp) prepareCommit(ctx sdk.Context, req *abci.RequestPrepareProposal, validators *stakeTypes.ValidatorSet) error {
	// A proposal at H+1 carries votes signed under the rules at H.
	if helper.IsPurley(req.Height - 1) {
		// VerifyVoteExtension keeps such votes out of consensus, but a LastCommit rebuilt from a
		// block-synced extended commit skips it. CometBFT panics on a PrepareProposal error, so
		// propose anyway: every validator rejects the proposal and the round moves on.
		if _, err := app.signedCommitVotes(ctx, req.Height, &req.LocalLastCommit, validators); err != nil {
			app.Logger().Error("Proposing a last commit that fails authentication", "height", req.Height, "error", err)
		}
		return nil
	}
	votes, err := filterVoteExtensions(ctx, req.Height, req.LocalLastCommit.Votes, req.LocalLastCommit.Round, validators, app.MilestoneKeeper, req.MaxTxBytes, app.Logger())
	if err != nil {
		return err
	}
	req.LocalLastCommit.Votes = votes
	if err := ValidateNonRpVoteExtensions(ctx, req.Height, votes, validators, app.ChainManagerKeeper, app.CheckpointKeeper, app.caller, app.Logger()); err != nil {
		app.Logger().Warn("Error validating non-rp vote extensions", "error", err)
	}
	return nil
}

func (app *HeimdallApp) validateProposalCommit(ctx sdk.Context, req *abci.RequestProcessProposal, commit *abci.ExtendedCommitInfo, validators *stakeTypes.ValidatorSet) error {
	if helper.IsPurley(req.Height - 1) {
		_, err := app.signedCommitVotes(ctx, req.Height, commit, validators)
		return err
	}
	if err := ValidateVoteExtensions(ctx, req.Height, commit.Votes, req.ProposedLastCommit.Round, validators, app.MilestoneKeeper); err != nil {
		return err
	}
	return app.validateLegacyNonRpCommit(ctx, req, commit, validators)
}

func (app *HeimdallApp) validateLegacyNonRpCommit(ctx sdk.Context, req *abci.RequestProcessProposal, commit *abci.ExtendedCommitInfo, validators *stakeTypes.ValidatorSet) error {
	err := ValidateNonRpVoteExtensions(ctx, req.Height, commit.Votes, validators, app.ChainManagerKeeper, app.CheckpointKeeper, app.caller, app.Logger())
	if err == nil {
		return nil
	}
	if helper.IsPhuketHardfork(req.Height) && helper.IsKarwar(req.Height) && errors.Is(err, errNonRpSignature) {
		app.Logger().Error("Invalid non-rp vote extension signature, rejecting proposal", "error", err)
		return err
	}
	hasCheckpoint, decodeErr := app.proposalHasCheckpoint(req.Txs[1:])
	if decodeErr != nil {
		return decodeErr
	}
	tolerateBorErr := helper.IsZurichHardfork(req.Height) && (errors.Is(err, borTypes.ErrFailedToQueryBor) || errors.Is(err, borTypes.ErrBorBlockNotFound))
	if helper.IsPhuketHardfork(req.Height) && hasCheckpoint && !tolerateBorErr {
		return err
	}
	app.Logger().Warn("Invalid non-rp vote extension proposal", "error", err)
	return nil
}

func (app *HeimdallApp) proposalHasCheckpoint(txs [][]byte) (bool, error) {
	for _, raw := range txs {
		tx, err := app.TxDecode(raw)
		if err != nil {
			return false, err
		}
		for _, msg := range tx.GetMsgs() {
			if checkpointTypes.IsCheckpointMsg(msg) {
				return true, nil
			}
		}
	}
	return false, nil
}

func (app *HeimdallApp) validatorCommitVotes(ctx sdk.Context, height int64, commit *abci.ExtendedCommitInfo) (*stakeTypes.ValidatorSet, []abci.ExtendedVoteInfo, error) {
	validators, err := app.getValidatorSetForHeight(ctx, height)
	if err != nil {
		return nil, nil, err
	}
	votes := commit.Votes
	if helper.IsPurley(height - 1) {
		// ProcessProposal authenticated the signatures of a committed block.
		votes, err = app.commitVotes(ctx, height, commit, validators, false)
	}
	return validators, votes, err
}
