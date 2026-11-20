package app

import (
	"math/big"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/sidetxs"
	stakeTypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

// From Purley a proposal rejects stake updates past the next stored nonce, and PrepareProposal
// leaves them out. Stale ones stay in, so the relayer's following txs keep their sequence.
func TestProposalSerializesStakeEvents(t *testing.T) {
	const height = VoteExtBlockHeight + 1
	tests := []struct {
		name    string
		fork    int64
		offsets []uint64
		reject  bool
		kept    []int // indices into offsets that PrepareProposal keeps
	}{
		{"before fork reordered", height + 1, []uint64{2, 1}, false, []int{0, 1}},
		{"at fork next", height, []uint64{1}, false, []int{0}},
		{"at fork future", height, []uint64{2}, true, nil},
		{"after fork next", height - 1, []uint64{1}, false, []int{0}},
		{"after fork reordered", height - 1, []uint64{2, 1}, true, nil}, // dropping the first leaves a sequence gap
		{"after fork consecutive", height - 1, []uint64{1, 2}, true, []int{0}},
		{"after fork same nonce twice", height - 1, []uint64{1, 1}, false, []int{0, 1}},
		{"after fork stale", height - 1, []uint64{0}, false, []int{0}},
		{"after fork stale then next", height - 1, []uint64{0, 1}, false, []int{0, 1}},
		{"after fork next then stale", height - 1, []uint64{1, 0}, false, []int{0, 1}},
		{"after fork stale then future", height - 1, []uint64{0, 2}, true, []int{0}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			priv, app, ctx, keys := SetupAppWithABCICtxAndValidators(t, 2, log.NewNopLogger())
			previous := helper.GetPurleyHeight()
			t.Cleanup(func() { helper.SetPurleyHeight(previous) })
			helper.SetPurleyHeight(tc.fork)
			validators := app.StakeKeeper.GetAllValidators(ctx)
			var target *stakeTypes.Validator
			for _, val := range validators {
				if val.ValId != 0 {
					target = val
					break
				}
			}
			require.NotNil(t, target)
			dummy, err := GetDummyNonRpVoteExtension(height-1, app.ChainID())
			require.NoError(t, err)
			ve := &sidetxs.VoteExtension{Height: height - 1, BlockHash: common.BigToHash(big.NewInt(1)).Bytes()}
			if helper.IsPurley(height - 1) {
				ve = &sidetxs.VoteExtension{BlockHash: ve.BlockHash[:compactVEBlockHashLength], NonRpExtensionHash: nonRpExtensionHash(height-1, dummy)}
			}
			encoded, err := ve.Marshal()
			require.NoError(t, err)
			var votes []abci.ExtendedVoteInfo
			var canonical abci.CommitInfo
			for _, val := range validators {
				vote := abci.ExtendedVoteInfo{Validator: abci.Validator{Address: common.FromHex(val.Signer), Power: val.VotingPower}, BlockIdFlag: cmtproto.BlockIDFlagCommit, VoteExtension: encoded, NonRpVoteExtension: dummy}
				createSignatureForVoteExtension(t, height-1, keyForValidator(t, val, keys), encoded, dummy, &vote)
				votes = append(votes, vote)
				canonical.Votes = append(canonical.Votes, abci.VoteInfo{Validator: vote.Validator, BlockIdFlag: vote.BlockIdFlag})
			}
			commit, err := (&abci.ExtendedCommitInfo{Votes: votes}).Marshal()
			require.NoError(t, err)
			txs := [][]byte{commit}
			seq := app.AccountKeeper.GetAccount(ctx, sdk.AccAddress(priv.PubKey().Address())).GetSequence()
			for i, offset := range tc.offsets {
				nonce := target.Nonce + offset
				msg, err := stakeTypes.NewMsgStakeUpdate(priv.PubKey().Address().String(), target.ValId, math.NewInt(1_000_000_000_000_000_000), common.BigToHash(new(big.Int).SetUint64(nonce+1)).Bytes(), 0, nonce+1, nonce)
				require.NoError(t, err)
				tx, err := buildSignedTxWithSequence(msg, ctx, priv, app, seq+uint64(i))
				require.NoError(t, err)
				txs = append(txs, tx)
			}
			response, err := app.ProcessProposal(&abci.RequestProcessProposal{Height: height, Txs: txs, ProposedLastCommit: canonical})
			require.NoError(t, err)
			require.Equal(t, tc.reject, response.Status == abci.ResponseProcessProposal_REJECT)
			prepared, err := app.PrepareProposal(&abci.RequestPrepareProposal{Height: height, MaxTxBytes: 1_000_000, Txs: txs[1:], LocalLastCommit: abci.ExtendedCommitInfo{Votes: votes}})
			require.NoError(t, err)
			want := [][]byte{commit}
			for _, i := range tc.kept {
				want = append(want, txs[i+1])
			}
			require.Equal(t, want, prepared.Txs)
			response, err = app.ProcessProposal(&abci.RequestProcessProposal{Height: height, Txs: prepared.Txs, ProposedLastCommit: canonical})
			require.NoError(t, err)
			require.Equal(t, abci.ResponseProcessProposal_ACCEPT, response.Status)
		})
	}
}

// From Purley, CheckTx and ReCheck keep a stale stake update and the same relayer's next tx,
// and reject a nonce past the next one. Before Purley every nonce is admitted.
func TestCheckTxStakeNonceAdmission(t *testing.T) {
	tests := []struct {
		name    string
		purley  bool
		offsets []uint64
		admit   []bool
	}{
		{"after fork stale then next", true, []uint64{0, 1}, []bool{true, true}},
		{"after fork future", true, []uint64{2}, []bool{false}},
		{"before fork future", false, []uint64{2}, []bool{true}},
	}
	for _, tc := range tests {
		for _, kind := range []abci.CheckTxType{abci.CheckTxType_New, abci.CheckTxType_Recheck} {
			t.Run(tc.name+" "+kind.String(), func(t *testing.T) {
				priv, app, ctx, _ := SetupAppWithABCICtxAndValidators(t, 2, log.NewNopLogger())
				previous := helper.GetPurleyHeight()
				t.Cleanup(func() { helper.SetPurleyHeight(previous) })
				helper.SetPurleyHeight(0)
				if tc.purley {
					helper.SetPurleyHeight(app.LastBlockHeight() + 1)
				}
				var target *stakeTypes.Validator
				for _, val := range app.StakeKeeper.GetAllValidators(ctx) {
					if val.ValId != 0 {
						target = val
						break
					}
				}
				require.NotNil(t, target)
				seq := app.AccountKeeper.GetAccount(ctx, sdk.AccAddress(priv.PubKey().Address())).GetSequence()
				for i, offset := range tc.offsets {
					nonce := target.Nonce + offset
					msg, err := stakeTypes.NewMsgStakeUpdate(priv.PubKey().Address().String(), target.ValId, math.NewInt(1_000_000_000_000_000_000), common.BigToHash(new(big.Int).SetUint64(nonce+1)).Bytes(), 0, nonce+1, nonce)
					require.NoError(t, err)
					tx, err := buildSignedTxWithSequence(msg, ctx, priv, app, seq+uint64(i))
					require.NoError(t, err)
					resp, err := app.CheckTx(&abci.RequestCheckTx{Tx: tx, Type: kind})
					require.NoError(t, err)
					require.Equal(t, tc.admit[i], resp.Code == abci.CodeTypeOK, "tx %d: %s", i, resp.Log)
				}
			})
		}
	}
}
