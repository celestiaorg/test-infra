package dasbenchmarks

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/celestiaorg/celestia-node/nodebuilder"
	"github.com/celestiaorg/celestia-node/nodebuilder/node"
	"github.com/celestiaorg/test-infra/testkit"
	"github.com/celestiaorg/test-infra/testkit/nodekit"
	"github.com/celestiaorg/test-infra/tests/helpers/common"
	"github.com/testground/sdk-go/network"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
)

func RunLightNode(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Minute*time.Duration(runenv.IntParam("execution-time")),
	)
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

	bridgeCh := make(chan *testkit.BridgeNodeInfo)
	_, err = syncclient.Subscribe(ctx, testkit.BridgeNodeTopic, bridgeCh)
	if err != nil {
		return err
	}

	bridgeNode := <-bridgeCh
	if bridgeNode == nil {
		bridgeNode = <-bridgeCh
	}

	ndhome := fmt.Sprintf("/.celestia-light-%d", initCtx.GlobalSeq)
	runenv.RecordMessage(ndhome)

	ip, err := initCtx.NetClient.GetDataNetworkIP()
	if err != nil {
		return err
	}

	trustedPeers := []string{bridgeNode.Maddr}
	cfg := nodekit.NewConfig(node.Light, ip, trustedPeers, bridgeNode.TrustedHash)
	nd, err := nodekit.NewNode(
		ndhome,
		node.Light,
		cfg,
		nodebuilder.WithMetrics(
			[]otlpmetrichttp.Option{
				otlpmetrichttp.WithEndpoint("192.18.0.8:4318"),
				otlpmetrichttp.WithInsecure(),
			},
			nodekit.LightNodeType,
		),
	)

	if err != nil {
		return err
	}

	err = nd.Start(ctx)
	if err != nil {
		return err
	}

	runenv.RecordMessage("Starting DAS")

	_, err = nd.HeaderServ.GetByHeight(ctx, uint64(5))
	prevHead := []byte(bridgeNode.TrustedHash)

	for i := 0; i < runenv.IntParam("block-height"); i++ {
		header, err := nd.HeaderServ.Head(ctx)
		if err != nil {
			return err
		}

		if bytes.Compare(header.Commit.BlockID.Hash, prevHead) == 0 {
			runenv.RecordMessage("Failed to stay on top of the chain")
			return fmt.Errorf("xxx")
		}

		prevHead = header.Commit.BlockID.Hash

		if nd.HeaderServ.IsSyncing() {
			runenv.RecordMessage("Failed to stay on top of the chain")
			return fmt.Errorf("xxx")
		}

		st, err := nd.DASer.SamplingStats(ctx)
		if err != nil {
			return fmt.Errorf("xxx")
		}

		if st.CatchUpDone &&
			st.CatchupHead == uint64(header.RawHeader.Height) &&
			st.SampledChainHead == uint64(header.RawHeader.Height) {
			return fmt.Errorf("xxx")
		}

		nd.HeaderServ.GetByHeight(ctx, uint64(header.RawHeader.Height+1))
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
