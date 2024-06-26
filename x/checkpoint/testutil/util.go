package testutil

import (
	"crypto/rand"
)

func RandomBytes() []byte {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return b
}
