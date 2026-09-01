package processor

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	checkpointtypes "github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
)

func sigWithRecoveryID(r, s byte, recoveryID byte) []byte {
	sig := make([]byte, crypto.SignatureLength)
	sig[31] = r
	sig[63] = s
	sig[crypto.RecoveryIDOffset] = recoveryID
	return sig
}

func TestNormalizeCheckpointV(t *testing.T) {
	tests := []struct {
		name       string
		recoveryID byte
		want       int64
		wantErr    bool
	}{
		{"canonical 0 maps to 27", 0, 27, false},
		{"canonical 1 maps to 28", 1, 28, false},
		{"already-normalized 27 stays 27", 27, 27, false},
		{"already-normalized 28 stays 28", 28, 28, false},
		{"poisoned 54 rejected", 54, 0, true},
		{"poisoned 55 rejected", 55, 0, true},
		{"garbage 2 rejected", 2, 0, true},
		{"garbage 255 rejected", 255, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := normalizeCheckpointV(tt.recoveryID)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, v.Int64())
		})
	}
}

func TestParseCheckpointSignatures(t *testing.T) {
	cp := &CheckpointProcessor{}

	t.Run("empty set errors", func(t *testing.T) {
		_, err := cp.parseCheckpointSignatures(nil)
		require.Error(t, err)
	})

	t.Run("canonical recovery bytes normalize to 27/28", func(t *testing.T) {
		sigs := []checkpointtypes.CheckpointSignature{
			{ValidatorAddress: []byte{0x01}, Signature: sigWithRecoveryID(0xaa, 0xbb, 0)},
			{ValidatorAddress: []byte{0x02}, Signature: sigWithRecoveryID(0xcc, 0xdd, 1)},
		}
		out, err := cp.parseCheckpointSignatures(sigs)
		require.NoError(t, err)
		require.Len(t, out, 2)
		require.Equal(t, big.NewInt(27), out[0][2])
		require.Equal(t, big.NewInt(28), out[1][2])
		require.Equal(t, int64(0xaa), out[0][0].Int64())
		require.Equal(t, int64(0xbb), out[0][1].Int64())
	})

	t.Run("recovery byte 28 yields 28 not 55", func(t *testing.T) {
		sigs := []checkpointtypes.CheckpointSignature{
			{ValidatorAddress: []byte{0x01}, Signature: sigWithRecoveryID(0x11, 0x22, 28)},
		}
		out, err := cp.parseCheckpointSignatures(sigs)
		require.NoError(t, err)
		require.Equal(t, big.NewInt(28), out[0][2])
	})

	t.Run("poisoned recovery byte 55 is rejected", func(t *testing.T) {
		sigs := []checkpointtypes.CheckpointSignature{
			{ValidatorAddress: []byte{0x01}, Signature: sigWithRecoveryID(0x11, 0x22, 55)},
		}
		_, err := cp.parseCheckpointSignatures(sigs)
		require.Error(t, err)
	})

	t.Run("wrong length signature errors without panic", func(t *testing.T) {
		sigs := []checkpointtypes.CheckpointSignature{
			{ValidatorAddress: []byte{0x01}, Signature: []byte{0x00, 0x01, 0x02}},
		}
		require.NotPanics(t, func() {
			_, err := cp.parseCheckpointSignatures(sigs)
			require.Error(t, err)
		})
	})

	t.Run("output is sorted by validator address", func(t *testing.T) {
		sigs := []checkpointtypes.CheckpointSignature{
			{ValidatorAddress: []byte{0x03}, Signature: sigWithRecoveryID(0x03, 0x03, 0)},
			{ValidatorAddress: []byte{0x01}, Signature: sigWithRecoveryID(0x01, 0x01, 1)},
			{ValidatorAddress: []byte{0x02}, Signature: sigWithRecoveryID(0x02, 0x02, 0)},
		}
		out, err := cp.parseCheckpointSignatures(sigs)
		require.NoError(t, err)
		require.Len(t, out, 3)
		require.Equal(t, int64(0x01), out[0][0].Int64())
		require.Equal(t, int64(0x02), out[1][0].Int64())
		require.Equal(t, int64(0x03), out[2][0].Int64())
	})
}
