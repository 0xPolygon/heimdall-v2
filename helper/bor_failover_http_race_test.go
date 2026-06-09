package helper

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"cosmossdk.io/log"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/x/bor/failover"
	borgrpc "github.com/0xPolygon/heimdall-v2/x/bor/grpc"
)

// closeCountingGRPCClient embeds borgrpc.Client so it satisfies the interface
// without implementing every method; only Close is exercised here.
type closeCountingGRPCClient struct {
	borgrpc.Client
	closes atomic.Int32
}

func (c *closeCountingGRPCClient) Close(log.Logger) { c.closes.Add(1) }

// TestCloseBorChainClients_ConcurrentIsRaceFree drives the two shutdown paths
// (bridge teardown and the start-command cleanup goroutine) that reach
// CloseBorChainClients on the same SIGTERM. The package-level client pointers
// are read and written there, so without synchronization this trips -race.
func TestCloseBorChainClients_ConcurrentIsRaceFree(t *testing.T) {
	fake := &closeCountingGRPCClient{}
	borGRPCClient = fake
	t.Cleanup(func() { borGRPCClient = nil })

	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			CloseBorChainClients()
		}()
	}
	wg.Wait()

	require.Nil(t, borGRPCClient)
	require.Equal(t, int32(1), fake.closes.Load(), "set-to-nil under the lock must make the second call a no-op")
}

func TestBorHTTPFailover_SkipsCandidateDemotedBeforePromotion(t *testing.T) {
	tr := &borHTTPFailoverTransport{
		endpoints: []httpEndpoint{
			mustEndpointURL(t, "https://primary.example"),
			mustEndpointURL(t, "https://fallback.example"),
		},
		attemptTimeout: time.Second,
	}
	tr.base = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host == "fallback.example" {
			tr.health.Reclaim(0) // prober reclaimed while the fallback attempt was in flight
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"jsonrpc":"2.0","id":1,"result":"0x1"}`)),
				Request:    r,
			}, nil
		}
		return nil, errors.New("primary down")
	})
	tr.health = failover.New(2, tr.probe, failover.Metrics{}, log.NewNopLogger())
	tr.health.SetTuning(5*time.Millisecond, 1, 0, 50*time.Millisecond)
	tr.health.MarkSuccess(1)

	resp, err := tr.RoundTrip(jsonRPCPost(t, "https://primary.example", dummyReq))
	require.Error(t, err)
	require.Nil(t, resp)
	require.Equal(t, 0, tr.health.Active())
	require.Empty(t, tr.health.Candidates(0))
}
