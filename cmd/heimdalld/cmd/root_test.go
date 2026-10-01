package heimdalld

import (
	"testing"

	cmtcli "github.com/cometbft/cometbft/libs/cli"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestApplyHomeEnvOverride(t *testing.T) {
	newHomeFlagCmd := func(flagDefault string) *cobra.Command {
		cmd := &cobra.Command{Use: "test"}
		cmd.Flags().String(flags.FlagHome, flagDefault, "directory for config and data")
		return cmd
	}

	tests := []struct {
		name       string
		cmd        func() *cobra.Command
		envValue   string
		envSet     bool
		wantHome   string
		wantErrMsg string
	}{
		{
			name:     "HD_HOME overrides an unset --home flag",
			cmd:      func() *cobra.Command { return newHomeFlagCmd("/var/lib/heimdall") },
			envValue: "/custom/heimdall-home",
			envSet:   true,
			wantHome: "/custom/heimdall-home",
		},
		{
			name: "explicit --home flag wins over HD_HOME",
			cmd: func() *cobra.Command {
				cmd := newHomeFlagCmd("/var/lib/heimdall")
				require.NoError(t, cmd.Flags().Set(flags.FlagHome, "/explicit/home"))
				return cmd
			},
			envValue: "/custom/heimdall-home",
			envSet:   true,
			wantHome: "/explicit/home",
		},
		{
			name:     "no HD_HOME leaves the flag default untouched",
			cmd:      func() *cobra.Command { return newHomeFlagCmd("/var/lib/heimdall") },
			envSet:   false,
			wantHome: "/var/lib/heimdall",
		},
		{
			name:     "empty HD_HOME leaves the flag default untouched",
			cmd:      func() *cobra.Command { return newHomeFlagCmd("/var/lib/heimdall") },
			envValue: "",
			envSet:   true,
			wantHome: "/var/lib/heimdall",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envSet {
				t.Setenv(homeEnvVar, tt.envValue)
			}

			cmd := tt.cmd()
			err := applyHomeEnvOverride(cmd)
			require.NoError(t, err)

			home, err := cmd.Flags().GetString(flags.FlagHome)
			require.NoError(t, err)
			require.Equal(t, tt.wantHome, home)
		})
	}
}

// TestHDHomeAlreadyResolvesViaAutomaticEnv pins down a guarantee
// applyHomeEnvOverride's doc comment depends on: svrcmd.Execute passes "HD"
// as the env prefix into cmtcli.PrepareBaseCmd, which arms
// viper.SetEnvPrefix("HD") + viper.AutomaticEnv() on the global viper
// singleton via a cobra.OnInitialize hook that fires in Cobra's own
// preRun(), strictly before any PersistentPreRunE -- including cometbft's
// own bindFlagsLoadViper, prepended ahead of ours. So by the time
// bindFlagsLoadViper computes `homeDir := viper.GetString(HomeFlag)`, HD_HOME
// already resolves through viper's AutomaticEnv precedence tier, with zero
// help from applyHomeEnvOverride (which hasn't even run yet at that point).
// This is exercised here with the real cmtcli.PrepareBaseCmd wiring, not a
// hand-rolled approximation of it, so a future change to that wiring would
// break this test rather than only being caught in production. If this ever
// starts failing, do not "fix" it by reintroducing a viper.Set mirror in
// applyHomeEnvOverride -- find out why AutomaticEnv stopped applying first.
func TestHDHomeAlreadyResolvesViaAutomaticEnv(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	const overrideHome = "/custom/heimdall-home"
	t.Setenv(homeEnvVar, overrideHome)

	var observedInPersistentPreRunE string
	cmd := &cobra.Command{
		Use: "test",
		PersistentPreRunE: func(*cobra.Command, []string) error {
			observedInPersistentPreRunE = viper.GetString(flags.FlagHome)
			return nil
		},
		RunE: func(*cobra.Command, []string) error { return nil },
	}
	cmtcli.PrepareBaseCmd(cmd, "HD", "/var/lib/heimdall")

	cmd.SetArgs(nil)
	require.NoError(t, cmd.Execute())

	require.Equal(t, overrideHome, observedInPersistentPreRunE,
		"HD_HOME should already resolve via AutomaticEnv before applyHomeEnvOverride ever runs")
}

func TestApplyHomeEnvOverride_PropagatesSetError(t *testing.T) {
	// --home is registered as an Int flag here purely to force
	// cmd.Flags().Set to fail, so we can assert that
	// applyHomeEnvOverride propagates the error instead of swallowing it.
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().Int(flags.FlagHome, 0, "not a real home flag")

	t.Setenv(homeEnvVar, "not-an-int")

	require.Error(t, applyHomeEnvOverride(cmd))
}

func TestNewRootCmd_PersistentPreRunE_FailsClosedOnHomeEnvOverrideError(t *testing.T) {
	rootCmd := NewRootCmd()

	// NewRootCmd doesn't register --home itself (svrcmd.Execute does, via
	// cmtcli.PrepareBaseCmd, before PersistentPreRunE ever runs in
	// production), so applyHomeEnvOverride's cmd.Flags().Set call fails here
	// with "no such flag" as soon as HD_HOME is set. That's enough to prove
	// PersistentPreRunE returns the error instead of continuing past it.
	t.Setenv(homeEnvVar, "/tmp/does-not-matter")

	err := rootCmd.PersistentPreRunE(rootCmd, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), flags.FlagHome)
}
