package module

import (
	"fmt"

	"github.com/0xPolygon/heimdall-v2/x/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Configurator provides the hooks to allow modules to configure and register
// their services in the RegisterServices method. It is designed to eventually
// support module object capabilities isolation as described in
// https://github.com/cosmos/cosmos-sdk/issues/7093
type SideTxConfigurator interface {
	RegisterSideHandler(msgURL string, handler types.SideTxHandler) error

	RegisterPostHandler(msgURL string, handler types.PostTxHandler) error

	SideHandler(msg sdk.Msg) types.SideTxHandler

	PostHandler(msg sdk.Msg) types.PostTxHandler

	SideHandlerByTypeURL(typeURL string) types.SideTxHandler

	PostHandlerByTypeURL(typeURL string) types.PostTxHandler
}

type sideTxConfigurator struct {
	//sideHandlers to register sideHandler against the msgURl string value
	sideHandlers map[string]types.SideTxHandler

	//psotHandlers to register psotHandler against the msgURl string value
	postHandlers map[string]types.PostTxHandler
}

// NewConfigurator returns a new Configurator instance
func NewConfigurator() SideTxConfigurator {
	return &sideTxConfigurator{
		sideHandlers: make(map[string]types.SideTxHandler),
		postHandlers: make(map[string]types.PostTxHandler),
	}
}

// RegisterMigration implements the Configurator.RegisterMigration method
func (c *sideTxConfigurator) RegisterSideHandler(msgURL string, handler types.SideTxHandler) error {
	if c.sideHandlers[msgURL] == nil {
		c.sideHandlers[msgURL] = handler
		return nil
	}

	return fmt.Errorf("SideHandler corresponding to the following msg %s already exist", msgURL)
}

// RegisterMigration implements the Configurator.RegisterMigration method
func (c *sideTxConfigurator) RegisterPostHandler(msgURL string, handler types.PostTxHandler) error {

	if c.postHandlers[msgURL] == nil {
		c.postHandlers[msgURL] = handler
		return nil
	}

	return fmt.Errorf("PostHandler corresponding to the following msg %s already exist", msgURL)
}

// Handler returns the MsgServiceHandler for a given msg or nil if not found.
func (c *sideTxConfigurator) SideHandler(msg sdk.Msg) types.SideTxHandler {
	return c.sideHandlers[sdk.MsgTypeURL(msg)]
}

// Handler returns the MsgServiceHandler for a given msg or nil if not found.
func (c *sideTxConfigurator) PostHandler(msg sdk.Msg) types.PostTxHandler {
	return c.postHandlers[sdk.MsgTypeURL(msg)]
}

// SideHandlerByTypeURL returns the SideTxHandler for a given query route path or nil
// if not found.
func (c *sideTxConfigurator) SideHandlerByTypeURL(typeURL string) types.SideTxHandler {
	return c.sideHandlers[typeURL]
}

// PostHandlerByTypeURL returns the PostTxHandler for a given query route path or nil
// if not found.
func (c *sideTxConfigurator) PostHandlerByTypeURL(typeURL string) types.PostTxHandler {
	return c.postHandlers[typeURL]
}
