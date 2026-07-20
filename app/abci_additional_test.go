package app

import (
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/helper"
	borTypes "github.com/0xPolygon/heimdall-v2/x/bor/types"
	clerkTypes "github.com/0xPolygon/heimdall-v2/x/clerk/types"
	stakeTypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
	topupTypes "github.com/0xPolygon/heimdall-v2/x/topup/types"
)

func TestPrepareProposalHandler_SkipsBorSideMessages(t *testing.T) {
	priv, app, ctx, _ := SetupAppWithABCICtxAndValidators(t, 3)
	validators := app.StakeKeeper.GetAllValidators(ctx)
	seedSpan(t, app, ctx)
	emptyCommit := &abci.ExtendedCommitInfo{}

	t.Run("skips MsgVoteProducers before rio", func(t *testing.T) {
		origRio := helper.GetRioHeight()
		t.Cleanup(func() { helper.SetRioHeight(origRio) })

		lastSpan, err := app.BorKeeper.GetLastSpan(ctx)
		require.NoError(t, err)
		helper.SetRioHeight(int64(lastSpan.EndBlock + 2))

		msg := &borTypes.MsgVoteProducers{
			Voter:   validators[0].Signer,
			VoterId: validators[0].ValId,
			Votes:   borTypes.ProducerVotes{Votes: []uint64{validators[1].ValId}},
		}
		txBytes, err := buildSignedTx(msg, ctx, priv, app)
		require.NoError(t, err)

		resp, err := app.NewPrepareProposalHandler()(ctx, &abci.RequestPrepareProposal{
			Txs:             [][]byte{txBytes},
			MaxTxBytes:      1_000_000,
			LocalLastCommit: *emptyCommit,
			ProposerAddress: common.FromHex(validators[0].Signer),
			Height:          1,
		})
		require.NoError(t, err)
		require.Len(t, resp.Txs, 1)
	})

	t.Run("skips MsgSetProducerDowntime with target before fork", func(t *testing.T) {
		origZurich := helper.GetZurichHardforkHeight()
		t.Cleanup(func() { helper.SetZurichHardforkHeight(origZurich) })
		helper.SetZurichHardforkHeight(1_000_000)

		msg := &borTypes.MsgSetProducerDowntime{
			Producer: validators[0].Signer,
			DowntimeRange: borTypes.BlockRange{
				StartBlock: 100,
				EndBlock:   200,
			},
			TargetProducerId: 42,
		}
		txBytes, err := buildSignedTx(msg, ctx, priv, app)
		require.NoError(t, err)

		resp, err := app.NewPrepareProposalHandler()(ctx, &abci.RequestPrepareProposal{
			Txs:             [][]byte{txBytes},
			MaxTxBytes:      1_000_000,
			LocalLastCommit: *emptyCommit,
			ProposerAddress: common.FromHex(validators[0].Signer),
			Height:          1,
		})
		require.NoError(t, err)
		require.Len(t, resp.Txs, 1)
	})
}

