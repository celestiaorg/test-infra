package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/celestiaorg/celestia-app/app"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/testground/sdk-go/network"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
	"github.com/testground/sdk-go/sync"

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

type ValidatorNode struct {
	PubKey string
	IP     string
}

func AddPersistentPeers(path string, peers ...string) error {
	input, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	fmt.Println("file read successfuly")
	var peersStr string
	var port int = 26656
	var separator string = ","
	for k, peer := range peers {
		if k == (len(peers) - 1) {
			separator = ""
		}
		peersStr += fmt.Sprintf("%s:%d%s", peer, port, separator)
	}
	lines := strings.Split(string(input), "\n")

	for i, line := range lines {
		if strings.Contains(line, "persistent_peers") {
			lines[i] = fmt.Sprintf(`persistent_peers="%s"`, peersStr)
		}
	}
	output := strings.Join(lines, "\n")
	err = ioutil.WriteFile(path, []byte(output), 0644)
	if err != nil {
		return err
	}
	fmt.Println("file wrotte successfuly")

	return nil
}

func runSync(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	// const home string = "/Users/bidon4/.celestia-app-1"
	ctx := context.Background()
	client := initCtx.SyncClient
	netclient := network.NewClient(client, runenv)

	netclient.MustWaitNetworkInitialized(ctx)

	config := network.Config{
		// Control the "default" network. At the moment, this is the only network.
		Network: "default",

		// Enable this network. Setting this to false will disconnect this test
		// instance from this network. You probably don't want to do that.
		Enable: true,

		// Set the traffic shaping characteristics.
		Default: network.LinkShape{
			Latency:   100 * time.Millisecond,
			Bandwidth: 1 << 20, // 1Mib
		},

		// Set what state the sidecar should signal back to you when it's done.
		CallbackState: "network-configured",
	}

	topic := sync.NewTopic("ip-allocation", "")
	seq := client.MustPublish(ctx, topic, "")

	config.IPv4 = runenv.TestSubnet
	// Use the sequence number to fill in the last two octets.
	//
	// NOTE: Be careful not to modify the IP from `runenv.TestSubnet`.
	// That could trigger undefined behavior.
	ipC := byte((seq >> 8) + 1)
	ipD := byte(seq)
	config.IPv4.IP = append(config.IPv4.IP[0:2:2], ipC, ipD)

	err := netclient.ConfigureNetwork(ctx, &config)
	if err != nil {
		runenv.RecordCrash(err)
		return err
	}

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
		home,
		app.Name,
		app.ModuleBasics,
		appBuilder,
	)
	scrapStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetArgs([]string{"tendermint", "show-node-id", "--home", home})
	if err := svrcmd.Execute(cmd, app.DefaultNodeHome); err != nil {
		return err
	}

	w.Close()
	outStr, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	os.Stdout = scrapStdout

	valPubKey := string(outStr)
	fmt.Println("My node-id is:", valPubKey)
	fmt.Println("My config IP is:", config.IPv4.IP)
	valt := sync.NewTopic("validator-info", &ValidatorNode{})
	client.Publish(ctx, valt, &ValidatorNode{valPubKey, config.IPv4.IP.String()})

	rdySt := sync.State("ready")
	client.MustSignalEntry(ctx, rdySt)
	<-client.MustBarrier(ctx, rdySt, runenv.TestInstanceCount).C

	valCh := make(chan *ValidatorNode)
	client.Subscribe(ctx, valt, valCh)

	for i := 0; i < runenv.TestInstanceCount; i++ {
		val := <-valCh
		runenv.RecordMessage("Validator Received: %s, %s", val.IP, val.PubKey)
		if val.IP != config.IPv4.IP.String() {
			configPath := filepath.Join(home, "config", "config.toml")
			fmt.Println(configPath)
			AddPersistentPeers(configPath, val.PubKey+"@"+val.IP)
		}
	}

	cmd.ResetFlags()
	cmd.Flags().Set(flags.FlagHome, "")
	out = bytes.NewBuffer(nil)
	cmd.SetOut(out)
	cmd.SetArgs([]string{"start", "--home", home})

	if err := svrcmd.Execute(cmd, app.DefaultNodeHome); err != nil {
		return err
	}

	// outStr, err = ioutil.ReadAll(out)
	// if err != nil {
	// 	return err
	// }
	// fmt.Println("Dude ", string(outStr))

	// time.Sleep(5 * time.Second)

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
