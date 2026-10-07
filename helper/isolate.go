package helper

import (
	"errors"
	"fmt"
	"runtime"
)

// ErrRecoveredPanic marks an error as originating from a panic RunIsolated
// recovered, rather than an ordinary error fn returned normally.
var ErrRecoveredPanic = errors.New("recovered from panic in isolated external call")

// RunIsolated runs fn on its own goroutine and blocks until it returns,
// converting a panic into an error instead of letting it crash the process.
// It exists so a fault in an external RPC client (L1, Bor) can't take an
// ABCI++ handler, and therefore consensus, down with it.
//
// fn is expected to already be bounded by its own context deadline, as every
// IContractCaller method is. RunIsolated does not race that deadline against
// a second one of its own (e.g. selecting on ctx.Done() in addition to
// waiting on fn's result): the outcome would then depend on which fires
// first, and different validators could observe different results for the
// same input. The goroutine's termination is therefore bounded by fn's own
// deadline, not by anything RunIsolated adds.
func RunIsolated[T any](fn func() (T, error)) (T, error) {
	type outcome struct {
		val T
		err error
	}

	done := make(chan outcome, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				buf := make([]byte, 1<<16)
				n := runtime.Stack(buf, false)
				Logger.Error("recovered from panic in isolated external call",
					"panic", r, "stack", string(buf[:n]))

				var zero T
				done <- outcome{zero, fmt.Errorf("%w: %v", ErrRecoveredPanic, r)}
			}
		}()

		val, err := fn()
		done <- outcome{val, err}
	}()

	res := <-done
	return res.val, res.err
}
