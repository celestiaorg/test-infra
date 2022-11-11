package common

import (
	"context"
	"fmt"

	"github.com/celestiaorg/celestia-node/nodebuilder"
	"github.com/celestiaorg/celestia-node/nodebuilder/node"
	"github.com/celestiaorg/test-infra/testkit"
	"github.com/celestiaorg/test-infra/testkit/appkit"
	"github.com/celestiaorg/test-infra/testkit/nodekit"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
	"github.com/testground/sdk-go/sync"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
)

func BuildBridge(ctx context.Context, runenv *runtime.RunEnv, initCtx *run.InitContext) (*nodebuilder.Node, error) {
	syncclient := initCtx.SyncClient

	appNode, err := GetValidatorInfo(ctx, syncclient, runenv.IntParam("validator"), int(initCtx.GroupSeq))
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

	cfg := nodekit.NewConfig(node.Bridge, ip, []string{}, h)
	cfg.Core.IP = appNode.IP.To4().String()
	cfg.Core.RPCPort = "26657"
	cfg.Core.GRPCPort = "9090"
	cfg.Gateway.Enabled = true
	cfg.Gateway.Port = "26659"

	nd, err := nodekit.NewNode(
		ndhome,
		node.Bridge,
		cfg,
		nodebuilder.WithMetrics(
			[]otlpmetrichttp.Option{
				otlpmetrichttp.WithEndpoint("192.18.0.8:4318"),
				otlpmetrichttp.WithInsecure(),
			},
			nodekit.BridgeNodeType,
		),
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
