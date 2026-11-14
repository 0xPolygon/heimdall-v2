package util

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"cosmossdk.io/log"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/helper"
)

func TestUnconfirmedTxnCount(t *testing.T) {
	// Not parallel: points the global helper config and client at a stub.
	tests := []struct {
		name    string
		status  int
		body    string
		down    bool
		want    int
		wantErr string
	}{
		{name: "mempool size", status: http.StatusOK, body: `{"result":{"n_txs":"30","total":"1200"}}`, want: 1200},
		{name: "node unreachable", down: true, wantErr: "fetching"},
		{name: "non-200", status: http.StatusServiceUnavailable, body: `busy`, wantErr: "status 503"},
		{name: "malformed body", status: http.StatusOK, body: `{`, wantErr: "unexpected end of JSON input"},
		{name: "non-numeric total", status: http.StatusOK, body: `{"result":{"total":"lots"}}`, wantErr: `total "lots"`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, CometBFTUnconfirmedTxsCountURL, r.URL.Path)
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			url := srv.URL
			if tc.down {
				srv.Close()
			} else {
				defer srv.Close()
			}

			conf := helper.GetDefaultHeimdallConfig()
			conf.CometBFTRPCUrl = url
			helper.SetTestConfig(helper.CustomAppConfig{Config: *serverconfig.DefaultConfig(), Custom: conf})
			prev := helper.Client
			helper.Client = &http.Client{}
			t.Cleanup(func() { helper.Client = prev })

			got, err := UnconfirmedTxnCount()
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				require.Zero(t, got)

				logs := &bytes.Buffer{}
				prevLogger := helper.Logger
				helper.Logger = log.NewLogger(logs, log.ColorOption(false))
				t.Cleanup(func() { helper.Logger = prevLogger })
				require.Zero(t, GetUnconfirmedTxnCount())
				require.Contains(t, logs.String(), "Error fetching mempool txs count")
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
			require.Equal(t, tc.want, GetUnconfirmedTxnCount())
		})
	}
}

func TestCalculateTaskDelay(t *testing.T) {
	// Not parallel: points the global helper config, client and key at a stub.
	helper.SetTestPrivPubKey(secp256k1.GenPrivKey())
	self, err := helper.GetAddressString()
	require.NoError(t, err)
	const other = "0x00000000000000000000000000000000000000aa"
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())

	tests := []struct {
		name      string
		status    int
		signers   []string
		mempool   string
		wantOK    bool
		wantDelay time.Duration
		wantErr   bool
	}{
		{name: "second validator, quiet mempool", status: http.StatusOK, signers: []string{other, self}, mempool: "999", wantOK: true, wantDelay: 2 * TaskDelayBetweenEachVal},
		{name: "second validator, busy mempool", status: http.StatusOK, signers: []string{other, self}, mempool: "2500", wantOK: true, wantDelay: 6 * TaskDelayBetweenEachVal},
		{name: "first validator", status: http.StatusOK, signers: []string{self, other}, mempool: "1000", wantOK: true, wantDelay: 2 * TaskDelayBetweenEachVal},
		{name: "not a validator", status: http.StatusOK, signers: []string{other}, mempool: "2500"},
		{name: "validator set unavailable", status: http.StatusInternalServerError, wantErr: true},
		{name: "malformed signer", status: http.StatusOK, signers: []string{"0xzz"}, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == CometBFTUnconfirmedTxsCountURL {
					_, _ = fmt.Fprintf(w, `{"result":{"total":%q}}`, tc.mempool)
					return
				}
				require.Equal(t, CurrentValidatorSetURL, r.URL.Path)
				validators := make([]string, 0, len(tc.signers))
				for _, signer := range tc.signers {
					validators = append(validators, fmt.Sprintf(`{"signer":%q}`, signer))
				}
				w.WriteHeader(tc.status)
				_, _ = fmt.Fprintf(w, `{"validator_set":{"validators":[%s]}}`, strings.Join(validators, ","))
			}))
			defer srv.Close()

			server := *serverconfig.DefaultConfig()
			server.API.Address = srv.URL
			conf := helper.GetDefaultHeimdallConfig()
			conf.CometBFTRPCUrl = srv.URL
			helper.SetTestConfig(helper.CustomAppConfig{Config: server, Custom: conf})
			prev := helper.Client
			helper.Client = &http.Client{}
			t.Cleanup(func() { helper.Client = prev })

			ok, delay, err := CalculateTaskDelay(cdc)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tc.wantOK, ok)
			require.Equal(t, tc.wantDelay, delay)
		})
	}
}
