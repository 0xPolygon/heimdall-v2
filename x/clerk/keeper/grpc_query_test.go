package keeper_test

import (
	"time"

	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/clerk/types"
)

func (suite *KeeperTestSuite) TestGetGRPCRecord_Success() {
	ctx, ck, queryClient, require := suite.ctx, suite.keeper, suite.queryClient, suite.Require()

	testRecord1 := types.NewEventRecord(TxHash1, 1, 1, Address1, hmTypes.HexBytes{HexBytes: make([]byte, 1)}, "1", time.Now())

	err := ck.SetEventRecord(ctx, testRecord1)
	require.NoError(err)

	req := &types.RecordRequest{
		RecordID: testRecord1.ID,
	}

	res, err := queryClient.Record(ctx, req)
	require.NoError(err)
	require.NotNil(res.Record)
	require.Equal(res.Record.ID, testRecord1.ID)
	require.Equal(res.Record.Contract, testRecord1.Contract)
	require.Equal(res.Record.Data, testRecord1.Data)
	require.Equal(res.Record.TxHash, testRecord1.TxHash)
	require.Equal(res.Record.LogIndex, testRecord1.LogIndex)
	require.Equal(res.Record.BorChainID, testRecord1.BorChainID)
	/*
		Expected time is in UTC, but actual time is in Local timezone
		require.Equal(res.Record.RecordTime, testRecord1.RecordTime)
	*/
}

func (suite *KeeperTestSuite) TestGetGRPCRecord_NotFound() {
	ctx, queryClient, require := suite.ctx, suite.queryClient, suite.Require()

	req := &types.RecordRequest{
		RecordID: 1,
	}

	res, err := queryClient.Record(ctx, req)
	require.Error(err)
	require.Nil(res)
}
