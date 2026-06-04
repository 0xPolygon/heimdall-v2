package helper

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/0xPolygon/heimdall-v2/metrics"
	"github.com/0xPolygon/heimdall-v2/x/bor/failover"
)

const (
	primaryEndpoint = 0

	// primaryAnchorFailureThreshold is how many consecutive primary probe
	// failures must accrue before a fallback may provisionally establish the
	// expected Bor chain identity. The primary stays authoritative: once it is
	// reachable it reclaims the expectation (see checkChainID), so this only
	// lets failover engage during a window where the primary is unreachable.
	primaryAnchorFailureThreshold = 2

	// maxBudgetedEndpoints caps how many per-endpoint attempts a single Bor call's
	// time budget covers (BorRPCTimeout each), so a large endpoint count can't push
	// one call's worst case past CometBFT's ~10s ABCI budget. It is a time bound,
	// not a hard attempt count: a cascade still tries currently-healthy candidates
	// in priority order and may try more than this many if they fail fast enough to
	// fit the budget; it is just never granted time for more than this many slow
	// attempts. The deadline, not a counter, stops the cascade.
	maxBudgetedEndpoints = 3
)

// borRPCFailoverTransport holds the running HTTP failover transport so
// CloseBorChainClients can stop its prober and close its per-endpoint probe
// clients on shutdown; nil when HTTP failover is not configured.
var borRPCFailoverTransport *borHTTPFailoverTransport

type chainIDProbe interface {
	ChainID(context.Context) (*big.Int, error)
	Close()
}

// httpEndpoint pairs a Bor HTTP JSON-RPC URL with a single-endpoint client used
// only for health/identity probes (the actual traffic flows through the shared
// rpc.Client whose transport is the failover one).
type httpEndpoint struct {
	url   *url.URL
	probe chainIDProbe
}

// borHTTPFailoverTransport is an http.RoundTripper that sends each Bor JSON-RPC
// request to the active endpoint and, on a transport failure, a per-attempt
// timeout, or a 5xx response, cascades to the next validated endpoint in
// priority order. A background prober reverts to a higher-priority endpoint
// once it recovers.
type borHTTPFailoverTransport struct {
	endpoints            []httpEndpoint
	base                 http.RoundTripper
	health               *failover.Health
	attemptTimeout       time.Duration
	expectedChainID      atomic.Pointer[big.Int]
	expectedByPrimary    atomic.Bool
	primaryProbeFailures atomic.Int32
}

func (t *borHTTPFailoverTransport) Close() {
	if t == nil {
		return
	}
	if t.health != nil {
		t.health.Stop()
	}
	for _, ep := range t.endpoints {
		if ep.probe != nil {
			ep.probe.Close()
		}
	}
}

func (t *borHTTPFailoverTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	body, err := drainRequestBody(req)
	if err != nil {
		return nil, err
	}

	active := t.health.Active()
	resp, err := t.attempt(req, body, active)
	if succeeded(resp, err) || req.Context().Err() != nil {
		return resp, err
	}
	t.markFailed(active, resp, err)

	for _, i := range t.health.Candidates(active) {
		drainAndClose(resp)
		resp, err = t.attempt(req, body, i)
		if succeeded(resp, err) {
			t.health.Promote(active, i)
			return resp, err
		}
		if req.Context().Err() != nil {
			return resp, err
		}
		t.markFailed(i, resp, err)
	}

	return resp, err
}

// succeeded reports a usable response: a transport success whose status is not
// a server-side (5xx) error. A 4xx is returned as-is (a fallback would answer
// identically); a 5xx is treated as a failure worth cascading.
func succeeded(resp *http.Response, err error) bool {
	return err == nil && resp != nil && resp.StatusCode < http.StatusInternalServerError
}

func (t *borHTTPFailoverTransport) markFailed(i int, resp *http.Response, err error) {
	if err == nil && resp != nil {
		err = fmt.Errorf("bor endpoint returned http %d", resp.StatusCode)
	}
	t.health.MarkUnhealthy(i, err)
}

// attempt sends the request to endpoint i with its own attempt timeout. On
// success the timeout context is tied to the response body's Close so it stays
// alive while the caller reads the body.
func (t *borHTTPFailoverTransport) attempt(req *http.Request, body []byte, i int) (*http.Response, error) {
	ctx, cancel := failover.WithAttemptTimeout(req.Context(), t.attemptTimeout)
	resp, err := t.send(req.WithContext(ctx), body, i)
	if err != nil {
		cancel()
		return nil, err
	}
	resp.Body = &cancelOnClose{body: resp.Body, cancel: cancel}
	return resp, nil
}

