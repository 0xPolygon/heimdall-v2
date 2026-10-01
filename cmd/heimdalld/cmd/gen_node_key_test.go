package heimdalld

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	cmtcfg "github.com/cometbft/cometbft/config"
	"github.com/cometbft/cometbft/p2p"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// genNodeKeyCmdWithConfig returns genNodeKeyCmd() with a server context
// carrying cfg attached to its Context, standing in for the context
// cosmos-sdk's own PersistentPreRunE chain populates from --home/HD_HOME
// and the operator's config.toml when genNodeKeyCmd is wired into the real
// command tree.
func genNodeKeyCmdWithConfig(t *testing.T, cfg *cmtcfg.Config) *cobra.Command {
	t.Helper()

	cmd := genNodeKeyCmd()
	cmd.SetContext(context.WithValue(context.Background(), server.ServerContextKey, &server.Context{Config: cfg}))
	return cmd
}

func TestGenNodeKeyCmd(t *testing.T) {
	t.Run("writes node_key.json under the resolved home, not the working directory", func(t *testing.T) {
		home := t.TempDir()
		wd := t.TempDir()

		require.NoError(t, os.MkdirAll(filepath.Join(home, "config"), 0o755))

		restoreWd := chdir(t, wd)
		defer restoreWd()

		cfg := cmtcfg.DefaultConfig()
		cfg.SetRoot(home)

		cmd := genNodeKeyCmdWithConfig(t, cfg)
		require.NoError(t, cmd.RunE(cmd, nil))

		homeNodeKey := filepath.Join(home, "config", "node_key.json")
		require.FileExists(t, homeNodeKey)

		wdNodeKey := filepath.Join(wd, "config", "node_key.json")
		require.NoFileExists(t, wdNodeKey)

		nodeKey, err := p2p.LoadNodeKey(homeNodeKey)
		require.NoError(t, err)
		require.NotEmpty(t, nodeKey.ID())
	})

	t.Run("honors a custom node_key_file path from config.toml", func(t *testing.T) {
		// start/show-node-id resolve the node key through the same loaded
		// config, including any operator override of node_key_file away
		// from the "config/node_key.json" default; gen-node-key must land
		// in the same place or the two commands disagree on the node's key.
		home := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(home, "custom"), 0o755))

		cfg := cmtcfg.DefaultConfig()
		cfg.SetRoot(home)
		cfg.NodeKey = "custom/identity.json"

		cmd := genNodeKeyCmdWithConfig(t, cfg)
		require.NoError(t, cmd.RunE(cmd, nil))

		require.FileExists(t, filepath.Join(home, "custom", "identity.json"))
		require.NoFileExists(t, filepath.Join(home, "config", "node_key.json"))
	})

	t.Run("errors instead of overwriting an existing node key", func(t *testing.T) {
		home := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(home, "config"), 0o755))

		cfg := cmtcfg.DefaultConfig()
		cfg.SetRoot(home)

		cmd := genNodeKeyCmdWithConfig(t, cfg)
		require.NoError(t, cmd.RunE(cmd, nil))
		require.Error(t, cmd.RunE(cmd, nil))
	})

	t.Run("propagates the underlying node key generation error", func(t *testing.T) {
		// home has no "config" subdirectory, so writing node_key.json fails.
		home := t.TempDir()

		cfg := cmtcfg.DefaultConfig()
		cfg.SetRoot(home)

		cmd := genNodeKeyCmdWithConfig(t, cfg)
		require.Error(t, cmd.RunE(cmd, nil))
	})
}

// chdir switches the process working directory to dir and returns a func
// that restores the original one; tests run sequentially in this package so
// mutating the shared process cwd is safe as long as it's always restored.
func chdir(t *testing.T, dir string) func() {
	t.Helper()

	original, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))

	return func() {
		require.NoError(t, os.Chdir(original))
	}
}
