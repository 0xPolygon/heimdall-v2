package heimdalld

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/cosmos/cosmos-sdk/server"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/helper"
)

func TestPeerServingRootStartup(t *testing.T) {
	const childEnv = "HEIMDALL_SERVING_STARTUP_TEST"
	mode := os.Getenv(childEnv)
	if mode == "" {
		// Startup installs process-global configuration and metrics.
		for _, mode := range []string{"info", "error", "module", "invalid"} {
			t.Run(mode, func(t *testing.T) {
				cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestPeerServingRootStartup$", "-test.timeout=15s")
				cmd.Env = append(os.Environ(), childEnv+"="+mode)
				output, err := cmd.CombinedOutput()
				require.NoError(t, err, "%s", output)
				if mode == "error" || mode == "module" {
					require.NotContains(t, string(output), "Native peer serving enabled")
				}
			})
		}
		return
	}

	home := t.TempDir()
	t.Chdir(home)
	t.Setenv(homeEnvVar, home)
	helper.SetTestConfig(helper.CustomAppConfig{Custom: helper.GetDefaultHeimdallConfig()})
	root := NewRootCmd()
	start, _, err := root.Find([]string{"start"})
	require.NoError(t, err)
	ran := false
	start.RunE = func(*cobra.Command, []string) error { ran = true; return nil }
	level := "info"
	if mode == "error" {
		level = "error"
	} else if mode == "module" {
		level = "server:error,*:info"
	}
	args := []string{"start", "--home", home, "--log_level", level, "--log_format", "json"}
	if mode == "invalid" {
		args = append(args, "--peer-serving-memory=1")
	}
	output := &bytes.Buffer{}
	root.SetOut(output)
	root.SetErr(output)
	root.SetArgs(args)
	err = svrcmd.Execute(root, "HD", home)
	serverCtx := server.GetServerContextFromCmd(root)
	if mode == "invalid" {
		require.ErrorContains(t, err, "peer-serving-memory must be at least")
		require.False(t, ran)
		require.Nil(t, serverCtx.Config.P2P.ServingPolicy)
		return
	}
	require.NoError(t, err)
	require.True(t, ran)
	require.NotNil(t, serverCtx.Config.P2P.ServingPolicy)
	if mode != "info" {
		require.NotContains(t, output.String(), "Native peer serving enabled")
		return
	}
	for _, line := range strings.Split(output.String(), "\n") {
		if strings.Contains(line, "Native peer serving enabled") {
			var record map[string]any
			require.NoError(t, json.Unmarshal([]byte(line), &record))
			require.Equal(t, "server", record["module"])
			require.Equal(t, "observe", record["reputation_mode"])
			return
		}
	}
	t.Fatal("missing configured startup log")
}