func (t *borHTTPFailoverTransport) send(req *http.Request, body []byte, i int) (*http.Response, error) {
	clone := req.Clone(req.Context())
	ep := *t.endpoints[i].url
	clone.URL = &ep
	clone.Host = ep.Host
	if body != nil {
		clone.Body = io.NopCloser(bytes.NewReader(body))
		clone.ContentLength = int64(len(body))
	}

	return t.base.RoundTrip(clone)
}

// cancelOnClose cancels the per-attempt context when the response body is
// closed, mirroring how http.Client ties a request's timeout to its body.
type cancelOnClose struct {
	body   io.ReadCloser
	cancel context.CancelFunc
}

func (c *cancelOnClose) Read(p []byte) (int, error) { return c.body.Read(p) }

func (c *cancelOnClose) Close() error {
	c.cancel()
	return c.body.Close()
}

func drainAndClose(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
}

func drainRequestBody(req *http.Request) ([]byte, error) {
	if req.Body == nil {
		return nil, nil
	}
	defer req.Body.Close()

	return io.ReadAll(req.Body)
}

// probe validates endpoint i for the background prober: it must answer and
// report the expected Bor chain ID before it is treated as a healthy fallback.
// It tracks consecutive primary failures so a fallback can anchor identity when
// the primary is never reachable (see canAnchor).
func (t *borHTTPFailoverTransport) probe(i int) error {
	ctx, cancel := context.WithTimeout(context.Background(), t.health.ProbeTimeout())
	defer cancel()

	id, err := t.endpoints[i].probe.ChainID(ctx)
	if err != nil {
		if i == primaryEndpoint {
			t.primaryProbeFailures.Add(1)
		}
		return err
	}
	if i == primaryEndpoint {
		t.primaryProbeFailures.Store(0)
	}

	reclaiming := i == primaryEndpoint && !t.expectedByPrimary.Load()
	if err := t.checkChainID(i, id); err != nil {
		return err
	}
	if reclaiming {
		// The authoritative primary just (re)established the identity; if a stale
		// fallback was the active endpoint during the outage it may be serving
		// wrong-network data, so move traffic back to the primary at once instead
		// of waiting out the promotion threshold. Reclaim also demotes the other
		// endpoints so a fallback validated against the provisional identity can't
		// remain an in-call candidate until re-validated.
		t.health.Reclaim(primaryEndpoint)
	}

	return nil
}

// checkChainID compares endpoint i's chain ID against the expected one. The
// primary is authoritative: whenever it answers it (re)establishes the
// expectation, reclaiming a provisional one a fallback set while the primary was
// unreachable — so the real primary is never rejected. A fallback may
// provisionally anchor only after the primary has been unreachable for
// primaryAnchorFailureThreshold probes (see canAnchor); every endpoint that does
// not anchor must match the current expectation.
func (t *borHTTPFailoverTransport) checkChainID(i int, id *big.Int) error {
	if i == primaryEndpoint && !t.expectedByPrimary.Load() {
		t.expectedChainID.Store(id)
		t.expectedByPrimary.Store(true)
		return nil
	}

	expected := t.expectedChainID.Load()
	if expected == nil {
		if !t.canAnchor(i) {
			return fmt.Errorf("bor endpoint %d: expected chain id not yet known", i)
		}
		if t.expectedChainID.CompareAndSwap(nil, id) {
			return nil
		}
		expected = t.expectedChainID.Load() // lost the race; compare against the winner
	}

	if expected.Cmp(id) != 0 {
		return fmt.Errorf("bor endpoint %d chain id %s != expected %s", i, id, expected)
	}

	return nil
}

// canAnchor reports whether a fallback may provisionally establish the expected
// chain identity: only after the primary has failed primaryAnchorFailureThreshold
// consecutive probes, so failover still engages when the primary is never
// reachable at boot. The primary itself anchors via checkChainID's reclaim path.
func (t *borHTTPFailoverTransport) canAnchor(i int) bool {
	return i == primaryEndpoint || t.primaryProbeFailures.Load() >= primaryAnchorFailureThreshold
}

// newBorHTTPFailoverClient builds a Bor HTTP JSON-RPC client that fails over
// across the priority-ordered endpoints (index 0 = primary). The returned
// *rpc.Client / *ethclient.Client are the same concrete types as a plain
// rpc.Dial, so every existing Bor HTTP caller gets failover transparently.
func newBorHTTPFailoverClient(rawURLs []string, attemptTimeout time.Duration) (*rpc.Client, *ethclient.Client, error) {
	endpoints, err := dialHTTPEndpoints(rawURLs)
	if err != nil {
		return nil, nil, err
	}

	tr := &borHTTPFailoverTransport{
		endpoints:      endpoints,
		base:           http.DefaultTransport,
		attemptTimeout: attemptTimeout,
	}
	tr.captureExpectedChainID()
	tr.health = failover.New(len(endpoints), tr.probe, metrics.BorFailover("http"), Logger)

	rpcClient, err := rpc.DialOptions(context.Background(), endpoints[primaryEndpoint].url.String(),
		rpc.WithHTTPClient(&http.Client{Transport: tr}))
	if err != nil {
		return nil, nil, fmt.Errorf("constructing bor failover rpc client: %w", err)
	}

	tr.health.Start()
	borRPCFailoverTransport = tr

	return rpcClient, ethclient.NewClient(rpcClient), nil
}

