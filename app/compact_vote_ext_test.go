package app

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"testing"

	"cosmossdk.io/log"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/helper"
	helpermocks "github.com/0xPolygon/heimdall-v2/helper/mocks"
	"github.com/0xPolygon/heimdall-v2/sidetxs"
	chainManagerKeeper "github.com/0xPolygon/heimdall-v2/x/chainmanager/keeper"
	checkpointKeeper "github.com/0xPolygon/heimdall-v2/x/checkpoint/keeper"
	milestoneTypes "github.com/0xPolygon/heimdall-v2/x/milestone/types"
	stakeTypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

const compactForkHeight int64 = 100

var testBlockHash = common.HexToHash("0x0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20").Bytes()

type forkHeights struct{ compact, phuket, zurich, ithaca, kyoto int64 }

// set applies the heights for the test and restores the previous ones on cleanup.
func (f forkHeights) set(t *testing.T) {
	t.Helper()
	orig := forkHeights{helper.GetPurleyHeight(), helper.GetPhuketHardforkHeight(), helper.GetZurichHardforkHeight(), helper.GetIthacaHeight(), helper.GetKyotoHeight()}
	t.Cleanup(func() { orig.apply() })
	f.apply()
}

func (f forkHeights) apply() {
	helper.SetPurleyHeight(f.compact)
	helper.SetPhuketHardforkHeight(f.phuket)
	helper.SetZurichHardforkHeight(f.zurich)
	helper.SetIthacaHeight(f.ithaca)
	helper.SetKyotoHeight(f.kyoto)
}

// setLiveForks activates Purley at compactForkHeight on top of the forks live on mainnet and Amoy.
func setLiveForks(t *testing.T) {
	t.Helper()
	forkHeights{compact: compactForkHeight, phuket: 1, zurich: 1, ithaca: 1, kyoto: 1}.set(t)
}

// testProposition proposes blocks 10-11 with the head at the tail.
func testProposition(parent, latestHash []byte) *milestoneTypes.MilestoneProposition {
	return &milestoneTypes.MilestoneProposition{
		StartBlockNumber:  10,
		BlockHashes:       [][]byte{fill32(0x01), fill32(0x02)},
		BlockTds:          []uint64{1, 2},
		ParentHash:        parent,
		LatestBlockNumber: 11,
		LatestBlockHash:   latestHash,
	}
}

func legacyVE(height int64) *sidetxs.VoteExtension {
	return &sidetxs.VoteExtension{Height: height, BlockHash: testBlockHash, MilestoneProposition: testProposition(fill32(0x09), fill32(0x02))}
}

func compactVE() *sidetxs.VoteExtension {
	return &sidetxs.VoteExtension{BlockHash: testBlockHash[:compactVEBlockHashLength], MilestoneProposition: testProposition(fill32(0x09)[:8], nil)}
}

// signedTestVote signs ve and the non-rp placeholder for veHeight with val's key.
func signedTestVote(t *testing.T, ctx sdk.Context, val *stakeTypes.Validator, privKeys []secp256k1.PrivKey, veHeight int64, ve *sidetxs.VoteExtension) abci.ExtendedVoteInfo {
	t.Helper()
	nonRp, err := GetDummyNonRpVoteExtension(veHeight, ctx.ChainID())
	require.NoError(t, err)
	return signedTestVoteWithNonRp(t, ctx, val, privKeys, veHeight, ve, nonRp)
}

func signedTestVoteWithNonRp(t *testing.T, ctx sdk.Context, val *stakeTypes.Validator, privKeys []secp256k1.PrivKey, veHeight int64, ve *sidetxs.VoteExtension, nonRp []byte) abci.ExtendedVoteInfo {
	t.Helper()
	for _, pk := range privKeys {
		if bytes.Equal(pk.PubKey().Address(), common.FromHex(val.Signer)) {
			return createSignedVoteInfo(t, ctx, val, pk, ve, nonRp, veHeight+1, 0)
		}
	}
	t.Fatalf("no key for validator %s", val.Signer)
	return abci.ExtendedVoteInfo{}
}

func filterAt(ctx sdk.Context, app *HeimdallApp, valSet *stakeTypes.ValidatorSet, reqHeight int64, votes []abci.ExtendedVoteInfo) ([]abci.ExtendedVoteInfo, error) {
	return filterVoteExtensions(ctx.WithBlockHeight(reqHeight), reqHeight, votes, 0, valSet, app.MilestoneKeeper, 1_000_000, log.NewNopLogger())
}

