package app

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/server/api"
	"github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/cosmos/cosmos-sdk/simapp"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authsims "github.com/cosmos/cosmos-sdk/x/auth/simulation"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/auth/vesting"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	authzmodule "github.com/cosmos/cosmos-sdk/x/authz/module"
	"github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/capability"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	crisiskeeper "github.com/cosmos/cosmos-sdk/x/crisis/keeper"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	"github.com/cosmos/cosmos-sdk/x/evidence"
	evidencekeeper "github.com/cosmos/cosmos-sdk/x/evidence/keeper"
	evidencetypes "github.com/cosmos/cosmos-sdk/x/evidence/types"
	"github.com/cosmos/cosmos-sdk/x/feegrant"
	feegrantkeeper "github.com/cosmos/cosmos-sdk/x/feegrant/keeper"
	feegrantmodule "github.com/cosmos/cosmos-sdk/x/feegrant/module"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	packetforward "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v4/router"
	packetforwardkeeper "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v4/router/keeper"
	packetforwardtypes "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v4/router/types"
	ica "github.com/cosmos/ibc-go/v4/modules/apps/27-interchain-accounts"
	icahost "github.com/cosmos/ibc-go/v4/modules/apps/27-interchain-accounts/host"
	icahostkeeper "github.com/cosmos/ibc-go/v4/modules/apps/27-interchain-accounts/host/keeper"
	icahosttypes "github.com/cosmos/ibc-go/v4/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v4/modules/apps/27-interchain-accounts/types"
	"github.com/cosmos/ibc-go/v4/modules/apps/transfer"
	ibctransferkeeper "github.com/cosmos/ibc-go/v4/modules/apps/transfer/keeper"
	ibctransfertypes "github.com/cosmos/ibc-go/v4/modules/apps/transfer/types"
	ibc "github.com/cosmos/ibc-go/v4/modules/core"
	ibcclienttypes "github.com/cosmos/ibc-go/v4/modules/core/02-client/types"
	ibcporttypes "github.com/cosmos/ibc-go/v4/modules/core/05-port/types"
	ibchost "github.com/cosmos/ibc-go/v4/modules/core/24-host"
	ibckeeper "github.com/cosmos/ibc-go/v4/modules/core/keeper"
	"github.com/cosmos/interchain-security/v2/x/ccv/consumer"
	consumerKeeper "github.com/cosmos/interchain-security/v2/x/ccv/consumer/keeper"
	consumerTypes "github.com/cosmos/interchain-security/v2/x/ccv/consumer/types"
	"github.com/spf13/cast"
	paramauthorityibc "github.com/strangelove-ventures/paramauthority/x/ibc"
	paramauthorityibctypes "github.com/strangelove-ventures/paramauthority/x/ibc/types"
	paramauthority "github.com/strangelove-ventures/paramauthority/x/params"
	paramauthoritykeeper "github.com/strangelove-ventures/paramauthority/x/params/keeper"
	paramauthorityupgrade "github.com/strangelove-ventures/paramauthority/x/upgrade"
	paramauthorityupgradekeeper "github.com/strangelove-ventures/paramauthority/x/upgrade/keeper"

	abci "github.com/tendermint/tendermint/abci/types"
	tmjson "github.com/tendermint/tendermint/libs/json"
	"github.com/tendermint/tendermint/libs/log"
	tmos "github.com/tendermint/tendermint/libs/os"
	dbm "github.com/tendermint/tm-db"

	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/strangelove-ventures/noble/app/upgrades/argon"
	"github.com/strangelove-ventures/noble/app/upgrades/argon4"
	"github.com/strangelove-ventures/noble/app/upgrades/krypton"
	"github.com/strangelove-ventures/noble/app/upgrades/neon"
	"github.com/strangelove-ventures/noble/app/upgrades/radon"
	v3m1p0 "github.com/strangelove-ventures/noble/app/upgrades/v3.1.0"
	v4m0p0rc0 "github.com/strangelove-ventures/noble/app/upgrades/v4.0.0-rc0"
	"github.com/strangelove-ventures/noble/cmd"
	"github.com/strangelove-ventures/noble/docs"
	"github.com/strangelove-ventures/noble/x/blockibc"
	fiattokenfactorymodule "github.com/strangelove-ventures/noble/x/fiattokenfactory"
	fiattokenfactorymodulekeeper "github.com/strangelove-ventures/noble/x/fiattokenfactory/keeper"
	fiattokenfactorymoduletypes "github.com/strangelove-ventures/noble/x/fiattokenfactory/types"
	"github.com/strangelove-ventures/noble/x/globalfee"
	tariff "github.com/strangelove-ventures/noble/x/tariff"
	tariffkeeper "github.com/strangelove-ventures/noble/x/tariff/keeper"
	tarifftypes "github.com/strangelove-ventures/noble/x/tariff/types"
	tokenfactorymodule "github.com/strangelove-ventures/noble/x/tokenfactory"
	tokenfactorymodulekeeper "github.com/strangelove-ventures/noble/x/tokenfactory/keeper"
	tokenfactorymoduletypes "github.com/strangelove-ventures/noble/x/tokenfactory/types"

	cctp "github.com/circlefin/noble-cctp/x/cctp"
	cctpkeeper "github.com/circlefin/noble-cctp/x/cctp/keeper"
	cctptypes "github.com/circlefin/noble-cctp/x/cctp/types"
)

