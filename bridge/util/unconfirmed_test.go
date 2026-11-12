package util

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"cosmossdk.io/log"
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

			got, err := UnconfirmedTxnCount(nil)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				require.Zero(t, got)

				logs := &bytes.Buffer{}
				prevLogger := helper.Logger
				helper.Logger = log.NewLogger(logs, log.ColorOption(false))
				t.Cleanup(func() { helper.Logger = prevLogger })
				require.Zero(t, GetUnconfirmedTxnCount(nil))
				require.Contains(t, logs.String(), "Error fetching mempool txs count")
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
			require.Equal(t, tc.want, GetUnconfirmedTxnCount(nil))
		})
	}
}
