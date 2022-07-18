package synctest

import (
	"context"
	"fmt"
	"os"
	"time"

	logging "github.com/ipfs/go-log/v2"

	"github.com/celestiaorg/celestia-node/logs"
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*2)
	defer cancel()

	os.Setenv("GOLOG_OUTPUT", "stdout")
	level, err := logging.LevelFromString("INFO")
	if err != nil {
		return err
	}
	logs.SetAllLoggers(level)

	client := initCtx.SyncClient
	netclient := network.NewClient(client, runenv)

	netclient.MustWaitNetworkInitialized(ctx)

	config := network.Config{
		Network: "default",
		Enable:  true,
		Default: network.LinkShape{
			Latency:   100 * time.Millisecond,
			Bandwidth: 1 << 20, // 1Mib
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

	if initCtx.GroupSeq == 1 {
		_, err = client.Publish(ctx, testkit.BridgeTotalTopic, runenv.TestGroupInstanceCount)
		if err != nil {
			return err
		}
	}

	err = <-client.MustBarrier(ctx, testkit.AppStartedState, int(initCtx.GroupSeq)).C
	if err != nil {
		return err
	}

	appInfoCh := make(chan *testkit.AppNodeInfo)
	sub, err := client.Subscribe(ctx, testkit.AppNodeTopic, appInfoCh)
	if err != nil {
		return err
	}

	appNode, err := func(total int) (*testkit.AppNodeInfo, error) {
		for i := 0; i < runenv.TestGroupInstanceCount; i++ {
			select {
			case err = <-sub.Done():
				if err != nil {
					return nil, err
				}
			case appInfo := <-appInfoCh:
				if appInfo.ID == int(initCtx.GroupSeq) {
					return appInfo, nil
				}
			}
		}
		return nil, fmt.Errorf("nothing has been done for bridge node")
	}(runenv.TestGroupInstanceCount)

	if err != nil {
		return err
	}

	h, err := appkit.GetBlockHashByHeight(appNode.IP, 1)
	if err != nil {
		return err
	}
	runenv.RecordMessage("Block#1 Hash: %s", h)

	ndhome := fmt.Sprintf("/.celestia-bridge-%d", initCtx.GroupSeq)
	rc := fmt.Sprintf("%s:26657", appNode.IP.To4().String())
	runenv.RecordMessage(rc)

	ip, err := initCtx.NetClient.GetDataNetworkIP()
	if err != nil {
		return err
	}

	nd, err := nodekit.NewNode(ndhome, node.Bridge, ip, h, node.WithRemoteCore("tcp", rc))
	if err != nil {
		return err
	}

	nd.Start(ctx)
	if err != nil {
		return err
	}

	eh, err := nd.HeaderServ.GetByHeight(ctx, uint64(4))
	if err != nil {
		return err
	}

	runenv.RecordMessage("Reached Block#4 contains Hash: %s", eh.Commit.BlockID.Hash.String())

	//create a new subscription to publish bridge's multiaddress to full/light nodes
	addrs, err := peer.AddrInfoToP2pAddrs(host.InfoFromHost(nd.Host))
	if err != nil {
		return err
	}

	runenv.RecordMessage("Publishing bridgeID %d", int(initCtx.GroupSeq))
	runenv.RecordMessage("Publishing bridgeID Addr %s", addrs[0].String())

	_, err = client.SignalEntry(ctx, testkit.BridgeStartedState)
	if err != nil {
		return err
	}

	err = <-client.MustBarrier(ctx, testkit.BridgeStartedState, runenv.TestGroupInstanceCount).C
	if err != nil {
		return err
	}

	_, err = client.Publish(
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

	// testableInstances are full and light nodes. We are multiplying bridge's
	// by 2 as we have ratio on 1 app per 1 bridge node
	testableInstances := runenv.TestInstanceCount - (runenv.TestGroupInstanceCount * 2)
	err = <-client.MustBarrier(ctx, testkit.FinishState, testableInstances).C
	if err != nil {
		return err
	}

	err = nd.Stop(ctx)
	if err != nil {
		return err
	}

	return nil
}
