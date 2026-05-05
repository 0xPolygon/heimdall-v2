package listener

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"cosmossdk.io/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestJoinGraphQLErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		errs []graphqlError
		want string
	}{
		{name: "empty input", errs: nil, want: ""},
		{name: "single message", errs: []graphqlError{{Message: "field signerChanges not found"}}, want: "field signerChanges not found"},
		{name: "multiple messages joined with semicolons", errs: []graphqlError{{Message: "a"}, {Message: "b"}}, want: "a; b"},
		{name: "empty message rendered as placeholder", errs: []graphqlError{{Message: ""}, {Message: "real"}}, want: "<no message>; real"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, joinGraphQLErrors(tt.errs))
		})
	}
}

func TestFindLogByIndex(t *testing.T) {
	t.Parallel()

	logs := []*types.Log{
		{Index: 0, TxHash: common.HexToHash("0xa")},
		{Index: 3, TxHash: common.HexToHash("0xb")},
		{Index: 17, TxHash: common.HexToHash("0xc")},
	}

	t.Run("returns matching log", func(t *testing.T) {
		t.Parallel()
		got := findLogByIndex(logs, "3")
		require.NotNil(t, got)
		require.Equal(t, common.HexToHash("0xb"), got.TxHash)
	})

	t.Run("returns first log when index is 0", func(t *testing.T) {
		t.Parallel()
		got := findLogByIndex(logs, "0")
		require.NotNil(t, got)
		require.Equal(t, common.HexToHash("0xa"), got.TxHash)
	})

	t.Run("returns nil when index not found", func(t *testing.T) {
		t.Parallel()
		require.Nil(t, findLogByIndex(logs, "99"))
	})

	t.Run("returns nil when logs are empty", func(t *testing.T) {
		t.Parallel()
		require.Nil(t, findLogByIndex(nil, "0"))
	})

	t.Run("does not match by prefix", func(t *testing.T) {
		t.Parallel()
		require.Nil(t, findLogByIndex(logs, "1"))
	})
}

func TestMaxNonceFromResponse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		response stakeEventMaxNonceResponse
		want     uint64
		wantErr  bool
	}{
		{name: "all empty returns zero", response: stakeEventMaxNonceResponse{}, want: 0},
		{name: "only stake updates", response: makeMaxNonceResp("17", "", ""), want: 17},
		{name: "only signer changes", response: makeMaxNonceResp("", "23", ""), want: 23},
		{name: "only unstake inits", response: makeMaxNonceResp("", "", "9"), want: 9},
		{name: "max is signer change", response: makeMaxNonceResp("5", "11", "8"), want: 11},
		{name: "max is unstake init", response: makeMaxNonceResp("5", "11", "42"), want: 42},
		{name: "max is stake update", response: makeMaxNonceResp("99", "11", "8"), want: 99},
		{name: "ties are stable", response: makeMaxNonceResp("7", "7", "7"), want: 7},
		{name: "malformed nonce returns error", response: makeMaxNonceResp("not-a-number", "7", ""), wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := maxNonceFromResponse(tt.response)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestPickStakeEventHit(t *testing.T) {
	t.Parallel()

	stake := txAndLogIndex{TransactionHash: "0xstake", LogIndex: "1"}
	signer := txAndLogIndex{TransactionHash: "0xsigner", LogIndex: "2"}
	exit := txAndLogIndex{TransactionHash: "0xexit", LogIndex: "3"}

	tests := []struct {
		name     string
		response stakeEventByNonceResponse
		want     *txAndLogIndex
	}{
		{name: "no hit returns nil", response: stakeEventByNonceResponse{}, want: nil},
		{name: "stake update hit", response: byNonceResp(&stake, nil, nil), want: &stake},
		{name: "signer change hit", response: byNonceResp(nil, &signer, nil), want: &signer},
		{name: "unstake init hit", response: byNonceResp(nil, nil, &exit), want: &exit},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := pickStakeEventHit(tt.response)
			if tt.want == nil {
				require.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			require.Equal(t, *tt.want, *got)
		})
	}
}

func TestGetMaxL1NonceForValidator(t *testing.T) {
	t.Parallel()

	t.Run("returns max across three event types", func(t *testing.T) {
		t.Parallel()
		server := newSubgraph(`{"data":{"stakeUpdates":[{"nonce":"3"}],"signerChanges":[{"nonce":"7"}],"unstakeInits":[{"nonce":"5"}]}}`)
		defer server.Close()

		got, err := newSelfHealTestListener(server.URL).getMaxL1NonceForValidator(testContext(t), 42)
		require.NoError(t, err)
		require.Equal(t, uint64(7), got)
	})

	t.Run("returns zero when validator has no events on L1", func(t *testing.T) {
		t.Parallel()
		server := newSubgraph(`{"data":{"stakeUpdates":[],"signerChanges":[],"unstakeInits":[]}}`)
		defer server.Close()

		got, err := newSelfHealTestListener(server.URL).getMaxL1NonceForValidator(testContext(t), 42)
		require.NoError(t, err)
		require.Equal(t, uint64(0), got)
	})

	t.Run("returns error on malformed response", func(t *testing.T) {
		t.Parallel()
		server := newSubgraph(`not json`)
		defer server.Close()

		_, err := newSelfHealTestListener(server.URL).getMaxL1NonceForValidator(testContext(t), 42)
		require.Error(t, err)
	})

	t.Run("query body includes the validator id and all three entities", func(t *testing.T) {
		t.Parallel()
		var seen string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body := make([]byte, r.ContentLength)
			_, _ = r.Body.Read(body)
			seen = string(body)
			writeJSON(w, `{"data":{"stakeUpdates":[],"signerChanges":[],"unstakeInits":[]}}`)
		}))
		defer server.Close()

		_, err := newSelfHealTestListener(server.URL).getMaxL1NonceForValidator(testContext(t), 12345)
		require.NoError(t, err)
		require.Contains(t, seen, "validatorId: 12345")
		require.Contains(t, seen, "stakeUpdates")
		require.Contains(t, seen, "signerChanges")
		require.Contains(t, seen, "unstakeInits")
	})

	t.Run("fails closed on top-level GraphQL errors", func(t *testing.T) {
		t.Parallel()
		server := newSubgraph(`{"data":null,"errors":[{"message":"field signerChanges not found"}]}`)
		defer server.Close()

		_, err := newSelfHealTestListener(server.URL).getMaxL1NonceForValidator(testContext(t), 42)
		require.Error(t, err)
		require.Contains(t, err.Error(), "subgraph returned errors")
		require.Contains(t, err.Error(), "field signerChanges not found")
	})

	t.Run("fails closed when errors coexist with empty data", func(t *testing.T) {
		t.Parallel()
		server := newSubgraph(`{"data":{"stakeUpdates":[],"signerChanges":[],"unstakeInits":[]},"errors":[{"message":"partial failure"}]}`)
		defer server.Close()

		_, err := newSelfHealTestListener(server.URL).getMaxL1NonceForValidator(testContext(t), 42)
		require.Error(t, err)
		require.Contains(t, err.Error(), "partial failure")
	})

	t.Run("fails closed on malformed nonce", func(t *testing.T) {
		t.Parallel()
		server := newSubgraph(`{"data":{"stakeUpdates":[{"nonce":"not-a-number"}],"signerChanges":[],"unstakeInits":[]}}`)
		defer server.Close()

		_, err := newSelfHealTestListener(server.URL).getMaxL1NonceForValidator(testContext(t), 42)
		require.Error(t, err)
		require.Contains(t, err.Error(), "malformed nonce")
	})
}

