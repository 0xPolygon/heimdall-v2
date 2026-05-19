package helper

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// stubHTTPFetcher implements parityHTTPFetcher with a configurable call sequence.
type stubHTTPFetcher struct {
	calls []*ethTypes.Header // returned in order; nil entry means return an error
	errs  []error            // parallel error slice; takes priority when non-nil
	idx   int
}

func (s *stubHTTPFetcher) HeaderByNumber(_ context.Context, _ *big.Int) (*ethTypes.Header, error) {
	if s.idx >= len(s.calls) {
		return nil, errors.New("stub: no more responses configured")
	}
	h := s.calls[s.idx]
	var e error
	if s.idx < len(s.errs) {
		e = s.errs[s.idx]
	}
	s.idx++
	return h, e
}

// stubGRPCFetcher implements parityGRPCFetcher with a single fixed response.
type stubGRPCFetcher struct {
	header *ethTypes.Header
	err    error
}

func (s *stubGRPCFetcher) HeaderByNumber(_ context.Context, _ int64) (*ethTypes.Header, error) {
	return s.header, s.err
}

// makeHeader is a minimal helper that produces a non-zero ethTypes.Header with
// a specific block number. Two headers with different numbers will have
// different Hash() values because Hash covers the Number field.
func makeHeader(num int64) *ethTypes.Header {
	return &ethTypes.Header{
		Number:     big.NewInt(num),
		Difficulty: big.NewInt(1),
	}
}

// TestResolveParityTargetHeight covers all three logical branches:
//   - HTTP error / nil → (0, false)
//   - Chain too young (latestNum < borGRPCParityDepth) → (0, false)
//   - Happy path → (latestNum - borGRPCParityDepth, true)
func TestResolveParityTargetHeight(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("http_error_returns_false", func(t *testing.T) {
		t.Parallel()

		http := &stubHTTPFetcher{
			calls: []*ethTypes.Header{nil},
			errs:  []error{errors.New("rpc error")},
		}
		num, ok := resolveParityTargetHeight(ctx, http)
		require.False(t, ok)
		require.Equal(t, int64(0), num)
	})

	t.Run("nil_response_returns_false", func(t *testing.T) {
		t.Parallel()

		http := &stubHTTPFetcher{
			calls: []*ethTypes.Header{nil},
		}
		num, ok := resolveParityTargetHeight(ctx, http)
		require.False(t, ok)
		require.Equal(t, int64(0), num)
	})

	t.Run("chain_too_young_returns_false", func(t *testing.T) {
		t.Parallel()

		// latestNum=10 < borGRPCParityDepth=32 → too young.
		http := &stubHTTPFetcher{calls: []*ethTypes.Header{makeHeader(10)}}
		num, ok := resolveParityTargetHeight(ctx, http)
		require.False(t, ok)
		require.Equal(t, int64(0), num)
	})

	t.Run("exactly_at_depth_boundary_is_allowed", func(t *testing.T) {
		t.Parallel()

		http := &stubHTTPFetcher{calls: []*ethTypes.Header{makeHeader(borGRPCParityDepth)}}
		num, ok := resolveParityTargetHeight(ctx, http)
		require.True(t, ok)
		require.Equal(t, int64(0), num)
	})

	t.Run("one_below_depth_boundary_returns_false", func(t *testing.T) {
		t.Parallel()

		// latestNum = borGRPCParityDepth - 1 < borGRPCParityDepth → too young.
		http := &stubHTTPFetcher{calls: []*ethTypes.Header{makeHeader(borGRPCParityDepth - 1)}}
		num, ok := resolveParityTargetHeight(ctx, http)
		require.False(t, ok)
		require.Equal(t, int64(0), num)
	})

	t.Run("happy_path_returns_target_minus_depth", func(t *testing.T) {
		t.Parallel()

		// latestNum=100 > 32 → target = 100 - 32 = 68.
		http := &stubHTTPFetcher{calls: []*ethTypes.Header{makeHeader(100)}}
		num, ok := resolveParityTargetHeight(ctx, http)
		require.True(t, ok)
		require.Equal(t, int64(68), num)
	})
}

