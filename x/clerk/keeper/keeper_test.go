package keeper_test

import (
	"testing"
	"time"

	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttime "github.com/cometbft/cometbft/types/time"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	// TODO HV2 - uncomment when contractCaller is implemented
	// "github.com/0xPolygon/heimdall-v2/helper/mocks"

	hmModule "github.com/0xPolygon/heimdall-v2/module"
	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/clerk"
	clerkKeeper "github.com/0xPolygon/heimdall-v2/x/clerk/keeper"
	"github.com/0xPolygon/heimdall-v2/x/clerk/types"
)

var Address1 = "0xa316fa9fa91700d7084d377bfdc81eb9f232f5ff"
var Address2 = "0xb316fa9fa91700d7084d377bfdc81eb9f232f5ff"
var TxHash1 = "0xc316fa9fa91700d7084d377bfdc81eb9f232f5ff"

// createTestApp returns context and app on clerk keeper
// nolint: unparam
/*
func createTestApp(t *testing.T, isCheckTx bool) (*app.HeimdallApp, sdk.Context) {
	app := app.Setup(t, isCheckTx)
	ctx := app.BaseApp.NewContext(isCheckTx)

	return app, ctx
}
*/

// KeeperTestSuite integrate test suite context object
type KeeperTestSuite struct {
	suite.Suite

	ctx    sdk.Context
	keeper clerkKeeper.Keeper
	// app        *app.HeimdallApp
	chainID     string
	msgServer   types.MsgServer
	sideMsgCfg  hmModule.SideTxConfigurator
	queryClient types.QueryClient
	// TODO HV2 - uncomment when contractCaller is implemented
	// contractCaller mocks.IContractCaller
}

func TestKeeperTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(KeeperTestSuite))
}

