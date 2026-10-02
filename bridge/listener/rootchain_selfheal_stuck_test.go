package listener

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	staketypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

const stuckTestTxHash = "0x00000000000000000000000000000000000000000000000000000000000000aa"

// stuckSubgraph answers max-nonce queries with l1MaxNonce. The first by-nonce
// lookup returns an event ref; later ones return GraphQL errors, so a replay
// stops before the receipt fetch (the zero contract caller would panic).
func stuckSubgraph(t *testing.T, l1MaxNonce uint64, byNonceCalls *atomic.Int32) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		if strings.Contains(string(body), "orderBy: nonce") {
			writeJSON(w, fmt.Sprintf(`{"data":{"stakeUpdates":[{"nonce":"%d"}],"signerChanges":[],"unstakeInits":[]}}`, l1MaxNonce))
			return
		}
		if byNonceCalls.Add(1) == 1 {
			writeJSON(w, fmt.Sprintf(`{"data":{"stakeUpdates":[{"transactionHash":"%s","logIndex":"13"}],"signerChanges":[],"unstakeInits":[]}}`, stuckTestTxHash))
			return
		}
		writeJSON(w, `{"data":null,"errors":[{"message":"stop replay"}]}`)
	}
}

func TestRecoverStakeEventsForValidator_StuckNonceGuard(t *testing.T) {
	// Not parallel: mutates global helper config.
	const (
		validatorID   = uint64(10)
		heimdallNonce = uint64(256)
		l1MaxNonce    = uint64(277)
	)

	tests := []struct {
		name             string
		isOldStatus      int
		isOld            bool
		wantByNonceCalls int32
	}{
		{name: "next event already processed skips the validator", isOldStatus: http.StatusOK, isOld: true, wantByNonceCalls: 1},
		{name: "next event not processed falls through to replay", isOldStatus: http.StatusOK, isOld: false, wantByNonceCalls: 4},
		{name: "is-old-tx error falls through to replay", isOldStatus: http.StatusInternalServerError, wantByNonceCalls: 4},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var byNonceCalls atomic.Int32
			test := setupOrchestrationTest(t, stuckSubgraph(t, l1MaxNonce, &byNonceCalls))
			defer test.close()

			var isOldQueries atomic.Int32
			test.heimdall.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch {
				case strings.HasPrefix(r.URL.Path, "/stake/is-old-tx"):
					isOldQueries.Add(1)
					require.Equal(t, stuckTestTxHash, r.URL.Query().Get("tx_hash"))
					require.Equal(t, "13", r.URL.Query().Get("log_index"))
					if tc.isOldStatus != http.StatusOK {
						w.WriteHeader(tc.isOldStatus)
						return
					}
					body, err := test.listener.cliCtx.Codec.MarshalJSON(&staketypes.QueryStakeIsOldTxResponse{IsOld: tc.isOld})
					require.NoError(t, err)
					_, _ = w.Write(body)
				case strings.HasPrefix(r.URL.Path, "/stake/validator/"):
					_, _ = w.Write(marshalValidatorResponse(t, test.listener, validatorID, heimdallNonce))
				default:
					t.Errorf("unexpected heimdall path: %s", r.URL.Path)
				}
			})

			test.listener.recoverStakeEventsForValidator(testContext(t), validatorID)

			require.Equal(t, int32(1), isOldQueries.Load())
			require.Equal(t, tc.wantByNonceCalls, byNonceCalls.Load())
		})
	}
}

func TestStakeNonceStuck_SubgraphErrorReturnsFalse(t *testing.T) {
	// Not parallel: mutates global helper config.
	test := setupOrchestrationTest(t, fixedSubgraph(`{"data":null,"errors":[{"message":"down"}]}`))
	defer test.close()

	test.heimdall.Config.Handler = http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("heimdall must not be queried when the subgraph fails: %s", r.URL.Path)
	})

	require.False(t, test.listener.stakeNonceStuck(testContext(t), 10, 257))
}
