package keeper_test

import (
	"time"

	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/clerk/types"
)

func (suite *KeeperTestSuite) TestGetGRPCRecord_Success() {
	ctx, ck, queryClient, require := suite.ctx, suite.keeper, suite.queryClient, suite.Require()

	testRecord1 := types.NewEventRecord(TxHash1, 1, 1, Address1, hmTypes.HexBytes{HexBytes: make([]byte, 1)}, "1", time.Now())
	testRecord1.RecordTime = testRecord1.RecordTime.UTC()

	err := ck.SetEventRecord(ctx, testRecord1)
	require.NoError(err)

	req := &types.RecordRequest{
		RecordId: testRecord1.Id,
	}

	res, err := queryClient.Record(ctx, req)
	require.NoError(err)
	require.NotNil(res.Record)

}

func (suite *KeeperTestSuite) TestGetGRPCRecord_NotFound() {
	ctx, queryClient, require := suite.ctx, suite.queryClient, suite.Require()

	req := &types.RecordRequest{
		RecordId: 1,
	}

	res, err := queryClient.Record(ctx, req)
	require.Error(err)
	require.Nil(res)
}
