package keeper_test

import (
	"context"
	"time"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

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

// ---------------------------------------------------------------------------
// GetStateSyncsByTime gRPC handler tests
// ---------------------------------------------------------------------------

// TestGetStateSyncsByTime_HappyPath verifies the combined endpoint returns events
// when the stability gate passes (blockTime > cutoff) and a valid height resolves.
func (s *KeeperTestSuite) TestGetStateSyncsByTime_HappyPath() {
	ctx, ck := s.ctx, s.keeper
	require := s.Require()

	baseTime := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

	// Set upgrade boundary and create events with visibility heights
	require.NoError(ck.SetVisibilityTimeUpgradeID(ctx, 1))

	for i := uint64(1); i <= 3; i++ {
		rec := types.NewEventRecord(TxHash1, i, i, Address1, make([]byte, 1), "1",
			baseTime.Add(time.Duration(i)*time.Minute))
		require.NoError(ck.SetEventRecord(ctx, rec))
		require.NoError(ck.VisibilityHeightByID.Set(ctx, i, 100+i))
	}

	// Store block times so GetBlockHeightByTime can resolve
	for i := int64(0); i < 5; i++ {
		h := 100 + i
		bt := baseTime.Add(time.Duration(i*3) * time.Second)
		ctx = ctx.WithBlockHeight(h).WithBlockHeader(cmtproto.Header{Time: bt, Height: h})
		require.NoError(ck.StoreBlockTime(ctx))
	}

	// Set current block well past the cutoff so stability gate passes and
	// HeimdallHeight <= currentHeight validation succeeds inside the delegate.
	cutoffTime := baseTime.Add(10 * time.Minute)
	ctx = ctx.WithBlockHeight(10000).WithBlockHeader(cmtproto.Header{
		Time:   cutoffTime.Add(time.Minute), // blockTime > cutoff
		Height: 10000,
	})

	queryServer := clerkKeeper.NewQueryServer(&ck)
	resp, err := queryServer.(interface {
		GetStateSyncsByTime(context.Context, *types.StateSyncsByTimeRequest) (*types.StateSyncsByTimeResponse, error)
	}).GetStateSyncsByTime(ctx, &types.StateSyncsByTimeRequest{
		FromId:     1,
		ToTime:     cutoffTime,
		Pagination: query.PageRequest{Limit: 10, Key: []byte{0x00}},
	})
	require.NoError(err)
	require.NotNil(resp)
	require.NotEmpty(resp.EventRecords, "should return events when stability gate passes")
	require.Greater(resp.HeimdallHeight, int64(0), "should return a resolved height")
}

// TestGetStateSyncsByTime_StabilityGate verifies that when blockTime <= cutoff,
// the stability gate returns an empty response (height not yet frozen).
func (s *KeeperTestSuite) TestGetStateSyncsByTime_StabilityGate() {
	ctx, ck := s.ctx, s.keeper
	require := s.Require()

	baseTime := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

	// Create a record
	rec := types.NewEventRecord(TxHash1, 1, 1, Address1, make([]byte, 1), "1",
		baseTime.Add(-time.Minute))
	require.NoError(ck.SetEventRecord(ctx, rec))

	// Set blockTime equal to the cutoff (gate should fire: blockTime <= cutoff)
	cutoff := baseTime.Add(10 * time.Minute)
	ctx = ctx.WithBlockHeight(200).WithBlockHeader(cmtproto.Header{
		Time:   cutoff, // blockTime == cutoff
		Height: 200,
	})

	queryServer := clerkKeeper.NewQueryServer(&ck)
	resp, err := queryServer.(interface {
		GetStateSyncsByTime(context.Context, *types.StateSyncsByTimeRequest) (*types.StateSyncsByTimeResponse, error)
	}).GetStateSyncsByTime(ctx, &types.StateSyncsByTimeRequest{
		FromId:     1,
		ToTime:     cutoff,
		Pagination: query.PageRequest{Limit: 10, Key: []byte{0x00}},
	})
	require.NoError(err)
	require.NotNil(resp)
	require.Empty(resp.EventRecords, "stability gate: empty when blockTime <= cutoff")
	require.Equal(int64(0), resp.HeimdallHeight, "no height resolved when stability gate fires")
}

// TestGetStateSyncsByTime_NoEvents verifies empty response when no events exist.
func (s *KeeperTestSuite) TestGetStateSyncsByTime_NoEvents() {
	ctx, ck := s.ctx, s.keeper
	require := s.Require()

	baseTime := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

	// Store a block time so height resolution succeeds
	ctx = ctx.WithBlockHeight(100).WithBlockHeader(cmtproto.Header{Time: baseTime, Height: 100})
	require.NoError(ck.StoreBlockTime(ctx))

	// Set current block past the cutoff
	cutoff := baseTime.Add(time.Minute)
	ctx = ctx.WithBlockHeight(10000).WithBlockHeader(cmtproto.Header{
		Time:   cutoff.Add(time.Minute),
		Height: 10000,
	})

	queryServer := clerkKeeper.NewQueryServer(&ck)
	resp, err := queryServer.(interface {
		GetStateSyncsByTime(context.Context, *types.StateSyncsByTimeRequest) (*types.StateSyncsByTimeResponse, error)
	}).GetStateSyncsByTime(ctx, &types.StateSyncsByTimeRequest{
		FromId:     1,
		ToTime:     cutoff,
		Pagination: query.PageRequest{Limit: 10, Key: []byte{0x00}},
	})
	require.NoError(err)
	require.NotNil(resp)
	require.Empty(resp.EventRecords, "no events in store means empty response")
}

// TestGetStateSyncsByTime_InvalidArgs verifies validation errors for missing fields.
func (s *KeeperTestSuite) TestGetStateSyncsByTime_InvalidArgs() {
	ctx := s.ctx
	require := s.Require()

	queryServer := clerkKeeper.NewQueryServer(&s.keeper)
	qs := queryServer.(interface {
		GetStateSyncsByTime(context.Context, *types.StateSyncsByTimeRequest) (*types.StateSyncsByTimeResponse, error)
	})

	now := time.Now().UTC()

	tests := []struct {
		name string
		req  *types.StateSyncsByTimeRequest
	}{
		{
			name: "nil request",
			req:  nil,
		},
		{
			name: "missing from_id (0)",
			req: &types.StateSyncsByTimeRequest{
				FromId:     0,
				ToTime:     now,
				Pagination: query.PageRequest{Limit: 10, Key: []byte{0x00}},
			},
		},
		{
			name: "missing to_time (zero)",
			req: &types.StateSyncsByTimeRequest{
				FromId:     1,
				ToTime:     time.Time{},
				Pagination: query.PageRequest{Limit: 10, Key: []byte{0x00}},
			},
		},
		{
			name: "explicit unix epoch to_time",
			req: &types.StateSyncsByTimeRequest{
				FromId:     1,
				ToTime:     time.Unix(0, 0).UTC(),
				Pagination: query.PageRequest{Limit: 10, Key: []byte{0x00}},
			},
		},
		{
			name: "empty pagination",
			req: &types.StateSyncsByTimeRequest{
				FromId: 1,
				ToTime: now,
			},
		},
		{
			name: "limit 0",
			req: &types.StateSyncsByTimeRequest{
				FromId:     1,
				ToTime:     now,
				Pagination: query.PageRequest{Limit: 0, Key: []byte{0x00}},
			},
		},
		{
			name: "limit exceeds max",
			req: &types.StateSyncsByTimeRequest{
				FromId:     1,
				ToTime:     now,
				Pagination: query.PageRequest{Limit: 100, Key: []byte{0x00}},
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			resp, err := qs.GetStateSyncsByTime(ctx, tc.req)
			require.Error(err, "should reject invalid request: %s", tc.name)
			require.Nil(resp)
		})
	}
}

func (s *KeeperTestSuite) TestGetStateSyncsByTime_RejectsUnixEpochTimestamp() {
	ctx := s.ctx
	require := s.Require()

	queryServer := clerkKeeper.NewQueryServer(&s.keeper)
	qs := queryServer.(interface {
		GetStateSyncsByTime(context.Context, *types.StateSyncsByTimeRequest) (*types.StateSyncsByTimeResponse, error)
	})

	resp, err := qs.GetStateSyncsByTime(ctx, &types.StateSyncsByTimeRequest{
		FromId:     1,
		ToTime:     time.Unix(0, 0).UTC(),
		Pagination: query.PageRequest{Limit: 10, Key: []byte{0x00}},
	})
	require.Error(err)
	require.Nil(resp)
	require.Equal(codes.InvalidArgument, status.Code(err))
}

// TestGetStateSyncsByTime_HeightResolutionFails verifies that when no blocks
// exist in the index, the handler returns empty (not an error).
func (s *KeeperTestSuite) TestGetStateSyncsByTime_HeightResolutionFails() {
	ctx := s.ctx
	require := s.Require()

	baseTime := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

	// No blocks stored in the index; set blockTime past cutoff so stability gate passes
	cutoff := baseTime.Add(time.Minute)
	ctx = ctx.WithBlockHeight(200).WithBlockHeader(cmtproto.Header{
		Time:   cutoff.Add(time.Minute),
		Height: 200,
	})

	queryServer := clerkKeeper.NewQueryServer(&s.keeper)
	resp, err := queryServer.(interface {
		GetStateSyncsByTime(context.Context, *types.StateSyncsByTimeRequest) (*types.StateSyncsByTimeResponse, error)
	}).GetStateSyncsByTime(ctx, &types.StateSyncsByTimeRequest{
		FromId:     1,
		ToTime:     cutoff,
		Pagination: query.PageRequest{Limit: 10, Key: []byte{0x00}},
	})
	require.NoError(err, "height resolution failure (no blocks) should return empty, not error")
	require.NotNil(resp)
	require.Empty(resp.EventRecords)
}

// TestGetStateSyncsByTime_DelegatesToVisibleAtHeight verifies that the combined
// endpoint returns the same events as a direct GetRecordListVisibleAtHeight call.
func (s *KeeperTestSuite) TestGetStateSyncsByTime_DelegatesToVisibleAtHeight() {
	ctx, ck := s.ctx, s.keeper
	require := s.Require()

	baseTime := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

	require.NoError(ck.SetVisibilityTimeUpgradeID(ctx, 1))

	// Create events with visibility heights
	for i := uint64(1); i <= 4; i++ {
		rec := types.NewEventRecord(TxHash1, i, i, Address1, make([]byte, 1), "1",
			baseTime.Add(time.Duration(i)*time.Minute))
		require.NoError(ck.SetEventRecord(ctx, rec))
		require.NoError(ck.VisibilityHeightByID.Set(ctx, i, 100+i))
	}

	// Store block times at heights 100-104
	for i := int64(0); i < 5; i++ {
		h := 100 + i
		bt := baseTime.Add(time.Duration(i*3) * time.Second)
		ctx = ctx.WithBlockHeight(h).WithBlockHeader(cmtproto.Header{Time: bt, Height: h})
		require.NoError(ck.StoreBlockTime(ctx))
	}

	cutoff := baseTime.Add(10 * time.Minute)
	ctx = ctx.WithBlockHeight(10000).WithBlockHeader(cmtproto.Header{
		Time:   cutoff.Add(time.Minute),
		Height: 10000,
	})

	queryServer := clerkKeeper.NewQueryServer(&ck)

	// Call GetStateSyncsByTime
	combinedResp, err := queryServer.(interface {
		GetStateSyncsByTime(context.Context, *types.StateSyncsByTimeRequest) (*types.StateSyncsByTimeResponse, error)
	}).GetStateSyncsByTime(ctx, &types.StateSyncsByTimeRequest{
		FromId:     1,
		ToTime:     cutoff,
		Pagination: query.PageRequest{Limit: 10, Key: []byte{0x00}},
	})
	require.NoError(err)
	require.NotNil(combinedResp)

	// Call GetRecordListVisibleAtHeight with the resolved height
	directResp, err := queryServer.(interface {
		GetRecordListVisibleAtHeight(context.Context, *types.RecordListVisibleAtHeightRequest) (*types.RecordListVisibleAtHeightResponse, error)
	}).GetRecordListVisibleAtHeight(ctx, &types.RecordListVisibleAtHeightRequest{
		FromId:         1,
		HeimdallHeight: combinedResp.HeimdallHeight,
		ToTime:         cutoff,
		Pagination:     query.PageRequest{Limit: 10, Key: []byte{0x00}},
	})
	require.NoError(err)
	require.NotNil(directResp)

	// Both should return identical event sets
	require.Equal(len(directResp.EventRecords), len(combinedResp.EventRecords),
		"combined endpoint must return same events as direct visible-at-height query")
	for i := range combinedResp.EventRecords {
		require.Equal(directResp.EventRecords[i].Id, combinedResp.EventRecords[i].Id)
	}
}

// ---------------------------------------------------------------------------
// GetBlockHeightByTime gRPC handler tests
// ---------------------------------------------------------------------------

// TestGRPC_GetBlockHeightByTime_HappyPath verifies correct height resolution.
func (s *KeeperTestSuite) TestGRPC_GetBlockHeightByTime_HappyPath() {
	ctx, ck := s.ctx, s.keeper
	require := s.Require()

	baseTime := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

	// Store block times at heights 100-104
	for i := int64(0); i < 5; i++ {
		h := 100 + i
		bt := baseTime.Add(time.Duration(i*3) * time.Second)
		ctx = ctx.WithBlockHeight(h).WithBlockHeader(cmtproto.Header{Time: bt, Height: h})
		require.NoError(ck.StoreBlockTime(ctx))
	}

	queryServer := clerkKeeper.NewQueryServer(&ck)
	qs := queryServer.(interface {
		GetBlockHeightByTime(context.Context, *types.BlockHeightByTimeRequest) (*types.BlockHeightByTimeResponse, error)
	})

	// Cutoff at block 102's time (baseTime + 6s) should return height 102
	resp, err := qs.GetBlockHeightByTime(ctx, &types.BlockHeightByTimeRequest{
		CutoffTime: baseTime.Add(6 * time.Second).Unix(),
	})
	require.NoError(err)
	require.Equal(int64(102), resp.Height)

	// Cutoff between block 102 and 103 should still return 102
	resp, err = qs.GetBlockHeightByTime(ctx, &types.BlockHeightByTimeRequest{
		CutoffTime: baseTime.Add(7 * time.Second).Unix(),
	})
	require.NoError(err)
	require.Equal(int64(102), resp.Height)
}

// TestGRPC_GetBlockHeightByTime_NoBlockFound verifies NotFound when cutoff is
// before any stored block.
func (s *KeeperTestSuite) TestGRPC_GetBlockHeightByTime_NoBlockFound() {
	ctx, ck := s.ctx, s.keeper
	require := s.Require()

	bt := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	ctx = ctx.WithBlockHeight(100).WithBlockHeader(cmtproto.Header{Time: bt, Height: 100})
	require.NoError(ck.StoreBlockTime(ctx))

	queryServer := clerkKeeper.NewQueryServer(&ck)
	qs := queryServer.(interface {
		GetBlockHeightByTime(context.Context, *types.BlockHeightByTimeRequest) (*types.BlockHeightByTimeResponse, error)
	})

	// Cutoff before any block -> NotFound
	resp, err := qs.GetBlockHeightByTime(ctx, &types.BlockHeightByTimeRequest{
		CutoffTime: bt.Add(-time.Second).Unix(),
	})
	require.Error(err)
	require.Nil(resp)
	require.Contains(err.Error(), "no block found")
}

// TestGRPC_GetBlockHeightByTime_InvalidCutoff verifies error for zero and
// negative cutoff values.
func (s *KeeperTestSuite) TestGRPC_GetBlockHeightByTime_InvalidCutoff() {
	ctx := s.ctx
	require := s.Require()

	queryServer := clerkKeeper.NewQueryServer(&s.keeper)
	qs := queryServer.(interface {
		GetBlockHeightByTime(context.Context, *types.BlockHeightByTimeRequest) (*types.BlockHeightByTimeResponse, error)
	})

	tests := []struct {
		name   string
		cutoff int64
	}{
		{"zero cutoff", 0},
		{"negative cutoff", -100},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			resp, err := qs.GetBlockHeightByTime(ctx, &types.BlockHeightByTimeRequest{
				CutoffTime: tc.cutoff,
			})
			require.Error(err, "should reject %s", tc.name)
			require.Nil(resp)
			require.Contains(err.Error(), "cutoff_time must be a positive unix timestamp")
		})
	}
}

// TestGRPC_GetBlockHeightByTime_NilRequest verifies error on nil request.
func (s *KeeperTestSuite) TestGRPC_GetBlockHeightByTime_NilRequest() {
	ctx := s.ctx
	require := s.Require()

	queryServer := clerkKeeper.NewQueryServer(&s.keeper)
	qs := queryServer.(interface {
		GetBlockHeightByTime(context.Context, *types.BlockHeightByTimeRequest) (*types.BlockHeightByTimeResponse, error)
	})

	resp, err := qs.GetBlockHeightByTime(ctx, nil)
	require.Error(err)
	require.Nil(resp)
}
