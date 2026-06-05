package helper

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"cosmossdk.io/log"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/x/bor/failover"
)

func fakeBorRPC(t *testing.T, chainHex string, hits *int32) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(hits, 1)
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":1,"result":%q}`, chainHex)
	}))
}

func server5xx(t *testing.T, hits *int32) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(hits, 1)
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
}

// slowServer replies only after d (or when the client gives up), so the
// caller's shorter per-attempt timeout fires first. d is bounded so the test's
// server cleanup never blocks indefinitely.
func slowServer(t *testing.T, d time.Duration) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(d):
		case <-r.Context().Done():
		}
	}))
}

func mustEndpoint(t *testing.T, rawurl string) httpEndpoint {
	t.Helper()
	u, err := url.Parse(rawurl)
	require.NoError(t, err)
	rc, err := rpc.Dial(rawurl)
	require.NoError(t, err)
	return httpEndpoint{url: u, probe: ethclient.NewClient(rc)}
}

func mustEndpointURL(t *testing.T, rawurl string) httpEndpoint {
	t.Helper()
	u, err := url.Parse(rawurl)
	require.NoError(t, err)
	return httpEndpoint{url: u}
}

func jsonRPCPost(t *testing.T, rawurl, body string) *http.Request {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, rawurl, strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	return req
}

func newTestTransport(eps ...httpEndpoint) *borHTTPFailoverTransport {
	tr := &borHTTPFailoverTransport{endpoints: eps, base: http.DefaultTransport, attemptTimeout: time.Second}
	tr.health = failover.New(len(eps), tr.probe, failover.Metrics{}, log.NewNopLogger())
	tr.health.SetTuning(5*time.Millisecond, 1, 0, 50*time.Millisecond)
	return tr
}

const dummyReq = `{"jsonrpc":"2.0","id":1,"method":"eth_chainId"}`

func TestDrainRequestBody(t *testing.T) {
	noBody, _ := http.NewRequest(http.MethodPost, "http://x", nil)
	b, err := drainRequestBody(noBody)
	require.NoError(t, err)
	require.Nil(t, b)

	withBody := jsonRPCPost(t, "http://x", "hello")
	b, err = drainRequestBody(withBody)
	require.NoError(t, err)
	require.Equal(t, []byte("hello"), b)
}

func TestBorHTTPFailover_CascadesOnTransportError(t *testing.T) {
	var h2 int32
	good := fakeBorRPC(t, "0x1", &h2)
	defer good.Close()
	down := fakeBorRPC(t, "0x1", new(int32))
	downURL := down.URL
	down.Close() // connections now refused

	tr := newTestTransport(mustEndpoint(t, downURL), mustEndpoint(t, good.URL))
	tr.health.MarkSuccess(1) // secondary is a validated candidate

	resp, err := tr.RoundTrip(jsonRPCPost(t, downURL, dummyReq))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, 1, tr.health.Active())
	require.Positive(t, atomic.LoadInt32(&h2))
}

func TestBorHTTPFailover_MarksActiveSuccess(t *testing.T) {
	s := fakeBorRPC(t, "0x1", new(int32))
	defer s.Close()

	tr := newTestTransport(mustEndpoint(t, s.URL))
	tr.health.MarkUnhealthy(0, errors.New("transient"))

	resp, err := tr.RoundTrip(jsonRPCPost(t, s.URL, dummyReq))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, []int{0}, tr.health.Candidates(1))
}

func TestBorHTTPFailover_CascadesOn5xx(t *testing.T) {
	var bad, good int32
	bad5xx := server5xx(t, &bad)
	defer bad5xx.Close()
	healthy := fakeBorRPC(t, "0x1", &good)
	defer healthy.Close()

	tr := newTestTransport(mustEndpoint(t, bad5xx.URL), mustEndpoint(t, healthy.URL))
	tr.health.MarkSuccess(1)

	resp, err := tr.RoundTrip(jsonRPCPost(t, bad5xx.URL, dummyReq))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode) // failed over off the 503
	require.Equal(t, 1, tr.health.Active())
	require.Positive(t, atomic.LoadInt32(&bad)) // primary was tried and 5xx'd
}

