package clerk

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"cosmossdk.io/core/appmodule"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	gwruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/viper"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/0xPolygon/heimdall-v2/sidetxs"
	"github.com/0xPolygon/heimdall-v2/x/clerk/keeper"
	"github.com/0xPolygon/heimdall-v2/x/clerk/types"
)

// ConsensusVersion defines the current x/clerk module consensus version.
const ConsensusVersion = 1

var (
	_ module.HasABCIEndBlock = AppModule{}

	_ module.HasGenesis     = AppModule{}
	_ module.HasServices    = AppModule{}
	_ module.AppModuleBasic = AppModule{}

	_ appmodule.AppModule        = AppModule{}
	_ sidetxs.HasSideMsgServices = AppModule{}
)

// AppModule implements an application module for the clerk module.
type AppModule struct {
	keeper keeper.Keeper
}

// Name returns the clerk module's name.
func (AppModule) Name() string {
	return types.ModuleName
}

// RegisterLegacyAminoCodec registers the clerk module's types for the given codec.
func (AppModule) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	types.RegisterLegacyAminoCodec(cdc)
}

// DefaultGenesis returns default genesis state as raw bytes for the clerk module.
func (AppModule) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(types.DefaultGenesisState())
}

// ValidateGenesis performs genesis state validation for the clerk module.
func (AppModule) ValidateGenesis(cdc codec.JSONCodec, _ client.TxEncodingConfig, bz json.RawMessage) error {
	var data types.GenesisState
	if err := cdc.UnmarshalJSON(bz, &data); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", types.ModuleName, err)
	}

	return types.Validate(data)
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the clerk module.
func (AppModule) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *gwruntime.ServeMux) {
	if err := types.RegisterQueryHandlerClient(context.Background(), mux, types.NewQueryClient(clientCtx)); err != nil {
		panic(err)
	}
}

// RegisterInterfaces registers interfaces and implementations of the clerk module.
func (AppModule) RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	types.RegisterInterfaces(registry)
}

// IsAppModule implements the appmodule.AppModule interface.
func (am AppModule) IsAppModule() {}

// RegisterServices registers module services.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	types.RegisterMsgServer(cfg, keeper.NewMsgServerImpl(am.keeper))
	types.RegisterQueryServer(cfg, keeper.NewQueryServer(&am.keeper))
}

// RegisterSideMsgServices registers side handler module services.
func (am AppModule) RegisterSideMsgServices(sideCfg sidetxs.SideTxConfigurator) {
	types.RegisterSideMsgServer(sideCfg, keeper.NewSideMsgServerImpl(am.keeper))
}

// NewAppModule creates a new AppModule object
func NewAppModule(keeper keeper.Keeper) AppModule {
	return AppModule{
		keeper: keeper,
	}
}

// QuerierRoute returns the clerk module's querier route name.
func (AppModule) QuerierRoute() string { return types.RouterKey }

// InitGenesis performs genesis initialization for the clerk module.
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) {
	var genesisState types.GenesisState
	cdc.MustUnmarshalJSON(data, &genesisState)
	am.keeper.InitGenesis(ctx, &genesisState)
}

// ExportGenesis returns the exported genesis state as raw bytes for the clerk
// module.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	gs := am.keeper.ExportGenesis(ctx)
	return cdc.MustMarshalJSON(gs)
}

// ConsensusVersion implements AppModule/ConsensusVersion.
func (AppModule) ConsensusVersion() uint64 {
	return ConsensusVersion
}

// EndBlock runs at the end of every block which adds dummy event records for Mumbai.
func (am AppModule) EndBlock(ctx context.Context) ([]abci.ValidatorUpdate, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	am.keeper.Logger(sdkCtx).Info("EndBlock called")

	// Check if any dummy records already exists.
	hasFirstRecord := am.keeper.HasEventRecord(sdkCtx, 276851)
	am.keeper.Logger(sdkCtx).Info("Checking if first dummy record exists", "exists", hasFirstRecord)

	if !hasFirstRecord {
		am.keeper.Logger(sdkCtx).Info("Adding dummy event records for Mumbai in the EndBlock")

		dummyData := make([]byte, 32)
		dummyTxHash := "0x0000000000000000000000000000000000000000000000000000000000000000"
		dummyLogIndex := uint64(0)

		// Get the genesis time.
		genesisTime, err := am.getGenesisTime(sdkCtx)
		if err != nil {
			am.keeper.Logger(sdkCtx).Error("Failed to get genesis time", "error", err)
			return nil, nil
		}

		am.keeper.Logger(sdkCtx).Info("Got genesis time", "time", genesisTime)

		// Add the dummy event records for Mumbai.
		for eventID := uint64(276851); eventID <= 279428; eventID++ {
			// Check if the event record already exists or not.
			if am.keeper.HasEventRecord(sdkCtx, eventID) {
				am.keeper.Logger(sdkCtx).Info("Skipping adding dummy event record; already exists", "eventID", eventID)
				continue
			}

			am.keeper.Logger(sdkCtx).Info("Adding dummy clerk event record", "eventID", eventID)

			// Add the dummy event record.
			dummyEvent := types.EventRecord{
				Id:         eventID,
				Contract:   "0xcf73231f28b7331bbe3124b907840a94851f9f11",
				Data:       dummyData,
				TxHash:     dummyTxHash,
				LogIndex:   dummyLogIndex,
				BorChainId: "80001",
				RecordTime: genesisTime,
			}

			if err := am.keeper.SetEventRecord(sdkCtx, dummyEvent); err != nil {
				am.keeper.Logger(sdkCtx).Error("Error in storing Mumbai dummy event record", "id", eventID, "error", err)
				return nil, nil
			}
			am.keeper.Logger(sdkCtx).Info("Dummy event record added", "id", eventID)
		}
		am.keeper.Logger(sdkCtx).Info("Dummy event records added for Mumbai in EndBlock")
	} else {
		am.keeper.Logger(sdkCtx).Info("Dummy event records already added for Mumbai")
	}

	return nil, nil
}

type SimpleGenesisDoc struct {
	GenesisTime time.Time `json:"genesis_time"`
}

// getGenesisTime retrieves the genesis time from the genesis file.
func (am AppModule) getGenesisTime(ctx sdk.Context) (time.Time, error) {
	homeDir := viper.GetString(flags.FlagHome)
	if homeDir == "" {
		am.keeper.Logger(ctx).Error("Failed to get home directory")
		return time.Time{}, status.Error(codes.Internal, "failed to get home directory")
	}

	genesisPath := filepath.Join(homeDir, "config", "genesis.json")
	genesisBytes, err := os.ReadFile(genesisPath)
	if err != nil {
		am.keeper.Logger(ctx).Error("Failed to read genesis file", "path", genesisPath, "error", err)
		return time.Time{}, status.Error(codes.Internal, "failed to read genesis file")
	}

	var genesisDoc SimpleGenesisDoc
	if err := json.Unmarshal(genesisBytes, &genesisDoc); err != nil {
		am.keeper.Logger(ctx).Error("Failed to parse genesis file", "path", genesisPath, "error", err)
		return time.Time{}, status.Error(codes.Internal, "failed to parse genesis file")
	}

	return genesisDoc.GenesisTime, nil
}
