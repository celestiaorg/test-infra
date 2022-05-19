package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/celestiaorg/celestia-app/app"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/viper"
	"github.com/testground/sdk-go/network"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
	"github.com/testground/sdk-go/sync"

	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
)

var testcases = map[string]interface{}{
	"capp-1": run.InitializedTestCaseFn(runSync),
}

func main() {
	run.InvokeMap(testcases)
}

type ValidatorNode struct {
	PubKey string
	IP     net.IP
}

func AddPersistentPeers(path string, peers []string) error {

	var peersStr bytes.Buffer
	var port int = 26656
	var separator string = ","
	for k, peer := range peers {
		if k == (len(peers) - 1) {
			separator = ""
		}
		peersStr.WriteString(fmt.Sprintf("%s:%d%s", peer, port, separator))
	}

	fh, err := os.OpenFile(path, os.O_RDWR, 0777)
	if err != nil {
		return err
	}

	viper.SetConfigType("toml")
	err = viper.ReadConfig(fh)
	if err != nil {
		return err
	}

	viper.Set("p2p.persistent-peers", peersStr.String())
	err = viper.WriteConfigAs(path)
	if err != nil {
		return err
	}

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
	switch seq {
	case 1:
		home = runenv.StringParam("app1")
	case 2:
		home = runenv.StringParam("app2")
	case 3:
		home = runenv.StringParam("app3")
	case 4:
		home = runenv.StringParam("app4")
	case 5:
		home = runenv.StringParam("app5")
	case 6:
		home = runenv.StringParam("app6")
	case 7:
		home = runenv.StringParam("app7")
	case 8:
		home = runenv.StringParam("app8")
	case 9:
		home = runenv.StringParam("app9")
	case 10:
		home = runenv.StringParam("app10")
	}

	// if seq == 1 {
	// 	home = runenv.StringParam("app1")
	// } else {
	// 	home = runenv.StringParam("app2")
	// }
	fmt.Println(home)
	cmd := NewRootCmd()
	const envPrefix = "CELESTIA"

	scrapStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetArgs([]string{"tendermint", "show-node-id", "--home", home})
	if err := svrcmd.Execute(cmd, envPrefix, app.DefaultNodeHome); err != nil {
		return err
	}

	w.Close()
	outStr, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	os.Stdout = scrapStdout

	valPubKey := string(outStr)
	valPubKey = strings.ReplaceAll(valPubKey, "\n", "")
	fmt.Println("My node-id is:", valPubKey)
	fmt.Println("My config IP is:", config.IPv4.IP)
	valt := sync.NewTopic("validator-info", &ValidatorNode{})
	client.Publish(ctx, valt, &ValidatorNode{valPubKey, config.IPv4.IP})

	rdySt := sync.State("ready")
	client.MustSignalEntry(ctx, rdySt)
	<-client.MustBarrier(ctx, rdySt, runenv.TestInstanceCount).C

	valCh := make(chan *ValidatorNode)
	client.Subscribe(ctx, valt, valCh)

	var persPeers []string
	for i := 0; i < runenv.TestInstanceCount; i++ {
		val := <-valCh
		runenv.RecordMessage("Validator Received: %s, %s", val.IP, val.PubKey)
		if !val.IP.Equal(config.IPv4.IP) {
			persPeers = append(persPeers, fmt.Sprintf("%s@%s", val.PubKey, val.IP.To4().String()))
		}
	}

	configPath := filepath.Join(home, "config", "config.toml")
	err = AddPersistentPeers(configPath, persPeers)
	if err != nil {
		return err
	}

	cmd.ResetFlags()
	cmd.Flags().Set(flags.FlagHome, "")

	cmd.SetErr(os.Stdout)
	cmd.SetArgs([]string{"start", "--home", home, "--log_level", "info"})

	if err := svrcmd.Execute(cmd, envPrefix, app.DefaultNodeHome); err != nil {
		return err
	}

	return nil
}
