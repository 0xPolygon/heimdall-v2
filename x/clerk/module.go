package clerk

import (
	"context"
	"encoding/json"
	"fmt"

	"cosmossdk.io/core/appmodule"
	gwruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/0xPolygon/heimdall-v2/x/clerk/keeper"
	"github.com/0xPolygon/heimdall-v2/x/clerk/types"
)

var (
	_ module.HasGenesis = AppModule{}

	// TODO HV2 - check if we need the following (this was giving an error)
	// _ module.HasServices    = AppModule{}

	_ module.AppModuleBasic = AppModule{}
	_ appmodule.AppModule   = AppModule{}
)

// // AppModule implements an application module for the clerk module.
// type AppModule struct {
// 	keeper *keeper.Keeper
// }

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
func (AppModule) ValidateGenesis(cdc codec.JSONCodec, config client.TxEncodingConfig, bz json.RawMessage) error {
	var data types.GenesisState
	if err := cdc.UnmarshalJSON(bz, &data); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", types.ModuleName, err)
	}

	return types.ValidateGenesis(data)
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

// AppModule implements an application module for the clerk module.
type AppModule struct {
	keeper keeper.Keeper
}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (am AppModule) IsOnePerModuleType() {}

// IsAppModule implements the appmodule.AppModule interface.
func (am AppModule) IsAppModule() {}

// TODO HV2 - check if this is correct (cosmos way)
// RegisterServices registers module services.
func (am AppModule) RegisterServices(cfg module.Configurator) error {
	types.RegisterMsgServer(cfg, keeper.NewMsgServerImpl(am.keeper))
	types.RegisterQueryServer(cfg, keeper.NewQueryServer(am.keeper))

	return nil
}

// NewAppModule creates a new AppModule object
func NewAppModule(cdc codec.Codec, keeper keeper.Keeper) AppModule {
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
	InitGenesis(ctx, &am.keeper, &genesisState)
}

// ExportGenesis returns the exported genesis state as raw bytes for the clerk
// module.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	gs := ExportGenesis(ctx, &am.keeper)
	return cdc.MustMarshalJSON(gs)
}

/*
TODO HV2 - I believe we dont need the following simulation functions for clerk module
AppModuleSimulation functions
*/

/*
// GenerateGenesisState creates a randomized GenState of the clerk module.
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	simulation.RandomizedGenState(simState)
}

// ProposalMsgs returns msgs used for governance proposals for simulations.
func (AppModule) ProposalMsgs(simState module.SimulationState) []simtypes.WeightedProposalMsg {
	return simulation.ProposalMsgs()
}

// RegisterStoreDecoder registers a decoder for supply module's types
func (am AppModule) RegisterStoreDecoder(sdr simtypes.StoreDecoderRegistry) {
	// TODO HV2 - what to do of Schema?
	sdr[types.StoreKey] = simtypes.NewStoreDecoderFuncFromCollectionsSchema(am.keeper.Schema)
}

// WeightedOperations doesn't return any clerk module operation.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	return nil
}
*/