// TestParityConstants verifies the parity configuration constants.
func TestParityConstants(t *testing.T) {
	t.Parallel()

	require.Equal(t, 5*time.Second, borGRPCParityRetryInterval,
		"borGRPCParityRetryInterval must be exactly 5s")
	require.Equal(t, 60, borGRPCParityMaxAttempts,
		"borGRPCParityMaxAttempts must be 60")
	require.Equal(t, int64(32), borGRPCParityDepth,
		"borGRPCParityDepth must be 32")
	require.Equal(t, 3, borGRPCParityMismatchStreak,
		"borGRPCParityMismatchStreak must be 3")
}

// TestRunBorGRPCHashParityCheckWith covers the loop logic: early exit on success,
// fatal trigger on mismatch streak, and streak reset on transient.
func TestRunBorGRPCHashParityCheckWith(t *testing.T) {
	t.Parallel()

	const smallTimeout = 5 * time.Second

	t.Run("exits_early_on_first_success", func(t *testing.T) {
		t.Parallel()

		h := makeHeader(100)
		http := &stubHTTPFetcher{calls: []*ethTypes.Header{h, h, h, h, h, h}}
		grpc := &stubGRPCFetcher{header: h}

		fatalCalled := false
		runBorGRPCHashParityCheckWith(http, grpc, smallTimeout, 2, 0, func(msg string, _ ...interface{}) {
			fatalCalled = true
		})

		require.False(t, fatalCalled, "success on first attempt must not trigger fatal")
		// Exactly 3 HTTP calls consumed (1 for latest + 2 for stability check).
		// If the early-return is removed, the second attempt consumes 3 more (idx=6).
		require.Equal(t, 3, http.idx, "early-return on ok must exit after exactly one attempt")
	})

	t.Run("max_attempts_one_runs_exactly_once", func(t *testing.T) {
		t.Parallel()

		http := &stubHTTPFetcher{
			calls: []*ethTypes.Header{nil},
			errs:  []error{errors.New("transient")},
		}

		fatalCalled := false
		runBorGRPCHashParityCheckWith(http, &stubGRPCFetcher{header: makeHeader(50)}, smallTimeout, 1, 0, func(msg string, _ ...interface{}) {
			fatalCalled = true
		})

		require.False(t, fatalCalled)
		// One HTTP call was made for the single attempt (resolveParityTargetHeight).
		require.Equal(t, 1, http.idx, "loop body must execute exactly once with maxAttempts=1")
	})

	t.Run("streak_triggers_fatal_after_threshold", func(t *testing.T) {
		t.Parallel()

		// Three consecutive mismatches → streak reaches borGRPCParityMismatchStreak (3) → fatal.
		// Each parity check uses 3 HTTP calls. For 3 checks, we need 9 HTTP calls.
		httpHdr := makeHeader(100)
		grpcHdrDiff := makeHeader(100)
		grpcHdrDiff.GasLimit = 99999

		httpCalls := make([]*ethTypes.Header, 9)
		for i := range httpCalls {
			httpCalls[i] = httpHdr
		}
		http := &stubHTTPFetcher{calls: httpCalls}
		grpc := &stubGRPCFetcher{header: grpcHdrDiff}

		fatalCalled := false
		runBorGRPCHashParityCheckWith(http, grpc, smallTimeout, 10, 0, func(msg string, _ ...interface{}) {
			fatalCalled = true
		})

		require.True(t, fatalCalled, "3 consecutive mismatches must trigger fatal")
	})

	t.Run("transient_resets_streak_no_fatal", func(t *testing.T) {
		t.Parallel()

		httpHdr := makeHeader(100)
		grpcHdrDiff := makeHeader(100)
		grpcHdrDiff.GasLimit = 12345

		calls := make([]*ethTypes.Header, 0)
		errs := make([]error, 0)

		// Attempt 1 (mismatch): calls 1,2,3 all return httpHdr; grpc returns different
		for i := 0; i < 3; i++ {
			calls = append(calls, httpHdr)
			errs = append(errs, nil)
		}
		// Attempt 2 (mismatch): calls 4,5,6
		for i := 0; i < 3; i++ {
			calls = append(calls, httpHdr)
			errs = append(errs, nil)
		}
		// Attempt 3 (transient, the first HTTP call errors):
		calls = append(calls, nil)
		errs = append(errs, errors.New("transient"))
		// Attempt 4 (success): grpc will return the matching header here
		callIdx := 0
		httpFetcher := &funcHTTPFetcher{fn: func(_ context.Context, _ *big.Int) (*ethTypes.Header, error) {
			if callIdx < len(calls) {
				h := calls[callIdx]
				e := errs[callIdx]
				callIdx++
				return h, e
			}
			// Attempt 4: return stable matching header
			return httpHdr, nil
		}}

		// For grpc: different for first 2 mismatches (6 calls), matching for rest
		grpcCallIdx := 0
		grpcFetcher := &funcGRPCFetcher{fn: func(_ context.Context, _ int64) (*ethTypes.Header, error) {
			grpcCallIdx++
			if grpcCallIdx <= 2 {
				return grpcHdrDiff, nil
			}
			return httpHdr, nil
		}}

		fatalCalled := false
		runBorGRPCHashParityCheckWith(httpFetcher, grpcFetcher, smallTimeout, 10, 0, func(msg string, _ ...interface{}) {
			fatalCalled = true
		})
		require.False(t, fatalCalled, "streak reset by transient must prevent fatal")
	})

	t.Run("exhausted_attempts_does_not_fatal", func(t *testing.T) {
		t.Parallel()

		// All attempts return transient (http error) → mismatches stays 0, loop exhausts.
		http := &stubHTTPFetcher{calls: make([]*ethTypes.Header, 0)} // will always error

		fatalCalled := false
		runBorGRPCHashParityCheckWith(http, &stubGRPCFetcher{header: makeHeader(50)}, smallTimeout, 3, 0, func(msg string, _ ...interface{}) {
			fatalCalled = true
		})
		require.False(t, fatalCalled, "transient-only attempts must not trigger fatal even after exhaustion")
	})
}

