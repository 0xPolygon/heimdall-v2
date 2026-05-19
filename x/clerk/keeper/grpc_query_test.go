package keeper_test

import (
	"time"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/0xPolygon/heimdall-v2/helper"
	clerkKeeper "github.com/0xPolygon/heimdall-v2/x/clerk/keeper"
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

func (s *KeeperTestSuite) TestGetRecordListWithTime_RejectsUnixEpochTimestamp() {
	ctx, queryClient, require := s.ctx, s.queryClient, s.Require()

	res, err := queryClient.GetRecordListWithTime(ctx, &types.RecordListWithTimeRequest{
		FromId:     1,
		ToTime:     time.Unix(0, 0).UTC(),
		Pagination: query.PageRequest{Limit: 10, Key: []byte{0x00}},
	})
	require.Error(err)
	require.Nil(res)
	require.Equal(codes.InvalidArgument, status.Code(err))
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

func enableVisibilityTimeForTest(_ *KeeperTestSuite, height int64) func() {
	original := helper.GetV080HardforkHeight()
	helper.SetV080HardforkHeight(height)
	return func() { helper.SetV080HardforkHeight(original) }
}

// TestGetRecordListWithTime_Deterministic_HappyPath verifies the post-HF path
// returns events when the stability gate passes (blockTime > cutoff) and a
// valid height resolves.
func (s *KeeperTestSuite) TestGetRecordListWithTime_Deterministic_HappyPath() {
	ctx, ck := s.ctx, s.keeper
	require := s.Require()
	defer enableVisibilityTimeForTest(s, 1)()

	baseTime := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

	require.NoError(ck.SetVisibilityTimeUpgradeID(ctx, 1))

	for i := uint64(1); i <= 3; i++ {
		rec := types.NewEventRecord(TxHash1, i, i, Address1, make([]byte, 1), "1",
			baseTime.Add(time.Duration(i)*time.Second))
		require.NoError(ck.SetEventRecord(ctx, rec))
		require.NoError(ck.VisibilityHeightByID.Set(ctx, i, 100+i))
	}

	for i := int64(0); i < 5; i++ {
		h := 100 + i
		bt := baseTime.Add(time.Duration(i*3) * time.Second)
		ctx = ctx.WithBlockHeight(h).WithBlockHeader(cmtproto.Header{Time: bt, Height: h})
		require.NoError(ck.StoreBlockTime(ctx))
	}

	cutoffTime := baseTime.Add(10 * time.Second)
	ctx = ctx.WithBlockHeight(10000).WithBlockHeader(cmtproto.Header{
		Time:   cutoffTime.Add(time.Minute),
		Height: 10000,
	})

	resp, err := clerkKeeper.NewQueryServer(&ck).GetRecordListWithTime(ctx, &types.RecordListWithTimeRequest{
		FromId:     1,
		ToTime:     cutoffTime,
		Pagination: query.PageRequest{Limit: 10, Key: []byte{0x00}},
	})
	require.NoError(err)
	require.NotNil(resp)
	require.NotEmpty(resp.EventRecords, "should return events when stability gate passes")
}

// TestGetRecordListWithTime_Deterministic_UsesIndexedBlockTime verifies the
// stability gate uses the indexed block time, not sdkCtx.BlockTime (which is
// often zero on REST query paths).
func (s *KeeperTestSuite) TestGetRecordListWithTime_Deterministic_UsesIndexedBlockTime() {
	ctx, ck := s.ctx, s.keeper
	require := s.Require()
	defer enableVisibilityTimeForTest(s, 1)()

	baseTime := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

	require.NoError(ck.SetVisibilityTimeUpgradeID(ctx, 1))

	for i := uint64(1); i <= 3; i++ {
		rec := types.NewEventRecord(TxHash1, i, i, Address1, make([]byte, 1), "1",
			baseTime.Add(time.Duration(i)*time.Second))
		require.NoError(ck.SetEventRecord(ctx, rec))
		require.NoError(ck.VisibilityHeightByID.Set(ctx, i, 100+i))
	}

	for i := int64(0); i < 5; i++ {
		h := 100 + i
		bt := baseTime.Add(time.Duration(i*3) * time.Second)
		ctx = ctx.WithBlockHeight(h).WithBlockHeader(cmtproto.Header{Time: bt, Height: h})
		require.NoError(ck.StoreBlockTime(ctx))
	}

	queryCtx := ctx.WithBlockHeight(10000).WithBlockHeader(cmtproto.Header{Height: 10000})
	cutoffTime := baseTime.Add(10 * time.Second)

	resp, err := clerkKeeper.NewQueryServer(&ck).GetRecordListWithTime(queryCtx, &types.RecordListWithTimeRequest{
		FromId:     1,
		ToTime:     cutoffTime,
		Pagination: query.PageRequest{Limit: 10, Key: []byte{0x00}},
	})
	require.NoError(err)
	require.NotNil(resp)
	require.NotEmpty(resp.EventRecords, "should resolve via indexed block time even with zero query block time")
}

// TestGetRecordListWithTime_Deterministic_StabilityGate verifies that when the
// latest indexed block time <= cutoff, the post-HF path returns an empty
// response (height not yet frozen).
func (s *KeeperTestSuite) TestGetRecordListWithTime_Deterministic_StabilityGate() {
	ctx, ck := s.ctx, s.keeper
	require := s.Require()
	defer enableVisibilityTimeForTest(s, 1)()

	baseTime := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

	ctx = ctx.WithBlockHeight(100).WithBlockHeader(cmtproto.Header{
		Time:   baseTime.Add(10 * time.Minute),
		Height: 100,
	})
	require.NoError(ck.StoreBlockTime(ctx))

	cutoff := baseTime.Add(10 * time.Minute)
	ctx = ctx.WithBlockHeight(200).WithBlockHeader(cmtproto.Header{
		Time:   cutoff,
		Height: 200,
	})

	resp, err := clerkKeeper.NewQueryServer(&ck).GetRecordListWithTime(ctx, &types.RecordListWithTimeRequest{
		FromId:     1,
		ToTime:     cutoff,
		Pagination: query.PageRequest{Limit: 10, Key: []byte{0x00}},
	})
	require.NoError(err)
	require.NotNil(resp)
	require.Empty(resp.EventRecords, "stability gate: empty when latestIndexedTime <= cutoff")
}

// TestGetRecordListWithTime_Deterministic_NoEvents verifies empty response when
// no events exist at the resolved height.
func (s *KeeperTestSuite) TestGetRecordListWithTime_Deterministic_NoEvents() {
	ctx, ck := s.ctx, s.keeper
	require := s.Require()
	defer enableVisibilityTimeForTest(s, 1)()

	baseTime := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

	ctx = ctx.WithBlockHeight(100).WithBlockHeader(cmtproto.Header{Time: baseTime, Height: 100})
	require.NoError(ck.StoreBlockTime(ctx))
	ctx = ctx.WithBlockHeight(101).WithBlockHeader(cmtproto.Header{Time: baseTime.Add(2 * time.Minute), Height: 101})
	require.NoError(ck.StoreBlockTime(ctx))

	cutoff := baseTime.Add(time.Minute)
	ctx = ctx.WithBlockHeight(10000).WithBlockHeader(cmtproto.Header{
		Time:   cutoff.Add(2 * time.Minute),
		Height: 10000,
	})

	resp, err := clerkKeeper.NewQueryServer(&ck).GetRecordListWithTime(ctx, &types.RecordListWithTimeRequest{
		FromId:     1,
		ToTime:     cutoff,
		Pagination: query.PageRequest{Limit: 10, Key: []byte{0x00}},
	})
	require.NoError(err)
	require.NotNil(resp)
	require.Empty(resp.EventRecords, "no events in store means empty response")
}

// TestGetRecordListWithTime_Deterministic_HeightResolutionFails verifies that
// when only blocks AFTER the cutoff exist (stability gate passes, but
// GetBlockHeightByTime returns ErrNoBlockFound) the handler returns empty
// rather than erroring.
func (s *KeeperTestSuite) TestGetRecordListWithTime_Deterministic_HeightResolutionFails() {
	ctx, ck := s.ctx, s.keeper
	require := s.Require()
	defer enableVisibilityTimeForTest(s, 1)()

	baseTime := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

	ctx = ctx.WithBlockHeight(300).WithBlockHeader(cmtproto.Header{
		Time:   baseTime.Add(2 * time.Minute),
		Height: 300,
	})
	require.NoError(ck.StoreBlockTime(ctx))

	cutoff := baseTime.Add(time.Minute)
	ctx = ctx.WithBlockHeight(10000).WithBlockHeader(cmtproto.Header{
		Time:   cutoff.Add(3 * time.Minute),
		Height: 10000,
	})

	resp, err := clerkKeeper.NewQueryServer(&ck).GetRecordListWithTime(ctx, &types.RecordListWithTimeRequest{
		FromId:     1,
		ToTime:     cutoff,
		Pagination: query.PageRequest{Limit: 10, Key: []byte{0x00}},
	})
	require.NoError(err, "height resolution failure should return empty, not error")
	require.NotNil(resp)
	require.Empty(resp.EventRecords)
}

// TestGetRecordListWithTime_OffsetExceedsMax verifies the offset upper bound
// validation (added to bound scans in the post-HF visibility path where
// non-monotonic ordering prevents early termination).
func (s *KeeperTestSuite) TestGetRecordListWithTime_OffsetExceedsMax() {
	ctx, require := s.ctx, s.Require()

	resp, err := s.queryClient.GetRecordListWithTime(ctx, &types.RecordListWithTimeRequest{
		FromId: 1,
		ToTime: time.Now().UTC(),
		Pagination: query.PageRequest{
			Limit:  10,
			Offset: clerkKeeper.MaxRecordListOffset + 1,
			Key:    []byte{0x00},
		},
	})
	require.Error(err)
	require.Nil(resp)
	require.Equal(codes.InvalidArgument, status.Code(err))
}
