package module

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SideTxConfigurator provides the hooks to allow modules to configure and register
// their sideMsg services in the RegisterSideHandler and RegisterPostHandler method.
type SideTxConfigurator interface {
	RegisterSideHandler(msgURL string, handler SideTxHandler) error

	RegisterPostHandler(msgURL string, handler PostTxHandler) error

	SideHandler(msg sdk.Msg) SideTxHandler

	PostHandler(msg sdk.Msg) PostTxHandler
}

type sideTxConfigurator struct {
	// sideHandlers to register sideHandler against the msgURl string value
	sideHandlers map[string]SideTxHandler

	// postHandlers to register postHandler against the msgURl string value
	postHandlers map[string]PostTxHandler
}

// NewSideTxConfigurator returns a new Configurator instance
func NewSideTxConfigurator() SideTxConfigurator {
	return &sideTxConfigurator{
		sideHandlers: make(map[string]SideTxHandler),
		postHandlers: make(map[string]PostTxHandler),
	}
}

// RegisterSideHandler implements the SideTxConfigurator.RegisterSideHandler method
func (c *sideTxConfigurator) RegisterSideHandler(msgURL string, handler SideTxHandler) error {
	if c.sideHandlers[msgURL] == nil {
		c.sideHandlers[msgURL] = handler
		return nil
	}

	return fmt.Errorf("SideHandler corresponding to the following msg %s already exist", msgURL)
}

// RegisterPostHandler implements the SideTxConfigurator.RegisterPostHandler method
func (c *sideTxConfigurator) RegisterPostHandler(msgURL string, handler PostTxHandler) error {

	if c.postHandlers[msgURL] == nil {
		c.postHandlers[msgURL] = handler
		return nil
	}

	return fmt.Errorf("PostHandler corresponding to the following msg %s already exist", msgURL)
}

// SideHandler returns sideHandler for a given msg or nil if not found.
func (c *sideTxConfigurator) SideHandler(msg sdk.Msg) SideTxHandler {
	return c.sideHandlers[sdk.MsgTypeURL(msg)]
}

// PostHandler returns postHandler for a given msg or nil if not found.
func (c *sideTxConfigurator) PostHandler(msg sdk.Msg) PostTxHandler {
	return c.postHandlers[sdk.MsgTypeURL(msg)]
}

// HasSideMsgServices is the interface for modules to register sideTx services.
type HasSideMsgServices interface {
	// RegisterServices allows a module to register services.
	RegisterSideMsgServices(SideTxConfigurator)
}
