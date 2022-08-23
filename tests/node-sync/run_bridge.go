package nodesync

import (
	"context"
	"fmt"
	"time"

	"github.com/celestiaorg/celestia-node/node"
	"github.com/celestiaorg/test-infra/testkit"
	"github.com/celestiaorg/test-infra/testkit/appkit"
	"github.com/celestiaorg/test-infra/testkit/nodekit"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/testground/sdk-go/network"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
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
			Bandwidth: 5 << 26, // 320Mib
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
					return nil, err
				}
			case <-ctx.Done():
				return nil, fmt.Errorf("no app has been sent for this bridge to connect to remotely")
			case appInfo := <-appInfoCh:
				if (appInfo.ID % total) == (int(initCtx.GlobalSeq) % total) {
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

	eh, err := nd.HeaderServ.GetByHeight(ctx, uint64(3))
	if err != nil {
		return err
	}

	runenv.RecordMessage("Reached Block#3 contains Hash: %s", eh.Commit.BlockID.Hash.String())

	//create a new subscription to publish bridge's multiaddress to full/light nodes
	addrs, err := peer.AddrInfoToP2pAddrs(host.InfoFromHost(nd.Host))
	if err != nil {
		return err
	}

	runenv.RecordMessage("Publishing bridgeID %d", int(initCtx.GlobalSeq))
	runenv.RecordMessage("Publishing bridgeID Addr %s", addrs[0].String())

	_, err = syncclient.Publish(
		ctx,
		testkit.BridgeNodeTopic,
		&testkit.BridgeNodeInfo{
			ID:          int(initCtx.GlobalSeq),
			Maddr:       addrs[0].String(),
			TrustedHash: h,
		},
	)
	if err != nil {
		return err
	}

	runenv.RecordMessage("Finished published bridgeID Addr %d", int(initCtx.GlobalSeq))

	eh, err = nd.HeaderServ.GetByHeight(ctx, uint64(8))
	if err != nil {
		return err
	}
	runenv.RecordMessage("Reached Block#8 contains Hash: %s", eh.Commit.BlockID.Hash.String())

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