func (suite *KeeperTestSuite) SetupTest() {
	key := storetypes.NewKVStoreKey(types.StoreKey)
	storeService := runtime.NewKVStoreService(key)
	testCtx := testutil.DefaultContextWithDB(suite.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx.WithBlockHeader(cmtproto.Header{Time: cmttime.Now()})
	encCfg := moduletestutil.MakeTestEncodingConfig()

	keeper := clerkKeeper.NewKeeper(
		encCfg.Codec,
		storeService,
	)

	suite.ctx = ctx
	suite.keeper = keeper

	clerkGenesis := types.DefaultGenesisState()

	clerk.InitGenesis(ctx, &keeper, clerkGenesis)

	types.RegisterInterfaces(encCfg.InterfaceRegistry)
	queryHelper := baseapp.NewQueryServerTestHelper(ctx, encCfg.InterfaceRegistry)
	types.RegisterQueryServer(queryHelper, clerkKeeper.QueryServer{K: keeper})
	suite.queryClient = types.NewQueryClient(queryHelper)
	suite.msgServer = clerkKeeper.NewMsgServerImpl(keeper)

	suite.sideMsgCfg = hmModule.NewSideTxConfigurator()
	types.RegisterSideMsgServer(suite.sideMsgCfg, clerkKeeper.NewSideMsgServerImpl(keeper))
}

func (suite *KeeperTestSuite) TestHasGetSetEventRecord() {
	t, ctx, ck := suite.T(), suite.ctx, suite.keeper

	ac := address.NewHexCodec()
	addrBz, err := ac.StringToBytes(Address1)
	require.NoError(t, err)

	hHash := hmTypes.HeimdallHash{Hash: addrBz}

	testRecord1 := types.NewEventRecord(hHash, 1, 1, Address1, hmTypes.HexBytes{HexBytes: make([]byte, 0)}, "1", time.Now())

	// SetEventRecord
	err = ck.SetEventRecord(ctx, testRecord1)
	require.Nil(t, err)

	err = ck.SetEventRecord(ctx, testRecord1)
	require.NotNil(t, err)

	// GetEventRecord
	respRecord, err := ck.GetEventRecord(ctx, testRecord1.ID)
	require.Nil(t, err)
	require.Equal(t, (*respRecord).ID, testRecord1.ID)

	_, err = ck.GetEventRecord(ctx, testRecord1.ID+1)
	require.NotNil(t, err)

	// HasEventRecord
	recordPresent := ck.HasEventRecord(ctx, testRecord1.ID)
	require.True(t, recordPresent)

	recordPresent = ck.HasEventRecord(ctx, testRecord1.ID+1)
	require.False(t, recordPresent)

	recordList := ck.GetAllEventRecords(ctx)
	require.Len(t, recordList, 1)
}

func (suite *KeeperTestSuite) TestGetEventRecordList() {
	t, ctx, ck := suite.T(), suite.ctx, suite.keeper

	var i uint64

	ac := address.NewHexCodec()
	addrBz, err := ac.StringToBytes(Address1)
	require.NoError(t, err)
	hHash := hmTypes.HeimdallHash{Hash: addrBz}

	for i = 0; i < 60; i++ {
		testRecord := types.NewEventRecord(hHash, i, i, Address1, hmTypes.HexBytes{HexBytes: make([]byte, 0)}, "1", time.Now())
		err := ck.SetEventRecord(ctx, testRecord)
		require.NoError(t, err)
	}

	recordList, _ := ck.GetEventRecordList(ctx, 1, 20)
	require.Len(t, recordList, 20)

	recordList, _ = ck.GetEventRecordList(ctx, 2, 20)
	require.Len(t, recordList, 20)

	recordList, _ = ck.GetEventRecordList(ctx, 3, 30)
	require.Len(t, recordList, 0)

	recordList, _ = ck.GetEventRecordList(ctx, 1, 70)
	require.Len(t, recordList, 50)

	recordList, _ = ck.GetEventRecordList(ctx, 2, 60)
	require.Len(t, recordList, 10)
}

func (suite *KeeperTestSuite) TestGetEventRecordListTime() {
	t, ctx, ck := suite.T(), suite.ctx, suite.keeper

	var i uint64

	ac := address.NewHexCodec()
	addrBz, err := ac.StringToBytes(Address1)
	require.NoError(t, err)
	hHash := hmTypes.HeimdallHash{Hash: addrBz}

	for i = 0; i < 30; i++ {
		testRecord := types.NewEventRecord(hHash, i, i, Address1, hmTypes.HexBytes{HexBytes: make([]byte, 0)}, "1", time.Unix(int64(i), 0))
		err := ck.SetEventRecord(ctx, testRecord)
		require.NoError(t, err)
	}

	recordList, err := ck.GetEventRecordListWithTime(ctx, time.Unix(1, 0), time.Unix(6, 0), 0, 0)
	require.NoError(t, err)
	require.Len(t, recordList, 5)
	require.Equal(t, int64(5), recordList[len(recordList)-1].RecordTime.Unix())

	recordList, err = ck.GetEventRecordListWithTime(ctx, time.Unix(1, 0), time.Unix(6, 0), 1, 1)
	require.NoError(t, err)
	require.Len(t, recordList, 1)

	recordList, err = ck.GetEventRecordListWithTime(ctx, time.Unix(10, 0), time.Unix(20, 0), 0, 0)
	require.NoError(t, err)
	require.Len(t, recordList, 10)
	require.Equal(t, int64(10), recordList[0].RecordTime.Unix())
	require.Equal(t, int64(19), recordList[len(recordList)-1].RecordTime.Unix())
}

func (suite *KeeperTestSuite) TestSetHasGetRecordSequence() {
	t, ctx, ck := suite.T(), suite.ctx, suite.keeper

	testSeq := "testseq"

	ck.SetRecordSequence(ctx, testSeq)
	found := ck.HasRecordSequence(ctx, testSeq)
	require.True(t, found)

	found = ck.HasRecordSequence(ctx, "testSeq")
	require.False(t, found)

	recordSequences := ck.GetRecordSequences(ctx)
	require.Len(t, recordSequences, 1)
}
