package keeper_test

import (
	"math/rand"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/0xPolygon/heimdall-v2/app"
	// TODO HV2 - uncomment when contractCaller is implemented
	// "github.com/0xPolygon/heimdall-v2/helper/mocks"
	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/clerk/keeper"
	"github.com/0xPolygon/heimdall-v2/x/clerk/types"
	hmModule "github.com/0xPolygon/heimdall-v2/x/types/module"
)

// returns context and app on clerk keeper
// nolint: unparam
func createTestApp(isCheckTx bool) (*app.HeimdallApp, sdk.Context) {
	app := &app.HeimdallApp{}
	ctx := app.BaseApp.NewContext(isCheckTx)

	return app, ctx
}

//
// Test suite
//

// KeeperTestSuite integrate test suite context object
type KeeperTestSuite struct {
	suite.Suite

	ctx        sdk.Context
	app        *app.HeimdallApp
	chainID    string
	msgServer  types.MsgServer
	sideMsgCfg hmModule.SideTxConfigurator
	// TODO HV2 - uncomment when contractCaller is implemented
	// contractCaller mocks.IContractCaller
	r *rand.Rand
}

func (suite *KeeperTestSuite) SetupTest() {
	suite.app, suite.ctx = createTestApp(false)
	// TODO HV2 - uncomment when contract caller is implemented
	// suite.contractCaller = mocks.IContractCaller{}
	suite.msgServer = keeper.NewMsgServerImpl(suite.app.ClerkKeeper)

	// fetch chain id
	// TODO HV2 - uncomment when ChainKeeper is implemented
	// suite.chainID = suite.app.ChainKeeper.GetParams(suite.ctx).ChainParams.BorChainID

	// random generator
	s1 := rand.NewSource(time.Now().UnixNano())
	suite.r = rand.New(s1)
}

func TestKeeperTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(KeeperTestSuite))
}

//
// Tests
//

func (suite *KeeperTestSuite) TestHasGetSetEventRecord() {
	t, app, ctx := suite.T(), suite.app, suite.ctx

	hAddr := "some-address"
	// TODO HV2 - uncomment when auth PR is merged and hexCodec is implemented
	// hHash := hmTypes.BytesToHeimdallHash([]byte("some-address"))
	hHash := hmTypes.HeimdallHash{}
	testRecord1 := types.NewEventRecord(hHash, 1, 1, hAddr, hmTypes.HexBytes{HexBytes: make([]byte, 0)}, "1", time.Now())

	// SetEventRecord
	ck := app.ClerkKeeper
	err := ck.SetEventRecord(ctx, testRecord1)
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
	t, app, ctx := suite.T(), suite.app, suite.ctx

	var i uint64

	hAddr := "some-address"
	// TODO HV2 - uncomment when auth PR is merged and hexCodec is implemented
	// hHash := hmTypes.BytesToHeimdallHash([]byte("some-address"))
	hHash := hmTypes.HeimdallHash{}
	ck := app.ClerkKeeper

	for i = 0; i < 60; i++ {
		testRecord := types.NewEventRecord(hHash, i, i, hAddr, hmTypes.HexBytes{HexBytes: make([]byte, 0)}, "1", time.Now())
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
	t, app, ctx := suite.T(), suite.app, suite.ctx

	var i uint64

	hAddr := "some-address"
	// TODO HV2 - uncomment when auth PR is merged and hexCodec is implemented
	// hHash := hmTypes.BytesToHeimdallHash([]byte("some-address"))
	hHash := hmTypes.HeimdallHash{}
	ck := app.ClerkKeeper

	for i = 0; i < 30; i++ {
		testRecord := types.NewEventRecord(hHash, i, i, hAddr, hmTypes.HexBytes{HexBytes: make([]byte, 0)}, "1", time.Unix(int64(i), 0))
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

func (suite *KeeperTestSuite) TestGetEventRecordKey() {
	t, _, _ := suite.T(), suite.app, suite.ctx

	hAddr := "some-address"
	// TODO HV2 - uncomment when auth PR is merged and hexCodec is implemented
	// hHash := hmTypes.BytesToHeimdallHash([]byte("some-address"))
	hHash := hmTypes.HeimdallHash{}
	testRecord1 := types.NewEventRecord(hHash, 1, 1, hAddr, hmTypes.HexBytes{HexBytes: make([]byte, 0)}, "1", time.Now())

	respKey := keeper.GetEventRecordKey(testRecord1.ID)
	require.Equal(t, respKey, []byte{17, 49})
}

func (suite *KeeperTestSuite) TestSetHasGetRecordSequence() {
	t, app, ctx := suite.T(), suite.app, suite.ctx

	testSeq := "testseq"
	ck := app.ClerkKeeper
	ck.SetRecordSequence(ctx, testSeq)
	found := ck.HasRecordSequence(ctx, testSeq)
	require.True(t, found)

	found = ck.HasRecordSequence(ctx, "testSeq")
	require.False(t, found)

	recordSequences := ck.GetRecordSequences(ctx)
	require.Len(t, recordSequences, 1)
}
