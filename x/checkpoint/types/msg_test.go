package types

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnpackCheckpointSideSignBytes(t *testing.T) {
	msg := NewMsgCheckpointBlock("0x762893B6B6525C52Fa6B91C211Ee0D718561bF65", 0, 2, []byte("rootHash"), []byte("accountRootHash"), "1122")
	sideSignBytes := msg.GetSideSignBytes()
	assert.Equal(t, len(sideSignBytes), 192)
	_, err := UnpackCheckpointSideSignBytes(sideSignBytes)
	if err != nil {
		t.Fatalf("UnpackCheckpointSideSignBytes failed: %v", err)
	}
	bytes := new(big.Int).SetUint64(0).Bytes()
	assert.Equal(t, len(bytes), 8)
}
