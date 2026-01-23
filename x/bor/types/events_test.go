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

func TestEventTypeBackfillSpans(t *testing.T) {
	require.Equal(t, "backfill-spans", types.EventTypeBackfillSpans)
	require.NotEmpty(t, types.EventTypeBackfillSpans)
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

func TestAttributesKeyLatestSpanId(t *testing.T) {
	require.Equal(t, "latest-span-id", types.AttributesKeyLatestSpanId)
	require.NotEmpty(t, types.AttributesKeyLatestSpanId)
}

func TestAttributesKeyLatestBorSpanId(t *testing.T) {
	require.Equal(t, "latest-bor-span-id", types.AttributesKeyLatestBorSpanId)
	require.NotEmpty(t, types.AttributesKeyLatestBorSpanId)
}

func TestAttributeValueCategory(t *testing.T) {
	require.Equal(t, types.ModuleName, types.AttributeValueCategory)
	require.Equal(t, "bor", types.AttributeValueCategory)
}

func TestEventTypesUniqueness(t *testing.T) {
	// Test that event types are unique
	eventTypes := []string{
		types.EventTypeProposeSpan,
		types.EventTypeBackfillSpans,
	}

	require.Len(t, eventTypes, 2)
	require.NotEqual(t, eventTypes[0], eventTypes[1])
}

func TestAttributeKeysUniqueness(t *testing.T) {
	// Test that all attribute keys are unique
	attributeKeys := []string{
		types.AttributeKeySpanID,
		types.AttributeKeySpanStartBlock,
		types.AttributeKeySpanEndBlock,
		types.AttributesKeyLatestSpanId,
		types.AttributesKeyLatestBorSpanId,
	}

	// Check all keys are distinct
	for i := 0; i < len(attributeKeys); i++ {
		for j := i + 1; j < len(attributeKeys); j++ {
			require.NotEqual(t, attributeKeys[i], attributeKeys[j],
				"Attribute keys at positions %d and %d should be different", i, j)
		}
	}
}

func TestEventTypesFormat(t *testing.T) {
	// Test that event types follow the kebab-case convention
	eventTypes := []string{
		types.EventTypeProposeSpan,
		types.EventTypeBackfillSpans,
	}

	for _, eventType := range eventTypes {
		require.Contains(t, eventType, "-",
			"Event type should use kebab-case: %s", eventType)
	}
}

func TestAttributeKeysFormat(t *testing.T) {
	// Test that attribute keys follow kebab-case convention
	attributeKeys := []string{
		types.AttributeKeySpanID,
		types.AttributeKeySpanStartBlock,
		types.AttributeKeySpanEndBlock,
		types.AttributesKeyLatestSpanId,
		types.AttributesKeyLatestBorSpanId,
	}

	for _, key := range attributeKeys {
		require.Contains(t, key, "-",
			"Attribute key should use kebab-case: %s", key)
	}
}

func TestEventTypesContainSpan(t *testing.T) {
	// Test that span-related events contain "span" in their name
	require.Contains(t, types.EventTypeProposeSpan, "span")
	require.Contains(t, types.EventTypeBackfillSpans, "span")
}

func TestSpanAttributesContainSpan(t *testing.T) {
	// Test that span-id attributes contain "span" in their name
	spanAttributes := []string{
		types.AttributeKeySpanID,
		types.AttributesKeyLatestSpanId,
		types.AttributesKeyLatestBorSpanId,
	}

	for _, attr := range spanAttributes {
		require.Contains(t, attr, "span",
			"Span attribute should contain 'span': %s", attr)
	}
}

func TestBlockAttributes(t *testing.T) {
	// Test block-related attributes
	require.Contains(t, types.AttributeKeySpanStartBlock, "block")
	require.Contains(t, types.AttributeKeySpanEndBlock, "block")
}

func TestLatestAttributes(t *testing.T) {
	// Test latest-related attributes
	latestAttributes := []string{
		types.AttributesKeyLatestSpanId,
		types.AttributesKeyLatestBorSpanId,
	}

	for _, attr := range latestAttributes {
		require.Contains(t, attr, "latest",
			"Latest attribute should contain 'latest': %s", attr)
	}
}

func TestBorAttributes(t *testing.T) {
	// Test bor-specific attributes
	require.Contains(t, types.AttributesKeyLatestBorSpanId, "bor")
}

func TestProposeSpanEventType(t *testing.T) {
	require.Contains(t, types.EventTypeProposeSpan, "propose")
	require.Contains(t, types.EventTypeProposeSpan, "span")
}

func TestBackfillSpansEventType(t *testing.T) {
	require.Contains(t, types.EventTypeBackfillSpans, "backfill")
	require.Contains(t, types.EventTypeBackfillSpans, "spans")
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
		types.EventTypeBackfillSpans,
		types.AttributeKeySpanID,
		types.AttributeKeySpanStartBlock,
		types.AttributeKeySpanEndBlock,
		types.AttributesKeyLatestSpanId,
		types.AttributesKeyLatestBorSpanId,
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
		types.EventTypeBackfillSpans,
		types.AttributeKeySpanID,
		types.AttributeKeySpanStartBlock,
		types.AttributeKeySpanEndBlock,
		types.AttributesKeyLatestSpanId,
		types.AttributesKeyLatestBorSpanId,
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
