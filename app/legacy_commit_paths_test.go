package app

import (
	"bytes"
	"fmt"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/helper"
	helpermocks "github.com/0xPolygon/heimdall-v2/helper/mocks"
	borTypes "github.com/0xPolygon/heimdall-v2/x/bor/types"
)

// Before Purley an invalid non-rp majority rejects only proposals that carry a checkpoint.
func TestLegacyNonRpFailurePolicy(t *testing.T) {
	for _, checkpoint := range []bool{false, true} {
		t.Run(fmt.Sprint(checkpoint), func(t *testing.T) {
			s := newNonRpSigScenario(t, nonRpSigHeight)
			previous := helper.GetPurleyHeight()
			t.Cleanup(func() { helper.SetPurleyHeight(previous) })
			helper.SetPurleyHeight(nonRpSigHeight + 1)
			for i := range s.votes {
				s.votes[i].NonRpVoteExtension = bytes.Repeat([]byte{0x42}, 40)
				var err error
				s.votes[i].NonRpExtensionSignature, err = s.keys[i].Sign(s.votes[i].NonRpVoteExtension)
				require.NoError(t, err)
			}
			builder := s.app.txConfig.NewTxBuilder()
			require.NoError(t, builder.SetMsgs(&banktypes.MsgSend{}))
			bank, err := s.app.txConfig.TxEncoder()(builder.GetTx())
			require.NoError(t, err)
			txs := [][]byte{s.commit(t), bank}
			if checkpoint {
				txs = append(txs, s.cpTx)
			}
			err = s.app.validateLegacyNonRpCommit(s.ctx, &abci.RequestProcessProposal{Height: nonRpSigHeight, Txs: txs}, &abci.ExtendedCommitInfo{Votes: s.votes}, &s.vs)
			if checkpoint {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Contains(t, s.logs.String(), "Invalid non-rp vote extension proposal")
			}
		})
	}
}

// From Zurich a Bor outage while checking the non-rp majority does not reject a checkpoint proposal.
func TestLegacyCheckpointBorOutagePolicy(t *testing.T) {
	for _, zurich := range []int64{nonRpSigHeight - 1, nonRpSigHeight + 1} {
		t.Run(fmt.Sprint(zurich), func(t *testing.T) {
			s := newNonRpSigScenario(t, nonRpSigHeight)
			previous := helper.GetPurleyHeight()
			t.Cleanup(func() { helper.SetPurleyHeight(previous) })
			helper.SetPurleyHeight(nonRpSigHeight + 1)
			helper.SetZurichHardforkHeight(zurich)
			caller := new(helpermocks.IContractCaller)
			caller.On("CheckIfBlocksExist", mock.Anything, mock.Anything).Return(false, borTypes.ErrFailedToQueryBor)
			s.app.caller = caller
			err := s.app.validateLegacyNonRpCommit(s.ctx, &abci.RequestProcessProposal{Height: nonRpSigHeight, Txs: [][]byte{s.commit(t), s.cpTx}}, &abci.ExtendedCommitInfo{Votes: s.votes}, &s.vs)
			if zurich > nonRpSigHeight {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			caller.AssertExpectations(t)
		})
	}
}

// A Bor outage only logs during legacy preparation; the votes stay in the proposal.
func TestPrepareLegacyCommitRetainsVotesDuringBorOutage(t *testing.T) {
	s := newNonRpSigScenario(t, nonRpSigHeight)
	previous := helper.GetPurleyHeight()
	t.Cleanup(func() { helper.SetPurleyHeight(previous) })
	helper.SetPurleyHeight(nonRpSigHeight + 1)
	caller := new(helpermocks.IContractCaller)
	caller.On("CheckIfBlocksExist", mock.Anything, mock.Anything).Return(false, borTypes.ErrFailedToQueryBor)
	s.app.caller = caller
	req := &abci.RequestPrepareProposal{Height: nonRpSigHeight, MaxTxBytes: 1_000_000, LocalLastCommit: abci.ExtendedCommitInfo{Votes: s.votes}}
	require.NoError(t, s.app.prepareCommit(s.ctx, req, &s.vs))
	require.Contains(t, s.logs.String(), "Error validating non-rp vote extensions")
	for _, vote := range req.LocalLastCommit.Votes {
		require.NotEmpty(t, vote.VoteExtension)
	}
	caller.AssertExpectations(t)
}

// A late vote for the committed initial block is answered without the InitChain context Commit drops.
func TestPreForkLateVoteDoesNotNeedAnSDKContext(t *testing.T) {
	s := newNonRpSigScenario(t, nonRpSigHeight)
	previous := helper.GetPurleyHeight()
	t.Cleanup(func() { helper.SetPurleyHeight(previous) })
	helper.SetPurleyHeight(100)
	_, err := s.app.FinalizeBlock(&abci.RequestFinalizeBlock{Height: nonRpSigHeight, Time: s.ctx.BlockTime(), Txs: [][]byte{s.commit(t)}, ProposerAddress: s.votes[0].Validator.Address})
	require.NoError(t, err)
	_, err = s.app.Commit()
	require.NoError(t, err)
	require.Equal(t, nonRpSigHeight, s.app.LastBlockHeight())
	response, err := s.app.VerifyVoteExtension(&abci.RequestVerifyVoteExtension{Height: nonRpSigHeight, VoteExtension: []byte{0xff}})
	require.NoError(t, err)
	require.Equal(t, abci.ResponseVerifyVoteExtension_ACCEPT, response.Status)
}
