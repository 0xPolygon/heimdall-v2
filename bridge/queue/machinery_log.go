package queue

import (
	"fmt"
	"os"

	"cosmossdk.io/log"
	machinerylog "github.com/RichardKnop/machinery/v1/log"
)

// machineryLogFunc adapts a cosmossdk.io/log.Logger level method (Debug,
// Info, Warn, or Error) to machinery's logging.LoggerInterface, so machinery's
// internal task/worker logs go through our logger instead of the fixed
// plain-text logger it installs by default.
type machineryLogFunc func(msg string, keyVals ...any)

func (f machineryLogFunc) Print(v ...any)                 { f(fmt.Sprint(v...)) }
func (f machineryLogFunc) Printf(format string, v ...any) { f(fmt.Sprintf(format, v...)) }
func (f machineryLogFunc) Println(v ...any)               { f(fmt.Sprintln(v...)) }
func (f machineryLogFunc) Fatal(v ...any)                 { f(fmt.Sprint(v...)); os.Exit(1) }
func (f machineryLogFunc) Fatalf(format string, v ...any) { f(fmt.Sprintf(format, v...)); os.Exit(1) }
func (f machineryLogFunc) Fatalln(v ...any)               { f(fmt.Sprintln(v...)); os.Exit(1) }
func (f machineryLogFunc) Panic(v ...any)                 { s := fmt.Sprint(v...); f(s); panic(s) }
func (f machineryLogFunc) Panicf(format string, v ...any) {
	s := fmt.Sprintf(format, v...)
	f(s)
	panic(s)
}
func (f machineryLogFunc) Panicln(v ...any) { s := fmt.Sprintln(v...); f(s); panic(s) }

// redirectMachineryLog points machinery's package-level DEBUG/INFO/WARNING/
// ERROR/FATAL loggers at the given logger. Machinery has no per-instance
// logger, only this global registry, so this must run before any queue
// activity (task registration, worker start) that might log. The Set* calls
// are unsynchronized package-level assignments, so this assumes a single
// caller (NewQueueConnector, invoked once at bridge startup) — a second
// concurrent call would race.
func redirectMachineryLog(logger log.Logger) {
	machinerylog.SetDebug(machineryLogFunc(logger.Debug))
	machinerylog.SetInfo(machineryLogFunc(logger.Info))
	machinerylog.SetWarning(machineryLogFunc(logger.Warn))
	machinerylog.SetError(machineryLogFunc(logger.Error))
	// machinery has no FATAL-equivalent level on our logger; log it as an
	// error before exiting so the fatal condition isn't lost, matching the
	// default logger's log-then-os.Exit(1) behavior.
	machinerylog.SetFatal(machineryLogFunc(logger.Error))
}
