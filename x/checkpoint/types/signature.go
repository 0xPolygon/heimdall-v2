package types

import (
	"bytes"
	"math/big"

	"github.com/ethereum/go-ethereum/crypto"
)

var (
	secp256k1N     = crypto.S256().Params().N
	secp256k1HalfN = new(big.Int).Rsh(secp256k1N, 1)
)

// CanonicalSignature returns sig in the form L1 checkpoint verification accepts: S at most N/2 and the
// recovery id (0 or 1) that recovers to signer from digest. The consensus signature check does not cover
// the recovery byte. Returns false if neither id recovers to signer.
func CanonicalSignature(digest, signer, sig []byte) ([]byte, bool) {
	if len(sig) != crypto.SignatureLength {
		return nil, false
	}

	canonical := bytes.Clone(sig)
	if s := new(big.Int).SetBytes(canonical[32:64]); s.Cmp(secp256k1HalfN) > 0 {
		new(big.Int).Sub(secp256k1N, s).FillBytes(canonical[32:64])
	}

	for recoveryID := byte(0); recoveryID <= 1; recoveryID++ {
		canonical[crypto.RecoveryIDOffset] = recoveryID
		pubKey, err := crypto.SigToPub(digest, canonical)
		if err == nil && bytes.Equal(crypto.PubkeyToAddress(*pubKey).Bytes(), signer) {
			return canonical, true
		}
	}

	return nil, false
}
