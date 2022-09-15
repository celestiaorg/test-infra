package nodesync

import (
	"context"
	"fmt"
	"time"

	"github.com/celestiaorg/celestia-node/node"
	"github.com/celestiaorg/test-infra/testkit"
	"github.com/celestiaorg/test-infra/testkit/appkit"
	"github.com/celestiaorg/test-infra/testkit/nodekit"
	"github.com/celestiaorg/test-infra/tests/common"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/testground/sdk-go/network"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
	"github.com/testground/sdk-go/sync"
)

func RunBridgeNode(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*10)
	defer cancel()

	err := nodekit.SetLoggersLevel("INFO")
	if err != nil {
		return err
	}

	syncclient := initCtx.SyncClient
	netclient := network.NewClient(syncclient, runenv)

	netclient.MustWaitNetworkInitialized(ctx)

	config := network.Config{
		Network: "default",
		Enable:  true,
		Default: network.LinkShape{
			Latency:   time.Duration(runenv.IntParam("latency")),
			Bandwidth: common.GetBandwidthValue(runenv.StringParam("bandwidth")),
		},
		CallbackState: "network-configured",
		RoutingPolicy: network.AllowAll,
	}

	config.IPv4 = runenv.TestSubnet

	// using the assigned `GlobalSequencer` id per each of instance
	// to fill in the last 2 octects of the new IP address for the instance
	ipC := byte((initCtx.GlobalSeq >> 8) + 1)
	ipD := byte(initCtx.GlobalSeq)
	config.IPv4.IP = append(config.IPv4.IP[0:2:2], ipC, ipD)

	err = netclient.ConfigureNetwork(ctx, &config)
	if err != nil {
		return err
	}

	appInfoCh := make(chan *testkit.AppNodeInfo, runenv.IntParam("validator"))
	sub, err := syncclient.Subscribe(ctx, testkit.AppNodeTopic, appInfoCh)
	if err != nil {
		return err
	}

	appNode, err := func(total int) (*testkit.AppNodeInfo, error) {
		for {
			select {
			case err = <-sub.Done():
				if err != nil {
					return nil, fmt.Errorf("no app has been sent for this bridge to connect to remotely")
				}
			case appInfo := <-appInfoCh:
				if (appInfo.ID % total) == (int(initCtx.GroupSeq) % total) {
					return appInfo, nil
				}
			}
		}
	}(runenv.IntParam("validator"))

	if err != nil {
		return err
	}

	h, err := appkit.GetBlockHashByHeight(appNode.IP, 1)
	if err != nil {
		return err
	}
	runenv.RecordMessage("Block#1 Hash: %s", h)

	ndhome := fmt.Sprintf("/.celestia-bridge-%d", initCtx.GlobalSeq)
	runenv.RecordMessage(appNode.IP.To4().String())

	ip, err := initCtx.NetClient.GetDataNetworkIP()
	if err != nil {
		return err
	}

	nd, err := nodekit.NewNode(ndhome, node.Bridge, ip, h,
		node.WithRemoteCoreIP(appNode.IP.To4().String()),
		node.WithRemoteCorePort("26657"),
	)
	if err != nil {
		return err
	}

	err = nd.Start(ctx)
	if err != nil {
		return err
	}

	eh, err := nd.HeaderServ.GetByHeight(ctx, uint64(2))
	if err != nil {
		return err
	}

	runenv.RecordMessage("Reached Block#2 contains Hash: %s", eh.Commit.BlockID.Hash.String())

	//create a new subscription to publish bridge's multiaddress to full/light nodes
	addrs, err := peer.AddrInfoToP2pAddrs(host.InfoFromHost(nd.Host))
	if err != nil {
		return err
	}

	runenv.RecordMessage("Publishing bridgeID %d", int(initCtx.GroupSeq))
	runenv.RecordMessage("Publishing bridgeID Addr %s", addrs[0].String())

	_, err = syncclient.Publish(
		ctx,
		testkit.BridgeNodeTopic,
		&testkit.BridgeNodeInfo{
			ID:          int(initCtx.GroupSeq),
			Maddr:       addrs[0].String(),
			TrustedHash: h,
		},
	)
	if err != nil {
		return err
	}

	runenv.RecordMessage("Finished published bridgeID Addr %d", int(initCtx.GroupSeq))

	eh, err = nd.HeaderServ.GetByHeight(ctx, uint64(runenv.IntParam("block-height")))
	if err != nil {
		return err
	}
	runenv.RecordMessage("Reached Block#%d contains Hash: %s",
		runenv.IntParam("block-height"),
		eh.Commit.BlockID.Hash.String())

	if nd.HeaderServ.IsSyncing() {
		runenv.RecordFailure(fmt.Errorf("bridge node is still syncing the past"))
	}

	err = nd.Stop(ctx)
	if err != nil {
		return err
	}

	_, err = syncclient.SignalEntry(ctx, testkit.FinishState)
	if err != nil {
		return err
	}

	return nil
}

func GetBridgeNode(ctx context.Context, syncclient sync.Client, id int64, amountOfBridges int) (*testkit.BridgeNodeInfo, error) {
	bridgeCh := make(chan *testkit.BridgeNodeInfo, amountOfBridges)
	sub, err := syncclient.Subscribe(ctx, testkit.BridgeNodeTopic, bridgeCh)
	if err != nil {
		return nil, err
	}

	for {
		select {
		case err = <-sub.Done():
			if err != nil {
				return nil,
					fmt.Errorf("no bridge address has been sent to this light node to connect to")
			}
		case bridge := <-bridgeCh:
			fmt.Printf("Received Bridge ID = %d", bridge.ID)
			if (int(id) % amountOfBridges) == (bridge.ID % amountOfBridges) {
				return bridge, nil
			}
		}
	}

}
