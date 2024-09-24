package keeper_test

import (
	"errors"
	"math/big"
	"testing"

	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttime "github.com/cometbft/cometbft/types/time"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"

	"github.com/0xPolygon/heimdall-v2/helper/mocks"
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
	contractCaller     mocks.IContractCaller
	queryClient        types.QueryClient
	msgServer          types.MsgServer
	sideMsgServer      types.SideMsgServer
	encCfg             moduletestutil.TestEncodingConfig
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

func (s *KeeperTestSuite) SetupTest() {
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

	s.contractCaller = mocks.IContractCaller{}
	s.ctx = ctx

	s.borKeeper = keeper.NewKeeper(
		encCfg.Codec,
		storeService,
		s.chainManagerKeeper,
		s.stakeKeeper,
		&s.contractCaller,
	)

	types.RegisterInterfaces(encCfg.InterfaceRegistry)

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

	params := types.DefaultParams()

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

	stakeKeeper.EXPECT().GetSpanEligibleValidators(ctx).Return(vals).Times(len(testcases))
	stakeKeeper.EXPECT().GetValidatorSet(ctx).Return(valSet, nil).Times(len(testcases))
	stakeKeeper.EXPECT().GetValidatorFromValID(ctx, gomock.Any()).DoAndReturn(func(ctx sdk.Context, valID uint64) (staketypes.Validator, error) {
		for _, v := range vals {
			if v.ValId == valID {
				return v, nil
			}
		}
		return staketypes.Validator{}, errors.New("validator not found")
	}).AnyTimes()

	for _, tc := range testcases {
		s.T().Run(tc.name, func(t *testing.T) {
			params.ProducerCount = tc.producerCount
			err := borKeeper.SetParams(ctx, params)
			require.NoError(err)

			resLastEthBlock, err := borKeeper.GetLastEthBlock(ctx)
			require.NoError(err)
			require.Equal(tc.expLastEthBlock, resLastEthBlock)

			err = borKeeper.FreezeSet(ctx, tc.id, tc.startBlock, tc.endBlock, "test-chain", tc.seed)
			require.NoError(err)

			resSpan, err := borKeeper.GetSpan(ctx, tc.id)
			require.NoError(err)
			require.Equal(tc.id, resSpan.Id)
			require.Equal(tc.startBlock, resSpan.StartBlock)
			require.Equal(tc.endBlock, resSpan.EndBlock)
			require.Equal("test-chain", resSpan.ChainId)
			require.Equal(tc.expValSet, resSpan.ValidatorSet)
			require.LessOrEqual(uint64(len(resSpan.SelectedProducers)), tc.producerCount)

			resLastEthBlock, err = borKeeper.GetLastEthBlock(ctx)
			require.NoError(err)
			require.Equal(tc.expLastEthBlock.Add(tc.expLastEthBlock, big.NewInt(1)), resLastEthBlock)
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

func (s *KeeperTestSuite) TestIncrementLastEthBlock() {
	require, ctx, borKeeper := s.Require(), s.ctx, s.borKeeper

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
		s.T().Run(tc.name, func(t *testing.T) {
			if tc.setLastEthBlock != nil {
				err := borKeeper.SetLastEthBlock(ctx, tc.setLastEthBlock)
				require.NoError(err)
			}

			err := borKeeper.IncrementLastEthBlock(ctx)
			require.NoError(err)

			resLastEthBlock, err := borKeeper.GetLastEthBlock(ctx)
			require.NoError(err)
			require.Equal(tc.expLastEthBlock, resLastEthBlock)

		})
	}
}

func (s *KeeperTestSuite) TestFetchNextSpanSeed() {
	require, ctx, borKeeper, contractCaller := s.Require(), s.ctx, s.borKeeper, &s.contractCaller

	lastEthBlock := big.NewInt(10)
	nextEthBlock := big.NewInt(11)
	nextEthBlockHeader := &ethTypes.Header{Number: big.NewInt(11)}
	nextEthBlockHash := nextEthBlockHeader.Hash()
	err := borKeeper.SetLastEthBlock(ctx, lastEthBlock)
	require.NoError(err)
	contractCaller.On("GetMainChainBlock", nextEthBlock).Return(nextEthBlockHeader, nil).Times(1)

	res, err := borKeeper.FetchNextSpanSeed(ctx)
	require.NoError(err)
	require.Equal(nextEthBlockHash, res)
}

func (s *KeeperTestSuite) TestParamsGetterSetter() {
	require, ctx, borKeeper := s.Require(), s.ctx, s.borKeeper

	expParams := types.DefaultParams()
	expParams.ProducerCount = 66
	expParams.SpanDuration = 100
	expParams.SprintDuration = 64
	require.NoError(borKeeper.SetParams(ctx, expParams))
	resParams, err := borKeeper.FetchParams(ctx)
	require.NoError(err)
	require.True(expParams.Equal(resParams))
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
			ChainId:           "test-chain",
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
