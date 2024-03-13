package app

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	jsoniter "github.com/json-iterator/go"
	"github.com/spf13/cast"

	storetypes "cosmossdk.io/store/types"
	"cosmossdk.io/x/upgrade"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/std"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/consensus"
	consensusparamtypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	"github.com/cosmos/cosmos-sdk/x/distribution"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/cosmos/cosmos-sdk/x/params"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	paramproposal "github.com/cosmos/cosmos-sdk/x/params/types/proposal"

	"cosmossdk.io/client/v2/autocli"
	"cosmossdk.io/core/appmodule"
	"github.com/cosmos/cosmos-sdk/client/grpc/cmtservice"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	authcodec "github.com/cosmos/cosmos-sdk/x/auth/codec"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	paramsclient "github.com/cosmos/cosmos-sdk/x/params/client"
	"github.com/cosmos/gogoproto/proto"

	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
	reflectionv1 "cosmossdk.io/api/cosmos/reflection/v1"
	"cosmossdk.io/log"
	upgradekeeper "cosmossdk.io/x/upgrade/keeper"
	clerkkeeper "github.com/0xPolygon/heimdall-v2/x/clerk/keeper"
	abci "github.com/cometbft/cometbft/abci/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	nodeservice "github.com/cosmos/cosmos-sdk/client/grpc/node"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	runtimeservices "github.com/cosmos/cosmos-sdk/runtime/services"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/server/api"
	"github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/version"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	consensusparamkeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
)

var (
	DefaultNodeHome string
	// module account permissions
	maccPerms = map[string][]string{
		authtypes.FeeCollectorName: nil,
		govtypes.ModuleName:        {authtypes.Burner},
	}
)

var (
	_ runtime.AppI            = (*HeimdallApp)(nil)
	_ servertypes.Application = (*HeimdallApp)(nil)
)

type HeimdallApp struct {
	*baseapp.BaseApp

	legacyAmino       *codec.LegacyAmino //nolint:staticcheck
	appCodec          codec.Codec
	txConfig          client.TxConfig
	interfaceRegistry types.InterfaceRegistry

	keys    map[string]*storetypes.KVStoreKey
	tkeys   map[string]*storetypes.TransientStoreKey
	memKeys map[string]*storetypes.MemoryStoreKey

	// keepers
	AccountKeeper authkeeper.AccountKeeper
	BankKeeper    bankkeeper.Keeper
	// StakingKeeper *stakingkeeper.Keeper
	// TODO HV2: consider removing distribution module since rewards are distributed on L1
	DistrKeeper           distrkeeper.Keeper
	GovKeeper             govkeeper.Keeper
	UpgradeKeeper         *upgradekeeper.Keeper
	ParamsKeeper          paramskeeper.Keeper
	ConsensusParamsKeeper consensusparamkeeper.Keeper

	// Custom Keepers
	// TODO HV2: uncomment when implemented
	// StakeKeeper stakekeeper.Keeper
	// BorKeeper borkeeper.Keeper
	ClerkKeeper clerkkeeper.Keeper
	// CheckpointKeeper checkpointkeeper.Keeper
	// TopupKeeper topupkeeper.Keeper
	// ChainKeeper chainmanagerkeeper.Keeper

	// utility for invoking contracts in Ethereum and Bor chain
	// caller helper.ContractCaller

	mm           *module.Manager
	BasicManager module.BasicManager

	simulationManager *module.SimulationManager

	configurator module.Configurator
}

func init() {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	DefaultNodeHome = filepath.Join(userHomeDir, ".heimdalld")
}

