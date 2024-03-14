package chainmanager

import (
	"encoding/json"
	"fmt"
	"time"

	"cosmossdk.io/core/appmodule"
	"github.com/0xPolygon/heimdall-v2/x/chainmanager/keeper"
	"github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth/exported"
	gwruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"
)

var (
	// TODO HV2: uncomment when implemented
	_ module.AppModuleBasic = AppModule{}
	// _ module.AppModuleSimulation = AppModule{}
	_ module.HasGenesis  = AppModule{}
	_ module.HasServices = AppModule{}
	// _ module.HasInvariants = AppModule{}

	_ appmodule.AppModule = AppModule{}
)

// TODO HV2: what should this value be ?

// ConsensusVersion defines the current x/bank module consensus version.
const ConsensusVersion = 1

// AppModuleBasic defines the basic application module used by the chainmananer module.
// type AppModuleBasic struct{}

// Name returns the chainmanager module's name.
func (AppModule) Name() string { return types.ModuleName }

// RegisterLegacyAminoCodec registers the chainmanager module's types on the LegacyAmino codec.
func (AppModule) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	types.RegisterLegacyAminoCodec(cdc)
}

// DefaultGenesis returns default genesis state as raw bytes for the chainmanager
// module.
func (AppModule) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(types.DefaultGenesisState())
}

// ValidateGenesis performs genesis state validation for the chainmanager module.
func (AppModule) ValidateGenesis(cdc codec.JSONCodec, _ client.TxEncodingConfig, bz json.RawMessage) error {
	var data types.GenesisState
	if err := cdc.UnmarshalJSON(bz, &data); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", types.ModuleName, err)
	}

	return data.Validate()
}

// TODO HV2: implement when this PR is merged: https://github.com/0xPolygon/heimdall-v2/pull/20

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the chainmanager module.
func (AppModule) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *gwruntime.ServeMux) {}

// RegisterInterfaces registers interfaces and implementations of the chainmanager module.
func (AppModule) RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	types.RegisterInterfaces(registry)
}

// AppModule implements an application module for the chainmanager module.
type AppModule struct {
	keeper keeper.Keeper
	// contractCaller helper.IContractCaller

	// TODO HV2: do we need legacySubspace?
	// legacySubspace is used solely for migration of x/params managed parameters
	legacySubspace exported.Subspace
}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (am AppModule) IsOnePerModuleType() {}

// IsAppModule implements the appmodule.AppModule interface.
func (am AppModule) IsAppModule() {}

// TODO HV2: uncomment when keeper and types are implemented

// RegisterServices registers module services.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	types.RegisterQueryServer(cfg.QueryServer(), am.keeper)
}

// NewAppModule creates a new AppModule object
func NewAppModule(
	keeper keeper.Keeper,
	// contractCaller helper.IContractCaller,
	ss exported.Subspace,
) AppModule {
	return AppModule{
		keeper: keeper,
		// contractCaller: contractCaller,
		legacySubspace: ss,
	}
}

// TODO HV2: uncomment when types is implemented

// QuerierRoute returns the chainmanager module's querier route name.
func (AppModule) QuerierRoute() string { return types.RouterKey }

// TODO HV2: uncomment when keeper is implemented

// InitGenesis performs genesis initialization for the chainmanager module. It returns
// no validator updates.
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) {
	start := time.Now()
	var genesisState types.GenesisState
	cdc.MustUnmarshalJSON(data, &genesisState)
	telemetry.MeasureSince(start, "InitGenesis", "chainmanager", "unmarshal")

	am.keeper.InitGenesis(ctx, &genesisState)
}

// TODO HV2: uncomment when keeper is implemented

// ExportGenesis returns the exported genesis state as raw bytes for the chainmanager
// module.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(am.keeper.ExportGenesis(ctx))
}

// ConsensusVersion implements AppModule/ConsensusVersion.
func (AppModule) ConsensusVersion() uint64 { return ConsensusVersion }

// AppModuleSimulation functions

// TODO HV2: uncomment when simulation is implemented

// GenerateGenesisState creates a randomized GenState of the chainmanager module.
// func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
// 	simulation.RandomizedGenState(simState)
// }

// TODO HV2: this is a no-op in current heimdall. Probably no need to implement this
// looks equivalent to https://github.com/maticnetwork/heimdall/blob/249aa798c2f23c533d2421f2101127c11684c76e/chainmanager/module.go#L161C18-L161C34

// ProposalMsgs returns msgs used for governance proposals for simulations.
// func (AppModule) ProposalMsgs(simState module.SimulationState) []simtypes.WeightedProposalMsg {
// 	return simulation.ProposalMsgs()
// }

// TODO HV2: this is a no-op in current heimdall. Probably no need to implement this

// RegisterStoreDecoder registers a decoder for chainmanager module's types
// func (am AppModule) RegisterStoreDecoder(sdr simtypes.StoreDecoderRegistry) {
// 	sdr[types.StoreKey] = simtypes.NewStoreDecoderFuncFromCollectionsSchema(am.keeper.(keeper.BaseKeeper).Schema)
// }

// TODO HV2: uncomment when simulation is implemented

// RandomizedParams creates randomized param changes for the simulator.
// func (AppModule) RandomizedParams(r *rand.Rand) []simTypes.ParamChange {
// 	return simulation.ParamChanges(r)
// }

// TODO HV2: this is a no-op in current heimdall. Probably no need to implement this

// WeightedOperations returns the all the gov module operations with their respective weights.
// func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
// 	return simulation.WeightedOperations(
// 		simState.AppParams, simState.Cdc, simState.TxConfig, am.accountKeeper, am.keeper,
// 	)
// }
