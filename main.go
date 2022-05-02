package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/celestiaorg/celestia-app/app"
	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/spm/cosmoscmd"
	"github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/log"
	tmrand "github.com/tendermint/tendermint/libs/rand"
	"github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/privval"
	"github.com/tendermint/tendermint/proxy"
	"github.com/tendermint/tendermint/types"
	tmtime "github.com/tendermint/tendermint/types/time"
	db "github.com/tendermint/tm-db"

	"github.com/celestiaorg/celestia-node/libs/utils"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
)

const (
	keyPath           string = "/Users/bidon4/testground/celestia-app/keys"
	statePath         string = "/Users/bidon4/testground/celestia-app/state"
	path              string = "/Users/bidon4/testground/celestia-app"
	defaultValKeyType        = types.ABCIPubKeyTypeSecp256k1
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
	runenv.ToEnvVars()
	err := os.Setenv("HOME", "/Users/bidon4")

	if err != nil {
		return err
	}

	cfg, err := Init(path)
	if err != nil {
		return err
	}

	runenv.RecordMessage(cfg.RootDir)

	capp := setApp()

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

	if err != nil {
		return err
	}

	return nil
}

func setApp() *app.App {
	db, _ := openDB(path)

	skipHeights := make(map[int64]bool)

	tiaApp := app.New(
		log.NewTMLogger(log.NewSyncWriter(os.Stdout)),
		db,
		nil,
		true,
		skipHeights,
		path,
		0,
		cosmoscmd.MakeEncodingConfig(app.ModuleBasics),
		simapp.EmptyAppOptions{},
	)
	return tiaApp
}

func openDB(rootDir string) (db.DB, error) {
	dataDir := filepath.Join(rootDir, "data")
	return sdk.NewLevelDB("application", dataDir)
}

func Init(path string) (*config.Config, error) {

	var err error
	cfg := config.DefaultConfig()
	cfg.SetRoot(path)
	cfgPath := configPath(path)
	if !utils.Exists(cfgPath) {
		err = SaveConfig(cfgPath, cfg)
		if err != nil {
			return nil, fmt.Errorf("core: can't write config: %w", err)
		}
		// log.Info("New config is generated")
	} else {
		return nil, fmt.Errorf("core: cfg already exists")
	}
	// 2 - ensure private validator key
	var pv *privval.FilePV
	keyPath := cfg.PrivValidatorKeyFile()
	if !utils.Exists(keyPath) {
		pv = privval.GenFilePV(keyPath, cfg.PrivValidatorStateFile())
		pv.Save()
		// log.Info("New consensus private key is generated")
	} else {
		pv = privval.LoadFilePV(keyPath, cfg.PrivValidatorStateFile())
		// log.Info("Consensus private key already exists")
	}
	// 3 - ensure private p2p key
	_, err = p2p.LoadOrGenNodeKey(cfg.NodeKeyFile())
	if err != nil {
		return nil, fmt.Errorf("core: error with key: %w", err)
	}
	// 4 - ensure genesis
	genPath := cfg.GenesisFile()
	if !utils.Exists(genPath) {
		// log.Info("New stub genesis document is generated")
		// log.Warn("Stub genesis document must not be used in production environment!")
		pubKey, err := pv.GetPubKey()
		if err != nil {
			return nil, fmt.Errorf("can't get pubkey: %w", err)
		}

		params := types.DefaultConsensusParams()
		params.Validator.PubKeyTypes = []string{defaultValKeyType}
		genDoc := types.GenesisDoc{
			ChainID:         fmt.Sprintf("localnet-%v", tmrand.Str(6)),
			GenesisTime:     tmtime.Now(),
			ConsensusParams: params,
			Validators: []types.GenesisValidator{{
				Address: pubKey.Address(),
				PubKey:  pubKey,
				Power:   10,
			}},
		}

		err = genDoc.SaveAs(genPath)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("core: cfg already exists")
	}

	return cfg, nil
}

func configPath(base string) string {
	return filepath.Join(base, "config.toml")
}

func SaveConfig(path string, cfg *config.Config) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return toml.NewEncoder(f).Encode(cfg)
}
