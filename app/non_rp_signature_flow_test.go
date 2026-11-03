package app

import (
	"bytes"
	"testing"
	"time"

	"cosmossdk.io/log"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmtTypes "github.com/cometbft/cometbft/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/helper"
	helpermocks "github.com/0xPolygon/heimdall-v2/helper/mocks"
	"github.com/0xPolygon/heimdall-v2/sidetxs"
	stakeTypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

// nonRpSigHeight is the proposal height; block nonRpSigHeight-1 carried the checkpoint tx the votes approve.
const nonRpSigHeight int64 = 3

var nonRpSigForks = []struct {
	name     string
	karwar   int64
	enforced bool // invalid signatures reject the proposal; stored signatures are canonicalized
}{
	{"before karwar", nonRpSigHeight + 1, false},
	{"at karwar", nonRpSigHeight, true},
}

// keyForValidator returns the key that signs for val; the keys SetupApp returns are not in validator order.
func keyForValidator(t *testing.T, val *stakeTypes.Validator, keys []secp256k1.PrivKey) secp256k1.PrivKey {
	t.Helper()

	pubKey, err := getValidatorPublicKey(val)
	require.NoError(t, err)

	probe := []byte("probe")
	for _, key := range keys {
		sig, err := key.Sign(probe)
		require.NoError(t, err)
		if pubKey.VerifySignature(probe, sig) {
			return key
		}
	}

	require.FailNow(t, "no key matches validator", val.Signer)
	return nil
}

type nonRpSigScenario struct {
	app      *HeimdallApp
	ctx      sdk.Context
	vs       stakeTypes.ValidatorSet
	cpTxHash []byte
	endBlock uint64
	votes    []abci.ExtendedVoteInfo
	genuine  [][]byte
	logs     *bytes.Buffer
}

func newNonRpSigScenario(t *testing.T, karwar int64) *nonRpSigScenario {
	t.Helper()

	setHeight := func(get func() int64, set func(int64), height int64) {
		original := get()
		t.Cleanup(func() { set(original) })
		set(height)
	}
	setHeight(helper.GetPhuketHardforkHeight, helper.SetPhuketHardforkHeight, 1)
	setHeight(helper.GetZurichHardforkHeight, helper.SetZurichHardforkHeight, 1)
	setHeight(helper.GetIthacaHeight, helper.SetIthacaHeight, 1)
	setHeight(helper.GetKyotoHeight, helper.SetKyotoHeight, 1)
	setHeight(helper.GetKarwarHeight, helper.SetKarwarHeight, karwar)

	logs := new(bytes.Buffer)
	priv, app, ctx, keys := SetupAppWithABCICtxAndValidators(t, 4, log.NewLogger(logs, log.ColorOption(false)))
	ctx = ctx.WithBlockHeight(nonRpSigHeight).WithBlockTime(time.Unix(1700000000, 0))

	vs, err := app.StakeKeeper.GetValidatorSet(ctx)
	require.NoError(t, err)
	params, err := app.CheckpointKeeper.GetParams(ctx)
	require.NoError(t, err)

	end := params.AvgCheckpointLength - 1
	_, msg := buildCheckpointNonRpVoteExt(t, app, ctx, priv.PubKey().Address().String(), end)
	msg.StartBlock = 0
	nonRp := packExtensionWithVote(msg.GetSideSignBytes())
	txBytes, err := buildSignedTx(msg, ctx, priv, app)
	require.NoError(t, err)
	require.NoError(t, app.StakeKeeper.SetLastBlockTxs(ctx, [][]byte{txBytes}))
	txHash := cmtTypes.Tx(txBytes).Hash()

	caller := new(helpermocks.IContractCaller)
	caller.On("CheckIfBlocksExist", mock.Anything, mock.Anything).Return(true, nil)
	caller.On("GetRootHash", mock.Anything, msg.StartBlock, msg.EndBlock, params.MaxCheckpointLength).Return(msg.RootHash, nil)
	app.caller = caller

	ve := sidetxs.VoteExtension{
		Height:          nonRpSigHeight - 1,
		BlockHash:       common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"),
		SideTxResponses: []sidetxs.SideTxResponse{{TxHash: txHash, Result: sidetxs.Vote_VOTE_YES}},
	}
	veBytes, err := ve.Marshal()
	require.NoError(t, err)

	s := &nonRpSigScenario{app: app, ctx: ctx, vs: vs, cpTxHash: txHash, endBlock: end, logs: logs}
	for _, val := range vs.Validators {
		vote := abci.ExtendedVoteInfo{
			BlockIdFlag:        cmtproto.BlockIDFlagCommit,
			VoteExtension:      veBytes,
			Validator:          abci.Validator{Address: common.FromHex(val.Signer), Power: val.VotingPower},
			NonRpVoteExtension: nonRp,
		}
		createSignatureForVoteExtension(t, nonRpSigHeight-1, keyForValidator(t, val, keys), veBytes, nonRp, &vote)
		s.votes = append(s.votes, vote)
		s.genuine = append(s.genuine, bytes.Clone(vote.NonRpExtensionSignature))
	}

	return s
}

func (s *nonRpSigScenario) commit(t *testing.T) []byte {
	t.Helper()
	bz, err := (&abci.ExtendedCommitInfo{Votes: s.votes}).Marshal()
	require.NoError(t, err)
	return bz
}

func (s *nonRpSigScenario) processProposal(t *testing.T, commit []byte) abci.ResponseProcessProposal_ProposalStatus {
	t.Helper()
	committing := make([]abci.VoteInfo, 0, len(s.votes))
	for _, v := range s.votes {
		committing = append(committing, abci.VoteInfo{Validator: v.Validator, BlockIdFlag: cmtproto.BlockIDFlagCommit})
	}
	res, err := s.app.NewProcessProposalHandler()(s.ctx, &abci.RequestProcessProposal{
		Height:             nonRpSigHeight,
		Txs:                [][]byte{commit},
		ProposedLastCommit: abci.CommitInfo{Votes: committing},
	})
	require.NoError(t, err)
	return res.Status
}

func (s *nonRpSigScenario) requireSignaturesStored(t *testing.T, want [][]byte) {
	t.Helper()
	_, err := s.app.PreBlocker(s.ctx, &abci.RequestFinalizeBlock{Height: nonRpSigHeight, Txs: [][]byte{s.commit(t)}, ProposerAddress: s.votes[0].Validator.Address})
	require.NoError(t, err)

	sigs, err := s.app.CheckpointKeeper.GetCheckpointSignatures(s.ctx)
	require.NoError(t, err)
	require.Len(t, sigs.Signatures, len(want))
	for i, sig := range want {
		require.Equal(t, sig, sigs.Signatures[i].Signature)
	}
	storedHash, err := s.app.CheckpointKeeper.GetCheckpointSignaturesTxHash(s.ctx)
	require.NoError(t, err)
	require.Equal(t, common.Bytes2Hex(s.cpTxHash), storedHash)
	buffer, err := s.app.CheckpointKeeper.GetCheckpointFromBuffer(s.ctx)
	require.NoError(t, err)
	require.Equal(t, s.endBlock, buffer.EndBlock)
}

func (s *nonRpSigScenario) mutateAll(mutate func(sig []byte) []byte) [][]byte {
	sent := make([][]byte, len(s.votes))
	for i := range s.votes {
		s.votes[i].NonRpExtensionSignature = mutate(s.votes[i].NonRpExtensionSignature)
		sent[i] = s.votes[i].NonRpExtensionSignature
	}
	return sent
}

func flipRecoveryID(sig []byte) []byte {
	sig[crypto.RecoveryIDOffset] ^= 1
	return sig
}

func TestProcessProposal_NonRpSignatureOfPreviousCheckpoint(t *testing.T) {
	for _, fork := range nonRpSigForks {
		t.Run(fork.name, func(t *testing.T) {
			t.Run("genuine signatures are accepted and stored", func(t *testing.T) {
				s := newNonRpSigScenario(t, fork.karwar)
				require.Equal(t, abci.ResponseProcessProposal_ACCEPT, s.processProposal(t, s.commit(t)))
				s.requireSignaturesStored(t, s.genuine)
			})

			t.Run("invalid signatures", func(t *testing.T) {
				s := newNonRpSigScenario(t, fork.karwar)
				sent := s.mutateAll(func([]byte) []byte { return bytes.Repeat([]byte{0x42}, crypto.SignatureLength) })
				require.NoError(t, ValidateVoteExtensions(s.ctx, nonRpSigHeight, s.votes, 0, &s.vs, s.app.MilestoneKeeper))

				status := s.processProposal(t, s.commit(t))
				if fork.enforced {
					require.Equal(t, abci.ResponseProcessProposal_REJECT, status)
					require.Contains(t, s.logs.String(), "Invalid non-rp vote extension signature, rejecting proposal")
					return
				}
				require.Equal(t, abci.ResponseProcessProposal_ACCEPT, status)
				s.requireSignaturesStored(t, sent)
			})

			t.Run("flipped recovery id", func(t *testing.T) {
				s := newNonRpSigScenario(t, fork.karwar)
				sent := s.mutateAll(flipRecoveryID)

				// The recovery byte is outside the consensus signature check, so it does not affect validity.
				require.Equal(t, abci.ResponseProcessProposal_ACCEPT, s.processProposal(t, s.commit(t)))
				if fork.enforced {
					s.requireSignaturesStored(t, s.genuine)
					return
				}
				s.requireSignaturesStored(t, sent)
			})
		})
	}
}

// Differing recovery bytes across nodes' copies of a vote must not cost the proposer its quorum.
func TestPrepareProposal_DifferingRecoveryBytesKeepQuorum(t *testing.T) {
	for _, fork := range nonRpSigForks {
		t.Run(fork.name, func(t *testing.T) {
			s := newNonRpSigScenario(t, fork.karwar)
			s.mutateAll(flipRecoveryID)

			res, err := s.app.NewPrepareProposalHandler()(s.ctx, &abci.RequestPrepareProposal{
				Height:          nonRpSigHeight,
				MaxTxBytes:      1_000_000,
				LocalLastCommit: abci.ExtendedCommitInfo{Votes: s.votes},
			})
			require.NoError(t, err)
			require.NotEmpty(t, res.Txs)

			// A dropped vote would only warn in ProcessProposal, so check the quorum is intact directly.
			var kept abci.ExtendedCommitInfo
			require.NoError(t, kept.Unmarshal(res.Txs[0]))
			require.Len(t, kept.Votes, len(s.votes))
			for _, vote := range kept.Votes {
				require.NotEmpty(t, vote.NonRpVoteExtension)
				require.NotEmpty(t, vote.NonRpExtensionSignature)
			}

			require.Equal(t, abci.ResponseProcessProposal_ACCEPT, s.processProposal(t, res.Txs[0]))
		})
	}
}
