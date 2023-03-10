package reconstruction

import (
	"context"
	"fmt"
	"time"

	"github.com/celestiaorg/test-infra/testkit"
	"github.com/celestiaorg/test-infra/testkit/nodekit"
	"github.com/celestiaorg/test-infra/tests/helpers/common"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/testground/sdk-go/network"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
	"github.com/testground/sdk-go/sync"
)

func RunBridgeNode(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
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

	nd, err := common.BuildBridge(ctx, runenv, initCtx)
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

	if nodekit.IsSyncing(ctx, nd) {
		runenv.RecordFailure(fmt.Errorf("bridge node is still syncing the past"))
	}

	fullNode, err := func(ctx context.Context, syncclient sync.Client, total int) (fulls []*testkit.FullNodeInfo, err error) {
		fullCh := make(chan *testkit.FullNodeInfo, total)
		sub, err := syncclient.Subscribe(ctx, testkit.FullNodeTopic, fullCh)
		if err != nil {
			return nil, err
		}

		for i := 0; i < total; i++ {
			select {
			case err = <-sub.Done():
				if err != nil {
					return nil,
						fmt.Errorf("no full address has been sent to this bridge node to connect to")
				}
			case full := <-fullCh:
				fulls = append(fulls, full)
			}
		}
		return fulls, nil
	}(ctx, syncclient, runenv.IntParam("full"))
	if err != nil {
		return err
	}

	runenv.RecordMessage("Blacklisting bridge IPs for BridgeNode %d", int(initCtx.GroupSeq))

	for _, v := range fullNode {
		id, _ := peer.AddrInfoFromString(v.Maddr)
		nd.Host.Network().ClosePeer(id.ID)
		nd.ConnGater.BlockPeer(id.ID)
		if !nd.ConnGater.InterceptPeerDial(id.ID) {
			runenv.RecordMessage("blocked maddr %s", v.Maddr)
		}
	}

	eh, err = nd.HeaderServ.GetByHeight(ctx, uint64(runenv.IntParam("submit-times")-1))
	if err != nil {
		return err
	}
	runenv.RecordMessage("Reached Block#%d contains Hash: %s",
		runenv.IntParam("submit-times")-1,
		eh.Commit.BlockID.Hash.String())

	if nodekit.IsSyncing(ctx, nd) {
		runenv.RecordFailure(fmt.Errorf("full node is still syncing the past"))
	}

	err = nd.Stop(ctx)
	if err != nil {
		return err
	}

	_, err = syncclient.SignalEntry(ctx, testkit.FinishState)
	if err != nil {
		return err
	}

	return nil
}
