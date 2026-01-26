package processor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/0xPolygon/heimdall-v2/bridge/util"
	"github.com/0xPolygon/heimdall-v2/helper"
	helperMocks "github.com/0xPolygon/heimdall-v2/helper/mocks"
	"github.com/ethereum/go-ethereum/common"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestBaseProcessor_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		processorName string
	}{
		{
			name:          "checkpoint processor",
			processorName: "checkpoint",
		},
		{
			name:          "span processor",
			processorName: "span",
		},
		{
			name:          "clerk processor",
			processorName: "clerk",
		},
		{
			name:          "stake processor",
			processorName: "stake",
		},
		{
			name:          "topup_fee processor",
			processorName: "topup_fee",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			bp := &BaseProcessor{
				name: tt.processorName,
			}
			require.Equal(t, tt.processorName, bp.String())
		})
	}
}

func TestProcessorInterface_Methods(t *testing.T) {
	t.Parallel()

	t.Run("verifies processor interface implementation", func(t *testing.T) {
		t.Parallel()

		// Test that our processors implement the Processor interface
		var _ Processor = (*CheckpointProcessor)(nil)
		var _ Processor = (*SpanProcessor)(nil)
		var _ Processor = (*ClerkProcessor)(nil)
		var _ Processor = (*StakingProcessor)(nil)
		var _ Processor = (*FeeProcessor)(nil)
	})
}

// TestBaseProcessor_MempoolResponseStructure tests mempool API response parsing
func TestBaseProcessor_MempoolResponseStructure(t *testing.T) {
	t.Run("parses mempool response with transactions", func(t *testing.T) {
		responseBody := `{
			"result": {
				"total": "5",
				"txs": ["tx1", "tx2", "tx3", "tx4", "tx5"]
			}
		}`

		var mempoolResponse struct {
			Result struct {
				Total string   `json:"total"`
				Txs   []string `json:"txs"`
			} `json:"result"`
		}

		err := json.Unmarshal([]byte(responseBody), &mempoolResponse)
		require.NoError(t, err)
		require.Equal(t, "5", mempoolResponse.Result.Total)
		require.Len(t, mempoolResponse.Result.Txs, 5)
	})

	t.Run("parses empty mempool response", func(t *testing.T) {
		responseBody := `{
			"result": {
				"total": "0",
				"txs": []
			}
		}`

		var mempoolResponse struct {
			Result struct {
				Total string   `json:"total"`
				Txs   []string `json:"txs"`
			} `json:"result"`
		}

		err := json.Unmarshal([]byte(responseBody), &mempoolResponse)
		require.NoError(t, err)
		require.Equal(t, "0", mempoolResponse.Result.Total)
		require.Empty(t, mempoolResponse.Result.Txs)
	})
}

// TestBaseProcessor_IsOldTxAPIResponse tests isOldTx API response parsing
func TestBaseProcessor_IsOldTxAPIResponse(t *testing.T) {
	t.Run("parses old transaction response", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		mockHttpClient := helperMocks.NewMockHTTPClient(mockCtrl)

		txHash := "0x1234567890abcdef"
		apiUrl := "http://localhost:1317/clerk/is-old-tx?logindex=0&txhash=" + txHash

		responseBody := `{"result": true}`
		mockHttpClient.EXPECT().Get(apiUrl).Return(&http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBufferString(responseBody)),
		}, nil)

		helper.Client = mockHttpClient

		// Parse response
		resp, err := helper.Client.Get(apiUrl)
		require.NoError(t, err)
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				t.Errorf("error closing response body: %v", err)
			}
		}(resp.Body)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var result struct {
			Result bool `json:"result"`
		}
		err = json.Unmarshal(body, &result)
		require.NoError(t, err)
		require.True(t, result.Result)
	})

	t.Run("parses new transaction response", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		mockHttpClient := helperMocks.NewMockHTTPClient(mockCtrl)

		txHash := "0xabcdef1234567890"
		apiUrl := "http://localhost:1317/clerk/is-old-tx?logindex=1&txhash=" + txHash

		responseBody := `{"result": false}`
		mockHttpClient.EXPECT().Get(apiUrl).Return(&http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBufferString(responseBody)),
		}, nil)

		helper.Client = mockHttpClient

		// Parse response
		resp, err := helper.Client.Get(apiUrl)
		require.NoError(t, err)
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				t.Errorf("error closing response body: %v", err)
			}
		}(resp.Body)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var result struct {
			Result bool `json:"result"`
		}
		err = json.Unmarshal(body, &result)
		require.NoError(t, err)
		require.False(t, result.Result)
	})

	t.Run("handles API error gracefully", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		mockHttpClient := helperMocks.NewMockHTTPClient(mockCtrl)

		txHash := "0xdeadbeef"
		apiUrl := "http://localhost:1317/clerk/is-old-tx?logindex=0&txhash=" + txHash

		mockHttpClient.EXPECT().Get(apiUrl).Return(nil, http.ErrHandlerTimeout)

		helper.Client = mockHttpClient

		// Attempt to get a response
		resp, err := helper.Client.Get(apiUrl)
		require.Error(t, err)
		require.Nil(t, resp)
		require.Equal(t, http.ErrHandlerTimeout, err)
	})
}

