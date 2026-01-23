package util_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/bridge/util"
)

func TestBridgeEvent_Constants(t *testing.T) {
	require.Equal(t, util.BridgeEvent("staking"), util.StakingEvent)
	require.Equal(t, util.BridgeEvent("topup"), util.TopupEvent)
	require.Equal(t, util.BridgeEvent("clerk"), util.ClerkEvent)
	require.Equal(t, util.BridgeDBFlag, "bridge-db")
}

func TestBridgeEvent_Uniqueness(t *testing.T) {
	events := []util.BridgeEvent{
		util.StakingEvent,
		util.TopupEvent,
		util.ClerkEvent,
	}

	for i := 0; i < len(events); i++ {
		for j := i + 1; j < len(events); j++ {
			require.NotEqual(t, events[i], events[j],
				"Events at positions %d and %d should be different", i, j)
		}
	}
}

func TestURLConstants_NotEmpty(t *testing.T) {
	urls := map[string]string{
		"AccountDetailsURL":       util.AccountDetailsURL,
		"AccountParamsURL":        util.AccountParamsURL,
		"LastNoAckURL":            util.LastNoAckURL,
		"CheckpointParamsURL":     util.CheckpointParamsURL,
		"CheckpointSignaturesURL": util.CheckpointSignaturesURL,
		"ChainManagerParamsURL":   util.ChainManagerParamsURL,
		"ProposersURL":            util.ProposersURL,
		"BufferedCheckpointURL":   util.BufferedCheckpointURL,
		"LatestCheckpointURL":     util.LatestCheckpointURL,
		"CountCheckpointURL":      util.CountCheckpointURL,
		"CurrentProposerURL":      util.CurrentProposerURL,
		"LatestSpanURL":           util.LatestSpanURL,
		"SpanByIdURL":             util.SpanByIdURL,
		"NextSpanInfoURL":         util.NextSpanInfoURL,
		"NextSpanSeedURL":         util.NextSpanSeedURL,
		"ProducerVotesURL":        util.ProducerVotesURL,
		"DividendAccountRootURL":  util.DividendAccountRootURL,
		"ValidatorURL":            util.ValidatorURL,
		"CurrentValidatorSetURL":  util.CurrentValidatorSetURL,
		"StakingTxStatusURL":      util.StakingTxStatusURL,
		"TopupTxStatusURL":        util.TopupTxStatusURL,
		"ClerkTxStatusURL":        util.ClerkTxStatusURL,
		"ClerkEventRecordURL":     util.ClerkEventRecordURL,
	}

	for name, url := range urls {
		require.NotEmpty(t, url, "%s should not be empty", name)
	}
}

func TestURLConstants_StartsWithSlash(t *testing.T) {
	urls := []string{
		util.AccountDetailsURL,
		util.AccountParamsURL,
		util.LastNoAckURL,
		util.CheckpointParamsURL,
		util.ProposersURL,
		util.BufferedCheckpointURL,
		util.LatestCheckpointURL,
		util.CurrentProposerURL,
		util.LatestSpanURL,
		util.ValidatorURL,
		util.CurrentValidatorSetURL,
	}

	for _, url := range urls {
		require.True(t, strings.HasPrefix(url, "/"),
			"URL should start with /: %s", url)
	}
}

func TestURLConstants_WithFormatSpecifiers(t *testing.T) {
	// URLs with format specifiers should contain %v
	urlsWithFormat := []string{
		util.AccountDetailsURL,
		util.CheckpointSignaturesURL,
		util.ProposersURL,
		util.SpanByIdURL,
		util.NextSpanSeedURL,
		util.ProducerVotesURL,
		util.ValidatorURL,
		util.ClerkEventRecordURL,
	}

	for _, url := range urlsWithFormat {
		require.Contains(t, url, "%",
			"Formatted URL should contain %%: %s", url)
	}
}

func TestCometBFTURLConstants(t *testing.T) {
	require.Equal(t, "/unconfirmed_txs", util.CometBFTUnconfirmedTxsURL)
	require.Equal(t, "/num_unconfirmed_txs", util.CometBFTUnconfirmedTxsCountURL)
}

