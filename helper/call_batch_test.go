package helper

import (
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

// TestBuildBorBatchElems_Layout verifies that the exact layout of the returned slice is
//
//	[totalBlocks header elems] ++ [totalBlocks TD elems] ++ [non-genesis author elems]
func TestBuildBorBatchElems_Layout(t *testing.T) {
	t.Parallel()

	t.Run("range_1_to_3_layout", func(t *testing.T) {
		t.Parallel()

		start, end := int64(1), int64(3)
		totalBlocks := end - start + 1 // 3
		hdrs := make([]*ethTypes.Header, totalBlocks)
		tds := make([]*tdResp, totalBlocks)
		authors := make([]*common.Address, totalBlocks)

		elems := buildBorBatchElems(start, end, hdrs, tds, authors)

		// 3 headers + 3 TD + 3 authors (blocks 1,2,3 — all non-genesis)
		require.Len(t, elems, int(3*totalBlocks))

		// Header section: indices 0..2
		for i := int64(0); i < totalBlocks; i++ {
			blockNum := start + i
			expectedHex := fmt.Sprintf("0x%x", blockNum)
			require.Equal(t, "eth_getHeaderByNumber", elems[i].Method)
			require.Equal(t, expectedHex, elems[i].Args[0])
		}

		// TD section: indices 3..5
		for i := int64(0); i < totalBlocks; i++ {
			blockNum := start + i
			expectedHex := fmt.Sprintf("0x%x", blockNum)
			idx := int(totalBlocks) + int(i)
			require.Equal(t, "eth_getTdByNumber", elems[idx].Method)
			require.Equal(t, expectedHex, elems[idx].Args[0])
		}

		// Author section: indices 6..8 (all 3 blocks are non-genesis)
		for i := int64(0); i < totalBlocks; i++ {
			blockNum := start + i
			expectedHex := fmt.Sprintf("0x%x", blockNum)
			idx := 2*int(totalBlocks) + int(i)
			require.Equal(t, "bor_getAuthor", elems[idx].Method)
			require.Equal(t, expectedHex, elems[idx].Args[0])
		}
	})

	t.Run("range_starting_at_genesis_skips_genesis_author", func(t *testing.T) {
		t.Parallel()

		// start=0, end=2 → blocks 0,1,2 → 3 headers, 3 TDs, but only 2 authors (blocks 1,2)
		start, end := int64(0), int64(2)
		totalBlocks := end - start + 1 // 3
		hdrs := make([]*ethTypes.Header, totalBlocks)
		tds := make([]*tdResp, totalBlocks)
		authors := make([]*common.Address, totalBlocks)

		elems := buildBorBatchElems(start, end, hdrs, tds, authors)

		// 3 headers + 3 TD + 2 authors (genesis block 0 is skipped)
		require.Len(t, elems, int(2*totalBlocks+2))

		// Verify genesis (block 0) is not in author section
		authorSection := elems[2*int(totalBlocks):]
		require.Len(t, authorSection, 2)
		require.Equal(t, "0x1", authorSection[0].Args[0], "first author elem must be block 1, not genesis")
		require.Equal(t, "0x2", authorSection[1].Args[0])
	})

	t.Run("single_block_non_genesis", func(t *testing.T) {
		t.Parallel()

		start, end := int64(5), int64(5)
		totalBlocks := int64(1)
		hdrs := make([]*ethTypes.Header, totalBlocks)
		tds := make([]*tdResp, totalBlocks)
		authors := make([]*common.Address, totalBlocks)

		elems := buildBorBatchElems(start, end, hdrs, tds, authors)

		// 1 header + 1 TD + 1 author
		require.Len(t, elems, 3)
		require.Equal(t, "eth_getHeaderByNumber", elems[0].Method)
		require.Equal(t, "eth_getTdByNumber", elems[1].Method)
		require.Equal(t, "bor_getAuthor", elems[2].Method)
		require.Equal(t, "0x5", elems[0].Args[0])
	})

	t.Run("result_pointers_point_into_output_slices", func(t *testing.T) {
		t.Parallel()

		start, end := int64(10), int64(11)
		totalBlocks := end - start + 1
		hdrs := make([]*ethTypes.Header, totalBlocks)
		tds := make([]*tdResp, totalBlocks)
		authors := make([]*common.Address, totalBlocks)

		elems := buildBorBatchElems(start, end, hdrs, tds, authors)

		// Verify that result pointers actually point into the slices by writing
		// through them and checking the output slices.
		hdr10 := &ethTypes.Header{Number: big.NewInt(10)}
		td10 := &tdResp{TotalDifficulty: hexutil.Uint64(42)}
		addr10 := common.HexToAddress("0x1111")

		*elems[0].Result.(**ethTypes.Header) = hdr10
		*elems[int(totalBlocks)].Result.(**tdResp) = td10
		*elems[2*int(totalBlocks)].Result.(**common.Address) = &addr10

		require.Equal(t, hdr10, hdrs[0])
		require.Equal(t, td10, tds[0])
		require.Equal(t, &addr10, authors[0])
	})
}

// TestBorAuthorFromBatch verifies index arithmetic and error handling.
func TestBorAuthorFromBatch(t *testing.T) {
	t.Parallel()

	t.Run("non_genesis_start_happy_path", func(t *testing.T) {
		t.Parallel()

		// start=1, totalBlocks=3, i=0 → authorReqIndex = 2*3 + 0 = 6
		// (start != 0 → no decrement)
		want := common.HexToAddress("0xdeadbeef")
		authors := []*common.Address{&want, nil, nil}

		// Build a batchElems slice large enough: indices 0..5 are headers/TDs,
		// index 6 is the first author.
		batchElems := make([]rpc.BatchElem, 7)
		// batchElems[6].Error is nil by default

		addr, ok := borAuthorFromBatch(0, 1, 3, batchElems, authors)
		require.True(t, ok)
		require.Equal(t, want, addr)
	})

	t.Run("genesis_start_index_shifted", func(t *testing.T) {
		t.Parallel()

		// start=0, totalBlocks=3, i=1 (block 1)
		// Correct: authorReqIndex = 2*3 + 1 - 1 = 6
		want := common.HexToAddress("0xabcdef")
		authors := []*common.Address{nil, &want, nil}

		batchElems := make([]rpc.BatchElem, 7)
		// Inject errors at wrong indices to distinguish mutants:
		batchElems[0].Error = errors.New("wrong index 0")
		batchElems[4].Error = errors.New("wrong index 4")

		addr, ok := borAuthorFromBatch(1, 0, 3, batchElems, authors)
		require.True(t, ok, "correct index 6 has no error, should return the author")
		require.Equal(t, want, addr)
	})

	t.Run("elem_error_returns_false", func(t *testing.T) {
		t.Parallel()

		authors := []*common.Address{new(common.HexToAddress("0x1111"))}
		batchElems := make([]rpc.BatchElem, 3)
		batchElems[2].Error = errors.New("rpc error") // index = 2*1 + 0 = 2

		addr, ok := borAuthorFromBatch(0, 1, 1, batchElems, authors)
		require.False(t, ok)
		require.Equal(t, common.Address{}, addr)
	})

	t.Run("nil_author_returns_false", func(t *testing.T) {
		t.Parallel()

		// authors[0] == nil → return false even if no batch error
		authors := []*common.Address{nil}
		batchElems := make([]rpc.BatchElem, 3) // index 2, no error

		addr, ok := borAuthorFromBatch(0, 1, 1, batchElems, authors)
		require.False(t, ok)
		require.Equal(t, common.Address{}, addr)
	})
}

// TestCollateBorBatchResults verifies the "stop at first error" semantics and
// the genesis-block author bypass.
func TestCollateBorBatchResults(t *testing.T) {
	t.Parallel()

	// makeHdr returns a minimal non-nil header with the given block number.
	makeHdr := func(n int64) *ethTypes.Header {
		return &ethTypes.Header{Number: big.NewInt(n), Difficulty: big.NewInt(1)}
	}
	makeTD := func(v uint64) *tdResp { return &tdResp{TotalDifficulty: hexutil.Uint64(v)} }
	addr := func(s string) *common.Address { return new(common.HexToAddress(s)) }

	t.Run("all_blocks_good_non_genesis", func(t *testing.T) {
		t.Parallel()

		start, totalBlocks := int64(1), int64(2) // blocks 1 and 2
		hdrs := []*ethTypes.Header{makeHdr(1), makeHdr(2)}
		tds := []*tdResp{makeTD(100), makeTD(200)}
		authors := []*common.Address{addr("0x1111"), addr("0x2222")}

		// batchElems layout: [hdr0, hdr1, td0, td1, author0, author1]
		batchElems := buildBorBatchElems(start, start+totalBlocks-1, hdrs, tds, authors)

		headers, tdSlice, authorSlice := collateBorBatchResults(start, totalBlocks, batchElems, hdrs, tds, authors)

		require.Len(t, headers, 2)
		require.Len(t, tdSlice, 2)
		require.Len(t, authorSlice, 2)
		require.Equal(t, big.NewInt(1), headers[0].Number)
		require.Equal(t, big.NewInt(2), headers[1].Number)
		require.Equal(t, uint64(100), tdSlice[0])
		require.Equal(t, uint64(200), tdSlice[1])
		require.Equal(t, common.HexToAddress("0x1111"), authorSlice[0])
		require.Equal(t, common.HexToAddress("0x2222"), authorSlice[1])
	})

	t.Run("header_error_stops_at_first_bad_block", func(t *testing.T) {
		t.Parallel()

		start, totalBlocks := int64(1), int64(3) // blocks 1, 2, 3
		hdrs := []*ethTypes.Header{makeHdr(1), makeHdr(2), makeHdr(3)}
		tds := []*tdResp{makeTD(10), makeTD(20), makeTD(30)}
		authors := []*common.Address{addr("0x1"), addr("0x2"), addr("0x3")}

		batchElems := buildBorBatchElems(start, start+totalBlocks-1, hdrs, tds, authors)
		// Inject an error into block 2's header elem (index 1)
		batchElems[1].Error = errors.New("server error")

		headers, tdSlice, authorSlice := collateBorBatchResults(start, totalBlocks, batchElems, hdrs, tds, authors)

		// Must stop at block 2, returning only block 1
		require.Len(t, headers, 1)
		require.Len(t, tdSlice, 1)
		require.Len(t, authorSlice, 1)
		require.Equal(t, big.NewInt(1), headers[0].Number)
	})

	t.Run("nil_header_stops_early", func(t *testing.T) {
		t.Parallel()

		start, totalBlocks := int64(5), int64(2)
		hdrs := []*ethTypes.Header{makeHdr(5), nil} // second block has nil header
		tds := []*tdResp{makeTD(500), makeTD(600)}
		authors := []*common.Address{addr("0x5"), addr("0x6")}

		batchElems := buildBorBatchElems(start, start+totalBlocks-1, hdrs, tds, authors)

		headers, tdSlice, authorSlice := collateBorBatchResults(start, totalBlocks, batchElems, hdrs, tds, authors)

		require.Len(t, headers, 1, "nil header in position 1 must stop iteration")
		require.Len(t, tdSlice, 1)
		require.Len(t, authorSlice, 1)
	})

	t.Run("genesis_block_gets_zero_author", func(t *testing.T) {
		t.Parallel()

		// start=0, block 0 is genesis: blockNum=0 → author bypassed, zero address used.
		start, totalBlocks := int64(0), int64(1)
		hdrs := []*ethTypes.Header{makeHdr(0)}
		tds := []*tdResp{makeTD(0)}
		authors := []*common.Address{nil} // genesis has no author

		batchElems := buildBorBatchElems(start, start+totalBlocks-1, hdrs, tds, authors)

		headers, tdSlice, authorSlice := collateBorBatchResults(start, totalBlocks, batchElems, hdrs, tds, authors)

		require.Len(t, headers, 1)
		require.Len(t, tdSlice, 1)
		require.Len(t, authorSlice, 1)
		// Genesis block author is the zero address (not fetched from batch)
		require.Equal(t, common.Address{}, authorSlice[0])
	})

	t.Run("author_error_stops_iteration", func(t *testing.T) {
		t.Parallel()

		start, totalBlocks := int64(1), int64(2) // blocks 1, 2
		hdrs := []*ethTypes.Header{makeHdr(1), makeHdr(2)}
		tds := []*tdResp{makeTD(10), makeTD(20)}
		authors := []*common.Address{addr("0x1"), nil} // block 2 has nil author → borAuthorFromBatch returns false

		batchElems := buildBorBatchElems(start, start+totalBlocks-1, hdrs, tds, authors)

		headers, tdSlice, authorSlice := collateBorBatchResults(start, totalBlocks, batchElems, hdrs, tds, authors)

		// Block 1 is good; block 2's nil author causes break.
		require.Len(t, headers, 1)
		require.Len(t, tdSlice, 1)
		require.Len(t, authorSlice, 1)
		require.Equal(t, big.NewInt(1), headers[0].Number)
	})

	t.Run("empty_range_returns_empty_slices", func(t *testing.T) {
		t.Parallel()

		// totalBlocks=0 means the loop doesn't execute
		headers, tdSlice, authorSlice := collateBorBatchResults(1, 0, nil, nil, nil, nil)
		require.Empty(t, headers)
		require.Empty(t, tdSlice)
		require.Empty(t, authorSlice)
	})
}
