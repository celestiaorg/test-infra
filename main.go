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
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"

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

type BridgeId struct {
	ID          int
	Maddr       string
	TrustedHash string
	Amount      int
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

	appt := sync.NewTopic("app-id", &AppId{})
	bridget := sync.NewTopic("bridge-id", &BridgeId{})
	// finisht := sync.NewTopic("finish-test", bool)
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

		rdySt := sync.State("appReady")
		appseq := client.MustSignalEntry(ctx, rdySt)
		client.Publish(ctx, appt, &AppId{int(appseq), config.IPv4.IP})
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

		//TODO(@Bidon15): should be a goroutine as this is blocking func
		appkit.StartNode(cmd, home)

		//TODO(@Bidon15): subscribe to an event that tells that the test is finished from node part

	} else if runenv.TestGroupID == "bridge" {
		// os.Setenv("GOLOG_FILE", "/.bridge.log")
		os.Setenv("GOLOG_OUTPUT", "stdout")

		time.Sleep(10 * time.Second)
		level, err := logging.LevelFromString("INFO")
		if err != nil {
			return err
		}
		logs.SetAllLoggers(level)
		appIPCh := make(chan *AppId)
		client.Subscribe(ctx, appt, appIPCh)
		for i := 1; i <= runenv.TestGroupInstanceCount; i++ {
			appIP := <-appIPCh
			if appIP.ID == int(initCtx.GroupSeq) {
				h, err := appkit.GetBlockHashByHeight(appIP.IP, 1)
				if err != nil {
					return err
				}
				runenv.RecordMessage("Block#1 Hash: %s", h)

				ndhome := fmt.Sprintf("/.celestia-bridge-%d", initCtx.GroupSeq)
				rc := fmt.Sprintf("%s:26657", appIP.IP.To4().String())
				runenv.RecordMessage(rc)
				nd, err := nodekit.NewNode(ndhome, node.Bridge, config.IPv4.IP, node.WithTrustedHash(h), node.WithRemoteCore("tcp", rc))
				if err != nil {
					return err
				}

				ndCtx := context.Background()
				nd.Start(ndCtx)
				if err != nil {
					return err
				}

				eh, err := nd.HeaderServ.GetByHeight(ndCtx, uint64(4))
				if err != nil {
					return err
				}

				runenv.RecordMessage("Reached Block#4 contains Hash: %s", eh.Commit.BlockID.Hash.String())

				//create a new subscription to publish bridge's multiaddress to light nodes
				addrs, err := peer.AddrInfoToP2pAddrs(host.InfoFromHost(nd.Host))
				if err != nil {
					return err
				}
				rdySt := sync.State("bridgeReady")
				client.MustSignalEntry(ctx, rdySt)
				bseq, _ := client.Publish(ctx, bridget, &BridgeId{int(initCtx.GroupSeq), addrs[0].String(), h, runenv.TestGroupInstanceCount})
				<-client.MustBarrier(ctx, rdySt, runenv.TestGroupInstanceCount).C
				// bseq := client.MustPublish(ctx, bridget, &BridgeId{int(initCtx.GroupSeq), addrs[0].String(), h, runenv.TestGroupInstanceCount})

				runenv.RecordMessage("%s published bridge id", int(bseq))
			}
		}

		time.Sleep(1 * time.Minute)

	} else if runenv.TestGroupID == "full" {
		bridgeCh := make(chan *BridgeId)
		client.MustSubscribe(ctx, bridget, bridgeCh)

		for i := 1; i <= runenv.TestGroupInstanceCount; i++ {
			ndhome := fmt.Sprintf("/.celestia-full-%d", initCtx.GroupSeq)
			runenv.RecordMessage(ndhome)
			bridge := <-bridgeCh
			if int(initCtx.GroupSeq) == bridge.ID {
				nd, err := nodekit.NewNode(ndhome, node.Full, config.IPv4.IP, node.WithTrustedHash(bridge.TrustedHash), node.WithTrustedPeers(bridge.Maddr))
				if err != nil {
					return err
				}
				ndCtx := context.Background()
				err = nd.Start(ndCtx)
				if err != nil {
					return err
				}

				eh, err := nd.HeaderServ.GetByHeight(ndCtx, uint64(12))
				if err != nil {
					return err
				}
				runenv.RecordMessage("Reached Block#12 contains Hash: %s", eh.Commit.BlockID.Hash.String())
			}
		}

	} else if runenv.TestGroupID == "light" {
		bridgeCh := make(chan *BridgeId)
		client.Subscribe(ctx, bridget, bridgeCh)

		for i := 0; i < runenv.TestGroupInstanceCount; i++ {
			var nd *node.Node
			ndhome := fmt.Sprintf("/.celestia-light-%d", i)
			bridge := <-bridgeCh
			if i < runenv.TestGroupInstanceCount/3 && bridge.ID == 1 {
				nd, err = nodekit.NewNode(ndhome, node.Light, config.IPv4.IP, node.WithTrustedHash(bridge.TrustedHash), node.WithTrustedPeers(bridge.Maddr))
			} else if i >= runenv.TestGroupInstanceCount/3 && i < runenv.TestGroupInstanceCount/3*2 && bridge.ID == 2 {
				nd, err = nodekit.NewNode(ndhome, node.Light, config.IPv4.IP, node.WithTrustedHash(bridge.TrustedHash), node.WithTrustedPeers(bridge.Maddr))
			} else {
				nd, err = nodekit.NewNode(ndhome, node.Light, config.IPv4.IP, node.WithTrustedHash(bridge.TrustedHash), node.WithTrustedPeers(bridge.Maddr))
			}
			if err != nil {
				return err
			}

			ndCtx := context.Background()
			err = nd.Start(ndCtx)
			if err != nil {
				return err
			}

			eh, err := nd.HeaderServ.GetByHeight(ndCtx, uint64(12))
			if err != nil {
				return err
			}

			runenv.RecordMessage("Reached Block#12 contains Hash: %s", eh.Commit.BlockID.Hash.String())
		}

	}
	return nil
}
