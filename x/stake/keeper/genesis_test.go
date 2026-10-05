package keeper_test

import (
	"math/rand"
	"sort"
	"strconv"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/types/simulation"

	util "github.com/0xPolygon/heimdall-v2/common/hex"
	"github.com/0xPolygon/heimdall-v2/x/stake/types"
)

func (s *KeeperTestSuite) TestInitExportGenesis() {
	ctx, keeper, require := s.ctx, s.stakeKeeper, s.Require()

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	n := 5

	stakingSequence := make([]string, n)
	accounts := simulation.RandomAccounts(r, n)

	for i := range stakingSequence {
		stakingSequence[i] = strconv.Itoa(simulation.RandIntBetween(r, 1000, 100000))
	}

	validators := make([]*types.Validator, n)
	var err error
	for i := 0; i < len(validators); i++ {
		pk1 := secp256k1.GenPrivKey().PubKey()
		validators[i], err = types.NewValidator(
			uint64(i),
			0,
			0,
			uint64(i),
			int64(simulation.RandIntBetween(r, 10, 100)), // power
			pk1,
			accounts[i].Address.String(),
		)

		require.NoError(err)
	}

	validatorSet := types.NewValidatorSet(validators)

	genesisState := types.NewGenesisState(validators, *validatorSet, stakingSequence)
	keeper.InitGenesis(ctx, genesisState)
	valSet, err := keeper.GetPreviousBlockValidatorSet(ctx)
	require.NoError(err)
	require.Equal(validatorSet.Len(), valSet.Len())
	require.Equal(validatorSet.Proposer.Signer, valSet.Proposer.Signer)
	require.Equal(validatorSet.TotalVotingPower, valSet.TotalVotingPower)

	prevN := 3
	prevAccounts := simulation.RandomAccounts(r, prevN)
	prevValidators := make([]*types.Validator, prevN)
	for i := 0; i < prevN; i++ {
		pkPrev := secp256k1.GenPrivKey().PubKey()
		prevValidators[i], err = types.NewValidator(uint64(100+i), 0, 0, uint64(i), int64(simulation.RandIntBetween(r, 10, 100)), pkPrev, prevAccounts[i].Address.String())
		require.NoError(err)
	}
	prevSet := types.NewValidatorSet(prevValidators)
	err = keeper.UpdatePreviousBlockValidatorSetInStore(ctx, *prevSet)
	require.NoError(err)

	penN := 2
	penAccounts := simulation.RandomAccounts(r, penN)
	penValidators := make([]*types.Validator, penN)
	for i := 0; i < penN; i++ {
		pkPen := secp256k1.GenPrivKey().PubKey()
		penValidators[i], err = types.NewValidator(uint64(200+i), 0, 0, uint64(i), int64(simulation.RandIntBetween(r, 10, 100)), pkPen, penAccounts[i].Address.String())
		require.NoError(err)
	}
	penultimate := types.NewValidatorSet(penValidators)
	err = keeper.UpdatePenultimateBlockValidatorSetInStore(ctx, *penultimate)
	require.NoError(err)

	customTxs := [][]byte{[]byte("tx-a"), []byte("tx-b")}
	require.NoError(keeper.SetLastBlockTxs(ctx, customTxs))

	actualParams := keeper.ExportGenesis(ctx)
	require.NotNil(actualParams)
	require.LessOrEqual(n, len(actualParams.Validators))
	require.True(genesisState.CurrentValidatorSet.Equal(actualParams.CurrentValidatorSet))

	require.True(prevSet.Equal(actualParams.PreviousBlockValidatorSet))
	require.LessOrEqual(prevN, len(actualParams.Validators))

	require.True(penultimate.Equal(actualParams.PenultimateBlockValidatorSet))
	require.LessOrEqual(penN, len(actualParams.Validators))

	require.Equal(customTxs, actualParams.LastBlockTxs.Txs)
}

func (s *KeeperTestSuite) makeValidators(base uint64, count int) []*types.Validator {
	r := rand.New(rand.NewSource(int64(base) + 1))
	accounts := simulation.RandomAccounts(r, count)
	validators := make([]*types.Validator, count)
	for i := 0; i < count; i++ {
		pk := secp256k1.GenPrivKey().PubKey()
		v, err := types.NewValidator(base+uint64(i), 0, 0, uint64(i), int64(10+i), pk, accounts[i].Address.String())
		s.Require().NoError(err)
		validators[i] = v
	}
	return validators
}

