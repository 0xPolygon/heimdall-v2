package rest_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/types/rest"
)

func TestCommitTxProof(t *testing.T) {
	tests := []struct {
		name  string
		proof rest.CommitTxProof
	}{
		{
			name: "full proof",
			proof: rest.CommitTxProof{
				Vote:  "0x123abc",
				Sigs:  "0xsignature",
				Tx:    "0xtxhash",
				Proof: "0xproofdata",
			},
		},
		{
			name: "empty proof",
			proof: rest.CommitTxProof{
				Vote:  "",
				Sigs:  "",
				Tx:    "",
				Proof: "",
			},
		},
		{
			name: "partial proof",
			proof: rest.CommitTxProof{
				Vote: "0x123",
				Tx:   "0xabc",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify all fields are accessible
			require.Equal(t, tt.proof.Vote, tt.proof.Vote)
			require.Equal(t, tt.proof.Sigs, tt.proof.Sigs)
			require.Equal(t, tt.proof.Tx, tt.proof.Tx)
			require.Equal(t, tt.proof.Proof, tt.proof.Proof)
		})
	}
}

func TestCommitTxProofJSON(t *testing.T) {
	proof := rest.CommitTxProof{
		Vote:  "vote_data",
		Sigs:  "sigs_data",
		Tx:    "tx_data",
		Proof: "proof_data",
	}

	// Marshal to JSON
	data, err := json.Marshal(proof)
	require.NoError(t, err)
	require.NotEmpty(t, data)

	// Unmarshal from JSON
	var decoded rest.CommitTxProof
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	// Verify fields match
	require.Equal(t, proof.Vote, decoded.Vote)
	require.Equal(t, proof.Sigs, decoded.Sigs)
	require.Equal(t, proof.Tx, decoded.Tx)
	require.Equal(t, proof.Proof, decoded.Proof)
}

func TestSideTxProof(t *testing.T) {
	tests := []struct {
		name  string
		proof rest.SideTxProof
	}{
		{
			name: "full proof with multiple signatures",
			proof: rest.SideTxProof{
				Sigs: [][3]string{
					{"sig1_part1", "sig1_part2", "sig1_part3"},
					{"sig2_part1", "sig2_part2", "sig2_part3"},
				},
				Tx:   "0xtxhash",
				Data: "0xdata",
			},
		},
		{
			name: "empty proof",
			proof: rest.SideTxProof{
				Sigs: nil,
				Tx:   "",
				Data: "",
			},
		},
		{
			name: "single signature",
			proof: rest.SideTxProof{
				Sigs: [][3]string{
					{"v", "r", "s"},
				},
				Tx:   "0x123",
				Data: "0xabc",
			},
		},
		{
			name: "empty signatures array",
			proof: rest.SideTxProof{
				Sigs: [][3]string{},
				Tx:   "0x123",
				Data: "0xabc",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify all fields are accessible
			require.Equal(t, tt.proof.Sigs, tt.proof.Sigs)
			require.Equal(t, tt.proof.Tx, tt.proof.Tx)
			require.Equal(t, tt.proof.Data, tt.proof.Data)
		})
	}
}

func TestSideTxProofJSON(t *testing.T) {
	proof := rest.SideTxProof{
		Sigs: [][3]string{
			{"v1", "r1", "s1"},
			{"v2", "r2", "s2"},
		},
		Tx:   "tx_hash",
		Data: "tx_data",
	}

	// Marshal to JSON
	data, err := json.Marshal(proof)
	require.NoError(t, err)
	require.NotEmpty(t, data)

	// Unmarshal from JSON
	var decoded rest.SideTxProof
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	// Verify fields match
	require.Equal(t, proof.Tx, decoded.Tx)
	require.Equal(t, proof.Data, decoded.Data)
	require.Equal(t, len(proof.Sigs), len(decoded.Sigs))
	for i := range proof.Sigs {
		require.Equal(t, proof.Sigs[i], decoded.Sigs[i])
	}
}

func TestSideTxProofSignatureStructure(t *testing.T) {
	// Test that signatures have the correct structure [3]string
	proof := rest.SideTxProof{
		Sigs: [][3]string{
			{"part1", "part2", "part3"},
		},
		Tx:   "tx",
		Data: "data",
	}

	require.Len(t, proof.Sigs, 1)
	require.Len(t, proof.Sigs[0], 3)
	require.Equal(t, "part1", proof.Sigs[0][0])
	require.Equal(t, "part2", proof.Sigs[0][1])
	require.Equal(t, "part3", proof.Sigs[0][2])
}

func TestCommitTxProofFields(t *testing.T) {
	// Test that all fields can be set and retrieved
	proof := rest.CommitTxProof{}

	proof.Vote = "vote"
	proof.Sigs = "sigs"
	proof.Tx = "tx"
	proof.Proof = "proof"

	require.Equal(t, "vote", proof.Vote)
	require.Equal(t, "sigs", proof.Sigs)
	require.Equal(t, "tx", proof.Tx)
	require.Equal(t, "proof", proof.Proof)
}

func TestSideTxProofFields(t *testing.T) {
	// Test that all fields can be set and retrieved
	proof := rest.SideTxProof{}

	proof.Sigs = [][3]string{{"a", "b", "c"}}
	proof.Tx = "tx"
	proof.Data = "data"

	require.Len(t, proof.Sigs, 1)
	require.Equal(t, "tx", proof.Tx)
	require.Equal(t, "data", proof.Data)
}
