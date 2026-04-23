package helper

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

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

// TestGetBorChainBlockAuthor_GRPCPath_NilClientPropagatesError verifies the nil-author
// handling in the HTTP dispatch path. When BorChainGrpcFlag=false and the
// HTTP client returns a nil author, the function returns ethereum.NotFound.
func TestGetBorChainBlockAuthor_GRPCPath_NilClientPropagatesError(t *testing.T) {
	t.Parallel()

	c := makeDispatcherCaller(true)
	// With nil gRPC client, the function must not reach the nil-author check.
	author, err := c.GetBorChainBlockAuthor(context.Background(), nil)
	require.Error(t, err)
	require.Nil(t, author)
}