// borChain mocks a bor chain whose headers link by parent hash, with the last header as the head.
func borChain(headers []*ethTypes.Header) *helpermocks.IContractCaller {
	return borChainWithHead(headers, headers[len(headers)-1])
}

// borChainWithHead serves head as the latest header, independently of the batched headers.
func borChainWithHead(headers []*ethTypes.Header, head *ethTypes.Header) *helpermocks.IContractCaller {
	first := headers[0].Number.Int64()
	caller := new(helpermocks.IContractCaller)
	caller.On("GetBorChainBlock", mock.Anything, (*big.Int)(nil)).Return(head, nil)
	caller.On("GetBorChainBlock", mock.Anything, mock.Anything).Return(
		func(_ context.Context, n *big.Int) (*ethTypes.Header, error) { return headers[n.Int64()-first], nil })
	caller.On("GetBorChainBlockInfoInBatch", mock.Anything, mock.Anything, mock.Anything).Return(
		func(_ context.Context, start, end int64) ([]*ethTypes.Header, []uint64, []common.Address, error) {
			hdrs := headers[start-first : end-first+1]
			tds, authors := make([]uint64, len(hdrs)), make([]common.Address, len(hdrs))
			for i, h := range hdrs {
				tds[i] = h.Number.Uint64()
			}
			return hdrs, tds, authors, nil
		})
	return caller
}

func linkedHeaders(first, last int64, parent common.Hash) []*ethTypes.Header {
	headers := make([]*ethTypes.Header, 0, last-first+1)
	for n := first; n <= last; n++ {
		h := &ethTypes.Header{Number: big.NewInt(n), ParentHash: parent}
		headers = append(headers, h)
		parent = h.Hash()
	}
	return headers
}

func TestValidateVoteExtensionHeaderRejectsWrongLegacyHeight(t *testing.T) {
	setLiveForks(t)
	require.ErrorContains(t, validateVoteExtensionHeader(legacyVE(compactForkHeight-2), compactForkHeight-1), "invalid vote extension height")
}