func TestTaskDelayConstants(t *testing.T) {
	require.Equal(t, 10*time.Second, util.TaskDelayBetweenEachVal)
	require.Equal(t, 12*time.Second, util.RetryTaskDelay)
	require.Equal(t, 24*time.Second, util.RetryStateSyncTaskDelay)
}

func TestTaskDelayConstants_Positive(t *testing.T) {
	require.Greater(t, util.TaskDelayBetweenEachVal, time.Duration(0))
	require.Greater(t, util.RetryTaskDelay, time.Duration(0))
	require.Greater(t, util.RetryStateSyncTaskDelay, time.Duration(0))
}

func TestTaskDelayConstants_Ordering(t *testing.T) {
	// Retry delays should be longer than task delay
	require.Greater(t, util.RetryTaskDelay, util.TaskDelayBetweenEachVal)
	require.Greater(t, util.RetryStateSyncTaskDelay, util.RetryTaskDelay)
}

func TestBridgeDBFlag(t *testing.T) {
	require.Equal(t, "bridge-db", util.BridgeDBFlag)
	require.NotEmpty(t, util.BridgeDBFlag)
}

func TestURLConstants_CheckpointURLs(t *testing.T) {
	// Test checkpoint-related URLs
	checkpointURLs := []string{
		util.LastNoAckURL,
		util.CheckpointParamsURL,
		util.BufferedCheckpointURL,
		util.LatestCheckpointURL,
		util.CountCheckpointURL,
	}

	for _, url := range checkpointURLs {
		require.NotEmpty(t, url)
		require.Contains(t, url, "checkpoint")
	}
}

func TestURLConstants_StakeURLs(t *testing.T) {
	// Test stake-related URLs
	stakeURLs := []string{
		util.ProposersURL,
		util.CurrentProposerURL,
		util.ValidatorURL,
		util.CurrentValidatorSetURL,
		util.StakingTxStatusURL,
	}

	for _, url := range stakeURLs {
		require.NotEmpty(t, url)
		require.Contains(t, url, "stake")
	}
}

func TestURLConstants_BorURLs(t *testing.T) {
	// Test bor-related URLs
	borURLs := []string{
		util.LatestSpanURL,
		util.SpanByIdURL,
		util.NextSpanInfoURL,
		util.NextSpanSeedURL,
		util.ProducerVotesURL,
	}

	for _, url := range borURLs {
		require.NotEmpty(t, url)
		// Should contain at least one of: bor, span, producer
		containsExpected := strings.Contains(url, "bor") ||
			strings.Contains(url, "span") ||
			strings.Contains(url, "producer")
		require.True(t, containsExpected)
	}
}

func TestURLConstants_UniquePaths(t *testing.T) {
	// Collect all non-parameterized URLs
	urls := []string{
		util.AccountParamsURL,
		util.LastNoAckURL,
		util.CheckpointParamsURL,
		util.ChainManagerParamsURL,
		util.BufferedCheckpointURL,
		util.LatestCheckpointURL,
		util.CountCheckpointURL,
		util.CurrentProposerURL,
		util.LatestSpanURL,
		util.NextSpanInfoURL,
		util.DividendAccountRootURL,
		util.CurrentValidatorSetURL,
		util.StakingTxStatusURL,
		util.TopupTxStatusURL,
		util.ClerkTxStatusURL,
	}

	// Check all URLs are unique
	for i := 0; i < len(urls); i++ {
		for j := i + 1; j < len(urls); j++ {
			require.NotEqual(t, urls[i], urls[j],
				"URLs at positions %d and %d should be different", i, j)
		}
	}
}

