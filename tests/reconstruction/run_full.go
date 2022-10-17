package reconstruction

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/celestiaorg/celestia-node/nodebuilder/node"
	"github.com/celestiaorg/test-infra/testkit"
	"github.com/celestiaorg/test-infra/testkit/nodekit"
	"github.com/celestiaorg/test-infra/tests/common"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/testground/sdk-go/network"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
	"github.com/testground/sdk-go/sync"
)

func RunFullNode(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Minute*time.Duration(runenv.IntParam("execution-time")),
	)
	defer cancel()

	err := nodekit.SetLoggersLevel("DEBUG")
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

	bridgeNodes, err := func(ctx context.Context, syncclient sync.Client, amountOfBridges int) (bridges []*testkit.BridgeNodeInfo, err error) {
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
						fmt.Errorf("no bridge address has been sent to this full node to connect to")
				}
			case bridge := <-bridgeCh:
				bridges = append(bridges, bridge)
			}
		}

	}(ctx, syncclient, runenv.IntParam("bridge"))
	if err != nil {
		return err
	}

	ndhome := fmt.Sprintf("/.celestia-full-%d", initCtx.GlobalSeq)
	runenv.RecordMessage(ndhome)

	ip, err := initCtx.NetClient.GetDataNetworkIP()
	if err != nil {
		return err
	}

	trustedPeers := func(bridges []*testkit.BridgeNodeInfo) []string {
		var peers []string
		for _, v := range bridges {
			peers = append(peers, v.Maddr)
		}
		return peers
	}(bridgeNodes)
	cfg := nodekit.NewConfig(node.Full, ip, trustedPeers, bridgeNodes[0].TrustedHash)
	nd, err := nodekit.NewNode(
		ndhome,
		node.Full,
		cfg,
	)
	if err != nil {
		return err
	}

	addrs, err := peer.AddrInfoToP2pAddrs(host.InfoFromHost(nd.Host))
	if err != nil {
		return err
	}

	runenv.RecordMessage("Publishing Full ID %d", int(initCtx.GroupSeq))
	runenv.RecordMessage("Publishing Full Addr %s", addrs[0].String())

	_, err = syncclient.Publish(
		ctx,
		testkit.FullNodeTopic,
		&testkit.FullNodeInfo{
			ID:    int(initCtx.GroupSeq),
			Maddr: addrs[0].String(),
		},
	)
	if err != nil {
		return err
	}

	runenv.RecordMessage("Finished publishing FullNode %d", int(initCtx.GroupSeq))

	err = nd.Start(ctx)
	if err != nil {
		return err
	}

	eh, err := nd.HeaderServ.GetByHeight(ctx, uint64(runenv.IntParam("block-height")))
	if err != nil {
		return err
	}
	runenv.RecordMessage("Reached Block#%d contains Hash: %s",
		runenv.IntParam("block-height"),
		eh.Commit.BlockID.Hash.String())

	if nd.HeaderServ.IsSyncing() {
		runenv.RecordFailure(fmt.Errorf("full node is still syncing the past"))
	}

	runenv.RecordMessage("Blacklisting bridge IPs for FullNode %d", int(initCtx.GroupSeq))

	for _, v := range bridgeNodes {
		ip := strings.Split(v.Maddr, "/")[2]
		nd.ConnGater.BlockAddr(net.ParseIP(ip))
	}

	runenv.RecordMessage("FullNode %d is trying to reconstruct the block", int(initCtx.GroupSeq))
	eh, err = nd.HeaderServ.GetByHeight(ctx, uint64(runenv.IntParam("submit-times")-1))
	if err != nil {
		return err
	}
	runenv.RecordMessage("Reached Block#%d contains Hash: %s",
		runenv.IntParam("submit-times")-1,
		eh.Commit.BlockID.Hash.String())

	if nd.HeaderServ.IsSyncing() {
		runenv.RecordFailure(fmt.Errorf("full node is still syncing the past"))
	}

	err = nd.Stop(ctx)
	if err != nil {
		return err
	}

	_, err = syncclient.SignalEntry(ctx, testkit.FinishState)
	if err != nil {
		return err
	}

	return err
}