// captureExpectedChainID best-effort records the primary's chain ID at startup
// so fallbacks can be validated before the first request. If the primary is
// unreachable, the expectation is set later by the primary's first probe.
func (t *borHTTPFailoverTransport) captureExpectedChainID() {
	if id, ok := fetchChainID(t.endpoints[primaryEndpoint].probe, t.attemptTimeout); ok {
		if t.expectedChainID.CompareAndSwap(nil, id) {
			t.expectedByPrimary.Store(true)
		}
	}
}

func dialHTTPEndpoints(rawURLs []string) ([]httpEndpoint, error) {
	out := make([]httpEndpoint, 0, len(rawURLs))
	for _, raw := range rawURLs {
		u, err := url.Parse(raw)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			Logger.Warn("bor failover: skipping invalid or non-HTTP bor RPC URL", "url", redactURL(raw))
			continue
		}

		rc, derr := rpc.Dial(u.String())
		if derr != nil {
			Logger.Warn("bor failover: skipping undialable bor RPC URL", "url", redactURL(raw), "error", derr)
			continue
		}

		out = append(out, httpEndpoint{url: u, probe: ethclient.NewClient(rc)})
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("no valid http(s) bor RPC endpoints among %d configured", len(rawURLs))
	}

	return out, nil
}

func fetchChainID(c chainIDProbe, timeout time.Duration) (*big.Int, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	id, err := c.ChainID(ctx)
	if err != nil || id == nil {
		return nil, false
	}

	return id, true
}

// GetBorChainCallTimeout returns the time budget a caller should allow for a
// single Bor call. With failover it is the per-endpoint timeout (BorRPCTimeout)
// times the budgeted endpoint count, so a cascade can reach a fallback within one
// caller-scoped context — go-ethereum's rpc.Client re-checks that context after
// the request returns, so a budget of only one attempt would race a successful
// fallback against the expired deadline. A single endpoint yields BorRPCTimeout
// unchanged.
//
// The endpoint count is the larger of the HTTP and gRPC lists because both the
// HTTP client (used by the broadcaster) and the gRPC client (used by side
// handlers) share this single budget, and it is capped at maxBudgetedEndpoints so
// a large count can't push one call past CometBFT's ~10s ABCI budget. The cap is
// on the time budget, not the attempt count: the deadline stops the cascade, so
// fast-failing endpoints beyond the cap may still be tried within the budget,
// while slow ones beyond it are reached on later calls.
func GetBorChainCallTimeout() time.Duration {
	n := len(parseURLs(conf.Custom.BorRPCUrl))
	if conf.Custom.BorGRPCFlag {
		if g := len(parseURLs(conf.Custom.BorGRPCUrl)); g > n {
			n = g
		}
	}
	if n < 1 {
		n = 1
	}
	if n > maxBudgetedEndpoints {
		n = maxBudgetedEndpoints
	}

	return conf.Custom.BorRPCTimeout * time.Duration(n)
}

// redactURL masks secrets in raw for safe logging: query-parameter values
// (providers commonly pass API keys as ?apikey=...) and userinfo passwords (via
// url.Redacted). A path-embedded token cannot be detected generically and is
// left as-is.
func redactURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "<unparseable>"
	}
	if q := u.Query(); len(q) > 0 {
		for k := range q {
			q.Set(k, "xxxxx")
		}
		u.RawQuery = q.Encode()
	}
	return u.Redacted()
}

// redactURLs redacts every URL in a comma-separated list for safe logging.
func redactURLs(csv string) string {
	parts := parseURLs(csv)
	for i, p := range parts {
		parts[i] = redactURL(p)
	}
	return strings.Join(parts, ",")
}

// CloseBorChainClients stops the Bor failover background probers, closes the
// HTTP probe clients, and closes the gRPC connections. It is the termination
// path for those goroutines; wire it into Heimdall's shutdown. Safe to call when
// neither failover is configured.
func CloseBorChainClients() {
	if borRPCFailoverTransport != nil {
		borRPCFailoverTransport.Close()
		borRPCFailoverTransport = nil
	}
	if borGRPCClient != nil {
		borGRPCClient.Close(Logger)
	}
}
