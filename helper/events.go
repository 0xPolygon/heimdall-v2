package helper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// FindEvents find out the particular event based on the given function
func FindEvents(events []sdk.StringEvent, fn func(sdk.StringEvent) bool) *sdk.StringEvent {
	for _, event := range events {
		if fn(event) {
			return &event
		}
	}

	return nil
}

// FilterAttributes filter attributes by fn
func FilterAttributes(attributes []sdk.Attribute, fn func(sdk.Attribute) bool) *sdk.Attribute {
	for _, attribute := range attributes {
		if fn(attribute) {
			return &attribute
		}
	}

	return nil
}
