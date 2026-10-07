package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/x/bor/types"
)

func TestEventTypeProposeSpan(t *testing.T) {
	require.Equal(t, "propose-span", types.EventTypeProposeSpan)
	require.NotEmpty(t, types.EventTypeProposeSpan)
}

func TestAttributeKeySpanID(t *testing.T) {
	require.Equal(t, "span-id", types.AttributeKeySpanID)
	require.NotEmpty(t, types.AttributeKeySpanID)
}

func TestAttributeKeySpanStartBlock(t *testing.T) {
	require.Equal(t, "start-block", types.AttributeKeySpanStartBlock)
	require.NotEmpty(t, types.AttributeKeySpanStartBlock)
}

func TestAttributeKeySpanEndBlock(t *testing.T) {
	require.Equal(t, "end-block", types.AttributeKeySpanEndBlock)
	require.NotEmpty(t, types.AttributeKeySpanEndBlock)
}

func TestAttributeValueCategory(t *testing.T) {
	require.Equal(t, types.ModuleName, types.AttributeValueCategory)
	require.Equal(t, "bor", types.AttributeValueCategory)
}

func TestAttributeKeysUniqueness(t *testing.T) {
	// Test that all attribute keys are unique
	attributeKeys := []string{
		types.AttributeKeySpanID,
		types.AttributeKeySpanStartBlock,
		types.AttributeKeySpanEndBlock,
	}

	// Check all keys are distinct
	for i := 0; i < len(attributeKeys); i++ {
		for j := i + 1; j < len(attributeKeys); j++ {
			require.NotEqual(t, attributeKeys[i], attributeKeys[j],
				"Attribute keys at positions %d and %d should be different", i, j)
		}
	}
}

func TestAttributeKeysFormat(t *testing.T) {
	// Test that attribute keys follow kebab-case convention
	attributeKeys := []string{
		types.AttributeKeySpanID,
		types.AttributeKeySpanStartBlock,
		types.AttributeKeySpanEndBlock,
	}

	for _, key := range attributeKeys {
		require.Contains(t, key, "-",
			"Attribute key should use kebab-case: %s", key)
	}
}

func TestEventTypesContainSpan(t *testing.T) {
	require.Contains(t, types.EventTypeProposeSpan, "span")
}

func TestSpanAttributesContainSpan(t *testing.T) {
	require.Contains(t, types.AttributeKeySpanID, "span")
}

func TestBlockAttributes(t *testing.T) {
	// Test block-related attributes
	require.Contains(t, types.AttributeKeySpanStartBlock, "block")
	require.Contains(t, types.AttributeKeySpanEndBlock, "block")
}

func TestProposeSpanEventType(t *testing.T) {
	require.Contains(t, types.EventTypeProposeSpan, "propose")
	require.Contains(t, types.EventTypeProposeSpan, "span")
}

func TestSpanIDAttribute(t *testing.T) {
	require.Contains(t, types.AttributeKeySpanID, "span")
	require.Contains(t, types.AttributeKeySpanID, "id")
}

func TestStartBlockAttribute(t *testing.T) {
	require.Contains(t, types.AttributeKeySpanStartBlock, "start")
	require.Contains(t, types.AttributeKeySpanStartBlock, "block")
}

func TestEndBlockAttribute(t *testing.T) {
	require.Contains(t, types.AttributeKeySpanEndBlock, "end")
	require.Contains(t, types.AttributeKeySpanEndBlock, "block")
}

func TestAllEventConstantsNotEmpty(t *testing.T) {
	// Test that all event-related constants are not empty
	constants := []string{
		types.EventTypeProposeSpan,
		types.AttributeKeySpanID,
		types.AttributeKeySpanStartBlock,
		types.AttributeKeySpanEndBlock,
		types.AttributeValueCategory,
	}

	for _, constant := range constants {
		require.NotEmpty(t, constant)
	}
}

func TestAttributeValueCategoryMatchesModule(t *testing.T) {
	// Test that attribute value category matches module name
	require.Equal(t, types.ModuleName, types.AttributeValueCategory)
	require.Equal(t, types.StoreKey, types.AttributeValueCategory)
	require.Equal(t, types.RouterKey, types.AttributeValueCategory)
}

func TestEventConstantsAreLowercase(t *testing.T) {
	// Test that constants follow the lowercase-kebab-case convention
	constants := []string{
		types.EventTypeProposeSpan,
		types.AttributeKeySpanID,
		types.AttributeKeySpanStartBlock,
		types.AttributeKeySpanEndBlock,
		types.AttributeValueCategory,
	}

	for _, constant := range constants {
		// Should not contain uppercase letters
		for _, char := range constant {
			if char >= 'A' && char <= 'Z' {
				t.Errorf("Constant %s contains uppercase character", constant)
			}
		}
	}
}
