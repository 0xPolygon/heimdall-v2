package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewMsgCheckpointBlock(t *testing.T) {
	checkpointMsg := NewMsgCheckpointBlock("0xd07Dd60077D3a5628837Ada6002eA8Ac5E689795", 1, 2, []byte("d07Dd60077D3795"), []byte("d07Dd60077D39795"), "1")
	sideSignBytes := checkpointMsg.GetSideSignBytes()
	require.Len(t, sideSignBytes, 198)
}
