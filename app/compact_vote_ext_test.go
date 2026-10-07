package app

import (
	"bytes"
	"context"
	"math/big"
	"testing"

	"cosmossdk.io/log"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	promtestutil "github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/helper"
	helpermocks "github.com/0xPolygon/heimdall-v2/helper/mocks"
	"github.com/0xPolygon/heimdall-v2/metrics"
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
	orig := forkHeights{helper.GetCompactVoteExtHeight(), helper.GetPhuketHardforkHeight(), helper.GetZurichHardforkHeight(), helper.GetIthacaHeight(), helper.GetKyotoHeight()}
	t.Cleanup(func() { orig.apply() })
	f.apply()
}

func (f forkHeights) apply() {
	helper.SetCompactVoteExtHeight(f.compact)
	helper.SetPhuketHardforkHeight(f.phuket)
	helper.SetZurichHardforkHeight(f.zurich)
	helper.SetIthacaHeight(f.ithaca)
	helper.SetKyotoHeight(f.kyoto)
}

// setLiveForks activates the compact fork at compactForkHeight on top of the forks live on mainnet and Amoy.
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
	first := headers[0].Number.Int64()
	caller := new(helpermocks.IContractCaller)
	caller.On("GetBorChainBlock", mock.Anything, (*big.Int)(nil)).Return(headers[len(headers)-1], nil)
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

	ctx := setupContextWithVoteExtensionsEnableHeight(SetupApp(t, 1).App.BaseApp.NewContext(true), 1).WithChainID(chainID)
	tests := []struct {
		name    string
		height  int64
		ext     []byte
		wantErr string
	}{
		{"legacy before fork", compactForkHeight - 1, legacy, ""},
		{"compact at fork", compactForkHeight, compact, ""},
		{"compact before fork", compactForkHeight - 1, CompactDummyNonRpVoteExtension(compactForkHeight-1, chainID), "too small"},
		{"legacy at fork", compactForkHeight, legacy, "failed to validate checkpoint msg data"},
		{"stale height", compactForkHeight + 1, compact, "failed to validate checkpoint msg data"},
		{"below compact floor", compactForkHeight, compact[:compactMinNonRpVoteExtensionSize-1], "min: 9"},
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
		bz, err := ve.Marshal()
		require.NoError(t, err)
		nonRp, err := GetDummyNonRpVoteExtension(height, hApp.ChainID())
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

	// Compact header rejections keep the existing rejection metric labels.
	for reason, ve := range map[string]*sidetxs.VoteExtension{
		"height_mismatch": legacyVE(compactForkHeight),
		"hash_mismatch":   {BlockHash: testBlockHash},
	} {
		before := promtestutil.ToFloat64(metrics.VoteExtensionRejectedTotal.WithLabelValues(reason))
		require.Equal(t, abci.ResponseVerifyVoteExtension_REJECT, verify(compactForkHeight, ve))
		require.Equal(t, before+1, promtestutil.ToFloat64(metrics.VoteExtensionRejectedTotal.WithLabelValues(reason)), reason)
	}
}

// Proposal handling at H validates and tallies extensions produced at H-1.
func TestCompactVoteExtensionProposalBoundary(t *testing.T) {
	_, app, ctx, privKeys := SetupAppWithABCICtx(t)
	valSet, err := app.StakeKeeper.GetValidatorSet(ctx)
	require.NoError(t, err)
	setLiveForks(t)

	for _, tc := range compactBoundaryCases {
		t.Run(tc.name, func(t *testing.T) {
			reqHeight := tc.height + 1
			// A filtered placeholder from an earlier PrepareProposal must stay inert on both sides of the fork.
			placeholder := abci.ExtendedVoteInfo{BlockIdFlag: cmtproto.BlockIDFlagCommit, Validator: abci.Validator{Address: common.HexToAddress("0x01").Bytes()}}
			votes := []abci.ExtendedVoteInfo{signedTestVote(t, ctx, valSet.Validators[0], privKeys, tc.height, tc.ve), placeholder}

			err := ValidateVoteExtensions(ctx.WithBlockHeight(reqHeight), reqHeight, votes, 0, &valSet, app.MilestoneKeeper)
			_, aggErr := aggregateVotes(votes, &valSet, reqHeight, log.NewNopLogger())
			filtered, filterErr := filterAt(ctx, app, &valSet, reqHeight, votes)
			if !tc.wantOK {
				require.Error(t, err)
				require.Error(t, aggErr)
				require.Error(t, filterErr, "the only vote is filtered, leaving no majority")
				return
			}
			require.NoError(t, err)
			require.NoError(t, aggErr)
			require.NoError(t, filterErr)
			require.Equal(t, votes, filtered)
		})
	}
}

