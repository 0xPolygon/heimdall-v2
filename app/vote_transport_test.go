package app

import (
	"bytes"
	"fmt"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"

	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/sidetxs"
	stakeTypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

func TestVoteEnvelopeTransport(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*abci.RequestVerifyVoteExtension)
		reject bool
	}{
		{name: "signed envelope"},
		// From Purley the payload carries no height: the extension signature CometBFT checks before this
		// call binds it, so the envelope alone accepts another Purley height.
		{name: "changed request height", mutate: func(r *abci.RequestVerifyVoteExtension) { r.Height++ }},
		{name: "changed request block", reject: true, mutate: func(r *abci.RequestVerifyVoteExtension) { r.Hash = bytes.Repeat([]byte{7}, 32) }},
		// Only the prefix is carried; the signature binds height and round, which leaves one block to tell apart.
		{name: "request block with the signed prefix", mutate: func(r *abci.RequestVerifyVoteExtension) {
			r.Hash = append(bytes.Clone(r.Hash[:compactVEBlockHashLength]), bytes.Repeat([]byte{7}, 24)...)
		}},
		{name: "explicit height", reject: true, mutate: func(r *abci.RequestVerifyVoteExtension) {
			r.VoteExtension = protowire.AppendVarint(protowire.AppendTag(r.VoteExtension, 2, protowire.VarintType), uint64(r.Height))
		}},
		{name: "malformed protobuf", reject: true, mutate: func(r *abci.RequestVerifyVoteExtension) { r.VoteExtension = []byte{0xff, 0xff} }},
		{name: "unknown field", reject: true, mutate: func(r *abci.RequestVerifyVoteExtension) {
			r.VoteExtension = protowire.AppendBytes(protowire.AppendTag(r.VoteExtension, 20, protowire.BytesType), []byte{1})
		}},
		{name: "changed nr payload", reject: true, mutate: func(r *abci.RequestVerifyVoteExtension) { r.NonRpVoteExtension = bytes.Repeat([]byte{4}, 40) }},
		{name: "duplicate side opinion", reject: true, mutate: func(r *abci.RequestVerifyVoteExtension) {
			var v sidetxs.VoteExtension
			require.NoError(t, v.Unmarshal(r.VoteExtension))
			v.SideTxResponses = append(v.SideTxResponses, v.SideTxResponses[0])
			var err error
			r.VoteExtension, err = v.Marshal()
			require.NoError(t, err)
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := signedScenario(t)
			request := &abci.RequestVerifyVoteExtension{Height: scenarioVoteHeight, Hash: scenarioBlockHash, ValidatorAddress: s.votes[0].Validator.Address, VoteExtension: bytes.Clone(s.votes[0].VoteExtension), NonRpVoteExtension: bytes.Clone(s.votes[0].NonRpVoteExtension)}
			if tc.mutate != nil {
				tc.mutate(request)
			}
			result, err := s.app.VerifyVoteExtension(request)
			require.NoError(t, err)
			require.Equal(t, tc.reject, result.Status == abci.ResponseVerifyVoteExtension_REJECT)
			direct, err := s.app.VerifyVoteExtensionHandler()(s.ctx, request)
			require.NoError(t, err)
			require.Equal(t, result.Status, direct.Status)
		})
	}
}

// From Purley the non-rp floor is the compact placeholder's, below the legacy one.
func TestVoteEnvelopeNrSizeBoundaries(t *testing.T) {
	for _, size := range []int{compactMinNonRpVoteExtensionSize - 1, compactMinNonRpVoteExtensionSize, minNonRpVoteExtensionSize - 1, maxNonRpVoteExtensionSize, maxNonRpVoteExtensionSize + 1} {
		s := signedScenario(t)
		s.votes[0].NonRpVoteExtension = bytes.Repeat([]byte{1}, size)
		updateSignedVote(t, s, 0, func(*sidetxs.VoteExtension) {})
		result, err := s.app.VerifyVoteExtension(&abci.RequestVerifyVoteExtension{Height: scenarioVoteHeight, Hash: scenarioBlockHash, VoteExtension: s.votes[0].VoteExtension, NonRpVoteExtension: s.votes[0].NonRpVoteExtension})
		require.NoError(t, err)
		require.Equal(t, size < compactMinNonRpVoteExtensionSize || size > maxNonRpVoteExtensionSize, result.Status == abci.ResponseVerifyVoteExtension_REJECT, "size %d", size)
	}
}

