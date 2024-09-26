package keeper_test

import (
	"time"

	"github.com/0xPolygon/heimdall-v2/x/clerk/types"
)

func (s *KeeperTestSuite) TestGetGRPCRecord_Success() {
	ctx, ck, queryClient, require := s.ctx, s.keeper, s.queryClient, s.Require()

	testRecord1 := types.NewEventRecord(TxHash1, 1, 1, Address1, make([]byte, 1), "1", time.Now())
	testRecord1.RecordTime = testRecord1.RecordTime.UTC()

	err := ck.SetEventRecord(ctx, testRecord1)
	require.NoError(err)

	req := &types.RecordRequest{
		RecordId: testRecord1.Id,
	}

	res, err := queryClient.GetRecordById(ctx, req)
	require.NoError(err)
	require.NotNil(res.Record)

}

func (s *KeeperTestSuite) TestGetGRPCRecord_NotFound() {
	ctx, queryClient, require := s.ctx, s.queryClient, s.Require()

	req := &types.RecordRequest{
		RecordId: 1,
	}

	res, err := queryClient.GetRecordById(ctx, req)
	require.Error(err)
	require.Nil(res)
}
