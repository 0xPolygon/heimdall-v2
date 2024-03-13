package bor

import (
	"encoding/json"
	"fmt"
	"time"

	"cosmossdk.io/core/appmodule"
	"github.com/0xPolygon/heimdall-v2/x/bor/keeper"

	"github.com/0xPolygon/heimdall-v2/x/bor/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	gwruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"
)

var (
	_ module.AppModuleSimulation = AppModule{}
	_ module.HasGenesis          = AppModule{}
	_ module.HasServices         = AppModule{}
	_ module.AppModuleBasic      = AppModule{}

	_ appmodule.AppModule = AppModule{}
)

// TODO HV2: what should this value be ?

// ConsensusVersion defines the current x/bank module consensus version.
const ConsensusVersion = 1

// AppModuleBasic defines the basic application module used by the bor module.
// type AppModuleBasic struct{}

// Name returns the bor module's name.
func (AppModule) Name() string { return types.ModuleName }

// RegisterLegacyAminoCodec registers the bor module's types on the LegacyAmino codec.
func (AppModule) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	types.RegisterLegacyAminoCodec(cdc)
}

// DefaultGenesis returns default genesis state as raw bytes for the bor
// module.
func (AppModule) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(types.DefaultGenesisState())
}

// ValidateGenesis performs genesis state validation for the bor module.
func (AppModule) ValidateGenesis(cdc codec.JSONCodec, _ client.TxEncodingConfig, bz json.RawMessage) error {
	var data types.GenesisState
	if err := cdc.UnmarshalJSON(bz, &data); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", types.ModuleName, err)
	}

	return data.Validate()
}

// TODO HV2: implement when this PR is merged: https://github.com/0xPolygon/heimdall-v2/pull/20

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the bor module.
func (AppModule) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *gwruntime.ServeMux) {}

// RegisterInterfaces registers interfaces and implementations of the bor module.
func (AppModule) RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	types.RegisterInterfaces(registry)
}

// AppModule implements an application module for the bor module.
type AppModule struct {
	keeper keeper.Keeper
	// contractCaller helper.IContractCaller

}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (am AppModule) IsOnePerModuleType() {}

// IsAppModule implements the appmodule.AppModule interface.
func (am AppModule) IsAppModule() {}

// TODO HV2: uncomment when keeper and types are implemented

// RegisterServices registers module services.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	types.RegisterQueryServer(cfg.QueryServer(), am.keeper)

	// TODO HV2: probably don't need migration
	// m := keeper.NewMigrator(am.keeper, am.legacySubspace)
	// if err := cfg.RegisterMigration(types.ModuleName, 1, m.Migrate1to2); err != nil {
	// 	panic(fmt.Sprintf("failed to migrate x/bor from version 1 to 2: %v", err))
	// }

	// if err := cfg.RegisterMigration(types.ModuleName, 2, m.Migrate2to3); err != nil {
	// 	panic(fmt.Sprintf("failed to migrate x/bor from version 2 to 3: %v", err))
	// }

	// if err := cfg.RegisterMigration(types.ModuleName, 3, m.Migrate3to4); err != nil {
	// 	panic(fmt.Sprintf("failed to migrate x/bor from version 3 to 4: %v", err))
	// }
}

// NewAppModule creates a new AppModule object
func NewAppModule(
	keeper keeper.Keeper,
	// contractCaller helper.IContractCaller,
) AppModule {
	return AppModule{
		keeper: keeper,
		// contractCaller: contractCaller,
	}
}

// TODO HV2: uncomment when types is implemented

// QuerierRoute returns the bor module's querier route name.
func (AppModule) QuerierRoute() string { return types.RouterKey }

// TODO HV2: uncomment when keeper is implemented

// InitGenesis performs genesis initialization for the bor module. It returns
// no validator updates.
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) {
	start := time.Now()
	var genesisState types.GenesisState
	cdc.MustUnmarshalJSON(data, &genesisState)
	telemetry.MeasureSince(start, "InitGenesis", "bor", "unmarshal")

	am.keeper.InitGenesis(ctx, &genesisState)
}

// TODO HV2: uncomment when keeper is implemented

// ExportGenesis returns the exported genesis state as raw bytes for the bor
// module.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	gs := am.keeper.ExportGenesis(ctx)
	return cdc.MustMarshalJSON(gs)
}

// ConsensusVersion implements AppModule/ConsensusVersion.
func (AppModule) ConsensusVersion() uint64 { return ConsensusVersion }

// AppModuleSimulation functions

// TODO HV2: uncomment when simulation is implemented

// GenerateGenesisState creates a randomized GenState of the bor module.
// func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
// 	simulation.RandomizedGenState(simState)
// }

// TODO HV2: this is a no-op in current heimdall. Probably no need to implement this
// looks equivalent to https://github.com/maticnetwork/heimdall/blob/249aa798c2f23c533d2421f2101127c11684c76e/bor/module.go#L161C18-L161C34

// ProposalMsgs returns msgs used for governance proposals for simulations.
// func (AppModule) ProposalMsgs(_ module.SimulationState) []simtypes.WeightedProposalMsg {
// 	return nil
// }

// TODO HV2: this is a no-op in current heimdall. Probably no need to implement this

// RegisterStoreDecoder registers a decoder for bor module's types
// func (am AppModule) RegisterStoreDecoder(_ simtypes.StoreDecoderRegistry) {}

// TODO HV2: uncomment when simulation is implemented

// RandomizedParams creates randomized param changes for the simulator.
// func (AppModule) RandomizedParams(r *rand.Rand) []simTypes.ParamChange {
// 	return simulation.ParamChanges(r)
// }

// TODO HV2: this is a no-op in current heimdall. Probably no need to implement this

// WeightedOperations returns the all the gov module operations with their respective weights.
// func (am AppModule) WeightedOperations(_ module.SimulationState) []simtypes.WeightedOperation {
// 	return nil
// }
