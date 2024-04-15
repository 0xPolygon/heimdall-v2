package topup

import (
	"context"
	"encoding/json"
	"fmt"

	"cosmossdk.io/core/appmodule"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/types/time"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/types/simulation"
	gwruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"

	// TODO HV2: enable when module is implemented
	//mod "github.com/0xPolygon/heimdall-v2/module"
	"github.com/0xPolygon/heimdall-v2/x/topup/keeper"
	topupSimulation "github.com/0xPolygon/heimdall-v2/x/topup/simulation"
	"github.com/0xPolygon/heimdall-v2/x/topup/types"
)

var (
	_ module.AppModuleSimulation = AppModule{}
	_ module.HasGenesis          = AppModule{}
	_ module.HasServices         = AppModule{}
	_ module.AppModuleBasic      = AppModule{}
	_ appmodule.AppModule        = AppModule{}
)

// AppModule implements an application module for the topup module.
type AppModule struct {
	keeper keeper.Keeper
	// TODO HV2: enable contractCaller when implemented in heimdall-v2
	// contractCaller helper.IContractCaller
}

func NewAppModule(keeper keeper.Keeper) AppModule {
	return AppModule{
		keeper: keeper,
		// TODO HV2: enable contractCaller when implemented in heimdall-v2
		// contractCaller: contractCaller,
	}
}

// Name returns the topup module's name.
func (AppModule) Name() string {
	return types.ModuleName
}

// RegisterLegacyAminoCodec registers the topup module's types on the LegacyAmino codec.
func (AppModule) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	types.RegisterLegacyAminoCodec(cdc)
}

/* TODO HV2: enable when module is implemented
// RegisterSideMsgServices registers side handler module services.
func (am AppModule) RegisterSideMsgServices(sideCfg mod.SideTxConfigurator) {
	types.RegisterSideMsgServer(sideCfg, keeper.NewSideMsgServerImpl(&am.keeper))
}
*/

// DefaultGenesis returns default genesis state as raw bytes for the topup module.
func (AppModule) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(types.DefaultGenesisState())
}

// ValidateGenesis performs genesis state validation for the topup module.
func (AppModule) ValidateGenesis(cdc codec.JSONCodec, _ client.TxEncodingConfig, bz json.RawMessage) error {
	var data types.GenesisState
	if err := cdc.UnmarshalJSON(bz, &data); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", types.ModuleName, err)
	}

	return data.Validate()
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the topup module.
func (AppModule) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *gwruntime.ServeMux) {
	if err := types.RegisterQueryHandlerClient(context.Background(), mux, types.NewQueryClient(clientCtx)); err != nil {
		panic(err)
	}
}

// RegisterInterfaces registers interfaces and implementations of the chainmanager module.
func (AppModule) RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	types.RegisterInterfaces(registry)
}

// IsAppModule implements the appmodule.AppModule interface.
func (am AppModule) IsAppModule() {
}

// RegisterServices registers module services.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	types.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(&am.keeper))
	types.RegisterQueryServer(cfg.QueryServer(), keeper.NewQueryServer(&am.keeper))
}

// InitGenesis performs genesis initialization for the topup module.
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) {
	start := time.Now()
	var genesisState types.GenesisState
	cdc.MustUnmarshalJSON(data, &genesisState)
	telemetry.MeasureSince(start, "InitGenesis", "topup", "unmarshal")

	am.keeper.InitGenesis(ctx, &genesisState)
}

// ExportGenesis returns the exported genesis state as raw bytes for the topup module.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(am.keeper.ExportGenesis(ctx))
}

// GenerateGenesisState creates a randomized GenState of the topup module.
func (am AppModule) GenerateGenesisState(simState *module.SimulationState) {
	topupSimulation.RandomizeGenState(simState)
}

// RegisterStoreDecoder registers a decoder for topup module's types
func (am AppModule) RegisterStoreDecoder(_ simulation.StoreDecoderRegistry) {
}

// WeightedOperations returns the all the gov module operations with their respective weights.
func (am AppModule) WeightedOperations(_ module.SimulationState) []simulation.WeightedOperation {
	return nil
}

// BeginBlock returns the BeginBlocker for the x/topup module.
func (am AppModule) BeginBlock(ctx context.Context) error {
	return am.keeper.BeginBlocker(ctx)
}

// EndBlock returns the EndBlocker for the x/topup module.
func (am AppModule) EndBlock(ctx context.Context) ([]abci.ValidatorUpdate, error) {
	return am.keeper.EndBlocker(ctx)
}