func TestProcessProposalHandler_RejectsBorSideMessages(t *testing.T) {
	priv, app, ctx, _ := SetupAppWithABCICtxAndValidators(t, 3)
	validators := app.StakeKeeper.GetAllValidators(ctx)
	seedSpan(t, app, ctx)
	emptyCommit := &abci.ExtendedCommitInfo{}
	extCommitBytes, err := emptyCommit.Marshal()
	require.NoError(t, err)

	t.Run("rejects MsgVoteProducers before rio", func(t *testing.T) {
		origRio := helper.GetRioHeight()
		t.Cleanup(func() { helper.SetRioHeight(origRio) })

		lastSpan, err := app.BorKeeper.GetLastSpan(ctx)
		require.NoError(t, err)
		helper.SetRioHeight(int64(lastSpan.EndBlock + 2))

		msg := &borTypes.MsgVoteProducers{
			Voter:   validators[0].Signer,
			VoterId: validators[0].ValId,
			Votes:   borTypes.ProducerVotes{Votes: []uint64{validators[1].ValId}},
		}
		txBytes, err := buildSignedTx(msg, ctx, priv, app)
		require.NoError(t, err)

		resp, err := app.NewProcessProposalHandler()(ctx, &abci.RequestProcessProposal{
			Txs:                [][]byte{extCommitBytes, txBytes},
			Height:             1,
			ProposedLastCommit: abci.CommitInfo{Round: emptyCommit.Round},
		})
		require.NoError(t, err)
		require.Equal(t, abci.ResponseProcessProposal_REJECT, resp.Status)
	})

	t.Run("rejects MsgSetProducerDowntime with target before fork", func(t *testing.T) {
		origZurich := helper.GetZurichHardforkHeight()
		t.Cleanup(func() { helper.SetZurichHardforkHeight(origZurich) })
		helper.SetZurichHardforkHeight(1_000_000)

		msg := &borTypes.MsgSetProducerDowntime{
			Producer: validators[0].Signer,
			DowntimeRange: borTypes.BlockRange{
				StartBlock: 100,
				EndBlock:   200,
			},
			TargetProducerId: 42,
		}
		txBytes, err := buildSignedTx(msg, ctx, priv, app)
		require.NoError(t, err)

		resp, err := app.NewProcessProposalHandler()(ctx, &abci.RequestProcessProposal{
			Txs:                [][]byte{extCommitBytes, txBytes},
			Height:             1,
			ProposedLastCommit: abci.CommitInfo{Round: emptyCommit.Round},
		})
		require.NoError(t, err)
		require.Equal(t, abci.ResponseProcessProposal_REJECT, resp.Status)
	})
}

func TestExtractTxHash(t *testing.T) {
	validHash := common.BigToHash(common.Big1)
	validBytes := validHash.Bytes()

	tests := []struct {
		name string
		msg  sdk.Msg
		want common.Hash
		ok   bool
	}{
		{
			name: "clerk event record",
			msg:  &clerkTypes.MsgEventRecord{TxHash: validHash.Hex()},
			want: validHash,
			ok:   true,
		},
		{
			name: "validator join",
			msg: &stakeTypes.MsgValidatorJoin{
				TxHash: validBytes,
			},
			want: validHash,
			ok:   true,
		},
		{
			name: "stake update",
			msg: &stakeTypes.MsgStakeUpdate{
				TxHash: validBytes,
			},
			want: validHash,
			ok:   true,
		},
		{
			name: "signer update",
			msg: &stakeTypes.MsgSignerUpdate{
				TxHash: validBytes,
			},
			want: validHash,
			ok:   true,
		},
		{
			name: "validator exit",
			msg: &stakeTypes.MsgValidatorExit{
				TxHash: validBytes,
			},
			want: validHash,
			ok:   true,
		},
		{
			name: "topup tx",
			msg: &topupTypes.MsgTopupTx{
				TxHash: validBytes,
			},
			want: validHash,
			ok:   true,
		},
		{
			name: "invalid hex hash",
			msg:  &clerkTypes.MsgEventRecord{TxHash: "0x1234"},
			want: common.Hash{},
			ok:   false,
		},
		{
			name: "unsupported message",
			msg:  &borTypes.MsgVoteProducers{},
			want: common.Hash{},
			ok:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			hash, ok := extractTxHash(tc.msg)
			require.Equal(t, tc.ok, ok)
			require.Equal(t, tc.want, hash)
		})
	}

	require.True(t, verifyTxHash(validBytes))
	require.False(t, verifyTxHash(validBytes[:31]))
	require.True(t, verifyHexTxHash(validHash.Hex()))
	require.False(t, verifyHexTxHash("0x1234"))
	require.False(t, verifyHexTxHash(common.Bytes2Hex(validBytes[:31])))
}