const (
	AccountAddressPrefix = "noble"
	Name                 = "noble"
	ChainID              = "noble-1"
)

// this line is used by starport scaffolding # stargate/wasm/app/enabledProposals

var (
	// DefaultNodeHome default home directories for the application daemon
	DefaultNodeHome string

	// ModuleBasics defines the module BasicManager is in charge of setting up basic,
	// non-dependant module elements, such as codec registration
	// and genesis verification.
	ModuleBasics = module.NewBasicManager(
		auth.AppModuleBasic{},
		authzmodule.AppModuleBasic{},
		genutil.AppModuleBasic{},
		bank.AppModuleBasic{},
		staking.AppModuleBasic{},
		capability.AppModuleBasic{},
		paramauthority.AppModuleBasic{},
		crisis.AppModuleBasic{},
		slashing.AppModuleBasic{},
		feegrantmodule.AppModuleBasic{},
		ibc.AppModuleBasic{},
		paramauthorityupgrade.AppModuleBasic{},
		evidence.AppModuleBasic{},
		transfer.AppModuleBasic{},
		ica.AppModuleBasic{},
		vesting.AppModuleBasic{},
		tokenfactorymodule.AppModuleBasic{},
		fiattokenfactorymodule.AppModuleBasic{},
		packetforward.AppModuleBasic{},
		globalfee.AppModuleBasic{},
		tariff.AppModuleBasic{},
		cctp.AppModuleBasic{},
		paramauthorityibc.AppModuleBasic{},
		consumer.AppModuleBasic{},
	)

	// module account permissions
	maccPerms = map[string][]string{
		authtypes.FeeCollectorName:                 nil,
		icatypes.ModuleName:                        nil,
		ibctransfertypes.ModuleName:                {authtypes.Minter, authtypes.Burner},
		tokenfactorymoduletypes.ModuleName:         {authtypes.Minter, authtypes.Burner, authtypes.Staking},
		fiattokenfactorymoduletypes.ModuleName:     {authtypes.Minter, authtypes.Burner, authtypes.Staking},
		stakingtypes.BondedPoolName:                {authtypes.Burner, authtypes.Staking},
		stakingtypes.NotBondedPoolName:             {authtypes.Burner, authtypes.Staking},
		cctptypes.ModuleName:                       nil,
		consumerTypes.ConsumerRedistributeName:     nil,
		consumerTypes.ConsumerToSendToProviderName: nil,
	}
)

var (
	_ cmd.App                 = (*App)(nil)
	_ servertypes.Application = (*App)(nil)
	_ simapp.App              = (*App)(nil)
)

func init() {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	DefaultNodeHome = filepath.Join(userHomeDir, "."+Name)
}

// App extends an ABCI application, but with most of its parameters exported.
// They are exported for convenience in creating helper functions, as object
// capabilities aren't needed for testing.
type App struct {
	*baseapp.BaseApp

	cdc               *codec.LegacyAmino
	appCodec          codec.Codec
	interfaceRegistry types.InterfaceRegistry

	invCheckPeriod uint

	// keys to access the substores
	keys    map[string]*storetypes.KVStoreKey
	tkeys   map[string]*storetypes.TransientStoreKey
	memKeys map[string]*storetypes.MemoryStoreKey

	// keepers
	AccountKeeper       authkeeper.AccountKeeper
	AuthzKeeper         authzkeeper.Keeper
	BankKeeper          bankkeeper.Keeper
	StakingKeeper       stakingkeeper.Keeper
	CapabilityKeeper    *capabilitykeeper.Keeper
	SlashingKeeper      slashingkeeper.Keeper
	CrisisKeeper        crisiskeeper.Keeper
	UpgradeKeeper       paramauthorityupgradekeeper.Keeper
	ParamsKeeper        paramauthoritykeeper.Keeper
	IBCKeeper           *ibckeeper.Keeper // IBC Keeper must be a pointer in the app, so we can SetRouter on it correctly
	EvidenceKeeper      evidencekeeper.Keeper
	TransferKeeper      ibctransferkeeper.Keeper
	ICAHostKeeper       icahostkeeper.Keeper
	FeeGrantKeeper      feegrantkeeper.Keeper
	PacketForwardKeeper *packetforwardkeeper.Keeper
	ConsumerKeeper      consumerKeeper.Keeper

	// make scoped keepers public for test purposes
	ScopedIBCKeeper      capabilitykeeper.ScopedKeeper
	ScopedTransferKeeper capabilitykeeper.ScopedKeeper
	ScopedICAHostKeeper  capabilitykeeper.ScopedKeeper
	ScopedConsumerKeeper capabilitykeeper.ScopedKeeper

	TokenFactoryKeeper     *tokenfactorymodulekeeper.Keeper
	FiatTokenFactoryKeeper *fiattokenfactorymodulekeeper.Keeper
	TariffKeeper           tariffkeeper.Keeper
	CCTPKeeper             *cctpkeeper.Keeper

	// this line is used by starport scaffolding # stargate/app/keeperDeclaration

	// mm is the module manager
	mm *module.Manager

	// sm is the simulation manager
	sm           *module.SimulationManager
	configurator module.Configurator
}