func TestCreateURLWithQuery(t *testing.T) {
	t.Parallel()

	t.Run("creates URL with single parameter", func(t *testing.T) {
		t.Parallel()

		uri := "https://example.com/api"
		params := map[string]interface{}{
			"key": "value",
		}

		result, err := util.CreateURLWithQuery(uri, params)
		require.NoError(t, err)
		require.Contains(t, result, "key=value")
		require.Contains(t, result, "https://example.com/api")
	})

	t.Run("creates URL with multiple parameters", func(t *testing.T) {
		t.Parallel()

		uri := "https://example.com/api"
		params := map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
			"key3": 123,
		}

		result, err := util.CreateURLWithQuery(uri, params)
		require.NoError(t, err)
		require.Contains(t, result, "key1=value1")
		require.Contains(t, result, "key2=value2")
		require.Contains(t, result, "key3=123")
	})

	t.Run("creates URL with integer parameter", func(t *testing.T) {
		t.Parallel()

		uri := "https://example.com/api"
		params := map[string]interface{}{
			"number": 42,
		}

		result, err := util.CreateURLWithQuery(uri, params)
		require.NoError(t, err)
		require.Contains(t, result, "number=42")
	})

	t.Run("creates URL with boolean parameter", func(t *testing.T) {
		t.Parallel()

		uri := "https://example.com/api"
		params := map[string]interface{}{
			"enabled": true,
		}

		result, err := util.CreateURLWithQuery(uri, params)
		require.NoError(t, err)
		require.Contains(t, result, "enabled=true")
	})

	t.Run("handles empty parameter map", func(t *testing.T) {
		t.Parallel()

		uri := "https://example.com/api"
		params := map[string]interface{}{}

		result, err := util.CreateURLWithQuery(uri, params)
		require.NoError(t, err)
		require.Equal(t, uri, result)
	})

	t.Run("handles URL with existing query", func(t *testing.T) {
		t.Parallel()

		uri := "https://example.com/api?existing=param"
		params := map[string]interface{}{
			"new": "value",
		}

		result, err := util.CreateURLWithQuery(uri, params)
		require.NoError(t, err)
		require.Contains(t, result, "new=value")
		require.Contains(t, result, "existing=param")
	})

	t.Run("returns error for invalid URL", func(t *testing.T) {
		t.Parallel()

		uri := "://invalid-url"
		params := map[string]interface{}{
			"key": "value",
		}

		_, err := util.CreateURLWithQuery(uri, params)
		require.Error(t, err)
	})

	t.Run("URL encodes special characters", func(t *testing.T) {
		t.Parallel()

		uri := "https://example.com/api"
		params := map[string]interface{}{
			"key": "value with spaces",
		}

		result, err := util.CreateURLWithQuery(uri, params)
		require.NoError(t, err)
		require.Contains(t, result, "key=value+with+spaces")
	})

	t.Run("handles uint64 parameter", func(t *testing.T) {
		t.Parallel()

		uri := "https://example.com/api"
		params := map[string]interface{}{
			"id": uint64(999999999999),
		}

		result, err := util.CreateURLWithQuery(uri, params)
		require.NoError(t, err)
		require.Contains(t, result, "id=999999999999")
	})
}

func TestAppendPrefix(t *testing.T) {
	t.Parallel()

	t.Run("appends 0x04 prefix to public key", func(t *testing.T) {
		t.Parallel()

		signerPubKey := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
		result := util.AppendPrefix(signerPubKey)

		require.Len(t, result, len(signerPubKey)+1)
		require.Equal(t, byte(0x04), result[0])
		require.Equal(t, signerPubKey, result[1:])
	})

	t.Run("handles empty byte slice", func(t *testing.T) {
		t.Parallel()

		var signerPubKey []byte
		result := util.AppendPrefix(signerPubKey)

		require.Len(t, result, 1)
		require.Equal(t, byte(0x04), result[0])
	})

	t.Run("handles single byte", func(t *testing.T) {
		t.Parallel()

		signerPubKey := []byte{0xFF}
		result := util.AppendPrefix(signerPubKey)

		require.Len(t, result, 2)
		require.Equal(t, byte(0x04), result[0])
		require.Equal(t, byte(0xFF), result[1])
	})

	t.Run("handles 32-byte public key", func(t *testing.T) {
		t.Parallel()

		signerPubKey := make([]byte, 32)
		for i := range signerPubKey {
			signerPubKey[i] = byte(i)
		}

		result := util.AppendPrefix(signerPubKey)

		require.Len(t, result, 33)
		require.Equal(t, byte(0x04), result[0])
		require.Equal(t, signerPubKey, result[1:])
	})

	t.Run("handles 64-byte public key", func(t *testing.T) {
		t.Parallel()

		signerPubKey := make([]byte, 64)
		for i := range signerPubKey {
			signerPubKey[i] = byte(i % 256)
		}

		result := util.AppendPrefix(signerPubKey)

		require.Len(t, result, 65)
		require.Equal(t, byte(0x04), result[0])
		require.Equal(t, signerPubKey, result[1:])
	})

	t.Run("does not modify original slice", func(t *testing.T) {
		t.Parallel()

		original := []byte{0xAA, 0xBB, 0xCC}
		originalCopy := make([]byte, len(original))
		copy(originalCopy, original)

		result := util.AppendPrefix(original)

		// The result should be different from the original
		require.NotEqual(t, result, original)
		require.NotEqual(t, result, originalCopy)
		// Original should be unchanged
		require.Equal(t, originalCopy, original)
	})
}

