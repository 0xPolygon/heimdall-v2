package keeper_test

import (
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"

	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttime "github.com/cometbft/cometbft/types/time"
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
	"github.com/stretchr/testify/suite"

	"github.com/0xPolygon/heimdall-v2/helper/mocks"
	hmModule "github.com/0xPolygon/heimdall-v2/module"
	cmKeeper "github.com/0xPolygon/heimdall-v2/x/chainmanager/keeper"
	cmTypes "github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
	stakeKeeper "github.com/0xPolygon/heimdall-v2/x/stake/keeper"
	testUtil "github.com/0xPolygon/heimdall-v2/x/stake/testutil"
	"github.com/0xPolygon/heimdall-v2/x/stake/types"
	stakeTypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

var (
	pk       = secp256k1.GenPrivKey().PubKey()
	pk2      = secp256k1.GenPrivKey().PubKey()
	valAddr2 = sdk.ValAddress(pk2.Address())
)

type KeeperTestSuite struct {
	suite.Suite

	ctx         sdk.Context
	stakeKeeper *stakeKeeper.Keeper

	contractCaller   *mocks.IContractCaller
	checkpointKeeper *testUtil.MockCheckpointKeeper
	bankKeeper       *testUtil.MockBankKeeper
	cmKeeper         *cmKeeper.Keeper
	queryClient      stakeTypes.QueryClient
	msgServer        stakeTypes.MsgServer
	sideMsgCfg       hmModule.SideTxConfigurator
}

func (s *KeeperTestSuite) SetupTest() {
	key := storetypes.NewKVStoreKey(stakeTypes.StoreKey)
	storeService := runtime.NewKVStoreService(key)

	testCtx := testutil.DefaultContextWithDB(s.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx.WithBlockHeader(cmtproto.Header{Time: cmttime.Now()})
	encCfg := moduletestutil.MakeTestEncodingConfig()

	s.contractCaller = &mocks.IContractCaller{}

	authority := authtypes.NewModuleAddress(govtypes.ModuleName)
	cmk := cmKeeper.NewKeeper(encCfg.Codec, storeService, authority.String())
	err := cmk.SetParams(ctx, cmTypes.DefaultParams())
	s.Require().NoError(err)

	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()
	s.checkpointKeeper = testUtil.NewMockCheckpointKeeper(ctrl)

	s.bankKeeper = testUtil.NewMockBankKeeper(ctrl)

	keeper := stakeKeeper.NewKeeper(
		encCfg.Codec,
		storeService,
		s.checkpointKeeper,
		s.bankKeeper,
		cmk,
		addrCodec.NewHexCodec(),
		s.contractCaller,
	)

	s.ctx = ctx
	s.cmKeeper = &cmk
	s.stakeKeeper = &keeper

	stakeTypes.RegisterInterfaces(encCfg.InterfaceRegistry)
	queryHelper := baseapp.NewQueryServerTestHelper(ctx, encCfg.InterfaceRegistry)
	stakeTypes.RegisterQueryServer(queryHelper, stakeKeeper.NewQueryServer(&keeper))
	s.queryClient = stakeTypes.NewQueryClient(queryHelper)
	s.msgServer = stakeKeeper.NewMsgServerImpl(&keeper)

	s.sideMsgCfg = hmModule.NewSideTxConfigurator()
	types.RegisterSideMsgServer(s.sideMsgCfg, stakeKeeper.NewSideMsgServerImpl(&keeper))

}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

func (s *KeeperTestSuite) TestValidator() {
	ctx, keeper, require := s.ctx, s.stakeKeeper, s.Require()

	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	n := 5

	validators := make([]*types.Validator, n)
	accounts := simulation.RandomAccounts(r1, n)

	var err error

	for i := range validators {
		validators[i], err = types.NewValidator(
			uint64(i),
			0,
			0,
			1,
			int64(simulation.RandIntBetween(r1, 10, 100)), // power
			pk,
			accounts[i].Address.String(),
		)

		require.NoError(err)

		err = keeper.AddValidator(ctx, *validators[i])
		require.NoErrorf(err, "Error while adding validator to store")
	}

	// get random validator ID
	valId := simulation.RandIntBetween(r1, 0, n)

	// get validator info from state
	valInfo, err := keeper.GetValidatorInfo(ctx, validators[valId].Signer)
	require.NoErrorf(err, "Error while fetching Validator")

	// get signer address mapped with validatorId
	mappedSignerAddress, err := keeper.GetSignerFromValidatorID(ctx, validators[0].ValId)
	require.Nilf(err, "Signer Address not mapped to Validator Id")

	// check if validator matches in state
	require.Equal(valInfo, *validators[valId], "Validators in state doesn't match")
	require.Equal(mappedSignerAddress, validators[0].Signer, "Signer address doesn't match")
}

// tests VotingPower change, validator creation, validator set update when signer changes
func (s *KeeperTestSuite) TestUpdateSigner() {
	ctx, keeper, require := s.ctx, s.stakeKeeper, s.Require()

	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	n := 5

	validators := make([]*types.Validator, n)
	accounts := simulation.RandomAccounts(r1, n)

	var err error
	for i := range validators {
		validators[i], err = types.NewValidator(
			uint64(int64(i)),
			0,
			0,
			1,
			int64(simulation.RandIntBetween(r1, 10, 100)), // power
			pk,
			accounts[i].Address.String(),
		)

		require.NoError(err)

		err := keeper.AddValidator(ctx, *validators[i])
		require.NoErrorf(err, "Error while adding validator to store")
	}

	// fetch validator info from store
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
	signerAddress, err := keeper.GetSignerFromValidatorID(ctx, validators[0].ValId)
	require.Nilf(err, "Signer Address not mapped to Validator Id")
	require.Equal(addr2, signerAddress, "Validator ID should be mapped to Updated Signer Address")

	// check total validators
	totalValidators := keeper.GetAllValidators(ctx)
	require.LessOrEqual(6, len(totalValidators), "Total Validators should be six.")

	// check current validators
	s.checkpointKeeper.EXPECT().GetACKCount(gomock.Any()).Return(uint64(0)).Times(1)
	currentValidators := keeper.GetCurrentValidators(ctx)
	require.LessOrEqual(5, len(currentValidators), "Current Validators should be five.")
}

func (s *KeeperTestSuite) TestCurrentValidator() {

	ctx, keeper, require := s.ctx, s.stakeKeeper, s.Require()

	stakeKeeper := s.stakeKeeper

	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	n := 5

	accounts := simulation.RandomAccounts(r1, n)

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

	for i, item := range dataItems {
		s.Run(item.name, func() {
			newVal, err := types.NewValidator(1+uint64(i), item.startblock, item.startblock, uint64(0), item.VotingPower, accounts[i].PubKey, accounts[i].Address.String())

			require.NoError(err)

			// check current validator
			err = stakeKeeper.AddValidator(ctx, *newVal)
			require.NoError(err)

			s.checkpointKeeper.EXPECT().GetACKCount(gomock.Any()).Return(item.ackcount).Times(1)

			isCurrentVal := keeper.IsCurrentValidatorByAddress(ctx, newVal.Signer)
			require.Equal(item.result, isCurrentVal, item.resultmsg)
		})
	}
}

func (s *KeeperTestSuite) TestRemoveValidatorSetChange() {
	ctx, keeper, require := s.ctx, s.stakeKeeper, s.Require()

	// load 4 validators from state
	testUtil.LoadRandomValidatorSet(require, 4, keeper, ctx, false, 10)
	initValSet, err := keeper.GetValidatorSet(ctx)

	require.NoError(err)

	currentValSet := initValSet.Copy()
	prevValidatorSet := initValSet.Copy()

	prevValidatorSet.Validators[0].StartEpoch = 20

	err = keeper.AddValidator(ctx, *prevValidatorSet.Validators[0])
	require.NoError(err)

	setUpdates := types.GetUpdatedValidators(currentValSet, keeper.GetAllValidators(ctx), 5)
	err = currentValSet.UpdateWithChangeSet(setUpdates)
	require.NoError(err)

	updatedValSet := currentValSet

	require.Equal(len(prevValidatorSet.Validators)-1, len(updatedValSet.Validators), "Validator set should be reduced by one ")

	removedVal := prevValidatorSet.Validators[0].Signer

	for _, val := range updatedValSet.Validators {
		if strings.EqualFold(val.Signer, removedVal) {
			require.Fail("Validator is not removed from updated validator set")
		}
	}
}

func (s *KeeperTestSuite) TestAddValidatorSetChange() {
	ctx, keeper, require := s.ctx, s.stakeKeeper, s.Require()

	// load 4 validators from state
	testUtil.LoadRandomValidatorSet(require, 4, keeper, ctx, false, 10)
	initValSet, err := keeper.GetValidatorSet(ctx)

	require.NoError(err)

	validators := testUtil.GenRandomVals(1, 0, 10, 10, false, 1)
	prevValSet := initValSet.Copy()

	valToBeAdded := validators[0]
	currentValSet := initValSet.Copy()

	err = keeper.AddValidator(ctx, valToBeAdded)
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

func (s *KeeperTestSuite) TestUpdateValidatorSetChange() {
	ctx, keeper, require := s.ctx, s.stakeKeeper, s.Require()

	// load 4 validators to state
	testUtil.LoadRandomValidatorSet(require, 4, keeper, ctx, false, 10)
	initValSet, err := keeper.GetValidatorSet(ctx)
	require.NoError(err)

	err = keeper.IncrementAccum(ctx, 2)
	require.NoError(err)

	prevValSet := initValSet.Copy()
	currentValSet, err := keeper.GetValidatorSet(ctx)
	require.NoError(err)

	valToUpdate := currentValSet.Validators[0]
	newSigner := testUtil.GenRandomVals(1, 0, 10, 10, false, 1)

	err = keeper.UpdateSigner(ctx, newSigner[0].Signer, newSigner[0].PubKey, valToUpdate.Signer)
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
	ctx, keeper, require := s.ctx, s.stakeKeeper, s.Require()

	testUtil.LoadRandomValidatorSet(require, 4, keeper, ctx, false, 10)

	s.checkpointKeeper.EXPECT().GetACKCount(ctx).AnyTimes().Return(uint64(1))

	validators := keeper.GetCurrentValidators(ctx)
	activeValidatorInfo, err := keeper.GetActiveValidatorInfo(ctx, validators[0].Signer)
	require.NoError(err)
	require.Equal(validators[0], activeValidatorInfo)
}

func (s *KeeperTestSuite) TestGetCurrentProposer() {
	ctx, keeper, require := s.ctx, s.stakeKeeper, s.Require()

	testUtil.LoadRandomValidatorSet(require, 4, keeper, ctx, false, 10)
	currentValSet, err := keeper.GetValidatorSet(ctx)
	require.NoError(err)

	currentProposer := keeper.GetCurrentProposer(ctx)
	require.Equal(currentValSet.GetProposer(), currentProposer)
}

func (s *KeeperTestSuite) TestGetNextProposer() {
	ctx, keeper, require := s.ctx, s.stakeKeeper, s.Require()

	testUtil.LoadRandomValidatorSet(require, 4, keeper, ctx, false, 10)

	nextProposer := keeper.GetNextProposer(ctx)
	require.NotNil(nextProposer)
}

func (s *KeeperTestSuite) TestGetValidatorFromValID() {
	ctx, keeper, require := s.ctx, s.stakeKeeper, s.Require()

	testUtil.LoadRandomValidatorSet(require, 4, keeper, ctx, false, 10)
	s.checkpointKeeper.EXPECT().GetACKCount(ctx).AnyTimes().Return(uint64(1))

	validators := keeper.GetCurrentValidators(ctx)

	valInfo, err := keeper.GetValidatorFromValID(ctx, validators[0].ValId)
	require.NoError(err)
	require.Equal(validators[0], valInfo)
}

func (s *KeeperTestSuite) TestGetLastUpdated() {
	ctx, keeper, require := s.ctx, s.stakeKeeper, s.Require()

	testUtil.LoadRandomValidatorSet(require, 1, keeper, ctx, false, 10)
	s.checkpointKeeper.EXPECT().GetACKCount(ctx).AnyTimes().Return(uint64(1))

	validators := keeper.GetCurrentValidators(ctx)

	lastUpdated, err := keeper.GetLastUpdated(ctx, validators[0].ValId)
	require.NoError(err)
	require.Equal(validators[0].LastUpdated, lastUpdated)
}

func (s *KeeperTestSuite) TestGetSpanEligibleValidators() {
	ctx, keeper, require := s.ctx, s.stakeKeeper, s.Require()

	testUtil.LoadRandomValidatorSet(require, 4, keeper, ctx, false, 0)

	// Test ActCount = 0
	s.checkpointKeeper.EXPECT().GetACKCount(gomock.Any()).Return(uint64(0)).Times(1)

	valActCount0 := keeper.GetSpanEligibleValidators(ctx)
	require.LessOrEqual(len(valActCount0), 4)

	s.checkpointKeeper.EXPECT().GetACKCount(gomock.Any()).Return(uint64(0)).Times(20)

	validators := keeper.GetSpanEligibleValidators(ctx)
	require.LessOrEqual(len(validators), 4)
}

func (s *KeeperTestSuite) TestGetMilestoneProposer() {
	ctx, keeper, require := s.ctx, s.stakeKeeper, s.Require()

	testUtil.LoadRandomValidatorSet(require, 4, keeper, ctx, false, 10)
	currentValSet1, err := keeper.GetMilestoneValidatorSet(ctx)
	require.NoError(err)

	currentMilestoneProposer := keeper.GetMilestoneCurrentProposer(ctx)
	require.Equal(currentValSet1.GetProposer(), currentMilestoneProposer)

	keeper.MilestoneIncrementAccum(ctx, 1)

	currentValSet2, err := keeper.GetMilestoneValidatorSet(ctx)
	require.NoError(err)

	currentMilestoneProposer = keeper.GetMilestoneCurrentProposer(ctx)
	require.NotEqual(currentValSet1.GetProposer(), currentMilestoneProposer)
	require.Equal(currentValSet2.GetProposer(), currentMilestoneProposer)
}

func (s *KeeperTestSuite) TestMilestoneValidatorSetIncAccumChange() {
	ctx, keeper, require := s.ctx, s.stakeKeeper, s.Require()

	// load 4 validators to state
	testUtil.LoadRandomValidatorSet(require, 4, keeper, ctx, false, 10)

	initMilestoneValSet, err := keeper.GetMilestoneValidatorSet(ctx)
	require.NoError(err)

	initMilestoneValSetProp := initMilestoneValSet.Proposer

	initCheckpointValSet, err := keeper.GetValidatorSet(ctx)
	require.NoError(err)

	initCheckpointValSetProp := initCheckpointValSet.Proposer

	require.Equal(initMilestoneValSetProp, initCheckpointValSetProp)

	err = keeper.IncrementAccum(ctx, 1)
	require.NoError(err)

	initMilestoneValSet, err = keeper.GetMilestoneValidatorSet(ctx)
	require.NoError(err)

	initMilestoneValSetProp = initMilestoneValSet.Proposer

	initCheckpointValSet, err = keeper.GetValidatorSet(ctx)
	require.NoError(err)

	initCheckpointValSetProp = initCheckpointValSet.Proposer

	require.Equal(initMilestoneValSetProp, initCheckpointValSetProp)

	initValSet, err := keeper.GetMilestoneValidatorSet(ctx)

	keeper.MilestoneIncrementAccum(ctx, 1)

	require.NotNil(initValSet)
	initValSet.IncrementProposerPriority(1)
	_proposer := initValSet.Proposer

	currentValSet, err := keeper.GetMilestoneValidatorSet(ctx)
	require.NotNil(currentValSet)
	proposer := currentValSet.Proposer

	require.Equal(_proposer, proposer)
}

func (s *KeeperTestSuite) TestUpdateMilestoneValidatorSetChange() {
	ctx, keeper, require := s.ctx, s.stakeKeeper, s.Require()

	// load 4 validators to state
	testUtil.LoadRandomValidatorSet(require, 4, keeper, ctx, false, 10)
	initValSet, err := keeper.GetMilestoneValidatorSet(ctx)
	require.NoError(err)

	keeper.MilestoneIncrementAccum(ctx, 1)

	prevValSet := initValSet.Copy()
	currentValSet, err := keeper.GetMilestoneValidatorSet(ctx)
	require.NoError(err)

	valToUpdate := currentValSet.Validators[0]
	newSigner := testUtil.GenRandomVals(1, 0, 10, 10, false, 1)

	err = keeper.UpdateSigner(ctx, newSigner[0].Signer, newSigner[0].PubKey, valToUpdate.Signer)
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
