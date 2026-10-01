package queue

import (
	"testing"

	"cosmossdk.io/log"
	machinerylog "github.com/RichardKnop/machinery/v1/log"
	"github.com/stretchr/testify/require"
)

// fakeLogger implements cosmossdk.io/log.Logger by dispatching each level to
// an injectable func, so tests can assert exactly which level a call landed on.
type fakeLogger struct {
	debug, info, warn, err func(msg string, keyVals ...any)
}

func (f fakeLogger) Debug(msg string, keyVals ...any) { f.debug(msg, keyVals...) }
func (f fakeLogger) Info(msg string, keyVals ...any)  { f.info(msg, keyVals...) }
func (f fakeLogger) Warn(msg string, keyVals ...any)  { f.warn(msg, keyVals...) }
func (f fakeLogger) Error(msg string, keyVals ...any) { f.err(msg, keyVals...) }
func (f fakeLogger) With(...any) log.Logger           { return f }
func (f fakeLogger) Impl() any                        { return f }

func TestMachineryLogFuncFormatsLikeStdlibLog(t *testing.T) {
	var got string
	capture := machineryLogFunc(func(msg string, keyVals ...any) {
		got = msg
	})

	t.Run("Print concatenates operands", func(t *testing.T) {
		capture.Print("a", "b")
		require.Equal(t, "ab", got)
	})

	t.Run("Printf formats with verbs", func(t *testing.T) {
		capture.Printf("count=%d", 3)
		require.Equal(t, "count=3", got)
	})

	t.Run("Println joins with spaces and trailing newline", func(t *testing.T) {
		capture.Println("a", "b")
		require.Equal(t, "a b\n", got)
	})
}

func TestMachineryLogFuncPanicsOnPanicMethods(t *testing.T) {
	var got string
	capture := machineryLogFunc(func(msg string, keyVals ...any) {
		got = msg
	})

	require.PanicsWithValue(t, "boom", func() { capture.Panic("boom") })
	require.Equal(t, "boom", got, "message must be logged before panicking")

	require.PanicsWithValue(t, "code=42", func() { capture.Panicf("code=%d", 42) })
	require.PanicsWithValue(t, "boom\n", func() { capture.Panicln("boom") })
}

func TestRedirectMachineryLogRoutesEachLevel(t *testing.T) {
	debug, info, warn, errLvl, fatal := "", "", "", "", ""
	redirectMachineryLog(fakeLogger{
		debug: func(msg string, _ ...any) { debug = msg },
		info:  func(msg string, _ ...any) { info = msg },
		warn:  func(msg string, _ ...any) { warn = msg },
		err:   func(msg string, _ ...any) { errLvl = msg; fatal = msg },
	})

	machinerylog.DEBUG.Print("d")
	machinerylog.INFO.Print("i")
	machinerylog.WARNING.Print("w")
	machinerylog.ERROR.Print("e")

	require.Equal(t, "d", debug)
	require.Equal(t, "i", info)
	require.Equal(t, "w", warn)
	require.Equal(t, "e", errLvl)

	// Print never exits, unlike Fatal/Fatalf/Fatalln, so this safely exercises
	// the SetFatal wiring without ending the test process. err's closure feeds
	// both errLvl and fatal, so check fatal here before ERROR's "e" is gone.
	machinerylog.FATAL.Print("f")
	require.Equal(t, "f", fatal, "FATAL must route to our logger's Error level")
}
