package heimdalld

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	goproto "github.com/cosmos/gogoproto/proto"

	"github.com/stretchr/testify/require"
)

// This is a transaction from a devnet containing checkpoint data.
// It is used to test the GetVEsFromEndpoint function.
const mockVe = "Ev4DChkKFD8lS+LJZ4GOFSPHnlInKlXOXDVaGJBOGpQBCiBFKm295tFbO79Fv6H+LcXwTMs93jbCa82EWwaxzwXyFhCkCRokCiDlFvDlR6TrRx39NmXW4+5emv7jg526PygI7FQa9UvpDhABIkcKID+Vlc33ZOu6xc+eM0ziK8kEbnOv7JV09P6joVYZ4Q+CELYDGiBU7bPt6Oy5LpFz4pVPtg3fRkxgJVvCETEi+Irgc+YiYCJBoONnKbnZjeKZed80TGr9bY0x6kuYQRKgZ6RQ4w8SVc8zet6lgmx7L7x5EEZRZRyHcRq5gAZ0srb0Tbz7Y5W+UgEoAjLBAQEAAAAAAAAAAAAAAADBjKQ0Pi8/xEAJhuwmlo+M93Z16AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAB/AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAX7JNXRtZm3DFHC1AlXzudX4QVMiUBM8WdOCXCAI3ToxgDNGSa3dv9mWlW1sSbqq4WY6gE7Jf3ttdQAlMEKphDsaAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAJvg6QdDbohMITZaEP5gR+dK3RB5lfJZ9ZyN4FiIPMGRmerC7ZLruZsspRHyvdatmDSACYlesWuyMKsXEun+sURERxisBEv4DChkKFMGMpDQ+Lz/EQAmG7CaWj4z3dnXoGJBOGpQBCiBFKm295tFbO79Fv6H+LcXwTMs93jbCa82EWwaxzwXyFhCkCRokCiDlFvDlR6TrRx39NmXW4+5emv7jg526PygI7FQa9UvpDhABIkcKID+Vlc33ZOu6xc+eM0ziK8kEbnOv7JV09P6joVYZ4Q+CELYDGiBU7bPt6Oy5LpFz4pVPtg3fRkxgJVvCETEi+Irgc+YiYCJBfb8DFB6M8Al39EoXywnZaVl605QTB4n5m4/5rYaONu0etC8XhmiWNLhUu1EnH1GUw2UGXz5NshfgZFxOW/iNfAEoAjLBAQEAAAAAAAAAAAAAAADBjKQ0Pi8/xEAJhuwmlo+M93Z16AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAB/AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAX7JNXRtZm3DFHC1AlXzudX4QVMiUBM8WdOCXCAI3ToxgDNGSa3dv9mWlW1sSbqq4WY6gE7Jf3ttdQAlMEKphDsaAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAJvg6QTLOQ7CIuNzRuqJOAywdprWGMom+Oyqz3kyZYUe4brRfKnYQUwFYERcYLgHnNIiykjWPAvDYGsMeQ6mB0rq8Mx0B"

// TestGetVEsFromEndpoint_MockServer tests the getVEsFromEndpoint function using a mock server.
func TestGetVEsFromEndpoint_MockServer(t *testing.T) {
	mockResp := map[string]any{
		"result": map[string]any{
			"block": map[string]any{
				"data": map[string][]string{
					"txs": {mockVe},
				},
			},
		},
	}
	body, err := json.Marshal(mockResp)
	require.NoError(t, err)
	t.Logf("Mock response JSON: %s", string(body))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("Received request %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusOK)
		w.Write(body)
	}))
	defer server.Close()
	t.Logf("Mock server URL: %s", server.URL)

	u, err := url.Parse(server.URL)
	require.NoError(t, err)
	host := u.Hostname()
	port, err := strconv.ParseUint(u.Port(), 10, 64)
	require.NoError(t, err)
	t.Logf("Using host=%s port=%d", host, port)

	extInfo, err := GetVEsFromEndpoint(1189, host, port)
	require.NoError(t, err)
	require.NotNil(t, extInfo)
	t.Logf("Unmarshaled ExtendedCommitInfo: Round=%d, Votes=%d", extInfo.Round, len(extInfo.Votes))
	require.NotEmpty(t, extInfo.Votes)

	decoded, err := base64.StdEncoding.DecodeString(mockVe)
	require.NoError(t, err)
	var expected abci.ExtendedCommitInfo
	require.NoError(t, goproto.Unmarshal(decoded, &expected))
	t.Logf("Expected ExtendedCommitInfo: Round=%d, Votes=%d", expected.Round, len(expected.Votes))

	require.Equal(t, &expected, extInfo)
}

func TestBuildCommitJSON_MockVe(t *testing.T) {
	serverResp := map[string]any{
		"result": map[string]any{
			"block": map[string]any{
				"data": map[string][]string{
					"txs": {mockVe},
				},
			},
		},
	}
	body, err := json.Marshal(serverResp)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(body)
	}))
	defer server.Close()

	u, err := url.Parse(server.URL)
	require.NoError(t, err)
	host := u.Hostname()
	port, err := strconv.ParseUint(u.Port(), 10, 64)
	require.NoError(t, err)

	extInfo, err := GetVEsFromEndpoint(1189, host, port)
	require.NoError(t, err)

	out, err := BuildCommitJSON(1189, "heimdall-9976", extInfo)
	require.NoError(t, err)

	expectedBytes, err := os.ReadFile("testdata/mock-decoded-ve.json")
	require.NoError(t, err)

	require.JSONEq(t,
		string(expectedBytes),
		string(out),
	)
}

func TestGetVEsFromEndpoint_NoTxs(t *testing.T) {
	mockResp := map[string]any{
		"result": map[string]any{
			"block": map[string]any{
				"data": map[string][]string{
					"txs": {},
				},
			},
		},
	}
	body, err := json.Marshal(mockResp)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(body)
	}))
	defer server.Close()

	u, err := url.Parse(server.URL)
	require.NoError(t, err)
	host := u.Hostname()
	port, err := strconv.ParseUint(u.Port(), 10, 64)
	require.NoError(t, err)

	_, err = GetVEsFromEndpoint(1189, host, port)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no txs found in the block")
}