func NewHeimdallApp(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	loadLatest bool,
	appOpts servertypes.AppOptions,
	baseAppOptions ...func(*baseapp.BaseApp),
) *HeimdallApp {
	encodingConfig := RegisterEncodingConfig()
	appCodec := encodingConfig.Marshaler
	legacyAmino := encodingConfig.Amino
	txConfig := encodingConfig.TxConfig
	interfaceRegistry := encodingConfig.InterfaceRegistry

	std.RegisterLegacyAminoCodec(legacyAmino)
	std.RegisterInterfaces(interfaceRegistry)

	bApp := baseapp.NewBaseApp(AppName, logger, db, txConfig.TxDecoder(), baseAppOptions...)
	bApp.SetCommitMultiStoreTracer(traceStore)
	bApp.SetVersion(version.Version)
	bApp.SetInterfaceRegistry(interfaceRegistry)
	bApp.SetTxEncoder(txConfig.TxEncoder())

	keys := storetypes.NewKVStoreKeys(
		authtypes.StoreKey,
		banktypes.StoreKey,
		distrtypes.StoreKey,
		govtypes.StoreKey,
		paramstypes.StoreKey,
		upgradetypes.StoreKey,
		consensusparamtypes.StoreKey,
		// TODO HV2: uncomment when implemented
		// staketypes.StoreKey,
		// bortypes.StoreKey,
		// clerktypes.StoreKey,
		// checkpointtypes.StoreKey,
		// topuptypes.StoreKey,
		// chainmanagertypes.StoreKey,
	)

	// register streaming services
	if err := bApp.RegisterStreamingServices(appOpts, keys); err != nil {
		panic(err)
	}

	tkeys := storetypes.NewTransientStoreKeys(paramstypes.TStoreKey)
	// memKeys := storetypes.NewMemoryStoreKeys(capabilitytypes.MemStoreKey, ibcmock.MemStoreKey)

	app := &HeimdallApp{
		BaseApp:           bApp,
		legacyAmino:       legacyAmino,
		txConfig:          txConfig,
		appCodec:          appCodec,
		interfaceRegistry: interfaceRegistry,
		keys:              keys,
		tkeys:             tkeys,
		// memKeys:           memKeys,
	}

	// Contract caller
	// TODO HV2: uncomment when implemented

	// contractCallerObj, err := helper.NewContractCaller()
	// if err != nil {
	// 	cmn.Exit(err.Error())
	// }

	// app.caller = contractCallerObj

	// proposalHandler := abci.NewProposalHandler(logger, txConfig)
	// voteExtHandler := abci.NewVoteExtensionHandler(logger, randProvider)

	// Set ABCI++ Handlers
	// bApp.SetPrepareProposal(proposalHandler.PrepareProposalHandler())
	// bApp.SetProcessProposal(proposalHandler.ProcessProposalHandler())
	// bApp.SetExtendVoteHandler(voteExtHandler.ExtendVoteHandler())

	app.ParamsKeeper = initParamsKeeper(appCodec, legacyAmino, keys[paramstypes.StoreKey], tkeys[paramstypes.TStoreKey])

	moduleAccountAddresses := app.ModuleAccountAddrs()
	blockedAddr := app.BlockedModuleAccountAddrs(moduleAccountAddresses)

	// set the BaseApp's parameter store
	app.ConsensusParamsKeeper = consensusparamkeeper.NewKeeper(appCodec, runtime.NewKVStoreService(keys[consensusparamtypes.StoreKey]), authtypes.NewModuleAddress(govtypes.ModuleName).String(), runtime.EventService{})
	bApp.SetParamStore(app.ConsensusParamsKeeper.ParamsStore)

	// SDK module keepers

	// add keepers
	app.AccountKeeper = authkeeper.NewAccountKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[authtypes.StoreKey]),
		authtypes.ProtoBaseAccount,
		maccPerms,
		authcodec.NewBech32Codec(sdk.Bech32MainPrefix),
		sdk.Bech32MainPrefix,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.BankKeeper = bankkeeper.NewBaseKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[banktypes.StoreKey]),
		app.AccountKeeper,
		blockedAddr,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		logger,
	)

	// TODO HV2: initialise stake keeper here

	// TODO HV2: consider removing distribution module since rewards are distributed on L1
	app.DistrKeeper = distrkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[distrtypes.StoreKey]),
		app.AccountKeeper,
		app.BankKeeper,
		nil, // TODO HV2: pass stake keeper here once implemented
		authtypes.FeeCollectorName,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	govRouter := govv1beta1.NewRouter()
	govRouter.AddRoute(govtypes.RouterKey, govv1beta1.ProposalHandler).
		AddRoute(paramproposal.RouterKey, params.NewParamChangeProposalHandler(app.ParamsKeeper))

	govConfig := govtypes.DefaultConfig()

	govKeeper := govkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[govtypes.StoreKey]),
		app.AccountKeeper,
		app.BankKeeper,
		nil, // TODO HV2: add our modified stake keeper as the param
		app.DistrKeeper,
		app.MsgServiceRouter(),
		govConfig,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	// Set legacy router for backwards compatibility with gov v1beta1
	govKeeper.SetLegacyRouter(govRouter)

	app.GovKeeper = *govKeeper.SetHooks(
		govtypes.NewMultiGovHooks(
		// register the governance hooks
		),
	)

	// custom keepers
	// TODO HV2: initialize custom module keepers

	skipUpgradeHeights := map[int64]bool{}
	for _, h := range cast.ToIntSlice(appOpts.Get(server.FlagUnsafeSkipUpgrades)) {
		skipUpgradeHeights[int64(h)] = true
	}
	homePath := cast.ToString(appOpts.Get(flags.FlagHome))
	app.UpgradeKeeper = upgradekeeper.NewKeeper(
		skipUpgradeHeights,
		runtime.NewKVStoreService(keys[upgradetypes.StoreKey]),
		appCodec,
		homePath,
		app.BaseApp,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.mm = module.NewManager(
		// TODO HV2: add stake keeper once implemented
		// genutil.NewAppModule(app.AccountKeeper, app.StakeKeeper, app, txConfig),
		auth.NewAppModule(appCodec, app.AccountKeeper, nil, app.GetSubspace(authtypes.ModuleName)),
		bank.NewAppModule(appCodec, app.BankKeeper, app.AccountKeeper, app.GetSubspace(banktypes.ModuleName)),
		gov.NewAppModule(appCodec, &app.GovKeeper, app.AccountKeeper, app.BankKeeper, app.GetSubspace(govtypes.ModuleName)),
		// TODO HV2: consider removing distribution module since rewards are distributed on L1
		distribution.NewAppModule(appCodec, app.DistrKeeper, app.AccountKeeper, app.BankKeeper, nil, app.GetSubspace(distrtypes.ModuleName)),
		// TODO HV2: replace with our stake module
		// staking.NewAppModule(appCodec, app.StakingKeeper, app.AccountKeeper, app.BankKeeper, app.GetSubspace(stakingtypes.ModuleName)),
		upgrade.NewAppModule(app.UpgradeKeeper, app.AccountKeeper.AddressCodec()),
		params.NewAppModule(app.ParamsKeeper),
		consensus.NewAppModule(appCodec, app.ConsensusParamsKeeper),
		// TODO HV2: add custom modules
	)

	// Basic manager
	app.BasicManager = module.NewBasicManagerFromManager(
		app.mm,
		map[string]module.AppModuleBasic{
			genutiltypes.ModuleName: genutil.NewAppModuleBasic(genutiltypes.DefaultMessageValidator),
			govtypes.ModuleName: gov.NewAppModuleBasic(
				[]govclient.ProposalHandler{
					paramsclient.ProposalHandler,
				},
			),
		})

	app.BasicManager.RegisterLegacyAminoCodec(legacyAmino)
	app.BasicManager.RegisterInterfaces(interfaceRegistry)

	app.mm.SetOrderBeginBlockers(
		upgradetypes.ModuleName,
		// TODO HV2: consider removing distribution module since rewards are distributed on L1
		distrtypes.ModuleName,
		// TODO HV2: stakingtypes.ModuleName, replace with our stake module
		genutiltypes.ModuleName,
	)

	// NOTE: upgrade module is required to be prioritized
	app.mm.SetOrderPreBlockers(
		upgradetypes.ModuleName,
	)

	app.mm.SetOrderEndBlockers(
		govtypes.ModuleName,
		// TODO HV2: replace with our stake module
		// stakingtypes.ModuleName,
		genutiltypes.ModuleName,
	)

	genesisModuleOrder := []string{
		authtypes.ModuleName,
		banktypes.ModuleName,
		distrtypes.ModuleName, // TODO HV2: consider removing distribution module since rewards are distributed on L1
		govtypes.ModuleName,
		genutiltypes.ModuleName,
		upgradetypes.ModuleName,
		consensusparamtypes.ModuleName,
		// TODO HV2: uncomment when implemented
		// staketypes.ModuleName,
		// checkpointtypes.ModuleName,
		// bortypes.ModuleName,
		// clerktypes.ModuleName,
		// topuptypes.ModuleName,
		// chainmanagertypes.ModuleName,

	}

	app.mm.SetOrderInitGenesis(genesisModuleOrder...)
	app.mm.SetOrderExportGenesis(genesisModuleOrder...)

	app.configurator = module.NewConfigurator(
		app.appCodec,
		app.MsgServiceRouter(),
		app.GRPCQueryRouter(),
	)
	err := app.mm.RegisterServices(app.configurator)
	if err != nil {
		panic(err)
	}

	autocliv1.RegisterQueryServer(app.GRPCQueryRouter(), runtimeservices.NewAutoCLIQueryService(app.mm.Modules))

	reflectionSvc, err := runtimeservices.NewReflectionService()
	if err != nil {
		panic(err)
	}
	reflectionv1.RegisterReflectionServiceServer(app.GRPCQueryRouter(), reflectionSvc)

	testdata.RegisterQueryServer(app.GRPCQueryRouter(), testdata.QueryImpl{})

	// initialize stores
	app.MountKVStores(keys)
	app.MountTransientStores(tkeys)
	// app.MountMemoryStores(memKeys)
	// <Upgrade handler setup here>
	// initialize BaseApp
	app.SetInitChainer(app.InitChainer)
	app.SetPreBlocker(app.PreBlocker)
	app.SetBeginBlocker(app.BeginBlocker)
	app.SetEndBlocker(app.EndBlocker)
	app.setAnteHandler(txConfig)

	// At startup, after all modules have been registered, check that all prot
	// annotations are correct.
	protoFiles, err := proto.MergedRegistry()
	if err != nil {
		panic(err)
	}
	err = msgservice.ValidateProtoAnnotations(protoFiles)
	if err != nil {
		// Once we switch to using protoreflect-based antehandlers, we might
		// want to panic here instead of logging a warning.
		_, err := fmt.Fprintln(os.Stderr, err.Error())
		if err != nil {
			fmt.Println("could not write to stderr")
		}
	}

	if loadLatest {
		if err := app.LoadLatestVersion(); err != nil {
			panic(fmt.Errorf("error loading last version: %w", err))
		}
	}

	return app
}

