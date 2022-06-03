package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/celestiaorg/celestia-node/logs"
	"github.com/celestiaorg/celestia-node/node"
	"github.com/celestiaorg/test-infra/testkit/appkit"
	"github.com/celestiaorg/test-infra/testkit/nodekit"
	logging "github.com/ipfs/go-log/v2"

	"github.com/testground/sdk-go/network"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
	"github.com/testground/sdk-go/sync"
)

var testcases = map[string]interface{}{
	"capp-3":  run.InitializedTestCaseFn(runSync),
	"capp-10": run.InitializedTestCaseFn(runSync),
}

func main() {
	run.InvokeMap(testcases)
}

type AppId struct {
	ID int
	IP net.IP
}

func runSync(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	os.Setenv("GOLOG_OUTPUT", "stdout")
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
			Latency: 200 * time.Millisecond,
			// Jitter:    100 * time.Millisecond,
			// Loss:      2,
			// Corrupt:   2,
			Bandwidth: 1 << 20, // 1Mib
		},

		// Set what state the sidecar should signal back to you when it's done.
		CallbackState: "network-configured",
		RoutingPolicy: network.AllowAll,
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

	appt := sync.NewTopic("app-ip", &AppId{})
	if runenv.TestGroupID == "app" {
		// var home string
		home := runenv.StringParam(fmt.Sprintf("app%d", initCtx.GroupSeq))

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
		seq := client.MustSignalEntry(ctx, rdySt)
		client.Publish(ctx, appt, &AppId{int(seq), config.IPv4.IP})
		<-client.MustBarrier(ctx, rdySt, runenv.TestGroupInstanceCount).C

		valCh := make(chan *appkit.ValidatorNode)
		client.Subscribe(ctx, valt, valCh)

		var persPeers []string
		for i := 0; i < runenv.TestGroupInstanceCount; i++ {
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

		appkit.StartNode(cmd, home)
	} else {
		time.Sleep(10 * time.Second)
		level, err := logging.LevelFromString("INFO")
		if err != nil {
			return err
		}
		logs.SetAllLoggers(level)
		appIPCh := make(chan *AppId)
		client.Subscribe(ctx, appt, appIPCh)
		for i := 0; i < runenv.TestGroupInstanceCount; i++ {
			appIP := <-appIPCh
			if appIP.ID == int(initCtx.GroupSeq) {
				h, err := appkit.GetBlockHashByHeight(appIP.IP, 1)
				fmt.Print(h)

				ndhome := fmt.Sprintf("/.celestia-bridge-%d", seq)
				rc := fmt.Sprintf("%s:26657", appIP.IP.To4().String())
				nd, err := nodekit.NewNode(ndhome, node.Bridge, node.WithTrustedHash(h), node.WithRemoteCore("tcp", rc))
				if err != nil {
					return err
				}

				ndCtx := context.Background()
				nd.Start(ndCtx)
				if err != nil {
					return err
				}

				eh, err := nd.HeaderServ.GetByHeight(ndCtx, uint64(6))
				if err != nil {
					return err
				}

				fmt.Println(eh.Commit.BlockID.Hash.String())
			}
		}

		// app is publishing an ip and it's instance number to the topic
		// node is reading the topic and compares the received instance number from the app with the instance it has to be the same
	}

	return nil
}
