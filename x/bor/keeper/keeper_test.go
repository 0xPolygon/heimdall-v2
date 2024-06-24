package keeper_test

import (
	"encoding/json"
	"math/big"
	"testing"

	storetypes "cosmossdk.io/store/types"
	"github.com/0xPolygon/heimdall-v2/x/bor/keeper"
	bortestutil "github.com/0xPolygon/heimdall-v2/x/bor/testutil"
	"github.com/0xPolygon/heimdall-v2/x/bor/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttime "github.com/cometbft/cometbft/types/time"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/ethereum/go-ethereum/common"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"
)

type KeeperTestSuite struct {
	suite.Suite

	ctx                sdk.Context
	borKeeper          keeper.Keeper
	chainManagerKeeper *bortestutil.MockChainManagerKeeper
	stakeKeeper        *bortestutil.MockStakeKeeper
	// TODO HV2: blocked by contract caller
	// contractCaller     *bortestutil.ContractCaller
	queryClient   types.QueryClient
	msgServer     types.MsgServer
	sideMsgServer types.SideMsgServer
	encCfg        moduletestutil.TestEncodingConfig
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

func (suite *KeeperTestSuite) SetupTest() {
	key := storetypes.NewKVStoreKey(types.StoreKey)
	testCtx := testutil.DefaultContextWithDB(suite.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx.WithBlockHeader(cmtproto.Header{Time: cmttime.Now()})
	encCfg := moduletestutil.MakeTestEncodingConfig()

	storeService := runtime.NewKVStoreService(key)

	// gomock initializations
	ctrl := gomock.NewController(suite.T())
	chainManagerKeeper := bortestutil.NewMockChainManagerKeeper(ctrl)
	suite.chainManagerKeeper = chainManagerKeeper

	stakeKeeper := bortestutil.NewMockStakeKeeper(ctrl)
	suite.stakeKeeper = stakeKeeper

	// TODO HV2: blocked by contract caller
	// contractCaller := bortestutil.NewMockContractCaller(ctrl)
	// suite.contractCaller = contractCaller

	suite.ctx = ctx

	suite.borKeeper = keeper.NewKeeper(
		encCfg.Codec,
		storeService,
		suite.chainManagerKeeper,
		suite.stakeKeeper,
		// TODO HV2: blocked by contract caller
		// suite.contractCaller,
	)

	types.RegisterInterfaces(encCfg.InterfaceRegistry)

	queryHelper := baseapp.NewQueryServerTestHelper(ctx, encCfg.InterfaceRegistry)
	types.RegisterQueryServer(queryHelper, keeper.QueryServer{Keeper: suite.borKeeper})
	queryClient := types.NewQueryClient(queryHelper)

	suite.queryClient = queryClient
	suite.msgServer = keeper.NewMsgServerImpl(suite.borKeeper)
	suite.encCfg = encCfg

	msgServer := keeper.NewMsgServerImpl(suite.borKeeper)
	suite.msgServer = msgServer

	sideMsgServer := keeper.NewSideMsgServerImpl(&suite.borKeeper)
	suite.sideMsgServer = sideMsgServer

}

func (suite *KeeperTestSuite) TestAddNewSpan() {
	require := suite.Require()
	spans := suite.genTestSpans(2)

	testcases := []struct {
		name string
		span *types.Span
	}{
		{
			name: "First Span",
			span: spans[0],
		},
		{
			name: "Second Span",
			span: spans[1],
		},
	}

	for _, tc := range testcases {
		suite.T().Run(tc.name, func(t *testing.T) {

			err := suite.borKeeper.AddNewSpan(suite.ctx, tc.span)
			require.NoError(err)

			hasSpan, err := suite.borKeeper.HasSpan(suite.ctx, tc.span.Id)
			require.NoError(err)
			require.True(hasSpan)

			span, err := suite.borKeeper.GetSpan(suite.ctx, tc.span.Id)
			require.NoError(err)
			require.Equal(tc.span, span)

			lastSpan, err := suite.borKeeper.GetLastSpan(suite.ctx)
			require.NoError(err)
			require.Equal(tc.span, lastSpan)
		})
	}
}

func (suite *KeeperTestSuite) TestAddNewRawSpan() {
	require := suite.Require()
	spans := suite.genTestSpans(2)

	testcases := []struct {
		name string
		span *types.Span
	}{
		{
			name: "First Span",
			span: spans[0],
		},
		{
			name: "Second Span",
			span: spans[1],
		},
	}

	for _, tc := range testcases {
		suite.T().Run(tc.name, func(t *testing.T) {

			err := suite.borKeeper.AddNewRawSpan(suite.ctx, tc.span)
			require.NoError(err)

			hasSpan, err := suite.borKeeper.HasSpan(suite.ctx, tc.span.Id)
			require.NoError(err)
			require.True(hasSpan)

			span, err := suite.borKeeper.GetSpan(suite.ctx, tc.span.Id)
			require.NoError(err)
			require.Equal(tc.span, span)

			lastSpan, err := suite.borKeeper.GetLastSpan(suite.ctx)
			if tc.span.Id == spans[0].Id {
				require.Error(err)
				require.Nil(lastSpan)
			} else {
				require.NoError(err)
				require.NotEqual(tc.span, lastSpan)
				require.Equal(spans[0], lastSpan)
			}

			if tc.span.Id == spans[0].Id {
				err = suite.borKeeper.AddNewSpan(suite.ctx, tc.span)
				require.NoError(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestGetAllSpans() {
	require := suite.Require()
	spans := suite.genTestSpans(2)

	for _, s := range spans {
		err := suite.borKeeper.AddNewSpan(suite.ctx, s)
		require.NoError(err)
	}

	resSpans, err := suite.borKeeper.GetAllSpans(suite.ctx)
	require.NoError(err)
	require.Equal(spans, resSpans)
}

func (suite *KeeperTestSuite) TestFetchSpanList() {
	require := suite.Require()
	spans := suite.genTestSpans(30)

	expSpanList := make([]types.Span, 0, len(spans))
	for _, s := range spans {
		expSpanList = append(expSpanList, *s)
		err := suite.borKeeper.AddNewSpan(suite.ctx, s)
		require.NoError(err)
	}

	testcases := []struct {
		name         string
		page         uint64
		limit        uint64
		expSpanLists []types.Span
	}{
		{
			name:         "Normal limit",
			page:         1,
			limit:        10,
			expSpanLists: expSpanList[:10],
		},
		{
			name:         "Above limit",
			page:         1,
			limit:        30,
			expSpanLists: expSpanList[:20],
		},
	}

	for _, tc := range testcases {
		suite.T().Run(tc.name, func(t *testing.T) {
			resSpanLists, err := suite.borKeeper.FetchSpanList(suite.ctx, tc.page, tc.limit)
			require.NoError(err)

			require.Equal(tc.expSpanLists, resSpanLists)
		})
	}
}

func (suite *KeeperTestSuite) TestFreezeSet() {
	require := suite.Require()

	valSet, vals := suite.genTestValidators()
	suite.stakeKeeper.EXPECT().GetSpanEligibleValidators(suite.ctx).Return(vals).Times(1)
	suite.stakeKeeper.EXPECT().GetValidatorSet(suite.ctx).Return(valSet).Times(1)
	suite.stakeKeeper.EXPECT().GetValidatorFromValID(suite.ctx, gomock.Any()).DoAndReturn(func(ctx sdk.Context, valID uint64) (types.Validator, bool) {
		for _, v := range vals {
			if v.Id == valID {
				return v, true
			}
		}
		return types.Validator{}, false
	}).Times(len(vals))

	params := types.DefaultParams()

	testcases := []struct {
		name            string
		producerCount   uint64
		id              uint64
		startBlock      uint64
		endBlock        uint64
		seed            common.Hash
		expValSet       types.ValidatorSet
		expLastEthBlock *big.Int
	}{
		{
			name:            "Producer count is less than total validators",
			producerCount:   3,
			id:              1,
			startBlock:      1,
			endBlock:        100,
			seed:            common.HexToHash("testseed1"),
			expValSet:       valSet,
			expLastEthBlock: big.NewInt(0),
		},
		{
			name:            "Producer count is equal to total validators",
			producerCount:   5,
			id:              2,
			startBlock:      101,
			endBlock:        200,
			seed:            common.HexToHash("testseed2"),
			expValSet:       valSet,
			expLastEthBlock: big.NewInt(1),
		},
	}

	for _, tc := range testcases {
		suite.T().Run(tc.name, func(t *testing.T) {
			params.ProducerCount = tc.producerCount
			err := suite.borKeeper.SetParams(suite.ctx, params)
			require.NoError(err)

			resLastEthBlock, err := suite.borKeeper.GetLastEthBlock(suite.ctx)
			require.NoError(err)
			require.Equal(tc.expLastEthBlock, resLastEthBlock)

			err = suite.borKeeper.FreezeSet(suite.ctx, tc.id, tc.startBlock, tc.endBlock, "test-chain", tc.seed)
			require.NoError(err)

			resSpan, err := suite.borKeeper.GetSpan(suite.ctx, tc.id)
			require.NoError(err)
			require.Equal(tc.id, resSpan.Id)
			require.Equal(tc.startBlock, resSpan.StartBlock)
			require.Equal(tc.endBlock, resSpan.EndBlock)
			require.Equal("test-chain", resSpan.ChainId)
			require.Equal(tc.expValSet, resSpan.ValidatorSet)
			require.LessOrEqual(uint64(len(resSpan.SelectedProducers)), tc.producerCount)

			resLastEthBlock, err = suite.borKeeper.GetLastEthBlock(suite.ctx)
			require.NoError(err)
			require.Equal(tc.expLastEthBlock.Add(tc.expLastEthBlock, big.NewInt(1)), resLastEthBlock)
		})
	}

}

func (suite *KeeperTestSuite) TestUpdateLastSpan() {
	require := suite.Require()

	spans := suite.genTestSpans(2)

	for _, s := range spans {
		err := suite.borKeeper.AddNewSpan(suite.ctx, s)
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
		suite.T().Run(tc.name, func(t *testing.T) {
			resLastSpan, err := suite.borKeeper.GetLastSpan(suite.ctx)
			require.NoError(err)
			require.Equal(tc.expPrevLastSpan, resLastSpan)

			err = suite.borKeeper.UpdateLastSpan(suite.ctx, tc.expNewLastSpan.Id)
			require.NoError(err)

			resLastSpan, err = suite.borKeeper.GetLastSpan(suite.ctx)
			require.NoError(err)
			require.Equal(tc.expNewLastSpan, resLastSpan)

		})
	}
}

func (suite *KeeperTestSuite) TestIncrementLastEthBlock() {
	require := suite.Require()

	testcases := []struct {
		name            string
		setLastEthBlock *big.Int
		expLastEthBlock *big.Int
	}{
		{
			name:            "no eth block has been set",
			setLastEthBlock: nil,
			expLastEthBlock: big.NewInt(1),
		},
		{
			name:            "eth block gets correctly incremented",
			setLastEthBlock: big.NewInt(10),
			expLastEthBlock: big.NewInt(11),
		},
	}

	for _, tc := range testcases {
		suite.T().Run(tc.name, func(t *testing.T) {
			if tc.setLastEthBlock != nil {
				err := suite.borKeeper.SetLastEthBlock(suite.ctx, tc.setLastEthBlock)
				require.NoError(err)
			}

			err := suite.borKeeper.IncrementLastEthBlock(suite.ctx)
			require.NoError(err)

			resLastEthBlock, err := suite.borKeeper.GetLastEthBlock(suite.ctx)
			require.NoError(err)
			require.Equal(tc.expLastEthBlock, resLastEthBlock)

		})
	}
}

// TODO HV2: blocked by contract caller
func (suite *KeeperTestSuite) TestFetchNextSpanSeed() {}

func (suite *KeeperTestSuite) TestParamsGetterSetter() {
	ctx, borKeeper := suite.ctx, suite.borKeeper
	require := suite.Require()

	expParams := types.DefaultParams()
	expParams.ProducerCount = 66
	expParams.SpanDuration = 100
	expParams.SprintDuration = 64
	require.NoError(borKeeper.SetParams(ctx, expParams))
	resParams, err := borKeeper.FetchParams(ctx)
	require.NoError(err)
	require.True(expParams.Equal(resParams))
}

func (suite *KeeperTestSuite) genTestSpans(num uint64) []*types.Span {
	suite.T().Helper()
	valSet, vals := suite.genTestValidators()

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
			ChainId:           "test-chain",
		}
		spans = append(spans, &span)
	}

	return spans
}

func (suite *KeeperTestSuite) genTestValidators() (types.ValidatorSet, []types.Validator) {
	suite.T().Helper()
	var validators []*types.Validator
	err := json.Unmarshal([]byte(keeper.TestValidators), &validators)
	suite.Require().NoError(err)
	suite.Require().Equal(5, len(validators), "Total validators should be 5")

	valSet := types.ValidatorSet{
		Validators: validators,
	}

	vals := make([]types.Validator, 0, len(validators))
	for _, v := range validators {
		vals = append(vals, *v)
	}

	return valSet, vals

}
