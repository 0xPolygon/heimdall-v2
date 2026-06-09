package heimdalld

import (
	"bytes"
	"errors"
	"testing"

	"cosmossdk.io/log"
	"github.com/stretchr/testify/require"
)

type fakeGuardedApp struct {
	guardErr    error
	closeErr    error
	closeCalled bool
}

func (f *fakeGuardedApp) EnforceBorFailoverBPGuard() error { return f.guardErr }

func (f *fakeGuardedApp) Close() error {
	f.closeCalled = true
	return f.closeErr
}

func TestApplyBorFailoverBPGuard(t *testing.T) {
	guardErr := errors.New("bp guard tripped")

	t.Run("guard passes: app not closed, no error", func(t *testing.T) {
		var buf bytes.Buffer
		fake := &fakeGuardedApp{}

		require.NoError(t, applyBorFailoverBPGuard(log.NewLogger(&buf), fake))
		require.False(t, fake.closeCalled)
		require.NotContains(t, buf.String(), "failed to close app")
	})

	t.Run("guard fails: app closed, guard error returned", func(t *testing.T) {
		var buf bytes.Buffer
		fake := &fakeGuardedApp{guardErr: guardErr}

		err := applyBorFailoverBPGuard(log.NewLogger(&buf), fake)
		require.ErrorIs(t, err, guardErr)
		require.True(t, fake.closeCalled)
		require.NotContains(t, buf.String(), "failed to close app")
	})

	t.Run("guard fails and close fails: guard error returned, close error logged", func(t *testing.T) {
		var buf bytes.Buffer
		fake := &fakeGuardedApp{guardErr: guardErr, closeErr: errors.New("close boom")}

		err := applyBorFailoverBPGuard(log.NewLogger(&buf), fake)
		require.ErrorIs(t, err, guardErr)
		require.True(t, fake.closeCalled)
		require.Contains(t, buf.String(), "failed to close app")
	})
}

func TestMustApplyBorFailoverBPGuard(t *testing.T) {
	t.Run("guard fails: start path panics (fails closed)", func(t *testing.T) {
		fake := &fakeGuardedApp{guardErr: errors.New("bp guard tripped")}
		require.Panics(t, func() {
			mustApplyBorFailoverBPGuard(log.NewNopLogger(), fake)
		})
		require.True(t, fake.closeCalled)
	})

	t.Run("guard passes: start path proceeds", func(t *testing.T) {
		fake := &fakeGuardedApp{}
		require.NotPanics(t, func() {
			mustApplyBorFailoverBPGuard(log.NewNopLogger(), fake)
		})
		require.False(t, fake.closeCalled)
	})
}
