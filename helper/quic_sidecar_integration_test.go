package helper

import (
	"crypto/tls"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"strconv"
	"testing"
	"time"

	"github.com/quic-go/quic-go/http3"
	"github.com/stretchr/testify/require"
)

func TestQUICSidecarProxiesStatusOverHTTP3(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/status", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"catching_up":         false,
			"latest_block_height": "7",
		}))
	}))
	defer upstream.Close()

	cfg := CustomAppConfig{Custom: GetDefaultHeimdallConfig()}
	cfg.Custom.EnableQUICSidecar = true
	cfg.Custom.QUICSidecarUpstreamURL = upstream.URL
	cfg.Custom.QUICSidecarListenAddr = reserveLoopbackUDPPort(t)

	sidecar, err := buildQUICSidecarConfig(cfg)
	require.NoError(t, err)
	require.NotNil(t, sidecar)

	proxy := httputil.NewSingleHostReverseProxy(sidecar.upstream)
	server := &http3.Server{
		Addr:      sidecar.listenAddr,
		TLSConfig: sidecar.tlsConfig,
		Handler:   proxy,
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()
	t.Cleanup(func() {
		require.NoError(t, server.Close())
		select {
		case err := <-errCh:
			require.ErrorIs(t, err, http.ErrServerClosed)
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for QUIC sidecar shutdown")
		}
	})

	time.Sleep(150 * time.Millisecond)

	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http3.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion:         tls.VersionTLS13,
				InsecureSkipVerify: true,
			},
		},
	}

	resp, err := client.Get("https://" + sidecar.listenAddr + "/status")
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Equal(t, false, body["catching_up"])
	require.Equal(t, "7", body["latest_block_height"])
}

func reserveLoopbackUDPPort(t *testing.T) string {
	t.Helper()

	udpAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	require.NoError(t, err)

	udpConn, err := net.ListenUDP("udp", udpAddr)
	require.NoError(t, err)
	defer udpConn.Close()

	port := udpConn.LocalAddr().(*net.UDPAddr).Port
	return net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
}
