package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
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

type p2p struct {
	Laddr                  string `toml:"laddr"`
	ExtAddr                string `toml:"external_address"`
	Seeds                  string `toml:"seeds"`
	PersistentPeers        string `toml:"persistent_peers"`
	UPNP                   string `toml:"upnp"`
	AddrBookFile           string `toml:"addr_book_file"`
	AddrBookStrict         bool   `toml:"addr_book_strict"`
	MaxInboundPeers        int    `toml:"max_num_inbound_peers"`
	MaxOutboundPeers       int    `toml:"max_num_outbound_peers"`
	UnconditionalPeersIds  string `toml:"unconditional_peer_ids"`
	PersistentPeersMaxDial string `toml:"persistent_peers_max_dial_period"`
	FlushThrottle          string `toml:"flush_throttle_timeout"`
	MaxPacketPayload       int    `toml:"max_packet_msg_payload_size"`
	SendRate               int64  `toml:"send_rate"`
	RecvRate               int64  `toml:"recv_rate"`
	Pex                    bool   `toml:"pex"`
	SeedMode               bool   `toml:"seed_mode"`
	PrivatePeerIds         string `toml:"private_peer_ids"`
	AllowDuplicateIP       bool   `toml:"allow_duplicate_ip"`
	HandshakeTimeout       string `toml:"handshake_timeout"`
	DialTimeout            string `toml:"dial_timeout"`
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
		runenv.RecordMessage("Validator Received: ", val.IP, val.PubKey)
	}

	// var p2pcfg p2p
	// _, err = toml.DecodeFile("core-configs/celestia-app-1/config/config.toml", &p2pcfg)
	// if err != nil {
	// 	return err
	// }

	// fmt.Println(p2pcfg)

	cmd.Flags().Set(flags.FlagHome, "")
	out = bytes.NewBuffer(nil)
	cmd.SetOut(out)
	cmd.SetArgs([]string{"start", "--home", home})

	if err := svrcmd.Execute(cmd, app.DefaultNodeHome); err != nil {
		return err
	}

	outStr, err = ioutil.ReadAll(out)
	if err != nil {
		return err
	}
	fmt.Println("Dude ", string(outStr))

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
