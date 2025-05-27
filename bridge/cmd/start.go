package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
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
	waitDuration = 1 * time.Minute
)

// StartBridgeWithCtx starts bridge service, and it's able to shut down gracefully
// returns service errors, if any
func StartBridgeWithCtx(shutdownCtx context.Context, clientCtx client.Context) error {
	// create codec
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

	// queue connector & http client
	_queueConnector := queue.NewQueueConnector(helper.GetConfig().AmqpURL)
	_queueConnector.StartWorker()

	_txBroadcaster := broadcaster.NewTxBroadcaster(cdc, clientCtx, nil) //nolint:contextcheck
	_httpClient, err := rpchttp.New(helper.GetConfig().CometBFTRPCUrl, "/websocket")
	if err != nil {
		panic(fmt.Sprintf("Error connecting to server %v", err))
	}

	// selected services to start
	var services []common.Service
	services = append(services,
		listener.NewListenerService(cdc, _queueConnector, _httpClient),
		processor.NewProcessorService(cdc, _queueConnector, _httpClient, _txBroadcaster),
	)

	// Start http client
	err = _httpClient.Start()
	if err != nil {
		logger.Error("Error connecting to server: %v", err)
		return err
	}

	clientCtx.BroadcastMode = flags.BroadcastAsync

	// start bridge services only when node fully synced
	loop := true
	for loop {
		select {
		case <-shutdownCtx.Done():
			return nil
		case <-time.After(waitDuration):
			if !util.IsCatchingUp(clientCtx, shutdownCtx) {
				logger.Info("Node up to date, starting bridge services")

				loop = false
			} else {
				logger.Info("Waiting for heimdall to be synced")
			}
		}
	}

	// start services
	var g errgroup.Group

	for _, service := range services {
		// loop variable must be captured
		srv := service

		g.Go(func() error {
			if err := srv.Start(); err != nil {
				logger.Error("GetStartCmd | serv.Start", "Error", err)
				return err
			}
			<-srv.Quit()
			return nil
		})
	}

	// shutdown phase
	g.Go(func() error {
		// wait for interrupt and start the shut-down
		<-shutdownCtx.Done()

		logger.Info("Received stop signal - Stopping all heimdall bridge services")
		for _, service := range services {
			srv := service
			if srv.IsRunning() {
				if err := srv.Stop(); err != nil {
					logger.Error("GetStartCmd | service.Stop", "Error", err)
					return err
				}
			}
		}
		// stop http client
		if err := _httpClient.Stop(); err != nil {
			logger.Error("GetStartCmd | _httpClient.Stop", "Error", err)
			return err
		}
		// stop db instance
		util.CloseBridgeDBInstance()

		return nil
	})

	// wait for all routines to finish and log the error
	if err := g.Wait(); err != nil {
		logger.Error("Bridge stopped", "err", err)
		return err
	}

	return nil
}

// StartBridge starts bridge service, isStandAlone prevents os.Exit if the bridge started as side service
func StartBridge(isStandAlone bool) {
	// create codec
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(interfaceRegistry)
	authTypes.RegisterInterfaces(interfaceRegistry)
	checkpointTypes.RegisterInterfaces(interfaceRegistry)
	milestoneTypes.RegisterInterfaces(interfaceRegistry)
	clerkTypes.RegisterInterfaces(interfaceRegistry)
	stakeTypes.RegisterInterfaces(interfaceRegistry)
	topupTypes.RegisterInterfaces(interfaceRegistry)
	cdc := codec.NewProtoCodec(interfaceRegistry)

	// cli context
	cliCtx := client.Context{}.WithCodec(cdc)
	cliCtx.BroadcastMode = flags.BroadcastAsync

	if cliCtx.Codec == nil {
		cliCtx = cliCtx.WithCodec(cdc)
	}

	// queue connector & http client
	_queueConnector := queue.NewQueueConnector(helper.GetConfig().AmqpURL)
	_queueConnector.StartWorker()

	_txBroadcaster := broadcaster.NewTxBroadcaster(cdc, client.Context{}, nil)
	_httpClient, err := rpchttp.New(helper.GetConfig().CometBFTRPCUrl, "/websocket")
	if err != nil {
		panic(fmt.Sprintf("Error connecting to server %v", err))
	}

	// selected services to start
	var services []common.Service
	services = append(services,
		listener.NewListenerService(cdc, _queueConnector, _httpClient),
		processor.NewProcessorService(cdc, _queueConnector, _httpClient, _txBroadcaster),
	)

	// sync group
	var wg sync.WaitGroup

	// go routine to catch signal
	catchSignal := make(chan os.Signal, 1)
	signal.Notify(catchSignal, os.Interrupt, syscall.SIGTERM)

	go func() {
		// sig is a ^C, handle it
		for range catchSignal {
			// stop processes
			logger.Info("Received stop signal - Stopping all services")

			for _, service := range services {
				if err := service.Stop(); err != nil {
					logger.Error("GetStartCmd | service.Stop", "Error", err)
				}
			}

			// stop http client
			if err := _httpClient.Stop(); err != nil {
				logger.Error("GetStartCmd | _httpClient.Stop", "Error", err)
			}

			// stop db instance
			util.CloseBridgeDBInstance()

			// exit
			if isStandAlone {
				os.Exit(1)
			}
		}
	}()

	// Start http client
	err = _httpClient.Start()
	if err != nil {
		panic(fmt.Sprintf("Error connecting to server %v", err))
	}

	// start bridge services only when node fully synced
	for {
		if !util.IsCatchingUp(cliCtx, context.Background()) {
			logger.Info("Node upto date, starting bridge services")
			break
		} else {
			logger.Info("Waiting for heimdall to be synced")
		}

		time.Sleep(waitDuration)
	}

	// start all processes
	for _, service := range services {
		go func(serv common.Service) {
			defer wg.Done()
			if err := serv.Start(); err != nil {
				logger.Error("GetStartCmd | serv.Start", "Error", err)
			}

			<-serv.Quit()
		}(service)
	}

	// wait for all processes
	wg.Add(len(services))
	wg.Wait()
}

// GetStartCmd returns the start command to start bridge
func GetStartCmd() *cobra.Command {
	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start bridge server",
		Run: func(cmd *cobra.Command, args []string) {
			StartBridge(true)
		},
	}

	startCmd.Flags().Bool("all", false, "Start all bridge services")

	if err := viper.BindPFlag("all", startCmd.Flags().Lookup("all")); err != nil {
		logger.Error("GetStartCmd | BindPFlag | all", "Error", err)
	}

	startCmd.Flags().StringSlice("only", []string{}, "Comma separated bridge services to start")

	if err := viper.BindPFlag("only", startCmd.Flags().Lookup("only")); err != nil {
		logger.Error("GetStartCmd | BindPFlag | only", "Error", err)
	}

	return startCmd
}

func init() {
	rootCmd.AddCommand(GetStartCmd())
}
