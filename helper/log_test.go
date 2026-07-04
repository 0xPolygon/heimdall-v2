package helper

import (
	"bytes"
	"strings"
	"testing"

	logger "cosmossdk.io/log"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestLogLevelOption(t *testing.T) {
	t.Run("empty defaults to info", func(t *testing.T) {
		cfg := applyLogLevel(t, "")
		require.Equal(t, zerolog.InfoLevel, cfg.Level)
		require.Nil(t, cfg.Filter)
	})

	t.Run("plain level sets a global level", func(t *testing.T) {
		cfg := applyLogLevel(t, "debug")
		require.Equal(t, zerolog.DebugLevel, cfg.Level)
		require.Nil(t, cfg.Filter)
	})

	t.Run("module spec installs a per-module filter", func(t *testing.T) {
		cfg := applyLogLevel(t, "*:info,mempool:debug")
		require.NotNil(t, cfg.Filter)
		// FilterFunc returns true when the entry should be discarded.
		require.False(t, cfg.Filter("mempool", "debug"), "mempool debug must be kept")
		require.True(t, cfg.Filter("p2p", "debug"), "non-target debug must be dropped")
		require.False(t, cfg.Filter("p2p", "info"), "non-target info must be kept")
	})

	t.Run("malformed spec errors", func(t *testing.T) {
		_, err := LogLevelOption("mempool:notalevel")
		require.Error(t, err)
	})
}

// End-to-end: a real logger built with a module spec keeps the targeted
// module's debug and drops every other module's debug — the scenario that a
// global "debug" would have flooded.
func TestLogLevelOption_FiltersRealLogger(t *testing.T) {
	opt, err := LogLevelOption("*:info,mempool:debug")
	require.NoError(t, err)

	var buf bytes.Buffer
	log := logger.NewLogger(&buf, opt, logger.OutputJSONOption())

	log.Debug("kept", logger.ModuleKey, "mempool")
	log.Debug("dropped", logger.ModuleKey, "p2p")
	log.Info("kept", logger.ModuleKey, "p2p")

	out := buf.String()
	require.Contains(t, out, `"module":"mempool"`, "mempool debug must be emitted")
	require.Equal(t, 1, strings.Count(out, `"module":"p2p"`), "only p2p info, not p2p debug")
	require.NotContains(t, out, `"message":"dropped"`)
}

func TestLogLevelOptionOrDefault(t *testing.T) {
	t.Run("valid spec passes through without warning", func(t *testing.T) {
		warned := false
		opt := LogLevelOptionOrDefault("*:info,mempool:debug", func(string, ...any) { warned = true })
		var cfg logger.Config
		opt(&cfg)
		require.NotNil(t, cfg.Filter)
		require.False(t, warned, "valid spec must not warn")
	})

	t.Run("malformed spec warns and falls back to info", func(t *testing.T) {
		warned := false
		opt := LogLevelOptionOrDefault("mempool:notalevel", func(string, ...any) { warned = true })
		var cfg logger.Config
		opt(&cfg)
		require.Equal(t, zerolog.InfoLevel, cfg.Level)
		require.Nil(t, cfg.Filter)
		require.True(t, warned, "malformed spec must warn before falling back")
	})
}

func applyLogLevel(t *testing.T, s string) logger.Config {
	t.Helper()
	opt, err := LogLevelOption(s)
	require.NoError(t, err)
	var cfg logger.Config
	opt(&cfg)
	return cfg
}
