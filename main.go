package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"time"

	"github.com/celestiaorg/celestia-app/app"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"

	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/tendermint/spm/cosmoscmd"
	"github.com/tendermint/tendermint/libs/log"
	dbm "github.com/tendermint/tm-db"
)

var testcases = map[string]interface{}{
	"capp-1": run.InitializedTestCaseFn(runSync),
}

func main() {
	run.InvokeMap(testcases)
}

func runSync(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	// const home string = "/Users/bidon4/.celestia-app-1"
	ctx := context.Background()
	client := initCtx.SyncClient
	seq := client.MustSignalAndWait(ctx, "home_alloc", runenv.TestInstanceCount)

	var home string
	if seq == 1 {
		home = runenv.StringParam("app1")
	} else {
		home = runenv.StringParam("app2")
	}
	fmt.Println(home)
	cmd, _ := cosmoscmd.NewRootCmd(
		app.Name,
		app.AccountAddressPrefix,
		// app.DefaultNodeHome,
		home,
		app.Name,
		app.ModuleBasics,
		appBuilder,
		// this line is used by starport scaffolding # root/arguments
	)
	cmd.Flags().Set(flags.FlagHome, "")
	out := bytes.NewBuffer(nil)
	cmd.SetOut(out)
	cmd.SetArgs([]string{"start", "--home", home})

	if err := svrcmd.Execute(cmd, app.DefaultNodeHome); err != nil {
		return err
	}

	outStr, err := ioutil.ReadAll(out)
	if err != nil {
		return err
	}
	fmt.Println(string(outStr))

	time.Sleep(5 * time.Second)

	return nil
}

func appBuilder(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	loadLatest bool,
	skipUpgradeHeights map[int64]bool,
	homePath string,
	invCheckPeriod uint,
	encodingConfig cosmoscmd.EncodingConfig,
	appOpts servertypes.AppOptions,
	baseAppOptions ...func(*baseapp.BaseApp),
) cosmoscmd.App {
	return app.New(
		logger,
		db,
		traceStore,
		loadLatest,
		skipUpgradeHeights,
		homePath,
		invCheckPeriod,
		encodingConfig,
		appOpts,
		baseAppOptions...,
	)
}
