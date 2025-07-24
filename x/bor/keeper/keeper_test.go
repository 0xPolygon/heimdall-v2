package keeper_test

import (
	"errors"
	"math/big"
	"strconv"
	"testing"

	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttime "github.com/cometbft/cometbft/types/time"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/helper/mocks"
	"github.com/0xPolygon/heimdall-v2/sidetxs"
	"github.com/0xPolygon/heimdall-v2/x/bor/keeper"
	bortestutil "github.com/0xPolygon/heimdall-v2/x/bor/testutil"
	"github.com/0xPolygon/heimdall-v2/x/bor/types"
	staketypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

type KeeperTestSuite struct {
	suite.Suite

	ctx                sdk.Context
	borKeeper          keeper.Keeper
	chainManagerKeeper *bortestutil.MockChainManagerKeeper
	stakeKeeper        *bortestutil.MockStakeKeeper
	milestoneKeeper    *bortestutil.MockMilestoneKeeper
	contractCaller     mocks.IContractCaller
	queryClient        types.QueryClient
	msgServer          types.MsgServer
	sideMsgServer      sidetxs.SideMsgServer
	encCfg             moduletestutil.TestEncodingConfig
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

func (s *KeeperTestSuite) SetupTest() {
	helper.SetVeblopHeight(100000)
	key := storetypes.NewKVStoreKey(types.StoreKey)
	testCtx := testutil.DefaultContextWithDB(s.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx.WithBlockHeader(cmtproto.Header{Time: cmttime.Now()})
	encCfg := moduletestutil.MakeTestEncodingConfig()

	storeService := runtime.NewKVStoreService(key)

	ctrl := gomock.NewController(s.T())
	chainManagerKeeper := bortestutil.NewMockChainManagerKeeper(ctrl)
	s.chainManagerKeeper = chainManagerKeeper

	stakeKeeper := bortestutil.NewMockStakeKeeper(ctrl)
	s.stakeKeeper = stakeKeeper

	milestoneKeeper := bortestutil.NewMockMilestoneKeeper(ctrl)
	s.milestoneKeeper = milestoneKeeper

	s.contractCaller = mocks.IContractCaller{}
	s.ctx = ctx

	s.borKeeper = keeper.NewKeeper(
		encCfg.Codec,
		storeService,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		s.chainManagerKeeper,
		s.stakeKeeper,
		s.milestoneKeeper,
		nil,
	)

	s.borKeeper.SetContractCaller(&s.contractCaller)
	types.RegisterInterfaces(encCfg.InterfaceRegistry)

	borParams := types.DefaultParams()
	err := s.borKeeper.SetParams(ctx, borParams)
	s.Require().NoError(err)

	queryHelper := baseapp.NewQueryServerTestHelper(ctx, encCfg.InterfaceRegistry)
	types.RegisterQueryServer(queryHelper, keeper.NewQueryServer(&s.borKeeper))
	queryClient := types.NewQueryClient(queryHelper)

	s.queryClient = queryClient
	s.msgServer = keeper.NewMsgServerImpl(s.borKeeper)
	s.encCfg = encCfg

	msgServer := keeper.NewMsgServerImpl(s.borKeeper)
	s.msgServer = msgServer

	sideMsgServer := keeper.NewSideMsgServerImpl(&s.borKeeper)
	s.sideMsgServer = sideMsgServer
}

func (s *KeeperTestSuite) TestAddNewSpan() {
	require, ctx, borKeeper := s.Require(), s.ctx, s.borKeeper
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	spans := s.genTestSpans(2)

	testcases := []struct {
		name string
		span types.Span
	}{
		{
			name: "First Span",
			span: *spans[0],
		},
		{
			name: "Second Span",
			span: *spans[1],
		},
	}

	for _, tc := range testcases {
		s.T().Run(tc.name, func(t *testing.T) {
			err := borKeeper.AddNewSpan(ctx, &tc.span)
			require.NoError(err)

			hasSpan, err := borKeeper.HasSpan(ctx, tc.span.Id)
			require.NoError(err)
			require.True(hasSpan)

			span, err := borKeeper.GetSpan(ctx, tc.span.Id)
			require.NoError(err)
			require.Equal(tc.span, span)

			lastSpan, err := borKeeper.GetLastSpan(ctx)
			require.NoError(err)
			require.Equal(tc.span, lastSpan)

			// Ensure we've captured correct span insertion event
			events := sdkCtx.EventManager().ABCIEvents()
			event := events[len(events)-1]
			require.Equal(event.Type, types.EventTypeSpan)
			require.Equal(strconv.FormatUint(tc.span.Id, 10), event.Attributes[0].Value)
			require.Equal(strconv.FormatUint(tc.span.StartBlock, 10), event.Attributes[1].Value)
			require.Equal(strconv.FormatUint(tc.span.EndBlock, 10), event.Attributes[2].Value)
			require.Equal(tc.span.SelectedProducers[0].Signer, event.Attributes[3].Value)
		})
	}
}

func (s *KeeperTestSuite) TestAddNewRawSpan() {
	require, ctx, borKeeper := s.Require(), s.ctx, s.borKeeper
	spans := s.genTestSpans(2)

	testcases := []struct {
		name string
		span types.Span
	}{
		{
			name: "First Span",
			span: *spans[0],
		},
		{
			name: "Second Span",
			span: *spans[1],
		},
	}

	for _, tc := range testcases {
		s.T().Run(tc.name, func(t *testing.T) {
			err := borKeeper.AddNewRawSpan(ctx, &tc.span)
			require.NoError(err)

			hasSpan, err := borKeeper.HasSpan(ctx, tc.span.Id)
			require.NoError(err)
			require.True(hasSpan)

			span, err := borKeeper.GetSpan(ctx, tc.span.Id)
			require.NoError(err)
			require.Equal(tc.span, span)

			lastSpan, err := borKeeper.GetLastSpan(ctx)
			if tc.span.Id == spans[0].Id {
				require.Error(err)
				require.Empty(lastSpan)
			} else {
				require.NoError(err)
				require.NotEqual(tc.span, lastSpan)
				require.Equal(spans[0], &lastSpan)
			}

			if tc.span.Id == spans[0].Id {
				err = borKeeper.AddNewSpan(ctx, &tc.span)
				require.NoError(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestGetAllSpans() {
	require, ctx, borKeeper := s.Require(), s.ctx, s.borKeeper
	spans := s.genTestSpans(2)

	for _, span := range spans {
		err := borKeeper.AddNewSpan(ctx, span)
		require.NoError(err)
	}

	resSpans, err := borKeeper.GetAllSpans(ctx)
	require.NoError(err)
	require.Equal(spans, resSpans)
}

func (s *KeeperTestSuite) TestFreezeSet() {
	require, stakeKeeper, borKeeper, ctx := s.Require(), s.stakeKeeper, s.borKeeper, s.ctx

	valSet, vals := s.genTestValidators()

	// set genesis span
	err := borKeeper.AddNewSpan(ctx, &types.Span{
		Id: 0,
	})
	require.NoError(err)

	testcases := []struct {
		name            string
		producerCount   uint64
		id              uint64
		startBlock      uint64
		endBlock        uint64
		seed            common.Hash
		expValSet       staketypes.ValidatorSet
		expLastEthBlock *big.Int
	}{
		{
			name:            "Producer count is less than total validators",
			producerCount:   3,
			id:              1,
			startBlock:      1,
			endBlock:        100,
			seed:            common.HexToHash("testSeed1"),
			expValSet:       valSet,
			expLastEthBlock: big.NewInt(0),
		},
		{
			name:            "Producer count is equal to total validators",
			producerCount:   5,
			id:              2,
			startBlock:      101,
			endBlock:        200,
			seed:            common.HexToHash("testSeed2"),
			expValSet:       valSet,
			expLastEthBlock: big.NewInt(1),
		},
	}

	for _, tc := range testcases {
		stakeKeeper.EXPECT().GetSpanEligibleValidators(ctx).Return(vals).Times(1)
		stakeKeeper.EXPECT().GetValidatorSet(ctx).Return(valSet, nil).Times(1)
		stakeKeeper.EXPECT().GetValidatorFromValID(ctx, gomock.Any()).DoAndReturn(func(ctx sdk.Context, valID uint64) (staketypes.Validator, error) {
			for _, v := range vals {
				if v.ValId == valID {
					return v, nil
				}
			}
			return staketypes.Validator{}, errors.New("validator not found")
		}).AnyTimes()

		s.T().Run(tc.name, func(t *testing.T) {
			params := types.DefaultParams()
			params.ProducerCount = tc.producerCount
			err := borKeeper.SetParams(ctx, params)
			require.NoError(err)
			err = borKeeper.AddNewSpan(ctx, &types.Span{
				Id: tc.id,
			})
			require.NoError(err)
			err = borKeeper.FreezeSet(ctx, tc.id, tc.startBlock, tc.endBlock, "test-chain", tc.seed)
			require.NoError(err)

			span, err := borKeeper.GetSpan(ctx, tc.id)
			require.NoError(err)
			require.Equal(tc.expValSet, span.ValidatorSet)
		})
	}
}

func (s *KeeperTestSuite) TestUpdateLastSpan() {
	require, ctx, borKeeper := s.Require(), s.ctx, s.borKeeper

	spans := s.genTestSpans(2)

	for _, span := range spans {
		err := borKeeper.AddNewSpan(ctx, span)
		require.NoError(err)
	}

	testcases := []struct {
		name            string
		expPrevLastSpan *types.Span
		expNewLastSpan  *types.Span
	}{
		{
			name:            "Update last span",
			expPrevLastSpan: spans[1],
			expNewLastSpan:  spans[0],
		},
	}

	for _, tc := range testcases {
		s.T().Run(tc.name, func(t *testing.T) {
			resLastSpan, err := borKeeper.GetLastSpan(ctx)
			require.NoError(err)
			require.Equal(tc.expPrevLastSpan, &resLastSpan)

			err = borKeeper.UpdateLastSpan(ctx, tc.expNewLastSpan.Id)
			require.NoError(err)

			resLastSpan, err = borKeeper.GetLastSpan(ctx)
			require.NoError(err)
			require.Equal(tc.expNewLastSpan, &resLastSpan)
		})
	}
}

func (s *KeeperTestSuite) TestFetchNextSpanSeed() {
	require, ctx, borKeeper := s.Require(), s.ctx, s.borKeeper

	borParams := types.DefaultParams()
	err := borKeeper.SetParams(ctx, borParams)
	require.NoError(err)

	valSet, vals := s.genTestValidators()

	spans := []types.Span{
		{
			Id:                0,
			StartBlock:        0,
			EndBlock:          256,
			ValidatorSet:      valSet,
			SelectedProducers: vals,
			BorChainId:        "test-chain",
		},
		{
			Id:                1,
			StartBlock:        257,
			EndBlock:          6656,
			ValidatorSet:      valSet,
			SelectedProducers: vals,
			BorChainId:        "test-chain",
		},
		{
			Id:                2,
			StartBlock:        6657,
			EndBlock:          16656,
			ValidatorSet:      valSet,
			SelectedProducers: vals,
			BorChainId:        "test-chain",
		},
		{
			Id:                3,
			StartBlock:        16657,
			EndBlock:          26656,
			ValidatorSet:      valSet,
			SelectedProducers: vals,
			BorChainId:        "test-chain",
		},
	}

	for _, span := range spans {
		err := borKeeper.AddNewSpan(ctx, &span)
		require.NoError(err)
	}

	val1Addr := common.HexToAddress(vals[0].GetOperator())
	val2Addr := common.HexToAddress(vals[1].GetOperator())
	val3Addr := common.HexToAddress(vals[2].GetOperator())

	seedBlock1 := spans[0].EndBlock
	s.contractCaller.On("GetBorChainBlockAuthor", big.NewInt(int64(seedBlock1))).Return(&val2Addr, nil)

	seedBlock2 := spans[1].EndBlock - borParams.SprintDuration
	s.contractCaller.On("GetBorChainBlockAuthor", big.NewInt(int64(spans[1].EndBlock))).Return(&val2Addr, nil)
	s.contractCaller.On("GetBorChainBlockAuthor", big.NewInt(int64(seedBlock2))).Return(&val1Addr, nil)
	for block := spans[1].EndBlock - (2 * borParams.SprintDuration); block >= spans[1].StartBlock; block -= borParams.SprintDuration {
		s.contractCaller.On("GetBorChainBlockAuthor", big.NewInt(int64(block))).Return(&val1Addr, nil)
	}

	seedBlock3 := spans[2].EndBlock - (2 * borParams.SprintDuration)
	s.contractCaller.On("GetBorChainBlockAuthor", big.NewInt(int64(spans[2].EndBlock))).Return(&val1Addr, nil)
	s.contractCaller.On("GetBorChainBlockAuthor", big.NewInt(int64(spans[2].EndBlock-borParams.SprintDuration))).Return(&val2Addr, nil)
	s.contractCaller.On("GetBorChainBlockAuthor", big.NewInt(int64(seedBlock3))).Return(&val3Addr, nil)

	seedBlock4 := spans[3].EndBlock - borParams.SprintDuration
	s.contractCaller.On("GetBorChainBlockAuthor", big.NewInt(int64(spans[3].EndBlock))).Return(&val1Addr, nil)

	for block := spans[3].EndBlock; block >= spans[3].StartBlock; block -= borParams.SprintDuration {
		s.contractCaller.On("GetBorChainBlockAuthor", big.NewInt(int64(block))).Return(&val2Addr, nil)
	}

	blockHeader1 := ethTypes.Header{Number: big.NewInt(int64(seedBlock1))}
	blockHash1 := blockHeader1.Hash()
	blockHeader2 := ethTypes.Header{Number: big.NewInt(int64(seedBlock2))}
	blockHash2 := blockHeader2.Hash()
	blockHeader3 := ethTypes.Header{Number: big.NewInt(int64(seedBlock3))}
	blockHash3 := blockHeader3.Hash()
	blockHeader4 := ethTypes.Header{Number: big.NewInt(int64(seedBlock4))}
	blockHash4 := blockHeader4.Hash()

	s.contractCaller.On("GetBorChainBlock", mock.Anything, big.NewInt(int64(seedBlock1))).Return(&blockHeader1, nil)
	s.contractCaller.On("GetBorChainBlock", mock.Anything, big.NewInt(int64(seedBlock2))).Return(&blockHeader2, nil)
	s.contractCaller.On("GetBorChainBlock", mock.Anything, big.NewInt(int64(seedBlock3))).Return(&blockHeader3, nil)
	s.contractCaller.On("GetBorChainBlock", mock.Anything, big.NewInt(int64(seedBlock4))).Return(&blockHeader4, nil)

	testcases := []struct {
		name             string
		lastSpanId       uint64
		lastSeedProducer *common.Address
		expSeed          common.Hash
		expAuthor        *common.Address
	}{
		{
			name:             "Last seed producer is different than end block author",
			lastSeedProducer: &val2Addr,
			lastSpanId:       0,
			expSeed:          blockHash1,
			expAuthor:        &val2Addr,
		},
		{
			name:             "Last seed producer is same as end block author",
			lastSeedProducer: &val1Addr,
			lastSpanId:       1,
			expSeed:          blockHash2,
			expAuthor:        &val1Addr,
		},
		{
			name:             "Next seed producer should be different from previous recent seed producers",
			lastSeedProducer: &val2Addr,
			lastSpanId:       2,
			expSeed:          blockHash3,
			expAuthor:        &val3Addr,
		},
		{
			name:             "If no unique seed producer is found, first block with different author from previous seed producer is selected",
			lastSeedProducer: &val1Addr,
			lastSpanId:       3,
			expSeed:          blockHash4,
			expAuthor:        &val2Addr,
		},
	}

	lastSpanID := uint64(0)

	for _, tc := range testcases {
		err := borKeeper.StoreSeedProducer(ctx, tc.lastSpanId, tc.lastSeedProducer)
		require.NoError(err)

		lastSpanID = tc.lastSpanId
	}

	v1A := val1Addr

	err = borKeeper.StoreSeedProducer(ctx, lastSpanID+1, &v1A)
	require.NoError(err)

	for _, tc := range testcases {
		s.T().Run(tc.name, func(t *testing.T) {
			seed, seedAuthor, err := borKeeper.FetchNextSpanSeed(ctx, tc.lastSpanId+2)
			require.NoError(err)
			require.Equal(tc.expSeed.Bytes(), seed.Bytes())
			require.Equal(*tc.expAuthor, seedAuthor)
		})
	}
}

func (s *KeeperTestSuite) TestProposeSpanOne() {
	require, ctx, borKeeper, contractCaller := s.Require(), s.ctx, s.borKeeper, &s.contractCaller

	valSet, vals := s.genTestValidators()

	err := borKeeper.AddNewSpan(ctx, &types.Span{
		Id:                0,
		StartBlock:        0,
		EndBlock:          256,
		ValidatorSet:      valSet,
		SelectedProducers: vals,
		BorChainId:        "test-chain",
	})
	require.NoError(err)

	val1Addr := common.HexToAddress(vals[0].GetOperator())

	seedBlock1 := int64(1)
	contractCaller.On("GetBorChainBlockAuthor", big.NewInt(seedBlock1)).Return(&val1Addr, nil)

	blockHeader1 := ethTypes.Header{Number: big.NewInt(seedBlock1)}
	blockHash1 := blockHeader1.Hash()
	contractCaller.On("GetBorChainBlock", mock.Anything, big.NewInt(seedBlock1)).Return(&blockHeader1, nil)

	seed, seedAuthor, err := borKeeper.FetchNextSpanSeed(ctx, 1)
	s.Require().NoError(err)
	s.Require().Equal(blockHash1.Bytes(), seed.Bytes())
	s.Require().Equal(val1Addr.Bytes(), seedAuthor.Bytes())
}

func (s *KeeperTestSuite) TestGetSeedProducer() {
	borKeeper := s.borKeeper

	producer := common.HexToAddress("0xc0ffee254729296a45a3885639AC7E10F9d54979")
	err := borKeeper.StoreSeedProducer(s.ctx, 1, &producer)
	s.Require().NoError(err)

	author, err := borKeeper.GetSeedProducer(s.ctx, 1)
	s.Require().NoError(err)
	s.Require().Equal(producer.Bytes(), author.Bytes())
}

func (s *KeeperTestSuite) TestRollbackVotingPowers() {
	testcases := []struct {
		name    string
		valsOld []staketypes.Validator
		valsNew []staketypes.Validator
		expRes  []staketypes.Validator
	}{
		{
			name:    "Revert to old voting powers",
			valsOld: []staketypes.Validator{{ValId: 1, VotingPower: 100}, {ValId: 2, VotingPower: 200}, {ValId: 3, VotingPower: 300}},
			valsNew: []staketypes.Validator{{ValId: 1, VotingPower: 200}, {ValId: 2, VotingPower: 400}, {ValId: 3, VotingPower: 200}},
			expRes:  []staketypes.Validator{{ValId: 1, VotingPower: 100}, {ValId: 2, VotingPower: 200}, {ValId: 3, VotingPower: 300}},
		},
		{
			name:    "Validator joined",
			valsOld: []staketypes.Validator{{ValId: 1, VotingPower: 100}, {ValId: 2, VotingPower: 200}},
			valsNew: []staketypes.Validator{{ValId: 1, VotingPower: 100}, {ValId: 2, VotingPower: 200}, {ValId: 3, VotingPower: 300}},
			expRes:  []staketypes.Validator{{ValId: 1, VotingPower: 100}, {ValId: 2, VotingPower: 200}, {ValId: 3, VotingPower: 0}},
		},
		{
			name:    "Validator left",
			valsOld: []staketypes.Validator{{ValId: 1, VotingPower: 100}, {ValId: 2, VotingPower: 200}, {ValId: 3, VotingPower: 300}},
			valsNew: []staketypes.Validator{{ValId: 1, VotingPower: 100}, {ValId: 2, VotingPower: 200}},
			expRes:  []staketypes.Validator{{ValId: 1, VotingPower: 100}, {ValId: 2, VotingPower: 200}},
		},
	}

	for _, tc := range testcases {
		s.T().Run(tc.name, func(t *testing.T) {
			res := keeper.RollbackVotingPowers(tc.valsNew, tc.valsOld)
			s.Require().Equal(tc.expRes, res)
		})
	}
}

func (s *KeeperTestSuite) TestParamsGetterSetter() {
	require, ctx, borKeeper := s.Require(), s.ctx, s.borKeeper

	expParams := types.DefaultParams()
	expParams.ProducerCount = 66
	expParams.SpanDuration = 100
	expParams.SprintDuration = 16
	require.NoError(borKeeper.SetParams(ctx, expParams))
	resParams, err := borKeeper.FetchParams(ctx)
	require.NoError(err)
	require.True(expParams.Equal(resParams))
}

func (s *KeeperTestSuite) TestSpanByBlockNumber() {
	require, ctx, borKeeper := s.Require(), s.ctx, s.borKeeper

	// Spans with overlaps
	spans := []types.Span{
		{Id: 0, StartBlock: 0, EndBlock: 0},
		{Id: 1, StartBlock: 1, EndBlock: 100},
		{Id: 2, StartBlock: 101, EndBlock: 200},
		{Id: 3, StartBlock: 50, EndBlock: 200},
		{Id: 4, StartBlock: 51, EndBlock: 200},
		{Id: 5, StartBlock: 200, EndBlock: 300},
		{Id: 6, StartBlock: 52, EndBlock: 200},
	}

	for i := range spans {
		err := borKeeper.AddNewSpan(ctx, &spans[i])
		require.NoError(err)
	}

	testCases := []struct {
		name           string
		blockNumber    uint64
		expectedSpanID uint64
		expectError    bool
		errorContains  string
	}{
		{
			name:           "Block in multiple overlapping spans (1)",
			blockNumber:    51,
			expectedSpanID: 4,
		},
		{
			name:           "Block in multiple overlapping spans (2)",
			blockNumber:    52,
			expectedSpanID: 6,
		},
		{
			name:           "Block in multiple overlapping spans (3)",
			blockNumber:    75,
			expectedSpanID: 6,
		},
		{
			name:           "Block in single span",
			blockNumber:    25,
			expectedSpanID: 1,
		},
		{
			name:           "Block on boundary of multiple spans",
			blockNumber:    200,
			expectedSpanID: 6,
		},
		{
			name:           "Edge case - start of first span",
			blockNumber:    1,
			expectedSpanID: 1,
		},
		{
			name:           "Edge case - end of a span",
			blockNumber:    100,
			expectedSpanID: 6, // In spans 1, 3, 4, 6. Highest ID is 6.
		},
		{
			name:           "Edge case - end of last span",
			blockNumber:    300,
			expectedSpanID: 5,
		},
		{
			name:          "Block not found - after all spans",
			blockNumber:   301,
			expectError:   true,
			errorContains: "span not found for block 301",
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			span, err := borKeeper.SpanByBlockNumber(ctx, tc.blockNumber)

			if tc.expectError {
				require.Error(err)
				require.Contains(err.Error(), tc.errorContains)
			} else {
				require.NoError(err)
				require.Equal(tc.expectedSpanID, span.Id)
			}
		})
	}
}

func (s *KeeperTestSuite) genTestSpans(num uint64) []*types.Span {
	s.T().Helper()
	valSet, vals := s.genTestValidators()

	spans := make([]*types.Span, 0, num)
	startBlock, endBlock := uint64(0), uint64(0)

	for i := uint64(0); i < num; i++ {
		startBlock = endBlock + 1
		endBlock = startBlock + 100
		span := types.Span{
			Id:                i + 1,
			StartBlock:        startBlock,
			EndBlock:          endBlock,
			ValidatorSet:      valSet,
			SelectedProducers: vals,
			BorChainId:        "test-chain",
		}
		spans = append(spans, &span)
	}

	return spans
}

func (s *KeeperTestSuite) genTestValidators() (staketypes.ValidatorSet, []staketypes.Validator) {
	s.T().Helper()

	validators := make([]*staketypes.Validator, 0, len(keeper.TestValidators))
	for _, v := range keeper.TestValidators {
		validators = append(validators, &v)
	}
	valSet := staketypes.ValidatorSet{
		Validators: validators,
	}

	vals := make([]staketypes.Validator, 0, len(validators))
	for _, v := range validators {
		vals = append(vals, *v)
	}

	return valSet, vals
}
