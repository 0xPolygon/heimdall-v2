package helper

import (
	logger "cosmossdk.io/log"
	"github.com/rs/zerolog"
)

// LogLevelOption converts a log_level string into the matching logger option.
//
// A plain level ("info", "debug", ...) applies to every module and is gated
// cheaply by zerolog itself. A comma-separated list of "module:level" pairs
// with an optional "*:level" default ("*:info,mempool:debug") filters per
// module instead, so debug can be scoped to a single subsystem without the
// firehose that a global debug level produces on a busy node.
func LogLevelOption(logLevelStr string) (logger.Option, error) {
	if logLevelStr == "" {
		return logger.LevelOption(zerolog.InfoLevel), nil
	}

	if lvl, err := zerolog.ParseLevel(logLevelStr); err == nil {
		return logger.LevelOption(lvl), nil
	}

	filter, err := logger.ParseLogLevel(logLevelStr)
	if err != nil {
		return nil, err
	}
	return logger.FilterOption(filter), nil
}

// LogLevelOptionOrDefault is LogLevelOption with a fallback: on a malformed
// spec it warns via warnf and returns the info-level option. A typo in
// log_level then degrades to info visibly, instead of silently mis-filtering.
func LogLevelOptionOrDefault(logLevelStr string, warnf func(string, ...any)) logger.Option {
	opt, err := LogLevelOption(logLevelStr)
	if err != nil {
		warnf("invalid log_level, falling back to info", "log_level", logLevelStr, "error", err)
		return logger.LevelOption(zerolog.InfoLevel)
	}
	return opt
}
