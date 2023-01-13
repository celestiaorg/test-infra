package dasbenchmarks

import (
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
				otlpmetrichttp.WithEndpoint(runenv.StringParam("otel-collector-address")),
				otlpmetrichttp.WithInsecure(),
			},
			nodekit.LightNodeType,
		),
		nodebuilder.WithBlackboxMetrics(),
	)

	if err != nil {
		return err
	}

	err = nd.Start(ctx)
	if err != nil {
		return err
	}

	runenv.R().Counter("light-nodes-counter").Inc(1)

	// wait for the core network to reach height 1
	_, err = nd.HeaderServ.GetByHeight(ctx, uint64(1))
	if err != nil {
		runenv.RecordFailure(fmt.Errorf("Error: failed to wait for core network, %s", err))
		return err
	}

	// wait for the daser to catch up with the current height of the core network
	err = nd.DASer.WaitCatchUp(ctx)
	if err != nil {
		runenv.RecordFailure(fmt.Errorf("Error while waiting for the DASer to catch up, %s", err))
		return err
	}

	_, err = syncclient.SignalEntry(ctx, testkit.LightNodesStartedState)
	if err != nil {
		return err
	}

	// let the chain move!
	for i := 1; i < runenv.IntParam("block-height"); i++ {
		start := time.Now()
		nd.DASer.WaitCatchUp(ctx) // wait for the daser to catch up on every height
		stats, err := nd.DASer.SamplingStats(ctx)
		if err != nil {
			runenv.RecordFailure(err)
		} else {
			runenv.R().Gauge("daser-head").Update(
				float64(stats.SampledChainHead),
			)
			runenv.R().RecordPoint(fmt.Sprintf("das.time_to_catch_up,height=%v", stats.SampledChainHead), float64(time.Since(start).Milliseconds()))
		}

		// wait for the core network to reach height 1ß
		myHdr, err := nd.HeaderServ.GetByHeight(ctx, uint64(i+1))
		if err != nil {
			runenv.RecordFailure(fmt.Errorf("Error: failed to sync headers, %s", err))
			return err
		}

		coreNetHdr, err := nd.HeaderServ.SyncerHead(ctx) // for metrics
		if err != nil {
			runenv.RecordFailure(fmt.Errorf("Error: failed to sync headers, %s", err))
			return err
		}

		if coreNetHdr.RawHeader.Height != myHdr.RawHeader.Height {
			runenv.RecordMessage(
				"Light node lagging behind core network!",
				"core network height=", coreNetHdr.RawHeader.Height,
				"light node height=", myHdr.RawHeader.Height,
			)
		}
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
