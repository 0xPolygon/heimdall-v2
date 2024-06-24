package heimdalld

import (
	"context"
	"fmt"
	"os"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/0xPolygon/heimdall-v2/app"
	"github.com/0xPolygon/heimdall-v2/helper"
	cmtcmd "github.com/cometbft/cometbft/cmd/cometbft/commands"
	cmtcfg "github.com/cometbft/cometbft/config"
	"github.com/cometbft/cometbft/node"
	"github.com/cometbft/cometbft/p2p"
	"github.com/cometbft/cometbft/privval"
	"github.com/cometbft/cometbft/proxy"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/server"
	servercmtlog "github.com/cosmos/cosmos-sdk/server/log"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

func heimdallStart(hApp *app.HeimdallApp) *cobra.Command {
	cdc := codec.NewLegacyAmino()
	ctx := server.NewDefaultContext()

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Run the full node",
		Long: `Run the full node application with CometBFT in process.
Starting rest server is provided with the flag --rest-server and starting bridge with
the flag --bridge when starting CometBFT in process.
Pruning options can be provided via the '--pruning' flag. The options are as follows:

syncable: only those states not needed for state syncing will be deleted (keeps last 100 + every 10000th)
nothing: all historic states will be saved, nothing will be deleted (i.e. archiving node)
everything: all saved states will be deleted, storing only the current state

Node halting configurations exist in the form of two flags: '--halt-height' and '--halt-time'. During
the ABCI Commit phase, the node will check if the current block height is greater than or equal to
the halt-height or if the current block time is greater than or equal to the halt-time. If so, the
node will attempt to gracefully shutdown and the block will not be committed. In addition, the node
will not be able to commit subsequent blocks.

For profiling and benchmarking purposes, CPU profiling can be enabled via the '--cpu-profile' flag
which accepts a path for the resulting pprof file.
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, arg := range args {
				if !strings.HasPrefix(arg, "--") {
					return fmt.Errorf(
						"\tinvalid argument: %s \n\tall flags must start with --",
						arg)
				}
			}
			LogsWriterFile := viper.GetString(helper.LogsWriterFileFlag)
			if LogsWriterFile != "" {
				logWriter := helper.GetLogsWriter(LogsWriterFile)

				logger, err := server.CreateSDKLogger(ctx, logWriter)
				if err != nil {
					logger.Error("unable to setup logger", "err", err)
					return err
				}

				ctx.Logger = logger
			}

			ctx.Logger.Info("starting ABCI with CometBFT")

			startRestServer, _ := cmd.Flags().GetBool(helper.RestServerFlag)
			startBridge, _ := cmd.Flags().GetBool(helper.BridgeFlag)

			err := startInProcess(cmd, ctx, newApp, cdc, startRestServer, startBridge, hApp)
			return err
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			// bridge binding
			if err := viper.BindPFlag("all", cmd.Flags().Lookup("all")); err != nil {
				logger.Error("getstartcmd | bindpflag | all", "error", err)
			}

			if err := viper.BindPFlag("only", cmd.Flags().Lookup("only")); err != nil {
				logger.Error("getstartcmd | bindpflag | only", "error", err)
			}
		},
	}

	cmd.Flags().Bool(
		helper.RestServerFlag,
		false,
		"Start rest server",
	)

	cmd.Flags().Bool(
		helper.BridgeFlag,
		false,
		"Start bridge service",
	)

	cmd.PersistentFlags().String(helper.LogLevel, ctx.Config.LogLevel, "Log level")

	if err := viper.BindPFlag(helper.LogLevel, cmd.PersistentFlags().Lookup(helper.LogLevel)); err != nil {
		logger.Error("main | bindpflag | helper.loglevel", "error", err)
	}

	// bridge flags =  start flags (all, only) + root bridge cmd flags
	cmd.Flags().Bool("all", false, "start all bridge services")
	cmd.Flags().StringSlice("only", []string{}, "comma separated bridge services to start")
	// TODO HV2 - uncomment when we have bridge implemented
	// bridgeCmd.DecorateWithBridgeRootFlags(cmd, viper.GetViper(), logger, "main")

	// TODO HV2 - uncomment when we have server implemented
	// rest server flags
	// restServer.DecorateWithRestFlags(cmd)

	// core flags for the ABCI application
	cmd.Flags().String(flagAddress, "tcp://0.0.0.0:26658", "Listen address")
	cmd.Flags().String(flagTraceStore, "", "Enable KVStore tracing to an output file")
	cmd.Flags().String(flagPruning, "syncable", "Pruning strategy: syncable, nothing, everything")
	cmd.Flags().String(
		FlagMinGasPrices, "",
		"Minimum gas prices to accept for transactions; Any fee in a tx must meet this minimum (e.g. 0.01matic;0.0001pol)",
	)
	cmd.Flags().Uint64(FlagHaltHeight, 0, "Height at which to gracefully halt the chain and shutdown the node")
	cmd.Flags().Uint64(FlagHaltTime, 0, "Minimum block time (in Unix seconds) at which to gracefully halt the chain and shutdown the node")
	cmd.Flags().String(flagCPUProfile, "", "Enable CPU profiling and write to the provided file")
	cmd.Flags().String(helper.FlagClientHome, helper.DefaultCLIHome, "client's home directory")

	cmd.Flags().Bool(FlagOpenTracing, false, "start open tracing")
	cmd.Flags().String(FlagOpenCollectorEndpoint, helper.DefaultOpenCollectorEndpoint, "Default OpenTelemetry Collector Endpoint")

	// add support for all CometBFT-specific command line options
	cmtcmd.AddNodeFlags(cmd)

	return cmd
}

func startOpenTracing(cmd *cobra.Command) (*sdktrace.TracerProvider, *context.Context, error) {
	opentracingEnabled, _ := cmd.Flags().GetBool(FlagOpenTracing)
	if opentracingEnabled {
		openCollectorEndpoint, _ := cmd.Flags().GetString(FlagOpenCollectorEndpoint)
		ctx := context.Background()

		res, err := resource.New(ctx,
			resource.WithAttributes(
				// the service name used to display traces in backends
				semconv.ServiceNameKey.String("heimdall"),
			),
		)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create open telemetry resource for service: %v", err)
		}

		// Set up a trace exporter
		var traceExporter *otlptrace.Exporter

		traceExporterReady := make(chan *otlptrace.Exporter, 1)

		go func() {
			traceExporter, _ := otlptracegrpc.New(
				ctx,
				otlptracegrpc.WithInsecure(),
				otlptracegrpc.WithEndpoint(openCollectorEndpoint),
				otlptracegrpc.WithDialOption(grpc.WithBlock()),
			)
			traceExporterReady <- traceExporter
		}()

		select {
		case traceExporter = <-traceExporterReady:
			fmt.Println("TraceExporter Ready")
		case <-time.After(5 * time.Second):
			fmt.Println("TraceExporter Timed Out in 5 Seconds")
		}

		// Register the trace exporter with a TracerProvider, using a batch
		// span processor to aggregate spans before export.
		if traceExporter == nil {
			return nil, nil, nil
		}

		bsp := sdktrace.NewBatchSpanProcessor(traceExporter)
		tracerProvider := sdktrace.NewTracerProvider(
			sdktrace.WithSampler(sdktrace.AlwaysSample()),
			sdktrace.WithResource(res),
			sdktrace.WithSpanProcessor(bsp),
		)
		otel.SetTracerProvider(tracerProvider)

		// set global propagator to tracecontext (the default is no-op).
		otel.SetTextMapPropagator(propagation.TraceContext{})

		return tracerProvider, &ctx, nil
	}

	return nil, nil, nil
}

func startInProcess(cmd *cobra.Command, ctx *server.Context, appCreator servertypes.AppCreator, cdc *codec.LegacyAmino, startRestServer bool, startBridge bool, hApp *app.HeimdallApp) error {
	cfg := ctx.Config
	home := cfg.RootDir
	traceWriterFile := viper.GetString(flagTraceStore)
	vp := viper.New()

	// initialize heimdall if needed (do not force!)
	initConfig := &initHeimdallConfig{
		chainID:     "", // chain id should be auto generated if chain flag is not set to mumbai, amoy or mainnet
		chain:       viper.GetString(helper.ChainFlag),
		validatorID: 1, // default id for validator
		clientHome:  viper.GetString(helper.FlagClientHome),
		forceInit:   false,
	}

	if err := heimdallInit(ctx, cdc, initConfig, cfg, hApp.BasicManager, hApp.AutoCliOpts().ClientCtx.Codec); err != nil {
		return fmt.Errorf("failed init heimdall: %s", err)
	}

	db, err := openDB(home)
	if err != nil {
		return fmt.Errorf("failed to open db: %s", err)
	}

	traceWriter, err := openTraceWriter(traceWriterFile)
	if err != nil {
		return fmt.Errorf("failed to open trace writer: %s", err)
	}

	appc := appCreator(ctx.Logger, db, traceWriter, vp)

	nodeKey, err := p2p.LoadOrGenNodeKey(cfg.NodeKeyFile())
	if err != nil {
		return fmt.Errorf("failed to load or gen node key: %s", err)
	}

	cmtApp := server.NewCometABCIWrapper(appc)

	// create & start cometbft node
	tmNode, err := node.NewNode(
		cfg,
		privval.LoadOrGenFilePV(cfg.PrivValidatorKeyFile(), cfg.PrivValidatorStateFile()),
		nodeKey,
		proxy.NewLocalClientCreator(cmtApp),
		node.DefaultGenesisDocProviderFunc(cfg),
		cmtcfg.DefaultDBProvider,
		node.DefaultMetricsProvider(cfg.Instrumentation),
		servercmtlog.CometLoggerWrapper{Logger: ctx.Logger},
	)
	if err != nil {
		return fmt.Errorf("failed to create new node: %s", err)
	}

	// start CometBFT node here
	if err = tmNode.Start(); err != nil {
		return fmt.Errorf("failed to start cometbft node: %s", err)
	}

	var cpuProfileCleanup func()

	if cpuProfile := viper.GetString(flagCPUProfile); cpuProfile != "" {
		f, err := os.Create(cpuProfile)
		if err != nil {
			return err
		}

		ctx.Logger.Info("starting CPU profiler", "profile", cpuProfile)

		if err = pprof.StartCPUProfile(f); err != nil {
			return err
		}

		cpuProfileCleanup = func() {
			ctx.Logger.Info("stopping CPU profiler", "profile", cpuProfile)
			pprof.StopCPUProfile()
			if err = f.Close(); err != nil {
				ctx.Logger.Error("failed to close CPU profile", "error", err)
			}
		}
	}

	tracerProvider, traceCtx, _ := startOpenTracing(cmd)

	// using group context makes sense in case that if one of
	// the processes produces error the rest will go and shutdown
	g, gCtx := errgroup.WithContext(nil)
	// start rest
	if startRestServer {
		waitForREST := make(chan struct{})

		// TODO HV2 - uncomment when we have server implemented
		/*
			g.Go(func() error {
				return restServer.StartRestServer(gCtx, cdc, restServer.RegisterRoutes, waitForREST)
			})
		*/

		// hang here for a while, and wait for REST server to start
		<-waitForREST
	}

	// TODO HV2 - uncomment when we have bridge implemented
	/*
		// start bridge
		if startBridge {
			bridgeCmd.AdjustBridgeDBValue(cmd, viper.GetViper())
			g.Go(func() error {
				return bridgeCmd.StartBridgeWithCtx(gCtx)
			})
		}
	*/

	// stop phase for CometBFT node
	g.Go(func() error {
		// wait here for interrupt signal or
		// until something in the group returns non-nil error
		<-gCtx.Done()
		ctx.Logger.Info("exiting...")

		if tracerProvider != nil {
			// nolint: contextcheck
			if err := tracerProvider.Shutdown(*traceCtx); err == nil {
				ctx.Logger.Info("Shutting Down OpenTelemetry")
			}
		}

		if cpuProfileCleanup != nil {
			cpuProfileCleanup()
		}
		if tmNode.IsRunning() {
			return tmNode.Stop()
		}

		err = db.Close()
		if err != nil {
			return err
		}

		return nil
	})

	// wait here for all go routines to finish,
	// or something to break
	if err := g.Wait(); err != nil {
		ctx.Logger.Error("error shutting down services", "error", err)
		return err
	}

	logger.Info("Heimdall services stopped")

	return nil
}
