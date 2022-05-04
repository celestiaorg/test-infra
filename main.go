package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/celestiaorg/celestia-app/app"
	"github.com/celestiaorg/celestia-app/x/payment/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/tendermint/spm/cosmoscmd"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/privval"
	"github.com/tendermint/tendermint/proxy"
	tmtype "github.com/tendermint/tendermint/types"
	db "github.com/tendermint/tm-db"

	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
)

const (
	keyPath           string = "/Users/bidon4/testground/celestia-app/keys"
	statePath         string = "/Users/bidon4/testground/celestia-app/state"
	path              string = "/Users/bidon4/testground/celestia-app"
	defaultValKeyType        = tmtype.ABCIPubKeyTypeSecp256k1
)

var testcases = map[string]interface{}{
	"capp-1": run.InitializedTestCaseFn(runSync),
}

func main() {
	run.InvokeMap(testcases)
}

/*
Sync procedure:
1. config.toml should contain actors (one 1 actor creator of genesis, others, who are joining)
2. first actor creates a genesis file and shares to other participants
3. all actors do a N(max amount of actors) loop amount of editing the genesis file according to their address and amount of tokens
4. all actors share their own gentx to the network and should receive N(max amount of actors) amount of gentx back
5. all actors share their p2p address and should add N(max amount of actors) amount of p2p addresses as persistent peers

Actual experiment:
1. The chain starts
2. Block production is happening
*/

func runSync(runenv *runtime.RunEnv, initCtx *run.InitContext) error {

	cfg := config.DefaultConfig()
	cfg.SetRoot(path) // ~/testground/celestia-app
	config.EnsureRoot(path)

	runenv.RecordMessage(cfg.RootDir)

	capp, err := setApp()
	if err != nil {
		return err
	}

	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))

	nodeKey, err := p2p.LoadOrGenNodeKey(cfg.NodeKeyFile())
	if err != nil {
		return err
	}

	tmNode, err := node.NewNode(cfg,
		privval.LoadOrGenFilePV(cfg.PrivValidatorKeyFile(), cfg.PrivValidatorStateFile()),
		nodeKey,
		proxy.NewLocalClientCreator(capp),
		node.DefaultGenesisDocProviderFunc(cfg),
		node.DefaultDBProvider,
		node.DefaultMetricsProvider(cfg.Instrumentation),
		logger,
	)

	if err != nil {
		return err
	}

	err = tmNode.Start()

	time.Sleep(10 * time.Second)

	if err != nil {
		return err
	}

	return nil
}

func setApp() (*app.App, error) {
	db, _ := openDB(path)

	skipHeights := make(map[int64]bool)

	encCfg := cosmoscmd.MakeEncodingConfig(app.ModuleBasics)

	keyAcc := GenerateKeyringSigner(testAccName)

	tiaApp := app.New(
		log.NewTMLogger(log.NewSyncWriter(os.Stdout)),
		db,
		nil,
		true,
		skipHeights,
		path,
		0,
		encCfg,
		simapp.EmptyAppOptions{},
	)

	genesisState := NewDefaultGenesisState(encCfg.Marshaler)

	genesisState, err := AddGenesisAccount(keyAcc.GetSignerInfo().GetAddress(), genesisState, encCfg.Marshaler)
	if err != nil {
		return nil, err
	}

	stateBytes, err := json.MarshalIndent(genesisState, "", "  ")
	if err != nil {
		return nil, err
	}

	tiaApp.InitChain(
		abci.RequestInitChain{
			AppStateBytes: stateBytes,
		},
	)
	return tiaApp, nil
}

func openDB(rootDir string) (db.DB, error) {
	dataDir := filepath.Join(rootDir, "data")
	return sdk.NewLevelDB("application", dataDir)
}

func Init(path string) (*config.Config, error) {

	// var err error

	cfg := config.DefaultConfig()
	cfg.SetRoot(path) // ~/testground/celestia-app
	config.EnsureRoot(path)

	return cfg, nil
}