// funcHTTPFetcher is a test helper for runBorGRPCHashParityCheckWith that
// uses a function closure to produce arbitrary sequential responses.
type funcHTTPFetcher struct {
	fn func(ctx context.Context, number *big.Int) (*ethTypes.Header, error)
}

func (f *funcHTTPFetcher) HeaderByNumber(ctx context.Context, number *big.Int) (*ethTypes.Header, error) {
	return f.fn(ctx, number)
}

// funcGRPCFetcher is the gRPC equivalent.
type funcGRPCFetcher struct {
	fn func(ctx context.Context, blockID int64) (*ethTypes.Header, error)
}

func (f *funcGRPCFetcher) HeaderByNumber(ctx context.Context, blockID int64) (*ethTypes.Header, error) {
	return f.fn(ctx, blockID)
}

// TestCheckBorGRPCHashParityOnceWith covers all return paths of the
// testable core.
func TestCheckBorGRPCHashParityOnceWith(t *testing.T) {
	t.Parallel()

	timeout := 5 * time.Second

	t.Run("chain_too_young_returns_false_false", func(t *testing.T) {
		t.Parallel()

		h := makeHeader(5)
		http := &stubHTTPFetcher{calls: []*ethTypes.Header{h, h, h}} // call1=latest; calls2&3=stable check
		grpc := &stubGRPCFetcher{header: h}

		ok, mismatch := checkBorGRPCHashParityOnceWith(http, grpc, timeout)
		require.False(t, ok)
		require.False(t, mismatch)
	})

	t.Run("unstable_http_returns_false_false", func(t *testing.T) {
		t.Parallel()

		h := makeHeader(100)
		http := &stubHTTPFetcher{
			calls: []*ethTypes.Header{h, h, nil},
			errs:  []error{nil, nil, errors.New("network error")},
		}
		grpc := &stubGRPCFetcher{header: h}

		ok, mismatch := checkBorGRPCHashParityOnceWith(http, grpc, timeout)
		require.False(t, ok)
		require.False(t, mismatch)
	})

	t.Run("hash_mismatch_returns_false_true", func(t *testing.T) {
		t.Parallel()

		httpHdr := makeHeader(100)
		grpcHdrDiff := makeHeader(100)
		grpcHdrDiff.GasLimit = 12345 // different field → different hash

		http := &stubHTTPFetcher{calls: []*ethTypes.Header{httpHdr, httpHdr, httpHdr}}
		grpc := &stubGRPCFetcher{header: grpcHdrDiff}

		ok, mismatch := checkBorGRPCHashParityOnceWith(http, grpc, timeout)
		require.False(t, ok)
		require.True(t, mismatch, "differing hashes must be reported as a mismatch")
	})

	t.Run("hash_match_returns_true_false", func(t *testing.T) {
		t.Parallel()

		httpHdr := makeHeader(100)
		grpcHdr := makeHeader(100) // same fields → same hash

		http := &stubHTTPFetcher{calls: []*ethTypes.Header{httpHdr, httpHdr, httpHdr}}
		grpc := &stubGRPCFetcher{header: grpcHdr}

		ok, mismatch := checkBorGRPCHashParityOnceWith(http, grpc, timeout)
		require.True(t, ok, "matching hashes must return ok=true")
		require.False(t, mismatch)
	})
}

