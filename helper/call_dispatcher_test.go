package helper

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/metrics"
)

// histogramVecSampleCount returns the number of observations recorded so far for one label
// value of metrics.BorRPCCallDuration, by reading its internal dto.Metric. testutil.CollectAndCount
// is not suitable here because it counts distinct label combinations (time series), not the
// number of observations within a single series.
func histogramVecSampleCount(t *testing.T, method string) uint64 {
	t.Helper()
	observer := metrics.BorRPCCallDuration.WithLabelValues(method)
	h, ok := observer.(prometheus.Histogram)
	require.True(t, ok, "HistogramVec.WithLabelValues must return a prometheus.Histogram")
	m := &dto.Metric{}
	require.NoError(t, h.Write(m))
	return m.GetHistogram().GetSampleCount()
}

// makeDispatcherCaller returns a ContractCaller with the given gRPC flag and a
// nil gRPC client. The BorChainTimeout is set to a short value, so HTTP-path
// tests time out quickly without blocking the test suite.
func makeDispatcherCaller(grpcFlag bool) ContractCaller {
	return ContractCaller{
		BorChainGrpcFlag:   grpcFlag,
		BorChainGrpcClient: nil, // nil client triggers getRequiredBorGRPCClient error
		BorChainTimeout:    50 * time.Millisecond,
	}
}

// TestGetBorChainBlockInfoInBatch_GRPCNilClientError verifies that when
// BorChainGrpcFlag=true and BorChainGrpcClient=nil, the function returns an
// error rather than panicking or silently returning empty results.
func TestGetBorChainBlockInfoInBatch_GRPCNilClientError(t *testing.T) {
	t.Parallel()

	c := makeDispatcherCaller(true)
	headers, tds, authors, err := c.GetBorChainBlockInfoInBatch(context.Background(), 1, 5)

	require.Error(t, err, "nil gRPC client must return an error, not succeed")
	require.Nil(t, headers)
	require.Nil(t, tds)
	require.Nil(t, authors)
}

// TestGetBorChainBlockTd_GRPCNilClientError verifies the same for GetBorChainBlockTd.
func TestGetBorChainBlockTd_GRPCNilClientError(t *testing.T) {
	t.Parallel()

	c := makeDispatcherCaller(true)
	td, err := c.GetBorChainBlockTd(context.Background(), common.Hash{})

	require.Error(t, err, "nil gRPC client must return an error")
	require.Equal(t, uint64(0), td)
}

// TestGetBorChainBlockAuthor_GRPCNilClientError verifies the same for GetBorChainBlockAuthor.
func TestGetBorChainBlockAuthor_GRPCNilClientError(t *testing.T) {
	t.Parallel()

	c := makeDispatcherCaller(true)
	author, err := c.GetBorChainBlockAuthor(context.Background(), nil)

	require.Error(t, err, "nil gRPC client must return an error")
	require.Nil(t, author)
}

// TestGetBorChainBlock_GRPCNilClientError verifies that when BorChainGrpcFlag=true and
// BorChainGrpcClient=nil, GetBorChainBlock returns an error rather than panicking or silently
// returning a header, and that it records a Bor RPC call duration observation for
// method="get_bor_chain_block" regardless of the error (the metric is recorded via an
// unconditional defer at function entry).
func TestGetBorChainBlock_GRPCNilClientError(t *testing.T) {
	c := makeDispatcherCaller(true)

	before := histogramVecSampleCount(t, "get_bor_chain_block")

	header, err := c.GetBorChainBlock(context.Background(), nil)

	require.Error(t, err, "nil gRPC client must return an error, not succeed")
	require.Nil(t, header)

	after := histogramVecSampleCount(t, "get_bor_chain_block")
	require.Equal(t, before+1, after,
		"GetBorChainBlock must record a Bor RPC call duration observation for method=get_bor_chain_block even on error")
}

// TestGetBorChainBlockInfoInBatch_NonGRPCCancelledContext verifies that
// BorChainGrpcFlag=false falls through to the HTTP path.
func TestGetBorChainBlockInfoInBatch_NonGRPCCancelledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel so the HTTP call fails immediately

	c := makeDispatcherCaller(false)
	// BorChainClient is nil; calling getBorChainBlockInfoInBatchHTTP will
	// panic or error. We just need the function to NOT return the nil-client
	var (
		panicVal interface{}
		err      error
	)
	func() {
		defer func() { panicVal = recover() }()
		_, _, _, err = c.GetBorChainBlockInfoInBatch(ctx, 1, 5)
	}()

	// Either a panic (from nil client dereference) or a context error is fine —
	// what matters is that the gRPC nil-client error was not returned, which
	// means the gRPC branch was correctly skipped.
	if panicVal == nil {
		// If no panic, should be a network or context error, not a "gRPC client is nil" error
		require.Error(t, err)
		require.NotContains(t, err.Error(), "bor gRPC client is nil")
	}
	// If panicVal != nil, the nil BorChainClient was dereferenced — correct branch was taken.
}
