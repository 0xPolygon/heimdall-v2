package address

import "strings"

// FormatAddress makes sure the address is compliant with the heimdall-v2 format
func FormatAddress(hexAddr string) string {
	hexAddr = strings.ToLower(hexAddr)

	return "0x" + strings.TrimPrefix(hexAddr, "0x")
}