func TestSignedCommitRejectsIndependentlySignedNrReplay(t *testing.T) {
	s := signedScenario(t)
	old, err := GetDummyNonRpVoteExtension(nonRpSigHeight-2, s.ctx.ChainID())
	require.NoError(t, err)
	signature, err := s.keys[0].Sign(old)
	require.NoError(t, err)
	s.votes[0].NonRpVoteExtension = old
	s.votes[0].NonRpExtensionSignature = signature
	require.Equal(t, abci.ResponseProcessProposal_REJECT, s.processProposal(t, s.commit(t)))
}

func TestSignedCommitFormatStartsAfterVoteActivation(t *testing.T) {
	for _, fork := range []int64{nonRpSigHeight + 1, nonRpSigHeight, nonRpSigHeight - 1} {
		t.Run(fmt.Sprint(fork), func(t *testing.T) {
			s := newNonRpSigScenario(t, nonRpSigHeight)
			previous := helper.GetPurleyHeight()
			t.Cleanup(func() { helper.SetPurleyHeight(previous) })
			helper.SetPurleyHeight(fork)
			for i := range s.votes {
				updateSignedVote(t, s, i, func(*sidetxs.VoteExtension) {})
			}
			s.votes[0] = abci.ExtendedVoteInfo{Validator: s.votes[0].Validator, BlockIdFlag: cmtproto.BlockIDFlagCommit}
			want := abci.ResponseProcessProposal_ACCEPT
			if fork < nonRpSigHeight {
				want = abci.ResponseProcessProposal_REJECT
			}
			require.Equal(t, want, s.processProposal(t, s.commit(t)))
		})
	}
}

func TestSignedCommitRequiresStrictSupermajority(t *testing.T) {
	s := signedScenario(t)
	validators := stakeTypes.NewValidatorSet(s.vs.Validators[:3])
	votes := append([]abci.ExtendedVoteInfo(nil), s.votes[:3]...)
	votes[2] = abci.ExtendedVoteInfo{Validator: votes[2].Validator, BlockIdFlag: cmtproto.BlockIDFlagAbsent}
	_, err := s.app.signedCommitVotes(s.ctx, nonRpSigHeight, &abci.ExtendedCommitInfo{Votes: votes}, validators)
	require.Error(t, err)
}

func TestSignedCommitRejectsNonCommitPayload(t *testing.T) {
	for mask := 1; mask < 16; mask++ {
		t.Run(fmt.Sprint(mask), func(t *testing.T) {
			s := signedScenario(t)
			vote := abci.ExtendedVoteInfo{Validator: s.votes[3].Validator, BlockIdFlag: cmtproto.BlockIDFlagAbsent}
			fields := []*[]byte{&vote.VoteExtension, &vote.NonRpVoteExtension, &vote.ExtensionSignature, &vote.NonRpExtensionSignature}
			for i, field := range fields {
				if mask&(1<<i) != 0 {
					*field = []byte{1}
				}
			}
			s.votes[3] = vote
			_, err := s.app.signedCommitVotes(s.ctx, nonRpSigHeight, &abci.ExtendedCommitInfo{Votes: s.votes}, &s.vs)
			require.Error(t, err)
		})
	}
}

func TestSignedCommitRejectsInvalidSnapshotKey(t *testing.T) {
	s := signedScenario(t)
	s.vs.Validators[0].PubKey = nil
	require.NotPanics(t, func() {
		_, err := s.app.signedCommitVotes(s.ctx, nonRpSigHeight, &abci.ExtendedCommitInfo{Votes: s.votes}, &s.vs)
		require.Error(t, err)
	})
}
