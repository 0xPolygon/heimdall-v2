package app

import (
	"bytes"
	"math/big"
	"testing"

	"cosmossdk.io/log"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/helper"
)

func TestGetCheckpointSignatures_CanonicalFromKarwar(t *testing.T) {
	const height = int64(10)

	extension := []byte("checkpoint extension")
	digest := crypto.Keccak256(extension)

	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	signer := crypto.PubkeyToAddress(key.PublicKey).Bytes()
	genuine, err := crypto.Sign(digest, key)
	require.NoError(t, err)

	flipped := bytes.Clone(genuine)
	flipped[crypto.RecoveryIDOffset] ^= 1

	highS := bytes.Clone(genuine)
	n := crypto.S256().Params().N
	new(big.Int).Sub(n, new(big.Int).SetBytes(genuine[32:64])).FillBytes(highS[32:64])
	highS[crypto.RecoveryIDOffset] ^= 1

	invalid := bytes.Repeat([]byte{0x42}, crypto.SignatureLength)

	vote := func(ext, sig []byte) abci.ExtendedVoteInfo {
		return abci.ExtendedVoteInfo{
			Validator:               abci.Validator{Address: signer},
			BlockIdFlag:             cmtproto.BlockIDFlagCommit,
			NonRpVoteExtension:      ext,
			NonRpExtensionSignature: sig,
		}
	}

	tests := []struct {
		name   string
		karwar int64
		votes  []abci.ExtendedVoteInfo
		want   [][]byte
	}{
		{
			name:   "before karwar: signatures stored as received",
			karwar: height + 1,
			votes:  []abci.ExtendedVoteInfo{vote(extension, genuine), vote(extension, flipped), vote(extension, highS), vote(extension, invalid)},
			want:   [][]byte{genuine, flipped, highS, invalid},
		},
		{
			name:   "dormant: signatures stored as received",
			karwar: 0,
			votes:  []abci.ExtendedVoteInfo{vote(extension, flipped), vote(extension, invalid)},
			want:   [][]byte{flipped, invalid},
		},
		{
			name:   "at karwar: recoverable signatures are canonicalized and the rest dropped",
			karwar: height,
			votes:  []abci.ExtendedVoteInfo{vote(extension, genuine), vote(extension, flipped), vote(extension, highS), vote(extension, invalid)},
			want:   [][]byte{genuine, genuine, genuine},
		},
		{
			name:   "after karwar: other payloads are excluded",
			karwar: height - 1,
			votes:  []abci.ExtendedVoteInfo{vote(extension, genuine), vote([]byte("other extension"), genuine)},
			want:   [][]byte{genuine},
		},
	}

	original := helper.GetKarwarHeight()
	t.Cleanup(func() { helper.SetKarwarHeight(original) })

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			helper.SetKarwarHeight(tc.karwar)

			got := getCheckpointSignatures(height, extension, tc.votes).Signatures
			require.Len(t, got, len(tc.want))
			for i, sig := range tc.want {
				require.Equal(t, sig, got[i].Signature)
				require.Equal(t, signer, got[i].ValidatorAddress)
			}
		})
	}
}

// The signatures belong to the previous block's checkpoint, so from Karwar a proposal that carries an
// invalid one is rejected even when it holds no checkpoint tx of its own.
func TestProcessProposal_InvalidNonRpSignatureRejectedFromKarwar(t *testing.T) {
	const runHeight = int64(3)

	helper.SetPhuketHardforkHeight(1)
	t.Cleanup(func() { helper.SetPhuketHardforkHeight(0) })

	cases := []struct {
		name       string
		activation int64
		wantStatus abci.ResponseProcessProposal_ProposalStatus
	}{
		{name: "below activation: accepts", activation: runHeight + 1, wantStatus: abci.ResponseProcessProposal_ACCEPT},
		{name: "at activation: rejects", activation: runHeight, wantStatus: abci.ResponseProcessProposal_REJECT},
		{name: "above activation: rejects", activation: runHeight - 1, wantStatus: abci.ResponseProcessProposal_REJECT},
	}

	original := helper.GetKarwarHeight()
	t.Cleanup(func() { helper.SetKarwarHeight(original) })

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			helper.SetKarwarHeight(tc.activation)

			_, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
			validators := app.StakeKeeper.GetAllValidators(ctx)

			_, extCommit, _, err := buildExtensionCommits(t, app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, 2, nil)
			require.NoError(t, err)
			extCommit.Votes[0].NonRpExtensionSignature = bytes.Repeat([]byte{0x42}, crypto.SignatureLength)
			extCommitBytes, err := extCommit.Marshal()
			require.NoError(t, err)

			res, err := app.ProcessProposal(&abci.RequestProcessProposal{
				Txs:    [][]byte{extCommitBytes},
				Height: runHeight,
				ProposedLastCommit: abci.CommitInfo{
					Round: extCommit.Round,
					Votes: []abci.VoteInfo{{Validator: extCommit.Votes[0].Validator, BlockIdFlag: cmtproto.BlockIDFlagCommit}},
				},
			})
			require.NoError(t, err)
			require.NotNil(t, res)
			require.Equal(t, tc.wantStatus, res.Status)
		})
	}
}

func TestValidateNonRpVoteExtensions_SignatureCheckedBeforeMajority(t *testing.T) {
	const runHeight = int64(3)

	helper.SetPhuketHardforkHeight(1)
	t.Cleanup(func() { helper.SetPhuketHardforkHeight(0) })

	_, app, ctx, validatorPrivKeys := SetupAppWithABCICtxAndValidators(t, 4)
	ctx = ctx.WithBlockHeight(runHeight)
	validators := app.StakeKeeper.GetAllValidators(ctx)
	valSet, err := app.StakeKeeper.GetValidatorSet(ctx)
	require.NoError(t, err)

	dummyExt, err := GetDummyNonRpVoteExtension(runHeight-1, ctx.ChainID())
	require.NoError(t, err)

	votes := make([]abci.ExtendedVoteInfo, len(validators))
	for i, val := range validators {
		sig, err := keyForValidator(t, val, validatorPrivKeys).Sign(dummyExt)
		require.NoError(t, err)
		votes[i] = abci.ExtendedVoteInfo{
			Validator:               abci.Validator{Address: common.FromHex(val.Signer), Power: val.VotingPower},
			BlockIdFlag:             cmtproto.BlockIDFlagCommit,
			NonRpVoteExtension:      dummyExt,
			NonRpExtensionSignature: sig,
		}
	}

	validate := func() error {
		return ValidateNonRpVoteExtensions(ctx, runHeight, votes, &valSet, app.ChainManagerKeeper, app.CheckpointKeeper, app.caller, log.NewNopLogger())
	}

	require.NoError(t, validate())

	// Two diverging payloads split the majority and fail signature verification.
	votes[0].NonRpVoteExtension = append(bytes.Clone(dummyExt), 0x00)
	votes[1].NonRpVoteExtension = append(bytes.Clone(dummyExt), 0x01)

	require.ErrorIs(t, validate(), errNonRpSignature)

	// Failures other than a bad signature carry the same sentinel, so ProcessProposal rejects on them too.
	votes[0].Validator.Address = common.FromHex("0x00000000000000000000000000000000000000ff")
	err = validate()
	require.ErrorIs(t, err, errNonRpSignature)
	require.ErrorContains(t, err, "Failed to get validator")
}