// New returns a reference to an initialized blockchain app
func New(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	loadLatest bool,
	skipUpgradeHeights map[int64]bool,
	homePath string,
	invCheckPeriod uint,
	encodingConfig cmd.EncodingConfig,
	appOpts servertypes.AppOptions,
	baseAppOptions ...func(*baseapp.BaseApp),
) cmd.App {
	appCodec := encodingConfig.Marshaler
	cdc := encodingConfig.Amino
	interfaceRegistry := encodingConfig.InterfaceRegistry

	bApp := baseapp.NewBaseApp(Name, logger, db, encodingConfig.TxConfig.TxDecoder(), baseAppOptions...)
	bApp.SetCommitMultiStoreTracer(traceStore)
	bApp.SetVersion(version.Version)
	bApp.SetInterfaceRegistry(interfaceRegistry)

	keys := sdk.NewKVStoreKeys(
		authtypes.StoreKey, authz.ModuleName, banktypes.StoreKey, slashingtypes.StoreKey,
		paramstypes.StoreKey, ibchost.StoreKey, upgradetypes.StoreKey, feegrant.StoreKey, evidencetypes.StoreKey,
		ibctransfertypes.StoreKey, icahosttypes.StoreKey, capabilitytypes.StoreKey,
		tokenfactorymoduletypes.StoreKey, fiattokenfactorymoduletypes.StoreKey, packetforwardtypes.StoreKey, stakingtypes.StoreKey,
		cctptypes.StoreKey, consumerTypes.StoreKey,
	)
	tkeys := sdk.NewTransientStoreKeys(paramstypes.TStoreKey)
	memKeys := sdk.NewMemoryStoreKeys(capabilitytypes.MemStoreKey)

	app := &App{
		BaseApp:           bApp,
		cdc:               cdc,
		appCodec:          appCodec,
		interfaceRegistry: interfaceRegistry,
		invCheckPeriod:    invCheckPeriod,
		keys:              keys,
		tkeys:             tkeys,
		memKeys:           memKeys,
	}

	app.ParamsKeeper = initParamsKeeper(
		appCodec,
		cdc,
		keys[paramstypes.StoreKey],
		tkeys[paramstypes.TStoreKey],
	)

	// set the BaseApp's parameter store
	bApp.SetParamStore(app.ParamsKeeper.Subspace(baseapp.Paramspace).WithKeyTable(paramskeeper.ConsensusParamsKeyTable()))

	// add capability keeper and ScopeToModule for ibc module
	app.CapabilityKeeper = capabilitykeeper.NewKeeper(
		appCodec,
		keys[capabilitytypes.StoreKey],
		memKeys[capabilitytypes.MemStoreKey],
	)

	// grant capabilities for the ibc and ibc-transfer modules
	scopedIBCKeeper := app.CapabilityKeeper.ScopeToModule(ibchost.ModuleName)
	scopedTransferKeeper := app.CapabilityKeeper.ScopeToModule(ibctransfertypes.ModuleName)
	scopedICAHostKeeper := app.CapabilityKeeper.ScopeToModule(icahosttypes.SubModuleName)
	scopedConsumerKeeper := app.CapabilityKeeper.ScopeToModule(consumerTypes.ModuleName)
	// this line is used by starport scaffolding # stargate/app/scopedKeeper

	// add keepers
	app.AccountKeeper = authkeeper.NewAccountKeeper(
		appCodec,
		keys[authtypes.StoreKey],
		app.GetSubspace(authtypes.ModuleName),
		authtypes.ProtoBaseAccount,
		maccPerms,
	)

	app.AuthzKeeper = authzkeeper.NewKeeper(
		keys[authz.ModuleName],
		appCodec,
		app.MsgServiceRouter(),
	)

	app.BankKeeper = bankkeeper.NewBaseKeeper(
		appCodec,
		keys[banktypes.StoreKey],
		app.AccountKeeper,
		app.GetSubspace(banktypes.ModuleName),
		app.BlockedModuleAccountAddrs(),
	)

	app.StakingKeeper = stakingkeeper.NewKeeper(
		appCodec,
		keys[stakingtypes.StoreKey],
		app.AccountKeeper,
		app.BankKeeper,
		app.GetSubspace(stakingtypes.ModuleName),
	)

	app.SlashingKeeper = slashingkeeper.NewKeeper(
		appCodec,
		keys[slashingtypes.StoreKey],
		&app.ConsumerKeeper,
		app.GetSubspace(slashingtypes.ModuleName),
	)

	app.CrisisKeeper = crisiskeeper.NewKeeper(
		app.GetSubspace(crisistypes.ModuleName),
		invCheckPeriod,
		app.BankKeeper,
		authtypes.FeeCollectorName,
	)

	app.FeeGrantKeeper = feegrantkeeper.NewKeeper(
		appCodec,
		keys[feegrant.StoreKey],
		app.AccountKeeper,
	)

	app.UpgradeKeeper = paramauthorityupgradekeeper.NewKeeper(
		skipUpgradeHeights,
		keys[upgradetypes.StoreKey],
		appCodec,
		homePath,
		app.BaseApp,
		app.GetSubspace(upgradetypes.ModuleName),
	)

	// register the staking hooks
	// NOTE: stakingKeeper above is passed by reference, so that it will contain these hooks
	app.StakingKeeper = *app.StakingKeeper.SetHooks(app.SlashingKeeper.Hooks())

	// ... other modules keepers

	// Create IBC Keeper
	app.IBCKeeper = ibckeeper.NewKeeper(
		appCodec, keys[ibchost.StoreKey],
		app.GetSubspace(ibchost.ModuleName),
		&app.ConsumerKeeper,
		app.UpgradeKeeper,
		scopedIBCKeeper,
	)

	app.TariffKeeper = tariffkeeper.NewKeeper(
		app.GetSubspace(tarifftypes.ModuleName),
		app.AccountKeeper,
		app.BankKeeper,
		authtypes.FeeCollectorName,
		app.IBCKeeper.ChannelKeeper,
	)

	app.PacketForwardKeeper = packetforwardkeeper.NewKeeper(
		appCodec,
		keys[packetforwardtypes.StoreKey],
		app.GetSubspace(packetforwardtypes.ModuleName),
		app.TransferKeeper, // will be zero-value here. reference set later on with SetTransferKeeper.
		app.IBCKeeper.ChannelKeeper,
		nil, // TODO(@john): Why is this needed?
		app.BankKeeper,
		app.TariffKeeper,
	)

	app.ConsumerKeeper = consumerKeeper.NewKeeper(
		appCodec,
		keys[consumerTypes.StoreKey],
		app.GetSubspace(consumerTypes.ModuleName),
		scopedConsumerKeeper,
		app.IBCKeeper.ChannelKeeper,
		&app.IBCKeeper.PortKeeper,
		app.IBCKeeper.ConnectionKeeper,
		app.IBCKeeper.ClientKeeper,
		app.SlashingKeeper,
		app.BankKeeper,
		app.AccountKeeper,
		&app.TransferKeeper,
		app.IBCKeeper,
		authtypes.FeeCollectorName,
	)

	app.ConsumerKeeper.SetStandaloneStakingKeeper(app.StakingKeeper)

	app.ConsumerKeeper = *app.ConsumerKeeper.SetHooks(app.SlashingKeeper.Hooks())
	consumerModule := consumer.NewAppModule(app.ConsumerKeeper, app.GetSubspace(consumerTypes.ModuleName))

	// Create Transfer Keepers
	app.TransferKeeper = ibctransferkeeper.NewKeeper(
		appCodec,
		keys[ibctransfertypes.StoreKey],
		app.GetSubspace(ibctransfertypes.ModuleName),
		app.PacketForwardKeeper,
		app.IBCKeeper.ChannelKeeper,
		&app.IBCKeeper.PortKeeper,
		app.AccountKeeper,
		app.BankKeeper,
		scopedTransferKeeper,
	)

	app.PacketForwardKeeper.SetTransferKeeper(app.TransferKeeper)

	transferModule := transfer.NewAppModule(app.TransferKeeper)

	app.ICAHostKeeper = icahostkeeper.NewKeeper(
		appCodec, keys[icahosttypes.StoreKey],
		app.GetSubspace(icahosttypes.SubModuleName),
		app.IBCKeeper.ChannelKeeper,
		&app.IBCKeeper.PortKeeper,
		app.AccountKeeper,
		scopedICAHostKeeper,
		app.MsgServiceRouter(),
	)
	icaModule := ica.NewAppModule(nil, &app.ICAHostKeeper)
	icaHostIBCModule := icahost.NewIBCModule(app.ICAHostKeeper)

	// Create evidence Keeper for to register the IBC light client misbehaviour evidence route
	evidenceKeeper := evidencekeeper.NewKeeper(
		appCodec,
		keys[evidencetypes.StoreKey],
		&app.ConsumerKeeper,
		app.SlashingKeeper,
	)
	// If evidence needs to be handled for the app, set routes in router here and seal
	app.EvidenceKeeper = *evidenceKeeper

	app.TokenFactoryKeeper = tokenfactorymodulekeeper.NewKeeper(
		appCodec,
		keys[tokenfactorymoduletypes.StoreKey],
		app.GetSubspace(tokenfactorymoduletypes.ModuleName),

		app.BankKeeper,
	)
	tokenfactoryModule := tokenfactorymodule.NewAppModule(appCodec, app.TokenFactoryKeeper, app.AccountKeeper, app.BankKeeper)

	app.FiatTokenFactoryKeeper = fiattokenfactorymodulekeeper.NewKeeper(
		appCodec,
		keys[fiattokenfactorymoduletypes.StoreKey],
		app.GetSubspace(fiattokenfactorymoduletypes.ModuleName),

		app.BankKeeper,
	)
	fiattokenfactorymodule := fiattokenfactorymodule.NewAppModule(appCodec, app.FiatTokenFactoryKeeper, app.AccountKeeper, app.BankKeeper)

	app.CCTPKeeper = cctpkeeper.NewKeeper(
		appCodec,
		keys[cctptypes.StoreKey],
		app.GetSubspace(cctptypes.ModuleName),
		app.BankKeeper,
		app.FiatTokenFactoryKeeper,
	)

	var transferStack ibcporttypes.IBCModule
	transferStack = transfer.NewIBCModule(app.TransferKeeper)
	transferStack = packetforward.NewIBCMiddleware(
		transferStack,
		app.PacketForwardKeeper,
		0,
		packetforwardkeeper.DefaultForwardTransferPacketTimeoutTimestamp,
		packetforwardkeeper.DefaultRefundTransferPacketTimeoutTimestamp,
	)
	transferStack = blockibc.NewIBCMiddleware(transferStack, app.TokenFactoryKeeper, app.FiatTokenFactoryKeeper)

	// Create static IBC router, add transfer route, then set and seal it
	ibcRouter := ibcporttypes.NewRouter()
	ibcRouter.AddRoute(icahosttypes.SubModuleName, icaHostIBCModule).
		AddRoute(ibctransfertypes.ModuleName, transferStack).
		AddRoute(consumerTypes.ModuleName, consumerModule)

	// this line is used by starport scaffolding # ibc/app/router
	app.IBCKeeper.SetRouter(ibcRouter)

	/****  Module Options ****/

	// NOTE: we may consider parsing `appOpts` inside module constructors. For the moment
	// we prefer to be more strict in what arguments the modules expect.
	var skipGenesisInvariants = cast.ToBool(appOpts.Get(crisis.FlagSkipGenesisInvariants))

	// NOTE: Any module instantiated in the module manager that is later modified
	// must be passed by reference here.

	app.mm = module.NewManager(
		// TODO(@john): Can we simply remove this?
		//genutil.NewAppModule(
		//	app.AccountKeeper, app.StakingKeeper, app.BaseApp.DeliverTx,
		//	encodingConfig.TxConfig,
		//),
		auth.NewAppModule(appCodec, app.AccountKeeper, nil),
		authzmodule.NewAppModule(appCodec, app.AuthzKeeper, app.AccountKeeper, app.BankKeeper, app.interfaceRegistry),
		vesting.NewAppModule(app.AccountKeeper, app.BankKeeper),
		bank.NewAppModule(appCodec, app.BankKeeper, app.AccountKeeper),
		capability.NewAppModule(appCodec, *app.CapabilityKeeper),
		feegrantmodule.NewAppModule(appCodec, app.AccountKeeper, app.BankKeeper, app.FeeGrantKeeper, app.interfaceRegistry),
		crisis.NewAppModule(&app.CrisisKeeper, skipGenesisInvariants),
		slashing.NewAppModule(appCodec, app.SlashingKeeper, app.AccountKeeper, app.BankKeeper, app.ConsumerKeeper),
		paramauthorityupgrade.NewAppModule(app.UpgradeKeeper),
		evidence.NewAppModule(app.EvidenceKeeper),
		ibc.NewAppModule(app.IBCKeeper),
		paramauthority.NewAppModule(app.ParamsKeeper),
		transferModule,
		icaModule,
		tokenfactoryModule,
		fiattokenfactorymodule,
		packetforward.NewAppModule(app.PacketForwardKeeper),
		staking.NewAppModule(appCodec, app.StakingKeeper, app.AccountKeeper, app.BankKeeper),
		globalfee.NewAppModule(app.GetSubspace(globalfee.ModuleName)),
		tariff.NewAppModule(appCodec, app.TariffKeeper, app.AccountKeeper, app.BankKeeper),
		cctp.NewAppModule(appCodec, app.AccountKeeper, app.BankKeeper, app.CCTPKeeper),
		consumerModule,
	)

	// During begin block slashing happens after distr.BeginBlocker so that
	// there is nothing left over in the validator fee pool, so as to keep the
	// CanWithdrawInvariant invariant.
	// NOTE: staking module is required if HistoricalEntries param > 0
	app.mm.SetOrderBeginBlockers(
		// upgrades should be run first
		upgradetypes.ModuleName,
		capabilitytypes.ModuleName,
		tarifftypes.ModuleName,
		slashingtypes.ModuleName,
		evidencetypes.ModuleName,
		stakingtypes.ModuleName,
		authtypes.ModuleName,
		banktypes.ModuleName,
		crisistypes.ModuleName,
		ibctransfertypes.ModuleName,
		ibchost.ModuleName,
		icatypes.ModuleName,
		genutiltypes.ModuleName,
		packetforwardtypes.ModuleName,
		authz.ModuleName,
		feegrant.ModuleName,
		paramstypes.ModuleName,
		vestingtypes.ModuleName,
		tokenfactorymoduletypes.ModuleName,
		fiattokenfactorymoduletypes.ModuleName,
		globalfee.ModuleName,
		cctptypes.ModuleName,
		consumerTypes.ModuleName,
	)

	app.mm.SetOrderEndBlockers(
		crisistypes.ModuleName,
		stakingtypes.ModuleName,
		ibctransfertypes.ModuleName,
		ibchost.ModuleName,
		icatypes.ModuleName,
		packetforwardtypes.ModuleName,
		capabilitytypes.ModuleName,
		authtypes.ModuleName,
		banktypes.ModuleName,
		slashingtypes.ModuleName,
		genutiltypes.ModuleName,
		evidencetypes.ModuleName,
		authz.ModuleName,
		feegrant.ModuleName,
		paramstypes.ModuleName,
		upgradetypes.ModuleName,
		vestingtypes.ModuleName,
		tokenfactorymoduletypes.ModuleName,
		fiattokenfactorymoduletypes.ModuleName,
		globalfee.ModuleName,
		tarifftypes.ModuleName,
		cctptypes.ModuleName,
		consumerTypes.ModuleName,
	)

	// NOTE: The genutils module must occur after staking so that pools are
	// properly initialized with tokens from genesis accounts.
	// NOTE: Capability module must occur first so that it can initialize any capabilities
	// so that other modules that want to create or claim capabilities afterwards in InitChain
	// can do so safely.
	app.mm.SetOrderInitGenesis(
		capabilitytypes.ModuleName,
		ibctransfertypes.ModuleName,
		authtypes.ModuleName,
		banktypes.ModuleName,
		stakingtypes.ModuleName,
		tarifftypes.ModuleName,
		slashingtypes.ModuleName,
		crisistypes.ModuleName,
		genutiltypes.ModuleName,
		ibchost.ModuleName,
		icatypes.ModuleName,
		packetforwardtypes.ModuleName,
		evidencetypes.ModuleName,
		authz.ModuleName,
		feegrant.ModuleName,
		paramstypes.ModuleName,
		upgradetypes.ModuleName,
		vestingtypes.ModuleName,
		tokenfactorymoduletypes.ModuleName,
		fiattokenfactorymoduletypes.ModuleName,
		globalfee.ModuleName,
		cctptypes.ModuleName,
		consumerTypes.ModuleName,

		// this line is used by starport scaffolding # stargate/app/initGenesis
	)

	// Uncomment if you want to set a custom migration order here.
	// app.mm.SetOrderMigrations(custom order)

	app.mm.RegisterInvariants(&app.CrisisKeeper)
	app.mm.RegisterRoutes(app.Router(), app.QueryRouter(), encodingConfig.Amino)

	app.configurator = module.NewConfigurator(app.appCodec, app.MsgServiceRouter(), app.GRPCQueryRouter())
	app.mm.RegisterServices(app.configurator)

	// Register authoritative IBC client update and IBC upgrade msg handlers
	paramauthorityibctypes.RegisterMsgServer(
		app.configurator.MsgServer(),
		paramauthorityibc.NewMsgServer(app.UpgradeKeeper, app.IBCKeeper.ClientKeeper),
	)

	// create the simulation manager and define the order of the modules for deterministic simulations
	overrideModules := map[string]module.AppModuleSimulation{
		authtypes.ModuleName: auth.NewAppModule(app.appCodec, app.AccountKeeper, authsims.RandomGenesisAccounts),
	}
	app.sm = module.NewSimulationManagerFromAppModules(app.mm.Modules, overrideModules)
	app.sm.RegisterStoreDecoders()

	// initialize stores
	app.MountKVStores(keys)
	app.MountTransientStores(tkeys)
	app.MountMemoryStores(memKeys)

	// initialize BaseApp
	anteHandler, err := NewAnteHandler(
		HandlerOptions{
			HandlerOptions: ante.HandlerOptions{
				AccountKeeper:   app.AccountKeeper,
				BankKeeper:      app.BankKeeper,
				FeegrantKeeper:  app.FeeGrantKeeper,
				SignModeHandler: encodingConfig.TxConfig.SignModeHandler(),
				SigGasConsumer:  ante.DefaultSigVerificationGasConsumer,
			},
			tokenFactoryKeeper:     app.TokenFactoryKeeper,
			fiatTokenFactoryKeeper: app.FiatTokenFactoryKeeper,

			IBCKeeper:         app.IBCKeeper,
			ConsumerKeeper:    app.ConsumerKeeper,
			GlobalFeeSubspace: app.GetSubspace(globalfee.ModuleName),
		},
	)
	if err != nil {
		panic(fmt.Errorf("failed to create AnteHandler: %s", err))
	}

	app.SetAnteHandler(anteHandler)
	app.SetInitChainer(app.InitChainer)
	app.SetBeginBlocker(app.BeginBlocker)
	app.SetEndBlocker(app.EndBlocker)
	app.setupUpgradeHandlers(appOpts)

	if loadLatest {
		if err := app.LoadLatestVersion(); err != nil {
			tmos.Exit(err.Error())
		}
	}

	app.ScopedIBCKeeper = scopedIBCKeeper
	app.ScopedTransferKeeper = scopedTransferKeeper
	app.ScopedConsumerKeeper = scopedConsumerKeeper
	// this line is used by starport scaffolding # stargate/app/beforeInitReturn

	return app
}