func (app *HeimdallApp) setAnteHandler(txConfig client.TxConfig) {
	// TODO HV2: pass contract caller and keepers for chainmanager and distribution
	// see https://github.com/0xPolygon/heimdall-v2/commit/ea3bc8efd52d43bd620d51c317e2e1b1afd908f7
	// https://github.com/0xPolygon/heimdall-v2/commit/5ce56fb60634211798b32745358adfa8fd1bbbc5
	anteHandler, err := NewAnteHandler(
		HandlerOptions{
			ante.HandlerOptions{
				AccountKeeper:   app.AccountKeeper,
				BankKeeper:      app.BankKeeper,
				SignModeHandler: txConfig.SignModeHandler(),
				SigGasConsumer:  ante.DefaultSigVerificationGasConsumer,
			},
		},
	)
	if err != nil {
		panic(err)
	}

	// Set the AnteHandler for the app
	app.SetAnteHandler(anteHandler)
}

func (app *HeimdallApp) Name() string { return app.BaseApp.Name() }

// InitChainer application update at chain initialization
func (app *HeimdallApp) InitChainer(ctx sdk.Context, req *abci.RequestInitChain) (*abci.ResponseInitChain, error) {
	var genesisState GenesisState
	if err := jsoniter.ConfigFastest.Unmarshal(req.AppStateBytes, &genesisState); err != nil {
		panic(err)
	}
	app.UpgradeKeeper.SetModuleVersionMap(ctx, app.mm.GetVersionMap()) //nolint:errcheck

	// get validator updates
	if err := app.BasicManager.ValidateGenesis(app.AppCodec(), app.txConfig, genesisState); err != nil {
		panic(err)
	}

	// check fee collector module account
	if moduleAcc := app.AccountKeeper.GetModuleAccount(ctx, authtypes.FeeCollectorName); moduleAcc == nil {
		panic(fmt.Sprintf("%s module account has not been set", authtypes.FeeCollectorName))
	}

	// init genesis
	if _, err := app.mm.InitGenesis(ctx, app.AppCodec(), genesisState); err != nil {
		return &abci.ResponseInitChain{}, err
	}

	// TODO HV2: consider moving the validator set update logic to staking module's InitGenesis
	// stakingState := stakingTypes.GetGenesisStateFromAppState(genesisState)
	// checkpointState := checkpointTypes.GetGenesisStateFromAppState(genesisState)

	// // check if validator is current validator
	// // add to val updates else skip
	// var valUpdates []abci.ValidatorUpdate

	// for _, validator := range stakingState.Validators {
	// 	if validator.IsCurrentValidator(checkpointState.AckCount) {
	// 		// convert to Validator Update
	// 		updateVal := abci.ValidatorUpdate{
	// 			Power:  validator.VotingPower,
	// 			PubKey: validator.PubKey.ABCIPubKey(),
	// 		}
	// 		// Add validator to validator updated to be processed below
	// 		valUpdates = append(valUpdates, updateVal)
	// 	}
	// }

	// TODO: make sure old validtors don't go in validator updates i.e. deactivated validators have to be removed
	// update validators
	return &abci.ResponseInitChain{
		// validator updates
		// Validators: valUpdates,
	}, nil
}

