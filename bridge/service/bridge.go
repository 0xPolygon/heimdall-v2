package service

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"cosmossdk.io/x/tx/signing"
	common "github.com/cometbft/cometbft/libs/service"
	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	authTypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"

	"github.com/0xPolygon/heimdall-v2/app"
	"github.com/0xPolygon/heimdall-v2/bridge/broadcaster"
	"github.com/0xPolygon/heimdall-v2/bridge/listener"
	"github.com/0xPolygon/heimdall-v2/bridge/processor"
	"github.com/0xPolygon/heimdall-v2/bridge/queue"
	"github.com/0xPolygon/heimdall-v2/bridge/util"
	"github.com/0xPolygon/heimdall-v2/helper"
	checkpointTypes "github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
	clerkTypes "github.com/0xPolygon/heimdall-v2/x/clerk/types"
	milestoneTypes "github.com/0xPolygon/heimdall-v2/x/milestone/types"
	stakeTypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
	topupTypes "github.com/0xPolygon/heimdall-v2/x/topup/types"
)

const (
	waitDuration  = 1 * time.Minute
	borChainIDKey = "bor-chain-id"
	logsTypeKey   = "logs-type"
)

var logger = helper.Logger.With("module", "bridge/service/")

// AdjustDBValue sets/normalizes viper-config for bridge runtime based on flags present on root/start cmd
func AdjustDBValue(cmd *cobra.Command) {
	cometBftNode, _ := cmd.Flags().GetString(helper.CometBFTNodeFlag)
	withHeimdallConfigValue, _ := cmd.Flags().GetString(helper.WithHeimdallConfigFlag)
	bridgeDBValue, _ := cmd.Flags().GetString(util.BridgeDBFlag)
	borChainIDValue, _ := cmd.Flags().GetString(borChainIDKey)
	logsTypeValue, _ := cmd.Flags().GetString(logsTypeKey)

	// default bridge storage dir: <home>/bridge/storage
	if bridgeDBValue == "" {
		home := viper.GetString(flags.FlagHome)
		if home == "" {
			home = app.DefaultNodeHome
		}
		bridgeDBValue = filepath.Join(home, "bridge", "storage")
	}

	// set to viper
	viper.Set(helper.CometBFTNodeFlag, cometBftNode)
	viper.Set(helper.WithHeimdallConfigFlag, withHeimdallConfigValue)
	viper.Set(util.BridgeDBFlag, bridgeDBValue)
	if borChainIDValue != "" {
		viper.Set(borChainIDKey, borChainIDValue)
	}
	if logsTypeValue != "" {
		viper.Set(logsTypeKey, logsTypeValue)
	}
}

// StartWithCtx starts the bridge runtime as a side service of heimdalld and shuts down gracefully.
func StartWithCtx(ctx context.Context, clientCtx client.Context) error {
	// codec with proper interface registry and signing options
	interfaceRegistry, err := codectypes.NewInterfaceRegistryWithOptions(codectypes.InterfaceRegistryOptions{
		ProtoFiles: proto.HybridResolver,
		SigningOptions: signing.Options{
			AddressCodec:          address.HexCodec{},
			ValidatorAddressCodec: address.HexCodec{},
		},
	})
	if err != nil {
		panic(err)
	}

	cryptocodec.RegisterInterfaces(interfaceRegistry)
	authTypes.RegisterInterfaces(interfaceRegistry)
	checkpointTypes.RegisterInterfaces(interfaceRegistry)
	milestoneTypes.RegisterInterfaces(interfaceRegistry)
	clerkTypes.RegisterInterfaces(interfaceRegistry)
	stakeTypes.RegisterInterfaces(interfaceRegistry)
	topupTypes.RegisterInterfaces(interfaceRegistry)

	cdc := codec.NewProtoCodec(interfaceRegistry)
	if clientCtx.Codec == nil {
		clientCtx = clientCtx.WithCodec(cdc)
	}

	// queue connector & cometbft http client
	qc := queue.NewQueueConnector(helper.GetConfig().AmqpURL)
	qc.StartWorker()

	httpClient, err := rpchttp.New(helper.GetConfig().CometBFTRPCUrl, "/websocket")
	if err != nil {
		panic(fmt.Sprintf("Error connecting to server %v", err))
	}

	// selected services
	var services []common.Service

	// start cometbft http client
	if err := httpClient.Start(); err != nil {
		logger.Error("Error connecting to server", "err", err)
		return err
	}

	// set chainId
	chainId := clientCtx.ChainID
	if chainId == "" {
		logger.Info("ChainID is empty in clientCtx at bridge startup, fetching from node status")
		// Fetch chain ID from node status
		nodeStatus, err := helper.GetNodeStatus(clientCtx, ctx)
		if err != nil {
			logger.Error("Error while fetching heimdall node status", "error", err)
			return err
		}
		if nodeStatus.NodeInfo.Network == "" {
			return errors.New("network is empty in node status, cannot determine chain ID")
		}
		chainId = nodeStatus.NodeInfo.Network
		logger.Info("ChainID fetched from node status", "chainId", chainId)
	} else {
		logger.Info("ChainID set in clientCtx", "chainId", chainId)
	}

	// ensure clientCtx carries the chain-id for signing/broadcast
	clientCtx = clientCtx.WithChainID(chainId)

	clientCtx.BroadcastMode = flags.BroadcastAsync

	// start bridge services only when node fully synced
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(waitDuration):
			if !util.IsCatchingUp(clientCtx, ctx) {
				logger.Info("Node up to date, starting bridge services")
				goto startServices
			}
			logger.Info("Waiting for heimdall to be synced")
		}
	}

startServices:
	var g errgroup.Group

	// Create the broadcaster (it will still poll for the account if needed)
	txBroadcaster := broadcaster.NewTxBroadcaster(cdc, ctx, clientCtx, nil)

	// Wire services now that we’re ready
	services = append(services,
		listener.NewListenerService(cdc, qc, httpClient),
		processor.NewProcessorService(cdc, qc, httpClient, txBroadcaster),
	)

	for _, svc := range services {
		s := svc // capture
		g.Go(func() error {
			if err := s.Start(); err != nil {
				logger.Error("service.Start failed", "err", err)
				return err
			}
			<-s.Quit()
			return nil
		})
	}

	// shutdown phase
	g.Go(func() error {
		<-ctx.Done()

		logger.Info("Received stop signal - Stopping all heimdall bridge services")
		for _, s := range services {
			if s.IsRunning() {
				if err := s.Stop(); err != nil {
					logger.Error("service.Stop failed", "err", err)
					return err
				}
			}
		}
		if err := httpClient.Stop(); err != nil {
			logger.Error("httpClient.Stop failed", "err", err)
			return err
		}
		util.CloseBridgeDBInstance()
		return nil
	})

	if err := g.Wait(); err != nil {
		logger.Error("Bridge stopped", "err", err)
		return err
	}

	return nil
}
