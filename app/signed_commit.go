package app

import (
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmtTypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/codec/address"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/0xPolygon/heimdall-v2/sidetxs"
	milestoneAbci "github.com/0xPolygon/heimdall-v2/x/milestone/abci"
	stakeTypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

// signedCommitVotes authenticates the transported bytes before deriving any local view.
// A proposer cannot replace a committing validator's opinion with an empty placeholder.
func (app *HeimdallApp) signedCommitVotes(ctx sdk.Context, height int64, commit *abci.ExtendedCommitInfo, validators *stakeTypes.ValidatorSet) ([]abci.ExtendedVoteInfo, error) {
	return app.commitVotes(ctx, height, commit, validators, true)
}

// commitVotes skips the signature checks only for a commit ProcessProposal already authenticated.
func (app *HeimdallApp) commitVotes(ctx sdk.Context, height int64, commit *abci.ExtendedCommitInfo, validators *stakeTypes.ValidatorSet, verifySignatures bool) ([]abci.ExtendedVoteInfo, error) {
	if height <= retrieveVoteExtensionsEnableHeight(ctx) {
		return initialCommitVotes(commit)
	}
	seen := make(map[string]bool, len(commit.Votes))
	votes := append([]abci.ExtendedVoteInfo(nil), commit.Votes...)
	// Pin every committing vote to the block hash prefix of the first one; the >2/3 signature quorum
	// authenticates it, and the signature binds height and round, so the prefix only has to tell
	// blocks of the same round apart.
	var blockHash []byte
	var power int64
	for i, vote := range votes {
		validator, err := commitValidator(vote, validators, seen)
		if err != nil {
			return nil, err
		}
		if vote.BlockIdFlag != cmtproto.BlockIDFlagCommit {
			continue
		}
		votes[i].VoteExtension, blockHash, err = app.commitVoteView(ctx, height, commit.Round, vote, validator, blockHash, verifySignatures)
		if err != nil {
			return nil, err
		}
		power += validator.VotingPower
	}
	if power <= validators.GetTotalVotingPower()*2/3 {
		return nil, fmt.Errorf("insufficient signed commit voting power: %d", power)
	}
	return votes, nil
}

func authenticateCommitVote(ctx sdk.Context, height int64, round int32, vote abci.ExtendedVoteInfo, validator *stakeTypes.Validator, blockHash []byte, verifySignatures bool) (*sidetxs.VoteExtension, error) {
	if len(vote.VoteExtension) > maxSignedVESize {
		return nil, fmt.Errorf("oversized commit vote")
	}
	if blockHash == nil {
		var extension sidetxs.VoteExtension
		if err := extension.Unmarshal(vote.VoteExtension); err != nil {
			return nil, err
		}
		blockHash = extension.BlockHash
	}
	verified, err := validateVoteEnvelope(&abci.RequestVerifyVoteExtension{
		Height: height - 1, Hash: blockHash,
		VoteExtension: vote.VoteExtension, NonRpVoteExtension: vote.NonRpVoteExtension,
	})
	if err != nil || !verifySignatures {
		return verified, err
	}
	key, err := getValidatorPublicKey(validator)
	if err != nil {
		return nil, err
	}
	// The same sign bytes CometBFT signs and verifies votes with.
	extSignBytes, nonRpSignBytes := cmtTypes.VoteExtensionSignBytes(ctx.ChainID(), &cmtproto.Vote{
		Height: height - 1, Round: round, Extension: vote.VoteExtension, NonRpExtension: vote.NonRpVoteExtension,
	})
	if !key.VerifySignature(extSignBytes, vote.ExtensionSignature) || !key.VerifySignature(nonRpSignBytes, vote.NonRpExtensionSignature) {
		return nil, fmt.Errorf("invalid commit extension signature")
	}
	return verified, nil
}

func commitValidator(vote abci.ExtendedVoteInfo, validators *stakeTypes.ValidatorSet, seen map[string]bool) (*stakeTypes.Validator, error) {
	addr, err := (address.HexCodec{}).BytesToString(vote.Validator.Address)
	if err != nil || seen[addr] {
		return nil, fmt.Errorf("invalid or duplicate commit validator %X", vote.Validator.Address)
	}
	seen[addr] = true
	_, validator := validators.GetByAddress(addr)
	if validator == nil || validator.VotingPower != vote.Validator.Power || !isBlockIdFlagValid(vote.BlockIdFlag) {
		return nil, fmt.Errorf("invalid commit validator or power: %s", addr)
	}
	if vote.BlockIdFlag != cmtproto.BlockIDFlagCommit && len(vote.VoteExtension)+len(vote.NonRpVoteExtension)+len(vote.ExtensionSignature)+len(vote.NonRpExtensionSignature) != 0 {
		return nil, fmt.Errorf("extensions on non-committing validator %s", addr)
	}
	return validator, nil
}

// Milestone params can change after signing. Only the unusable proposition is
// omitted from this local view; signatures are always checked against raw bytes.
func (app *HeimdallApp) milestoneView(ctx sdk.Context, extension *sidetxs.VoteExtension, raw []byte) ([]byte, error) {
	if err := milestoneAbci.ValidateMilestoneProposition(ctx, &app.MilestoneKeeper, extension.MilestoneProposition); err == nil {
		return raw, nil
	}
	extension.MilestoneProposition = nil
	return extension.Marshal()
}

func initialCommitVotes(commit *abci.ExtendedCommitInfo) ([]abci.ExtendedVoteInfo, error) {
	if len(commit.Votes) != 0 {
		return nil, fmt.Errorf("non-empty initial commit")
	}
	return nil, nil
}

func (app *HeimdallApp) commitVoteView(ctx sdk.Context, height int64, round int32, vote abci.ExtendedVoteInfo, validator *stakeTypes.Validator, blockHash []byte, verifySignatures bool) ([]byte, []byte, error) {
	extension, err := authenticateCommitVote(ctx, height, round, vote, validator, blockHash, verifySignatures)
	if err != nil {
		return nil, nil, fmt.Errorf("validator %s: %w", validator.Signer, err)
	}
	view, err := app.milestoneView(ctx.WithBlockHeight(height-1), extension, vote.VoteExtension)
	return view, extension.BlockHash, err
}
