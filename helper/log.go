package helper

import (
	logger "cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/rs/zerolog"
)

// LogLevelOption converts a log_level string into the matching logger option
// and, for per-module specs, the filter to install with WithModuleFilter.
//
// A plain level ("info", "debug", ...) applies to every module and is gated
// cheaply by zerolog itself; the returned filter is nil. A comma-separated
// list of "module:level" pairs with an optional "*:level" default
// ("*:info,mempool:debug") filters per module instead, so debug can be scoped
// to a single subsystem without the firehose that a global debug level
// produces on a busy node.
//
// The SDK's FilterOption is deliberately not used: it gates at the writer,
// after zerolog has serialized the line, by JSON-decoding every emitted line.
// On a consensus-mode mainnet sentry that cost several cores and pushed
// heimdalld below chain pace.
func LogLevelOption(logLevelStr string) (logger.Option, logger.FilterFunc, error) {
	if logLevelStr == "" {
		return logger.LevelOption(zerolog.InfoLevel), nil, nil
	}

	if lvl, err := zerolog.ParseLevel(logLevelStr); err == nil {
		return logger.LevelOption(lvl), nil, nil
	}

	filter, err := logger.ParseLogLevel(logLevelStr)
	if err != nil {
		return nil, nil, err
	}
	// The filter decides per module; zerolog only needs to let debug through.
	return logger.LevelOption(zerolog.DebugLevel), filter, nil
}

// LogLevelOptionOrDefault is LogLevelOption with a fallback: on a malformed
// spec it warns via warnf and returns the info-level option. A typo in
// log_level then degrades to info visibly, instead of silently mis-filtering.
func LogLevelOptionOrDefault(logLevelStr string, warnf func(string, ...any)) (logger.Option, logger.FilterFunc) {
	opt, filter, err := LogLevelOption(logLevelStr)
	if err != nil {
		warnf("invalid log_level, falling back to info", "log_level", logLevelStr, "error", err)
		return logger.LevelOption(zerolog.InfoLevel), nil
	}
	return opt, filter
}

// WithModuleFilter wraps l so that lines the filter discards are dropped
// before they reach zerolog. A nil filter returns l unchanged. Impl() still
// returns the underlying zerolog logger, which bypasses the filter.
func WithModuleFilter(l logger.Logger, filter logger.FilterFunc) logger.Logger {
	if filter == nil {
		return l
	}
	return moduleFilterLogger{Logger: l, filter: filter}
}

// moduleFilterLogger tracks the module set through With, so the per-line
// check is a map lookup instead of decoding the serialized line.
type moduleFilterLogger struct {
	logger.Logger
	filter logger.FilterFunc
	module string
}

func (l moduleFilterLogger) Info(msg string, keyVals ...any) {
	if l.keep(zerolog.LevelInfoValue, keyVals) {
		l.Logger.Info(msg, keyVals...)
	}
}

func (l moduleFilterLogger) Warn(msg string, keyVals ...any) {
	if l.keep(zerolog.LevelWarnValue, keyVals) {
		l.Logger.Warn(msg, keyVals...)
	}
}

func (l moduleFilterLogger) Error(msg string, keyVals ...any) {
	if l.keep(zerolog.LevelErrorValue, keyVals) {
		l.Logger.Error(msg, keyVals...)
	}
}

func (l moduleFilterLogger) Debug(msg string, keyVals ...any) {
	if l.keep(zerolog.LevelDebugValue, keyVals) {
		l.Logger.Debug(msg, keyVals...)
	}
}

func (l moduleFilterLogger) With(keyVals ...any) logger.Logger {
	return moduleFilterLogger{
		Logger: l.Logger.With(keyVals...),
		filter: l.filter,
		module: moduleOf(keyVals, l.module),
	}
}

func (l moduleFilterLogger) keep(level string, keyVals []any) bool {
	return !l.filter(moduleOf(keyVals, l.module), level)
}

// moduleOf returns the last module value in keyVals, or current if there is
// none. Last wins, matching how the SDK's writer-side filter resolved a
// duplicated module field.
func moduleOf(keyVals []any, current string) string {
	for i := 0; i+1 < len(keyVals); i += 2 {
		if key, ok := keyVals[i].(string); !ok || key != logger.ModuleKey {
			continue
		}
		if module, ok := keyVals[i+1].(string); ok {
			current = module
		}
	}
	return current
}

// LogFormatOption converts a log_format string into the matching logger
// option: flags.OutputFormatJSON ("json") selects JSON output, anything else
// (including empty/unset) selects console output with the given coloring.
func LogFormatOption(logFormat string, noColor bool) logger.Option {
	if logFormat == flags.OutputFormatJSON {
		return logger.OutputJSONOption()
	}
	return logger.ColorOption(!noColor)
}
