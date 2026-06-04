package service

import (
	"context"
	"errors"
	"testing"
	"time"

	common "github.com/cometbft/cometbft/libs/service"
	"github.com/stretchr/testify/require"
)

// fakeService satisfies common.Service via the embedded (nil) interface;
// only the lifecycle methods runServices/stopBridge call are overridden.
type fakeService struct {
	common.Service
	running  bool
	startErr error
	stopErr  error
	stopped  bool
	quit     chan struct{}
}

func (f *fakeService) Start() error    { return f.startErr }
func (f *fakeService) IsRunning() bool { return f.running }
func (f *fakeService) Stop() error     { f.stopped = true; return f.stopErr }
func (f *fakeService) Quit() <-chan struct{} {
	if f.quit == nil {
		f.quit = make(chan struct{})
	}
	return f.quit
}

type fakeStoppable struct {
	err    error
	called bool
}

func (f *fakeStoppable) Stop() error { f.called = true; return f.err }

type fakeWorkerStopper struct{ called bool }

func (f *fakeWorkerStopper) StopWorker() { f.called = true }

func TestStopBridge_StopsRunningServicesAndClients(t *testing.T) {
	running := &fakeService{running: true}
	idle := &fakeService{running: false}
	qc := &fakeWorkerStopper{}
	hc := &fakeStoppable{}

	require.NoError(t, stopBridge([]common.Service{running, idle}, hc, qc))
	require.True(t, qc.called)       // worker stopped
	require.True(t, running.stopped) // running service stopped
	require.False(t, idle.stopped)   // non-running service skipped
	require.True(t, hc.called)       // comet client stopped
}

func TestStopBridge_ReturnsServiceStopError(t *testing.T) {
	boom := errors.New("service stop failed")
	bad := &fakeService{running: true, stopErr: boom}
	hc := &fakeStoppable{}

	require.ErrorIs(t, stopBridge([]common.Service{bad}, hc, &fakeWorkerStopper{}), boom)
	require.False(t, hc.called) // returns before stopping the comet client
}

func TestStopBridge_ReturnsHTTPClientStopError(t *testing.T) {
	boom := errors.New("http client stop failed")
	hc := &fakeStoppable{err: boom}

	require.ErrorIs(t, stopBridge(nil, hc, &fakeWorkerStopper{}), boom)
}

func TestRunServices_PropagatesShutdownError(t *testing.T) {
	boom := errors.New("service stop failed")
	svc := &fakeService{running: true, stopErr: boom, quit: make(chan struct{})}

	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(20 * time.Millisecond); cancel() }() // trigger the shutdown controller

	err := runServices(ctx, []common.Service{svc}, &fakeStoppable{}, &fakeWorkerStopper{})
	require.ErrorIs(t, err, boom) // the shutdown error propagates out of runServices
	require.True(t, svc.stopped)
}

func TestRunServices_PropagatesServiceStartError(t *testing.T) {
	boom := errors.New("service start failed")
	svc := &fakeService{startErr: boom, quit: make(chan struct{})}

	err := runServices(context.Background(), []common.Service{svc}, &fakeStoppable{}, &fakeWorkerStopper{})
	require.ErrorIs(t, err, boom) // a service Start failure cancels the group and surfaces
}
