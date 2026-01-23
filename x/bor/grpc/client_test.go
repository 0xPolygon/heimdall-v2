package grpc

import (
	"testing"

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

		client, err := NewBorGRPCClient("://invalid", logger)
		require.Error(t, err)
		require.Nil(t, client)
	})

	t.Run("rejects https URL with empty host", func(t *testing.T) {
		t.Parallel()

		client, err := NewBorGRPCClient("https://", logger)
		require.Error(t, err)
		require.Contains(t, err.Error(), "empty host")
		require.Nil(t, client)
	})

	t.Run("rejects unix URL with empty path", func(t *testing.T) {
		t.Parallel()

		client, err := NewBorGRPCClient("unix://", logger)
		require.Error(t, err)
		require.Contains(t, err.Error(), "empty path")
		require.Nil(t, client)
	})

	t.Run("rejects unsupported URL scheme", func(t *testing.T) {
		t.Parallel()

		client, err := NewBorGRPCClient("ftp://example.com", logger)
		require.Error(t, err)
		require.Contains(t, err.Error(), "unsupported")
		require.Nil(t, client)
	})

	t.Run("rejects non-local address without scheme", func(t *testing.T) {
		t.Parallel()

		client, err := NewBorGRPCClient("example.com:8545", logger)
		require.Error(t, err)
		require.Contains(t, err.Error(), "insecure non-local")
		require.Nil(t, client)
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
