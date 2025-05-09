package hex

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

const MaxProofLength = 1024

// FormatAddress makes sure the address is compliant with the heimdall-v2 format
func FormatAddress(hexAddr string) string {
	hexAddr = strings.TrimSpace(strings.ToLower(hexAddr))
	return "0x" + strings.TrimPrefix(hexAddr, "0x")
}

// IsValidTxHash returns true if the input is a valid 32-byte Ethereum tx hash (0x-prefixed, 64 hex chars).
func IsValidTxHash(s string) bool {
	if !strings.HasPrefix(s, "0x") || len(s) != 66 {
		return false
	}
	_, err := hex.DecodeString(s[2:])
	return err == nil
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