// Name returns the name of the App
func (app *App) Name() string { return app.BaseApp.Name() }

// GetBaseApp returns the base app of the application
func (app App) GetBaseApp() *baseapp.BaseApp { return app.BaseApp }

// BeginBlocker application updates every begin block
func (app *App) BeginBlocker(ctx sdk.Context, req abci.RequestBeginBlock) abci.ResponseBeginBlock {
	return app.mm.BeginBlock(ctx, req)
}

// EndBlocker application updates every end block
func (app *App) EndBlocker(ctx sdk.Context, req abci.RequestEndBlock) abci.ResponseEndBlock {
	return app.mm.EndBlock(ctx, req)
}

// InitChainer application update at chain initialization
func (app *App) InitChainer(ctx sdk.Context, req abci.RequestInitChain) abci.ResponseInitChain {
	var genesisState GenesisState
	if err := tmjson.Unmarshal(req.AppStateBytes, &genesisState); err != nil {
		panic(err)
	}
	app.UpgradeKeeper.SetModuleVersionMap(ctx, app.mm.GetVersionMap())
	return app.mm.InitGenesis(ctx, app.appCodec, genesisState)
}

// LoadHeight loads a particular height
func (app *App) LoadHeight(height int64) error {
	return app.LoadVersion(height)
}

