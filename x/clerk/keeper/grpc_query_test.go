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
			now.Add(-time.Duration(i)*time.Minute), // Decreasing timestamp.
		)
		require.NoError(ck.SetEventRecord(ctx, rec))
	}

	type testCase struct {
		name          string
		fromID        uint64
		toTime        time.Time
		limit         uint64
		offset        uint64
		expectedIDs   []uint64
		expectedError bool
	}

	tests := []testCase{
		{
			name:        "limit 1, from_id 1",
			fromID:      1,
			toTime:      now,
			limit:       1,
			expectedIDs: []uint64{1},
		},
		{
			name:        "limit 5, from_id 3",
			fromID:      3,
			toTime:      now,
			limit:       5,
			expectedIDs: []uint64{3, 4, 5, 6, 7},
		},
		{
			name:          "limit beyond max (truncates), limit cannot be greater than 50",
			fromID:        1,
			toTime:        now,
			limit:         100,
			expectedError: true,
		},
		{
			name:        "skipping first 5",
			fromID:      6,
			toTime:      now,
			limit:       5,
			expectedIDs: []uint64{6, 7, 8, 9, 10},
		},
		{
			name:        "from_id beyond dataset",
			fromID:      50,
			toTime:      now,
			limit:       5,
			expectedIDs: []uint64{},
		},
		{
			name:        "to_time before all records",
			fromID:      1,
			toTime:      now.Add(-11 * time.Minute),
			limit:       5,
			expectedIDs: []uint64{},
		},
		{
			name:          "empty pagination limit, limit cannot be 0",
			fromID:        1,
			toTime:        now,
			limit:         0,
			expectedError: true,
		},
		{
			name:        "limit 3 with offset 2",
			fromID:      1,
			toTime:      now,
			limit:       3,
			offset:      2,
			expectedIDs: []uint64{3, 4, 5},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			req := &types.RecordListWithTimeRequest{
				FromId:     tc.fromID,
				ToTime:     tc.toTime,
				Pagination: query.PageRequest{Key: []byte{0x00}, Limit: tc.limit, Offset: tc.offset},
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