func TestLogElapsedTimeForStateSyncedEvent(t *testing.T) {
	t.Parallel()

	t.Run("handles nil event", func(t *testing.T) {
		t.Parallel()

		require.NotPanics(t, func() {
			util.LogElapsedTimeForStateSyncedEvent(nil, "testFunction", time.Now())
		})
	})

	t.Run("handles non-StateSynced event", func(t *testing.T) {
		t.Parallel()

		require.NotPanics(t, func() {
			util.LogElapsedTimeForStateSyncedEvent("not a state synced event", "testFunction", time.Now())
		})
	})

	t.Run("handles elapsed time calculation", func(t *testing.T) {
		t.Parallel()

		startTime := time.Now().Add(-100 * time.Millisecond)

		require.NotPanics(t, func() {
			util.LogElapsedTimeForStateSyncedEvent(nil, "testFunction", startTime)
		})
	})
}

func TestLogger(t *testing.T) {
	t.Parallel()

	t.Run("returns logger instance", func(t *testing.T) {
		t.Parallel()

		logger := util.Logger()
		require.NotNil(t, logger)
	})

	t.Run("logger has correct module", func(t *testing.T) {
		t.Parallel()

		// Just verify it doesn't panic
		logger := util.Logger()
		require.NotNil(t, logger)
		// Logger methods should work
		require.NotPanics(t, func() {
			logger.Info("test message")
		})
	})
}

func TestBridgeEvent_TypeSafety(t *testing.T) {
	t.Parallel()

	t.Run("BridgeEvent is a string type", func(t *testing.T) {
		t.Parallel()

		var event util.BridgeEvent = "test"
		require.IsType(t, util.BridgeEvent(""), event)
	})

	t.Run("BridgeEvent constants are BridgeEvent type", func(t *testing.T) {
		t.Parallel()

		var stakingEvent = util.StakingEvent
		var topupEvent = util.TopupEvent
		var clerkEvent = util.ClerkEvent

		require.Equal(t, "staking", string(stakingEvent))
		require.Equal(t, "topup", string(topupEvent))
		require.Equal(t, "clerk", string(clerkEvent))
	})
}

func TestCreateURLWithQuery_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("handles URL with fragment", func(t *testing.T) {
		t.Parallel()

		uri := "https://example.com/api#section"
		params := map[string]interface{}{
			"key": "value",
		}

		result, err := util.CreateURLWithQuery(uri, params)
		require.NoError(t, err)
		require.Contains(t, result, "key=value")
		require.Contains(t, result, "#section")
	})

	t.Run("handles URL with port", func(t *testing.T) {
		t.Parallel()

		uri := "https://example.com:8080/api"
		params := map[string]interface{}{
			"key": "value",
		}

		result, err := util.CreateURLWithQuery(uri, params)
		require.NoError(t, err)
		require.Contains(t, result, "8080")
		require.Contains(t, result, "key=value")
	})

	t.Run("handles nil parameter map", func(t *testing.T) {
		t.Parallel()

		uri := "https://example.com/api"
		var params map[string]interface{}

		result, err := util.CreateURLWithQuery(uri, params)
		require.NoError(t, err)
		require.Equal(t, uri, result)
	})
}

func TestAppendPrefix_Idempotence(t *testing.T) {
	t.Parallel()

	t.Run("appending prefix twice produces different result", func(t *testing.T) {
		t.Parallel()

		original := []byte{0x01, 0x02, 0x03}
		result1 := util.AppendPrefix(original)
		result2 := util.AppendPrefix(result1)

		require.NotEqual(t, result1, result2)
		require.Len(t, result2, len(result1)+1)
		require.Equal(t, byte(0x04), result2[0])
		require.Equal(t, byte(0x04), result2[1])
	})
}
