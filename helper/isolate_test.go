package helper

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunIsolated(t *testing.T) {
	sentinelErr := errors.New("boom")

	tests := []struct {
		name    string
		fn      func() (int, error)
		wantVal int
		wantErr error
		// panicErr, when set, means we expect an error but can't compare it
		// with require.ErrorIs (the panic message is wrapped, not the
		// original error), so we only assert it's non-nil and contains the text.
		wantPanicSubstr string
	}{
		{
			name: "success path returns the value and nil error unchanged",
			fn: func() (int, error) {
				return 42, nil
			},
			wantVal: 42,
			wantErr: nil,
		},
		{
			name: "a plain error from fn propagates unchanged",
			fn: func() (int, error) {
				return 0, sentinelErr
			},
			wantVal: 0,
			wantErr: sentinelErr,
		},
		{
			name: "a panic with an error value is recovered as an error, not a crash",
			fn: func() (int, error) {
				panic(sentinelErr)
			},
			wantVal:         0,
			wantPanicSubstr: "boom",
		},
		{
			name: "a panic with a string value is recovered as an error, not a crash",
			fn: func() (int, error) {
				panic("client exploded")
			},
			wantVal:         0,
			wantPanicSubstr: "client exploded",
		},
		{
			name: "a nil-pointer-dereference-style panic is recovered as an error, not a crash",
			fn: func() (int, error) {
				var p *int
				return *p, nil //nolint:govet // deliberate nil deref to exercise panic recovery
			},
			wantVal:         0,
			wantPanicSubstr: "nil pointer dereference",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := RunIsolated(tt.fn)

			require.Equal(t, tt.wantVal, val)

			switch {
			case tt.wantPanicSubstr != "":
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantPanicSubstr)
				require.ErrorIs(t, err, ErrRecoveredPanic,
					"a recovered panic must be identifiable via errors.Is(err, ErrRecoveredPanic)")
			case tt.wantErr != nil:
				require.ErrorIs(t, err, tt.wantErr)
				require.NotErrorIs(t, err, ErrRecoveredPanic,
					"an ordinary returned error must not match ErrRecoveredPanic")
			default:
				require.NoError(t, err)
			}
		})
	}
}

// TestRunIsolated_MultipleValues exercises RunIsolated with a struct type,
// matching how call sites pack multi-return-value external calls (e.g.
// GetHeaderInfo, GetBorChainBlockInfoInBatch) across the isolation boundary.
func TestRunIsolated_MultipleValues(t *testing.T) {
	type packed struct {
		a int
		b string
	}

	val, err := RunIsolated(func() (packed, error) {
		return packed{a: 7, b: "seven"}, nil
	})

	require.NoError(t, err)
	require.Equal(t, packed{a: 7, b: "seven"}, val)
}
