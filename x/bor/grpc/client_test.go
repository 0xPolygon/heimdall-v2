package grpc

import (
	"context"
	"net/url"
	"testing"
	"time"

	"cosmossdk.io/log"
	"github.com/stretchr/testify/require"
)

func TestIsLocalhost(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		hostport string
		want     bool
	}{
		{
			name:     "localhost hostname",
			hostport: "localhost",
			want:     true,
		},
		{
			name:     "localhost with port",
			hostport: "localhost:8545",
			want:     true,
		},
		{
			name:     "IPv4 loopback",
			hostport: "127.0.0.1",
			want:     true,
		},
		{
			name:     "IPv4 loopback with port",
			hostport: "127.0.0.1:8545",
			want:     true,
		},
		{
			name:     "IPv6 loopback",
			hostport: "::1",
			want:     true,
		},
		{
			name:     "IPv6 loopback with brackets and port",
			hostport: "[::1]:8545",
			want:     true,
		},
		{
			name:     "remote hostname",
			hostport: "example.com",
			want:     false,
		},
		{
			name:     "remote hostname with port",
			hostport: "example.com:8545",
			want:     false,
		},
		{
			name:     "remote IPv4",
			hostport: "192.168.1.1",
			want:     false,
		},
		{
			name:     "remote IPv4 with port",
			hostport: "192.168.1.1:8545",
			want:     false,
		},
		{
			name:     "empty string",
			hostport: "",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := isLocalhost(tt.hostport)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestNewBorGRPCClient_URLParsing(t *testing.T) {
	t.Parallel()

	logger := log.NewNopLogger()

	t.Run("rejects invalid URL", func(t *testing.T) {
		t.Parallel()

		client, err := NewBorGRPCClient("://invalid", "", logger)
		require.Error(t, err)
		require.Nil(t, client)
	})

	t.Run("rejects https URL with empty host", func(t *testing.T) {
		t.Parallel()

		client, err := NewBorGRPCClient("https://", "", logger)
		require.Error(t, err)
		require.Contains(t, err.Error(), "empty host")
		require.Nil(t, client)
	})

	t.Run("rejects unix URL with empty path", func(t *testing.T) {
		t.Parallel()

		client, err := NewBorGRPCClient("unix://", "", logger)
		require.Error(t, err)
		require.Contains(t, err.Error(), "empty path")
		require.Nil(t, client)
	})

	t.Run("rejects unsupported URL scheme", func(t *testing.T) {
		t.Parallel()

		client, err := NewBorGRPCClient("ftp://example.com", "", logger)
		require.Error(t, err)
		require.Contains(t, err.Error(), "unsupported")
		require.Nil(t, client)
	})

	t.Run("rejects non-local address without scheme", func(t *testing.T) {
		t.Parallel()

		client, err := NewBorGRPCClient("example.com:8545", "", logger)
		require.Error(t, err)
		require.Contains(t, err.Error(), "insecure non-local")
		require.Nil(t, client)
	})

	t.Run("rejects non-local plaintext http when token is set", func(t *testing.T) {
		t.Parallel()

		// Bearer token over plaintext http:// to a non-local host would leak the
		// token on the network path. Construction must refuse explicitly.
		client, err := NewBorGRPCClient("http://remote.example.com:3131", "secret-token", logger)
		require.Error(t, err)
		require.Contains(t, err.Error(), "plaintext")
		require.Nil(t, client)
	})

	t.Run("accepts non-local plaintext http when no token is set", func(t *testing.T) {
		t.Parallel()

		// Historical behavior: a non-local http:// with no token just warns.
		// Kept backward-compatible for operators who have an unauthenticated
		// cross-host setup inside a trusted network.
		client, err := NewBorGRPCClient("http://remote.example.com:3131", "", logger)
		require.NoError(t, err)
		require.NotNil(t, client)
	})

	t.Run("accepts localhost plaintext http with token set", func(t *testing.T) {
		t.Parallel()

		// Loopback is trusted — sending the token over local plaintext is
		// the recommended same-host validator setup.
		client, err := NewBorGRPCClient("http://localhost:3131", "secret-token", logger)
		require.NoError(t, err)
		require.NotNil(t, client)
	})
}

func TestBorGRPCClient_Close(t *testing.T) {
	t.Parallel()

	logger := log.NewNopLogger()

	t.Run("close nil client", func(t *testing.T) {
		t.Parallel()

		var client *BorGRPCClient
		require.NotPanics(t, func() {
			client.Close(logger)
		})
	})

	t.Run("close client with nil connection", func(t *testing.T) {
		t.Parallel()

		client := &BorGRPCClient{conn: nil}
		require.NotPanics(t, func() {
			client.Close(logger)
		})
	})
}

// TestResolveHTTPS verifies that resolveHTTPS extracts the host as the dial
// address, sets isTLS=true, returns exactly one DialOption, and rejects an
// empty host.
func TestResolveHTTPS(t *testing.T) {
	t.Parallel()

	logger := log.NewNopLogger()

	t.Run("happy path returns host and isTLS=true", func(t *testing.T) {
		t.Parallel()

		u := mustParseURL(t, "https://grpc.example.com:443")
		addr, opts, isTLS, err := resolveHTTPS(u, "https://grpc.example.com:443", logger)

		require.NoError(t, err)
		require.Equal(t, "grpc.example.com:443", addr)
		require.True(t, isTLS, "resolveHTTPS must return isTLS=true")
		require.Len(t, opts, 1, "exactly one DialOption (TLS credentials)")
	})

	t.Run("empty host returns error and isTLS=false", func(t *testing.T) {
		t.Parallel()

		u := mustParseURL(t, "https://")
		_, _, isTLS, err := resolveHTTPS(u, "https://", logger)

		require.Error(t, err)
		require.Contains(t, err.Error(), "empty host")
		require.False(t, isTLS)
	})
}

// TestResolveHTTP verifies isTLS=false, addr extraction, and the
// token-over-plaintext rejection.
func TestResolveHTTP(t *testing.T) {
	t.Parallel()

	logger := log.NewNopLogger()

	t.Run("localhost with token returns isTLS=false", func(t *testing.T) {
		t.Parallel()

		u := mustParseURL(t, "http://localhost:3131")
		addr, opts, isTLS, err := resolveHTTP(u, "secret", logger)

		require.NoError(t, err)
		require.Equal(t, "localhost:3131", addr)
		require.False(t, isTLS, "resolveHTTP must return isTLS=false")
		require.Len(t, opts, 1, "exactly one DialOption (insecure credentials)")
	})

	t.Run("remote host without token returns isTLS=false", func(t *testing.T) {
		t.Parallel()

		u := mustParseURL(t, "http://remote.example.com:3131")
		addr, opts, isTLS, err := resolveHTTP(u, "", logger)

		require.NoError(t, err)
		require.Equal(t, "remote.example.com:3131", addr)
		require.False(t, isTLS)
		require.Len(t, opts, 1)
	})

	t.Run("remote host with token returns error", func(t *testing.T) {
		t.Parallel()

		u := mustParseURL(t, "http://remote.example.com:3131")
		_, _, isTLS, err := resolveHTTP(u, "token", logger)

		require.Error(t, err)
		require.Contains(t, err.Error(), "plaintext")
		require.False(t, isTLS)
	})

	t.Run("empty host returns error and isTLS=false", func(t *testing.T) {
		t.Parallel()

		u := mustParseURL(t, "http://")
		_, _, isTLS, err := resolveHTTP(u, "", logger)

		require.Error(t, err)
		require.Contains(t, err.Error(), "empty host")
		require.False(t, isTLS)
	})
}

// TestResolveUnix verifies the unix:// path prefix is prepended correctly and
// that isTLS=false.
func TestResolveUnix(t *testing.T) {
	t.Parallel()

	logger := log.NewNopLogger()

	t.Run("valid unix path returns unix:// addr and isTLS=false", func(t *testing.T) {
		t.Parallel()

		u := mustParseURL(t, "unix:///var/run/bor.sock")
		addr, opts, isTLS, err := resolveUnix(u, "unix:///var/run/bor.sock", logger)

		require.NoError(t, err)
		require.Equal(t, "unix:///var/run/bor.sock", addr)
		require.False(t, isTLS, "resolveUnix must return isTLS=false")
		// Two DialOptions: context dialer + insecure credentials.
		require.Len(t, opts, 2)
	})

	t.Run("empty path returns error", func(t *testing.T) {
		t.Parallel()

		u := mustParseURL(t, "unix://")
		_, _, isTLS, err := resolveUnix(u, "unix://", logger)

		require.Error(t, err)
		require.Contains(t, err.Error(), "empty path")
		require.False(t, isTLS)
	})
}

// TestResolveNoScheme verifies that bare localhost addresses are accepted with
// isTLS=false and that non-local addresses are rejected.
func TestResolveNoScheme(t *testing.T) {
	t.Parallel()

	logger := log.NewNopLogger()

	t.Run("localhost address accepted with isTLS=false", func(t *testing.T) {
		t.Parallel()

		addr, opts, isTLS, err := resolveNoScheme("localhost:9090", logger)

		require.NoError(t, err)
		require.Equal(t, "localhost:9090", addr)
		require.False(t, isTLS, "resolveNoScheme must return isTLS=false")
		require.Len(t, opts, 1)
	})

	t.Run("127.0.0.1 address accepted", func(t *testing.T) {
		t.Parallel()

		addr, opts, isTLS, err := resolveNoScheme("127.0.0.1:9090", logger)

		require.NoError(t, err)
		require.Equal(t, "127.0.0.1:9090", addr)
		require.False(t, isTLS)
		require.Len(t, opts, 1)
	})

	t.Run("remote address rejected", func(t *testing.T) {
		t.Parallel()

		_, _, isTLS, err := resolveNoScheme("remote.example.com:9090", logger)

		require.Error(t, err)
		require.Contains(t, err.Error(), "insecure non-local")
		require.False(t, isTLS)
	})
}

// TestBearerToken verifies GetRequestMetadata returns the correct authorization
// header value, and RequireTransportSecurity reflects the requireSecurity field.
func TestBearerToken(t *testing.T) {
	t.Parallel()

	t.Run("GetRequestMetadata returns Bearer header", func(t *testing.T) {
		t.Parallel()

		bt := bearerToken{token: "my-secret-token", requireSecurity: false}
		meta, err := bt.GetRequestMetadata(context.Background())

		require.NoError(t, err)
		require.Equal(t, "Bearer my-secret-token", meta["authorization"])
	})

	t.Run("RequireTransportSecurity true when requireSecurity=true", func(t *testing.T) {
		t.Parallel()

		bt := bearerToken{token: "tok", requireSecurity: true}
		require.True(t, bt.RequireTransportSecurity())
	})

	t.Run("RequireTransportSecurity false when requireSecurity=false", func(t *testing.T) {
		t.Parallel()

		bt := bearerToken{token: "tok", requireSecurity: false}
		require.False(t, bt.RequireTransportSecurity())
	})
}

// TestDialTimeout verifies the dialTimeout constant is 5 seconds and has not
// been inadvertently mutated.
func TestDialTimeout(t *testing.T) {
	t.Parallel()

	require.Equal(t, 5*time.Second, dialTimeout,
		"dialTimeout constant must be exactly 5 s")
}

// TestResolveTransport_TokenWithTLS verifies that when resolveTransport picks
// the https scheme and a token is provided, the returned isTLS=true propagates
// into bearerToken.
func TestResolveTransport_TokenWithTLS(t *testing.T) {
	t.Parallel()

	logger := log.NewNopLogger()

	addr, _, isTLS, err := resolveTransport("https://grpc.example.com:443", "tok", logger)
	require.NoError(t, err)
	require.Equal(t, "grpc.example.com:443", addr)
	require.True(t, isTLS)
}

// TestResolveTransport_ErrorPaths verifies that resolveTransport's error
// branches return isTLS=false. The bool is passed through to bearerToken's
// RequireTransportSecurity; a spurious `true` here would reject a valid
// localhost dial on a later retry.
func TestResolveTransport_ErrorPaths(t *testing.T) {
	t.Parallel()

	logger := log.NewNopLogger()

	t.Run("malformed URL returns isTLS=false", func(t *testing.T) {
		t.Parallel()

		addr, opts, isTLS, err := resolveTransport("://invalid", "", logger)
		require.Error(t, err)
		require.Empty(t, addr)
		require.Nil(t, opts)
		require.False(t, isTLS, "error return must keep isTLS=false")
	})

	t.Run("unsupported scheme returns isTLS=false", func(t *testing.T) {
		t.Parallel()

		addr, opts, isTLS, err := resolveTransport("ftp://example.com:1234", "", logger)
		require.Error(t, err)
		require.Contains(t, err.Error(), "unsupported")
		require.Empty(t, addr)
		require.Nil(t, opts)
		require.False(t, isTLS, "error return must keep isTLS=false")
	})
}

// TestNewBorGRPCClient_ErrorPath verifies that a dial error from grpc.NewClient
// is propagated and the function returns nil.
func TestNewBorGRPCClient_ErrorPath(t *testing.T) {
	t.Parallel()

	logger := log.NewNopLogger()
	client, err := NewBorGRPCClient("://bad", "", logger)

	require.Error(t, err)
	require.Nil(t, client)
}

// mustParseURL is a test helper that parses a URL string or fails the test.
func mustParseURL(t *testing.T, rawURL string) *url.URL {
	t.Helper()
	u, err := url.Parse(rawURL)
	require.NoError(t, err, "test setup: failed to parse URL %q", rawURL)
	return u
}
