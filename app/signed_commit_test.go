package app

import (
	"bytes"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmtTypes "github.com/cometbft/cometbft/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"

	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/sidetxs"
	milestoneTypes "github.com/0xPolygon/heimdall-v2/x/milestone/types"
)

func signedScenario(t *testing.T) *nonRpSigScenario {
	t.Helper()
	s := newNonRpSigScenario(t, nonRpSigHeight)
	previous := helper.GetPurleyHeight()
	t.Cleanup(func() { helper.SetPurleyHeight(previous) })
	helper.SetPurleyHeight(nonRpSigHeight - 1)
	for i := range s.votes {
		updateSignedVote(t, s, i, func(*sidetxs.VoteExtension) {})
	}
	return s
}

// scenarioVoteHeight is the height the scenario votes are signed at.
const scenarioVoteHeight = nonRpSigHeight - 1

// scenarioBlockHash is the full hash of the block the scenario votes are for.
var scenarioBlockHash = common.Hex2Bytes("a1a2a3a4a5a6a7a800000000000000000000000000000000000000000001dead")

// updateSignedVote re-encodes vote i in the format active at the signed height, applies update and re-signs it.
func updateSignedVote(t *testing.T, s *nonRpSigScenario, i int, update func(*sidetxs.VoteExtension)) {
	t.Helper()
	encodeSignedVote(t, s, i, helper.IsPurley(scenarioVoteHeight), update)
}

// encodeSignedVote re-signs vote i in the compact or legacy format, whichever fork is active.
func encodeSignedVote(t *testing.T, s *nonRpSigScenario, i int, compact bool, update func(*sidetxs.VoteExtension)) {
	t.Helper()
	var extension sidetxs.VoteExtension
	require.NoError(t, extension.Unmarshal(s.votes[i].VoteExtension))
	extension.Height, extension.BlockHash = scenarioVoteHeight, scenarioBlockHash
	if compact {
		extension.Height, extension.BlockHash = 0, scenarioBlockHash[:compactVEBlockHashLength]
	}
	extension.NonRpExtensionHash = nonRpExtensionHash(scenarioVoteHeight, s.votes[i].NonRpVoteExtension)
	update(&extension)
	encoded, err := extension.Marshal()
	require.NoError(t, err)
	s.votes[i].VoteExtension = encoded
	createSignatureForVoteExtension(t, scenarioVoteHeight, s.keys[i], encoded, s.votes[i].NonRpVoteExtension, &s.votes[i])
}

func TestSignedCommitAdmission(t *testing.T) {
	tests := []struct {
		name       string
		mutate     func(*nonRpSigScenario)
		wantReject bool
	}{
		{name: "original envelopes"},
		{name: "empty committing placeholder", wantReject: true, mutate: func(s *nonRpSigScenario) {
			s.votes[0] = abci.ExtendedVoteInfo{Validator: s.votes[0].Validator, BlockIdFlag: cmtproto.BlockIDFlagCommit}
		}},
		{name: "non-rp replacement", wantReject: true, mutate: func(s *nonRpSigScenario) { s.votes[0].NonRpVoteExtension = bytes.Repeat([]byte{3}, 40) }},
		{name: "oversized non-rp", wantReject: true, mutate: func(s *nonRpSigScenario) { s.votes[0].NonRpVoteExtension = make([]byte, maxNonRpVoteExtensionSize+1) }},
		{name: "oversized rp", wantReject: true, mutate: func(s *nonRpSigScenario) { s.votes[0].VoteExtension = make([]byte, maxSignedVESize+1) }},
		{name: "bad non-rp signature", wantReject: true, mutate: func(s *nonRpSigScenario) { s.votes[0].NonRpExtensionSignature[0] ^= 1 }},
		{name: "full-size signed rp", mutate: func(s *nonRpSigScenario) { padSignedVote(t, s, maxSignedVESize) }},
		{name: "oversized signed rp", wantReject: true, mutate: func(s *nonRpSigScenario) { padSignedVote(t, s, maxSignedVESize+2) }},
		{name: "bad signature", wantReject: true, mutate: func(s *nonRpSigScenario) { s.votes[0].ExtensionSignature[0] ^= 1 }},
		{name: "duplicate validator", wantReject: true, mutate: func(s *nonRpSigScenario) { s.votes[1] = s.votes[0] }},
		{name: "changed power", wantReject: true, mutate: func(s *nonRpSigScenario) { s.votes[0].Validator.Power++ }},
		{name: "signed height field", wantReject: true, mutate: func(s *nonRpSigScenario) {
			updateSignedVote(t, s, 0, func(v *sidetxs.VoteExtension) { v.Height = scenarioVoteHeight })
		}},
		{name: "signed at another height", wantReject: true, mutate: func(s *nonRpSigScenario) {
			createSignatureForVoteExtension(t, nonRpSigHeight, s.keys[0], s.votes[0].VoteExtension, s.votes[0].NonRpVoteExtension, &s.votes[0])
		}},
		{name: "signed full block hash", wantReject: true, mutate: func(s *nonRpSigScenario) {
			updateSignedVote(t, s, 0, func(v *sidetxs.VoteExtension) { v.BlockHash = scenarioBlockHash })
		}},
		{name: "signed different block prefix", wantReject: true, mutate: func(s *nonRpSigScenario) {
			updateSignedVote(t, s, 1, func(v *sidetxs.VoteExtension) { v.BlockHash = bytes.Repeat([]byte{7}, compactVEBlockHashLength) })
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := signedScenario(t)
			if tc.mutate != nil {
				tc.mutate(s)
			}
			want := abci.ResponseProcessProposal_ACCEPT
			if tc.wantReject {
				want = abci.ResponseProcessProposal_REJECT
			}
			require.Equal(t, want, s.processProposal(t, s.commit(t)))
		})
	}
}

