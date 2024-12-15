package keeper_test

import (
	"math/rand"
	"strings"
	"testing"
	"time"

	storetypes "cosmossdk.io/store/types"
	util "github.com/0xPolygon/heimdall-v2/common/address"
	"github.com/0xPolygon/heimdall-v2/helper/mocks"
	"github.com/0xPolygon/heimdall-v2/sidetxs"
	cmKeeper "github.com/0xPolygon/heimdall-v2/x/chainmanager/keeper"
	cmTypes "github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
	stakeKeeper "github.com/0xPolygon/heimdall-v2/x/stake/keeper"
	testUtil "github.com/0xPolygon/heimdall-v2/x/stake/testutil"
	"github.com/0xPolygon/heimdall-v2/x/stake/types"
	stakeTypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttime "github.com/cometbft/cometbft/types/time"
	"github.com/cosmos/cosmos-sdk/baseapp"
	addrCodec "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/types/simulation"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"
)

var (
	pk       = secp256k1.GenPrivKey().PubKey()
	pk2      = secp256k1.GenPrivKey().PubKey()
	valAddr2 = sdk.ValAddress(pk2.Address())
)

type KeeperTestSuite struct {
	suite.Suite

	ctx              sdk.Context
	stakeKeeper      *stakeKeeper.Keeper
	contractCaller   *mocks.IContractCaller
	checkpointKeeper *testUtil.MockCheckpointKeeper
	bankKeeper       *testUtil.MockBankKeeper
	cmKeeper         *cmKeeper.Keeper
	queryClient      stakeTypes.QueryClient
	msgServer        stakeTypes.MsgServer
	sideMsgCfg       sidetxs.SideTxConfigurator
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
		s.bankKeeper,
		cmk,
		addrCodec.NewHexCodec(),
		s.contractCaller,
	)

	keeper.SetCheckpointKeeper(s.checkpointKeeper)

	s.ctx = ctx
	s.cmKeeper = &cmk
	s.stakeKeeper = &keeper

	stakeTypes.RegisterInterfaces(encCfg.InterfaceRegistry)
	queryHelper := baseapp.NewQueryServerTestHelper(ctx, encCfg.InterfaceRegistry)
	stakeTypes.RegisterQueryServer(queryHelper, stakeKeeper.NewQueryServer(&keeper))
	s.queryClient = stakeTypes.NewQueryClient(queryHelper)
	s.msgServer = stakeKeeper.NewMsgServerImpl(&keeper)

	s.sideMsgCfg = sidetxs.NewSideTxConfigurator()
	types.RegisterSideMsgServer(s.sideMsgCfg, stakeKeeper.NewSideMsgServerImpl(&keeper))

}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

func (s *KeeperTestSuite) TestValidator() {
	ctx, keeper, require := s.ctx, s.stakeKeeper, s.Require()

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	n := 5

	validators := make([]*types.Validator, n)
	accounts := simulation.RandomAccounts(r, n)

	var err error

	for i := range validators {
		validators[i], err = types.NewValidator(
			uint64(i),
			0,
			0,
			1,
			int64(simulation.RandIntBetween(r, 10, 100)), // power
			pk,
			accounts[i].Address.String(),
		)

		require.NoError(err)

		err = keeper.AddValidator(ctx, *validators[i])
		require.NoErrorf(err, "Error while adding validator to store")
	}

	// get random validator ID
	valId := simulation.RandIntBetween(r, 0, n)

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
	ctx, keeper, require, checkpointKeeper := s.ctx, s.stakeKeeper, s.Require(), s.checkpointKeeper

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	n := 5

	validators := make([]*types.Validator, n)
	accounts := simulation.RandomAccounts(r, n)

	var err error
	for i := range validators {
		validators[i], err = types.NewValidator(
			uint64(int64(i)),
			0,
			0,
			1,
			int64(simulation.RandIntBetween(r, 10, 100)), // power
			pk,
			accounts[i].Address.String(),
		)

		require.NoError(err)

		err = keeper.AddValidator(ctx, *validators[i])
		require.NoErrorf(err, "Error while adding validator to store")
	}

	// fetch validator info from store
	valInfo, err := keeper.GetValidatorInfo(ctx, validators[0].Signer)
	require.NoErrorf(err, "Error while fetching Validator Info from store")

	addr2 := util.FormatAddress(valAddr2.String())

	err = keeper.UpdateSigner(ctx, addr2, pk2.Bytes(), valInfo.Signer)
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
	checkpointKeeper.EXPECT().GetAckCount(gomock.Any()).Return(uint64(0), nil).Times(1)
	currentValidators := keeper.GetCurrentValidators(ctx)
	require.LessOrEqual(5, len(currentValidators), "Current Validators should be five.")
}