func TestBorHTTPFailover_UsesFallbackURLQueryAndUser(t *testing.T) {
	var calls int
	base := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			require.Equal(t, "primary.example", r.URL.Host)
			require.Equal(t, "/primary", r.URL.Path)
			require.Equal(t, "token=primary", r.URL.RawQuery)
			return nil, errors.New("primary down")
		}

		require.Equal(t, "fallback.example", r.URL.Host)
		require.Equal(t, "fallback.example", r.Host)
		require.Equal(t, "/fallback", r.URL.Path)
		require.Equal(t, "token=fallback", r.URL.RawQuery)
		user := r.URL.User
		require.NotNil(t, user)
		require.Equal(t, "fallback-user", user.Username())
		password, ok := user.Password()
		require.True(t, ok)
		require.Equal(t, "fallback-pass", password)

		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"jsonrpc":"2.0","id":1,"result":"0x1"}`)),
			Request:    r,
		}, nil
	})

	tr := &borHTTPFailoverTransport{
		endpoints: []httpEndpoint{
			mustEndpointURL(t, "https://primary-user:primary-pass@primary.example/primary?token=primary"),
			mustEndpointURL(t, "https://fallback-user:fallback-pass@fallback.example/fallback?token=fallback"),
		},
		base:           base,
		attemptTimeout: time.Second,
	}
	tr.health = failover.New(2, tr.probe, failover.Metrics{}, log.NewNopLogger())
	tr.health.SetTuning(5*time.Millisecond, 1, 0, 50*time.Millisecond)
	tr.health.MarkSuccess(1)

	resp, err := tr.RoundTrip(jsonRPCPost(t, "https://primary-user:primary-pass@primary.example/primary?token=primary", dummyReq))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, 2, calls)
}

func TestBorHTTPFailover_All5xxPreservesLastResponse(t *testing.T) {
	s1 := server5xx(t, new(int32))
	defer s1.Close()
	s2 := server5xx(t, new(int32))
	defer s2.Close()

	tr := newTestTransport(mustEndpoint(t, s1.URL), mustEndpoint(t, s2.URL))
	tr.health.MarkSuccess(1)

	resp, err := tr.RoundTrip(jsonRPCPost(t, s1.URL, dummyReq))
	require.NoError(t, err) // not a transport error — the real 5xx is preserved
	defer resp.Body.Close()
	require.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestBorHTTPFailover_PerAttemptTimeoutCascades(t *testing.T) {
	slow := slowServer(t, 300*time.Millisecond)
	defer slow.Close()
	var good int32
	healthy := fakeBorRPC(t, "0x1", &good)
	defer healthy.Close()

	tr := newTestTransport(mustEndpoint(t, slow.URL), mustEndpoint(t, healthy.URL))
	tr.attemptTimeout = 30 * time.Millisecond // primary stalls past this
	tr.health.MarkSuccess(1)

	resp, err := tr.RoundTrip(jsonRPCPost(t, slow.URL, dummyReq))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, 1, tr.health.Active())
}

func TestBorHTTPFailover_IntegrationCascadeThroughRPCClient(t *testing.T) {
	// Exercises the real path: ethclient -> rpc.Client.CallContext -> our
	// transport. After the transport fails over, rpc.Client.op.wait re-checks the
	// caller's context, so the caller budget must span the whole cascade (as
	// GetBorChainCallTimeout sizes it) or a successful fallback races an expired
	// deadline.
	slow := slowServer(t, 200*time.Millisecond)
	defer slow.Close()
	healthy := fakeBorRPC(t, "0x1", new(int32))
	defer healthy.Close()

	tr := newTestTransport(mustEndpoint(t, slow.URL), mustEndpoint(t, healthy.URL))
	tr.attemptTimeout = 40 * time.Millisecond
	tr.health.MarkSuccess(1)

	rc, err := rpc.DialOptions(context.Background(), tr.endpoints[0].url.String(),
		rpc.WithHTTPClient(&http.Client{Transport: tr}))
	require.NoError(t, err)
	ec := ethclient.NewClient(rc)

	ctx, cancel := context.WithTimeout(context.Background(), 3*40*time.Millisecond) // per-attempt x endpoint count
	defer cancel()
	id, err := ec.ChainID(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(1), id.Int64())
	require.Equal(t, 1, tr.health.Active())
}

func TestBorHTTPFailover_NoCascadeWhenCallerCtxDone(t *testing.T) {
	down := fakeBorRPC(t, "0x1", new(int32))
	downURL := down.URL
	down.Close()
	var h2 int32
	s2 := fakeBorRPC(t, "0x1", &h2)
	defer s2.Close()

	tr := newTestTransport(mustEndpoint(t, downURL), mustEndpoint(t, s2.URL))
	tr.health.MarkSuccess(1)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := tr.RoundTrip(jsonRPCPost(t, downURL, dummyReq).WithContext(ctx))
	require.Error(t, err)
	require.Zero(t, atomic.LoadInt32(&h2)) // secondary never tried after cancellation
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestBorHTTPFailover_StopsCascadeOnMidLoopCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var calls int32
	base := roundTripFunc(func(*http.Request) (*http.Response, error) {
		if atomic.AddInt32(&calls, 1) == 2 {
			cancel() // caller abandons the call mid-cascade
		}
		return nil, errors.New("transport down")
	})

	tr := &borHTTPFailoverTransport{
		endpoints:      []httpEndpoint{mustEndpoint(t, "http://a.invalid"), mustEndpoint(t, "http://b.invalid"), mustEndpoint(t, "http://c.invalid")},
		base:           base,
		attemptTimeout: time.Second,
	}
	tr.health = failover.New(3, tr.probe, failover.Metrics{}, log.NewNopLogger())
	tr.health.SetTuning(5*time.Millisecond, 1, 0, 50*time.Millisecond)
	tr.health.MarkSuccess(1)
	tr.health.MarkSuccess(2)

	_, err := tr.RoundTrip(jsonRPCPost(t, "http://a.invalid", dummyReq).WithContext(ctx))
	require.Error(t, err)
	require.Equal(t, int32(2), atomic.LoadInt32(&calls)) // third endpoint never tried after cancel
}

func TestBorHTTPFailover_NoCascadeToUnvalidatedFallback(t *testing.T) {
	down := fakeBorRPC(t, "0x1", new(int32))
	downURL := down.URL
	down.Close()
	var h2 int32
	s2 := fakeBorRPC(t, "0x1", &h2) // never marked healthy (e.g., failed identity)
	defer s2.Close()

	tr := newTestTransport(mustEndpoint(t, downURL), mustEndpoint(t, s2.URL))

	_, err := tr.RoundTrip(jsonRPCPost(t, downURL, dummyReq))
	require.Error(t, err)
	require.Zero(t, atomic.LoadInt32(&h2)) // unvalidated fallback is never used
}

func TestBorHTTPTransport_ProbeValidatesChainID(t *testing.T) {
	primary := fakeBorRPC(t, "0x5", new(int32))
	defer primary.Close()
	match := fakeBorRPC(t, "0x5", new(int32))
	defer match.Close()
	wrong := fakeBorRPC(t, "0x63", new(int32)) // different chain id
	defer wrong.Close()

	tr := newTestTransport(mustEndpoint(t, primary.URL), mustEndpoint(t, match.URL), mustEndpoint(t, wrong.URL))

	require.NoError(t, tr.probe(0)) // primary establishes the expected chain id
	require.NoError(t, tr.probe(1)) // same chain id → ok
	require.Error(t, tr.probe(2))   // mismatched chain id → rejected
}

func TestBorHTTPTransport_PrimaryReclaimAdoptsActive(t *testing.T) {
	primary := fakeBorRPC(t, "0x1", new(int32))
	defer primary.Close()
	fallback := fakeBorRPC(t, "0x2", new(int32)) // different (wrong) network
	defer fallback.Close()

	tr := newTestTransport(mustEndpoint(t, primary.URL), mustEndpoint(t, fallback.URL))
	// Boot window: the primary was unreachable, so the fallback provisionally
	// anchored its own identity and became the active endpoint.
	tr.primaryProbeFailures.Store(primaryAnchorFailureThreshold)
	require.NoError(t, tr.probe(1))
	tr.health.SetActive(1)
	require.Equal(t, 1, tr.health.Active())

	// The primary recovers: its probe reclaims identity and moves traffic back at once.
	require.NoError(t, tr.probe(0))
	require.Equal(t, 0, tr.health.Active())
}

func TestBorHTTPTransport_ProbeRejectsUnreachable(t *testing.T) {
	down := fakeBorRPC(t, "0x1", new(int32))
	downURL := down.URL
	down.Close()

	tr := newTestTransport(mustEndpoint(t, downURL))
	require.Error(t, tr.probe(0))
}

func TestBorHTTPFailover_RPCClientRoutesAndCascades(t *testing.T) {
	var h1, h2 int32
	s1 := fakeBorRPC(t, "0x1", &h1)
	s2 := fakeBorRPC(t, "0x1", &h2)
	defer s2.Close()

	tr := newTestTransport(mustEndpoint(t, s1.URL), mustEndpoint(t, s2.URL))
	tr.health.MarkSuccess(1)

	rc, err := rpc.DialOptions(context.Background(), tr.endpoints[0].url.String(),
		rpc.WithHTTPClient(&http.Client{Transport: tr}))
	require.NoError(t, err)
	ec := ethclient.NewClient(rc)

	id, err := ec.ChainID(context.Background())
	require.NoError(t, err)
	require.Equal(t, int64(1), id.Int64())

	s1.Close() // primary down → next call cascades in-line
	id, err = ec.ChainID(context.Background())
	require.NoError(t, err)
	require.Equal(t, int64(1), id.Int64())
	require.Equal(t, 1, tr.health.Active())
	require.Positive(t, atomic.LoadInt32(&h2))
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("read failed") }
func (errReader) Close() error             { return nil }

func TestRoundTrip_BodyReadError(t *testing.T) {
	s := fakeBorRPC(t, "0x1", new(int32))
	defer s.Close()

	tr := newTestTransport(mustEndpoint(t, s.URL))
	req, err := http.NewRequest(http.MethodPost, s.URL, errReader{})
	require.NoError(t, err)

	_, err = tr.RoundTrip(req)
	require.Error(t, err)
}

func TestNewBorHTTPFailoverClient_ErrorWhenNoValidEndpoints(t *testing.T) {
	_, _, err := newBorHTTPFailoverClient([]string{"ws://x", "ftp://y"}, time.Second)
	require.Error(t, err)
}

func TestDialHTTPEndpoints_SkipsInvalidFallbackScheme(t *testing.T) {
	s1 := fakeBorRPC(t, "0x1", new(int32))
	defer s1.Close()

	// "://bad" fails url.Parse (nil URL); "ws://" parses but is the wrong scheme.
	got, err := dialHTTPEndpoints([]string{s1.URL, "://bad", "ws://localhost:8546"})
	require.NoError(t, err)
	require.Len(t, got, 1)
}

func TestDialHTTPEndpoints_RejectsInvalidPrimary(t *testing.T) {
	s1 := fakeBorRPC(t, "0x1", new(int32))
	defer s1.Close()

	_, err := dialHTTPEndpoints([]string{"ws://localhost:8546", s1.URL})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid primary")

	_, err = dialHTTPEndpoints([]string{"://bad", s1.URL})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid primary")

	_, err = dialHTTPEndpoints([]string{"ws://localhost:8546"})
	require.Error(t, err)
}

func TestFetchChainID(t *testing.T) {
	s := fakeBorRPC(t, "0x9", new(int32))
	defer s.Close()
	id, ok := fetchChainID(mustEndpoint(t, s.URL).probe, time.Second)
	require.True(t, ok)
	require.Equal(t, int64(9), id.Int64())

	down := fakeBorRPC(t, "0x9", new(int32))
	downURL := down.URL
	down.Close()
	id2, ok2 := fetchChainID(mustEndpoint(t, downURL).probe, 200*time.Millisecond)
	require.False(t, ok2)
	require.Nil(t, id2)
}

func TestCaptureExpectedChainID(t *testing.T) {
	s := fakeBorRPC(t, "0x7", new(int32))
	defer s.Close()
	tr := &borHTTPFailoverTransport{endpoints: []httpEndpoint{mustEndpoint(t, s.URL)}, attemptTimeout: time.Second}
	tr.captureExpectedChainID()
	require.NotNil(t, tr.expectedChainID.Load())
	require.Equal(t, int64(7), tr.expectedChainID.Load().Int64())

	down := fakeBorRPC(t, "0x7", new(int32))
	downURL := down.URL
	down.Close()
	tr2 := &borHTTPFailoverTransport{endpoints: []httpEndpoint{mustEndpoint(t, downURL)}, attemptTimeout: 200 * time.Millisecond}
	tr2.captureExpectedChainID()
	require.Nil(t, tr2.expectedChainID.Load()) // unreachable primary → nothing captured
}

type trackCloser struct{ closed bool }

func (c *trackCloser) Read([]byte) (int, error) { return 0, io.EOF }
func (c *trackCloser) Close() error             { c.closed = true; return errors.New("close boom") }

func TestCancelOnClose(t *testing.T) {
	var cancelled bool
	body := &trackCloser{}
	c := &cancelOnClose{body: body, cancel: func() { cancelled = true }}

	err := c.Close()
	require.Error(t, err)        // the underlying body's Close error is propagated
	require.True(t, cancelled)   // the per-attempt context is cancelled
	require.True(t, body.closed) // the underlying body is closed
}