// PreBlocker application updates every pre block
func (app *HeimdallApp) PreBlocker(ctx sdk.Context, req *abci.RequestFinalizeBlock) (*sdk.ResponsePreBlock, error) {
	// TODO HV2: Implement VE processing logic here

	return app.mm.PreBlock(ctx)
}

// BeginBlocker application updates every begin block
func (app *HeimdallApp) BeginBlocker(ctx sdk.Context) (sdk.BeginBlock, error) {
	// TODO HV2: implement
	// app.AccountKeeper.SetBlockProposer(
	// 	ctx,
	// 	types.BytesToHeimdallAddress(req.Header.GetProposerAddress()),
	// )
	return app.mm.BeginBlock(ctx)
}

// EndBlocker application updates every end block
func (app *HeimdallApp) EndBlocker(ctx sdk.Context) (sdk.EndBlock, error) {
	// TODO HV2: consider moving the validator set update logic to staking module's EndBlock
	// under x/staking/module.go

	// transfer fees to current proposer
	// if proposer, ok := app.AccountKeeper.GetBlockProposer(ctx); ok {
	// 	moduleAccount := app.AccountKeeper.GetModuleAccount(ctx, authtypes.FeeCollectorName)
	// 	amount := moduleAccount.GetCoins().AmountOf(authTypes.FeeToken)
	// 	if !amount.IsZero() {
	// 		coins := sdk.Coins{sdk.Coin{Denom: authtypes.FeeToken, Amount: amount}}
	// 		if err := app.SupplyKeeper.SendCoinsFromModuleToAccount(ctx, authTypes.FeeCollectorName, proposer, coins); err != nil {
	// 			logger.Error("EndBlocker | SendCoinsFromModuleToAccount", "Error", err)
	// 		}
	// 	}
	// 	// remove block proposer
	// 	app.AccountKeeper.RemoveBlockProposer(ctx)
	// }

	// var tmValUpdates []abci.ValidatorUpdate

	// // --- Start update to new validators
	// currentValidatorSet := app.StakingKeeper.GetValidatorSet(ctx)
	// allValidators := app.StakingKeeper.GetAllValidators(ctx)
	// ackCount := app.CheckpointKeeper.GetACKCount(ctx)

	// // get validator updates
	// setUpdates := helper.GetUpdatedValidators(
	// 	&currentValidatorSet, // pointer to current validator set -- UpdateValidators will modify it
	// 	allValidators,        // All validators
	// 	ackCount,             // ack count
	// )

	// if len(setUpdates) > 0 {
	// 	// create new validator set
	// 	if err := currentValidatorSet.UpdateWithChangeSet(setUpdates); err != nil {
	// 		// return with nothing
	// 		logger.Error("Unable to update current validator set", "Error", err)
	// 		return abci.ResponseEndBlock{}
	// 	}

	// 	//Hardfork to remove the rotation of validator list on stake update
	// 	if ctx.BlockHeight() < helper.GetAalborgHardForkHeight() {
	// 		// increment proposer priority
	// 		currentValidatorSet.IncrementProposerPriority(1)
	// 	}

	// 	// validator set change
	// 	logger.Debug("[ENDBLOCK] Updated current validator set", "proposer", currentValidatorSet.GetProposer())

	// 	// save set in store
	// 	if err := app.StakingKeeper.UpdateValidatorSetInStore(ctx, currentValidatorSet); err != nil {
	// 		// return with nothing
	// 		logger.Error("Unable to update current validator set in state", "Error", err)
	// 		return abci.ResponseEndBlock{}
	// 	}

	// 	// convert updates from map to array
	// 	for _, v := range setUpdates {
	// 		tmValUpdates = append(tmValUpdates, abci.ValidatorUpdate{
	// 			Power:  v.VotingPower,
	// 			PubKey: v.PubKey.ABCIPubKey(),
	// 		})
	// 	}
	// }

	// TODO HV2: consider moving the rootchain contract address update logic to chainmanager's EndBlock()
	// under x/chainmanager/module.go

	// // Change root chain contract addresses if required
	// if chainManagerAddressMigration, found := helper.GetChainManagerAddressMigration(ctx.BlockHeight()); found {
	// 	params := app.ChainKeeper.GetParams(ctx)

	// 	params.ChainParams.MaticTokenAddress = chainManagerAddressMigration.MaticTokenAddress
	// 	params.ChainParams.StakingManagerAddress = chainManagerAddressMigration.StakingManagerAddress
	// 	params.ChainParams.RootChainAddress = chainManagerAddressMigration.RootChainAddress
	// 	params.ChainParams.SlashManagerAddress = chainManagerAddressMigration.SlashManagerAddress
	// 	params.ChainParams.StakingInfoAddress = chainManagerAddressMigration.StakingInfoAddress
	// 	params.ChainParams.StateSenderAddress = chainManagerAddressMigration.StateSenderAddress

	// 	// update chain manager state
	// 	app.ChainKeeper.SetParams(ctx, params)
	// 	logger.Info("Updated chain manager state", "params", params)
	// }

	return app.mm.EndBlock(ctx)
}

