package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/x/clerk/types"
)

func TestEventTypeRecord(t *testing.T) {
	require.Equal(t, "record", types.EventTypeRecord)
}

func TestAttributeKeyRecordTxHash(t *testing.T) {
	require.Equal(t, "record-tx-hash", types.AttributeKeyRecordTxHash)
}

func TestAttributeKeyRecordTxLogIndex(t *testing.T) {
	require.Equal(t, "record-tx-log-index", types.AttributeKeyRecordTxLogIndex)
}

func TestAttributeKeyRecordID(t *testing.T) {
	require.Equal(t, "record-id", types.AttributeKeyRecordID)
}

func TestAttributeKeyRecordContract(t *testing.T) {
	require.Equal(t, "record-contract", types.AttributeKeyRecordContract)
}

func TestAttributeValueCategory(t *testing.T) {
	require.Equal(t, types.ModuleName, types.AttributeValueCategory)
}

func TestAllAttributeKeys_ContainRecord(t *testing.T) {
	require.Contains(t, types.AttributeKeyRecordTxHash, "record")
	require.Contains(t, types.AttributeKeyRecordTxLogIndex, "record")
	require.Contains(t, types.AttributeKeyRecordID, "record")
	require.Contains(t, types.AttributeKeyRecordContract, "record")
}

func TestAttributeKeys_Uniqueness(t *testing.T) {
	keys := []string{
		types.AttributeKeyRecordTxHash,
		types.AttributeKeyRecordTxLogIndex,
		types.AttributeKeyRecordID,
		types.AttributeKeyRecordContract,
	}

	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			require.NotEqual(t, keys[i], keys[j])
		}
	}
}

func TestAllEventConstants_NotEmpty(t *testing.T) {
	require.NotEmpty(t, types.EventTypeRecord)
	require.NotEmpty(t, types.AttributeKeyRecordTxHash)
	require.NotEmpty(t, types.AttributeKeyRecordTxLogIndex)
	require.NotEmpty(t, types.AttributeKeyRecordID)
	require.NotEmpty(t, types.AttributeKeyRecordContract)
	require.NotEmpty(t, types.AttributeValueCategory)
}