func TestLateVoteCompletesSideApproval(t *testing.T) {
	s := signedScenario(t)
	updateSignedVote(t, s, 0, func(v *sidetxs.VoteExtension) { v.SideTxResponses[0].Result = sidetxs.Vote_VOTE_NO })
	late := s.votes[3]
	s.votes[3] = abci.ExtendedVoteInfo{Validator: late.Validator, BlockIdFlag: cmtproto.BlockIDFlagAbsent}
	votes, err := s.app.signedCommitVotes(s.ctx, nonRpSigHeight, &abci.ExtendedCommitInfo{Votes: s.votes}, &s.vs)
	require.NoError(t, err)
	approved, _, _, err := tallyVotes(votes, s.app.Logger(), &s.vs, nonRpSigHeight)
	require.NoError(t, err)
	require.Empty(t, approved)
	result, err := s.app.VerifyVoteExtension(&abci.RequestVerifyVoteExtension{
		Height: scenarioVoteHeight, Hash: scenarioBlockHash, ValidatorAddress: late.Validator.Address,
		VoteExtension: late.VoteExtension, NonRpVoteExtension: late.NonRpVoteExtension,
	})
	require.NoError(t, err)
	require.Equal(t, abci.ResponseVerifyVoteExtension_ACCEPT, result.Status)
	s.votes[3] = late
	votes, err = s.app.signedCommitVotes(s.ctx, nonRpSigHeight, &abci.ExtendedCommitInfo{Votes: s.votes}, &s.vs)
	require.NoError(t, err)
	approved, _, _, err = tallyVotes(votes, s.app.Logger(), &s.vs, nonRpSigHeight)
	require.NoError(t, err)
	require.Equal(t, [][]byte{s.cpTxHash}, approved)
}

func TestMilestoneValidityDoesNotDiscardSideVotes(t *testing.T) {
	s := signedScenario(t)
	updateSignedVote(t, s, 0, func(v *sidetxs.VoteExtension) {
		v.MilestoneProposition = &milestoneTypes.MilestoneProposition{BlockHashes: [][]byte{{1}}, BlockTds: []uint64{1}}
	})
	original := bytes.Clone(s.votes[0].VoteExtension)
	votes, err := s.app.signedCommitVotes(s.ctx, nonRpSigHeight, &abci.ExtendedCommitInfo{Votes: s.votes}, &s.vs)
	require.NoError(t, err)
	require.Equal(t, original, s.votes[0].VoteExtension)
	var derived sidetxs.VoteExtension
	require.NoError(t, derived.Unmarshal(votes[0].VoteExtension))
	require.Nil(t, derived.MilestoneProposition)
	approved, _, _, err := tallyVotes(votes, s.app.Logger(), &s.vs, nonRpSigHeight)
	require.NoError(t, err)
	require.Equal(t, [][]byte{s.cpTxHash}, approved)
	require.Equal(t, abci.ResponseProcessProposal_ACCEPT, s.processProposal(t, s.commit(t)))
}

