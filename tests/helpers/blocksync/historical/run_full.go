package blocksynchistorical

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

	bridgeNodes, err := common.GetBridgeNodes(ctx, syncclient, runenv.IntParam("bridge"))
	if err != nil {
		return err
	}

	var bridgeNode *testkit.BridgeNodeInfo
	for _, bridge := range bridgeNodes {
		if (int(initCtx.GroupSeq) % runenv.IntParam("bridge")) == (bridge.ID % runenv.IntParam("bridge")) {
			bridgeNode = bridge
		}
	}
	trustedPeers := []string{}
	for _, bridge := range bridgeNodes {
		trustedPeers = append(trustedPeers, bridge.Maddr)
	}

	ndhome := fmt.Sprintf("/.celestia-full-%d", initCtx.GlobalSeq)
	runenv.RecordMessage(ndhome)

	ip, err := initCtx.NetClient.GetDataNetworkIP()
	if err != nil {
		return err
	}

	cfg := nodekit.NewConfig(node.Full, ip, trustedPeers, bridgeNode.TrustedHash)

	switch runenv.StringParam("getter") {
	case "shrex":
		cfg.Share.UseShareExchange = true
	case "ipld":
		fallthrough
	default:
		cfg.Share.UseShareExchange = false
	}


	nd, err := nodekit.NewNode(
		ndhome,
		node.Full,
		cfg,
	)
	if err != nil {
		return err
	}

	runenv.RecordMessage("Waiting for historical blocks to be generated...")

	runenv.RecordMessage("Starting full node")
	err = nd.Start(ctx)
	if err != nil {
		return err
	}

	runenv.RecordMessage("Full node is syncing")

	eh, err := nd.HeaderServ.GetByHeight(ctx, uint64(runenv.IntParam("block-height")))
	if err != nil {
		return err
	}
	runenv.RecordMessage("Reached Block#%d contains Hash: %s",
		runenv.IntParam("block-height"),
		eh.Commit.BlockID.Hash.String())

	if nd.HeaderServ.IsSyncing(ctx) {
		runenv.RecordFailure(fmt.Errorf("full node is still syncing the past"))
	}

	l, err := syncclient.Barrier(ctx, testkit.FullsFinishedSyncingState, runenv.IntParam("historical-syncers-max-id"))
	if err != nil {
		runenv.RecordFailure(err)
		return err
	}
	lerr := <-l.C
	if lerr != nil {
		runenv.RecordFailure(lerr)
		return err
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
