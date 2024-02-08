package keeper_test

import (
	"bytes"
	"math/rand"
	"testing"
	"time"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttime "github.com/cometbft/cometbft/types/time"
	"github.com/stretchr/testify/suite"

	storetypes "cosmossdk.io/store/types"

	"github.com/0xPolygon/heimdall-v2/helper"
	stakingkeeper "github.com/0xPolygon/heimdall-v2/x/staking/keeper"
	testUtil "github.com/0xPolygon/heimdall-v2/x/staking/testutil"
	"github.com/0xPolygon/heimdall-v2/x/staking/types"
	stakingtypes "github.com/0xPolygon/heimdall-v2/x/staking/types"
	hmTypes "github.com/0xPolygon/heimdall-v2/x/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/types/simulation"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

var (
	PKs = simtestutil.CreateTestPubKeys(500)
)

type KeeperTestSuite struct {
	suite.Suite

	ctx           sdk.Context
	stakingKeeper *stakingkeeper.Keeper
	queryClient   stakingtypes.QueryClient
	msgServer     stakingtypes.MsgServer
}

func (s *KeeperTestSuite) SetupTest() {
	key := storetypes.NewKVStoreKey(stakingtypes.StoreKey)
	storeService := runtime.NewKVStoreService(key)
	testCtx := testutil.DefaultContextWithDB(s.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx.WithBlockHeader(cmtproto.Header{Time: cmttime.Now()})
	encCfg := moduletestutil.MakeTestEncodingConfig()

	keeper := stakingkeeper.NewKeeper(
		encCfg.Codec,
		storeService,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		testUtil.ModuleCommunicatorMock{AckCount: uint64(0)},
		helper.ContractCaller{},
	)

	s.ctx = ctx
	s.stakingKeeper = keeper

	stakingtypes.RegisterInterfaces(encCfg.InterfaceRegistry)
	queryHelper := baseapp.NewQueryServerTestHelper(ctx, encCfg.InterfaceRegistry)
	stakingtypes.RegisterQueryServer(queryHelper, stakingkeeper.Querier{Keeper: keeper})
	s.queryClient = stakingtypes.NewQueryClient(queryHelper)
	s.msgServer = stakingkeeper.NewMsgServerImpl(keeper)
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

// tests setter/getters for validatorSignerMaps , validator set/get
func (s *KeeperTestSuite) TestValidator() {
	ctx, keeper := s.ctx, s.stakingKeeper
	require := s.Require()

	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	n := 5

	validators := make([]*hmTypes.Validator, n)
	accounts := simulation.RandomAccounts(r1, n)

	for i := range validators {
		// validator
		validators[i] = hmTypes.NewValidator(
			hmTypes.NewValidatorID(uint64(int64(i))),
			0,
			0,
			1,
			int64(simulation.RandIntBetween(r1, 10, 100)), // power
			pk1,
			hmTypes.HeimdallAddress{accounts[i].Address},
		)

		err := keeper.AddValidator(ctx, *validators[i])
		require.NoErrorf(err, "Error while adding validator to store")
	}

	// Get random validator ID
	valId := simulation.RandIntBetween(r1, 0, n)

	// Get Validator Info from state
	valInfo, err := keeper.GetValidatorInfo(ctx, validators[valId].Signer.Bytes())
	require.NoErrorf(err, "Error while fetching Validator")

	// Get Signer Address mapped with ValidatorId
	mappedSignerAddress, isMapped := keeper.GetSignerFromValidatorID(ctx, validators[0].ID)
	require.Truef(isMapped, "Signer Address not mapped to Validator Id")

	// Check if Validator matches in state
	require.Equal(valInfo, *validators[valId], "Validators in state doesn't match")
	require.Equal(hmTypes.HexToHeimdallAddress(mappedSignerAddress.Hex()), validators[0].Signer, "Signer address doesn't match")
}

// tests VotingPower change, validator creation, validator set update when signer changes
func (s *KeeperTestSuite) TestUpdateSigner() {
	ctx, keeper := s.ctx, s.stakingKeeper
	require := s.Require()

	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	n := 5

	validators := make([]*hmTypes.Validator, n)
	accounts := simulation.RandomAccounts(r1, n)

	for i := range validators {
		// validator
		validators[i] = hmTypes.NewValidator(
			hmTypes.NewValidatorID(uint64(int64(i))),
			0,
			0,
			1,
			int64(simulation.RandIntBetween(r1, 10, 100)), // power
			pk1,
			hmTypes.HeimdallAddress{accounts[i].Address},
		)

		err := keeper.AddValidator(ctx, *validators[i])
		require.NoErrorf(err, "Error while adding validator to store")

	}

	// Fetch Validator Info from Store
	valInfo, err := keeper.GetValidatorInfo(ctx, validators[0].Signer.Bytes())
	require.NoErrorf(err, "Error while fetching Validator Info from store")

	pkAny2, err := codectypes.NewAnyWithValue(pk2)
	require.NoError(err)

	addr2 := hmTypes.HeimdallAddress{valAddr2}

	err = keeper.UpdateSigner(ctx, addr2, pkAny2, valInfo.Signer)
	require.NoErrorf(err, "Error while updating Signer Address ")

	// Check Validator Info of Prev Signer
	prevSginerValInfo, err := keeper.GetValidatorInfo(ctx, validators[0].Signer.Bytes())
	require.NoErrorf(err, "Error while fetching Validator Info for Prev Signer")

	require.Equal(int64(0), prevSginerValInfo.VotingPower, "VotingPower of Prev Signer should be zero")

	// Check Validator Info of Updated Signer
	updatedSignerValInfo, err := keeper.GetValidatorInfo(ctx, addr2.GetAddress())
	require.NoError(err, "Error while fetching Validator Info for Updater Signer")

	require.Equal(validators[0].VotingPower, updatedSignerValInfo.VotingPower, "VotingPower of updated signer should match with prev signer VotingPower")

	// Check If ValidatorId is mapped To Updated Signer
	signerAddress, isMapped := keeper.GetSignerFromValidatorID(ctx, validators[0].ID)
	require.Truef(isMapped, "Signer Address not mapped to Validator Id")
	require.Equal(addr2, hmTypes.HexToHeimdallAddress(signerAddress.Hex()), "Validator ID should be mapped to Updated Signer Address")

	// Check total Validators
	totalValidators := keeper.GetAllValidators(ctx)
	require.LessOrEqual(6, len(totalValidators), "Total Validators should be six.")

	// Check current Validators
	currentValidators := keeper.GetCurrentValidators(ctx)
	require.LessOrEqual(5, len(currentValidators), "Current Validators should be five.")
}

// func (s *KeeperTestSuite) TestCurrentValidator() {
// 	type TestDataItem struct {
// 		name        string
// 		startblock  uint64
// 		VotingPower int64
// 		ackcount    uint64
// 		result      bool
// 		resultmsg   string
// 	}

// 	dataItems := []TestDataItem{
// 		{
// 			name:        "VotingPower zero",
// 			startblock:  uint64(0),
// 			VotingPower: int64(0),
// 			ackcount:    uint64(1),
// 			result:      false,
// 			resultmsg:   "should not be current validator as VotingPower is zero.",
// 		},
// 		{
// 			name:        "start epoch greater than ackcount",
// 			startblock:  uint64(3),
// 			VotingPower: int64(10),
// 			ackcount:    uint64(1),
// 			result:      false,
// 			resultmsg:   "should not be current validator as start epoch greater than ackcount.",
// 		},
// 	}
// 	ctx, keeper := s.ctx, s.stakingKeeper

// 	stakingKeeper, checkpointKeeper := s.StakingKeeper, app.CheckpointKeeper

// 	for i, item := range dataItems {
// 		suite.Run(item.name, func() {
// 			// Create a Validator [startEpoch, endEpoch, VotingPower]
// 			privKep := secp256k1.GenPrivKey()
// 			pubkey := types.NewPubKey(privKep.PubKey().Bytes())
// 			newVal := types.Validator{
// 				ID:               types.NewValidatorID(1 + uint64(i)),
// 				StartEpoch:       item.startblock,
// 				EndEpoch:         item.startblock,
// 				Nonce:            0,
// 				VotingPower:      item.VotingPower,
// 				Signer:           types.HexToHeimdallAddress(pubkey.Address().String()),
// 				PubKey:           pubkey,
// 				ProposerPriority: 0,
// 			}
// 			// check current validator
// 			err := stakingKeeper.AddValidator(ctx, newVal)
// 			require.NoError(t, err)
// 			checkpointKeeper.UpdateACKCountWithValue(ctx, item.ackcount)

// 			isCurrentVal := keeper.IsCurrentValidatorByAddress(ctx, newVal.Signer.Bytes())
// 			require.Equal(t, item.result, isCurrentVal, item.resultmsg)
// 		})
// 	}
// }

func (s *KeeperTestSuite) TestRemoveValidatorSetChange() {
	// create sub test to check if validator remove
	ctx, keeper := s.ctx, s.stakingKeeper
	require := s.Require()

	// load 4 validators to state
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

	removedVal := prevValidatorSet.Validators[0].Signer.GetAddress()

	for _, val := range updatedValSet.Validators {
		if bytes.Equal(val.Signer.GetAddress(), removedVal) {
			require.Fail("Validator is not removed from updatedvalidator set")
		}
	}
}

func (s *KeeperTestSuite) TestAddValidatorSetChange() {
	// create sub test to check if validator remove
	ctx, keeper := s.ctx, s.stakingKeeper
	require := s.Require()

	// load 4 validators to state
	testUtil.LoadValidatorSet(require, 4, keeper, ctx, false, 10)
	initValSet := keeper.GetValidatorSet(ctx)

	validators := testUtil.GenRandomVal(1, 0, 10, 10, false, 1)
	prevValSet := initValSet.Copy()

	valToBeAdded := validators[0]
	currentValSet := initValSet.Copy()

	err := keeper.AddValidator(ctx, valToBeAdded)
	require.NoError(err)

	_, err = keeper.GetValidatorInfo(ctx, valToBeAdded.GetSigner().Address)
	require.NoError(err)

	setUpdates := types.GetUpdatedValidators(currentValSet, keeper.GetAllValidators(ctx), 5)
	err = currentValSet.UpdateWithChangeSet(setUpdates)
	require.NoError(err)
	require.Equal(len(prevValSet.Validators)+1, len(currentValSet.Validators), "Number of validators should be increased by 1")
	require.Equal(true, currentValSet.HasAddress(valToBeAdded.Signer.Bytes()), "New Validator should be added")
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
	// create sub test to check if validator remove
	ctx, keeper := s.ctx, s.stakingKeeper
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

	index, _ := currentValSet.GetByAddress(valToUpdate.Signer.Bytes())
	require.Equal(-1, index, "Prev Validator should not be present in CurrentValSet")

	_, newVal := currentValSet.GetByAddress(newSigner[0].Signer.Bytes())
	require.Equal(newSigner[0].Signer, newVal.Signer, "Signer address should change")
	require.Equal(newSigner[0].PubKey, newVal.PubKey, "Signer pubkey should change")

	require.Equal(prevValSet.GetTotalVotingPower(), currentValSet.GetTotalVotingPower(), "Total VotingPower should not change")
}

func (s *KeeperTestSuite) TestGetCurrentValidators() {
	ctx, keeper := s.ctx, s.stakingKeeper
	require := s.Require()
	testUtil.LoadValidatorSet(require, 4, keeper, ctx, false, 10)
	validators := keeper.GetCurrentValidators(ctx)
	activeValidatorInfo, err := keeper.GetActiveValidatorInfo(ctx, validators[0].Signer.Bytes())
	require.NoError(err)
	require.Equal(validators[0], activeValidatorInfo)
}

func (s *KeeperTestSuite) TestGetCurrentProposer() {
	ctx, keeper := s.ctx, s.stakingKeeper
	require := s.Require()

	testUtil.LoadValidatorSet(require, 4, keeper, ctx, false, 10)
	currentValSet := keeper.GetValidatorSet(ctx)
	currentProposer := keeper.GetCurrentProposer(ctx)
	require.Equal(currentValSet.GetProposer(), currentProposer)
}

func (s *KeeperTestSuite) TestGetNextProposer() {
	ctx, keeper := s.ctx, s.stakingKeeper
	require := s.Require()
	testUtil.LoadValidatorSet(require, 4, keeper, ctx, false, 10)

	nextProposer := keeper.GetNextProposer(ctx)
	require.NotNil(nextProposer)
}

func (s *KeeperTestSuite) TestGetValidatorFromValID() {
	ctx, keeper := s.ctx, s.stakingKeeper
	require := s.Require()
	testUtil.LoadValidatorSet(require, 4, keeper, ctx, false, 10)
	validators := keeper.GetCurrentValidators(ctx)

	valInfo, ok := keeper.GetValidatorFromValID(ctx, validators[0].ID)
	require.Equal(ok, true)
	require.Equal(validators[0], valInfo)
}

func (s *KeeperTestSuite) TestGetLastUpdated() {
	ctx, keeper := s.ctx, s.stakingKeeper
	require := s.Require()
	testUtil.LoadValidatorSet(require, 4, keeper, ctx, false, 10)
	validators := keeper.GetCurrentValidators(ctx)

	lastUpdated, ok := keeper.GetLastUpdated(ctx, validators[0].ID)
	require.Equal(ok, true)
	require.Equal(validators[0].LastUpdated, lastUpdated)
}

// func (s *KeeperTestSuite) TestGetSpanEligibleValidators() {
// 	ctx, keeper := s.ctx, s.stakingKeeper
// 	require := s.Require()
// 	testUtil.LoadValidatorSet(require, 4, keeper, ctx, false, 0)

// 	// Test ActCount = 0
// 	app.CheckpointKeeper.UpdateACKCountWithValue(ctx, 0)

// 	valActCount0 := keeper.GetSpanEligibleValidators(ctx)
// 	require.LessOrEqual(len(valActCount0), 4)

// 	app.CheckpointKeeper.UpdateACKCountWithValue(ctx, 20)

// 	validators := keeper.GetSpanEligibleValidators(ctx)
// 	require.LessOrEqual(len(validators), 4)
// }

func (s *KeeperTestSuite) TestGetMilestoneProposer() {
	ctx, keeper := s.ctx, s.stakingKeeper
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
	// create sub test to check if validator remove
	ctx, keeper := s.ctx, s.stakingKeeper
	require := s.Require()

	// load 4 validators to state
	testUtil.LoadValidatorSet(require, 4, keeper, ctx, false, 10)

	initMilestoneValSetProp := keeper.GetMilestoneValidatorSet(ctx).Proposer //Getter for Milestone Validator Set Proposer
	initCheckpointValSetProp := keeper.GetValidatorSet(ctx).Proposer         //Getter for Checkpoint Validator Set Proposer

	require.Equal(initMilestoneValSetProp, initCheckpointValSetProp)

	keeper.IncrementAccum(ctx, 1)

	initMilestoneValSetProp = keeper.GetMilestoneValidatorSet(ctx).Proposer //Getter for Milestone Validator Set Proposer
	initCheckpointValSetProp = keeper.GetValidatorSet(ctx).Proposer         //Getter for Checkpoint Validator Set Proposer

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
	// create sub test to check if validator remove
	ctx, keeper := s.ctx, s.stakingKeeper
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

	index, _ := currentValSet.GetByAddress(valToUpdate.Signer.Bytes())
	require.Equal(-1, index, "Prev Validator should not be present in CurrentValSet")

	_, newVal := currentValSet.GetByAddress(newSigner[0].Signer.Bytes())
	require.Equal(newSigner[0].Signer, newVal.Signer, "Signer address should change")
	require.Equal(newSigner[0].PubKey, newVal.PubKey, "Signer pubkey should change")

	require.Equal(prevValSet.GetTotalVotingPower(), currentValSet.GetTotalVotingPower(), "Total VotingPower should not change")
}
