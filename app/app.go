package app

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
	reflectionv1 "cosmossdk.io/api/cosmos/reflection/v1"
	"cosmossdk.io/client/v2/autocli"
	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	"cosmossdk.io/x/tx/signing"
	abci "github.com/cometbft/cometbft/abci/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/grpc/cmtservice"
	nodeservice "github.com/cosmos/cosmos-sdk/client/grpc/node"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	runtimeservices "github.com/cosmos/cosmos-sdk/runtime/services"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/server/api"
	"github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/cosmos/cosmos-sdk/std"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authcodec "github.com/cosmos/cosmos-sdk/x/auth/codec"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/consensus"
	consensusparamkeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	consensusparamtypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/cosmos/cosmos-sdk/x/params"
	paramsclient "github.com/cosmos/cosmos-sdk/x/params/client"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	paramproposal "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	"github.com/cosmos/gogoproto/proto"

	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/sidetxs"
	"github.com/0xPolygon/heimdall-v2/x/chainmanager"
	chainmanagerkeeper "github.com/0xPolygon/heimdall-v2/x/chainmanager/keeper"
	chainmanagertypes "github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
	"github.com/0xPolygon/heimdall-v2/x/checkpoint"
	checkpointKeeper "github.com/0xPolygon/heimdall-v2/x/checkpoint/keeper"
	checkpointTypes "github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
	"github.com/0xPolygon/heimdall-v2/x/clerk"
	clerkkeeper "github.com/0xPolygon/heimdall-v2/x/clerk/keeper"
	clerktypes "github.com/0xPolygon/heimdall-v2/x/clerk/types"
	"github.com/0xPolygon/heimdall-v2/x/milestone"
	milestoneKeeper "github.com/0xPolygon/heimdall-v2/x/milestone/keeper"
	milestoneTypes "github.com/0xPolygon/heimdall-v2/x/milestone/types"
	"github.com/0xPolygon/heimdall-v2/x/stake"
	stakeKeeper "github.com/0xPolygon/heimdall-v2/x/stake/keeper"
	staketypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
	"github.com/0xPolygon/heimdall-v2/x/topup"
	topupKeeper "github.com/0xPolygon/heimdall-v2/x/topup/keeper"
	topupTypes "github.com/0xPolygon/heimdall-v2/x/topup/types"
)

var (
	DefaultNodeHome string
	// maccPerms represent the module accounts' permissions
	maccPerms = map[string][]string{
		authtypes.FeeCollectorName: nil,
		govtypes.ModuleName:        nil,
		topupTypes.ModuleName:      {authtypes.Minter, authtypes.Burner},
	}
)

var (
	_ runtime.AppI            = (*HeimdallApp)(nil)
	_ servertypes.Application = (*HeimdallApp)(nil)
)

type HeimdallApp struct {
	*baseapp.BaseApp

	legacyAmino       *codec.LegacyAmino
	appCodec          codec.Codec
	txConfig          client.TxConfig
	interfaceRegistry types.InterfaceRegistry

	keys    map[string]*storetypes.KVStoreKey
	tKeys   map[string]*storetypes.TransientStoreKey
	memKeys map[string]*storetypes.MemoryStoreKey

	// keepers
	AccountKeeper         authkeeper.AccountKeeper
	BankKeeper            bankkeeper.Keeper
	GovKeeper             govkeeper.Keeper
	ParamsKeeper          paramskeeper.Keeper
	ConsensusParamsKeeper consensusparamkeeper.Keeper

	// Custom Keepers
	ClerkKeeper        clerkkeeper.Keeper
	StakeKeeper        stakeKeeper.Keeper
	TopupKeeper        topupKeeper.Keeper
	ChainManagerKeeper chainmanagerkeeper.Keeper
	CheckpointKeeper   checkpointKeeper.Keeper
	MilestoneKeeper    milestoneKeeper.Keeper
	// TODO HV2: uncomment when bor module is implemented
	// BorKeeper borkeeper.Keeper

	// utility for invoking contracts in Ethereum and Bor chain
	caller helper.ContractCaller

	mm           *module.Manager
	BasicManager module.BasicManager

	simulationManager *module.SimulationManager

	configurator module.Configurator

	// Vote Extension handler
	VoteExtensionProcessor *VoteExtensionProcessor
}