// ModuleAccountAddrs returns all the app's module account addresses.
func (app *App) ModuleAccountAddrs() map[string]bool {
	modAccAddrs := make(map[string]bool)
	for acc := range maccPerms {
		modAccAddrs[authtypes.NewModuleAddress(acc).String()] = true
	}

	return modAccAddrs
}

// BlockedModuleAccountAddrs returns all the app's blocked module account
// addresses.
func (app *App) BlockedModuleAccountAddrs() map[string]bool {
	modAccAddrs := app.ModuleAccountAddrs()

	delete(modAccAddrs, authtypes.NewModuleAddress(consumerTypes.ConsumerToSendToProviderName).String())

	return modAccAddrs
}

// LegacyAmino returns SimApp's amino codec.
//
// NOTE: This is solely to be used for testing purposes as it may be desirable
// for modules to register their own custom testing types.
func (app *App) LegacyAmino() *codec.LegacyAmino {
	return app.cdc
}

// AppCodec returns an app codec.
//
// NOTE: This is solely to be used for testing purposes as it may be desirable
// for modules to register their own custom testing types.
func (app *App) AppCodec() codec.Codec {
	return app.appCodec
}

// InterfaceRegistry returns an InterfaceRegistry
func (app *App) InterfaceRegistry() types.InterfaceRegistry {
	return app.interfaceRegistry
}

