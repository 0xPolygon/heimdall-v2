package clerk

import (
	"context"
	"encoding/json"
	"fmt"

	gwruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/0xPolygon/heimdall-v2/x/clerk/client/cli"
	"github.com/0xPolygon/heimdall-v2/x/clerk/keeper"
	"github.com/0xPolygon/heimdall-v2/x/clerk/types"

	"github.com/spf13/cobra"
)

// TODO HV2 - check consensus version
// ConsensusVersion defines the current x/clerk module consensus version.
const ConsensusVersion = 1

var (
	_ module.AppModule      = AppModule{}
	_ module.AppModuleBasic = AppModule{}
	// _ module.AppModuleSimulation = AppModule{}
	_ module.HasGenesis  = AppModule{}
	_ module.HasServices = AppModule{}
	// TODO HV2 - check if we need to add invariants
	// _ module.HasInvariants       = AppModule{}

)

// AppModuleBasic defines the basic application module used by the clerk module.
type AppModuleBasic struct {
	cdc *codec.Codec
}

// Name returns the clerk module's name.
func (AppModuleBasic) Name() string {
	return types.ModuleName
}

// RegisterLegacyAminoCodec registers the clerk module's types for the given codec.
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	types.RegisterLegacyAminoCodec(cdc)
}

// DefaultGenesis returns default genesis state as raw bytes for the clerk module.
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	// TODO HV2 - fix the error
	return cdc.MustMarshalJSON(types.DefaultGenesisState())
}

// ValidateGenesis performs genesis state validation for the clerk module.
func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, config client.TxEncodingConfig, bz json.RawMessage) error {
	// TODO HV2 - fix the errors
	var data types.GenesisState
	if err := cdc.UnmarshalJSON(bz, &data); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", types.ModuleName, err)
	}

	// TODO HV2 - which Validate to use?
	// `types.ValidateGenesis(data)` or `data.Validate()`
	return types.ValidateGenesis(data)
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the clerk module.
func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *gwruntime.ServeMux) {
	// TODO HV2 - fix the errors
	if err := types.RegisterQueryHandlerClient(context.Background(), mux, types.NewQueryClient(clientCtx)); err != nil {
		panic(err)
	}
}

// RegisterInterfaces registers interfaces and implementations of the clerk module.
func (AppModuleBasic) RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	// TODO HV2 - fix the error
	types.RegisterInterfaces(registry)
}

// GetTxCmd returns the root tx command for the clerk module.
func (ab AppModuleBasic) GetTxCmd() *cobra.Command {
	// TODO HV2 - write cli module and implement GetTxCmd()
	return cli.GetTxCmd(ab.cdc)
}

// GetQueryCmd returns the root query command for the auth module.
func (ab AppModuleBasic) GetQueryCmd(cdc *codec.Codec) *cobra.Command {
	return cli.GetQueryCmd(cdc)
}

// AppModule implements an application module for the clerk module.
type AppModule struct {
	AppModuleBasic

	keeper keeper.Keeper
	// TODO HV2 - uncomment when we have the contractCaller
	// contractCaller helper.IContractCaller
}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (am AppModule) IsOnePerModuleType() {}

// IsAppModule implements the appmodule.AppModule interface.
func (am AppModule) IsAppModule() {}

// RegisterServices registers module services.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	types.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(am.keeper))
	// querier := keeper.Querier{Keeper: am.keeper}
	// TODO HV2 - pass the querier to RegisterQueryServer
	types.RegisterQueryServer(cfg.QueryServer(), nil)

	// TODO HV2 - pass the querier to NewMigrator
	_ = keeper.NewMigrator(am.keeper, nil)

	// TODO HV2 - pass the migrator to RegisterMigration
	if err := cfg.RegisterMigration(types.ModuleName, 1, nil); err != nil {
		panic(fmt.Sprintf("failed to migrate x/clerk from version 1 to 2: %v", err))
	}
}

// NewAppModule creates a new AppModule object
// TODO HV2 - uncomment when we have the contractCaller
// func NewAppModule(cdc codec.Codec, keeper *keeper.Keeper, contractCaller helper.IContractCaller) AppModule {
func NewAppModule(cdc codec.Codec, keeper keeper.Keeper) AppModule {
	return AppModule{
		AppModuleBasic: AppModuleBasic{cdc: &cdc},
		keeper:         keeper,
		// TODO HV2 - uncomment when we have the contractCaller
		// contractCaller: helper.IContractCaller,
	}
}

// TODO HV2 - check if we need to add invariants
// // RegisterInvariants registers the clerk module invariants.
// func (am AppModule) RegisterInvariants(ir sdk.InvariantRegistry) {
// 	keeper.RegisterInvariants(ir, am.keeper)
// }

// QuerierRoute returns the clerk module's querier route name.
func (AppModule) QuerierRoute() string { return types.RouterKey }

// InitGenesis performs genesis initialization for the clerk module. It returns
// no validator updates.
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

// ConsensusVersion implements AppModule/ConsensusVersion.
func (AppModule) ConsensusVersion() uint64 { return ConsensusVersion }

// TODO HV2 - I believe we dont need the following simulation functions for clerk module
// AppModuleSimulation functions

// // GenerateGenesisState creates a randomized GenState of the clerk module.
// func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
// 	simulation.RandomizedGenState(simState)
// }

// // ProposalMsgs returns msgs used for governance proposals for simulations.
// func (AppModule) ProposalMsgs(simState module.SimulationState) []simtypes.WeightedProposalMsg {
// 	return simulation.ProposalMsgs()
// }

// // RegisterStoreDecoder registers a decoder for supply module's types
// func (am AppModule) RegisterStoreDecoder(sdr simtypes.StoreDecoderRegistry) {
// 	// TODO HV2 - what to do of Schema?
// 	sdr[types.StoreKey] = simtypes.NewStoreDecoderFuncFromCollectionsSchema(am.keeper.Schema)
// }

// // WeightedOperations doesn't return any clerk module operation.
// func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
// 	return nil
// }