func init() {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	DefaultNodeHome = filepath.Join(userHomeDir, "/var/lib/heimdall")
}

func NewHeimdallApp(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	loadLatest bool,
	appOpts servertypes.AppOptions,
	baseAppOptions ...func(*baseapp.BaseApp),
) *HeimdallApp {

	legacyAmino := codec.NewLegacyAmino()
	interfaceRegistry, err := types.NewInterfaceRegistryWithOptions(types.InterfaceRegistryOptions{
		ProtoFiles: proto.HybridResolver,
		SigningOptions: signing.Options{
			AddressCodec:          address.HexCodec{},
			ValidatorAddressCodec: address.HexCodec{},
		},
	})
	if err != nil {
		panic(err)
	}
	appCodec := codec.NewProtoCodec(interfaceRegistry)
	txConfig := authtx.NewTxConfig(appCodec, authtx.DefaultSignModes)

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
		consensusparamtypes.StoreKey,
		govtypes.StoreKey,
		paramstypes.StoreKey,
		clerktypes.StoreKey,
		staketypes.StoreKey,
		checkpointTypes.StoreKey,
		topupTypes.StoreKey,
		chainmanagertypes.StoreKey,
		milestoneTypes.StoreKey,
		// TODO HV2: uncomment when bor module is implemented
		// bortypes.StoreKey,
	)

	// register streaming services
	if err := bApp.RegisterStreamingServices(appOpts, keys); err != nil {
		panic(err)
	}

	tKeys := storetypes.NewTransientStoreKeys(paramstypes.TStoreKey)
	// TODO HV2: are memKeys needed?
	// memKeys := storetypes.NewMemoryStoreKeys(capabilitytypes.MemStoreKey, ibcmock.MemStoreKey)

	app := &HeimdallApp{
		BaseApp:           bApp,
		legacyAmino:       legacyAmino,
		txConfig:          txConfig,
		appCodec:          appCodec,
		interfaceRegistry: interfaceRegistry,
		keys:              keys,
		tKeys:             tKeys,
		// memKeys:        memKeys,
	}

	// Contract caller
	contractCallerObj, err := helper.NewContractCaller()
	if err != nil {
		panic(err)
	}

	app.caller = contractCallerObj

	moduleAccountAddresses := app.ModuleAccountAddrs()
	blockedAddr := app.BlockedModuleAccountAddrs(moduleAccountAddresses)

	// Set ABCI++ Handlers
	bApp.SetPrepareProposal(app.NewPrepareProposalHandler())
	bApp.SetProcessProposal(app.NewProcessProposalHandler())

	// set the BaseApp's parameter store
	app.ParamsKeeper = initParamsKeeper(appCodec, legacyAmino, keys[paramstypes.StoreKey], tKeys[paramstypes.TStoreKey])
	app.ConsensusParamsKeeper = consensusparamkeeper.NewKeeper(appCodec, runtime.NewKVStoreService(keys[consensusparamtypes.StoreKey]), authtypes.NewModuleAddress(govtypes.ModuleName).String(), runtime.EventService{})
	bApp.SetParamStore(app.ConsensusParamsKeeper.ParamsStore)

	// sdk module keepers

	app.AccountKeeper = authkeeper.NewAccountKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[authtypes.StoreKey]),
		authtypes.ProtoBaseAccount,
		maccPerms,
		authcodec.NewHexCodec(),
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

	govRouter := govv1beta1.NewRouter()
	govRouter.AddRoute(govtypes.RouterKey, govv1beta1.ProposalHandler).
		AddRoute(paramproposal.RouterKey, params.NewParamChangeProposalHandler(app.ParamsKeeper))
	govConfig := govtypes.DefaultConfig()
	govKeeper := govkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[govtypes.StoreKey]),
		app.AccountKeeper,
		app.BankKeeper,
		&app.StakeKeeper,
		nil,
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

	app.ChainManagerKeeper = chainmanagerkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[chainmanagertypes.StoreKey]),
		authtypes.NewModuleAddress(govtypes.ModuleName).String())

	app.StakeKeeper = stakeKeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[staketypes.StoreKey]),
		app.CheckpointKeeper,
		app.BankKeeper,
		app.ChainManagerKeeper,
		address.HexCodec{},
		&app.caller,
	)

	app.ClerkKeeper = clerkkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[clerktypes.StoreKey]),
		app.ChainManagerKeeper,
		&app.caller,
	)

	app.TopupKeeper = topupKeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[topupTypes.StoreKey]),
		app.BankKeeper,
		app.ChainManagerKeeper,
		&app.caller,
	)

	app.CheckpointKeeper = checkpointKeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[checkpointTypes.StoreKey]),
		&app.StakeKeeper,
		app.ChainManagerKeeper,
		&app.TopupKeeper,
		&app.caller,
	)

	app.MilestoneKeeper = milestoneKeeper.NewKeeper(
		appCodec,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		runtime.NewKVStoreService(keys[topupTypes.StoreKey]),
		&app.StakeKeeper,
		&app.caller,
	)

	// TODO HV2: add bor module keeper here

	app.mm = module.NewManager(
		genutil.NewAppModule(app.AccountKeeper, app.StakeKeeper, app, txConfig),
		auth.NewAppModule(appCodec, app.AccountKeeper, nil, app.GetSubspace(authtypes.ModuleName)),
		bank.NewAppModule(appCodec, app.BankKeeper, app.AccountKeeper, app.GetSubspace(banktypes.ModuleName)),
		gov.NewAppModule(appCodec, &app.GovKeeper, app.AccountKeeper, app.BankKeeper, app.GetSubspace(govtypes.ModuleName)),
		stake.NewAppModule(app.StakeKeeper, app.caller),
		params.NewAppModule(app.ParamsKeeper),
		consensus.NewAppModule(appCodec, app.ConsensusParamsKeeper),
		clerk.NewAppModule(appCodec, app.ClerkKeeper),
		chainmanager.NewAppModule(app.ChainManagerKeeper),
		topup.NewAppModule(app.TopupKeeper, app.caller),
		checkpoint.NewAppModule(&app.CheckpointKeeper),
		milestone.NewAppModule(&app.MilestoneKeeper),
		// TODO HV2: uncomment when bor module is implemented
		// bor.NewAppModule(app.BorKeeper),
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

	sideTxCfg := sidetxs.NewSideTxConfigurator()
	app.RegisterSideMsgServices(sideTxCfg)

	// Create the voteExtProcessor using sideTxCfg
	voteExtProcessor := NewVoteExtensionProcessor(sideTxCfg)
	app.VoteExtensionProcessor = voteExtProcessor

	// Set the voteExtension methods to HeimdallApp
	bApp.SetExtendVoteHandler(app.VoteExtensionProcessor.ExtendVote())
	bApp.SetVerifyVoteExtensionHandler(app.VoteExtensionProcessor.VerifyVoteExtension())

	// TODO HV2: is this order correct?
	app.mm.SetOrderBeginBlockers(
		genutiltypes.ModuleName,
		staketypes.ModuleName,
	)

	// TODO HV2: is this order correct? Do we need any other module?
	app.mm.SetOrderEndBlockers(
		govtypes.ModuleName,
		genutiltypes.ModuleName,
		staketypes.ModuleName,
	)

	// TODO HV2: is this order correct?
	genesisModuleOrder := []string{
		authtypes.ModuleName,
		banktypes.ModuleName,
		govtypes.ModuleName,
		genutiltypes.ModuleName,
		consensusparamtypes.ModuleName,
		chainmanagertypes.ModuleName,
		topupTypes.ModuleName,
		staketypes.ModuleName,
		checkpointTypes.ModuleName,
		milestoneTypes.ModuleName,
		clerktypes.ModuleName,
		// TODO HV2: uncomment when bor module is implemented
		// bortypes.ModuleName,
	}

	app.mm.SetOrderInitGenesis(genesisModuleOrder...)
	app.mm.SetOrderExportGenesis(genesisModuleOrder...)

	app.configurator = module.NewConfigurator(
		app.appCodec,
		app.MsgServiceRouter(),
		app.GRPCQueryRouter(),
	)
	err = app.mm.RegisterServices(app.configurator)
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
	app.MountTransientStores(tKeys)
	// TODO HV2: are memKeys needed?
	// app.MountMemoryStores(memKeys)
	// initialize BaseApp
	app.SetInitChainer(app.InitChainer)
	app.SetPreBlocker(app.PreBlocker)
	app.SetBeginBlocker(app.BeginBlocker)
	app.SetEndBlocker(app.EndBlocker)
	app.setAnteHandler(txConfig)

	// At startup, after all modules have been registered, check that all proto
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
	if err := json.Unmarshal(req.AppStateBytes, &genesisState); err != nil {
		panic(err)
	}

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

	stakingState := staketypes.GetGenesisStateFromAppState(app.appCodec, genesisState)
	checkpointState := checkpointTypes.GetGenesisStateFromAppState(app.appCodec, genesisState)

	// check if validator is current validator
	// add to val updates else skip
	var valUpdates []abci.ValidatorUpdate

	for _, validator := range stakingState.Validators {
		if validator.IsCurrentValidator(checkpointState.AckCount) {
			cmtProtoPk, err := validator.CmtConsPublicKey()
			if err != nil {
				panic(err)
			}

			// convert to Validator Update
			updateVal := abci.ValidatorUpdate{
				Power:  validator.VotingPower,
				PubKey: cmtProtoPk,
			}
			// Add validator to validator updated to be processed below
			valUpdates = append(valUpdates, updateVal)
		}
	}

	// TODO: make sure old validators don't go in validator updates i.e. deactivated validators have to be removed
	// update validators
	return &abci.ResponseInitChain{
		Validators: valUpdates,
	}, nil
}

