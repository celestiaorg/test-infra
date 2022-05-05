package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/celestiaorg/celestia-app/app"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/cosmos/cosmos-sdk/x/staking/client/cli"
	"github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/cosmos/go-bip39"
	"github.com/tendermint/spm/cosmoscmd"
	"github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/privval"
	"github.com/tendermint/tendermint/proxy"
	tmtypes "github.com/tendermint/tendermint/types"
	db "github.com/tendermint/tm-db"

	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
)

const (
	keyPath           string = "/Users/bidon4/testground/celestia-app/keys"
	statePath         string = "/Users/bidon4/testground/celestia-app/state"
	path              string = "/Users/bidon4/testground/celestia-app"
	defaultValKeyType        = tmtypes.ABCIPubKeyTypeSecp256k1
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
	runenv.RecordMessage("config***********")
	_, valPubKey, err := initGenesis(cfg)
	if err != nil {
		return err
	}
	runenv.RecordMessage("config passed")

	runenv.RecordMessage("CREATING ACCOUNT")
	acc, err := addAccount(cfg, valPubKey)
	if err != nil {
		return err
	}

	runenv.RecordMessage("Account Genesis Creation")
	err = addAccGenesis(acc.GetAddress(), cfg)
	if err != nil {
		return err
	}

	err = genTx(acc.GetAddress(), cfg)
	if err != nil {
		return err
	}

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

func initGenesis(cfg *config.Config) (string, cryptotypes.PubKey, error) {
	interfaceRegistry := cdctypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(interfaceRegistry)
	mbm := module.NewBasicManager()

	cfg.Moniker = "simba"
	chainID := "tia-test"
	nodeID, valPubKey, err := genutil.InitializeNodeValidatorFilesFromMnemonic(cfg, "")
	if err != nil {
		return "", nil, err
	}

	genFile := cfg.GenesisFile()
	appState, err := json.MarshalIndent(mbm.DefaultGenesis(cdc), "", " ")
	if err != nil {
		return "", nil, errors.Wrap(err, "Failed to marshall default genesis state")
	}
	genDoc := &tmtypes.GenesisDoc{}
	if _, err := os.Stat(genFile); err != nil {
		if !os.IsNotExist(err) {
			return "", nil, err
		}
	} else {
		genDoc, err = tmtypes.GenesisDocFromFile(genFile)
		if err != nil {
			return "", nil, errors.Wrap(err, "Failed to read genesis doc from file")
		}
	}

	genDoc.ChainID = chainID
	genDoc.Validators = nil
	genDoc.AppState = appState

	if err = genutil.ExportGenesisFile(genDoc, genFile); err != nil {
		return "", nil, errors.Wrap(err, "Failed to export gensis file")
	}

	return nodeID, valPubKey, nil
}

func addAccount(cfg *config.Config, pk cryptotypes.PubKey) (keyring.Info, error) {
	name := "lion"

	var kr keyring.Keyring
	kr, err := keyring.New("celes", keyring.BackendTest, path, nil)
	if err != nil {
		return nil, err
	}

	entropySeed, err := bip39.NewEntropy(256)
	if err != nil {
		return nil, err
	}

	mnemonic, err := bip39.NewMnemonic(entropySeed)
	if err != nil {
		return nil, err
	}

	hdPath := hd.CreateHDPath(118, 0, 0).String()

	fmt.Println("#########Account#############")

	// err = kr.Delete(name)
	// if err != nil {
	// 	return nil, err
	// }
	fmt.Println("#########Begin Creatinion of Accoutt#############")
	info, err := kr.NewAccount(name, mnemonic, "", hdPath, hd.Ed25519Type)
	if err != nil {
		return nil, err
	}
	fmt.Println("#########Begin asdasdasd of Accoutt#############")
	fmt.Println(info.GetName())
	fmt.Println(info.GetPubKey().String())

	// info, err = kr.SavePubKey(name, pk, hd.Secp256k1.Name())
	// if err != nil {
	// 	return nil, err
	// }

	fmt.Println(info.GetAddress().String())
	fmt.Println(info.GetPubKey().String())
	fmt.Println(pk.String())

	return info, nil

}

func addAccGenesis(addr sdk.AccAddress, cfg *config.Config) error {
	fmt.Println(addr.String())
	fmt.Println("----------BEGIN ACCOUNT GENESIS-----------")
	interfaceRegistry := cdctypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(interfaceRegistry)
	coins, err := sdk.ParseCoinsNormalized("1000000000celes")
	if err != nil {
		return fmt.Errorf("failed to parse coins: %w", err)
	}
	// create concrete account type based on input parameters
	var genAccount authtypes.GenesisAccount

	balances := banktypes.Balance{Address: addr.String(), Coins: coins.Sort()}
	genAccount = authtypes.NewBaseAccount(addr, nil, 0, 0)

	if err := genAccount.Validate(); err != nil {
		return fmt.Errorf("failed to validate new genesis account: %w", err)
	}

	genFile := cfg.GenesisFile()
	appState, genDoc, err := genutiltypes.GenesisStateFromGenFile(genFile)
	if err != nil {
		return fmt.Errorf("failed to unmarshal genesis state: %w", err)
	}
	authGenState := authtypes.GetGenesisStateFromAppState(cdc, appState)

	accs, err := authtypes.UnpackAccounts(authGenState.Accounts)
	if err != nil {
		return fmt.Errorf("failed to get accounts from any: %w", err)
	}

	if accs.Contains(addr) {
		return fmt.Errorf("cannot add account at existing address %s", addr)
	}

	// Add the new account to the set of genesis accounts and sanitize the
	// accounts afterwards.
	accs = append(accs, genAccount)
	fmt.Println(accs)
	accs = authtypes.SanitizeGenesisAccounts(accs)

	genAccs, err := authtypes.PackAccounts(accs)
	if err != nil {
		return fmt.Errorf("failed to convert accounts into any's: %w", err)
	}
	authGenState.Accounts = genAccs

	authGenStateBz, err := cdc.MarshalJSON(&authGenState)
	if err != nil {
		return fmt.Errorf("failed to marshal auth genesis state: %w", err)
	}

	appState[authtypes.ModuleName] = authGenStateBz

	bankGenState := banktypes.GetGenesisStateFromAppState(cdc, appState)
	bankGenState.Balances = append(bankGenState.Balances, balances)
	bankGenState.Balances = banktypes.SanitizeGenesisBalances(bankGenState.Balances)
	bankGenState.Supply = bankGenState.Supply.Add(balances.Coins...)

	bankGenStateBz, err := cdc.MarshalJSON(bankGenState)
	if err != nil {
		return fmt.Errorf("failed to marshal bank genesis state: %w", err)
	}

	appState[banktypes.ModuleName] = bankGenStateBz

	appStateJSON, err := json.Marshal(appState)
	if err != nil {
		return fmt.Errorf("failed to marshal application genesis state: %w", err)
	}

	genDoc.AppState = appStateJSON

	err = genutil.ExportGenesisFile(genDoc, genFile)
	if err != nil {
		return err
	}

	return nil

}

func genTx(addr sdk.AccAddress, cfg *config.Config) error {

	nodeID, valPubKey, err := genutil.InitializeNodeValidatorFiles(cfg)
	if err != nil {
		return errors.Wrap(err, "failed to initialize node validator files")
	}

	genDoc, err := tmtypes.GenesisDocFromFile(cfg.GenesisFile())
	if err != nil {
		return errors.Wrapf(err, "failed to read genesis doc file %s", cfg.GenesisFile())
	}

	var genesisState map[string]json.RawMessage
	err = json.Unmarshal(genDoc.AppState, &genesisState)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal genesis state")
	}

	valCfg := cli.TxCreateValidatorConfig{
		NodeID:                  nodeID,
		PubKey:                  valPubKey,
		Moniker:                 cfg.Moniker,
		ChainID:                 genDoc.ChainID,
		CommissionRate:          "0.1",
		CommissionMaxRate:       "0.2",
		CommissionMaxChangeRate: "0.01",
		MinSelfDelegation:       "1",
	}

	desc := types.NewDescription(cfg.Moniker, "", "", "", "")

	amount := "50000000celes"
	coins, err := sdk.ParseCoinNormalized(amount)
	if err != nil {
		return errors.Wrap(err, "failed to parse coins")
	}

	valCfg.Amount = amount

	rate, err := sdk.NewDecFromStr(valCfg.CommissionRate)
	if err != nil {
		return err
	}

	maxRate, err := sdk.NewDecFromStr(valCfg.CommissionMaxRate)
	if err != nil {
		return err
	}

	maxChangeRate, err := sdk.NewDecFromStr(valCfg.CommissionMaxChangeRate)
	if err != nil {
		return err
	}

	commission := types.NewCommissionRates(rate, maxRate, maxChangeRate)

	minSelfDelegation, _ := sdk.NewIntFromString(valCfg.MinSelfDelegation)

	msg, err := types.NewMsgCreateValidator(
		sdk.ValAddress(addr), valCfg.PubKey, coins, desc, commission, minSelfDelegation,
	)
	if err != nil {
		return err
	}

	interfaceRegistry := cdctypes.NewInterfaceRegistry()
	protoCodec := codec.NewProtoCodec(interfaceRegistry)
	txConfig := tx.NewTxConfig(protoCodec, tx.DefaultSignModes)
	txBuilder := txConfig.NewTxBuilder()

	err = txBuilder.SetMsgs(msg)
	if err != nil {
		return err
	}
	genTxs := []sdk.Tx{txBuilder.GetTx()}

	appGenesisState, err := genutil.SetGenTxsInAppGenesisState(protoCodec, txConfig.TxJSONEncoder(), genesisState, genTxs)
	if err != nil {
		return err
	}

	appState, err := json.MarshalIndent(appGenesisState, "", "  ")
	if err != nil {
		return err
	}

	genDoc.AppState = appState
	err = genutil.ExportGenesisFile(genDoc, cfg.GenesisFile())
	if err != nil {
		return err
	}

	return nil
}

// func collectGenTx() error {

// }

func setApp() (*app.App, error) {
	db, _ := openDB(path)

	skipHeights := make(map[int64]bool)

	encCfg := cosmoscmd.MakeEncodingConfig(app.ModuleBasics)

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

	return tiaApp, nil
}

func openDB(rootDir string) (db.DB, error) {
	dataDir := filepath.Join(rootDir, "data")
	return sdk.NewLevelDB("application", dataDir)
}
