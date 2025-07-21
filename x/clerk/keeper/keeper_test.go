package keeper_test

import (
	"math/rand"
	"strconv"
	"testing"
	"time"

	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttime "github.com/cometbft/cometbft/types/time"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/runtime"
	cosmostestutil "github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/0xPolygon/heimdall-v2/helper/mocks"
	"github.com/0xPolygon/heimdall-v2/sidetxs"
	clerkKeeper "github.com/0xPolygon/heimdall-v2/x/clerk/keeper"
	"github.com/0xPolygon/heimdall-v2/x/clerk/testutil"
	"github.com/0xPolygon/heimdall-v2/x/clerk/types"
)

var (
	Address1 = "0xa316fa9fa91700d7084d377bfdc81eb9f232f5ff"
	Address2 = "0xb316fa9fa91700d7084d377bfdc81eb9f232f5ff"
	TxHash1  = "0x000000000000000000000000000000000000000000000000000000000000dead"
)

// KeeperTestSuite integrate test suite context object
type KeeperTestSuite struct {
	suite.Suite

	ctx            sdk.Context
	keeper         clerkKeeper.Keeper
	chainId        string
	msgServer      types.MsgServer
	sideMsgCfg     sidetxs.SideTxConfigurator
	queryClient    types.QueryClient
	contractCaller mocks.IContractCaller
}

func TestKeeperTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(KeeperTestSuite))
}

