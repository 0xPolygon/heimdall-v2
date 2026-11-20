package app

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"

	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/metrics"
	"github.com/0xPolygon/heimdall-v2/sidetxs"
	milestoneTypes "github.com/0xPolygon/heimdall-v2/x/milestone/types"
	stakeTypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

// The signed transport applies from the height the votes were signed at, not the proposal height.
// A commit failing it is still proposed, since CometBFT panics on a PrepareProposal error, and
// validators reject it.
func TestPrepareProposalUsesVoteActivationHeight(t *testing.T) {
	for _, fork := range []int64{nonRpSigHeight - 1, nonRpSigHeight, nonRpSigHeight + 1} {
		t.Run(fmt.Sprint(fork), func(t *testing.T) {
			s := newNonRpSigScenario(t, nonRpSigHeight)
			previous := helper.GetPurleyHeight()
			t.Cleanup(func() { helper.SetPurleyHeight(previous) })
			helper.SetPurleyHeight(fork)
			s.votes[0].VoteExtension = nil
			s.votes[0].NonRpVoteExtension = nil
			s.votes[0].ExtensionSignature = nil
			s.votes[0].NonRpExtensionSignature = nil
			res, err := s.app.NewPrepareProposalHandler()(s.ctx, &abci.RequestPrepareProposal{
				Height: nonRpSigHeight, MaxTxBytes: 1_000_000, LocalLastCommit: abci.ExtendedCommitInfo{Votes: s.votes},
			})
			require.NoError(t, err)
			signed := fork < nonRpSigHeight
			require.Equal(t, signed, strings.Contains(s.logs.String(), "Proposing a last commit that fails authentication"))
			if signed {
				require.Equal(t, abci.ResponseProcessProposal_REJECT, s.processProposal(t, res.Txs[0]))
			}
		})
	}
}

// Before Purley the non-rp hash field is rejected on the wire even when empty, as older binaries do.
func TestLegacyWireRulesUseTheSignedHeight(t *testing.T) {
	s := newNonRpSigScenario(t, nonRpSigHeight)
	previous := helper.GetPurleyHeight()
	t.Cleanup(func() { helper.SetPurleyHeight(previous) })
	helper.SetPurleyHeight(nonRpSigHeight)
	raw := protowire.AppendBytes(protowire.AppendTag(bytes.Clone(s.votes[0].VoteExtension), 5, protowire.BytesType), nil)
	s.votes[0].VoteExtension = raw
	createSignatureForVoteExtension(t, nonRpSigHeight-1, s.keys[0], raw, s.votes[0].NonRpVoteExtension, &s.votes[0])
	require.Error(t, ValidateVoteExtensions(s.ctx, nonRpSigHeight, s.votes, 0, &s.vs, s.app.MilestoneKeeper))
	filtered, err := filterVoteExtensions(s.ctx, nonRpSigHeight, s.votes, 0, &s.vs, s.app.MilestoneKeeper, 1_000_000, s.app.Logger())
	require.NoError(t, err)
	require.Len(t, filtered, len(s.votes))
	require.Empty(t, filtered[0].VoteExtension)
	require.NotEmpty(t, filtered[1].VoteExtension)
}

// A legacy commit without more than 2/3 of the voting power cannot be proposed.
func TestPrepareProposalRejectsInsufficientLegacyCommit(t *testing.T) {
	s := newNonRpSigScenario(t, nonRpSigHeight)
	previous := helper.GetPurleyHeight()
	t.Cleanup(func() { helper.SetPurleyHeight(previous) })
	helper.SetPurleyHeight(nonRpSigHeight + 1)
	_, err := s.app.NewPrepareProposalHandler()(s.ctx, &abci.RequestPrepareProposal{
		Height: nonRpSigHeight, MaxTxBytes: 1_000_000, LocalLastCommit: abci.ExtendedCommitInfo{Votes: s.votes[:2]},
	})
	require.ErrorContains(t, err, "insufficient cumulative voting power")
}

// A proposal whose transactions cannot be decoded is rejected while checking for checkpoints.
func TestLegacyProposalRejectsUndecodableTransactions(t *testing.T) {
	s := newNonRpSigScenario(t, nonRpSigHeight+1)
	previous := helper.GetPurleyHeight()
	t.Cleanup(func() { helper.SetPurleyHeight(previous) })
	helper.SetPurleyHeight(nonRpSigHeight + 1)
	s.mutateAll(func([]byte) []byte { return bytes.Repeat([]byte{0x42}, 65) })
	request := &abci.RequestProcessProposal{Height: nonRpSigHeight, Txs: [][]byte{s.commit(t), {0xff}}}
	require.Error(t, s.app.validateLegacyNonRpCommit(s.ctx, request, &abci.ExtendedCommitInfo{Votes: s.votes}, &s.vs))
}

// Without the historical validator set the signed commit cannot be authenticated.
func TestSnapshotFailureStopsCommitProcessing(t *testing.T) {
	s := signedScenario(t)
	store := s.ctx.KVStore(s.app.GetKey(stakeTypes.StoreKey))
	for _, suffix := range [][]byte{stakeTypes.PreviousBlockValidatorSetKey, stakeTypes.PenultimateBlockValidatorSetKey} {
		store.Delete(append(bytes.Clone(stakeTypes.ValidatorSetKey), suffix...))
	}
	_, _, err := s.app.validatorCommitVotes(s.ctx, nonRpSigHeight, &abci.ExtendedCommitInfo{Votes: s.votes})
	require.Error(t, err)
}

// Envelope rejections are counted so operators can see them.
func TestInvalidEnvelopeRecordsRejection(t *testing.T) {
	s := signedScenario(t)
	counter := metrics.VoteExtensionRejectedTotal.WithLabelValues("invalid_envelope")
	before := testutil.ToFloat64(counter)
	response, err := s.app.VerifyVoteExtension(&abci.RequestVerifyVoteExtension{Height: nonRpSigHeight, VoteExtension: []byte{0xff}})
	require.NoError(t, err)
	require.Equal(t, abci.ResponseVerifyVoteExtension_REJECT, response.Status)
	require.Equal(t, before+1, testutil.ToFloat64(counter))
}

// The milestone view is validated under the fork rules of the signed height and never alters the signed bytes.
func TestMilestoneViewUsesSignedForkRules(t *testing.T) {
	for _, ithaca := range []int64{1, nonRpSigHeight} {
		t.Run(fmt.Sprint(ithaca), func(t *testing.T) {
			s := signedScenario(t)
			helper.SetIthacaHeight(ithaca)
			hash := bytes.Repeat([]byte{3}, 32)
			// The head is the proposition tail, so its hash is implied in the compact form.
			updateSignedVote(t, s, 0, func(v *sidetxs.VoteExtension) {
				v.MilestoneProposition = &milestoneTypes.MilestoneProposition{StartBlockNumber: 1, BlockHashes: [][]byte{hash}, BlockTds: []uint64{1}, LatestBlockNumber: 1}
			})
			votes, err := s.app.signedCommitVotes(s.ctx, nonRpSigHeight, &abci.ExtendedCommitInfo{Votes: s.votes}, &s.vs)
			require.NoError(t, err)
			var view sidetxs.VoteExtension
			require.NoError(t, view.Unmarshal(votes[0].VoteExtension))
			if ithaca == 1 {
				require.NotNil(t, view.MilestoneProposition)
				require.Equal(t, s.votes[0].VoteExtension, votes[0].VoteExtension)
			} else {
				require.Nil(t, view.MilestoneProposition)
			}
		})
	}
}
