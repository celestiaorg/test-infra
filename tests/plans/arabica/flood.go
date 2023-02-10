package arabica

import (
	"context"
	"fmt"
	"github.com/celestiaorg/celestia-node/nodebuilder"
	"github.com/celestiaorg/celestia-node/nodebuilder/node"
	"github.com/celestiaorg/test-infra/testkit"
	"github.com/celestiaorg/test-infra/testkit/appkit"
	"github.com/celestiaorg/test-infra/testkit/nodekit"
	"github.com/celestiaorg/test-infra/tests/helpers/common"
	"github.com/testground/sdk-go/network"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
	"time"
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
	// to fill in the last 2 octets of the new IP address for the instance
	ipC := byte((initCtx.GlobalSeq >> 8) + 1)
	ipD := byte(initCtx.GlobalSeq)
	config.IPv4.IP = append(config.IPv4.IP[0:2:2], ipC, ipD)

	err = netclient.ConfigureNetwork(ctx, &config)
	if err != nil {
		return err
	}

	// first we query limani for latest block in arabica
	restEndpoint := runenv.StringParam("rest-endpoint")
	uri := fmt.Sprintf("http://%s/block", restEndpoint)
	resBlock, err := appkit.GetResponse(uri)
	if err != nil {
		return err
	}

	query := fmt.Sprintf("%s?height=%d", uri, 1)
	// if the latest height is greater than `block-height`, then we query
	// for hash of the (block-`block-height`) to set it as trusted hash later on
	if resBlock.Block.Height > int64(runenv.IntParam("block-height")) {
		height := resBlock.Block.Height - int64(runenv.IntParam("block-height"))
		query = fmt.Sprintf("%s?height=%d", uri, height)
	}

	resp, err := appkit.GetResponse(query)
	if err != nil {
		return err
	}

	trustedHash := resp.BlockID.Hash.String()

	netId := runenv.StringParam("p2p-network")
	ndHome := fmt.Sprintf("/.celestia-light-%s", netId)
	runenv.RecordMessage(ndHome)

	cfg := nodebuilder.DefaultConfig(node.Light)
	cfg.Header.TrustedHash = trustedHash
	nd, err := nodekit.NewNode(ndHome, node.Light, netId, cfg)
	if err != nil {
		return err
	}

	err = nd.Start(ctx)
	if err != nil {
		return err
	}

	eh, err := nd.HeaderServ.GetByHeight(ctx, uint64(resBlock.Block.Height))
	if err != nil {
		return err
	}
	runenv.RecordMessage("Reached Block#%d contains Hash: %s",
		runenv.IntParam("block-height"),
		eh.Commit.BlockID.Hash.String())

	if nd.HeaderServ.IsSyncing(ctx) {
		runenv.RecordFailure(fmt.Errorf("light node is still syncing the past"))
	}

	err = nd.DASer.WaitCatchUp(ctx)
	if err != nil {
		return err
	}

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
