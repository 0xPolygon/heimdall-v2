package keeper_test

import (
	"time"

	"github.com/cosmos/cosmos-sdk/types/query"

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

func (s *KeeperTestSuite) TestGetRecordListWithTime_Success() {
	ctx, ck, queryClient, require := s.ctx, s.keeper, s.queryClient, s.Require()

	now := time.Now().UTC()

	for i := uint64(1); i <= 3; i++ {
		rec := types.NewEventRecord(TxHash1, i, i, Address1, make([]byte, 1), "1", now.Add(-time.Duration(i)*time.Minute))
		require.NoError(ck.SetEventRecord(ctx, rec))
	}

	req := &types.RecordListWithTimeRequest{
		FromId:     1,
		ToTime:     now,
		Pagination: query.PageRequest{Limit: 10},
	}

	res, err := queryClient.GetRecordListWithTime(ctx, req)
	require.NoError(err)
	require.NotNil(res)
	require.Len(res.EventRecords, 3)

	for _, rec := range res.EventRecords {
		require.True(rec.RecordTime.Before(now))
		require.GreaterOrEqual(rec.Id, uint64(1))
	}
}

func (s *KeeperTestSuite) TestGetRecordListWithTime_Pagination() {
	ctx, ck, queryClient, require := s.ctx, s.keeper, s.queryClient, s.Require()
	now := time.Now().UTC()

	// Insert 10 test records
	for i := uint64(1); i <= 10; i++ {
		rec := types.NewEventRecord(
			TxHash1, i, i, Address1, make([]byte, 1), "1",
			now.Add(-time.Duration(i)*time.Minute), // decreasing timestamp
		)
		require.NoError(ck.SetEventRecord(ctx, rec))
	}

	type testCase struct {
		name          string
		fromID        uint64
		toTime        time.Time
		pagination    query.PageRequest
		expectedIDs   []uint64
		expectedError bool
	}

	tests := []testCase{
		{
			name:        "limit 1, from_id 1",
			fromID:      1,
			toTime:      now,
			pagination:  query.PageRequest{Offset: 0, Limit: 1},
			expectedIDs: []uint64{1},
		},
		{
			name:        "limit 5, from_id 3",
			fromID:      3,
			toTime:      now,
			pagination:  query.PageRequest{Offset: 0, Limit: 5},
			expectedIDs: []uint64{3, 4, 5, 6, 7},
		},
		{
			name:        "limit beyond max (truncates)",
			fromID:      1,
			toTime:      now,
			pagination:  query.PageRequest{Offset: 0, Limit: 100},
			expectedIDs: []uint64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
		{
			name:        "offset skipping first 5",
			fromID:      1,
			toTime:      now,
			pagination:  query.PageRequest{Offset: 5, Limit: 5},
			expectedIDs: []uint64{6, 7, 8, 9, 10},
		},
		{
			name:        "from_id beyond dataset",
			fromID:      50,
			toTime:      now,
			pagination:  query.PageRequest{Limit: 5},
			expectedIDs: []uint64{},
		},
		{
			name:        "to_time before all records",
			fromID:      1,
			toTime:      now.Add(-11 * time.Minute),
			pagination:  query.PageRequest{Limit: 5},
			expectedIDs: []uint64{},
		},
		{
			name:        "empty pagination default limit",
			fromID:      1,
			toTime:      now,
			pagination:  query.PageRequest{},
			expectedIDs: []uint64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			req := &types.RecordListWithTimeRequest{
				FromId:     tc.fromID,
				ToTime:     tc.toTime,
				Pagination: tc.pagination,
			}
			res, err := queryClient.GetRecordListWithTime(ctx, req)

			if tc.expectedError {
				require.Error(err)
			} else {
				require.NoError(err)
				require.Len(res.EventRecords, len(tc.expectedIDs))
				for i, rec := range res.EventRecords {
					require.Equal(tc.expectedIDs[i], rec.Id)
				}
			}
		})
	}
}
