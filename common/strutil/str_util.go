package strutil

import (
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

func HashesToString(hashes [][]byte) string {
	hashStrs := make([]string, 0, len(hashes))
	for _, hash := range hashes {
		hashStrs = append(hashStrs, "0x"+common.Bytes2Hex(hash))
	}
	return strings.Join(hashStrs, " ")
}
