package listener

import (
	common "github.com/cometbft/cometbft/libs/service"
	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	"github.com/cosmos/cosmos-sdk/codec"

	"github.com/0xPolygon/heimdall-v2/bridge/queue"
	"github.com/0xPolygon/heimdall-v2/bridge/util"
	"github.com/0xPolygon/heimdall-v2/helper"
)

const (
	listenerServiceStr = "listener"

	rootChainListenerStr = "rootchain"
	heimdallListenerStr  = "heimdall"
	borChainListenerStr  = "borchain"
)

// ListenerService starts and stops all chain event listeners
type ListenerService struct {
	// Base service
	common.BaseService
	listeners []Listener
}

// NewListenerService returns new service object for listneing to events
func NewListenerService(cdc codec.Codec, queueConnector *queue.QueueConnector, httpClient *rpchttp.HTTP) *ListenerService {
	// creating listener object
	listenerService := &ListenerService{}

	listenerService.BaseService = *common.NewBaseService(util.Logger().With("service", "listener"), listenerServiceStr, listenerService)

	rootchainListener := NewRootChainListener()
	rootchainListener.BaseListener = *NewBaseListener(cdc, queueConnector, httpClient, helper.GetMainClient(), rootChainListenerStr, rootchainListener)
	listenerService.listeners = append(listenerService.listeners, rootchainListener)

	borchainListener := &BorChainListener{}
	borchainListener.BaseListener = *NewBaseListener(cdc, queueConnector, httpClient, helper.GetBorClient(), borChainListenerStr, borchainListener)
	listenerService.listeners = append(listenerService.listeners, borchainListener)

	heimdallListener := &HeimdallListener{}
	heimdallListener.BaseListener = *NewBaseListener(cdc, queueConnector, httpClient, nil, heimdallListenerStr, heimdallListener)
	listenerService.listeners = append(listenerService.listeners, heimdallListener)

	return listenerService
}

// OnStart starts new block subscription
func (listenerService *ListenerService) OnStart() error {
	if err := listenerService.BaseService.OnStart(); err != nil {
		listenerService.Logger.Error("OnStart | OnStart", "Error", err)
	} // Always call the overridden method.

	// start chain listeners
	for _, listener := range listenerService.listeners {
		if err := listener.Start(); err != nil {
			listenerService.Logger.Error("OnStart | Start", "Error", err)
		}
	}

	listenerService.Logger.Info("all listeners Started")

	return nil
}

// OnStop stops all necessary go routines
func (listenerService *ListenerService) OnStop() {
	listenerService.BaseService.OnStop() // Always call the overridden method.

	// start chain listeners
	for _, listener := range listenerService.listeners {
		listener.Stop()
	}

	listenerService.Logger.Info("all listeners stopped")
}