func TestPreForkRejectsNonRpHashFieldOnWire(t *testing.T) {
	s := newNonRpSigScenario(t, nonRpSigHeight)
	previous := helper.GetPurleyHeight()
	t.Cleanup(func() { helper.SetPurleyHeight(previous) })
	helper.SetPurleyHeight(nonRpSigHeight)
	for _, data := range [][]byte{nil, {1}} {
		encoded := protowire.AppendTag(bytes.Clone(s.votes[0].VoteExtension), 5, protowire.BytesType)
		encoded = protowire.AppendBytes(encoded, data)
		require.Error(t, rejectVoteEnvelopeFields(encoded, nonRpSigHeight-1))
	}
}

func TestSignedCommitFitsBlockCapacity(t *testing.T) {
	votes := make([]abci.ExtendedVoteInfo, 104)
	for i := range votes {
		votes[i] = abci.ExtendedVoteInfo{
			Validator: abci.Validator{Address: make([]byte, 20), Power: 1 << 60}, BlockIdFlag: cmtproto.BlockIDFlagCommit,
			VoteExtension: make([]byte, maxSignedVESize), NonRpVoteExtension: make([]byte, maxNonRpVoteExtensionSize),
			ExtensionSignature: make([]byte, cmtTypes.MaxSignatureSize), NonRpExtensionSignature: make([]byte, cmtTypes.MaxSignatureSize),
		}
	}
	encoded, err := (&abci.ExtendedCommitInfo{Votes: votes}).Marshal()
	require.NoError(t, err)
	capacity := cmtTypes.MaxDataBytes(2*1024*1024, 1024*1024, 104)
	require.LessOrEqual(t, cmtTypes.ComputeProtoSizeForTxs([]cmtTypes.Tx{encoded}), capacity)
}

func padSignedVote(t *testing.T, s *nonRpSigScenario, target int) {
	t.Helper()
	raw := bytes.Clone(s.votes[0].VoteExtension)
	var extension sidetxs.VoteExtension
	require.NoError(t, extension.Unmarshal(raw))
	// Repeating the height field with its decoded value keeps the payload valid in either format.
	for len(raw) < target {
		raw = protowire.AppendTag(raw, 2, protowire.VarintType)
		raw = protowire.AppendVarint(raw, uint64(extension.Height))
	}
	s.votes[0].VoteExtension = raw
	createSignatureForVoteExtension(t, scenarioVoteHeight, s.keys[0], raw, s.votes[0].NonRpVoteExtension, &s.votes[0])
}

func TestSignedCommitAbsentVotesAndQuorum(t *testing.T) {
	s := signedScenario(t)
	commit := &abci.ExtendedCommitInfo{Votes: s.votes}
	commit.Votes[3].BlockIdFlag = cmtproto.BlockIDFlagAbsent
	_, err := s.app.signedCommitVotes(s.ctx, nonRpSigHeight, commit, &s.vs)
	require.Error(t, err)
	commit.Votes[3] = abci.ExtendedVoteInfo{Validator: s.votes[3].Validator, BlockIdFlag: cmtproto.BlockIDFlagAbsent}
	_, err = s.app.signedCommitVotes(s.ctx, nonRpSigHeight, commit, &s.vs)
	require.NoError(t, err)
	commit.Votes[2] = abci.ExtendedVoteInfo{Validator: s.votes[2].Validator, BlockIdFlag: cmtproto.BlockIDFlagAbsent}
	_, err = s.app.signedCommitVotes(s.ctx, nonRpSigHeight, commit, &s.vs)
	require.Error(t, err)
	_, err = s.app.signedCommitVotes(s.ctx, retrieveVoteExtensionsEnableHeight(s.ctx), commit, &s.vs)
	require.Error(t, err)
	_, err = s.app.signedCommitVotes(s.ctx, retrieveVoteExtensionsEnableHeight(s.ctx), &abci.ExtendedCommitInfo{}, &s.vs)
	require.NoError(t, err)
}