// GetKey returns the KVStoreKey for the provided store key.
//
// NOTE: This is solely to be used for testing purposes.
func (app *App) GetKey(storeKey string) *storetypes.KVStoreKey {
	return app.keys[storeKey]
}

// GetTKey returns the TransientStoreKey for the provided store key.
//
// NOTE: This is solely to be used for testing purposes.
func (app *App) GetTKey(storeKey string) *storetypes.TransientStoreKey {
	return app.tkeys[storeKey]
}

// GetMemKey returns the MemStoreKey for the provided mem key.
//
// NOTE: This is solely used for testing purposes.
func (app *App) GetMemKey(storeKey string) *storetypes.MemoryStoreKey {
	return app.memKeys[storeKey]
}

// GetSubspace returns a param subspace for a given module name.
//
// NOTE: This is solely to be used for testing purposes.
func (app *App) GetSubspace(moduleName string) paramstypes.Subspace {
	subspace, _ := app.ParamsKeeper.GetSubspace(moduleName)
	return subspace
}

// RegisterAPIRoutes registers all application module routes with the provided
// API server.
func (app *App) RegisterAPIRoutes(apiSvr *api.Server, apiConfig config.APIConfig) {
	clientCtx := apiSvr.ClientCtx
	// Register new tx routes from grpc-gateway.
	authtx.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)
	// Register new tendermint queries routes from grpc-gateway.
	tmservice.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// Register grpc-gateway routes for all modules.
	ModuleBasics.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// register app's OpenAPI routes.
	apiSvr.Router.Handle("/static/openapi.yml", http.FileServer(http.FS(docs.Docs)))
}

