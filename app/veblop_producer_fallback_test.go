package app

import (
	"strings"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	cmtsecp256k1 "github.com/cometbft/cometbft/crypto/secp256k1"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/sidetxs"
	borKeeper "github.com/0xPolygon/heimdall-v2/x/bor/keeper"
	borTypes "github.com/0xPolygon/heimdall-v2/x/bor/types"
	milestoneTypes "github.com/0xPolygon/heimdall-v2/x/milestone/types"
	stakeTypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

// singletonScenario holds the staged degenerate VEBLOP state: the elected producer set
// has collapsed to the single current producer, and a >2/3 milestone is finalized by the
// other validators while the current producer withholds its milestone vote (so it is
// excluded from the supporting set). Future-span creation in PreBlocker then asks
// SelectNextSpanProducer for a replacement and the candidate set (singleton minus the
// current producer) is empty.
type singletonScenario struct {
	app               *HeimdallApp
	ctx               sdk.Context
	height            int64
	lastSpan          borTypes.Span
	currentProducerID uint64
	validators        []*stakeTypes.Validator
	extCommitBytes    []byte
}

func stageSingletonProducer(t *testing.T) *singletonScenario {
	t.Helper()
	_, app, ctx, validatorPrivKeys := SetupAppWithABCICtxAndValidators(t, 4)

	oldRio := helper.GetRioHeight()
	helper.SetRioHeight(100)
	t.Cleanup(func() { helper.SetRioHeight(oldRio) })

	height := app.LastBlockHeight() + 1
	ctx = ctx.WithBlockHeight(height)

	validators := app.StakeKeeper.GetAllValidators(ctx)
	require.Len(t, validators, 4)

	currentProducerID := validators[0].ValId
	lastSpan := borTypes.Span{
		Id:                1,
		StartBlock:        100,
		EndBlock:          199,
		BorChainId:        "1",
		ValidatorSet:      stakeTypes.ValidatorSet{Validators: validators, Proposer: validators[0]},
		SelectedProducers: []stakeTypes.Validator{*validators[0]},
	}
	require.NoError(t, app.BorKeeper.AddNewSpan(ctx, &lastSpan))

	// Every validator casts a one-entry ballot for the current producer, collapsing
	// CalculateProducerSet to the singleton [currentProducer].
	msgServer := borKeeper.NewMsgServerImpl(app.BorKeeper)
	for _, validator := range validators {
		_, err := msgServer.VoteProducers(ctx, &borTypes.MsgVoteProducers{
			Voter:   validator.Signer,
			VoterId: validator.ValId,
			Votes:   borTypes.ProducerVotes{Votes: []uint64{currentProducerID}},
		})
		require.NoError(t, err)
	}
	candidates, err := app.BorKeeper.CalculateProducerSet(ctx, helper.GetProducerSetLimit(ctx))
	require.NoError(t, err)
	require.Equal(t, []uint64{currentProducerID}, candidates)

	previousMilestoneHash := common.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa").Bytes()
	require.NoError(t, app.MilestoneKeeper.AddMilestone(ctx, milestoneTypes.Milestone{
		Proposer:        validators[1].Signer,
		Hash:            previousMilestoneHash,
		StartBlock:      1,
		EndBlock:        lastSpan.StartBlock - 1,
		BorChainId:      "1",
		MilestoneId:     "previous",
		Timestamp:       1,
		TotalDifficulty: 1,
	}))

	winningMilestone := &milestoneTypes.MilestoneProposition{
		StartBlockNumber: lastSpan.StartBlock,
		ParentHash:       previousMilestoneHash,
		BlockHashes:      [][]byte{common.HexToHash("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb").Bytes()},
		BlockTds:         []uint64{2},
	}

	// Validators 1, 2 and 3 (75% of voting power) finalize the milestone; the current
	// producer (validator 0) withholds, so it is absent from supportingValidatorIDs.
	extCommitBytes := milestoneExtendedCommit(t, app.ChainID(), height, validators, validatorPrivKeys, currentProducerID, winningMilestone)

	return &singletonScenario{
		app:               app,
		ctx:               ctx,
		height:            height,
		lastSpan:          lastSpan,
		currentProducerID: currentProducerID,
		validators:        validators,
		extCommitBytes:    extCommitBytes,
	}
}

// Pre-Ithaca the fallback is gated off, so the singleton-minus-current empty set still
// errors out of PreBlocker. This pins the gate boundary: the fix changes nothing before
// the fork activates.
func TestVeBlopEmptyElectedSetPreIthacaErrors(t *testing.T) {
	s := stageSingletonProducer(t)

	oldIthaca := helper.GetIthacaHeight()
	helper.SetIthacaHeight(0) // disabled: IsIthaca is always false
	t.Cleanup(func() { helper.SetIthacaHeight(oldIthaca) })

	_, err := s.app.PreBlocker(s.ctx, &abci.RequestFinalizeBlock{
		Height:          s.height,
		Txs:             [][]byte{s.extCommitBytes},
		ProposerAddress: common.FromHex(s.validators[1].Signer),
	})
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "no candidates found"), err.Error())

	// Nothing committed: the failing height replays identically.
	milestoneCount, err := s.app.MilestoneKeeper.GetMilestoneCount(s.ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(1), milestoneCount)
	span, err := s.app.BorKeeper.GetLastSpan(s.ctx)
	require.NoError(t, err)
	require.Equal(t, s.lastSpan.Id, span.Id)
}