// BeginBlocker application updates every begin block
func (app *HeimdallApp) BeginBlocker(ctx sdk.Context) (sdk.BeginBlock, error) {
	// TODO HV2: is this needed?

	if proposer, ok := app.AccountKeeper.GetBlockProposer(ctx); ok {
		account, err := sdk.AccAddressFromHex(proposer.String())
		if err != nil {
			app.Logger().Error("error while converting the proposer from hex to account address", "error", err)
			return sdk.BeginBlock{}, err
		}
		err = app.AccountKeeper.SetBlockProposer(ctx, account)
		if err != nil {
			app.Logger().Error("error while setting the block proposer", "error", err)
			return sdk.BeginBlock{}, err
		}
	}

	return app.mm.BeginBlock(ctx)
}

// EndBlocker application updates every end block
func (app *HeimdallApp) EndBlocker(ctx sdk.Context) (sdk.EndBlock, error) {
	// TODO HV2: is this needed?

	// transfer fees to current proposer
	if proposer, ok := app.AccountKeeper.GetBlockProposer(ctx); ok {
		moduleAccount := app.AccountKeeper.GetModuleAccount(ctx, authtypes.FeeCollectorName)
		coins := app.BankKeeper.GetBalance(ctx, moduleAccount.GetAddress(), authtypes.FeeToken)
		if !coins.Amount.IsZero() {
			coins := sdk.Coins{sdk.Coin{Denom: authtypes.FeeToken, Amount: coins.Amount}}
			if err := app.BankKeeper.SendCoinsFromModuleToAccount(ctx, authtypes.FeeCollectorName, proposer, coins); err != nil {
				app.Logger().Error("EndBlocker | SendCoinsFromModuleToAccount", "error", err)
			}
		}
		// remove block proposer
		err := app.AccountKeeper.RemoveBlockProposer(ctx)
		if err != nil {
			app.Logger().Error("EndBlocker | RemoveBlockProposer", "error", err)
		}
	}

	var tmValUpdates []abci.ValidatorUpdate

	// Start updating new validators
	currentValidatorSet, err := app.StakeKeeper.GetValidatorSet(ctx)
	if err != nil {
		return sdk.EndBlock{}, err
	}

	allValidators := app.StakeKeeper.GetAllValidators(ctx)
	ackCount, err := app.CheckpointKeeper.GetAckCount(ctx)
	if err != nil {
		return sdk.EndBlock{}, err
	}

	// get validator updates
	setUpdates := staketypes.GetUpdatedValidators(
		&currentValidatorSet, // pointer to current validator set -- UpdateValidators will modify it
		allValidators,        // All validators
		ackCount,             // ack count
	)

	if len(setUpdates) > 0 {
		// create new validator set
		if err := currentValidatorSet.UpdateWithChangeSet(setUpdates); err != nil {
			// return with nothing
			app.Logger().Error("unable to update current validator set", "error", err)
			return sdk.EndBlock{}, err
		}

		// validator set change
		app.Logger().Debug("Updated current validator set in EndBlocker", "proposer", currentValidatorSet.GetProposer())

		// save set in store
		if err := app.StakeKeeper.UpdateValidatorSetInStore(ctx, currentValidatorSet); err != nil {
			// return with nothing
			app.Logger().Error("unable to update current validator set in state", "error", err)
			return sdk.EndBlock{}, err
		}

		// convert updates from map to array
		for _, v := range setUpdates {
			cmtProtoPk, err := v.CmtConsPublicKey()
			if err != nil {
				// return with nothing
				app.Logger().Error("unable to get validator public key", "error", err)
				return sdk.EndBlock{}, err
			}
			tmValUpdates = append(tmValUpdates, abci.ValidatorUpdate{
				Power:  v.VotingPower,
				PubKey: cmtProtoPk,
			})
		}
	}

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
	delete(modAccAddrs, authtypes.NewModuleAddress(topupTypes.ModuleName).String())
	// TODO HV2: any other module to remove from the BlockedModuleAccountAddrs? So that they can send/receive tokens. Maybe bank module?
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
	modules := make(map[string]appmodule.AppModule)
	for _, m := range app.mm.Modules {
		if moduleWithName, ok := m.(module.HasName); ok {
			moduleName := moduleWithName.Name()
			if appModule, ok := moduleWithName.(appmodule.AppModule); ok {
				modules[moduleName] = appModule
			}
		}
	}

	return autocli.AppOptions{
		Modules:               modules,
		ModuleOptions:         runtimeservices.ExtractAutoCLIOptions(app.mm.Modules),
		AddressCodec:          authcodec.NewHexCodec(),
		ValidatorAddressCodec: authcodec.NewHexCodec(),
		ConsensusAddressCodec: authcodec.NewHexCodec(),
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
	return app.tKeys[storeKey]
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

func (app *HeimdallApp) RegisterSideMsgServices(cfg sidetxs.SideTxConfigurator) {
	for _, md := range app.mm.Modules {
		if sideMsgModule, ok := md.(sidetxs.HasSideMsgServices); ok {
			sideMsgModule.RegisterSideMsgServices(cfg)
		}
	}
}

type EmptyAppOptions struct{}

func (ao EmptyAppOptions) Get(_ string) interface{} {
	return nil
}

// TODO HV2: params will be soon deprecated, remove paramskeeper once it's done

// initParamsKeeper init params keeper and its subspaces
func initParamsKeeper(appCodec codec.BinaryCodec, legacyAmino *codec.LegacyAmino, key, storeKey storetypes.StoreKey) paramskeeper.Keeper {
	paramsKeeper := paramskeeper.NewKeeper(appCodec, legacyAmino, key, storeKey)

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

// cacheTxContext returns a new context based off of the provided context with
// a cache wrapped multi-store.
func (app *HeimdallApp) cacheTxContext(ctx sdk.Context, _ []byte) (sdk.Context, storetypes.CacheMultiStore) {
	ms := ctx.MultiStore()
	msCache := ms.CacheMultiStore()

	return ctx.WithMultiStore(msCache), msCache
}

// GetMaccPerms returns a copy of the module account permissions
func GetMaccPerms() map[string][]string {
	dupMaccPerms := make(map[string][]string)
	for k, v := range maccPerms {
		dupMaccPerms[k] = v
	}

	return dupMaccPerms
}