// RegisterTxService implements the Application.RegisterTxService method.
func (app *App) RegisterTxService(clientCtx client.Context) {
	authtx.RegisterTxService(app.BaseApp.GRPCQueryRouter(), clientCtx, app.BaseApp.Simulate, app.interfaceRegistry)
}

// RegisterTendermintService implements the Application.RegisterTendermintService method.
func (app *App) RegisterTendermintService(clientCtx client.Context) {
	tmservice.RegisterTendermintService(
		app.BaseApp.GRPCQueryRouter(),
		clientCtx,
		app.interfaceRegistry,
	)
}

// GetMaccPerms returns a copy of the module account permissions
func GetMaccPerms() map[string][]string {
	dupMaccPerms := make(map[string][]string)
	for k, v := range maccPerms {
		dupMaccPerms[k] = v
	}
	return dupMaccPerms
}

// initParamsKeeper init params keeper and its subspaces
func initParamsKeeper(appCodec codec.BinaryCodec, legacyAmino *codec.LegacyAmino, key, tkey storetypes.StoreKey) paramauthoritykeeper.Keeper {
	paramsKeeper := paramauthoritykeeper.NewKeeper(appCodec, legacyAmino, key, tkey)

	paramsKeeper.Subspace(authtypes.ModuleName)
	paramsKeeper.Subspace(banktypes.ModuleName)
	paramsKeeper.Subspace(stakingtypes.ModuleName)
	paramsKeeper.Subspace(tarifftypes.ModuleName)
	paramsKeeper.Subspace(slashingtypes.ModuleName)
	paramsKeeper.Subspace(crisistypes.ModuleName)
	paramsKeeper.Subspace(ibctransfertypes.ModuleName)
	paramsKeeper.Subspace(packetforwardtypes.ModuleName).WithKeyTable(packetforwardtypes.ParamKeyTable())
	paramsKeeper.Subspace(ibchost.ModuleName)
	paramsKeeper.Subspace(icahosttypes.SubModuleName)
	paramsKeeper.Subspace(tokenfactorymoduletypes.ModuleName)
	paramsKeeper.Subspace(fiattokenfactorymoduletypes.ModuleName)
	paramsKeeper.Subspace(upgradetypes.ModuleName)
	paramsKeeper.Subspace(globalfee.ModuleName)
	paramsKeeper.Subspace(cctptypes.ModuleName)
	paramsKeeper.Subspace(consumerTypes.ModuleName)
	// this line is used by starport scaffolding # stargate/app/paramSubspace

	return paramsKeeper
}