func TestDummyNonRpVoteExtensionCompact(t *testing.T) {
	setLiveForks(t)
	const chainID = "heimdallv2-137"
	compact := append([]byte{0x00, 0, 0, 0, 0, 0, 0, 0, byte(compactForkHeight)}, chainID...)

	legacy, err := GetDummyNonRpVoteExtension(compactForkHeight-1, chainID)
	require.NoError(t, err)
	got, err := GetDummyNonRpVoteExtension(compactForkHeight, chainID)
	require.NoError(t, err)
	require.Equal(t, compact, got)
	require.Equal(t, minNonRpVoteExtensionSize, minNonRpVoteExtensionSizeAt(compactForkHeight-1))
	require.Equal(t, compactMinNonRpVoteExtensionSize, minNonRpVoteExtensionSizeAt(compactForkHeight))

	// From Purley only ExtendVote's warning-only self-check reaches the non-rp data checks; the envelope
	// bounds carried placeholders (TestEnvelopeAcceptsCompactPlaceholder, TestVoteEnvelopeNrSizeBoundaries).
	ctx := setupContextWithVoteExtensionsEnableHeight(SetupApp(t, 1).App.BaseApp.NewContext(true), 1).WithChainID(chainID)
	tests := []struct {
		name    string
		height  int64
		ext     []byte
		wantErr string
	}{
		{"legacy before fork", compactForkHeight - 1, legacy, ""},
		{"compact before fork", compactForkHeight - 1, CompactDummyNonRpVoteExtension(compactForkHeight-1, chainID), "too small"},
		{"compact at fork", compactForkHeight, got, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateNonRpVoteExtensionData(ctx, tc.height, tc.ext, chainManagerKeeper.Keeper{}, checkpointKeeper.Keeper{}, nil)
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}

func TestNonRpDataFloorAtFork(t *testing.T) {
	setLiveForks(t)
	ctx := setupContextWithVoteExtensionsEnableHeight(SetupApp(t, 1).App.BaseApp.NewContext(true), 1).WithChainID("heimdallv2-137")
	validate := func(size int) error {
		return validateNonRpVoteExtensionData(ctx, compactForkHeight, make([]byte, size), chainManagerKeeper.Keeper{}, checkpointKeeper.Keeper{}, nil)
	}
	floorErr := fmt.Sprintf("min: %d", compactMinNonRpVoteExtensionSize)
	require.ErrorContains(t, validate(compactMinNonRpVoteExtensionSize-1), floorErr)
	err := validate(compactMinNonRpVoteExtensionSize)
	require.Error(t, err)
	require.NotContains(t, err.Error(), floorErr)
}

// Proposition cases carry an implied head, so a handler that gates fork rules on the omitted ve.Height
// (0) instead of the authenticated height rejects compact extensions. Idle cases break only the header.
var compactBoundaryCases = []struct {
	name   string
	height int64
	ve     *sidetxs.VoteExtension
	wantOK bool
}{
	{"legacy before fork", compactForkHeight - 1, legacyVE(compactForkHeight - 1), true},
	{"compact before fork", compactForkHeight - 1, compactVE(), false},
	{"compact at fork", compactForkHeight, compactVE(), true},
	{"legacy at fork", compactForkHeight, legacyVE(compactForkHeight), false},
	{"idle with height at fork", compactForkHeight, &sidetxs.VoteExtension{Height: compactForkHeight, BlockHash: testBlockHash[:compactVEBlockHashLength]}, false},
	{"idle with full block hash at fork", compactForkHeight, &sidetxs.VoteExtension{BlockHash: testBlockHash}, false},
}

func TestVerifyVoteExtensionCompactBoundary(t *testing.T) {
	hApp := SetupApp(t, 1).App
	ctx := setupContextWithVoteExtensionsEnableHeight(hApp.BaseApp.NewContext(true), 1)
	validators := hApp.StakeKeeper.GetAllValidators(ctx)
	setLiveForks(t)

	verify := func(height int64, ve *sidetxs.VoteExtension) abci.ResponseVerifyVoteExtension_VerifyStatus {
		nonRp, err := GetDummyNonRpVoteExtension(height, hApp.ChainID())
		require.NoError(t, err)
		bound := *ve
		bound.NonRpExtensionHash = nonRpExtensionHash(height, nonRp)
		bz, err := bound.Marshal()
		require.NoError(t, err)
		res, err := hApp.VerifyVoteExtensionHandler()(ctx, &abci.RequestVerifyVoteExtension{
			Height:             height,
			Hash:               testBlockHash,
			ValidatorAddress:   common.FromHex(validators[0].Signer),
			VoteExtension:      bz,
			NonRpVoteExtension: nonRp,
		})
		require.NoError(t, err)
		return res.Status
	}

	for _, tc := range compactBoundaryCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.wantOK, verify(tc.height, tc.ve) == abci.ResponseVerifyVoteExtension_ACCEPT)
		})
	}
	wrongPrefix := compactVE()
	wrongPrefix.BlockHash = testBlockHash[1 : 1+compactVEBlockHashLength]
	require.Equal(t, abci.ResponseVerifyVoteExtension_REJECT, verify(compactForkHeight, wrongPrefix))
}

// Proposal handling at H validates and tallies extensions produced at H-1, through the placeholder
// rules before Purley and the signed transport from it.
func TestCompactVoteExtensionProposalBoundary(t *testing.T) {
	_, app, ctx, privKeys := SetupAppWithABCICtx(t)
	valSet, err := app.StakeKeeper.GetValidatorSet(ctx)
	require.NoError(t, err)
	setLiveForks(t)

	for _, tc := range compactBoundaryCases {
		t.Run(tc.name, func(t *testing.T) {
			reqHeight := tc.height + 1
			commit := abci.ExtendedCommitInfo{Votes: []abci.ExtendedVoteInfo{signedTestVote(t, ctx, valSet.Validators[0], privKeys, tc.height, boundTestVE(t, ctx, tc.height, tc.ve))}}
			if !helper.IsPurley(tc.height) {
				// A filtered placeholder from an earlier PrepareProposal stays inert before the fork.
				commit.Votes = append(commit.Votes, abci.ExtendedVoteInfo{BlockIdFlag: cmtproto.BlockIDFlagCommit, Validator: abci.Validator{Address: common.HexToAddress("0x01").Bytes()}})
			}
			commitBytes, err := commit.Marshal()
			require.NoError(t, err)
			pctx := ctx.WithBlockHeight(reqHeight)

			err = app.validateProposalCommit(pctx, &abci.RequestProcessProposal{Height: reqHeight, Txs: [][]byte{commitBytes}}, &commit, &valSet)
			_, aggErr := aggregateVotes(commit.Votes, &valSet, reqHeight, log.NewNopLogger())
			if !tc.wantOK {
				require.Error(t, err)
				require.Error(t, aggErr)
				return
			}
			require.NoError(t, err)
			require.NoError(t, aggErr)
			prepare := &abci.RequestPrepareProposal{Height: reqHeight, MaxTxBytes: 1_000_000, LocalLastCommit: commit}
			require.NoError(t, app.prepareCommit(pctx, prepare, &valSet))
			require.Equal(t, commit.Votes, prepare.LocalLastCommit.Votes)
		})
	}
}

