package keeper_test

import (
	"math/rand"
	"strings"
	"testing"
	"time"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttime "github.com/cometbft/cometbft/types/time"
	"github.com/stretchr/testify/suite"

	storetypes "cosmossdk.io/store/types"

	"github.com/0xPolygon/heimdall-v2/helper/mocks"
	hmModule "github.com/0xPolygon/heimdall-v2/module"
	cmKeeper "github.com/0xPolygon/heimdall-v2/x/chainmanager/keeper"
	cmTypes "github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
	stakeKeeper "github.com/0xPolygon/heimdall-v2/x/stake/keeper"
	testUtil "github.com/0xPolygon/heimdall-v2/x/stake/testutil"
	"github.com/0xPolygon/heimdall-v2/x/stake/types"
	stakeTypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	addrCodec "github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/types/simulation"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

var (
	pk1      = secp256k1.GenPrivKey().PubKey()
	pk2      = secp256k1.GenPrivKey().PubKey()
	pk3      = secp256k1.GenPrivKey().PubKey()
	valAddr1 = sdk.ValAddress(pk1.Address())
	valAddr2 = sdk.ValAddress(pk2.Address())
	valAddr3 = sdk.ValAddress(pk3.Address())
)

type KeeperTestSuite struct {
	suite.Suite

	ctx                sdk.Context
	contractCaller     *mocks.IContractCaller
	moduleCommunicator *testUtil.ModuleCommunicatorMock
	cmKeeper           *cmKeeper.Keeper
	stakeKeeper        *stakeKeeper.Keeper
	queryClient        stakeTypes.QueryClient
	msgServer          stakeTypes.MsgServer
	sideMsgCfg         hmModule.SideTxConfigurator
}

func (s *KeeperTestSuite) SetupTest() {
	key := storetypes.NewKVStoreKey(stakeTypes.StoreKey)
	storeService := runtime.NewKVStoreService(key)

	testCtx := testutil.DefaultContextWithDB(s.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx.WithBlockHeader(cmtproto.Header{Time: cmttime.Now()})
	encCfg := moduletestutil.MakeTestEncodingConfig()

	s.contractCaller = &mocks.IContractCaller{}

	cmKeeper := cmKeeper.NewKeeper(encCfg.Codec, storeService)
	_ = cmKeeper.SetParams(ctx, cmTypes.DefaultParams())

	s.moduleCommunicator = &testUtil.ModuleCommunicatorMock{AckCount: uint64(0)}

	keeper := stakeKeeper.NewKeeper(
		encCfg.Codec,
		storeService,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		s.moduleCommunicator,
		&cmKeeper,
		addrCodec.NewHexCodec(),
		s.contractCaller,
	)

	s.ctx = ctx
	s.cmKeeper = &cmKeeper
	s.stakeKeeper = keeper

	stakeTypes.RegisterInterfaces(encCfg.InterfaceRegistry)
	queryHelper := baseapp.NewQueryServerTestHelper(ctx, encCfg.InterfaceRegistry)
	stakeTypes.RegisterQueryServer(queryHelper, stakeKeeper.Querier{Keeper: keeper})
	s.queryClient = stakeTypes.NewQueryClient(queryHelper)
	s.msgServer = stakeKeeper.NewMsgServerImpl(keeper)

	s.sideMsgCfg = hmModule.NewSideTxConfigurator()
	types.RegisterSideMsgServer(s.sideMsgCfg, stakeKeeper.NewSideMsgServerImpl(keeper))

}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

func (s *KeeperTestSuite) TestValidator() {
	ctx, keeper := s.ctx, s.stakeKeeper
	require := s.Require()

	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	n := 5

	validators := make([]*types.Validator, n)
	accounts := simulation.RandomAccounts(r1, n)

	for i := range validators {
		validators[i] = types.NewValidator(
			uint64(i),
			0,
			0,
			1,
			int64(simulation.RandIntBetween(r1, 10, 100)), // power
			pk1,
			accounts[i].Address.String(),
		)

		err := keeper.AddValidator(ctx, *validators[i])
		require.NoErrorf(err, "Error while adding validator to store")
	}

	// get random validator ID
	valId := simulation.RandIntBetween(r1, 0, n)

	// get validator info from state
	valInfo, err := keeper.GetValidatorInfo(ctx, validators[valId].Signer)
	require.NoErrorf(err, "Error while fetching Validator")

	// get signer address mapped with validatorId
	mappedSignerAddress, isMapped := keeper.GetSignerFromValidatorID(ctx, validators[0].ValId)
	require.Truef(isMapped, "Signer Address not mapped to Validator Id")

	// check if validator matches in state
	require.Equal(valInfo, *validators[valId], "Validators in state doesn't match")
	require.Equal(strings.ToLower(mappedSignerAddress.String()), validators[0].Signer, "Signer address doesn't match")
}

// tests VotingPower change, validator creation, validator set update when signer changes
func (s *KeeperTestSuite) TestUpdateSigner() {
	ctx, keeper := s.ctx, s.stakeKeeper
	require := s.Require()

	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	n := 5

	validators := make([]*types.Validator, n)
	accounts := simulation.RandomAccounts(r1, n)

	for i := range validators {
		validators[i] = types.NewValidator(
			uint64(int64(i)),
			0,
			0,
			1,
			int64(simulation.RandIntBetween(r1, 10, 100)), // power
			pk1,
			accounts[i].Address.String(),
		)

		err := keeper.AddValidator(ctx, *validators[i])
		require.NoErrorf(err, "Error while adding validator to store")
	}

	// Fetch Validator Info from Store
	valInfo, err := keeper.GetValidatorInfo(ctx, validators[0].Signer)
	require.NoErrorf(err, "Error while fetching Validator Info from store")

	pkAny2, err := codectypes.NewAnyWithValue(pk2)
	require.NoError(err)

	addr2 := strings.ToLower(valAddr2.String())

	err = keeper.UpdateSigner(ctx, addr2, pkAny2, valInfo.Signer)
	require.NoErrorf(err, "Error while updating Signer Address ")

	// check validator info of prev signer
	prevSignerValInfo, err := keeper.GetValidatorInfo(ctx, validators[0].Signer)
	require.NoErrorf(err, "Error while fetching Validator Info for Prev Signer")

	require.Equal(int64(0), prevSignerValInfo.VotingPower, "VotingPower of Prev Signer should be zero")

	// check validator info of updated signer
	updatedSignerValInfo, err := keeper.GetValidatorInfo(ctx, addr2)
	require.NoError(err, "Error while fetching Validator Info for Updater Signer")

	require.Equal(validators[0].VotingPower, updatedSignerValInfo.VotingPower, "VotingPower of updated signer should match with prev signer VotingPower")

	// check if validatorId is mapped to updated signer
	signerAddress, isMapped := keeper.GetSignerFromValidatorID(ctx, validators[0].ValId)
	require.Truef(isMapped, "Signer Address not mapped to Validator Id")
	require.Equal(addr2, strings.ToLower(signerAddress.String()), "Validator ID should be mapped to Updated Signer Address")

	// check total validators
	totalValidators := keeper.GetAllValidators(ctx)
	require.LessOrEqual(6, len(totalValidators), "Total Validators should be six.")

	// check current validators
	currentValidators := keeper.GetCurrentValidators(ctx)
	require.LessOrEqual(5, len(currentValidators), "Current Validators should be five.")
}

// TODO HV2
/*
func (s *KeeperTestSuite) TestCurrentValidator() {
	type TestDataItem struct {
		name        string
		startblock  uint64
		VotingPower int64
		ackcount    uint64
		result      bool
		resultmsg   string
	}

	dataItems := []TestDataItem{
		{
			name:        "VotingPower zero",
			startblock:  uint64(0),
			VotingPower: int64(0),
			ackcount:    uint64(1),
			result:      false,
			resultmsg:   "should not be current validator as VotingPower is zero.",
		},
		{
			name:        "start epoch greater than ackcount",
			startblock:  uint64(3),
			VotingPower: int64(10),
			ackcount:    uint64(1),
			result:      false,
			resultmsg:   "should not be current validator as start epoch greater than ackcount.",
		},
	}
	ctx, keeper := s.ctx, s.stakeKeeper

	stakeKeeper, checkpointKeeper := s.stakeKeeper, app.CheckpointKeeper

	for i, item := range dataItems {
		suite.Run(item.name, func() {
			// Create a Validator [startEpoch, endEpoch, VotingPower]
			privKep := secp256k1.GenPrivKey()
			pubkey := types.NewPubKey(privKep.PubKey().Bytes())
			newVal := types.Validator{
				ID:               types.NewValidatorID(1 + uint64(i)),
				StartEpoch:       item.startblock,
				EndEpoch:         item.startblock,
				Nonce:            0,
				VotingPower:      item.VotingPower,
				Signer:           types.HexToHeimdallAddress(pubkey.Address().String()),
				PubKey:           pubkey,
				ProposerPriority: 0,
			}
			// check current validator
			err := stakeKeeper.AddValidator(ctx, newVal)
			require.NoError(t, err)
			checkpointKeeper.UpdateACKCountWithValue(ctx, item.ackcount)

			isCurrentVal := keeper.IsCurrentValidatorByAddress(ctx, newVal.Signer.Bytes())
			require.Equal(t, item.result, isCurrentVal, item.resultmsg)
		})
	}
}
*/

func (s *KeeperTestSuite) TestRemoveValidatorSetChange() {
	ctx, keeper := s.ctx, s.stakeKeeper
	require := s.Require()

	// load 4 validators from state
	testUtil.LoadValidatorSet(require, 4, keeper, ctx, false, 10)
	initValSet := keeper.GetValidatorSet(ctx)

	currentValSet := initValSet.Copy()
	prevValidatorSet := initValSet.Copy()

	prevValidatorSet.Validators[0].StartEpoch = 20

	err := keeper.AddValidator(ctx, *prevValidatorSet.Validators[0])
	require.NoError(err)

	setUpdates := types.GetUpdatedValidators(currentValSet, keeper.GetAllValidators(ctx), 5)
	err = currentValSet.UpdateWithChangeSet(setUpdates)
	require.NoError(err)

	updatedValSet := currentValSet

	require.Equal(len(prevValidatorSet.Validators)-1, len(updatedValSet.Validators), "Validator set should be reduced by one ")

	removedVal := prevValidatorSet.Validators[0].Signer

	for _, val := range updatedValSet.Validators {
		if strings.ToLower(val.Signer) == strings.ToLower(removedVal) {
			require.Fail("Validator is not removed from updated validator set")
		}
	}
}

func (s *KeeperTestSuite) TestAddValidatorSetChange() {
	ctx, keeper := s.ctx, s.stakeKeeper
	require := s.Require()

	// load 4 validators from state
	testUtil.LoadValidatorSet(require, 4, keeper, ctx, false, 10)
	initValSet := keeper.GetValidatorSet(ctx)

	validators := testUtil.GenRandomVal(1, 0, 10, 10, false, 1)
	prevValSet := initValSet.Copy()

	valToBeAdded := validators[0]
	currentValSet := initValSet.Copy()

	err := keeper.AddValidator(ctx, valToBeAdded)
	require.NoError(err)

	_, err = keeper.GetValidatorInfo(ctx, valToBeAdded.GetSigner())
	require.NoError(err)

	setUpdates := types.GetUpdatedValidators(currentValSet, keeper.GetAllValidators(ctx), 5)
	err = currentValSet.UpdateWithChangeSet(setUpdates)
	require.NoError(err)
	require.Equal(len(prevValSet.Validators)+1, len(currentValSet.Validators), "Number of validators should be increased by 1")
	require.Equal(true, currentValSet.HasAddress(valToBeAdded.Signer), "New Validator should be added")
	require.Equal(prevValSet.GetTotalVotingPower()+valToBeAdded.VotingPower, currentValSet.GetTotalVotingPower(), "Total VotingPower should be increased")
}

/*
	 Validator Set changes When
		1. When ackCount changes
		2. When new validator joins
		3. When validator updates stake
		4. When signer is updatedctx
		5. When Validator Exits

*
*/
func (s *KeeperTestSuite) TestUpdateValidatorSetChange() {
	ctx, keeper := s.ctx, s.stakeKeeper
	require := s.Require()

	// load 4 validators to state
	testUtil.LoadValidatorSet(require, 4, keeper, ctx, false, 10)
	initValSet := keeper.GetValidatorSet(ctx)

	keeper.IncrementAccum(ctx, 2)

	prevValSet := initValSet.Copy()
	currentValSet := keeper.GetValidatorSet(ctx)

	valToUpdate := currentValSet.Validators[0]
	newSigner := testUtil.GenRandomVal(1, 0, 10, 10, false, 1)

	err := keeper.UpdateSigner(ctx, newSigner[0].Signer, newSigner[0].PubKey, valToUpdate.Signer)
	require.NoError(err)

	setUpdates := types.GetUpdatedValidators(&currentValSet, keeper.GetAllValidators(ctx), 5)
	err = currentValSet.UpdateWithChangeSet(setUpdates)
	require.NoError(err)

	require.Equal(len(prevValSet.Validators), len(currentValSet.Validators), "Number of validators should remain same")

	index, _ := currentValSet.GetByAddress(valToUpdate.Signer)
	require.Equal(-1, index, "Prev Validator should not be present in CurrentValSet")

	_, newVal := currentValSet.GetByAddress(newSigner[0].Signer)
	require.Equal(newSigner[0].Signer, newVal.Signer, "Signer address should be update")
	require.Equal(newSigner[0].PubKey, newVal.PubKey, "Signer pubkey should should be updated")

	require.Equal(prevValSet.GetTotalVotingPower(), currentValSet.GetTotalVotingPower(), "Total VotingPower should not change")
}

func (s *KeeperTestSuite) TestGetCurrentValidators() {
	ctx, keeper := s.ctx, s.stakeKeeper
	require := s.Require()
	testUtil.LoadValidatorSet(require, 4, keeper, ctx, false, 10)
	validators := keeper.GetCurrentValidators(ctx)
	activeValidatorInfo, err := keeper.GetActiveValidatorInfo(ctx, validators[0].Signer)
	require.NoError(err)
	require.Equal(validators[0], activeValidatorInfo)
}

func (s *KeeperTestSuite) TestGetCurrentProposer() {
	ctx, keeper := s.ctx, s.stakeKeeper
	require := s.Require()

	testUtil.LoadValidatorSet(require, 4, keeper, ctx, false, 10)
	currentValSet := keeper.GetValidatorSet(ctx)
	currentProposer := keeper.GetCurrentProposer(ctx)
	require.Equal(currentValSet.GetProposer(), currentProposer)
}

func (s *KeeperTestSuite) TestGetNextProposer() {
	ctx, keeper := s.ctx, s.stakeKeeper
	require := s.Require()
	testUtil.LoadValidatorSet(require, 4, keeper, ctx, false, 10)

	nextProposer := keeper.GetNextProposer(ctx)
	require.NotNil(nextProposer)
}

func (s *KeeperTestSuite) TestGetValidatorFromValID() {
	ctx, keeper := s.ctx, s.stakeKeeper
	require := s.Require()
	testUtil.LoadValidatorSet(require, 4, keeper, ctx, false, 10)
	validators := keeper.GetCurrentValidators(ctx)

	valInfo, ok := keeper.GetValidatorFromValID(ctx, validators[0].ValId)
	require.Equal(ok, true)
	require.Equal(validators[0], valInfo)
}

func (s *KeeperTestSuite) TestGetLastUpdated() {
	ctx, keeper := s.ctx, s.stakeKeeper
	require := s.Require()
	testUtil.LoadValidatorSet(require, 4, keeper, ctx, false, 10)
	validators := keeper.GetCurrentValidators(ctx)

	lastUpdated, ok := keeper.GetLastUpdated(ctx, validators[0].ValId)
	require.Equal(ok, true)
	require.Equal(validators[0].LastUpdated, lastUpdated)
}

func (s *KeeperTestSuite) TestGetSpanEligibleValidators() {
	ctx, keeper := s.ctx, s.stakeKeeper
	require := s.Require()
	testUtil.LoadValidatorSet(require, 4, keeper, ctx, false, 0)

	// Test ActCount = 0
	s.moduleCommunicator.AckCount = 0

	valActCount0 := keeper.GetSpanEligibleValidators(ctx)
	require.LessOrEqual(len(valActCount0), 4)

	s.moduleCommunicator.AckCount = 20

	validators := keeper.GetSpanEligibleValidators(ctx)
	require.LessOrEqual(len(validators), 4)
}

func (s *KeeperTestSuite) TestGetMilestoneProposer() {
	ctx, keeper := s.ctx, s.stakeKeeper
	require := s.Require()
	testUtil.LoadValidatorSet(require, 4, keeper, ctx, false, 10)
	currentValSet1 := keeper.GetMilestoneValidatorSet(ctx)
	currentMilestoneProposer := keeper.GetMilestoneCurrentProposer(ctx)
	require.Equal(currentValSet1.GetProposer(), currentMilestoneProposer)

	keeper.MilestoneIncrementAccum(ctx, 1)

	currentValSet2 := keeper.GetMilestoneValidatorSet(ctx)
	currentMilestoneProposer = keeper.GetMilestoneCurrentProposer(ctx)
	require.NotEqual(currentValSet1.GetProposer(), currentMilestoneProposer)
	require.Equal(currentValSet2.GetProposer(), currentMilestoneProposer)
}

func (s *KeeperTestSuite) TestMilestoneValidatorSetIncAccumChange() {
	ctx, keeper := s.ctx, s.stakeKeeper
	require := s.Require()

	// load 4 validators to state
	testUtil.LoadValidatorSet(require, 4, keeper, ctx, false, 10)

	initMilestoneValSetProp := keeper.GetMilestoneValidatorSet(ctx).Proposer
	initCheckpointValSetProp := keeper.GetValidatorSet(ctx).Proposer

	require.Equal(initMilestoneValSetProp, initCheckpointValSetProp)

	keeper.IncrementAccum(ctx, 1)

	initMilestoneValSetProp = keeper.GetMilestoneValidatorSet(ctx).Proposer
	initCheckpointValSetProp = keeper.GetValidatorSet(ctx).Proposer

	require.Equal(initMilestoneValSetProp, initCheckpointValSetProp)

	initValSet := keeper.GetMilestoneValidatorSet(ctx)

	keeper.MilestoneIncrementAccum(ctx, 1)

	initValSet.IncrementProposerPriority(1)
	_proposer := initValSet.Proposer

	currentValSet := keeper.GetMilestoneValidatorSet(ctx)
	proposer := currentValSet.Proposer

	require.Equal(_proposer, proposer)
}

func (s *KeeperTestSuite) TestUpdateMilestoneValidatorSetChange() {
	ctx, keeper := s.ctx, s.stakeKeeper
	require := s.Require()

	// load 4 validators to state
	testUtil.LoadValidatorSet(require, 4, keeper, ctx, false, 10)
	initValSet := keeper.GetMilestoneValidatorSet(ctx)

	keeper.MilestoneIncrementAccum(ctx, 1)

	prevValSet := initValSet.Copy()
	currentValSet := keeper.GetMilestoneValidatorSet(ctx)

	valToUpdate := currentValSet.Validators[0]
	newSigner := testUtil.GenRandomVal(1, 0, 10, 10, false, 1)

	err := keeper.UpdateSigner(ctx, newSigner[0].Signer, newSigner[0].PubKey, valToUpdate.Signer)
	require.NoError(err)

	setUpdates := types.GetUpdatedValidators(&currentValSet, keeper.GetAllValidators(ctx), 5)
	err = currentValSet.UpdateWithChangeSet(setUpdates)
	require.NoError(err)

	require.Equal(len(prevValSet.Validators), len(currentValSet.Validators), "Number of validators should remain same")

	index, _ := currentValSet.GetByAddress(valToUpdate.Signer)
	require.Equal(-1, index, "Prev Validator should not be present in CurrentValSet")

	_, newVal := currentValSet.GetByAddress(newSigner[0].Signer)
	require.Equal(newSigner[0].Signer, newVal.Signer, "Signer address should change")
	require.Equal(newSigner[0].PubKey, newVal.PubKey, "Signer pubkey should change")

	require.Equal(prevValSet.GetTotalVotingPower(), currentValSet.GetTotalVotingPower(), "Total VotingPower should not change")
}
