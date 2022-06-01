package main

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/celestiaorg/celestia-node/node"
	"github.com/celestiaorg/test-infra/testkit/appkit"
	"github.com/celestiaorg/test-infra/testkit/nodekit"

	"github.com/testground/sdk-go/network"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
	"github.com/testground/sdk-go/sync"
)

var testcases = map[string]interface{}{
	"capp-1": run.InitializedTestCaseFn(runSync),
}

func main() {
	run.InvokeMap(testcases)
}

func runSync(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
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
			Latency:   200 * time.Millisecond,
			Jitter:    100 * time.Millisecond,
			Loss:      2,
			Corrupt:   2,
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

	// var home string
	home := runenv.StringParam(fmt.Sprintf("app%d", seq))

	fmt.Println(home)
	cmd := appkit.NewRootCmd()

	nodeId, err := appkit.GetNodeId(cmd, home)
	if err != nil {
		runenv.RecordCrash(err)
		return err
	}

	valt := sync.NewTopic("validator-info", &appkit.ValidatorNode{})
	client.Publish(ctx, valt, &appkit.ValidatorNode{nodeId, config.IPv4.IP})

	rdySt := sync.State("ready")
	client.MustSignalEntry(ctx, rdySt)
	<-client.MustBarrier(ctx, rdySt, runenv.TestInstanceCount).C

	valCh := make(chan *appkit.ValidatorNode)
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
	err = appkit.AddPersistentPeers(configPath, persPeers)
	if err != nil {
		return err
	}

	go appkit.StartNode(cmd, home)

	time.Sleep(40 * time.Second)

	h, err := appkit.GetBlockHashByHeight(1)

	fmt.Print(h)

	ndhome := fmt.Sprintf("/.celestia-bridge-%d", seq)
	nd, err := nodekit.NewNode(ndhome, node.Bridge, node.WithTrustedHash(h), node.WithRemoteCore("tcp", "127.0.0.1:26657"))
	if err != nil {
		return err
	}
	nd.Start(ctx)
	if err != nil {
		return err
	}
	time.Sleep(100 * time.Second)
	return nil
}
