package heimdalld

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func blockLimits(t *testing.T, genesisData map[string]interface{}) map[string]interface{} {
	t.Helper()
	cp := genesisData["consensus_params"].(map[string]interface{})
	return cp["block"].(map[string]interface{})
}

func TestAddMissingCometBFTConsensusParams_BlockLimits(t *testing.T) {
	tests := []struct {
		name         string
		block        map[string]interface{}
		wantMaxBytes string
		wantMaxGas   string
	}{
		{
			name:         "non-default source limits are preserved",
			block:        map[string]interface{}{"max_bytes": "22020096", "max_gas": "-1"},
			wantMaxBytes: "22020096",
			wantMaxGas:   "-1",
		},
		{
			name:         "missing limits are filled with CometBFT defaults",
			block:        nil,
			wantMaxBytes: "2097152",
			wantMaxGas:   "10000000",
		},
		{
			name:         "mainnet-shaped limits are unchanged",
			block:        map[string]interface{}{"max_bytes": "2097152", "max_gas": "10000000"},
			wantMaxBytes: "2097152",
			wantMaxGas:   "10000000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			consensusParams := map[string]interface{}{
				"evidence": map[string]interface{}{},
			}
			if tt.block != nil {
				consensusParams["block"] = tt.block
			}
			genesisData := map[string]interface{}{"consensus_params": consensusParams}

			require.NoError(t, addMissingCometBFTConsensusParams(genesisData, 100))

			block := blockLimits(t, genesisData)
			require.Equal(t, tt.wantMaxBytes, block["max_bytes"])
			require.Equal(t, tt.wantMaxGas, block["max_gas"])
		})
	}
}

// A malformed block node (present but not an object) makes the fill-only block-limit
// writes fail, and the error must propagate rather than be swallowed.
func TestAddMissingCometBFTConsensusParams_PropagatesBlockError(t *testing.T) {
	genesisData := map[string]interface{}{
		"consensus_params": map[string]interface{}{
			"evidence": map[string]interface{}{},
			"block":    "not-a-map",
		},
	}
	require.Error(t, addMissingCometBFTConsensusParams(genesisData, 100))
}

// After filling block limits the function must still set the remaining CometBFT params;
// guards against an early return between the block-limit writes and the rest.
func TestAddMissingCometBFTConsensusParams_SetsAllParams(t *testing.T) {
	genesisData := map[string]interface{}{
		"consensus_params": map[string]interface{}{"evidence": map[string]interface{}{}},
	}
	require.NoError(t, addMissingCometBFTConsensusParams(genesisData, 7))

	cp := genesisData["consensus_params"].(map[string]interface{})
	require.Contains(t, cp, "validator")
	require.Contains(t, cp, "version")
	abci, ok := cp["abci"].(map[string]interface{})
	require.True(t, ok)
	require.Equal(t, "7", abci["vote_extensions_enable_height"])
}