func TestGetStakeEventLogByNonce_subgraphPaths(t *testing.T) {
	t.Parallel()

	t.Run("returns error when no entity matches", func(t *testing.T) {
		t.Parallel()
		server := newSubgraph(`{"data":{"stakeUpdates":[],"signerChanges":[],"unstakeInits":[]}}`)
		defer server.Close()

		got, err := newSelfHealTestListener(server.URL).getStakeEventLogByNonce(testContext(t), 42, 5)
		require.Nil(t, got)
		require.Error(t, err)
		require.Contains(t, err.Error(), "no stake event found")
	})

	t.Run("query body includes validator id and nonce", func(t *testing.T) {
		t.Parallel()
		var seen string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body := make([]byte, r.ContentLength)
			_, _ = r.Body.Read(body)
			seen = string(body)
			writeJSON(w, `{"data":{"stakeUpdates":[],"signerChanges":[],"unstakeInits":[]}}`)
		}))
		defer server.Close()

		_, _ = newSelfHealTestListener(server.URL).getStakeEventLogByNonce(testContext(t), 12345, 678)
		require.Contains(t, seen, "validatorId: 12345")
		require.Contains(t, seen, "nonce: 678")
	})

	t.Run("fails closed on top-level GraphQL errors", func(t *testing.T) {
		t.Parallel()
		server := newSubgraph(`{"data":null,"errors":[{"message":"field unstakeInits not found"}]}`)
		defer server.Close()

		got, err := newSelfHealTestListener(server.URL).getStakeEventLogByNonce(testContext(t), 42, 5)
		require.Nil(t, got)
		require.Error(t, err)
		require.Contains(t, err.Error(), "subgraph returned errors")
		require.Contains(t, err.Error(), "field unstakeInits not found")
	})
}

func makeMaxNonceResp(stakeUpdate, signerChange, unstakeInit string) stakeEventMaxNonceResponse {
	r := stakeEventMaxNonceResponse{}
	if stakeUpdate != "" {
		r.Data.StakeUpdates = []nonceOnly{{Nonce: stakeUpdate}}
	}
	if signerChange != "" {
		r.Data.SignerChanges = []nonceOnly{{Nonce: signerChange}}
	}
	if unstakeInit != "" {
		r.Data.UnstakeInits = []nonceOnly{{Nonce: unstakeInit}}
	}
	return r
}

func byNonceResp(stake, signer, exit *txAndLogIndex) stakeEventByNonceResponse {
	r := stakeEventByNonceResponse{}
	if stake != nil {
		r.Data.StakeUpdates = []txAndLogIndex{*stake}
	}
	if signer != nil {
		r.Data.SignerChanges = []txAndLogIndex{*signer}
	}
	if exit != nil {
		r.Data.UnstakeInits = []txAndLogIndex{*exit}
	}
	return r
}

func newSubgraph(body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, body)
	}))
}

func newSelfHealTestListener(graphURL string) *RootChainListener {
	return &RootChainListener{
		BaseListener: BaseListener{
			Logger: log.NewNopLogger(),
		},
		subGraphClient: &subGraphClient{
			graphUrl:   graphURL,
			httpClient: &http.Client{Timeout: 5 * time.Second},
		},
	}
}

func writeJSON(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(body))
}

func testContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}