// Post-Ithaca the fallback supplies a deterministic non-empty candidate set drawn from
// the milestone supporters, so future-span creation succeeds, the milestone/span commit,
// and the new producer is a supporting validator rather than the excluded incumbent.
func TestVeBlopEmptyElectedSetPostIthacaFallsBack(t *testing.T) {
	s := stageSingletonProducer(t)

	oldIthaca := helper.GetIthacaHeight()
	helper.SetIthacaHeight(s.height)
	t.Cleanup(func() { helper.SetIthacaHeight(oldIthaca) })

	_, err := s.app.PreBlocker(s.ctx, &abci.RequestFinalizeBlock{
		Height:          s.height,
		Txs:             [][]byte{s.extCommitBytes},
		ProposerAddress: common.FromHex(s.validators[1].Signer),
	})
	require.NoError(t, err)

	// The milestone committed and a new span was frozen.
	milestoneCount, err := s.app.MilestoneKeeper.GetMilestoneCount(s.ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(2), milestoneCount)

	span, err := s.app.BorKeeper.GetLastSpan(s.ctx)
	require.NoError(t, err)
	require.Equal(t, s.lastSpan.Id+1, span.Id)
	require.Len(t, span.SelectedProducers, 1)

	// The replacement must be a supporting validator, never the excluded incumbent.
	newProducer := span.SelectedProducers[0].ValId
	require.NotEqual(t, s.currentProducerID, newProducer)
	supporters := map[uint64]struct{}{
		s.validators[1].ValId: {},
		s.validators[2].ValId: {},
		s.validators[3].ValId: {},
	}
	_, ok := supporters[newProducer]
	require.True(t, ok, "new producer %d should be one of the milestone supporters", newProducer)
}

