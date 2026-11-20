package app

import (
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govlegacy "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/sidetxs"
)

const zeroRatio = "0.000000000000000000"

var purleyBoundaryCases = []struct {
	name         string
	purleyHeight int64
	height       int64
	normalized   bool
}{
	{"purley disabled", 0, 5, false},
	{"h-1", 5, 4, false},
	{"h", 5, 5, true},
	{"h+1", 5, 6, true},
}

func setPurleyHeight(t *testing.T, height int64) {
	previous := helper.GetPurleyHeight()
	t.Cleanup(func() { helper.SetPurleyHeight(previous) })
	helper.SetPurleyHeight(height)
}

func TestGovCancellationChargeAcrossPurleyBoundary(t *testing.T) {
	for _, tc := range purleyBoundaryCases {
		t.Run(tc.name, func(t *testing.T) {
			priv, app, ctx, _ := SetupAppWithABCICtx(t)
			setPurleyHeight(t, tc.purleyHeight)
			ctx = ctx.WithBlockHeight(tc.height)

			proposer := sdk.AccAddress(priv.PubKey().Address())
			gov := app.GovKeeper.GetGovernanceAccount(ctx).GetAddress()
			params, err := app.GovKeeper.Params.Get(ctx)
			require.NoError(t, err)
			params.ProposalCancelDest = ""
			params.ProposalCancelRatio = "0.500000000000000000"
			require.NoError(t, app.GovKeeper.Params.Set(ctx, params))

			govBefore := app.BankKeeper.GetAllBalances(ctx, gov)
			content, err := govv1.NewLegacyContent(govlegacy.NewTextProposal("title", "description"), gov.String())
			require.NoError(t, err)
			proposal, err := app.GovKeeper.SubmitProposal(ctx, []sdk.Msg{content}, "", "title", "summary", proposer, false)
			require.NoError(t, err)
			deposit := sdk.Coins(params.MinDeposit)
			_, err = app.GovKeeper.AddDeposit(ctx, proposal.Id, proposer, deposit)
			require.NoError(t, err)
			proposerBefore := app.BankKeeper.GetAllBalances(ctx, proposer)

			require.NoError(t, app.normalizeCancelCharge(ctx, tc.height))
			require.NoError(t, app.GovKeeper.CancelProposal(ctx, proposal.Id, proposer.String()))

			stored, err := app.GovKeeper.Params.Get(ctx)
			require.NoError(t, err)
			wantRatio, wantRefund := "0.500000000000000000", sdk.NewCoins()
			if tc.normalized {
				wantRatio, wantRefund = zeroRatio, deposit
			} else {
				for _, c := range deposit {
					wantRefund = wantRefund.Add(sdk.NewCoin(c.Denom, c.Amount.QuoRaw(2)))
				}
			}
			require.Equal(t, wantRatio, stored.ProposalCancelRatio)
			require.Equal(t, wantRefund.String(), app.BankKeeper.GetAllBalances(ctx, proposer).Sub(proposerBefore...).String())
			require.Equal(t, deposit.Sub(wantRefund...).String(), app.BankKeeper.GetAllBalances(ctx, gov).Sub(govBefore...).String())
		})
	}
}

// PreBlocker resets the ratio on every Purley block, so a later param update that restores a charge
// without a destination is undone before the next block's cancel txs run.
func TestGovPreBlockerNormalizesCancelChargeAcrossPurleyBoundary(t *testing.T) {
	for _, tc := range purleyBoundaryCases {
		t.Run(tc.name, func(t *testing.T) {
			s := newNonRpSigScenario(t, nonRpSigHeight)
			purley := tc.purleyHeight
			if purley != 0 {
				purley += nonRpSigHeight - tc.height
			}
			setPurleyHeight(t, purley)
			if helper.IsPurley(scenarioVoteHeight) {
				for i := range s.votes {
					updateSignedVote(t, s, i, func(*sidetxs.VoteExtension) {})
				}
			}

			params, err := s.app.GovKeeper.Params.Get(s.ctx)
			require.NoError(t, err)
			params.ProposalCancelDest = ""
			params.ProposalCancelRatio = "0.500000000000000000"
			require.NoError(t, s.app.GovKeeper.Params.Set(s.ctx, params))

			_, err = s.app.PreBlocker(s.ctx, &abci.RequestFinalizeBlock{Height: nonRpSigHeight, Txs: [][]byte{s.commit(t)}, ProposerAddress: s.votes[0].Validator.Address})
			require.NoError(t, err)
			got, err := s.app.GovKeeper.Params.Get(s.ctx)
			require.NoError(t, err)
			require.Equal(t, tc.normalized, got.ProposalCancelRatio == zeroRatio)
		})
	}
}

func TestGovCancelChargeLeftAlone(t *testing.T) {
	priv, app, ctx, _ := SetupAppWithABCICtx(t)
	setPurleyHeight(t, 1)
	for _, tc := range []struct{ name, dest, ratio string }{
		{"destination set", sdk.AccAddress(priv.PubKey().Address()).String(), "0.500000000000000000"},
		{"ratio already 0", "", "0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			params, err := app.GovKeeper.Params.Get(ctx)
			require.NoError(t, err)
			params.ProposalCancelDest, params.ProposalCancelRatio = tc.dest, tc.ratio
			require.NoError(t, app.GovKeeper.Params.Set(ctx, params))
			require.NoError(t, app.normalizeCancelCharge(ctx, 1))
			got, err := app.GovKeeper.Params.Get(ctx)
			require.NoError(t, err)
			require.Equal(t, params, got)
		})
	}
}

// A Purley block fails rather than skip the cancel charge normalization.
func TestGovPreBlockerStopsWithoutParams(t *testing.T) {
	s := newNonRpSigScenario(t, nonRpSigHeight)
	setPurleyHeight(t, nonRpSigHeight)
	require.NoError(t, s.app.GovKeeper.Params.Remove(s.ctx))
	_, err := s.app.PreBlocker(s.ctx, &abci.RequestFinalizeBlock{Height: nonRpSigHeight, Txs: [][]byte{s.commit(t)}, ProposerAddress: s.votes[0].Validator.Address})
	require.Error(t, err)
}