func (s *KeeperTestSuite) SetupTest() {
	key := storetypes.NewKVStoreKey(types.StoreKey)
	storeService := runtime.NewKVStoreService(key)
	testCtx := cosmostestutil.DefaultContextWithDB(s.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx.WithBlockHeader(cmtproto.Header{Time: cmttime.Now()})
	encCfg := moduletestutil.MakeTestEncodingConfig()
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()
	chainKeeper := testutil.NewMockChainKeeper(ctrl)
	s.contractCaller = mocks.IContractCaller{}

	keeper := clerkKeeper.NewKeeper(
		encCfg.Codec,
		storeService,
		chainKeeper,
		&s.contractCaller,
	)

	s.ctx = ctx
	s.keeper = keeper

	s.chainId = "15001"

	clerkGenesis := types.DefaultGenesisState()

	keeper.InitGenesis(ctx, clerkGenesis)

	types.RegisterInterfaces(encCfg.InterfaceRegistry)
	queryHelper := baseapp.NewQueryServerTestHelper(ctx, encCfg.InterfaceRegistry)
	types.RegisterQueryServer(queryHelper, clerkKeeper.NewQueryServer(&keeper))
	s.queryClient = types.NewQueryClient(queryHelper)
	s.msgServer = clerkKeeper.NewMsgServerImpl(keeper)

	s.sideMsgCfg = sidetxs.NewSideTxConfigurator()
	types.RegisterSideMsgServer(s.sideMsgCfg, clerkKeeper.NewSideMsgServerImpl(keeper))
}

func (s *KeeperTestSuite) TestHasGetSetEventRecord() {
	t, ctx, ck := s.T(), s.ctx, s.keeper

	testRecord1 := types.NewEventRecord(TxHash1, 1, 1, Address1, make([]byte, 1), "1", time.Now())
	testRecord1.RecordTime = testRecord1.RecordTime.UTC()

	// SetEventRecord
	err := ck.SetEventRecord(ctx, testRecord1)
	require.Nil(t, err)

	err = ck.SetEventRecord(ctx, testRecord1)
	require.NotNil(t, err)

	// GetEventRecord
	respRecord, err := ck.GetEventRecord(ctx, testRecord1.Id)
	require.Nil(t, err)
	require.Equal(t, testRecord1, *respRecord)

	_, err = ck.GetEventRecord(ctx, testRecord1.Id+1)
	require.NotNil(t, err)

	// HasEventRecord
	recordPresent := ck.HasEventRecord(ctx, testRecord1.Id)
	require.True(t, recordPresent)

	recordPresent = ck.HasEventRecord(ctx, testRecord1.Id+1)
	require.False(t, recordPresent)

	recordList := ck.GetAllEventRecords(ctx)
	require.Len(t, recordList, 1)
}

func (s *KeeperTestSuite) TestGetEventRecordList() {
	t, ctx, ck := s.T(), s.ctx, s.keeper

	var i uint64

	var testRecords []types.EventRecord

	for i = 1; i <= 60; i++ {
		testRecord := types.NewEventRecord(TxHash1, i, i, Address1, make([]byte, 1), "1", time.Now())
		testRecord.RecordTime = testRecord.RecordTime.UTC()
		err := ck.SetEventRecord(ctx, testRecord)
		require.NoError(t, err)

		testRecords = append(testRecords, testRecord)
	}

	recordList := ck.GetAllEventRecords(ctx)
	require.Len(t, recordList, 60)

	for i, record := range recordList {
		require.Equal(t, testRecords[i], record)
	}

	recordList, err := ck.GetEventRecordList(ctx, 1, 20)
	require.NoError(t, err)
	require.Len(t, recordList, 20)

	recordList, err = ck.GetEventRecordList(ctx, 2, 20)
	require.NoError(t, err)
	require.Len(t, recordList, 20)

	recordList, err = ck.GetEventRecordList(ctx, 3, 30)
	require.NotNil(t, err)
	require.Len(t, recordList, 0)

	// Page cannot be 0.
	recordList, err = ck.GetEventRecordList(ctx, 0, 10)
	require.Error(t, err)

	// Limit cannot be 0.
	recordList, err = ck.GetEventRecordList(ctx, 1, 0)
	require.Error(t, err)

	// Limit cannot be greater than 50.
	recordList, err = ck.GetEventRecordList(ctx, 1, 70)
	require.Error(t, err)

	recordList, err = ck.GetEventRecordList(ctx, 2, 50)
	require.NoError(t, err)
	require.Len(t, recordList, 10)
}

func (s *KeeperTestSuite) TestGetEventRecordListTime() {
	t, ctx, ck := s.T(), s.ctx, s.keeper

	var i uint64

	for i = 0; i < 30; i++ {
		testRecord := types.NewEventRecord(TxHash1, i, i, Address1, make([]byte, 1), "1", time.Unix(int64(i), 0))
		testRecord.RecordTime = testRecord.RecordTime.UTC()
		err := ck.SetEventRecord(ctx, testRecord)
		require.NoError(t, err)
	}

	recordList, err := ck.GetEventRecordListWithTime(ctx, time.Unix(1, 0), time.Unix(6, 0), 0, 1)
	// Page cannot be 0.
	require.Error(t, err)

	// Limit cannot be 0.
	recordList, err = ck.GetEventRecordListWithTime(ctx, time.Unix(1, 0), time.Unix(6, 0), 1, 0)
	require.Error(t, err)

	recordList, err = ck.GetEventRecordListWithTime(ctx, time.Unix(1, 0), time.Unix(6, 0), 1, 1)
	require.NoError(t, err)
	require.Len(t, recordList, 1)

	recordList, err = ck.GetEventRecordListWithTime(ctx, time.Unix(10, 0), time.Unix(20, 0), 1, 10)
	require.NoError(t, err)
	require.Len(t, recordList, 10)
	require.Equal(t, int64(10), recordList[0].RecordTime.Unix())
	require.Equal(t, int64(19), recordList[len(recordList)-1].RecordTime.Unix())
}

func (s *KeeperTestSuite) TestSetHasGetRecordSequence() {
	t, ctx, ck := s.T(), s.ctx, s.keeper

	testSeq := "testseq"

	ck.SetRecordSequence(ctx, testSeq)
	found := ck.HasRecordSequence(ctx, testSeq)
	require.True(t, found)

	found = ck.HasRecordSequence(ctx, "testSeq")
	require.False(t, found)

	recordSequences := ck.GetRecordSequences(ctx)
	require.Len(t, recordSequences, 1)
}

func (s *KeeperTestSuite) TestInitExportGenesis() {
	t, ctx, ck := s.T(), s.ctx, s.keeper

	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	recordSequences := make([]string, 5)
	eventRecords := make([]types.EventRecord, 1)

	for i := range recordSequences {
		recordSequences[i] = strconv.Itoa(simulation.RandIntBetween(r1, 1000, 100000))
	}

	testEventRecord := types.NewEventRecord(TxHash1, 1, 1, Address1, make([]byte, 1), "1", time.Now())
	testEventRecord.RecordTime = testEventRecord.RecordTime.UTC()
	eventRecords[0] = testEventRecord

	genesisState := types.GenesisState{
		EventRecords:    eventRecords,
		RecordSequences: recordSequences,
	}

	ck.InitGenesis(ctx, &genesisState)
	actualParams := ck.ExportGenesis(ctx)
	require.Equal(t, len(recordSequences), len(actualParams.RecordSequences))
	require.Equal(t, len(eventRecords), len(actualParams.EventRecords))
}
