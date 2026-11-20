package app

import (
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/helper"
	checkpointTypes "github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
)

func TestSignedCommitSurvivesCheckpointAck(t *testing.T) {
	s := signedScenario(t)
	_, err := s.app.PrepareProposal(&abci.RequestPrepareProposal{Height: nonRpSigHeight, MaxTxBytes: 1_000_000, LocalLastCommit: abci.ExtendedCommitInfo{Votes: s.votes}})
	require.NoError(t, err)
	_, err = s.app.ProcessProposal(&abci.RequestProcessProposal{Height: nonRpSigHeight, Txs: [][]byte{s.commit(t)}})
	require.NoError(t, err)
	require.NoError(t, s.app.CheckpointKeeper.AddCheckpoint(s.ctx, checkpointTypes.Checkpoint{Id: 1, StartBlock: 0, EndBlock: s.endBlock}))
	require.NoError(t, s.app.CheckpointKeeper.UpdateAckCountWithValue(s.ctx, 1))
	response, err := s.app.NewPrepareProposalHandler()(s.ctx, &abci.RequestPrepareProposal{
		Height: nonRpSigHeight, MaxTxBytes: 1_000_000, LocalLastCommit: abci.ExtendedCommitInfo{Votes: s.votes}, Txs: [][]byte{s.cpTx},
	})
	require.NoError(t, err)
	require.Len(t, response.Txs, 2)
	canonical := make([]abci.VoteInfo, len(s.votes))
	for i, vote := range s.votes {
		canonical[i] = abci.VoteInfo{Validator: vote.Validator, BlockIdFlag: vote.BlockIdFlag}
	}
	result, err := s.app.NewProcessProposalHandler()(s.ctx, &abci.RequestProcessProposal{
		Height: nonRpSigHeight, Txs: response.Txs, ProposedLastCommit: abci.CommitInfo{Votes: canonical},
	})
	require.NoError(t, err)
	require.Equal(t, abci.ResponseProcessProposal_ACCEPT, result.Status)
}

// At the activation height the proposal still carries legacy votes, signed before an ACK advanced the
// checkpoint state; a checkpoint tx next to them must not get the proposal rejected from Purley.
func TestActivationProposalAcceptsLegacyCheckpointVotesAfterAck(t *testing.T) {
	tests := []struct {
		name   string
		purley int64
		status abci.ResponseProcessProposal_ProposalStatus
	}{
		{name: "purley at proposal height", purley: nonRpSigHeight, status: abci.ResponseProcessProposal_ACCEPT},
		{name: "purley after proposal height", purley: nonRpSigHeight + 1, status: abci.ResponseProcessProposal_REJECT},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := newNonRpSigScenario(t, nonRpSigHeight)
			previous := helper.GetPurleyHeight()
			t.Cleanup(func() { helper.SetPurleyHeight(previous) })
			canonical := make([]abci.VoteInfo, len(s.votes))
			for i, vote := range s.votes {
				canonical[i] = abci.VoteInfo{Validator: vote.Validator, BlockIdFlag: vote.BlockIdFlag}
			}
			// Prime the proposal states at activation, which skips Bor, so x/checkpoint's package-level
			// cache doesn't keep a Bor result for later tests.
			helper.SetPurleyHeight(nonRpSigHeight)
			primed, err := s.app.PrepareProposal(&abci.RequestPrepareProposal{Height: nonRpSigHeight, MaxTxBytes: 1_000_000, LocalLastCommit: abci.ExtendedCommitInfo{Votes: s.votes}})
			require.NoError(t, err)
			_, err = s.app.ProcessProposal(&abci.RequestProcessProposal{Height: nonRpSigHeight, Txs: primed.Txs, ProposedLastCommit: abci.CommitInfo{Votes: canonical}})
			require.NoError(t, err)
			helper.SetPurleyHeight(tc.purley)
			require.NoError(t, s.app.CheckpointKeeper.AddCheckpoint(s.ctx, checkpointTypes.Checkpoint{Id: 1, StartBlock: 0, EndBlock: s.endBlock}))
			require.NoError(t, s.app.CheckpointKeeper.UpdateAckCountWithValue(s.ctx, 1))

			response, err := s.app.NewPrepareProposalHandler()(s.ctx, &abci.RequestPrepareProposal{
				Height: nonRpSigHeight, MaxTxBytes: 1_000_000, LocalLastCommit: abci.ExtendedCommitInfo{Votes: s.votes}, Txs: [][]byte{s.cpTx},
			})
			require.NoError(t, err)
			result, err := s.app.NewProcessProposalHandler()(s.ctx, &abci.RequestProcessProposal{
				Height: nonRpSigHeight, Txs: [][]byte{response.Txs[0], s.cpTx}, ProposedLastCommit: abci.CommitInfo{Votes: canonical},
			})
			require.NoError(t, err)
			require.Equal(t, tc.status, result.Status)
		})
	}
}

func TestSignedCommitActivationKeepsLegacyCheckpointVotes(t *testing.T) {
	s := newNonRpSigScenario(t, nonRpSigHeight)
	previous := helper.GetPurleyHeight()
	t.Cleanup(func() { helper.SetPurleyHeight(previous) })
	require.NoError(t, s.app.CheckpointKeeper.AddCheckpoint(s.ctx, checkpointTypes.Checkpoint{Id: 1, StartBlock: 0, EndBlock: s.endBlock}))
	require.NoError(t, s.app.CheckpointKeeper.UpdateAckCountWithValue(s.ctx, 1))
	helper.SetPurleyHeight(nonRpSigHeight)
	require.NoError(t, ValidateNonRpVoteExtensions(s.ctx, nonRpSigHeight, s.votes, &s.vs, s.app.ChainManagerKeeper, s.app.CheckpointKeeper, s.app.caller, s.app.Logger()))
	helper.SetPurleyHeight(nonRpSigHeight + 1)
	require.Error(t, ValidateNonRpVoteExtensions(s.ctx, nonRpSigHeight, s.votes, &s.vs, s.app.ChainManagerKeeper, s.app.CheckpointKeeper, s.app.caller, s.app.Logger()))
}
