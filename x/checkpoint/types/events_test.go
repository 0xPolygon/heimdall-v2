package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
)

func TestEventTypeCheckpoint(t *testing.T) {
	require.Equal(t, "checkpoint", types.EventTypeCheckpoint)
	require.NotEmpty(t, types.EventTypeCheckpoint)
}

func TestEventTypeCheckpointAck(t *testing.T) {
	require.Equal(t, "checkpoint-ack", types.EventTypeCheckpointAck)
	require.NotEmpty(t, types.EventTypeCheckpointAck)
}

func TestEventTypeCheckpointNoAck(t *testing.T) {
	require.Equal(t, "checkpoint-noack", types.EventTypeCheckpointNoAck)
	require.NotEmpty(t, types.EventTypeCheckpointNoAck)
}

func TestAttributeKeyProposer(t *testing.T) {
	require.Equal(t, "proposer", types.AttributeKeyProposer)
}

func TestAttributeKeyStartBlock(t *testing.T) {
	require.Equal(t, "start-block", types.AttributeKeyStartBlock)
}

func TestAttributeKeyEndBlock(t *testing.T) {
	require.Equal(t, "end-block", types.AttributeKeyEndBlock)
}

func TestAttributeKeyHeaderIndex(t *testing.T) {
	require.Equal(t, "header-index", types.AttributeKeyHeaderIndex)
}

func TestAttributeKeyNewProposer(t *testing.T) {
	require.Equal(t, "new-proposer", types.AttributeKeyNewProposer)
}

func TestAttributeKeyRootHash(t *testing.T) {
	require.Equal(t, "root-hash", types.AttributeKeyRootHash)
}

func TestAttributeKeyAccountHash(t *testing.T) {
	require.Equal(t, "account-hash", types.AttributeKeyAccountHash)
}

func TestAttributeValueCategory(t *testing.T) {
	require.Equal(t, types.ModuleName, types.AttributeValueCategory)
	require.Equal(t, "checkpoint", types.AttributeValueCategory)
}

func TestEventTypes_Uniqueness(t *testing.T) {
	eventTypes := []string{
		types.EventTypeCheckpoint,
		types.EventTypeCheckpointAck,
		types.EventTypeCheckpointNoAck,
	}

	for i := 0; i < len(eventTypes); i++ {
		for j := i + 1; j < len(eventTypes); j++ {
			require.NotEqual(t, eventTypes[i], eventTypes[j])
		}
	}
}

func TestAttributeKeys_Uniqueness(t *testing.T) {
	keys := []string{
		types.AttributeKeyProposer,
		types.AttributeKeyStartBlock,
		types.AttributeKeyEndBlock,
		types.AttributeKeyHeaderIndex,
		types.AttributeKeyNewProposer,
		types.AttributeKeyRootHash,
		types.AttributeKeyAccountHash,
	}

	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			require.NotEqual(t, keys[i], keys[j])
		}
	}
}

func TestEventTypes_ContainCheckpoint(t *testing.T) {
	require.Contains(t, types.EventTypeCheckpoint, "checkpoint")
	require.Contains(t, types.EventTypeCheckpointAck, "checkpoint")
	require.Contains(t, types.EventTypeCheckpointNoAck, "checkpoint")
}

func TestAckEvents_ContainAck(t *testing.T) {
	require.Contains(t, types.EventTypeCheckpointAck, "ack")
	require.Contains(t, types.EventTypeCheckpointNoAck, "ack")
}

func TestBlockAttributes(t *testing.T) {
	require.Contains(t, types.AttributeKeyStartBlock, "block")
	require.Contains(t, types.AttributeKeyEndBlock, "block")
}

func TestHashAttributes(t *testing.T) {
	require.Contains(t, types.AttributeKeyRootHash, "hash")
	require.Contains(t, types.AttributeKeyAccountHash, "hash")
}

func TestProposerAttributes(t *testing.T) {
	require.Contains(t, types.AttributeKeyProposer, "proposer")
	require.Contains(t, types.AttributeKeyNewProposer, "proposer")
}

func TestAllEventConstants_NotEmpty(t *testing.T) {
	constants := []string{
		types.EventTypeCheckpoint,
		types.EventTypeCheckpointAck,
		types.EventTypeCheckpointNoAck,
		types.AttributeKeyProposer,
		types.AttributeKeyStartBlock,
		types.AttributeKeyEndBlock,
		types.AttributeKeyHeaderIndex,
		types.AttributeKeyNewProposer,
		types.AttributeKeyRootHash,
		types.AttributeKeyAccountHash,
		types.AttributeValueCategory,
	}

	for _, constant := range constants {
		require.NotEmpty(t, constant)
	}
}

func TestEventConstants_LowercaseKebabCase(t *testing.T) {
	constants := []string{
		types.EventTypeCheckpoint,
		types.EventTypeCheckpointAck,
		types.EventTypeCheckpointNoAck,
		types.AttributeKeyProposer,
		types.AttributeKeyStartBlock,
		types.AttributeKeyEndBlock,
		types.AttributeKeyHeaderIndex,
		types.AttributeKeyNewProposer,
		types.AttributeKeyRootHash,
		types.AttributeKeyAccountHash,
	}

	for _, constant := range constants {
		// Should not contain uppercase
		for _, char := range constant {
			if char >= 'A' && char <= 'Z' {
				t.Errorf("Constant %s contains uppercase character", constant)
			}
		}
	}
}

func TestAttributeValueCategoryMatchesModule(t *testing.T) {
	require.Equal(t, types.ModuleName, types.AttributeValueCategory)
	require.Equal(t, types.StoreKey, types.AttributeValueCategory)
	require.Equal(t, types.RouterKey, types.AttributeValueCategory)
}
