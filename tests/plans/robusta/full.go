package robusta

import (
	"context"
	"fmt"
	"time"

	"github.com/celestiaorg/celestia-node/nodebuilder/node"
	"github.com/celestiaorg/test-infra/testkit"
	"github.com/celestiaorg/test-infra/testkit/nodekit"
	"github.com/celestiaorg/test-infra/tests/helpers/common"
	"github.com/testground/sdk-go/network"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
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
	// to fill in the last 2 octets of the new IP address for the instance
	ipC := byte((initCtx.GlobalSeq >> 8) + 1)
	ipD := byte(initCtx.GlobalSeq)
	config.IPv4.IP = append(config.IPv4.IP[0:2:2], ipC, ipD)

	err = netclient.ConfigureNetwork(ctx, &config)
	if err != nil {
		return err
	}

	netId := runenv.StringParam("p2p-network")
	ndHome := fmt.Sprintf("/.celestia-full")
	runenv.RecordMessage(ndHome)

	// get the ip address
	ip, err := initCtx.NetClient.GetDataNetworkIP()
	if err != nil {
		return err
	}

	// generate the new config for the Full node
	cfg := nodekit.NewConfig(
		node.Full,
		ip,
		[]string{"/dns/51.159.11.217/tcp/2121/p2p/12D3KooWLD5aFJo3R7HxQYDfu1ssipuQcc8W1xgWk5muwnq9DFbn"},
		"97273F7F7DEA75CABCF1A1BE074E0952815B63880AB905BE0A3DEF016CFED271",
	)

	nd, err := nodekit.NewNode(ndHome, node.Full, netId, cfg)
	if err != nil {
		return err
	}

	err = nd.Start(ctx)
	if err != nil {
		return err
	}

	time.Sleep(time.Minute * 15)
	err = nd.Stop(ctx)
	if err != nil {
		return err
	}
	_, err = syncclient.SignalAndWait(ctx, testkit.FinishState, runenv.IntParam("full"))
	if err != nil {
		return err
	}

	return err
}
