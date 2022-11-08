package syncpast

import (
	"context"
	"fmt"
	"time"

	"github.com/celestiaorg/celestia-node/das"
	"github.com/celestiaorg/celestia-node/nodebuilder/node"
	"github.com/celestiaorg/test-infra/testkit"
	"github.com/celestiaorg/test-infra/testkit/nodekit"
	"github.com/celestiaorg/test-infra/tests/helpers/common"
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

	bridgeNode, err := common.GetBridgeNode(ctx, syncclient, initCtx.GroupSeq, runenv.IntParam("bridge"))
	if err != nil {
		return err
	}

	ndhome := fmt.Sprintf("/.celestia-light-%d", initCtx.GlobalSeq)
	runenv.RecordMessage(ndhome)

	ip, err := initCtx.NetClient.GetDataNetworkIP()
	if err != nil {
		return err
	}

	// We wait until the bridge reaches a certain height and then start syncing the chain
	b, err := syncclient.Barrier(ctx, testkit.PastBlocksGeneratedState, runenv.IntParam("bridge"))
	berr := <-b.C
	if err != nil || berr != nil {
		return fmt.Errorf("error occured on barriering: err - %s, barrier err - %s", err, berr)
	}

	trustedPeers := []string{bridgeNode.Maddr}
	cfg := nodekit.NewConfig(node.Light, ip, trustedPeers, bridgeNode.TrustedHash)
	nd, err := nodekit.NewNode(
		ndhome,
		node.Light,
		cfg,
	)
	if err != nil {
		return err
	}

	err = nd.Start(ctx)
	if err != nil {
		return err
	}

	eh, err := nd.HeaderServ.GetByHeight(ctx, uint64(runenv.IntParam("block-height")))
	if err != nil {
		return err
	}
	runenv.RecordMessage("Reached Block#%d contains Hash: %s",
		runenv.IntParam("block-height"),
		eh.Commit.BlockID.Hash.String())

	if nd.HeaderServ.IsSyncing() {
		runenv.RecordFailure(fmt.Errorf("light node is still syncing the past"))
	}

	bh := uint64(runenv.IntParam("block-height"))

	if !checkDaserStatus(ctx, nd.DASer, bh) {
		return fmt.Errorf("light node is still dasing past headers")
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

func checkDaserStatus(ctx context.Context, daser *das.DASer, bh uint64) bool {
	timeout := time.After(2 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)

	for {
		st, err := daser.SamplingStats(ctx)
		if err != nil {
			return false
		}
		select {
		case <-timeout:
			return false
		case <-ticker.C:
			if st.CatchUpDone && st.CatchupHead >= bh && st.SampledChainHead >= bh {
				return true
			}
		}
	}
}
