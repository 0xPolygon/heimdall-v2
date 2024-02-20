package module

import (
	"fmt"

	"github.com/cosmos/gogoproto/grpc"

	"github.com/0xPolygon/heimdall-v2/x/types"
	"github.com/cosmos/cosmos-sdk/codec"
)

// Configurator provides the hooks to allow modules to configure and register
// their services in the RegisterServices method. It is designed to eventually
// support module object capabilities isolation as described in
// https://github.com/cosmos/cosmos-sdk/issues/7093
type SideTxConfigurator interface {
	RegisterSideHandler(msgURL string, handler types.SideTxHandler) error

	RegisterPostHandler(msgURL string, handler types.PostTxHandler) error
}

type sideTxConfigurator struct {
	//sideHandlers to register sideHandler against the msgURl string value
	sideHandlers map[string]types.SideTxHandler

	//psotHandlers to register psotHandler against the msgURl string value
	postHandlers map[string]types.PostTxHandler
}

// NewConfigurator returns a new Configurator instance
func NewConfigurator(cdc codec.Codec, msgServer, queryServer grpc.Server) SideTxConfigurator {
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
