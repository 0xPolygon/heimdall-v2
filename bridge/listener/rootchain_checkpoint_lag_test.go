package listener

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	checkpointTypes "github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
)

func TestFetchCommittedCheckpoint(t *testing.T) {
	t.Run("returns committed checkpoint", func(t *testing.T) {
		test := setupOrchestrationTest(t, fixedSubgraph(`{"data":{}}`))
		defer test.close()

		resp := checkpointTypes.QueryCheckpointLatestResponse{
			Checkpoint: checkpointTypes.Checkpoint{Id: 100, StartBlock: 9000, EndBlock: 9599},
		}
		body, err := test.listener.cliCtx.Codec.MarshalJSON(&resp)
		require.NoError(t, err)

		test.heimdall.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(body)
		})

		cp, ok := test.listener.fetchCommittedCheckpoint()
		require.True(t, ok)
		require.NotNil(t, cp)
		require.Equal(t, uint64(100), cp.Id)
		require.Equal(t, uint64(9599), cp.EndBlock)
	})

	t.Run("fails open on REST error", func(t *testing.T) {
		test := setupOrchestrationTest(t, fixedSubgraph(`{"data":{}}`))
		defer test.close()

		test.heimdall.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "boom", http.StatusInternalServerError)
		})

		cp, ok := test.listener.fetchCommittedCheckpoint()
		require.False(t, ok)
		require.Nil(t, cp)
	})
}

func TestComputeCheckpointLag(t *testing.T) {
	tests := []struct {
		name          string
		l1Id          uint64
		l1End         uint64
		committedId   uint64
		committedEnd  uint64
		wantRaw       float64
		wantEffective float64
		wantBehind    bool
	}{
		{"synced", 100, 9599, 100, 9599, 0, 0, false},
		{"normal in-flight window", 101, 9700, 100, 9599, 101, 0, false},
		{"behind by two", 102, 9751, 100, 9599, 152, 152, true},
		{"far behind", 130, 12000, 100, 9599, 2401, 2401, true},
		{"committed ahead clamps", 100, 9599, 101, 9700, 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw, effective, behind := computeCheckpointLag(tt.l1Id, tt.l1End, tt.committedId, tt.committedEnd)
			if raw != tt.wantRaw || effective != tt.wantEffective || behind != tt.wantBehind {
				t.Errorf("computeCheckpointLag(%d,%d,%d,%d) = (%v,%v,%v), want (%v,%v,%v)",
					tt.l1Id, tt.l1End, tt.committedId, tt.committedEnd,
					raw, effective, behind, tt.wantRaw, tt.wantEffective, tt.wantBehind)
			}
		})
	}
}

func TestCheckpointLagInterval(t *testing.T) {
	if checkpointLagInterval != 30*time.Second {
		t.Errorf("checkpointLagInterval = %v, want %v", checkpointLagInterval, 30*time.Second)
	}
}

func TestCheckpointLagBlocks(t *testing.T) {
	tests := []struct {
		name         string
		l1End        uint64
		committedEnd uint64
		want         float64
	}{
		{"synced", 9599, 9599, 0},
		{"behind", 9751, 9599, 152},
		{"ahead clamps to zero", 9599, 9751, 0},
		{"normal in-flight window", 9700, 9599, 101},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := checkpointLagBlocks(tt.l1End, tt.committedEnd); got != tt.want {
				t.Errorf("checkpointLagBlocks(%d, %d) = %v, want %v", tt.l1End, tt.committedEnd, got, tt.want)
			}
		})
	}
}

func TestCheckpointBehind(t *testing.T) {
	tests := []struct {
		name        string
		l1Id        uint64
		committedId uint64
		want        bool
	}{
		{"synced", 100, 100, false},
		{"normal one-in-flight window", 101, 100, false},
		{"behind by two", 102, 100, true},
		{"far behind", 130, 100, true},
		{"committed ahead", 100, 101, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := checkpointBehind(tt.l1Id, tt.committedId); got != tt.want {
				t.Errorf("checkpointBehind(%d, %d) = %v, want %v", tt.l1Id, tt.committedId, got, tt.want)
			}
		})
	}
}

func TestEffectiveLag(t *testing.T) {
	tests := []struct {
		name   string
		lag    float64
		behind bool
		want   float64
	}{
		{"behind reports raw lag", 152, true, 152},
		{"not behind suppressed", 101, false, 0},
		{"behind zero lag", 0, true, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := effectiveLag(tt.lag, tt.behind); got != tt.want {
				t.Errorf("effectiveLag(%v, %v) = %v, want %v", tt.lag, tt.behind, got, tt.want)
			}
		})
	}
}
