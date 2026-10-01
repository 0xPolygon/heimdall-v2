package helper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"cosmossdk.io/log"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/helper/mocks"
)

func TestFetchAndValidateReceipt(t *testing.T) {
	logger := log.NewNopLogger()

	tests := []struct {
		name           string
		receipt        *ethTypes.Receipt
		err            error
		params         ReceiptValidationParams
		expectedResult bool // true if the receipt should be returned (not nil)
	}{
		{
			name: "valid receipt with matching block number",
			receipt: &ethTypes.Receipt{
				BlockNumber: big.NewInt(100),
			},
			err: nil,
			params: ReceiptValidationParams{
				TxHash:         common.Hex2Bytes("0x1234"),
				MsgBlockNumber: 100,
				Confirmations:  6,
				ModuleName:     "test",
			},
			expectedResult: true,
		},
		{
			name:    "nil receipt",
			receipt: nil,
			err:     nil,
			params: ReceiptValidationParams{
				TxHash:         common.Hex2Bytes("0x1234"),
				MsgBlockNumber: 100,
				Confirmations:  6,
				ModuleName:     "test",
			},
			expectedResult: false,
		},
		{
			name:    "error fetching receipt",
			receipt: nil,
			err:     errors.New("network error"),
			params: ReceiptValidationParams{
				TxHash:         common.Hex2Bytes("0x1234"),
				MsgBlockNumber: 100,
				Confirmations:  6,
				ModuleName:     "test",
			},
			expectedResult: false,
		},
		{
			name: "block number mismatch",
			receipt: &ethTypes.Receipt{
				BlockNumber: big.NewInt(99),
			},
			err: nil,
			params: ReceiptValidationParams{
				TxHash:         common.Hex2Bytes("0x1234"),
				MsgBlockNumber: 100,
				Confirmations:  6,
				ModuleName:     "test",
			},
			expectedResult: false,
		},
		{
			// receipt.Logs is []*ethTypes.Log; a malformed/hostile RPC response
			// can unmarshal a nil entry, which every Decode*Event caller
			// dereferences without a nil check.
			name: "receipt contains a nil log entry",
			receipt: &ethTypes.Receipt{
				BlockNumber: big.NewInt(100),
				Logs:        []*ethTypes.Log{nil},
			},
			err: nil,
			params: ReceiptValidationParams{
				TxHash:         common.Hex2Bytes("0x1234"),
				MsgBlockNumber: 100,
				Confirmations:  6,
				ModuleName:     "test",
			},
			expectedResult: false,
		},
		{
			// An anonymous event (zero topics) elsewhere in the receipt is
			// valid content and must not invalidate the whole receipt; see
			// helper.UnpackLog's own test for the per-log check.
			name: "receipt contains an unrelated log entry with no topics",
			receipt: &ethTypes.Receipt{
				BlockNumber: big.NewInt(100),
				Logs:        []*ethTypes.Log{{Topics: nil}},
			},
			err: nil,
			params: ReceiptValidationParams{
				TxHash:         common.Hex2Bytes("0x1234"),
				MsgBlockNumber: 100,
				Confirmations:  6,
				ModuleName:     "test",
			},
			expectedResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCaller := mocks.NewIContractCaller(t)

			// Set up the mock expectation for GetConfirmedTxReceipt
			txHash := common.BytesToHash(tt.params.TxHash)
			mockCaller.On("GetConfirmedTxReceipt", mock.Anything, txHash, tt.params.Confirmations).
				Return(tt.receipt, tt.err)

			result := FetchAndValidateReceipt(context.Background(), mockCaller, tt.params, logger)

			if tt.expectedResult {
				require.NotNil(t, result, "expected receipt to be returned")
				require.Equal(t, tt.receipt, result)
			} else {
				require.Nil(t, result, "expected nil receipt")
			}
		})
	}
}

