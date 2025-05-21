package hex

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

const MaxProofLength = 1024

// FormatAddress normalizes a hexadecimal Ethereum address string.
// It trims whitespaces, and returns the checksummed (EIP-55) version of the address
func FormatAddress(hexAddr string) string {
	hexAddr = strings.TrimSpace(hexAddr)
	return common.HexToAddress(hexAddr).Hex()
}

// IsTxHashNonEmpty returns true if the input is a non-empty string.
func IsTxHashNonEmpty(s string) bool {
	return strings.TrimSpace(s) != ""
}

// ValidateProof checks if the proof is a valid hex string representing N 32-byte chunks, and not too long.
func ValidateProof(proof string) error {
	proofBytes := common.FromHex(proof)
	if len(proofBytes) == 0 {
		return errors.New("proof is empty or invalid hex")
	}
	if len(proofBytes)%32 != 0 {
		return errors.New("proof length must be a multiple of 32 bytes")
	}
	if len(proofBytes) > MaxProofLength {
		return fmt.Errorf("proof exceeds maximum allowed size of %d bytes", MaxProofLength)
	}
	return nil
}
