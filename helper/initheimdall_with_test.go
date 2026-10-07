package helper

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/cometbft/cometbft/privval"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

// TestInitHeimdallConfigWithSetsIthacaHeightPerChain drives the
// real init path through each chain branch so the Ithaca assignment lines in
// InitHeimdallConfigWith stay covered. A tiny JSON-RPC stub and a real
// priv_validator_key.json keep the init code on the happy path without touching
// production services.
func TestInitHeimdallConfigWithSetsIthacaHeightPerChain(t *testing.T) {
	restoreInitGlobals(t)
	rpcURL := newRPCStub(t)

	cases := []struct {
		name              string
		chain             string
		wantRioHeight     int64
		wantInitHeight    int64
		wantIthacaHeight  int64
		wantCompactHeight int64
		wantPhuket        int64
		wantZurich        int64
		wantFeeGate       int64
	}{
		{name: "mainnet", chain: MainChain, wantRioHeight: 77414656, wantInitHeight: 24404501, wantIthacaHeight: 50185000, wantPhuket: 44070000, wantZurich: 47880000, wantFeeGate: 46361000},
		{name: "mumbai", chain: MumbaiChain, wantRioHeight: 48473856},
		{name: "amoy", chain: AmoyChain, wantRioHeight: 26272256, wantInitHeight: 8788501, wantIthacaHeight: 40776000, wantPhuket: 32276400, wantZurich: 37750000, wantFeeGate: 35914000},
		// Local devnets run every live fork from genesis and switch to compact vote extensions mid-run.
		{name: "default", chain: "local", wantRioHeight: 128, wantIthacaHeight: 1, wantCompactHeight: 500, wantPhuket: 1, wantZurich: 1, wantFeeGate: 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			conf = CustomAppConfig{}

			home := mkTestHome(t, rpcURL, tc.chain)
			InitHeimdallConfigWith(home, "")

			require.Equal(t, tc.wantIthacaHeight, GetIthacaHeight())
			require.Equal(t, tc.wantCompactHeight, GetCompactVoteExtHeight())
			require.Equal(t, tc.wantPhuket, GetPhuketHardforkHeight())
			require.Equal(t, tc.wantZurich, GetZurichHardforkHeight())
			require.Equal(t, tc.wantFeeGate, GetFeeWithdrawValidatorGateHeight())
			require.Equal(t, tc.wantRioHeight, GetRioHeight())
			require.Equal(t, tc.wantInitHeight, GetInitialHeight())
		})
	}
}

// The per-module filter only helps if the init path installs it on the
// package logger; a plain level must leave the logger unwrapped.
func TestInitHeimdallConfigWithInstallsModuleFilter(t *testing.T) {
	restoreInitGlobals(t)
	rpcURL := newRPCStub(t)
	origLevel := viper.GetString(flags.FlagLogLevel)
	t.Cleanup(func() { viper.Set(flags.FlagLogLevel, origLevel) })

	cases := []struct {
		level   string
		wrapped bool
	}{
		{"p2p:error,*:info", true},
		{"info", false},
	}
	for _, tc := range cases {
		t.Run(tc.level, func(t *testing.T) {
			conf = CustomAppConfig{}
			viper.Set(flags.FlagLogLevel, tc.level)

			InitHeimdallConfigWith(mkTestHome(t, rpcURL, "local"), "")

			_, wrapped := Logger.(moduleFilterLogger)
			require.Equal(t, tc.wrapped, wrapped)
		})
	}
}

// restoreInitGlobals snapshots the package state InitHeimdallConfigWith
// mutates, including the package Logger it rebuilds, and restores it when the
// test ends.
func restoreInitGlobals(t *testing.T) {
	t.Helper()
	origConf := conf
	origLogger := Logger
	origMainRPCClient := mainRPCClient
	origBorRPCClient := borRPCClient
	origBorClient := borClient
	origBorGRPCClient := borGRPCClient
	origPrivKey := privKeyObject
	origPubKey := pubKeyObject
	origProducerVotes := producerVotes
	origRio := rioHeight
	origTally := tallyFixHeight
	origDisableVP := disableVPCheckHeight
	origDisableVal := disableValSetCheckHeight
	origInitial := initialHeight
	origProducerDown := producerDowntimeHeight
	origPhuket := phuketHardforkHeight
	origFeeGate := feeWithdrawValidatorGateHeight
	origZurich := zurichHardforkHeight
	origSpan := ithacaHeight
	origKyoto := kyotoHeight
	origCompact := compactVoteExtHeight
	t.Cleanup(func() {
		conf = origConf
		Logger = origLogger
		mainRPCClient = origMainRPCClient
		borRPCClient = origBorRPCClient
		borClient = origBorClient
		borGRPCClient = origBorGRPCClient
		privKeyObject = origPrivKey
		pubKeyObject = origPubKey
		producerVotes = origProducerVotes
		rioHeight = origRio
		tallyFixHeight = origTally
		disableVPCheckHeight = origDisableVP
		disableValSetCheckHeight = origDisableVal
		initialHeight = origInitial
		producerDowntimeHeight = origProducerDown
		phuketHardforkHeight = origPhuket
		feeWithdrawValidatorGateHeight = origFeeGate
		zurichHardforkHeight = origZurich
		ithacaHeight = origSpan
		kyotoHeight = origKyoto
		compactVoteExtHeight = origCompact
	})
}

func newRPCStub(t *testing.T) string {
	t.Helper()
	rpcStub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x1"}`))
	}))
	t.Cleanup(rpcStub.Close)
	return rpcStub.URL
}

func mkTestHome(t *testing.T, rpcURL, chain string) string {
	t.Helper()

	home := t.TempDir()
	configDir := filepath.Join(home, "config")
	require.NoError(t, os.MkdirAll(configDir, 0o755))

	pv := privval.GenFilePV(
		filepath.Join(configDir, privValJsonFile),
		filepath.Join(configDir, "priv_validator_state.json"),
	)
	pv.Save()

	appToml := fmt.Sprintf(`
[custom]
eth_rpc_url = %q
bor_rpc_url = %q
bor_grpc_flag = false
bor_grpc_url = ""
chain = %q
producer_votes = ""
`, rpcURL, rpcURL, chain)
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "app.toml"), []byte(appToml), 0o644))
	return home
}
