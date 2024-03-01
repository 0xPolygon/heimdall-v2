package module

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Configurator provides the hooks to allow modules to configure and register
// their services in the RegisterServices method. It is designed to eventually
// support module object capabilities isolation as described in
// https://github.com/cosmos/cosmos-sdk/issues/7093
type SideTxConfigurator interface {
	RegisterSideHandler(msgURL string, handler SideTxHandler) error

	RegisterPostHandler(msgURL string, handler PostTxHandler) error

	SideHandler(msg sdk.Msg) SideTxHandler

	PostHandler(msg sdk.Msg) PostTxHandler

	SideHandlerByTypeURL(typeURL string) SideTxHandler

	PostHandlerByTypeURL(typeURL string) PostTxHandler
}

type sideTxConfigurator struct {
	//sideHandlers to register sideHandler against the msgURl string value
	sideHandlers map[string]SideTxHandler

	//psotHandlers to register psotHandler against the msgURl string value
	postHandlers map[string]PostTxHandler
}

// NewConfigurator returns a new Configurator instance
func NewConfigurator() SideTxConfigurator {
	return &sideTxConfigurator{
		sideHandlers: make(map[string]SideTxHandler),
		postHandlers: make(map[string]PostTxHandler),
	}
}

// RegisterMigration implements the Configurator.RegisterMigration method
func (c *sideTxConfigurator) RegisterSideHandler(msgURL string, handler SideTxHandler) error {
	if c.sideHandlers[msgURL] == nil {
		c.sideHandlers[msgURL] = handler
		return nil
	}

	return fmt.Errorf("SideHandler corresponding to the following msg %s already exist", msgURL)
}

// RegisterMigration implements the Configurator.RegisterMigration method
func (c *sideTxConfigurator) RegisterPostHandler(msgURL string, handler PostTxHandler) error {

	if c.postHandlers[msgURL] == nil {
		c.postHandlers[msgURL] = handler
		return nil
	}

	return fmt.Errorf("PostHandler corresponding to the following msg %s already exist", msgURL)
}

// Handler returns the MsgServiceHandler for a given msg or nil if not found.
func (c *sideTxConfigurator) SideHandler(msg sdk.Msg) SideTxHandler {
	return c.sideHandlers[sdk.MsgTypeURL(msg)]
}

// Handler returns the MsgServiceHandler for a given msg or nil if not found.
func (c *sideTxConfigurator) PostHandler(msg sdk.Msg) PostTxHandler {
	return c.postHandlers[sdk.MsgTypeURL(msg)]
}

// SideHandlerByTypeURL returns the SideTxHandler for a given query route path or nil
// if not found.
func (c *sideTxConfigurator) SideHandlerByTypeURL(typeURL string) SideTxHandler {
	return c.sideHandlers[typeURL]
}

// PostHandlerByTypeURL returns the PostTxHandler for a given query route path or nil
// if not found.
func (c *sideTxConfigurator) PostHandlerByTypeURL(typeURL string) PostTxHandler {
	return c.postHandlers[typeURL]
}

// HasServices is the interface for modules to register services.
type HasSideMsgServices interface {
	// RegisterServices allows a module to register services.
	RegisterSideMsgServices(SideTxConfigurator)
}
