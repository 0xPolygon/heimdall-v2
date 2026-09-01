package cli

import (
	"errors"
	"strings"
	"testing"
	"time"

	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestBroadcastMsgRetriesAccountSequenceMismatch(t *testing.T) {
	originalBroadcast := broadcastMsgOnceFunc
	originalSleep := broadcastRetrySleep
	t.Cleanup(func() {
		broadcastMsgOnceFunc = originalBroadcast
		broadcastRetrySleep = originalSleep
	})

	sequenceErr := errors.New("account sequence mismatch, expected 8, got 7: incorrect account sequence")
	var attempts int
	var sleeps int

	broadcastMsgOnceFunc = func(clientCtx client.Context, sender string, msg sdk.Msg, logger log.Logger) error {
		attempts++
		if attempts == 1 {
			return sequenceErr
		}

		return nil
	}
	broadcastRetrySleep = func(time.Duration) {
		sleeps++
	}

	err := BroadcastMsg(client.Context{}, "0xabc", nil, log.NewNopLogger())
	if err != nil {
		t.Fatalf("expected no error after retry, got %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
	if sleeps != 1 {
		t.Fatalf("expected 1 retry sleep, got %d", sleeps)
	}
}

func TestRetryBroadcastMsgReturnsOnFirstSuccess(t *testing.T) {
	var attempts int
	var sleeps int

	err := retryBroadcastMsg(func() error {
		attempts++
		return nil
	}, func(time.Duration) {
		sleeps++
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if attempts != 1 {
		t.Fatalf("expected 1 attempt, got %d", attempts)
	}
	if sleeps != 0 {
		t.Fatalf("expected no retry sleeps, got %d", sleeps)
	}
}

func TestRetryBroadcastMsgDoesNotRetryOtherErrors(t *testing.T) {
	wantErr := errors.New("insufficient funds")
	var attempts int

	err := retryBroadcastMsg(func() error {
		attempts++
		return wantErr
	}, func(time.Duration) {
		t.Fatal("did not expect retry sleep")
	})

	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if attempts != 1 {
		t.Fatalf("expected 1 attempt, got %d", attempts)
	}
}

func TestRetryBroadcastMsgStopsWhenRetryReturnsOtherError(t *testing.T) {
	sequenceErr := errors.New("account sequence mismatch, expected 8, got 7: incorrect account sequence")
	wantErr := errors.New("insufficient funds")
	var attempts int
	var sleeps int

	err := retryBroadcastMsg(func() error {
		attempts++
		if attempts == 1 {
			return sequenceErr
		}
		if attempts == 2 {
			return wantErr
		}

		return nil
	}, func(time.Duration) {
		sleeps++
	})

	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
	if sleeps != 1 {
		t.Fatalf("expected 1 retry sleep, got %d", sleeps)
	}
}

func TestRetryBroadcastMsgRetriesAccountSequenceMismatch(t *testing.T) {
	sequenceErr := errors.New("account sequence mismatch, expected 8, got 7: incorrect account sequence")
	var attempts int
	var sleeps []time.Duration

	err := retryBroadcastMsg(func() error {
		attempts++
		if attempts < 3 {
			return sequenceErr
		}
		return nil
	}, func(delay time.Duration) {
		sleeps = append(sleeps, delay)
	})

	if err != nil {
		t.Fatalf("expected no error after retry, got %v", err)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
	if len(sleeps) != 2 {
		t.Fatalf("expected 2 retry sleeps, got %d", len(sleeps))
	}
	for _, delay := range sleeps {
		if delay != broadcastRetryDelay {
			t.Fatalf("expected retry delay %s, got %s", broadcastRetryDelay, delay)
		}
	}
}

func TestRetryBroadcastMsgReturnsFinalSequenceMismatch(t *testing.T) {
	firstErr := errors.New("account sequence mismatch, expected 8, got 7: incorrect account sequence")
	finalErr := errors.New("account sequence mismatch, expected 9, got 8: incorrect account sequence")
	attempts := 0

	err := retryBroadcastMsg(func() error {
		attempts++
		if attempts == 3 {
			return finalErr
		}
		return firstErr
	}, func(time.Duration) {})

	if !errors.Is(err, finalErr) {
		t.Fatalf("expected final error %v, got %v", finalErr, err)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

func TestNonOKBroadcastErrorIncludesRawLog(t *testing.T) {
	rawLog := "account sequence mismatch, expected 8, got 7: incorrect account sequence"

	err := nonOKBroadcastError(32, rawLog)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "response code: 32") {
		t.Fatalf("expected response code in error, got %q", err)
	}
	if !strings.Contains(err.Error(), rawLog) {
		t.Fatalf("expected raw log in error, got %q", err)
	}
}

func TestNonOKBroadcastErrorWithoutRawLog(t *testing.T) {
	err := nonOKBroadcastError(32, "")
	if err == nil {
		t.Fatal("expected error")
	}

	want := "broadcast succeeded but received non-ok response code: 32"
	if err.Error() != want {
		t.Fatalf("expected %q, got %q", want, err.Error())
	}
}
