package helper

import (
	"errors"
	"math/big"
	"net/http"
	"testing"
	"time"

	"cosmossdk.io/log"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/x/bor/failover"
	borgrpc "github.com/0xPolygon/heimdall-v2/x/bor/grpc"
)

func TestParseURLs(t *testing.T) {
	require.Nil(t, parseURLs(""))
	require.Equal(t, []string{"http://a"}, parseURLs("http://a"))
	require.Equal(t, []string{"http://a", "http://b"}, parseURLs("http://a,http://b"))
	require.Equal(t, []string{"http://a", "http://b"}, parseURLs(" http://a , , http://b "))
	require.Empty(t, parseURLs(" , , "))
}

func TestRedactURLs(t *testing.T) {
	require.Equal(t, "http://user:xxxxx@host:8545", redactURL("http://user:pass@host:8545"))
	require.Equal(t, "https://host/rpc?apikey=xxxxx&token=xxxxx", redactURL("https://host/rpc?token=abc&apikey=secret"))
	require.Equal(t, "<unparseable>", redactURL("://nope"))
	require.Equal(t,
		"http://u:xxxxx@a:8545,https://b/rpc?key=xxxxx",
		redactURLs("http://u:p@a:8545, https://b/rpc?key=zzz"))
}

func TestGetBorChainCallTimeout(t *testing.T) {
	cfg := CustomAppConfig{Custom: GetDefaultHeimdallConfig()}
	cfg.Custom.BorRPCTimeout = time.Second

	// 3 HTTP endpoints, at the in-call cascade cap.
	cfg.Custom.BorGRPCFlag = false
	cfg.Custom.BorRPCUrl = "http://a,http://b,http://c"
	SetTestConfig(cfg)
	require.Equal(t, 3*time.Second, GetBorChainCallTimeout())

	// gRPC enabled with fewer gRPC endpoints than HTTP: sized by the larger count
	// so the HTTP client (broadcaster) is never under-budgeted.
	cfg.Custom.BorGRPCFlag = true
	cfg.Custom.BorGRPCUrl = "localhost:1,localhost:2"
	SetTestConfig(cfg)
	require.Equal(t, 3*time.Second, GetBorChainCallTimeout()) // max(3 HTTP, 2 gRPC)

	// Many endpoints are capped at the budgeted endpoint count.
	cfg.Custom.BorGRPCFlag = false
	cfg.Custom.BorRPCUrl = "http://a,http://b,http://c,http://d,http://e"
	SetTestConfig(cfg)
	require.Equal(t, maxBudgetedEndpoints*time.Second, GetBorChainCallTimeout())

	// Single endpoint unchanged.
	cfg.Custom.BorRPCUrl = "http://a"
	SetTestConfig(cfg)
	require.Equal(t, time.Second, GetBorChainCallTimeout())
}

func TestBuildBorGRPCClient(t *testing.T) {
	primary, single, err := buildBorGRPCClient([]string{"localhost:3131"}, "", time.Second)
	require.NoError(t, err)
	require.NotNil(t, primary)
	require.IsType(t, &borgrpc.BorGRPCClient{}, single)

	_, multi, err := buildBorGRPCClient([]string{"localhost:3131", "localhost:3132"}, "", time.Second)
	require.NoError(t, err)
	require.IsType(t, &borgrpc.MultiBorGRPCClient{}, multi)
	multi.Close(log.NewNopLogger())
}

func TestBuildBorGRPCClient_DialError(t *testing.T) {
	_, _, err := buildBorGRPCClient([]string{"1.2.3.4:3131"}, "", time.Second)
	require.Error(t, err)
}

func TestBuildBorGRPCClient_SkipsInvalidAmongValid(t *testing.T) {
	// "1.2.3.4:3131" is a non-localhost bare host:port (rejected by the dialer);
	// the valid localhost endpoint survives, so startup proceeds with one client.
	primary, client, err := buildBorGRPCClient([]string{"localhost:3131", "1.2.3.4:3131"}, "", time.Second)
	require.NoError(t, err)
	require.NotNil(t, primary)
	require.IsType(t, &borgrpc.BorGRPCClient{}, client)
}

func TestInitBorGRPCClient_DisabledByFlag(t *testing.T) {
	cfg := CustomAppConfig{Custom: GetDefaultHeimdallConfig()}
	cfg.Custom.BorGRPCFlag = false
	cfg.Custom.BorGRPCUrl = "localhost:3131"
	SetTestConfig(cfg)
	borGRPCClient = nil

	initBorGRPCClient()
	require.Nil(t, borGRPCClient) // gRPC disabled → no client built
}

