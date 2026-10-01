package peerpolicy

import (
	"testing"

	"github.com/cometbft/cometbft/p2p/observation"
	"github.com/stretchr/testify/require"
)

func TestPeerPolicyMetrics(t *testing.T) {
	t.Parallel()
	tr, now, reg := fixture(t)
	for i := 0; i < 2; i++ {
		tr.Observe("private-node-id", observation.Event{Kind: observation.InvalidEncoding, Bytes: 17})
	}
	*now = windowWidth
	tr.Observe("private-node-id", observation.Event{Kind: observation.InvalidMessage})
	metrics, err := reg.Gather()
	require.NoError(t, err)
	found := make(map[string]float64)
	for _, family := range metrics {
		for _, metric := range family.Metric {
			key := family.GetName()
			for _, label := range metric.Label {
				require.NotEqual(t, "peer_id", label.GetName())
				require.NotEqual(t, "private-node-id", label.GetValue())
				key += "/" + label.GetValue()
			}
			if metric.Histogram != nil {
				require.Equal(t, uint64(3), metric.Histogram.GetSampleCount())
				require.Equal(t, float64(220), metric.Histogram.GetSampleSum())
			}
			if metric.Counter != nil {
				found[key] = metric.Counter.GetValue()
			}
		}
	}
	require.Equal(t, float64(2), found["heimdall_peer_reputation_reason_events_total/invalid_encoding"])
	require.Equal(t, float64(1), found["heimdall_peer_reputation_reason_windows_total/invalid_encoding"])
	require.Equal(t, float64(1), found["heimdall_peer_reputation_reason_windows_total/invalid_message"])
	require.Equal(t, float64(1), found["heimdall_peer_reputation_would_actions_total/throttle"])
	require.Equal(t, float64(1), found["heimdall_peer_reputation_would_actions_total/jail"])
	require.Equal(t, float64(34), found["heimdall_peer_reputation_bytes_total/other"])
}
