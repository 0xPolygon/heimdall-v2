package app

import (
	"bytes"
	"fmt"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/sidetxs"
)

// The compact format and the signed transport switch at the same vote height: votes signed at H-1 are
// legacy and carried at H under the placeholder rules, votes signed at H are compact and carried at H+1
// as signed. The scenario signs at scenarioVoteHeight and proposes one block later.
func TestCompactSignedTransportBoundary(t *testing.T) {
	tests := []struct {
		name    string
		purley  int64
		compact bool
	}{
		{"votes signed at H-1", scenarioVoteHeight + 1, false},
		{"votes signed at H", scenarioVoteHeight, true},
		{"votes signed at H+1", scenarioVoteHeight - 1, true},
	}
	for _, tc := range tests {
		for _, format := range []bool{tc.compact, !tc.compact} {
			t.Run(fmt.Sprintf("%s/compact=%t", tc.name, format), func(t *testing.T) {
				s := newNonRpSigScenario(t, nonRpSigHeight)
				previous := helper.GetPurleyHeight()
				t.Cleanup(func() { helper.SetPurleyHeight(previous) })
				helper.SetPurleyHeight(tc.purley)
				for i := range s.votes {
					encodeSignedVote(t, s, i, format, func(*sidetxs.VoteExtension) {})
				}
				if format != tc.compact {
					require.Equal(t, abci.ResponseProcessProposal_REJECT, s.processProposal(t, s.commit(t)))
					return
				}
				require.Equal(t, abci.ResponseProcessProposal_ACCEPT, s.processProposal(t, s.commit(t)))
				s.requireSignaturesStored(t, s.genuine)
			})
		}
	}
}

// A late vote verified on its own is checked against the voted block, while a commit pins every
// vote to the prefix carried by the first one, so a single vote for another block rejects the proposal.
func TestSignedCommitPinsBlockPrefix(t *testing.T) {
	for i := range 4 {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			s := signedScenario(t)
			other := bytes.Clone(scenarioBlockHash[:compactVEBlockHashLength])
			other[compactVEBlockHashLength-1] ^= 1
			updateSignedVote(t, s, i, func(v *sidetxs.VoteExtension) { v.BlockHash = other })
			_, err := s.app.signedCommitVotes(s.ctx, nonRpSigHeight, &abci.ExtendedCommitInfo{Votes: s.votes}, &s.vs)
			require.ErrorContains(t, err, "block hash mismatch")
		})
	}
}

// The compact placeholder is shorter than the legacy floor, so the envelope sizes the non-rp floor at
// the signed height; a legacy floor would reject every idle vote from Purley.
func TestEnvelopeAcceptsCompactPlaceholder(t *testing.T) {
	s := signedScenario(t)
	placeholder := CompactDummyNonRpVoteExtension(scenarioVoteHeight, s.ctx.ChainID())
	require.Less(t, len(placeholder), minNonRpVoteExtensionSize)
	for i := range s.votes {
		s.votes[i].NonRpVoteExtension = placeholder
		updateSignedVote(t, s, i, func(*sidetxs.VoteExtension) {})
	}
	result, err := s.app.VerifyVoteExtension(&abci.RequestVerifyVoteExtension{
		Height: scenarioVoteHeight, Hash: scenarioBlockHash, VoteExtension: s.votes[0].VoteExtension, NonRpVoteExtension: placeholder,
	})
	require.NoError(t, err)
	require.Equal(t, abci.ResponseVerifyVoteExtension_ACCEPT, result.Status)
	require.Equal(t, abci.ResponseProcessProposal_ACCEPT, s.processProposal(t, s.commit(t)))
}