func TestInitBorGRPCClient_Enabled(t *testing.T) {
	cfg := CustomAppConfig{Custom: GetDefaultHeimdallConfig()}
	cfg.Custom.BorGRPCFlag = true
	cfg.Custom.BorGRPCUrl = "localhost:3131"
	cfg.Custom.BorRPCTimeout = 100 * time.Millisecond
	SetTestConfig(cfg)
	borClient = nil
	borGRPCClient = nil

	initBorGRPCClient()
	require.NotNil(t, borGRPCClient) // gRPC enabled → client built (lazy dial)
}

func TestSucceeded(t *testing.T) {
	require.True(t, succeeded(&http.Response{StatusCode: http.StatusOK}, nil))
	require.True(t, succeeded(&http.Response{StatusCode: http.StatusNotFound}, nil))             // 4xx is returned as-is
	require.False(t, succeeded(&http.Response{StatusCode: http.StatusInternalServerError}, nil)) // 500 → cascade
	require.False(t, succeeded(&http.Response{StatusCode: http.StatusBadGateway}, nil))
	require.False(t, succeeded(nil, errors.New("boom")))
}

func TestCheckChainID(t *testing.T) {
	tr := &borHTTPFailoverTransport{}
	require.Error(t, tr.checkChainID(1, big.NewInt(5)))   // expected unknown + fallback → rejected
	require.NoError(t, tr.checkChainID(0, big.NewInt(5))) // primary establishes the expectation
	require.Equal(t, int64(5), tr.expectedChainID.Load().Int64())
	require.NoError(t, tr.checkChainID(1, big.NewInt(5))) // matches
	require.Error(t, tr.checkChainID(2, big.NewInt(9)))   // mismatch
}

func TestCanAnchor(t *testing.T) {
	tr := &borHTTPFailoverTransport{}
	require.True(t, tr.canAnchor(primaryEndpoint))               // primary always anchors
	require.False(t, tr.canAnchor(1))                            // fallback blocked while primary reachable
	tr.primaryProbeFailures.Store(primaryAnchorFailureThreshold) // primary unreachable through startup
	require.True(t, tr.canAnchor(1))                             // fallback may now anchor
}

func TestCheckChainID_FallbackAnchorsWhenPrimaryUnreachable(t *testing.T) {
	tr := &borHTTPFailoverTransport{}
	tr.primaryProbeFailures.Store(primaryAnchorFailureThreshold)

	require.NoError(t, tr.checkChainID(1, big.NewInt(7))) // fallback provisionally establishes the expectation
	require.Equal(t, int64(7), tr.expectedChainID.Load().Int64())
	require.NoError(t, tr.checkChainID(2, big.NewInt(7))) // another fallback on the same chain matches
	require.Error(t, tr.checkChainID(2, big.NewInt(9)))   // a mismatched endpoint is still rejected
}

func TestCheckChainID_PrimaryReclaimsProvisionalAnchor(t *testing.T) {
	tr := &borHTTPFailoverTransport{}
	tr.primaryProbeFailures.Store(primaryAnchorFailureThreshold)

	require.NoError(t, tr.checkChainID(1, big.NewInt(7)))  // fallback provisionally anchors 7
	require.NoError(t, tr.checkChainID(0, big.NewInt(11))) // primary reclaims with its own id, never rejected
	require.Equal(t, int64(11), tr.expectedChainID.Load().Int64())
	require.Error(t, tr.checkChainID(1, big.NewInt(7))) // the stale provisional fallback now mismatches the primary
}

func TestCloseBorChainClients(t *testing.T) {
	// safe to call when neither Bor failover is configured
	borRPCFailoverHealth = nil
	borGRPCClient = nil
	require.NotPanics(t, CloseBorChainClients)

	// stops a running HTTP failover prober (CloseBorChainClients must return,
	// i.e. join the prober goroutine, rather than hang)
	h := failover.New(2, func(int) error { return nil }, failover.Metrics{}, log.NewNopLogger())
	h.SetTuning(5*time.Millisecond, 1, 0, 50*time.Millisecond)
	h.Start()
	borRPCFailoverHealth = h
	CloseBorChainClients()
	borRPCFailoverHealth = nil
}