// boundTestVE binds ve to the placeholder signed with it at veHeight, as ExtendVote does.
func boundTestVE(t *testing.T, ctx sdk.Context, veHeight int64, ve *sidetxs.VoteExtension) *sidetxs.VoteExtension {
	t.Helper()
	nonRp, err := GetDummyNonRpVoteExtension(veHeight, ctx.ChainID())
	require.NoError(t, err)
	bound := *ve
	bound.NonRpExtensionHash = nonRpExtensionHash(veHeight, nonRp)
	return &bound
}

// filterVoteExtensions only sees votes signed before Purley, so it keeps the legacy non-rp floor and a
// compact placeholder signed before the fork is undersized and leaves no majority.
func TestFilterKeepsLegacyNonRpFloor(t *testing.T) {
	_, app, ctx, privKeys := SetupAppWithABCICtx(t)
	valSet, err := app.StakeKeeper.GetValidatorSet(ctx)
	require.NoError(t, err)
	setLiveForks(t)

	const veHeight = compactForkHeight - 1
	vote := signedTestVoteWithNonRp(t, ctx, valSet.Validators[0], privKeys, veHeight, legacyVE(veHeight), CompactDummyNonRpVoteExtension(veHeight, ctx.ChainID()))
	_, err = filterAt(ctx, app, &valSet, veHeight+1, []abci.ExtendedVoteInfo{vote})
	require.ErrorContains(t, err, "insufficient cumulative voting power")
}

// An idle compact extension (no side txs, no proposition) must stay above the minVESize floor the
// envelope enforces, or every idle validator would be rejected from Purley.
func TestIdleCompactVoteExtensionPassesEnvelope(t *testing.T) {
	setLiveForks(t)
	const chainID = "heimdallv2-137"
	nonRp := CompactDummyNonRpVoteExtension(compactForkHeight, chainID)
	idle := &sidetxs.VoteExtension{BlockHash: testBlockHash[:compactVEBlockHashLength], NonRpExtensionHash: nonRpExtensionHash(compactForkHeight, nonRp)}
	bz, err := idle.Marshal()
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(bz), minVESize)
	_, err = validateVoteEnvelope(&abci.RequestVerifyVoteExtension{Height: compactForkHeight, Hash: testBlockHash, VoteExtension: bz, NonRpVoteExtension: nonRp})
	require.NoError(t, err)
}

// From Purley PrepareProposal proposes the signed commit as is and ProcessProposal authenticates every
// vote, so a single legacy-format vote at the fork rejects the proposal instead of becoming a placeholder.
func TestCompactVoteExtensionPrepareProcessSymmetry(t *testing.T) {
	_, app, ctx, privKeys := SetupAppWithABCICtxAndValidators(t, 4)
	valSet, err := app.StakeKeeper.GetValidatorSet(ctx)
	require.NoError(t, err)
	setLiveForks(t)

	const reqHeight = compactForkHeight + 1
	pctx := ctx.WithBlockHeight(reqHeight)
	for _, legacyFirst := range []bool{false, true} {
		commit := abci.ExtendedCommitInfo{}
		for i, val := range valSet.Validators {
			ve := compactVE()
			if legacyFirst && i == 0 {
				ve = legacyVE(compactForkHeight)
			}
			commit.Votes = append(commit.Votes, signedTestVote(t, ctx, val, privKeys, compactForkHeight, boundTestVE(t, ctx, compactForkHeight, ve)))
		}
		prepare := &abci.RequestPrepareProposal{Height: reqHeight, MaxTxBytes: 1_000_000, LocalLastCommit: commit}
		require.NoError(t, app.prepareCommit(pctx, prepare, &valSet))
		require.Equal(t, commit.Votes, prepare.LocalLastCommit.Votes)

		err := app.validateProposalCommit(pctx, &abci.RequestProcessProposal{Height: reqHeight}, &prepare.LocalLastCommit, &valSet)
		if legacyFirst {
			require.Error(t, err)
			continue
		}
		require.NoError(t, err)
		_, err = aggregateVotes(prepare.LocalLastCommit.Votes, &valSet, reqHeight, log.NewNopLogger())
		require.NoError(t, err)
	}
}

