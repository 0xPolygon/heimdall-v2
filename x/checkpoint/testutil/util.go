package testutil

import (
	"crypto/rand"
)

func RandomBytes() []byte {
	b := make([]byte, 32)
	rand.Read(b)
	return b
}