// A genesis re-bootstrap imports an exported genesis: the penultimate set it
// carries must survive InitGenesis, otherwise the H-2 set is empty on the first
// block and vote-extension verification fails.
func (s *KeeperTestSuite) TestInitGenesisRestoresExportedPenultimateSet() {
	ctx, keeper, require := s.ctx, s.stakeKeeper, s.Require()

	current := s.makeValidators(0, 4)
	currentSet := types.NewValidatorSet(current)
	penultimate := types.NewValidatorSet(s.makeValidators(200, 2))

	gen := types.NewGenesisState(current, *currentSet, nil)
	gen.PenultimateBlockValidatorSet = *penultimate

	keeper.InitGenesis(ctx, gen)

	got, err := keeper.GetPenultimateBlockValidatorSet(ctx)
	require.NoError(err)
	require.True(penultimate.Equal(got))
}

func (s *KeeperTestSuite) TestInitGenesisPenultimateFallsBackToCurrent() {
	ctx, keeper, require := s.ctx, s.stakeKeeper, s.Require()

	current := s.makeValidators(0, 4)
	currentSet := types.NewValidatorSet(current)

	// no PenultimateBlockValidatorSet (legacy/pre-field export)
	gen := types.NewGenesisState(current, *currentSet, nil)

	keeper.InitGenesis(ctx, gen)

	got, err := keeper.GetPenultimateBlockValidatorSet(ctx)
	require.NoError(err)
	require.Equal(currentSet.Len(), got.Len())
}

type genesisKey struct {
	pub    cryptotypes.PubKey
	signer string
}

// sortedGenesisKeys returns n keys ordered by signer address, so a test can pick which
// record GetAllValidators iterates last.
func sortedGenesisKeys(n int) []genesisKey {
	keys := make([]genesisKey, n)
	for i := range keys {
		pub := secp256k1.GenPrivKey().PubKey()
		keys[i] = genesisKey{pub: pub, signer: util.FormatAddress(pub.Address().String())}
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].signer < keys[j].signer })
	return keys
}

func (s *KeeperTestSuite) newValidator(id, startEpoch, endEpoch, nonce uint64, power int64, key genesisKey) *types.Validator {
	v, err := types.NewValidator(id, startEpoch, endEpoch, nonce, power, key.pub, key.signer)
	s.Require().NoError(err)
	return v
}

// seedDivergedStore builds the store a running chain reaches but a set-only import loses:
// validator 1's record moved past its set copy (a stake update that kept its voting
// power), validator 9 joined and waits for its start epoch, and validator 2 rotated to a
// signer that sorts before its old record.
func (s *KeeperTestSuite) seedDivergedStore() (rotatedSigner string) {
	ctx, keeper, require := s.ctx, s.stakeKeeper, s.Require()

	keys := sortedGenesisKeys(5)
	rotatedNew, stale, steady, pending, rotatedOld := keys[0], keys[1], keys[2], keys[3], keys[4]

	current := []*types.Validator{
		s.newValidator(1, 0, 0, 3, 10, stale),
		s.newValidator(2, 0, 0, 3, 20, rotatedOld),
		s.newValidator(3, 0, 0, 3, 30, steady),
	}
	keeper.InitGenesis(ctx, types.NewGenesisState(current, *types.NewValidatorSet(current), nil))

	moved, err := keeper.GetValidatorFromValID(ctx, 1)
	require.NoError(err)
	moved.Nonce = 7
	moved.LastUpdated = "777"
	require.NoError(keeper.AddValidator(ctx, moved))

	require.NoError(keeper.AddValidator(ctx, *s.newValidator(9, 100, 0, 1, 5, pending)))

	old, err := keeper.GetValidatorFromValID(ctx, 2)
	require.NoError(err)
	rotated := old
	old.VotingPower = 0
	old.EndEpoch = 4
	old.Nonce = 4
	require.NoError(keeper.AddValidator(ctx, old))
	rotated.Signer = rotatedNew.signer
	rotated.PubKey = rotatedNew.pub.Bytes()
	rotated.Nonce = 4
	require.NoError(keeper.AddValidator(ctx, rotated))

	// the stake EndBlocker swaps the rotated member into the set; validator 1's set copy
	// keeps its old nonce because its power didn't change
	set, err := keeper.GetValidatorSet(ctx)
	require.NoError(err)
	for i, member := range set.Validators {
		if member.ValId == rotated.ValId {
			swapped := rotated
			set.Validators[i] = &swapped
		}
	}
	sort.Sort(types.ValidatorsByAddress(set.Validators))
	require.NoError(keeper.UpdateValidatorSetInStore(ctx, set))

	// a previous block set that differs from the current one, as after a set change
	require.NoError(keeper.UpdatePreviousBlockValidatorSetInStore(ctx, *types.NewValidatorSet(current[:2])))

	return rotatedNew.signer
}