// filterVoteExtensions sizes the non-rp floor by the extension's own height, so a compact placeholder
// signed before the fork is undersized and leaves no majority.
func TestFilterSizesNonRpFloorAtExtensionHeight(t *testing.T) {
	_, app, ctx, privKeys := SetupAppWithABCICtx(t)
	valSet, err := app.StakeKeeper.GetValidatorSet(ctx)
	require.NoError(t, err)
	setLiveForks(t)

	const veHeight = compactForkHeight - 1
	vote := signedTestVoteWithNonRp(t, ctx, valSet.Validators[0], privKeys, veHeight, legacyVE(veHeight), CompactDummyNonRpVoteExtension(veHeight, ctx.ChainID()))
	_, err = filterAt(ctx, app, &valSet, veHeight+1, []abci.ExtendedVoteInfo{vote})
	require.ErrorContains(t, err, "insufficient cumulative voting power")
}

// An idle compact extension (no side txs, no proposition) sits on the minVESize floor that
// filterVoteExtensions enforces with `veSize < minVESize`. Raising the floor or flipping the comparison
// would filter every idle validator and stall proposals.
func TestIdleCompactVoteExtensionSurvivesFiltering(t *testing.T) {
	idle := &sidetxs.VoteExtension{BlockHash: testBlockHash[:compactVEBlockHashLength]}
	bz, err := idle.Marshal()
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(bz), minVESize)

	_, app, ctx, privKeys := SetupAppWithABCICtx(t)
	valSet, err := app.StakeKeeper.GetValidatorSet(ctx)
	require.NoError(t, err)
	setLiveForks(t)

	votes := []abci.ExtendedVoteInfo{signedTestVote(t, ctx, valSet.Validators[0], privKeys, compactForkHeight, idle)}
	filtered, err := filterAt(ctx, app, &valSet, compactForkHeight+1, votes)
	require.NoError(t, err)
	require.False(t, isFilteredPlaceholder(filtered[0]), "idle compact extension was filtered")
}

// What a proposer emits from PrepareProposal at the fork must pass ProcessProposal and PreBlocker on
// every validator, with a malformed extension kept as a placeholder for the completeness check.
func TestCompactVoteExtensionPrepareProcessSymmetry(t *testing.T) {
	_, app, ctx, privKeys := SetupAppWithABCICtxAndValidators(t, 4)
	valSet, err := app.StakeKeeper.GetValidatorSet(ctx)
	require.NoError(t, err)
	setLiveForks(t)

	const reqHeight = compactForkHeight + 1
	votes := make([]abci.ExtendedVoteInfo, 0, len(valSet.Validators))
	for i, val := range valSet.Validators {
		ve := compactVE()
		if i == 0 {
			ve = legacyVE(compactForkHeight)
		}
		votes = append(votes, signedTestVote(t, ctx, val, privKeys, compactForkHeight, ve))
	}

	filtered, err := filterAt(ctx, app, &valSet, reqHeight, votes)
	require.NoError(t, err)
	require.Len(t, filtered, len(votes))
	require.True(t, isFilteredPlaceholder(filtered[0]))

	require.NoError(t, ValidateVoteExtensions(ctx.WithBlockHeight(reqHeight), reqHeight, filtered, 0, &valSet, app.MilestoneKeeper))
	_, err = aggregateVotes(filtered, &valSet, reqHeight, log.NewNopLogger())
	require.NoError(t, err)
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

	extendAndVerify := func(height int64) (*sidetxs.VoteExtension, int) {
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
		return &ve, len(resp.VoteExtension)
	}

	legacy, legacySize := extendAndVerify(compactForkHeight - 1)
	require.Equal(t, compactForkHeight-1, legacy.Height)
	require.Len(t, legacy.BlockHash, common.HashLength)
	require.Equal(t, lastMilestone.Hash, legacy.MilestoneProposition.ParentHash)
	require.NotEmpty(t, legacy.MilestoneProposition.LatestBlockHash)

	compact, compactSize := extendAndVerify(compactForkHeight)
	require.Zero(t, compact.Height)
	require.Len(t, compact.BlockHash, compactVEBlockHashLength)
	require.Equal(t, lastMilestone.Hash[:8], compact.MilestoneProposition.ParentHash)
	require.Equal(t, legacy.MilestoneProposition.LatestBlockNumber, compact.MilestoneProposition.LatestBlockNumber)
	require.Empty(t, compact.MilestoneProposition.LatestBlockHash)
	// Block hash and parent each drop 24 bytes and the head hash its 32.
	require.GreaterOrEqual(t, legacySize-compactSize, 24+24+32)
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
