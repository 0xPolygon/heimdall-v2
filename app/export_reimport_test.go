package app_test

import (
	"encoding/json"
	"testing"

	"cosmossdk.io/log"
	abci "github.com/cometbft/cometbft/abci/types"
	cryptoenc "github.com/cometbft/cometbft/crypto/encoding"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/app"
	util "github.com/0xPolygon/heimdall-v2/common/hex"
	"github.com/0xPolygon/heimdall-v2/sidetxs"
	checkpointTypes "github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
	stakeTypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

// An export taken right after a block that changed the validator set must load back
// through InitChain and keep producing blocks: baseapp rejects InitChain unless the
// exported comet validators match the ones InitChainer derives from the stake records, and
// the second block verifies vote extensions signed by those genesis validators.
func TestExportAppStateAndValidators_ReimportsThroughInitChain(t *testing.T) {
	setup := app.SetupApp(t, 3)
	hApp := setup.App
	ctx := hApp.NewUncachedContext(false, cmtproto.Header{Height: hApp.LastBlockHeight()})
	signedBy := hApp.StakeKeeper.GetCurrentValidators(ctx)

	repowered, err := hApp.StakeKeeper.GetValidatorFromValID(ctx, 1)
	require.NoError(t, err)
	repowered.VotingPower = 150
	require.NoError(t, hApp.StakeKeeper.AddValidator(ctx, repowered))

	pendingPub := secp256k1.GenPrivKey().PubKey()
	pending, err := stakeTypes.NewValidator(10, 1000, 0, 1, 100, pendingPub, pendingPub.Address().String())
	require.NoError(t, err)
	require.NoError(t, hApp.StakeKeeper.AddValidator(ctx, *pending))

	rotated, err := hApp.StakeKeeper.GetValidatorFromValID(ctx, 0)
	require.NoError(t, err)
	newPub := secp256k1.GenPrivKey().PubKey()
	require.NoError(t, hApp.StakeKeeper.UpdateSigner(ctx, newPub.Address().String(), newPub.Bytes(), rotated.Signer))
	unchanged, err := hApp.StakeKeeper.GetValidatorFromValID(ctx, 2)
	require.NoError(t, err)

	// the stake EndBlocker of this block applies the changes to the validator set
	finalizeAndCommit(t, hApp, hApp.LastBlockHeight()+1, signedBy, signedBy)

	exported, err := hApp.ExportAppStateAndValidators(false, nil, nil)
	require.NoError(t, err)

	// the pre-rotation record and the pending validator stay out
	names := make([]string, 0, len(exported.Validators))
	vals := make([]abci.ValidatorUpdate, 0, len(exported.Validators))
	for _, v := range exported.Validators {
		names = append(names, util.FormatAddress(v.Name))
		pk, err := cryptoenc.PubKeyToProto(v.PubKey)
		require.NoError(t, err)
		vals = append(vals, abci.ValidatorUpdate{PubKey: pk, Power: v.Power})
	}
	require.ElementsMatch(t, []string{
		util.FormatAddress(newPub.Address().String()),
		util.FormatAddress(repowered.Signer),
		util.FormatAddress(unchanged.Signer),
	}, names)

	var appState map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(exported.AppState, &appState))
	genesisSet := stakeTypes.GetGenesisStateFromAppState(hApp.AppCodec(), appState).CurrentValidatorSet
	genesisSigners := make([]stakeTypes.Validator, 0, len(genesisSet.Validators))
	for _, v := range genesisSet.Validators {
		genesisSigners = append(genesisSigners, *v)
	}

	appOptions := make(simtestutil.AppOptionsMap)
	appOptions[flags.FlagHome] = app.DefaultNodeHome
	fresh := app.NewHeimdallApp(log.NewTestLogger(t), dbm.NewMemDB(), nil, true, appOptions)

	require.Equal(t, exported.Height, exported.ConsensusParams.Abci.VoteExtensionsEnableHeight)
	res, err := fresh.InitChain(&abci.RequestInitChain{
		ChainId:         hApp.ChainID(),
		Validators:      vals,
		ConsensusParams: &exported.ConsensusParams,
		AppStateBytes:   exported.AppState,
		InitialHeight:   exported.Height,
	})
	require.NoError(t, err)
	require.Len(t, res.Validators, len(vals))

	// the initial block has no last commit; comet signs it with the genesis validators,
	// whose vote extensions the second block verifies
	finalizeAndCommit(t, fresh, exported.Height, genesisSigners, nil)
	finalizeAndCommit(t, fresh, exported.Height+1, genesisSigners, genesisSigners)
}