// TestFetchStableHeadersAtHeight covers all stability / error branches.
func TestFetchStableHeadersAtHeight(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	targetNum := int64(50)

	t.Run("first_http_error_returns_not_stable", func(t *testing.T) {
		t.Parallel()

		h := makeHeader(50)
		http := &stubHTTPFetcher{
			calls: []*ethTypes.Header{h, h, h}, // non-nil + error on call 1; calls 2,3 for mutant path
			errs:  []error{errors.New("http error"), nil, nil},
		}
		grpc := &stubGRPCFetcher{header: h}

		httpHdr, grpcHdr, stable := fetchStableHeadersAtHeight(ctx, http, grpc, targetNum)
		require.False(t, stable)
		require.Nil(t, httpHdr)
		require.Nil(t, grpcHdr)
	})

	t.Run("grpc_error_returns_not_stable", func(t *testing.T) {
		t.Parallel()

		http := &stubHTTPFetcher{calls: []*ethTypes.Header{makeHeader(50), makeHeader(50)}}
		grpc := &stubGRPCFetcher{err: errors.New("grpc error")}

		_, _, stable := fetchStableHeadersAtHeight(ctx, http, grpc, targetNum)
		require.False(t, stable)
	})

	t.Run("reorg_during_check_returns_not_stable", func(t *testing.T) {
		t.Parallel()

		// First HTTP read returns header A; second HTTP read returns header B (different hash).
		headerA := makeHeader(50)
		// Modify B slightly so its hash differs from A.
		headerB := makeHeader(50)
		headerB.GasLimit = 999 // same number but different hash

		http := &stubHTTPFetcher{calls: []*ethTypes.Header{headerA, headerB}}
		grpc := &stubGRPCFetcher{header: makeHeader(50)}

		_, _, stable := fetchStableHeadersAtHeight(ctx, http, grpc, targetNum)
		require.False(t, stable, "diverging http re-reads indicate a reorg; must return not stable")
	})

	t.Run("all_agree_returns_stable", func(t *testing.T) {
		t.Parallel()

		// Both HTTP reads return the same header; gRPC also returns a header.
		h := makeHeader(50)
		http := &stubHTTPFetcher{calls: []*ethTypes.Header{h, h}}
		grpc := &stubGRPCFetcher{header: makeHeader(50)}

		httpHdr, grpcHdr, stable := fetchStableHeadersAtHeight(ctx, http, grpc, targetNum)
		require.True(t, stable)
		require.NotNil(t, httpHdr)
		require.NotNil(t, grpcHdr)
	})

	t.Run("nil_grpc_response_returns_not_stable", func(t *testing.T) {
		t.Parallel()

		http := &stubHTTPFetcher{calls: []*ethTypes.Header{makeHeader(50), makeHeader(50)}}
		grpc := &stubGRPCFetcher{header: nil}

		_, _, stable := fetchStableHeadersAtHeight(ctx, http, grpc, targetNum)
		require.False(t, stable)
	})
}
