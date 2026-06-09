package heimdalld

import (
	"bytes"
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"cosmossdk.io/log"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
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
		var borClientsClosed atomic.Bool

		require.NoError(t, applyBorFailoverBPGuard(log.NewLogger(&buf), fake, func() {
			borClientsClosed.Store(true)
		}))
		require.False(t, fake.closeCalled)
		require.False(t, borClientsClosed.Load())
		require.NotContains(t, buf.String(), "failed to close app")
	})

	t.Run("guard fails: app closed, guard error returned", func(t *testing.T) {
		var buf bytes.Buffer
		fake := &fakeGuardedApp{guardErr: guardErr}
		var borClientsClosed atomic.Bool

		err := applyBorFailoverBPGuard(log.NewLogger(&buf), fake, func() {
			borClientsClosed.Store(true)
		})
		require.ErrorIs(t, err, guardErr)
		require.True(t, fake.closeCalled)
		require.True(t, borClientsClosed.Load())
		require.NotContains(t, buf.String(), "failed to close app")
	})

	t.Run("guard fails and close fails: guard error returned, close error logged", func(t *testing.T) {
		var buf bytes.Buffer
		fake := &fakeGuardedApp{guardErr: guardErr, closeErr: errors.New("close boom")}
		var borClientsClosed atomic.Bool

		err := applyBorFailoverBPGuard(log.NewLogger(&buf), fake, func() {
			borClientsClosed.Store(true)
		})
		require.ErrorIs(t, err, guardErr)
		require.True(t, fake.closeCalled)
		require.True(t, borClientsClosed.Load())
		require.Contains(t, buf.String(), "failed to close app")
	})
}

func TestMustApplyBorFailoverBPGuard(t *testing.T) {
	t.Run("guard fails: start path panics (fails closed)", func(t *testing.T) {
		fake := &fakeGuardedApp{guardErr: errors.New("bp guard tripped")}
		var borClientsClosed atomic.Bool
		require.Panics(t, func() {
			mustApplyBorFailoverBPGuard(log.NewNopLogger(), fake, func() {
				borClientsClosed.Store(true)
			})
		})
		require.True(t, fake.closeCalled)
		require.True(t, borClientsClosed.Load())
	})

	t.Run("guard passes: start path proceeds", func(t *testing.T) {
		fake := &fakeGuardedApp{}
		var borClientsClosed atomic.Bool
		require.NotPanics(t, func() {
			mustApplyBorFailoverBPGuard(log.NewNopLogger(), fake, func() {
				borClientsClosed.Store(true)
			})
		})
		require.False(t, fake.closeCalled)
		require.False(t, borClientsClosed.Load())
	})
}

func TestRegisterBorChainClientCleanup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var g errgroup.Group
	var closed atomic.Bool

	registerBorChainClientCleanup(ctx, &g, func() {
		closed.Store(true)
	})

	require.False(t, closed.Load())
	cancel()
	require.NoError(t, g.Wait())
	require.True(t, closed.Load())
}