func finalizeAndCommit(t *testing.T, hApp *app.HeimdallApp, height int64, proposers, signedBy []stakeTypes.Validator) {
	t.Helper()

	nonRpExt, err := app.GetDummyNonRpVoteExtension(height, hApp.ChainID())
	require.NoError(t, err)
	ext, err := (&sidetxs.VoteExtension{SideTxResponses: []sidetxs.SideTxResponse{}, Height: height - 1}).Marshal()
	require.NoError(t, err)

	commit := abci.ExtendedCommitInfo{}
	for _, v := range signedBy {
		commit.Votes = append(commit.Votes, abci.ExtendedVoteInfo{
			VoteExtension:      ext,
			NonRpVoteExtension: nonRpExt,
			BlockIdFlag:        cmtproto.BlockIDFlagCommit,
			Validator:          abci.Validator{Address: common.FromHex(v.Signer), Power: v.VotingPower},
		})
	}
	commitInfo, err := commit.Marshal()
	require.NoError(t, err)

	proposer := common.FromHex(proposers[0].Signer)
	// the test's vote extensions are unsigned, so only an empty commit can go through
	// ProcessProposal, which checks the vote extension quorum
	if len(signedBy) == 0 {
		processed, err := hApp.ProcessProposal(&abci.RequestProcessProposal{
			Txs:             [][]byte{commitInfo},
			Height:          height,
			ProposerAddress: proposer,
		})
		require.NoError(t, err)
		require.Equal(t, abci.ResponseProcessProposal_ACCEPT, processed.Status)
	}

	_, err = hApp.FinalizeBlock(&abci.RequestFinalizeBlock{
		Txs:             [][]byte{commitInfo},
		Height:          height,
		ProposerAddress: proposer,
	})
	require.NoError(t, err)
	_, err = hApp.Commit()
	require.NoError(t, err)
}

// A current record outside the stored set would give CometBFT a validator the vote
// extension checks don't know, so InitChain rejects it.
func TestInitChainRejectsCurrentRecordOutsideTheSet(t *testing.T) {
	setup := app.SetupApp(t, 3)
	hApp := setup.App
	ctx := hApp.NewUncachedContext(false, cmtproto.Header{Height: hApp.LastBlockHeight()})
	signedBy := hApp.StakeKeeper.GetCurrentValidators(ctx)

	rotated, err := hApp.StakeKeeper.GetValidatorFromValID(ctx, 0)
	require.NoError(t, err)
	newPub := secp256k1.GenPrivKey().PubKey()
	require.NoError(t, hApp.StakeKeeper.UpdateSigner(ctx, newPub.Address().String(), newPub.Bytes(), rotated.Signer))
	finalizeAndCommit(t, hApp, hApp.LastBlockHeight()+1, signedBy, signedBy)

	exported, err := hApp.ExportAppStateAndValidators(false, nil, nil)
	require.NoError(t, err)

	var appState map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(exported.AppState, &appState))
	stakeState := stakeTypes.GetGenesisStateFromAppState(hApp.AppCodec(), appState)
	for _, v := range stakeState.Validators {
		if util.FormatAddress(v.Signer) == util.FormatAddress(rotated.Signer) {
			v.VotingPower, v.EndEpoch = 100, 0
		}
	}
	appState[stakeTypes.ModuleName] = hApp.AppCodec().MustMarshalJSON(stakeState)
	appStateBytes, err := json.Marshal(appState)
	require.NoError(t, err)

	appOptions := make(simtestutil.AppOptionsMap)
	appOptions[flags.FlagHome] = app.DefaultNodeHome
	fresh := app.NewHeimdallApp(log.NewTestLogger(t), dbm.NewMemDB(), nil, true, appOptions)

	_, err = fresh.InitChain(&abci.RequestInitChain{
		ChainId:         hApp.ChainID(),
		ConsensusParams: &exported.ConsensusParams,
		AppStateBytes:   appStateBytes,
		InitialHeight:   exported.Height,
	})
	require.ErrorContains(t, err, "invalid stake genesis: validator 0 is current but not in the current set")
}

func TestExportAppStateAndValidators_FailsOnUnreadableAckCount(t *testing.T) {
	setup := app.SetupApp(t, 1)
	hApp := setup.App
	ctx := hApp.NewUncachedContext(false, cmtproto.Header{Height: hApp.LastBlockHeight()})

	// an ack count too short to decode
	ctx.KVStore(hApp.GetKey(checkpointTypes.StoreKey)).Set(checkpointTypes.AckCountPrefixKey, []byte{0x01})

	_, err := hApp.ExportAppStateAndValidators(false, nil, nil)
	require.ErrorContains(t, err, "wanted at least 8, got: 1")
}
