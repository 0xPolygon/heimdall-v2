package helper

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/cometbft/cometbft/privval"
	"github.com/stretchr/testify/require"
)

// TestInitHeimdallConfigWithSetsIthacaHeightPerChain drives the
// real init path through each chain branch so the Ithaca assignment lines in
// InitHeimdallConfigWith stay covered. A tiny JSON-RPC stub and a real
// priv_validator_key.json keep the init code on the happy path without touching
// production services.
func TestInitHeimdallConfigWithSetsIthacaHeightPerChain(t *testing.T) {
	origConf := conf
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
	t.Cleanup(func() {
		conf = origConf
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
	})

	rpcStub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x1"}`))
	}))
	defer rpcStub.Close()

	mkHome := func(t *testing.T, chain string) string {
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
`, rpcStub.URL, rpcStub.URL, chain)
		require.NoError(t, os.WriteFile(filepath.Join(configDir, "app.toml"), []byte(appToml), 0o644))
		return home
	}

	cases := []struct {
		name             string
		chain            string
		wantRioHeight    int64
		wantInitHeight   int64
		wantIthacaHeight int64
	}{
		{name: "mainnet", chain: MainChain, wantRioHeight: 77414656, wantInitHeight: 24404501, wantIthacaHeight: 50185000},
		{name: "mumbai", chain: MumbaiChain, wantRioHeight: 48473856, wantInitHeight: 0, wantIthacaHeight: 0},
		{name: "amoy", chain: AmoyChain, wantRioHeight: 26272256, wantInitHeight: 8788501, wantIthacaHeight: 40776000},
		{name: "default", chain: "local", wantRioHeight: 128, wantInitHeight: 0, wantIthacaHeight: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			conf = CustomAppConfig{}

			home := mkHome(t, tc.chain)
			InitHeimdallConfigWith(home, "")

			require.Equal(t, tc.wantIthacaHeight, GetIthacaHeight())
			require.Equal(t, tc.wantRioHeight, GetRioHeight())
			require.Equal(t, tc.wantInitHeight, GetInitialHeight())
		})
	}
}