func (s *KeeperTestSuite) TestExportImportGenesisKeepsValidatorStore() {
	rotatedSigner := s.seedDivergedStore()
	exported := s.stakeKeeper.ExportGenesis(s.ctx)
	s.Require().NotNil(exported)
	s.Require().NoError(exported.Validate())

	s.SetupTest()
	ctx, keeper, require := s.ctx, s.stakeKeeper, s.Require()
	keeper.InitGenesis(ctx, exported)

	stale, err := keeper.GetValidatorFromValID(ctx, 1)
	require.NoError(err)
	require.Equal(uint64(7), stale.Nonce)
	require.Equal("777", stale.LastUpdated)

	pending, err := keeper.GetValidatorFromValID(ctx, 9)
	require.NoError(err)
	require.Equal(uint64(100), pending.StartEpoch)

	signer, err := keeper.GetSignerFromValidatorID(ctx, 2)
	require.NoError(err)
	require.Equal(rotatedSigner, signer)

	currentSet, err := keeper.GetValidatorSet(ctx)
	require.NoError(err)
	require.True(exported.CurrentValidatorSet.Equal(currentSet))
	// the seeded previous set differs from the current one; the import must not restore it
	require.False(exported.PreviousBlockValidatorSet.Equal(exported.CurrentValidatorSet))
	previousSet, err := keeper.GetPreviousBlockValidatorSet(ctx)
	require.NoError(err)
	require.True(exported.CurrentValidatorSet.Equal(previousSet))
	penultimateSet, err := keeper.GetPenultimateBlockValidatorSet(ctx)
	require.NoError(err)
	require.True(exported.PenultimateBlockValidatorSet.Equal(penultimateSet))

	reexported := keeper.ExportGenesis(ctx)
	require.Equal(exported.Validators, reexported.Validators)
	require.Equal(exported.ValidatorSigners, reexported.ValidatorSigners)
}

// Genesis files that predate ValidatorSigners (the v1 to v2 migration outputs) must
// import exactly as before, stale set copies and dropped records included.
func (s *KeeperTestSuite) TestInitGenesisWithoutValidatorSignersKeepsLegacyImport() {
	s.seedDivergedStore()
	exported := s.stakeKeeper.ExportGenesis(s.ctx)
	s.Require().NotNil(exported)
	exported.ValidatorSigners = nil

	s.SetupTest()
	ctx, keeper, require := s.ctx, s.stakeKeeper, s.Require()
	keeper.InitGenesis(ctx, exported)

	stale, err := keeper.GetValidatorFromValID(ctx, 1)
	require.NoError(err)
	require.Equal(uint64(3), stale.Nonce)

	exists, err := keeper.DoesValIdExist(ctx, 9)
	require.NoError(err)
	require.False(exists)

	require.Len(keeper.GetAllValidators(ctx), len(exported.CurrentValidatorSet.Validators))
}

func (s *KeeperTestSuite) TestInitGenesisPanicsOnInconsistentValidatorSigners() {
	s.seedDivergedStore()
	exported := s.stakeKeeper.ExportGenesis(s.ctx)
	s.Require().NotNil(exported)
	exported.ValidatorSigners = exported.ValidatorSigners[1:]

	s.SetupTest()
	s.Require().PanicsWithError(
		"invalid validator signers in stake genesis: validator 1 has no signer mapping",
		func() { s.stakeKeeper.InitGenesis(s.ctx, exported) },
	)
}

func (s *KeeperTestSuite) TestInitGenesisWithoutCurrentSetBuildsItFromValidators() {
	validators := s.makeValidators(0, 3)

	s.stakeKeeper.InitGenesis(s.ctx, types.NewGenesisState(validators, types.ValidatorSet{}, nil))

	set, err := s.stakeKeeper.GetValidatorSet(s.ctx)
	s.Require().NoError(err)
	s.Require().Len(set.Validators, len(validators))
}