func (app *App) setupUpgradeHandlers(options servertypes.AppOptions) {
	// neon upgrade
	app.UpgradeKeeper.SetUpgradeHandler(
		neon.UpgradeName,
		neon.CreateNeonUpgradeHandler(
			app.mm,
			app.configurator,
			*app.FiatTokenFactoryKeeper,
			app.BankKeeper,
			app.AccountKeeper))

	// radon upgrade
	app.UpgradeKeeper.SetUpgradeHandler(
		radon.UpgradeName,
		radon.CreateRadonUpgradeHandler(
			app.mm,
			app.configurator,
			app.ParamsKeeper,
			app.FiatTokenFactoryKeeper,
		))

	// v3.1.0 upgrade
	app.UpgradeKeeper.SetUpgradeHandler(
		v3m1p0.UpgradeName,
		v3m1p0.CreateUpgradeHandler(
			app.mm,
			app.configurator,
		))

	// argon upgrade
	app.UpgradeKeeper.SetUpgradeHandler(
		argon.UpgradeName,
		argon.CreateUpgradeHandler(
			app.mm,
			app.configurator,
			app.FiatTokenFactoryKeeper,
			app.ParamsKeeper,
			app.CCTPKeeper,
		),
	)

	// argon4 upgrade
	app.UpgradeKeeper.SetUpgradeHandler(
		argon4.UpgradeName,
		argon4.CreateUpgradeHandler(
			app.mm,
			app.configurator,
		),
	)

	// v4.0.0-rc0 upgrade
	app.UpgradeKeeper.SetUpgradeHandler(
		v4m0p0rc0.UpgradeName,
		v4m0p0rc0.CreateUpgradeHandler(
			app.mm,
			app.configurator,
		),
	)

	// krypton upgrade
	app.UpgradeKeeper.SetUpgradeHandler(
		krypton.UpgradeName,
		krypton.CreateUpgradeHandler(
			app.mm,
			app.configurator,
			app.appCodec,
			options,
			app.ConsumerKeeper,
			app.FiatTokenFactoryKeeper,
		),
	)

	upgradeInfo, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		panic(fmt.Errorf("failed to read upgrade info from disk: %w", err))
	}
	if app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		return
	}

	var storeLoader baseapp.StoreLoader

	switch upgradeInfo.Name {
	case neon.UpgradeName:
		storeLoader = neon.CreateStoreLoader(upgradeInfo.Height)
	case radon.UpgradeName:
		storeLoader = radon.CreateStoreLoader(upgradeInfo.Height)
	case v3m1p0.UpgradeName:
		storeLoader = v3m1p0.CreateStoreLoader(upgradeInfo.Height)
	case argon.UpgradeName:
		storeLoader = argon.CreateStoreLoader(upgradeInfo.Height)
	case argon4.UpgradeName:
		storeLoader = argon4.CreateStoreLoader(upgradeInfo.Height)
	case v4m0p0rc0.UpgradeName:
		storeLoader = v4m0p0rc0.CreateStoreLoader(upgradeInfo.Height)
	case krypton.UpgradeName:
		storeLoader = krypton.CreateStoreLoader(upgradeInfo.Height)
	}

	if storeLoader != nil {
		// configure store loader that checks if version == upgradeHeight and applies store upgrades
		app.SetStoreLoader(storeLoader)
	}
}

// SimulationManager implements the SimulationApp interface
func (app *App) SimulationManager() *module.SimulationManager {
	return app.sm
}

// ConsumerApp interface implementations for e2e tests

// GetTxConfig implements the TestingApp interface.
func (app *App) GetTxConfig() client.TxConfig {
	return cmd.MakeEncodingConfig(ModuleBasics).TxConfig
}

// GetIBCKeeper implements the TestingApp interface.
func (app *App) GetIBCKeeper() *ibckeeper.Keeper {
	return app.IBCKeeper
}

// GetStakingKeeper implements the TestingApp interface.
func (app *App) GetStakingKeeper() ibcclienttypes.StakingKeeper {
	return &app.ConsumerKeeper
}

// GetScopedIBCKeeper implements the TestingApp interface.
func (app *App) GetScopedIBCKeeper() capabilitykeeper.ScopedKeeper {
	return app.ScopedIBCKeeper
}