// ExtendVote emits the legacy form before the fork and the compact form from it, each accepted by
// VerifyVoteExtension at its own height.
func TestExtendVoteCompactRoundTrip(t *testing.T) {
	_, app, ctx, _ := SetupAppWithABCICtx(t)
	validators := app.StakeKeeper.GetAllValidators(ctx)
	setLiveForks(t)

	lastMilestone := milestoneTypes.Milestone{EndBlock: 100, Hash: fill32(0x0A), BorChainId: "1"}
	require.NoError(t, app.MilestoneKeeper.AddMilestone(ctx, lastMilestone))
	caller := borChain(linkedHeaders(101, 103, common.BytesToHash(lastMilestone.Hash)))
	app.caller = caller
	app.MilestoneKeeper.IContractCaller = caller
	extCommit, err := (&abci.ExtendedCommitInfo{}).Marshal()
	require.NoError(t, err)

	extendAndVerify := func(height int64) (*sidetxs.VoteExtension, int, []byte) {
		resp, err := app.ExtendVoteHandler()(ctx, &abci.RequestExtendVote{Txs: [][]byte{extCommit}, Hash: testBlockHash, Height: height})
		require.NoError(t, err)
		res, err := app.VerifyVoteExtensionHandler()(ctx, &abci.RequestVerifyVoteExtension{
			Height:             height,
			Hash:               testBlockHash,
			ValidatorAddress:   common.FromHex(validators[0].Signer),
			VoteExtension:      resp.VoteExtension,
			NonRpVoteExtension: resp.NonRpExtension,
		})
		require.NoError(t, err)
		require.Equal(t, abci.ResponseVerifyVoteExtension_ACCEPT, res.Status)
		var ve sidetxs.VoteExtension
		require.NoError(t, ve.Unmarshal(resp.VoteExtension))
		require.NotNil(t, ve.MilestoneProposition)
		return &ve, len(resp.VoteExtension), resp.NonRpExtension
	}

	legacy, legacySize, legacyNonRp := extendAndVerify(compactForkHeight - 1)
	require.Empty(t, legacy.NonRpExtensionHash)
	require.Equal(t, legacyNonRp, mustLegacyDummy(t, compactForkHeight-1, ctx.ChainID()))
	require.Equal(t, compactForkHeight-1, legacy.Height)
	require.Len(t, legacy.BlockHash, common.HashLength)
	require.Equal(t, lastMilestone.Hash, legacy.MilestoneProposition.ParentHash)
	require.NotEmpty(t, legacy.MilestoneProposition.LatestBlockHash)

	compact, compactSize, compactNonRp := extendAndVerify(compactForkHeight)
	require.Equal(t, CompactDummyNonRpVoteExtension(compactForkHeight, ctx.ChainID()), compactNonRp)
	require.Equal(t, nonRpExtensionHash(compactForkHeight, compactNonRp), compact.NonRpExtensionHash)
	require.Zero(t, compact.Height)
	require.Len(t, compact.BlockHash, compactVEBlockHashLength)
	require.Equal(t, lastMilestone.Hash[:8], compact.MilestoneProposition.ParentHash)
	require.Equal(t, legacy.MilestoneProposition.LatestBlockNumber, compact.MilestoneProposition.LatestBlockNumber)
	require.Empty(t, compact.MilestoneProposition.LatestBlockHash)
	// Block hash and parent each drop 24 bytes and the head hash its 32; Purley adds the 34-byte non-rp hash field.
	require.GreaterOrEqual(t, legacySize-(compactSize-(2+common.HashLength)), 24+24+32)
}

