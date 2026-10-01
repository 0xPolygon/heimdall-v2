package helper

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cometbft/cometbft/privval"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

// TestUnmarshalExactToleratesLegacyLogsType guards against a regression
// UnmarshalExact would otherwise cause: any app.toml still setting the
// historical (always-a-no-op) "custom.logs_type" key would fail decoding and
// fatal at startup once the field it used to map to was removed from
// CustomConfig. LogsTypeDeprecated exists solely to keep that key decodable.
func TestUnmarshalExactToleratesLegacyLogsType(t *testing.T) {
	configDir := t.TempDir()
	appToml := `
[custom]
eth_rpc_url = "http://localhost:9545"
bor_rpc_url = "http://localhost:8545"
logs_type = "json"
`
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "app.toml"), []byte(appToml), 0o644))

	v := viper.New()
	v.SetConfigName("app")
	v.AddConfigPath(configDir)
	require.NoError(t, v.ReadInConfig())

	var decoded CustomAppConfig
	require.NoError(t, v.UnmarshalExact(&decoded), "a legacy logs_type key must not fail exact unmarshalling")
	require.Equal(t, "json", decoded.Custom.LogsTypeDeprecated)
}

// newLegacyLogsTypeHome writes a minimal, valid home directory (priv
// validator key, app.toml) so InitHeimdallConfigWith can run to completion
// against a stubbed RPC endpoint instead of log.Fatal-ing on a real dial.
// Shared by TestInitHeimdallConfigWithWarnsOnLegacyLogsType and
// TestInitHeimdallConfigWithLogFormatFallback.
func newLegacyLogsTypeHome(t *testing.T, rpcURL, logsType string) (home, logFile string) {
	t.Helper()

	home = t.TempDir()
	configDir := filepath.Join(home, "config")
	require.NoError(t, os.MkdirAll(configDir, 0o755))

	pv := privval.GenFilePV(
		filepath.Join(configDir, privValJsonFile),
		filepath.Join(configDir, "priv_validator_state.json"),
	)
	pv.Save()

	logFile = filepath.Join(home, "heimdalld.log")

	logsTypeLine := ""
	if logsType != "" {
		logsTypeLine = fmt.Sprintf("logs_type = %q\n", logsType)
	}
	appToml := fmt.Sprintf(`
[custom]
eth_rpc_url = %q
bor_rpc_url = %q
bor_grpc_flag = false
bor_grpc_url = ""
chain = "local"
producer_votes = ""
logs_writer_file = %q
%s`, rpcURL, rpcURL, logFile, logsTypeLine)
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "app.toml"), []byte(appToml), 0o644))
	return home, logFile
}

// TestInitHeimdallConfigWithWarnsOnLegacyLogsType drives the real init path so
// the operator gets a signal that a leftover custom.logs_type in app.toml no
// longer does anything, instead of the value being silently swallowed. The
// warning fires after Logger is reassigned to conf.Custom.LogsWriterFile, so
// that file (rather than a pre-set Logger, which the reassignment replaces)
// is what the test reads back.
func TestInitHeimdallConfigWithWarnsOnLegacyLogsType(t *testing.T) {
	origConf := conf
	origLogger := Logger
	t.Cleanup(func() {
		conf = origConf
		Logger = origLogger
	})

	rpcStub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x1"}`))
	}))
	defer rpcStub.Close()

	t.Run("legacy logs_type present warns", func(t *testing.T) {
		conf = CustomAppConfig{}
		home, logFile := newLegacyLogsTypeHome(t, rpcStub.URL, "json")

		InitHeimdallConfigWith(home, "")

		out, err := os.ReadFile(logFile)
		require.NoError(t, err)
		require.Contains(t, string(out), "custom.logs_type in app.toml is deprecated")
	})

	t.Run("no legacy logs_type stays silent", func(t *testing.T) {
		conf = CustomAppConfig{}
		home, logFile := newLegacyLogsTypeHome(t, rpcStub.URL, "")

		InitHeimdallConfigWith(home, "")

		out, err := os.ReadFile(logFile)
		require.NoError(t, err)
		require.NotContains(t, string(out), "custom.logs_type in app.toml is deprecated")
	})
}

// TestInitHeimdallConfigWithLogFormatFallback exercises the actual
// vulnerability a review found in an earlier version of this fallback (which
// mutated the --log_format flag directly): mutating the flag promotes it to
// viper's changed-pflag precedence tier, which outranks AutomaticEnv --
// silently overriding an operator's real HD_LOG_FORMAT (or equivalent)
// environment choice. viper.SetDefault only occupies the lowest tier above
// the compiled-in default, so every other explicit source -- flag, env var,
// config.toml -- must still win.
func TestInitHeimdallConfigWithLogFormatFallback(t *testing.T) {
	origConf := conf
	origLogger := Logger
	t.Cleanup(func() {
		conf = origConf
		Logger = origLogger
	})

	rpcStub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x1"}`))
	}))
	defer rpcStub.Close()

	t.Run("legacy json with nothing else set falls back to json", func(t *testing.T) {
		conf = CustomAppConfig{}
		viper.Reset()
		t.Cleanup(viper.Reset)
		home, _ := newLegacyLogsTypeHome(t, rpcStub.URL, "json")

		InitHeimdallConfigWith(home, "")

		require.Equal(t, "json", viper.GetString(flags.FlagLogFormat))
	})

	t.Run("explicit environment variable wins over legacy json", func(t *testing.T) {
		conf = CustomAppConfig{}
		viper.Reset()
		t.Cleanup(viper.Reset)
		viper.SetEnvPrefix("HD")
		viper.AutomaticEnv()
		t.Setenv("HD_LOG_FORMAT", "plain")
		home, _ := newLegacyLogsTypeHome(t, rpcStub.URL, "json")

		InitHeimdallConfigWith(home, "")

		require.Equal(t, "plain", viper.GetString(flags.FlagLogFormat),
			"an operator's explicit HD_LOG_FORMAT must win over the legacy logs_type fallback")
	})

	t.Run("explicit config.toml value wins over legacy json", func(t *testing.T) {
		conf = CustomAppConfig{}
		viper.Reset()
		t.Cleanup(viper.Reset)
		viper.SetConfigType("toml")
		require.NoError(t, viper.ReadConfig(strings.NewReader(`log_format = "plain"`)))
		home, _ := newLegacyLogsTypeHome(t, rpcStub.URL, "json")

		InitHeimdallConfigWith(home, "")

		require.Equal(t, "plain", viper.GetString(flags.FlagLogFormat))
	})

	t.Run("no legacy logs_type leaves the compiled default untouched", func(t *testing.T) {
		conf = CustomAppConfig{}
		viper.Reset()
		t.Cleanup(viper.Reset)
		home, _ := newLegacyLogsTypeHome(t, rpcStub.URL, "")

		InitHeimdallConfigWith(home, "")

		require.Equal(t, "", viper.GetString(flags.FlagLogFormat),
			"no flag is registered against this bare viper in the test, so the compiled default never applies here -- only confirming SetDefault wasn't called")
	})
}