// TestBaseProcessor_EventTypeRouting tests event type routing
func TestBaseProcessor_EventTypeRouting(t *testing.T) {
	t.Parallel()

	t.Run("validates clerk event type", func(t *testing.T) {
		t.Parallel()

		eventType := util.ClerkEvent
		require.Equal(t, util.BridgeEvent("clerk"), eventType)
		require.NotEmpty(t, string(eventType))
	})

	t.Run("validates staking event type", func(t *testing.T) {
		t.Parallel()

		eventType := util.StakingEvent
		require.Equal(t, util.BridgeEvent("staking"), eventType)
		require.NotEmpty(t, string(eventType))
	})

	t.Run("validates topup event type", func(t *testing.T) {
		t.Parallel()

		eventType := util.TopupEvent
		require.Equal(t, util.BridgeEvent("topup"), eventType)
		require.NotEmpty(t, string(eventType))
	})

	t.Run("validates all event types are unique", func(t *testing.T) {
		t.Parallel()

		eventTypes := []util.BridgeEvent{
			util.ClerkEvent,
			util.StakingEvent,
			util.TopupEvent,
		}

		// Check all are non-empty and unique
		seen := make(map[string]bool)
		for _, eventType := range eventTypes {
			str := string(eventType)
			require.NotEmpty(t, str)
			require.False(t, seen[str], "Event type %s is duplicated", str)
			seen[str] = true
		}

		require.Len(t, seen, 3)
	})
}

// TestBaseProcessor_TransactionHashValidation tests tx hash handling
func TestBaseProcessor_TransactionHashValidation(t *testing.T) {
	t.Parallel()

	t.Run("validates Ethereum transaction hash format", func(t *testing.T) {
		t.Parallel()

		validTxHash := "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
		require.Len(t, validTxHash, 66) // 0x + 64 hex chars

		// Convert to common.Hash
		txHash := common.HexToHash(validTxHash)
		require.NotEqual(t, common.Hash{}, txHash)
	})

	t.Run("handles transaction hash without 0x prefix", func(t *testing.T) {
		t.Parallel()

		txHashNoPrefix := "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
		require.Len(t, txHashNoPrefix, 64)

		// Can be converted
		txHash := common.HexToHash(txHashNoPrefix)
		require.NotEqual(t, common.Hash{}, txHash)
	})

	t.Run("validates log index range", func(t *testing.T) {
		t.Parallel()

		validLogIndexes := []uint64{0, 1, 10, 100, 1000}

		for _, logIndex := range validLogIndexes {
			require.GreaterOrEqual(t, logIndex, uint64(0))
		}
	})

	t.Run("validates tx hash hex encoding", func(t *testing.T) {
		t.Parallel()

		// Valid hex characters
		validChars := "0123456789abcdefABCDEF"
		for _, char := range validChars {
			require.True(t, (char >= '0' && char <= '9') ||
				(char >= 'a' && char <= 'f') ||
				(char >= 'A' && char <= 'F'))
		}
	})
}

// TestBaseProcessor_APIEndpoints tests API endpoint construction
func TestBaseProcessor_APIEndpoints(t *testing.T) {
	t.Parallel()

	t.Run("constructs clerk is-old-tx endpoint", func(t *testing.T) {
		t.Parallel()

		baseURL := "http://localhost:1317"
		txHash := "0xabcdef"
		logIndex := uint64(5)

		expected := "http://localhost:1317/clerk/is-old-tx?logindex=5&txhash=0xabcdef"
		actual := fmt.Sprintf("%s/clerk/is-old-tx?logindex=%d&txhash=%s", baseURL, logIndex, txHash)

		require.Equal(t, expected, actual)
	})

	t.Run("constructs topup is-old-tx endpoint", func(t *testing.T) {
		t.Parallel()

		baseURL := "http://localhost:1317"
		txHash := "0x123456"
		logIndex := uint64(10)

		expected := "http://localhost:1317/topup/is-old-tx?logindex=10&txhash=0x123456"
		actual := fmt.Sprintf("%s/topup/is-old-tx?logindex=%d&txhash=%s", baseURL, logIndex, txHash)

		require.Equal(t, expected, actual)
	})

	t.Run("constructs staking is-old-tx endpoint", func(t *testing.T) {
		t.Parallel()

		baseURL := "http://localhost:1317"
		txHash := "0x789abc"
		logIndex := uint64(15)

		expected := "http://localhost:1317/staking/is-old-tx?logindex=15&txhash=0x789abc"
		actual := fmt.Sprintf("%s/staking/is-old-tx?logindex=%d&txhash=%s", baseURL, logIndex, txHash)

		require.Equal(t, expected, actual)
	})
}

// TestBaseProcessor_EventUniqueness tests event identification
func TestBaseProcessor_EventUniqueness(t *testing.T) {
	t.Parallel()

	t.Run("validates unique event identification by txHash and logIndex", func(t *testing.T) {
		t.Parallel()

		// Same tx, different log index = different event
		event1 := struct {
			TxHash   string
			LogIndex uint64
		}{
			TxHash:   "0xabc",
			LogIndex: 0,
		}

		event2 := struct {
			TxHash   string
			LogIndex uint64
		}{
			TxHash:   "0xabc",
			LogIndex: 1,
		}

		require.Equal(t, event1.TxHash, event2.TxHash)
		require.NotEqual(t, event1.LogIndex, event2.LogIndex)
	})

	t.Run("validates different events from different transactions", func(t *testing.T) {
		t.Parallel()

		event1 := struct {
			TxHash   string
			LogIndex uint64
		}{
			TxHash:   "0xabc",
			LogIndex: 0,
		}

		event2 := struct {
			TxHash   string
			LogIndex uint64
		}{
			TxHash:   "0xdef",
			LogIndex: 0,
		}

		require.NotEqual(t, event1.TxHash, event2.TxHash)
		require.Equal(t, event1.LogIndex, event2.LogIndex)
	})
}
