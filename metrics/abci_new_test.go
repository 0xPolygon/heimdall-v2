package metrics

import (
	"testing"
	"time"

	dto "github.com/prometheus/client_model/go"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

// histogramSampleCount returns the number of observations recorded so far in
// a plain (non-vector) prometheus.Histogram, by reading its internal dto.Metric.
func histogramSampleCount(t *testing.T, h interface{ Write(*dto.Metric) error }) uint64 {
	t.Helper()
	m := &dto.Metric{}
	require.NoError(t, h.Write(m))
	return m.GetHistogram().GetSampleCount()
}

// TestRecordExtendVoteElapsed verifies that RecordExtendVoteElapsed observes a
// sample into the correct phase's histogram bucket.
func TestRecordExtendVoteElapsed(t *testing.T) {
	phase := "unit_test_phase_extend_vote_elapsed"

	before := testutil.CollectAndCount(ExtendVoteElapsedSeconds)

	RecordExtendVoteElapsed(phase, time.Now().Add(-5*time.Millisecond))

	after := testutil.CollectAndCount(ExtendVoteElapsedSeconds)
	require.Equal(t, before+1, after, "RecordExtendVoteElapsed must add exactly one observation")
}

// TestRecordMilestoneNoNewHeaders verifies that RecordMilestoneNoNewHeaders
// increments the no-new-headers counter.
func TestRecordMilestoneNoNewHeaders(t *testing.T) {
	before := testutil.ToFloat64(MilestoneNoNewHeadersTotal)

	RecordMilestoneNoNewHeaders()

	after := testutil.ToFloat64(MilestoneNoNewHeadersTotal)
	require.Equal(t, before+1, after, "RecordMilestoneNoNewHeaders must increment the counter by exactly one")
}

// TestRecordMilestoneMajorityFound verifies that RecordMilestoneMajorityFound
// increments the counter for the given threshold label only.
func TestRecordMilestoneMajorityFound(t *testing.T) {
	threshold := "unit_test_threshold_majority_found"

	before := testutil.ToFloat64(MilestoneMajorityFoundTotal.WithLabelValues(threshold))

	RecordMilestoneMajorityFound(threshold)

	after := testutil.ToFloat64(MilestoneMajorityFoundTotal.WithLabelValues(threshold))
	require.Equal(t, before+1, after, "RecordMilestoneMajorityFound must increment the counter for the given threshold by exactly one")
}

// TestRecordVoteExtensionRejected verifies that RecordVoteExtensionRejected
// increments the counter for the given reason label only.
func TestRecordVoteExtensionRejected(t *testing.T) {
	reason := "unit_test_reason_vote_extension_rejected"

	before := testutil.ToFloat64(VoteExtensionRejectedTotal.WithLabelValues(reason))

	RecordVoteExtensionRejected(reason)

	after := testutil.ToFloat64(VoteExtensionRejectedTotal.WithLabelValues(reason))
	require.Equal(t, before+1, after, "RecordVoteExtensionRejected must increment the counter for the given reason by exactly one")
}

// TestRecordBorRPCCallDuration verifies that RecordBorRPCCallDuration observes
// a sample into the correct method's histogram bucket.
func TestRecordBorRPCCallDuration(t *testing.T) {
	method := "unit_test_method_bor_rpc_call_duration"

	before := testutil.CollectAndCount(BorRPCCallDuration)

	RecordBorRPCCallDuration(method, time.Now().Add(-5*time.Millisecond))

	after := testutil.CollectAndCount(BorRPCCallDuration)
	require.Equal(t, before+1, after, "RecordBorRPCCallDuration must add exactly one observation")
}

// TestRecordMilestoneMajorityCommitSuppressed verifies that
// RecordMilestoneMajorityCommitSuppressed increments the counter for the
// given reason label only.
func TestRecordMilestoneMajorityCommitSuppressed(t *testing.T) {
	reason := "unit_test_reason_majority_commit_suppressed"

	before := testutil.ToFloat64(MilestoneMajorityCommitSuppressedTotal.WithLabelValues(reason))

	RecordMilestoneMajorityCommitSuppressed(reason)

	after := testutil.ToFloat64(MilestoneMajorityCommitSuppressedTotal.WithLabelValues(reason))
	require.Equal(t, before+1, after, "RecordMilestoneMajorityCommitSuppressed must increment the counter for the given reason by exactly one")
}

// TestRecordMilestoneGenerationDuration verifies that
// RecordMilestoneGenerationDuration observes a sample into the
// milestone-proposition-generation histogram.
func TestRecordMilestoneGenerationDuration(t *testing.T) {
	before := histogramSampleCount(t, MilestoneGenerationDuration)

	RecordMilestoneGenerationDuration(time.Now().Add(-5 * time.Millisecond))

	after := histogramSampleCount(t, MilestoneGenerationDuration)
	require.Equal(t, before+1, after, "RecordMilestoneGenerationDuration must add exactly one observation")
}