func (app *HeimdallApp) LoadHeight(height int64) error {
	return app.LoadVersion(height)
}

func (app *HeimdallApp) ModuleAccountAddrs() map[string]bool {
	modAccAddrs := make(map[string]bool)
	for acc := range maccPerms {
		modAccAddrs[authtypes.NewModuleAddress(acc).String()] = true
	}

	return modAccAddrs
}

func (app *HeimdallApp) BlockedModuleAccountAddrs(modAccAddrs map[string]bool) map[string]bool {
	delete(modAccAddrs, authtypes.NewModuleAddress(govtypes.ModuleName).String())
	// TODO HV2: add topup module to enable it to receive/send tokens.
	// See https://github.com/0xPolygon/cosmos-sdk/pull/5#discussion_r1513037980
	return modAccAddrs
}

func (app *HeimdallApp) LegacyAmino() *codec.LegacyAmino {
	return app.legacyAmino
}

func (app *HeimdallApp) AppCodec() codec.Codec {
	return app.appCodec
}

func (app *HeimdallApp) InterfaceRegistry() types.InterfaceRegistry {
	return app.interfaceRegistry
}

func (app *HeimdallApp) GetTxConfig() client.TxConfig {
	return app.txConfig
}

// AutoCliOpts returns the autocli options for the app.
func (app *HeimdallApp) AutoCliOpts() autocli.AppOptions {
	modules := make(map[string]appmodule.AppModule, 0)
	for _, m := range app.mm.Modules {
		if moduleWithName, ok := m.(module.HasName); ok {
			moduleName := moduleWithName.Name()
			if appModule, ok := moduleWithName.(appmodule.AppModule); ok {
				modules[moduleName] = appModule
			}
		}
	}

	return autocli.AppOptions{
		Modules:       modules,
		ModuleOptions: runtimeservices.ExtractAutoCLIOptions(app.mm.Modules),
		// TODO HV2: replace with authcodec.hexCodec once https://github.com/0xPolygon/cosmos-sdk/pull/3 is merged
		AddressCodec:          authcodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix()),
		ValidatorAddressCodec: authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix()),
		ConsensusAddressCodec: authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ConsensusAddrPrefix()),
	}
}