func (s *KeeperTestSuite) TestCurrentValidator() {

	ctx, keeper, require, sKeeper, checkpointKeeper := s.ctx, s.stakeKeeper, s.Require(), s.stakeKeeper, s.checkpointKeeper

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	n := 5

	accounts := simulation.RandomAccounts(r, n)

	type TestDataItem struct {
		name        string
		startBlock  uint64
		VotingPower int64
		ackCount    uint64
		result      bool
		resultMsg   string
	}

	dataItems := []TestDataItem{
		{
			name:        "VotingPower zero",
			startBlock:  uint64(0),
			VotingPower: int64(0),
			ackCount:    uint64(1),
			result:      false,
			resultMsg:   "should not be current validator as VotingPower is zero.",
		},
		{
			name:        "start epoch greater than ackCount",
			startBlock:  uint64(3),
			VotingPower: int64(10),
			ackCount:    uint64(1),
			result:      false,
			resultMsg:   "should not be current validator as start epoch greater than ackCount.",
		},
	}

	for i, item := range dataItems {
		s.Run(item.name, func() {
			newVal, err := types.NewValidator(1+uint64(i), item.startBlock, item.startBlock, uint64(0), item.VotingPower, accounts[i].PubKey, accounts[i].Address.String())

			require.NoError(err)

			// check current validator
			err = sKeeper.AddValidator(ctx, *newVal)
			require.NoError(err)

			checkpointKeeper.EXPECT().GetAckCount(gomock.Any()).Return(item.ackCount, nil).Times(1)

			isCurrentVal := keeper.IsCurrentValidatorByAddress(ctx, newVal.Signer)
			require.Equal(item.result, isCurrentVal, item.resultMsg)
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

	_, err = keeper.GetValidatorInfo(ctx, util.FormatAddress(valToBeAdded.GetSigner()))
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
	ctx, keeper, require, checkpointKeeper := s.ctx, s.stakeKeeper, s.Require(), s.checkpointKeeper

	testUtil.LoadRandomValidatorSet(require, 4, keeper, ctx, false, 10)

	checkpointKeeper.EXPECT().GetAckCount(ctx).AnyTimes().Return(uint64(1), nil)

	validators := keeper.GetCurrentValidators(ctx)
	activeValidatorInfo, err := keeper.GetActiveValidatorInfo(ctx, util.FormatAddress(validators[0].Signer))
	require.NoError(err)
	require.Equal(validators[0], activeValidatorInfo)
}

func (s *KeeperTestSuite) TestGetPreviousBlockValidatorSet() {
	ctx, keeper, require := s.ctx, s.stakeKeeper, s.Require()

	// Load 4 validators into state
	testUtil.LoadRandomValidatorSet(require, 4, keeper, ctx, false, 10)

	// Get the current validator set
	currentValSet, err := keeper.GetValidatorSet(ctx)
	require.NoError(err)

	// Update the previous block validator set in store
	err = keeper.UpdatePreviousBlockValidatorSetInStore(ctx, currentValSet)
	require.NoError(err)

	// Retrieve the previous block validator set
	prevValSet, err := keeper.GetPreviousBlockValidatorSet(ctx)
	require.NoError(err)

	// Check if the previous block validator set matches the current validator set
	require.Equal(currentValSet, prevValSet, "Previous block validator set should match the current validator set")

	// Call IncrementAccum, which affects the current validator set but not the previous one
	err = keeper.IncrementAccum(ctx, 1)
	require.NoError(err)

	// Get the updated current validator set
	updatedValSet, err := keeper.GetValidatorSet(ctx)
	require.NoError(err)

	// Retrieve the previous block validator set again
	prevValSetAfterIncrement, err := keeper.GetPreviousBlockValidatorSet(ctx)
	require.NoError(err)

	// Check if the previous block validator set has not changed
	require.Equal(prevValSet, prevValSetAfterIncrement, "Previous block validator set should not change after IncrementAccum")

	// Check if the current validator set has changed
	require.NotEqual(currentValSet, updatedValSet, "Current validator set should change after IncrementAccum")
}

func (s *KeeperTestSuite) TestUpdatePreviousBlockValidatorSetInStore() {
	ctx, keeper, require := s.ctx, s.stakeKeeper, s.Require()

	// Load 4 validators into state
	testUtil.LoadRandomValidatorSet(require, 4, keeper, ctx, false, 10)

	// Get the current validator set
	currentValSet, err := keeper.GetValidatorSet(ctx)
	require.NoError(err)

	// Update the previous block validator set in store
	err = keeper.UpdatePreviousBlockValidatorSetInStore(ctx, currentValSet)
	require.NoError(err)

	// Retrieve the previous block validator set
	prevValSet, err := keeper.GetPreviousBlockValidatorSet(ctx)
	require.NoError(err)

	// Check if the previous block validator set matches the current validator set
	require.Equal(currentValSet, prevValSet, "Previous block validator set should match the current validator set")

	// Modify the current validator set
	currentValSet.Validators[0].VotingPower += 10

	// Update the previous block validator set in store again
	err = keeper.UpdatePreviousBlockValidatorSetInStore(ctx, currentValSet)
	require.NoError(err)

	// Retrieve the updated previous block validator set
	updatedPrevValSet, err := keeper.GetPreviousBlockValidatorSet(ctx)
	require.NoError(err)

	// Check if the updated previous block validator set matches the modified current validator set
	require.Equal(currentValSet, updatedPrevValSet, "Updated previous block validator set should match the modified current validator set")

	// Call IncrementAccum which affects the current validator set but not the previous one
	err = keeper.IncrementAccum(ctx, 1)
	require.NoError(err)

	// Get the updated current validator set
	updatedValSet, err := keeper.GetValidatorSet(ctx)
	require.NoError(err)

	// Retrieve the previous block validator set again
	prevValSetAfterIncrement, err := keeper.GetPreviousBlockValidatorSet(ctx)
	require.NoError(err)

	// Check if the previous block validator set has not changed
	require.Equal(updatedPrevValSet, prevValSetAfterIncrement, "Previous block validator set should not change after IncrementAccum")

	// Check if the current validator set has changed
	require.NotEqual(currentValSet, updatedValSet, "Current validator set should change after IncrementAccum")
}

func (s *KeeperTestSuite) TestSetAndGetLastBlockTxs() {
	ctx, keeper, require := s.ctx, s.stakeKeeper, s.Require()

	// first height
	firstHeightTxs := [][]byte{
		[]byte("tx1"),
		[]byte("tx2"),
	}

	// check that GetLastBlockTxs returns an error before setting them
	_, err := keeper.GetLastBlockTxs(ctx)
	require.Error(err, "Getting last block txs should produce an error")

	// set and get firstHeightTxs for first height
	err = keeper.SetLastBlockTxs(ctx, firstHeightTxs)
	require.NoError(err, "Setting last block txs should not produce an error")
	retrievedFirstBlockTxs, err := keeper.GetLastBlockTxs(ctx)
	require.NoError(err, "Getting last block txs should not produce an error")
	require.Equal(firstHeightTxs, retrievedFirstBlockTxs.Txs, "Retrieved txs should match the set txs")

	// second height
	secondHeightTxs := [][]byte{
		[]byte("tx3"),
		[]byte("tx4"),
	}

	// set and get txs for second height
	err = keeper.SetLastBlockTxs(ctx, secondHeightTxs)
	require.NoError(err, "Setting last block txs should not produce an error")
	retrievedSecondBlockTxs, err := keeper.GetLastBlockTxs(ctx)
	require.NoError(err, "Getting last block txs should not produce an error")
	require.Equal(secondHeightTxs, retrievedSecondBlockTxs.Txs, "Retrieved txs should match the set tx for the second height")

	// ensure fetching the txs again returns the correct txs
	retrievedSecondBlockTxs, err = keeper.GetLastBlockTxs(ctx)
	require.NoError(err, "Getting last block txs should not produce an error")
	require.NotEqual(firstHeightTxs, retrievedSecondBlockTxs.Txs, "Retrieved txs for the first height should not match the latest set txs")
	require.Equal(secondHeightTxs, retrievedSecondBlockTxs.Txs, "Retrieved txs for the second height should match the latest set txs")
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
	ctx, keeper, require, checkpointKeeper := s.ctx, s.stakeKeeper, s.Require(), s.checkpointKeeper

	testUtil.LoadRandomValidatorSet(require, 4, keeper, ctx, false, 10)
	checkpointKeeper.EXPECT().GetAckCount(ctx).AnyTimes().Return(uint64(1), nil)

	validators := keeper.GetCurrentValidators(ctx)

	valInfo, err := keeper.GetValidatorFromValID(ctx, validators[0].ValId)
	require.NoError(err)
	require.Equal(validators[0], valInfo)
}

func (s *KeeperTestSuite) TestGetLastUpdated() {
	ctx, keeper, require, checkpointKeeper := s.ctx, s.stakeKeeper, s.Require(), s.checkpointKeeper

	testUtil.LoadRandomValidatorSet(require, 1, keeper, ctx, false, 10)
	checkpointKeeper.EXPECT().GetAckCount(ctx).AnyTimes().Return(uint64(1), nil)

	validators := keeper.GetCurrentValidators(ctx)

	lastUpdated, err := keeper.GetLastUpdated(ctx, validators[0].ValId)
	require.NoError(err)
	require.Equal(validators[0].LastUpdated, lastUpdated)
}

func (s *KeeperTestSuite) TestGetSpanEligibleValidators() {
	ctx, keeper, require, checkpointKeeper := s.ctx, s.stakeKeeper, s.Require(), s.checkpointKeeper

	testUtil.LoadRandomValidatorSet(require, 4, keeper, ctx, false, 0)

	// Test ActCount = 0
	checkpointKeeper.EXPECT().GetAckCount(gomock.Any()).Return(uint64(0), nil).Times(1)

	valActCount0 := keeper.GetSpanEligibleValidators(ctx)
	require.LessOrEqual(len(valActCount0), 4)

	checkpointKeeper.EXPECT().GetAckCount(gomock.Any()).Return(uint64(0), nil).Times(20)

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
	require.NotNil(initValSet)
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
