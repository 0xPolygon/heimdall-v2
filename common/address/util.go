package address

import "strings"

// FormatAddress makes sure the address is compliant with the heimdall-v2 format
func FormatAddress(hexAddr string) string {
	hexAddr = strings.ToLower(hexAddr)

	if !has0xPrefix(hexAddr) {
		hexAddr = "0x" + hexAddr
	}
	return hexAddr
}

// has0xPrefix validates str begins with '0x' or '0X'.
func has0xPrefix(str string) bool {
	return len(str) >= 2 && str[0] == '0' && (str[1] == 'x' || str[1] == 'X')
}
