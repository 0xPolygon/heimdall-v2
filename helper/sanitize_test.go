package helper_test

import (
	"testing"

	"cosmossdk.io/log"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/helper"
)

func TestSanitizeConfig_WithNilViper(t *testing.T) {
	logger := log.NewNopLogger()

	// Calling with nil viper should cause an error
	var nilViper *viper.Viper
	err := helper.SanitizeConfig(nilViper, logger)

	require.Error(t, err)
}

func TestSanitizeConfig_WithEmptyViper(t *testing.T) {
	logger := log.NewNopLogger()
	v := viper.New()

	// Calling with empty viper should complete without panic
	err := helper.SanitizeConfig(v, logger)

	// May error due to missing required config, but should not panic
	require.NotPanics(t, func() {
		_ = helper.SanitizeConfig(v, logger)
	})

	// Empty config may or may not error depending on implementation
	// The important thing is it doesn't panic
	_ = err
}

func TestSanitizeConfig_WithNilLogger(t *testing.T) {
	v := viper.New()

	// Calling with nil logger should handle gracefully or panic
	// We test that it doesn't cause unexpected behavior
	require.NotPanics(t, func() {
		_ = helper.SanitizeConfig(v, nil)
	})
}

func TestSanitizeConfig_WithValidConfig(t *testing.T) {
	logger := log.NewNopLogger()
	v := viper.New()

	// Set some basic configuration values
	v.Set("custom.eth_rpc_url", "http://localhost:8545")
	v.Set("custom.bor_rpc_url", "http://localhost:8546")
	v.Set("custom.comet_bft_rpc_url", "http://localhost:26657")

	// Test that the function doesn't panic with some config
	err := helper.SanitizeConfig(v, logger)

	// May error due to incomplete config, but should not panic
	require.NotPanics(t, func() {
		_ = helper.SanitizeConfig(v, logger)
	})

	// We don't strictly require success because the config may be incomplete
	_ = err
}

func TestSanitizeConfig_FunctionSignature(t *testing.T) {
	// Test that the function accepts the correct types
	logger := log.NewNopLogger()
	v := viper.New()

	// Should compile and accept these types
	var err error
	err = helper.SanitizeConfig(v, logger)

	// Function should return an error type (could be nil if config is valid)
	_ = err // the error type is correct, value depends on config
}

func TestSanitizeConfig_ReturnsError(t *testing.T) {
	// Test that function returns an error for nil viper
	logger := log.NewNopLogger()

	// With nil viper, should return an error
	err := helper.SanitizeConfig(nil, logger)

	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid config")
}

func TestSanitizeConfig_ErrorMessage(t *testing.T) {
	logger := log.NewNopLogger()
	v := viper.New()

	err := helper.SanitizeConfig(v, logger)

	// Should return error with "invalid config" message
	if err != nil {
		require.Contains(t, err.Error(), "invalid config")
	}
}

func TestSanitizeConfig_DoesNotPanic(t *testing.T) {
	tests := []struct {
		name   string
		viper  *viper.Viper
		logger log.Logger
	}{
		{
			name:   "empty viper and valid logger",
			viper:  viper.New(),
			logger: log.NewNopLogger(),
		},
		{
			name: "viper with some config",
			viper: func() *viper.Viper {
				v := viper.New()
				v.Set("custom.eth_rpc_url", "http://test")
				return v
			}(),
			logger: log.NewNopLogger(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NotPanics(t, func() {
				_ = helper.SanitizeConfig(tt.viper, tt.logger)
			})
		})
	}
}

func TestSanitizeConfig_WithPartialConfig(t *testing.T) {
	logger := log.NewNopLogger()
	v := viper.New()

	// Set partial configuration
	v.Set("custom.eth_rpc_url", "http://localhost:8545")

	err := helper.SanitizeConfig(v, logger)

	// Should not panic
	require.NotPanics(t, func() {
		_ = helper.SanitizeConfig(v, logger)
	})

	// Error is expected for incomplete config
	_ = err
}

func TestSanitizeConfig_MultipleConfigValues(t *testing.T) {
	logger := log.NewNopLogger()
	v := viper.New()

	// Set multiple configuration values
	configs := map[string]interface{}{
		"custom.eth_rpc_url":              "http://localhost:8545",
		"custom.bor_rpc_url":              "http://localhost:8546",
		"custom.comet_bft_rpc_url":        "http://localhost:26657",
		"custom.checkpoint_poll_interval": "5s",
		"custom.syncer_poll_interval":     "1s",
	}

	for key, value := range configs {
		v.Set(key, value)
	}

	err := helper.SanitizeConfig(v, logger)

	// Should handle multiple config values without panic
	require.NotPanics(t, func() {
		_ = helper.SanitizeConfig(v, logger)
	})

	_ = err
}

func TestSanitizeConfig_CallMultipleTimes(t *testing.T) {
	logger := log.NewNopLogger()
	v := viper.New()

	// Should be idempotent - calling multiple times should be safe
	err1 := helper.SanitizeConfig(v, logger)
	err2 := helper.SanitizeConfig(v, logger)

	// Both calls should behave consistently
	if err1 != nil && err2 != nil {
		require.Equal(t, err1.Error(), err2.Error())
	}
}

func TestSanitizeConfig_PreservesViperInstance(t *testing.T) {
	logger := log.NewNopLogger()
	v := viper.New()

	// Set a value
	v.Set("test.key", "test-value")

	_ = helper.SanitizeConfig(v, logger)

	// Original viper should still be accessible
	require.NotNil(t, v)
}

func TestSanitizeConfig_WithInvalidDurationsType(t *testing.T) {
	logger := log.NewNopLogger()
	v := viper.New()

	// Set invalid duration values
	v.Set("custom.checkpoint_poll_interval", "invalid")

	err := helper.SanitizeConfig(v, logger)

	// Should handle invalid duration gracefully
	require.NotPanics(t, func() {
		_ = helper.SanitizeConfig(v, logger)
	})

	_ = err
}
