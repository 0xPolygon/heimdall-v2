package types

import (
	"bytes"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

// signWithRecoveryID signs digest with fresh keys until the signature carries the wanted recovery id.
func signWithRecoveryID(t *testing.T, digest []byte, id byte) (sig, signer []byte) {
	t.Helper()
	for range 64 {
		key, err := crypto.GenerateKey()
		require.NoError(t, err)
		got, err := crypto.Sign(digest, key)
		require.NoError(t, err)
		if got[crypto.RecoveryIDOffset] == id {
			return got, crypto.PubkeyToAddress(key.PublicKey).Bytes()
		}
	}
	t.Fatalf("no signature with recovery id %d", id)
	return nil, nil
}

func TestCanonicalSignature(t *testing.T) {
	digest := crypto.Keccak256([]byte{0x01}, []byte("checkpoint"))

	for _, recoveryID := range []byte{0, 1} {
		t.Run(fmt.Sprintf("recovery id %d", recoveryID), func(t *testing.T) {
			other, err := crypto.GenerateKey()
			require.NoError(t, err)
			genuine, signer := signWithRecoveryID(t, digest, recoveryID)

			withRecoveryID := func(id byte) []byte {
				sig := bytes.Clone(genuine)
				sig[crypto.RecoveryIDOffset] = id
				return sig
			}
			highS := bytes.Clone(genuine)
			new(big.Int).Sub(secp256k1N, new(big.Int).SetBytes(genuine[32:64])).FillBytes(highS[32:64])
			highS[crypto.RecoveryIDOffset] ^= 1

			for _, tc := range []struct {
				name   string
				digest []byte
				signer []byte
				sig    []byte
				ok     bool
			}{
				{"genuine", digest, signer, genuine, true},
				{"flipped recovery id", digest, signer, withRecoveryID(recoveryID ^ 1), true},
				{"27-offset recovery id", digest, signer, withRecoveryID(recoveryID + 27), true},
				{"out-of-range recovery id", digest, signer, withRecoveryID(255), true},
				{"high S", digest, signer, highS, true},
				{"other signer", digest, crypto.PubkeyToAddress(other.PublicKey).Bytes(), genuine, false},
				{"other digest", crypto.Keccak256([]byte("other")), signer, genuine, false},
				{"invalid signature", digest, signer, bytes.Repeat([]byte{0x42}, crypto.SignatureLength), false},
				{"short signature", digest, signer, genuine[:crypto.RecoveryIDOffset], false},
			} {
				t.Run(tc.name, func(t *testing.T) {
					sig, ok := CanonicalSignature(tc.digest, tc.signer, tc.sig)
					require.Equal(t, tc.ok, ok)
					if ok {
						require.Equal(t, genuine, sig)
					}
				})
			}

			t.Run("input is not modified", func(t *testing.T) {
				sig := bytes.Clone(highS)
				_, ok := CanonicalSignature(digest, signer, sig)
				require.True(t, ok)
				require.Equal(t, highS, sig)
			})
		})
	}

	t.Run("S of exactly N/2 is kept", func(t *testing.T) {
		key, err := crypto.GenerateKey()
		require.NoError(t, err)
		sig, err := crypto.Sign(digest, key)
		require.NoError(t, err)
		secp256k1HalfN.FillBytes(sig[32:64])
		pubKey, err := crypto.SigToPub(digest, sig)
		require.NoError(t, err)

		out, ok := CanonicalSignature(digest, crypto.PubkeyToAddress(*pubKey).Bytes(), sig)
		require.True(t, ok)
		require.Equal(t, sig, out)
	})
}