func mustLegacyDummy(t *testing.T, height int64, chainID string) []byte {
	t.Helper()
	ext, err := LegacyDummyNonRpVoteExtension(height, chainID)
	require.NoError(t, err)
	return ext
}

// Runs the real handlers across the fork. ExtendVote at H precedes FinalizeBlock at H, so: the fork
// block finalizes the legacy proposition from L+1, fork+1 is the first height to propose on top of that
// milestone (compact, 8-byte parent), and fork+2 finalizes it.
func TestCompactVoteExtensionAcrossFork(t *testing.T) {
	tests := []struct {
		name  string
		forks forkHeights
	}{
		{"live networks and devnets", forkHeights{phuket: 1, zurich: 1, ithaca: 1, kyoto: 1}},
		{"before phuket, zurich and ithaca", forkHeights{kyoto: 1}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, app, ctx, privKeys := SetupAppWithABCICtx(t)
			validators := app.StakeKeeper.GetAllValidators(ctx)
			fork := app.LastBlockHeight() + 2
			tc.forks.compact = fork
			tc.forks.set(t)
			// Keep producer voting off so propositions skip the per-block author check.
			origRio := helper.GetRioHeight()
			t.Cleanup(func() { helper.SetRioHeight(origRio) })
			helper.SetRioHeight(1 << 40)
			setMockCallerOnAllKeepers(app, borChain(linkedHeaders(0, 19, common.Hash{})))
			seedSpan(t, app, app.NewUncachedContext(false, cmtproto.Header{}))

			_, extCommit, _, err := buildExtensionCommits(t, app, testBlockHash, validators, privKeys, app.LastBlockHeight(), nil)
			require.NoError(t, err)
			var milestoneEnds []uint64
			for range 4 {
				resp := executeHeight(t, ctx, app, *extCommit, nil)
				vote := abci.ExtendedVoteInfo{
					BlockIdFlag:        cmtproto.BlockIDFlagCommit,
					Validator:          abci.Validator{Address: common.FromHex(validators[0].Signer), Power: validators[0].VotingPower},
					VoteExtension:      resp.VoteExtension,
					NonRpVoteExtension: resp.NonRpExtension,
				}
				createSignatureForVoteExtension(t, app.LastBlockHeight(), privKeys[0], vote.VoteExtension, vote.NonRpVoteExtension, &vote)
				_, extCommit, _, err = buildExtensionCommits(t, app, testBlockHash, validators, privKeys, app.LastBlockHeight(), &vote)
				require.NoError(t, err)

				if m, err := app.MilestoneKeeper.GetLastMilestone(app.BaseApp.NewContext(true)); err == nil {
					milestoneEnds = append(milestoneEnds, m.EndBlock)
				}
			}
			require.Equal(t, []uint64{9, 9, 19}, milestoneEnds)
		})
	}
}

// The latest header and the proposition batch come from separate RPC calls and can disagree at the
// same height mid-reorg. ExtendVote must drop such a proposition in both formats rather than let
// compaction hide the mismatch.
func TestExtendVoteDropsInconsistentHeadSnapshot(t *testing.T) {
	_, app, ctx, _ := SetupAppWithABCICtx(t)
	setLiveForks(t)

	lastMilestone := milestoneTypes.Milestone{EndBlock: 100, Hash: fill32(0x0A), BorChainId: "1"}
	require.NoError(t, app.MilestoneKeeper.AddMilestone(ctx, lastMilestone))
	headers := linkedHeaders(101, 103, common.BytesToHash(lastMilestone.Hash))
	reorgedHead := &ethTypes.Header{Number: big.NewInt(103), ParentHash: common.HexToHash("0xbeef")}
	caller := borChainWithHead(headers, reorgedHead)
	app.caller = caller
	app.MilestoneKeeper.IContractCaller = caller
	extCommit, err := (&abci.ExtendedCommitInfo{}).Marshal()
	require.NoError(t, err)

	for _, height := range []int64{compactForkHeight - 1, compactForkHeight} {
		resp, err := app.ExtendVoteHandler()(ctx, &abci.RequestExtendVote{Txs: [][]byte{extCommit}, Hash: testBlockHash, Height: height})
		require.NoError(t, err)
		var ve sidetxs.VoteExtension
		require.NoError(t, ve.Unmarshal(resp.VoteExtension))
		require.Nil(t, ve.MilestoneProposition, "height %d", height)
	}
}
