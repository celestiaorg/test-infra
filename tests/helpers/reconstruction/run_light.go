package reconstruction

import (
	"context"
	"fmt"
	"github.com/celestiaorg/celestia-node/nodebuilder"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"time"

	"github.com/celestiaorg/celestia-node/nodebuilder/node"
	"github.com/celestiaorg/test-infra/testkit"
	"github.com/celestiaorg/test-infra/testkit/nodekit"
	"github.com/celestiaorg/test-infra/tests/helpers/common"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/testground/sdk-go/network"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
	"github.com/testground/sdk-go/sync"
)

func RunLightNode(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
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

	bridgeNode, err := common.GetBridgeNode(ctx, syncclient, initCtx.GroupSeq, runenv.IntParam("bridge"))
	if err != nil {
		return err
	}

	fullNodes, err := func(ctx context.Context, syncclient sync.Client, total int) (fulls []*testkit.FullNodeInfo, err error) {
		fullCh := make(chan *testkit.FullNodeInfo, total)
		sub, err := syncclient.Subscribe(ctx, testkit.FullNodeTopic, fullCh)
		if err != nil {
			return nil, err
		}

		for i := 0; i < total; i++ {
			select {
			case err = <-sub.Done():
				if err != nil {
					return nil,
						fmt.Errorf("no full address has been sent to this light node to connect to")
				}
			case full := <-fullCh:
				fulls = append(fulls, full)
			}
		}
		return fulls, nil
	}(ctx, syncclient, runenv.IntParam("full"))
	if err != nil {
		return err
	}

	trustedPeers, bannedPeersIds := getPeers(int(initCtx.GroupSeq), runenv.TestGroupInstanceCount, fullNodes)
	trustedPeers = append(trustedPeers, bridgeNode.Maddr)

	ndhome := fmt.Sprintf("/.celestia-light-%d", int(initCtx.GlobalSeq))
	runenv.RecordMessage(ndhome)
	ip, err := initCtx.NetClient.GetDataNetworkIP()
	if err != nil {
		return err
	}

	cfg := nodekit.NewConfig(node.Light, ip, trustedPeers, bridgeNode.TrustedHash)
	optlOpts := []otlpmetrichttp.Option{
		otlpmetrichttp.WithEndpoint(runenv.StringParam("otel-collector-address")),
		otlpmetrichttp.WithInsecure(),
	}
	nd, err := nodekit.NewNode(ndhome, node.Light, runenv.StringParam("p2p-network"), cfg,
		nodebuilder.WithMetrics(
			optlOpts,
			node.Light,
			node.BuildInfo{},
		))
	if err != nil {
		return err
	}

	for _, v := range bannedPeersIds {
		nd.ConnGater.BlockPeer(v)
	}

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

	if nodekit.IsSyncing(ctx, nd) {
		runenv.RecordFailure(fmt.Errorf("light node is still syncing the past"))
	}

	runenv.RecordMessage("Light Node %d is serving shares back to the Full Node", int(initCtx.GroupSeq))
	eh, err = nd.HeaderServ.GetByHeight(ctx, uint64(runenv.IntParam("submit-times")-1))
	if err != nil {
		return err
	}
	runenv.RecordMessage("Reached Block#%d contains Hash: %s",
		runenv.IntParam("submit-times")-2,
		eh.Commit.BlockID.Hash.String())

	if nodekit.IsSyncing(ctx, nd) {
		runenv.RecordFailure(fmt.Errorf("light node is still syncing the past"))
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

func getPeers(groupId int, total int, fullNodes []*testkit.FullNodeInfo) (tp []string, bp []peer.ID) {
	split := int(total / len(fullNodes))
	for _, v := range fullNodes {
		if groupId <= split*v.ID {
			tp = append(tp, v.Maddr)
		} else if groupId > split*len(fullNodes) {
			tp = append(tp, v.Maddr)
		} else {
			id, _ := peer.AddrInfoFromString(v.Maddr)
			bp = append(bp, id.ID)
		}
	}
	return tp, bp
}