// DefaultGenesis returns a default genesis from the registered AppModuleBasic's.
func (app *HeimdallApp) DefaultGenesis() map[string]json.RawMessage {
	return app.BasicManager.DefaultGenesis(app.appCodec)
}

// GetKey returns the KVStoreKey for the provided store key.
//
// NOTE: This is solely to be used for testing purposes.
func (app *HeimdallApp) GetKey(storeKey string) *storetypes.KVStoreKey {
	return app.keys[storeKey]
}

// GetTKey returns the TransientStoreKey for the provided store key.
//
// NOTE: This is solely to be used for testing purposes.
func (app *HeimdallApp) GetTKey(storeKey string) *storetypes.TransientStoreKey {
	return app.tkeys[storeKey]
}

// GetStoreKeys returns all the stored store keys.
func (app *HeimdallApp) GetStoreKeys() []storetypes.StoreKey {
	keys := make([]storetypes.StoreKey, 0, len(app.keys))
	for _, key := range app.keys {
		keys = append(keys, key)
	}

	return keys
}

// SimulationManager implements the SimulationApp interface
func (app *HeimdallApp) SimulationManager() *module.SimulationManager {
	return app.simulationManager
}

func (app *HeimdallApp) RegisterAPIRoutes(apiSvr *api.Server, apiConfig config.APIConfig) {
	clientCtx := apiSvr.ClientCtx
	// Register new tx routes from grpc-gateway.
	authtx.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// Register new CometBFT queries routes from grpc-gateway.
	cmtservice.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// Register node gRPC service for grpc-gateway.
	nodeservice.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// Register grpc-gateway routes for all modules.
	app.BasicManager.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// register swagger API from root so that other applications can override easily
	if err := server.RegisterSwaggerAPI(apiSvr.ClientCtx, apiSvr.Router, apiConfig.Swagger); err != nil {
		panic(err)
	}
}

