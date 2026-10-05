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

// stuckHeimdall serves /stake/validator/ and /stake/is-old-tx. The first
// validator read returns 256; later reads return recheckNonce, or fail when
// recheckStatus is set. is-old-tx returns isOld, or isOldBody verbatim when set.
type stuckHeimdall struct {
	isOldStatus   int
	isOld         bool
	isOldBody     string
	recheckNonce  uint64
	recheckStatus int

	validatorCalls atomic.Int32
	isOldQueries   atomic.Int32
}

func (h *stuckHeimdall) handler(t *testing.T, test *orchestrationTest, validatorID uint64) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasPrefix(r.URL.Path, "/stake/is-old-tx"):
			h.isOldQueries.Add(1)
			require.Equal(t, stuckTestTxHash, r.URL.Query().Get("tx_hash"))
			require.Equal(t, "13", r.URL.Query().Get("log_index"))
			if h.isOldStatus != http.StatusOK {
				w.WriteHeader(h.isOldStatus)
				return
			}
			if h.isOldBody != "" {
				_, _ = w.Write([]byte(h.isOldBody))
				return
			}
			body, err := test.listener.cliCtx.Codec.MarshalJSON(&staketypes.QueryStakeIsOldTxResponse{IsOld: h.isOld})
			require.NoError(t, err)
			_, _ = w.Write(body)
		case strings.HasPrefix(r.URL.Path, "/stake/validator/"):
			nonce := uint64(256)
			if h.validatorCalls.Add(1) > 1 {
				if h.recheckStatus != 0 {
					w.WriteHeader(h.recheckStatus)
					return
				}
				nonce = h.recheckNonce
			}
			_, _ = w.Write(marshalValidatorResponse(t, test.listener, validatorID, nonce))
		default:
			t.Errorf("unexpected heimdall path: %s", r.URL.Path)
		}
	}
}

func TestRecoverStakeEventsForValidator_StuckNonceGuard(t *testing.T) {
	// Not parallel: mutates global helper config.
	const (
		validatorID = uint64(10)
		l1MaxNonce  = uint64(277)
	)

	tests := []struct {
		name               string
		heimdall           *stuckHeimdall
		wantByNonceCalls   int32
		wantValidatorCalls int32
	}{
		{
			name:               "next event processed and nonce still behind skips the validator",
			heimdall:           &stuckHeimdall{isOldStatus: http.StatusOK, isOld: true, recheckNonce: 256},
			wantByNonceCalls:   1,
			wantValidatorCalls: 2,
		},
		{
			name:               "next event committed between reads falls through to replay",
			heimdall:           &stuckHeimdall{isOldStatus: http.StatusOK, isOld: true, recheckNonce: 257},
			wantByNonceCalls:   4,
			wantValidatorCalls: 2,
		},
		{
			name:               "nonce re-read error falls through to replay",
			heimdall:           &stuckHeimdall{isOldStatus: http.StatusOK, isOld: true, recheckStatus: http.StatusInternalServerError},
			wantByNonceCalls:   4,
			wantValidatorCalls: 2,
		},
		{
			name:               "next event not processed falls through to replay",
			heimdall:           &stuckHeimdall{isOldStatus: http.StatusOK, isOld: false},
			wantByNonceCalls:   4,
			wantValidatorCalls: 1,
		},
		{
			name:               "is-old-tx error falls through to replay",
			heimdall:           &stuckHeimdall{isOldStatus: http.StatusInternalServerError},
			wantByNonceCalls:   4,
			wantValidatorCalls: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var byNonceCalls atomic.Int32
			test := setupOrchestrationTest(t, stuckSubgraph(t, l1MaxNonce, &byNonceCalls))
			defer test.close()
			test.heimdall.Config.Handler = tc.heimdall.handler(t, test, validatorID)

			test.listener.recoverStakeEventsForValidator(testContext(t), validatorID)

			require.Equal(t, int32(1), tc.heimdall.isOldQueries.Load())
			require.Equal(t, tc.wantByNonceCalls, byNonceCalls.Load())
			require.Equal(t, tc.wantValidatorCalls, tc.heimdall.validatorCalls.Load())
		})
	}
}

func TestStakeNonceStuck_LookupErrorsReturnFalse(t *testing.T) {
	// Not parallel: mutates global helper config.
	tests := []struct {
		name             string
		subgraphBody     string
		heimdall         *stuckHeimdall
		wantIsOldQueries int32
	}{
		{
			name:         "invalid log index",
			subgraphBody: fmt.Sprintf(`{"data":{"stakeUpdates":[{"transactionHash":"%s","logIndex":"not-a-number"}],"signerChanges":[],"unstakeInits":[]}}`, stuckTestTxHash),
			heimdall:     &stuckHeimdall{},
		},
		{
			name:             "unparseable is-old-tx response",
			subgraphBody:     fmt.Sprintf(`{"data":{"stakeUpdates":[{"transactionHash":"%s","logIndex":"13"}],"signerChanges":[],"unstakeInits":[]}}`, stuckTestTxHash),
			heimdall:         &stuckHeimdall{isOldStatus: http.StatusOK, isOldBody: "{not json"},
			wantIsOldQueries: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			test := setupOrchestrationTest(t, fixedSubgraph(tc.subgraphBody))
			defer test.close()
			test.heimdall.Config.Handler = tc.heimdall.handler(t, test, 10)

			require.False(t, test.listener.stakeNonceStuck(testContext(t), 10, 257))
			require.Equal(t, tc.wantIsOldQueries, tc.heimdall.isOldQueries.Load())
			require.Zero(t, tc.heimdall.validatorCalls.Load())
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
