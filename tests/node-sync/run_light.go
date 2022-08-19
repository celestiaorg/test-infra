package nodesync

import (
	"context"
	"fmt"
	"time"

	"github.com/celestiaorg/celestia-node/node"
	"github.com/celestiaorg/test-infra/testkit"
	"github.com/celestiaorg/test-infra/testkit/nodekit"
	"github.com/testground/sdk-go/network"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
)

func RunLightNode(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*2)
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
			// Latency:   100 * time.Millisecond,
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

	// bridgeTotalCh := make(chan int)
	// sub, err := syncclient.Subscribe(ctx, testkit.BridgeTotalTopic, bridgeTotalCh)
	// if err != nil {
	// 	return err
	// }

	// var bridgeTotal int
	// select {
	// case err = <-sub.Done():
	// 	if err != nil {
	// 		return err
	// 	}
	// case bridgeTotal = <-bridgeTotalCh:
	// 	err = <-syncclient.MustBarrier(ctx, testkit.BridgeStartedState, bridgeTotal).C
	// 	if err != nil {
	// 		return err
	// 	}
	// }

	bridgeCh := make(chan *testkit.BridgeNodeInfo)
	sub, err := syncclient.Subscribe(ctx, testkit.BridgeNodeTopic, bridgeCh)
	if err != nil {
		return err
	}

	bridgeNode, err := func(total int) (*testkit.BridgeNodeInfo, error) {
		for i := 0; i < total; i++ {
			select {
			case err = <-sub.Done():
				if err != nil {
					return nil, err
				}
			case bridge := <-bridgeCh:
				//we receive bridgeIDs that contain the ID of bridge and the total amount of bridges
				//we need to assign light nodes 30/30/30 per each bridge
				id := int(initCtx.GlobalSeq) - runenv.IntParam("light") - runenv.IntParam("full")
				if id%total == bridge.ID%total {
					return bridge, nil
				}
			}
		}
		return nil,
			fmt.Errorf("no bridge address has been sent to this light node to connect to")
	}(runenv.IntParam("bridge"))

	if err != nil {
		return err
	}

	ndhome := fmt.Sprintf("/.celestia-light-%d", int(initCtx.GlobalSeq))
	runenv.RecordMessage(ndhome)
	ip, err := initCtx.NetClient.GetDataNetworkIP()
	if err != nil {
		return err
	}

	nd, err := nodekit.NewNode(
		ndhome,
		node.Light,
		ip,
		bridgeNode.TrustedHash,
		node.WithTrustedPeers(bridgeNode.Maddr),
	)
	if err != nil {
		return err
	}

	err = nd.Start(ctx)
	if err != nil {
		return err
	}

	eh, err := nd.HeaderServ.GetByHeight(ctx, uint64(9))
	if err != nil {
		return err
	}

	runenv.RecordMessage("Reached Block#9 contains Hash: %s", eh.Commit.BlockID.Hash.String())
	runenv.RecordSuccess()

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
