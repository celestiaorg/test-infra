package robusta

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/celestiaorg/celestia-app/app"
	"github.com/celestiaorg/celestia-app/app/encoding"
	cmdnode "github.com/celestiaorg/celestia-node/cmd"
	"github.com/celestiaorg/celestia-node/nodebuilder"
	"github.com/celestiaorg/celestia-node/nodebuilder/node"
	"github.com/celestiaorg/test-infra/testkit"
	"github.com/celestiaorg/test-infra/testkit/nodekit"
	"github.com/celestiaorg/test-infra/tests/helpers/common"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/spf13/cobra"
	"github.com/testground/sdk-go/network"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
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
	// to fill in the last 2 octets of the new IP address for the instance
	ipC := byte((initCtx.GlobalSeq >> 8) + 1)
	ipD := byte(initCtx.GlobalSeq)
	config.IPv4.IP = append(config.IPv4.IP[0:2:2], ipC, ipD)

	err = netclient.ConfigureNetwork(ctx, &config)
	if err != nil {
		return err
	}

	//netId := runenv.StringParam("p2p-network")

	//ndHome := fmt.Sprintf("/.celestia-light")
	//runenv.RecordMessage(ndHome)

	////os.Setenv("CELESTIA_CUSTOM", "robusta-nightly-1:97273F7F7DEA75CABCF1A1BE074E0952815B63880AB905BE0A3DEF016CFED271")

	//cmdCtx := cmdnode.WithNetwork(ctx, "robusta-nightly-1")
	////cfg := nodebuilder.DefaultConfig(node.Light)
	//cmdCtx = cmdnode.WithNodeType(cmdCtx, node.Light)
	//cfg := cmdnode.NodeConfig(cmdCtx)
	//cfg.Header.TrustedPeers = []string{"/dns/51.159.11.217/tcp/2121/p2p/12D3KooWLD5aFJo3R7HxQYDfu1ssipuQcc8W1xgWk5muwnq9DFbn"}

	//optlOpts := []otlpmetrichttp.Option{
	//	otlpmetrichttp.WithEndpoint(runenv.StringParam("otel-collector-address")),
	//	otlpmetrichttp.WithInsecure(),
	//}

	//nd, err := nodekit.NewNode(ndHome, node.Light, "robusta-nightly-1", &cfg,
	//	nodebuilder.WithMetrics(
	//		optlOpts,
	//		node.Light,
	//	))

	ndHome := fmt.Sprintf("/.celestia-light")

	cbr := cobra.Command{}
	cmdCtx := cbr.Context()
	cmdCtx = cmdnode.WithNetwork(cmdCtx, "robusta-nightly-1")
	cmdCtx = cmdnode.WithNodeType(cmdCtx, node.Light)
	cfg := cmdnode.NodeConfig(cmdCtx)
	cfg.Header.TrustedPeers = []string{"/dns/51.159.11.217/tcp/2121/p2p/12D3KooWLD5aFJo3R7HxQYDfu1ssipuQcc8W1xgWk5muwnq9DFbn"}

	cmdCtx = cmdnode.WithStorePath(cmdCtx, ndHome)
	err = nodebuilder.Init(cfg, cmdnode.StorePath(cmdCtx), node.Light)
	if err != nil {
		return err
	}

	storePath := cmdnode.StorePath(cmdCtx)
	keysPath := filepath.Join(storePath, "keys")

	encConf := encoding.MakeConfig(app.ModuleEncodingRegisters...)
	ring, err := keyring.New(app.Name, cfg.State.KeyringBackend, keysPath, os.Stdin, encConf.Codec)
	if err != nil {
		return err
	}

	store, err := nodebuilder.OpenStore(storePath, ring)
	if err != nil {
		return err
	}

	nd, err := nodebuilder.NewWithConfig(cmdnode.NodeType(cmdCtx), cmdnode.Network(cmdCtx), store, &cfg, cmdnode.NodeOptions(cmdCtx)...)
	if err != nil {
		return err
	}

	if err != nil {
		return err
	}

	durationCount := time.Duration(initCtx.GroupSeq)
	time.Sleep(time.Second * durationCount * 5)
	err = nd.Start(ctx)
	if err != nil {
		return err
	}

	time.Sleep(time.Minute * 40)
	err = nd.Stop(ctx)
	if err != nil {
		return err
	}
	_, err = syncclient.SignalAndWait(ctx, testkit.FinishState, runenv.IntParam("light"))
	if err != nil {
		return err
	}

	return err
}