// TestFetchAndValidateReceipt_ContractCallerPanic verifies that a panic
// inside GetConfirmedTxReceipt (e.g. a malformed RPC response tripping a bad
// type assertion in the client) is recovered as a nil receipt, not a crash.
func TestFetchAndValidateReceipt_ContractCallerPanic(t *testing.T) {
	logger := log.NewNopLogger()

	mockCaller := mocks.NewIContractCaller(t)
	mockCaller.On("GetConfirmedTxReceipt", mock.Anything, mock.Anything, mock.Anything).
		Run(func(mock.Arguments) { panic("bor client exploded") })

	params := ReceiptValidationParams{
		TxHash:         common.Hex2Bytes("0x1234"),
		MsgBlockNumber: 100,
		Confirmations:  6,
		ModuleName:     "test",
	}

	result := FetchAndValidateReceipt(context.Background(), mockCaller, params, logger)

	require.Nil(t, result, "expected nil receipt after a recovered panic")
}

// fakeBatchReceiptRPC serves a single batch JSON-RPC response so
// PrefetchReceipts can be exercised against a real *ContractCaller instead
// of the IContractCaller mock (BatchGetMainChainTxReceipts type-asserts to
// *ContractCaller internally, so the mock never reaches this code path).
func fakeBatchReceiptRPC(t *testing.T, blockNumberHex string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqs []struct {
			ID     int64  `json:"id"`
			Method string `json:"method"`
		}
		require.NoError(t, jsonDecode(r.Body, &reqs))

		emptyBloom := "0x" + strings.Repeat("0", 512)

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, "[")
		for i, req := range reqs {
			if i > 0 {
				fmt.Fprint(w, ",")
			}
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%d,"result":{`+
				`"blockNumber":%q,"status":"0x1","cumulativeGasUsed":"0x5208","gasUsed":"0x5208",`+
				`"logsBloom":%q,"logs":[],"transactionHash":"0x%064x"}}`,
				req.ID, blockNumberHex, emptyBloom, i+1)
		}
		fmt.Fprint(w, "]")
	}))
}

// TestPrefetchReceipts_Success verifies the isolated RunIsolated wrapping
// around BatchGetMainChainTxReceipts leaves the success path unaffected: a
// real batch RPC response is fetched and cached under prefetchedReceipts.
func TestPrefetchReceipts_Success(t *testing.T) {
	logger := log.NewNopLogger()

	server := fakeBatchReceiptRPC(t, "0x64")
	defer server.Close()

	rc, err := rpc.Dial(server.URL)
	require.NoError(t, err)
	defer rc.Close()

	caller := &ContractCaller{
		MainChainRPCClient: rc,
		prefetchMu:         &sync.RWMutex{},
		prefetchedReceipts: make(map[common.Hash]*ethTypes.Receipt),
	}

	hashes := []common.Hash{common.HexToHash("0x1"), common.HexToHash("0x2")}
	PrefetchReceipts(context.Background(), caller, hashes, logger)

	caller.prefetchMu.RLock()
	defer caller.prefetchMu.RUnlock()
	require.Len(t, caller.prefetchedReceipts, len(hashes), "both receipts from the batch response must be cached")
	for _, h := range hashes {
		receipt, ok := caller.prefetchedReceipts[h]
		require.True(t, ok, "expected a cached receipt for %s", h)
		require.Equal(t, uint64(100), receipt.BlockNumber.Uint64())
	}
}

// TestPrefetchReceipts_EmptyInput verifies no RPC call is made, and no panic
// occurs, when there is nothing to prefetch.
func TestPrefetchReceipts_EmptyInput(t *testing.T) {
	logger := log.NewNopLogger()
	caller := &ContractCaller{prefetchMu: &sync.RWMutex{}}
	PrefetchReceipts(context.Background(), caller, nil, logger)
}

// TestClearPrefetchedReceipts verifies a stale cache is actually dropped, not
// left in place, so a panic mid-batch can't leave partial results being served.
func TestClearPrefetchedReceipts(t *testing.T) {
	caller := &ContractCaller{
		prefetchMu:         &sync.RWMutex{},
		prefetchedReceipts: map[common.Hash]*ethTypes.Receipt{common.HexToHash("0x1"): {}},
	}

	clearPrefetchedReceipts(caller)

	caller.prefetchMu.RLock()
	defer caller.prefetchMu.RUnlock()
	require.Nil(t, caller.prefetchedReceipts, "stale cache must be cleared, not left in place")
}

func jsonDecode(r io.Reader, v any) error {
	return json.NewDecoder(r).Decode(v)
}
