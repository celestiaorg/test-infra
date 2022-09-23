package common

import (
	"context"
	"fmt"

	"github.com/celestiaorg/celestia-node/core"
	coremodule "github.com/celestiaorg/celestia-node/nodebuilder/core"
	"github.com/celestiaorg/celestia-node/nodebuilder/node"
	"github.com/celestiaorg/test-infra/testkit"
	"github.com/celestiaorg/test-infra/testkit/appkit"
	"github.com/celestiaorg/test-infra/testkit/nodekit"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
	"github.com/testground/sdk-go/sync"
)

func BuildBridge(ctx context.Context, runenv *runtime.RunEnv, initCtx *run.InitContext) (*node.Node, error) {
	syncclient := initCtx.SyncClient

	appInfoCh := make(chan *testkit.AppNodeInfo, runenv.IntParam("validator"))
	sub, err := syncclient.Subscribe(ctx, testkit.AppNodeTopic, appInfoCh)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	h, err := appkit.GetBlockHashByHeight(appNode.IP, 1)
	if err != nil {
		return nil, err
	}
	runenv.RecordMessage("Block#1 Hash: %s", h)

	ndhome := fmt.Sprintf("/.celestia-bridge-%d", initCtx.GlobalSeq)
	runenv.RecordMessage(appNode.IP.To4().String())

	ip, err := initCtx.NetClient.GetDataNetworkIP()
	if err != nil {
		return nil, err
	}

	coreclient, err := core.NewRemote(appNode.IP.To4().String(), "26657")
	if err != nil {
		return nil, err
	}

	nd, err := nodekit.NewNode(ndhome, node.Bridge, ip, h,
		coremodule.WithClient(coreclient),
	)
	if err != nil {
		return nil, err
	}

	err = nd.Start(ctx)
	if err != nil {
		return nil, err
	}

	eh, err := nd.HeaderServ.GetByHeight(ctx, uint64(2))
	if err != nil {
		return nil, err
	}

	runenv.RecordMessage("Reached Block#2 contains Hash: %s", eh.Commit.BlockID.Hash.String())

	//create a new subscription to publish bridge's multiaddress to full/light nodes
	addrs, err := peer.AddrInfoToP2pAddrs(host.InfoFromHost(nd.Host))
	if err != nil {
		return nil, err
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
		return nil, err
	}

	runenv.RecordMessage("Finished published bridgeID Addr %d", int(initCtx.GroupSeq))

	return nd, nil
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