func (app *HeimdallApp) RegisterTxService(clientCtx client.Context) {
	authtx.RegisterTxService(app.BaseApp.GRPCQueryRouter(), clientCtx, app.BaseApp.Simulate, app.interfaceRegistry)
}

// RegisterTendermintService implements the Application.RegisterTendermintService method.
func (app *HeimdallApp) RegisterTendermintService(clientCtx client.Context) {
	cmtApp := server.NewCometABCIWrapper(app)
	cmtservice.RegisterTendermintService(
		clientCtx,
		app.BaseApp.GRPCQueryRouter(),
		app.interfaceRegistry,
		cmtApp.Query,
	)
}

func (app *HeimdallApp) RegisterNodeService(clientCtx client.Context, cfg config.Config) {
	nodeservice.RegisterNodeService(clientCtx, app.GRPCQueryRouter(), cfg)
}

func (app *HeimdallApp) OnTxSucceeded(_ sdk.Context, _, _ string, _ []byte, _ []byte) {
}

func (app *HeimdallApp) OnTxFailed(_ sdk.Context, _, _ string, _ []byte, _ []byte) {
}

func (app *HeimdallApp) GetBaseApp() *baseapp.BaseApp {
	return app.BaseApp
}

type EmptyAppOptions struct{}

func (ao EmptyAppOptions) Get(_ string) interface{} {
	return nil
}

// initParamsKeeper init params keeper and its subspaces
func initParamsKeeper(appCodec codec.BinaryCodec, legacyAmino *codec.LegacyAmino, key, tkey storetypes.StoreKey) paramskeeper.Keeper {
	paramsKeeper := paramskeeper.NewKeeper(appCodec, legacyAmino, key, tkey)

	return paramsKeeper
}

// GetSubspace returns a param subspace for a given module name.
//
// NOTE: This is solely to be used for testing purposes.
func (app *HeimdallApp) GetSubspace(moduleName string) paramstypes.Subspace {
	subspace, _ := app.ParamsKeeper.GetSubspace(moduleName)
	return subspace
}

func (app *HeimdallApp) GetMemKey(storeKey string) *storetypes.MemoryStoreKey {
	return app.memKeys[storeKey]
}

// GetMaccPerms returns a copy of the module account permissions
func GetMaccPerms() map[string][]string {
	dupMaccPerms := make(map[string][]string)
	for k, v := range maccPerms {
		dupMaccPerms[k] = v
	}

	return dupMaccPerms
}
