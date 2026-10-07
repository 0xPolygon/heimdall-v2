package hex

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

const MaxProofLength = 1024

const txHashStringLength = 2 + common.HashLength*2

// FormatHex returns a checksum hex string prefixed with 0x.
func FormatHex(data []byte) string {
	return "0x" + common.Bytes2Hex(data)
}

// FormatAddress makes sure the address is compliant with the heimdall-v2 format
func FormatAddress(hexAddr string) string {
	hexAddr = strings.TrimSpace(strings.ToLower(hexAddr))
	return "0x" + strings.TrimPrefix(hexAddr, "0x")
}

// IsTxHashNonEmpty returns true if the input is a non-empty string.
func IsTxHashNonEmpty(s string) bool {
	return strings.TrimSpace(s) != ""
}

// IsValidTxHash reports whether s is a 0x-prefixed, 32-byte hexadecimal hash.
func IsValidTxHash(s string) bool {
	if len(s) != txHashStringLength || !hasHexPrefix(s) {
		return false
	}

	for i := 2; i < len(s); i++ {
		if _, ok := hexNibble(s[i]); !ok {
			return false
		}
	}

	return true
}

// NormalizeTxHash returns the same 32-byte hash representation as common.HexToHash.
// It retains only the last 32 decoded bytes, avoiding an allocation proportional to
// the size of a legacy input string.
func NormalizeTxHash(s string) string {
	if hasHexPrefix(s) {
		s = s[2:]
	}

	var tail [common.HashLength]byte
	decoded := 0

	if len(s)%2 == 1 {
		value, ok := hexNibble(s[0])
		if !ok {
			return common.Hash{}.Hex()
		}
		tail[0] = value
		decoded = 1
		s = s[1:]
	}

	for i := 0; i < len(s); i += 2 {
		high, highOK := hexNibble(s[i])
		low, lowOK := hexNibble(s[i+1])
		if !highOK || !lowOK {
			break
		}
		tail[decoded%common.HashLength] = high<<4 | low
		decoded++
	}

	var hash common.Hash
	start := decoded % common.HashLength
	copied := copy(hash[:], tail[start:])
	copy(hash[copied:], tail[:start])
	return hash.Hex()
}

func hasHexPrefix(s string) bool {
	return strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X")
}

func hexNibble(char byte) (byte, bool) {
	switch {
	case char >= '0' && char <= '9':
		return char - '0', true
	case char >= 'a' && char <= 'f':
		return char - 'a' + 10, true
	case char >= 'A' && char <= 'F':
		return char - 'A' + 10, true
	default:
		return 0, false
	}
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