// A supporter drawn from the penultimate set may have exited by the time the future span is
// frozen: its record survives with zero power and it is absent from the current validator set.
// The fallback must not select it, otherwise the span's sole producer has zero power, which Bor
// reads as a validator deletion and cannot build a producer snapshot from (chain halt). The
// selected producer must instead be a positive-power member of the frozen validator set.
func TestVeBlopFallbackSkipsExitedSupporter(t *testing.T) {
	s := stageSingletonProducer(t)

	oldIthaca := helper.GetIthacaHeight()
	helper.SetIthacaHeight(s.height)
	t.Cleanup(func() { helper.SetIthacaHeight(oldIthaca) })

	// Deactivate the lowest-ID supporter — the one the deterministic (ascending) fallback would
	// otherwise pick first — reproducing an approved MsgValidatorExit at this transition.
	removed := s.validators[1]
	for _, validator := range s.validators[2:] {
		if validator.ValId < removed.ValId {
			removed = validator
		}
	}
	deactivated := removed.Copy()
	deactivated.EndEpoch = 1
	deactivated.VotingPower = 0
	require.NoError(t, s.app.StakeKeeper.AddValidator(s.ctx, *deactivated))

	// The real EndBlock update keeps the older snapshot used for vote-extension tallying and
	// removes the validator from the current set used to freeze the new span.
	updates, err := s.app.StakeKeeper.ApplyAndReturnValidatorSetUpdates(s.ctx)
	require.NoError(t, err)
	require.Len(t, updates, 1)
	require.Zero(t, updates[0].Power)

	_, err = s.app.PreBlocker(s.ctx, &abci.RequestFinalizeBlock{
		Height:          s.height,
		Txs:             [][]byte{s.extCommitBytes},
		ProposerAddress: common.FromHex(s.validators[1].Signer),
	})
	require.NoError(t, err)

	span, err := s.app.BorKeeper.GetLastSpan(s.ctx)
	require.NoError(t, err)
	require.Equal(t, s.lastSpan.Id+1, span.Id)
	require.Len(t, span.SelectedProducers, 1)

	producer := span.SelectedProducers[0]
	require.NotEqual(t, removed.ValId, producer.ValId, "exited validator must not be selected as producer")
	require.Positive(t, producer.VotingPower)

	inFrozenSet := false
	for _, validator := range span.ValidatorSet.Validators {
		if validator.ValId == producer.ValId {
			inFrozenSet = true
		}
		require.NotEqual(t, removed.ValId, validator.ValId, "exited validator must be absent from the frozen set")
	}
	require.True(t, inFrozenSet, "selected producer must be a member of the frozen validator set")
}

func milestoneExtendedCommit(t *testing.T, chainID string, height int64, validators []*stakeTypes.Validator, validatorPrivKeys []cmtsecp256k1.PrivKey, currentProducerID uint64, milestone *milestoneTypes.MilestoneProposition) []byte {
	t.Helper()
	require.Len(t, validatorPrivKeys, len(validators))

	dummyNonRpExt, err := GetDummyNonRpVoteExtension(height-1, chainID)
	require.NoError(t, err)

	extCommit := &abci.ExtendedCommitInfo{Round: 0, Votes: make([]abci.ExtendedVoteInfo, 0, len(validators))}
	for i, validator := range validators {
		var prop *milestoneTypes.MilestoneProposition
		if validator.ValId != currentProducerID {
			prop = milestone
		}

		voteExt := sidetxs.VoteExtension{
			BlockHash:            []byte("repro-previous-block"),
			Height:               height - 1,
			MilestoneProposition: prop,
			SideTxResponses:      []sidetxs.SideTxResponse{},
		}
		voteExtBytes, err := voteExt.Marshal()
		require.NoError(t, err)

		voteInfo := abci.ExtendedVoteInfo{
			Validator: abci.Validator{
				Address: common.FromHex(validator.Signer),
				Power:   validator.VotingPower,
			},
			VoteExtension:      voteExtBytes,
			NonRpVoteExtension: dummyNonRpExt,
			BlockIdFlag:        cmtproto.BlockIDFlagCommit,
		}
		createSignatureForVoteExtension(t, height-1, validatorPrivKeys[i], voteExtBytes, dummyNonRpExt, &voteInfo)
		extCommit.Votes = append(extCommit.Votes, voteInfo)
	}

	bz, err := extCommit.Marshal()
	require.NoError(t, err)
	return bz
}
