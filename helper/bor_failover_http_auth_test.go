package helper

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"cosmossdk.io/log"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/x/bor/failover"
)

func TestBorHTTPFailover_RewritesBasicAuthPerEndpoint(t *testing.T) {
	var calls int
	base := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		user, pass, ok := r.BasicAuth()
		require.True(t, ok)
		if calls == 1 {
			require.Equal(t, "primary.example", r.URL.Host)
			require.Equal(t, "primary-user", user)
			require.Equal(t, "primary-pass", pass)
			return nil, errors.New("primary down")
		}

		require.Equal(t, "fallback.example", r.URL.Host)
		require.Equal(t, "fallback-user", user)
		require.Equal(t, "fallback-pass", pass)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"jsonrpc":"2.0","id":1,"result":"0x1"}`)),
			Request:    r,
		}, nil
	})

	tr := &borHTTPFailoverTransport{
		endpoints: []httpEndpoint{
			mustEndpointURL(t, "https://primary-user:primary-pass@primary.example"),
			mustEndpointURL(t, "https://fallback-user:fallback-pass@fallback.example"),
		},
		base:           base,
		attemptTimeout: time.Second,
	}
	tr.health = failover.New(2, tr.probe, failover.Metrics{}, log.NewNopLogger())
	tr.health.SetTuning(5*time.Millisecond, 1, 0, 50*time.Millisecond)
	tr.health.MarkSuccess(1)

	resp, err := (&http.Client{Transport: tr}).Do(jsonRPCPost(t, "https://primary-user:primary-pass@primary.example", dummyReq))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, 2, calls)
}
