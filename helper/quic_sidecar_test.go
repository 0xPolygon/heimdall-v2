package helper

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildQUICSidecarConfigDisabled(t *testing.T) {
	t.Parallel()

	cfg := CustomAppConfig{Custom: GetDefaultHeimdallConfig()}
	sidecar, err := buildQUICSidecarConfig(cfg)
	require.NoError(t, err)
	require.Nil(t, sidecar)
}

func TestBuildQUICSidecarConfigDerivesLoopbackDefaults(t *testing.T) {
	t.Parallel()

	cfg := CustomAppConfig{Custom: GetDefaultHeimdallConfig()}
	cfg.Custom.EnableQUICSidecar = true
	cfg.API.Address = "tcp://127.0.0.1:1317"

	sidecar, err := buildQUICSidecarConfig(cfg)
	require.NoError(t, err)
	require.NotNil(t, sidecar)
	require.Equal(t, "127.0.0.1:1317", sidecar.listenAddr)
	require.Equal(t, "http://127.0.0.1:1317", sidecar.upstream.String())
	require.NotNil(t, sidecar.tlsConfig)
	require.Len(t, sidecar.tlsConfig.Certificates, 1)
}

func TestBuildQUICSidecarConfigRejectsNonLoopbackAutoTLS(t *testing.T) {
	t.Parallel()

	cfg := CustomAppConfig{Custom: GetDefaultHeimdallConfig()}
	cfg.Custom.EnableQUICSidecar = true
	cfg.API.Address = "tcp://0.0.0.0:1317"

	sidecar, err := buildQUICSidecarConfig(cfg)
	require.Nil(t, sidecar)
	require.ErrorContains(t, err, "requires cert and key files")
}
