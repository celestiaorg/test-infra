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
	optlOpts := []otlpmetrichttp.Option{
		otlpmetrichttp.WithEndpoint(runenv.StringParam("otel-collector-address")),
		otlpmetrichttp.WithInsecure(),
	}
	nd, err := nodekit.NewNode(
		ndhome,
		node.Light,
		cfg,
		nodebuilder.WithMetrics(
			optlOpts,
			nodekit.LightNodeType,
		),
		nodebuilder.WithBlackboxMetrics(),
	)

	if err != nil {
		return err
	}

	runenv.RecordMessage("Starting light node")
	err = nd.Start(ctx)
	if err != nil {
		runenv.RecordFailure(err)
		return err
	}

	// wait for the bridge node
	l, err := syncclient.Barrier(ctx, testkit.BridgeStartedState, runenv.IntParam("bridge"))
	if err != nil {
		runenv.RecordFailure(err)
	}
	lerr := <-l.C
	if lerr != nil {
		runenv.RecordFailure(lerr)
	}

	// signal startup
	_, err = syncclient.SignalEntry(ctx, testkit.LightNodesStartedState)
	if err != nil {
		return err
	}

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

	for i := 1; i < runenv.IntParam("block-height"); i++ {
		nd.DASer.WaitCatchUp(ctx) // wait for the daser to catch up on every height
		// wait for the core network to reach height 1
		_, err := nd.HeaderServ.GetByHeight(ctx, uint64(i+1))
		if err != nil {
			runenv.RecordFailure(fmt.Errorf("Error: failed to sync headers, %s", err))
			return err
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