func SaveConfig(path string, cfg *config.Config) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return toml.NewEncoder(f).Encode(cfg)
}

// AddGenesisAccount mimics the cli addGenesisAccount command, providing an
// account with an allocation of to "token" and "celes" tokens in the genesis
// state
func AddGenesisAccount(addr sdk.AccAddress, appState map[string]json.RawMessage, cdc codec.Codec) (map[string]json.RawMessage, error) {
	// create concrete account type based on input parameters
	var genAccount authtypes.GenesisAccount

	coins := sdk.Coins{
		sdk.NewCoin("token", sdk.NewInt(1000000)),
		sdk.NewCoin(app.BondDenom, sdk.NewInt(1000000)),
	}

	balances := banktypes.Balance{Address: addr.String(), Coins: coins.Sort()}
	baseAccount := authtypes.NewBaseAccount(addr, nil, 0, 0)

	genAccount = baseAccount

	if err := genAccount.Validate(); err != nil {
		return appState, fmt.Errorf("failed to validate new genesis account: %w", err)
	}

	authGenState := authtypes.GetGenesisStateFromAppState(cdc, appState)

	accs, err := authtypes.UnpackAccounts(authGenState.Accounts)
	if err != nil {
		return appState, fmt.Errorf("failed to get accounts from any: %w", err)
	}

	if accs.Contains(addr) {
		return appState, fmt.Errorf("cannot add account at existing address %s", addr)
	}

	// Add the new account to the set of genesis accounts and sanitize the
	// accounts afterwards.
	accs = append(accs, genAccount)
	accs = authtypes.SanitizeGenesisAccounts(accs)

	genAccs, err := authtypes.PackAccounts(accs)
	if err != nil {
		return appState, fmt.Errorf("failed to convert accounts into any's: %w", err)
	}
	authGenState.Accounts = genAccs

	authGenStateBz, err := cdc.MarshalJSON(&authGenState)
	if err != nil {
		return appState, fmt.Errorf("failed to marshal auth genesis state: %w", err)
	}

	appState[authtypes.ModuleName] = authGenStateBz

	bankGenState := banktypes.GetGenesisStateFromAppState(cdc, appState)
	bankGenState.Balances = append(bankGenState.Balances, balances)
	bankGenState.Balances = banktypes.SanitizeGenesisBalances(bankGenState.Balances)

	bankGenStateBz, err := cdc.MarshalJSON(bankGenState)
	if err != nil {
		return appState, fmt.Errorf("failed to marshal bank genesis state: %w", err)
	}

	appState[banktypes.ModuleName] = bankGenStateBz
	return appState, nil
}

func generateKeyring(accts ...string) keyring.Keyring {
	kb := keyring.NewInMemory()

	for _, acc := range accts {
		_, _, err := kb.NewMnemonic(acc, keyring.English, "", "", hd.Secp256k1)
		if err != nil {
			return nil
		}
	}

	_, err := kb.NewAccount(testAccName, testMnemo, "1234", "", hd.Secp256k1)
	if err != nil {
		panic(err)
	}

	return kb
}

// GenerateKeyringSigner creates a types.KeyringSigner with keys generated for
// the provided accounts
func GenerateKeyringSigner(acct string) *types.KeyringSigner {
	kr := generateKeyring(acct)
	return types.NewKeyringSigner(kr, acct, testChainID)
}

const (
	// nolint:lll
	testMnemo   = `ramp soldier connect gadget domain mutual staff unusual first midnight iron good deputy wage vehicle mutual spike unlock rocket delay hundred script tumble choose`
	testAccName = "test-account"
	testChainID = "test-chain-1"
)

// NewDefaultGenesisState generates the default state for the application.
func NewDefaultGenesisState(cdc codec.JSONCodec) app.GenesisState {
	return app.ModuleBasics.DefaultGenesis(cdc)
}
