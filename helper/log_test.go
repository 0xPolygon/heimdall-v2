package helper

import (
	"bytes"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	logger "cosmossdk.io/log"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestLogLevelOption(t *testing.T) {
	t.Run("empty defaults to info", func(t *testing.T) {
		cfg, filter := applyLogLevel(t, "")
		require.Equal(t, zerolog.InfoLevel, cfg.Level)
		require.Nil(t, filter)
	})

	t.Run("plain level sets a global level", func(t *testing.T) {
		cfg, filter := applyLogLevel(t, "debug")
		require.Equal(t, zerolog.DebugLevel, cfg.Level)
		require.Nil(t, filter)
	})

	t.Run("module spec returns a filter and leaves the writer unfiltered", func(t *testing.T) {
		cfg, filter := applyLogLevel(t, "*:info,mempool:debug")
		require.Equal(t, zerolog.DebugLevel, cfg.Level)
		require.Nil(t, cfg.Filter, "writer-side filter decodes every line")
		require.NotNil(t, filter)
		// FilterFunc returns true when the entry should be discarded.
		require.False(t, filter("mempool", "debug"), "mempool debug must be kept")
		require.True(t, filter("p2p", "debug"), "non-target debug must be dropped")
		require.False(t, filter("p2p", "info"), "non-target info must be kept")
	})

	t.Run("malformed spec errors", func(t *testing.T) {
		_, _, err := LogLevelOption("mempool:notalevel")
		require.Error(t, err)
	})
}

func TestLogLevelOptionOrDefault(t *testing.T) {
	t.Run("valid spec passes through without warning", func(t *testing.T) {
		warned := false
		_, filter := LogLevelOptionOrDefault("*:info,mempool:debug", func(string, ...any) { warned = true })
		require.NotNil(t, filter)
		require.False(t, warned, "valid spec must not warn")
	})

	t.Run("malformed spec warns and falls back to info", func(t *testing.T) {
		warned := false
		opt, filter := LogLevelOptionOrDefault("mempool:notalevel", func(string, ...any) { warned = true })
		var cfg logger.Config
		opt(&cfg)
		require.Equal(t, zerolog.InfoLevel, cfg.Level)
		require.Nil(t, filter)
		require.True(t, warned, "malformed spec must warn before falling back")
	})
}

func TestWithModuleFilter_NilFilterIsIdentity(t *testing.T) {
	l := logger.NewNopLogger()
	require.Equal(t, l, WithModuleFilter(l, nil))
}

func TestWithModuleFilter_GatesEveryLevel(t *testing.T) {
	type logCall func(logger.Logger)
	calls := map[string]logCall{
		"debug": func(l logger.Logger) { l.Debug("m") },
		"info":  func(l logger.Logger) { l.Info("m") },
		"warn":  func(l logger.Logger) { l.Warn("m") },
		"error": func(l logger.Logger) { l.Error("m") },
	}

	tests := []struct {
		spec   string
		module string
		kept   []string
	}{
		{"*:warn", "p2p", []string{"warn", "error"}},
		{"p2p:error,*:info", "p2p", []string{"error"}},
		{"p2p:error,*:info", "consensus", []string{"info", "warn", "error"}},
		{"mempool:debug,*:error", "mempool", []string{"debug", "info", "warn", "error"}},
	}

	for _, tt := range tests {
		for level, call := range calls {
			t.Run(tt.spec+"/"+tt.module+"/"+level, func(t *testing.T) {
				l, buf := newFilteredLogger(t, tt.spec, nil)
				call(l.With(logger.ModuleKey, tt.module))

				lines := decodeLines(t, buf)
				if !slices.Contains(tt.kept, level) {
					require.Empty(t, lines)
					return
				}
				require.Len(t, lines, 1)
				require.Equal(t, level, lines[0]["level"])
				require.Equal(t, tt.module, lines[0]["module"])
			})
		}
	}
}

// Regression: a dropped line has to stop before zerolog builds it. Gating at
// the writer instead serialized and then decoded every line, which cost
// several cores on busy consensus-mode nodes.
func TestWithModuleFilter_DroppedLineNeverReachesZerolog(t *testing.T) {
	hook := &countingHook{}
	l, buf := newFilteredLogger(t, "p2p:error,*:info", hook)
	p2p := l.With(logger.ModuleKey, "p2p")

	p2p.Info("dropped")
	p2p.Debug("dropped")
	require.Zero(t, hook.n)
	require.Empty(t, buf.String())

	p2p.Error("kept")
	require.Equal(t, 1, hook.n)
}

func TestWithModuleFilter_ModuleResolution(t *testing.T) {
	t.Run("per-call module overrides the With module", func(t *testing.T) {
		l, buf := newFilteredLogger(t, "p2p:error,*:info", nil)
		l.With(logger.ModuleKey, "p2p").Info("kept", logger.ModuleKey, "consensus")
		l.With(logger.ModuleKey, "consensus").Info("dropped", logger.ModuleKey, "p2p")

		out := buf.String()
		require.Contains(t, out, `"message":"kept"`)
		require.NotContains(t, out, `"message":"dropped"`)
	})

	t.Run("module survives a With that adds other keys", func(t *testing.T) {
		l, buf := newFilteredLogger(t, "p2p:error,*:info", nil)
		l.With(logger.ModuleKey, "p2p").With("peer", "abc").Info("dropped")
		require.Empty(t, buf.String())
	})

	t.Run("no module falls back to the default level", func(t *testing.T) {
		l, buf := newFilteredLogger(t, "p2p:error,*:info", nil)
		l.Info("kept")
		l.Debug("dropped")
		out := buf.String()
		require.Contains(t, out, `"message":"kept"`)
		require.NotContains(t, out, `"message":"dropped"`)
	})
}

func TestModuleOf(t *testing.T) {
	tests := []struct {
		name    string
		keyVals []any
		want    string
	}{
		{"no keyvals keeps current", nil, "cur"},
		{"module key sets it", []any{logger.ModuleKey, "p2p"}, "p2p"},
		{"last module wins", []any{logger.ModuleKey, "p2p", logger.ModuleKey, "consensus"}, "consensus"},
		{"other keys are ignored", []any{"peer", "abc", "height", 7}, "cur"},
		{"module after other keys", []any{"peer", "abc", logger.ModuleKey, "p2p"}, "p2p"},
		{"non-string value is ignored", []any{logger.ModuleKey, 7}, "cur"},
		{"non-string key is ignored", []any{7, "p2p"}, "cur"},
		{"dangling key is ignored", []any{logger.ModuleKey}, "cur"},
		{"value named module is not a key", []any{"peer", logger.ModuleKey, "x", "y"}, "cur"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, moduleOf(tt.keyVals, "cur"))
		})
	}
}

type countingHook struct{ n int }

func (h *countingHook) Run(*zerolog.Event, zerolog.Level, string) { h.n++ }

func newFilteredLogger(t *testing.T, spec string, hook zerolog.Hook) (logger.Logger, *bytes.Buffer) {
	t.Helper()
	opt, filter, err := LogLevelOption(spec)
	require.NoError(t, err)

	opts := []logger.Option{opt, logger.OutputJSONOption()}
	if hook != nil {
		opts = append(opts, logger.HooksOption(hook))
	}
	var buf bytes.Buffer
	return WithModuleFilter(logger.NewLogger(&buf, opts...), filter), &buf
}

func decodeLines(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	var lines []map[string]any
	for _, raw := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if raw == "" {
			continue
		}
		var line map[string]any
		require.NoError(t, json.Unmarshal([]byte(raw), &line))
		lines = append(lines, line)
	}
	return lines
}

func applyLogLevel(t *testing.T, s string) (logger.Config, logger.FilterFunc) {
	t.Helper()
	opt, filter, err := LogLevelOption(s)
	require.NoError(t, err)
	var cfg logger.Config
	opt(&cfg)
	return cfg, filter
}
