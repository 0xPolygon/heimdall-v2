package processor

import (
	"bytes"
	"math/big"
	"slices"
	"testing"

	"cosmossdk.io/log"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	checkpointtypes "github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
)

var testSignedData = []byte("checkpoint side sign bytes")

// signCheckpoint signs testSignedData with a fresh key.
func signCheckpoint(t *testing.T) checkpointtypes.CheckpointSignature {
	t.Helper()
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	sig, err := crypto.Sign(crypto.Keccak256([]byte{0x01}, testSignedData), key)
	require.NoError(t, err)
	return checkpointtypes.CheckpointSignature{
		ValidatorAddress: crypto.PubkeyToAddress(key.PublicKey).Bytes(),
		Signature:        sig,
	}
}

func TestParseCheckpointSignatures(t *testing.T) {
	var logs bytes.Buffer
	cp := &CheckpointProcessor{}
	cp.BaseProcessor.Logger = log.NewLogger(&logs, log.ColorOption(false))

	t.Run("empty set errors", func(t *testing.T) {
		_, err := cp.parseCheckpointSignatures(nil, testSignedData)
		require.Error(t, err)
	})

	t.Run("sorted by signer with recovery ids as 27/28", func(t *testing.T) {
		sigs := make([]checkpointtypes.CheckpointSignature, 8)
		for i := range sigs {
			sigs[i] = signCheckpoint(t)
		}

		logs.Reset()
		out, err := cp.parseCheckpointSignatures(sigs, testSignedData)
		require.NoError(t, err)
		require.Empty(t, logs.String())

		slices.SortFunc(sigs, func(a, b checkpointtypes.CheckpointSignature) int {
			return bytes.Compare(a.ValidatorAddress, b.ValidatorAddress)
		})
		require.Len(t, out, len(sigs))
		for i, sig := range sigs {
			require.Equal(t, new(big.Int).SetBytes(sig.Signature[:32]), out[i][0])
			require.Equal(t, big.NewInt(int64(sig.Signature[crypto.RecoveryIDOffset])+27), out[i][2])
		}
	})

	t.Run("high S is submitted in the low-S form L1 accepts", func(t *testing.T) {
		genuine := signCheckpoint(t)
		highS := genuine
		highS.Signature = bytes.Clone(genuine.Signature)
		n := crypto.S256().Params().N
		new(big.Int).Sub(n, new(big.Int).SetBytes(genuine.Signature[32:64])).FillBytes(highS.Signature[32:64])
		highS.Signature[crypto.RecoveryIDOffset] ^= 1

		out, err := cp.parseCheckpointSignatures([]checkpointtypes.CheckpointSignature{highS}, testSignedData)
		require.NoError(t, err)
		require.Len(t, out, 1)
		require.Equal(t, new(big.Int).SetBytes(genuine.Signature[32:64]), out[0][1])
		require.Equal(t, big.NewInt(int64(genuine.Signature[crypto.RecoveryIDOffset])+27), out[0][2])
	})

	t.Run("signatures that recover to no validator are dropped", func(t *testing.T) {
		good := signCheckpoint(t)

		invalid := signCheckpoint(t)
		invalid.Signature = bytes.Repeat([]byte{0x42}, crypto.SignatureLength)

		wrongSigner := signCheckpoint(t)
		wrongSigner.ValidatorAddress = signCheckpoint(t).ValidatorAddress

		short := signCheckpoint(t)
		short.Signature = short.Signature[:crypto.RecoveryIDOffset]

		logs.Reset()
		out, err := cp.parseCheckpointSignatures([]checkpointtypes.CheckpointSignature{invalid, good, wrongSigner, short}, testSignedData)
		require.NoError(t, err)
		require.Len(t, out, 1)
		require.Equal(t, new(big.Int).SetBytes(good.Signature[:32]), out[0][0])
		require.Contains(t, logs.String(), "dropped checkpoint signatures")
		require.Contains(t, logs.String(), "dropped=3 total=4")
	})

	t.Run("errors when no signature recovers", func(t *testing.T) {
		_, err := cp.parseCheckpointSignatures([]checkpointtypes.CheckpointSignature{signCheckpoint(t)}, []byte("other checkpoint"))
		require.Error(t, err)
	})
}
