package blocksynclatestnetpartition

import (
	"context"
	"fmt"
	"time"

	"github.com/celestiaorg/test-infra/testkit"
	"github.com/celestiaorg/test-infra/testkit/nodekit"
	"github.com/celestiaorg/test-infra/tests/helpers/common"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/testground/sdk-go/network"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
)

func RunBridgeNode(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Minute*time.Duration(runenv.IntParam("execution-time")),
	)
	defer cancel()

	err := nodekit.SetLoggersLevel("INFO")
	if err != nil {
		runenv.RecordFailure(err)
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
		runenv.RecordFailure(err)
		return err
	}

	nd, err := common.BuildBridge(ctx, runenv, initCtx)
	if err != nil {
		runenv.RecordFailure(err)
		return err
	}

	_, err = syncclient.SignalEntry(ctx, testkit.BridgeStartedState)
	if err != nil {
		runenv.RecordFailure(err)
		return err
	}

	l, err := syncclient.Barrier(ctx, testkit.ValidatorReadyTopic, runenv.IntParam("validator"))
	if err != nil {
		runenv.RecordFailure(err)
		return err
	}
	lerr := <-l.C
	if lerr != nil {
		runenv.RecordFailure(lerr)
		return lerr
	}

	runenv.RecordMessage("Connecting to other bridge nodes")
	if runenv.IntParam("interconnect-bridges") == 1 {
		bridgeNodes, _ := common.GetBridgeNodes(ctx, syncclient, runenv.IntParam("bridge"))
		for _, bridge := range bridgeNodes {
			if bridge.AddrInfo.ID != host.InfoFromHost(nd.Host).ID {
				nd.Host.Connect(ctx, bridge.AddrInfo)
				runenv.RecordMessage("Connected to Bridge:", bridge.AddrInfo.Addrs)
			}
		}
	}

	for i := 0; i < runenv.IntParam("block-height"); i++ {
		// After reaching a dedicated block-height, we can signal other node types
		// to start syncing the past
		eh, err := nd.HeaderServ.GetByHeight(ctx, uint64(i+1))
		if err != nil {
			runenv.RecordFailure(err)
			return err
		}
		runenv.RecordMessage(
			"Reached Block#%d contains Hash: %s",
			runenv.IntParam("block-height"),
			eh.Commit.BlockID.Hash.String(),
		)
	}

	if nd.HeaderServ.IsSyncing(ctx) {
		runenv.RecordFailure(fmt.Errorf("Bridge node is still syncing the past"))
	}

	err = nd.Stop(ctx)
	if err != nil {
		runenv.RecordFailure(err)
		return err
	}

	_, err = syncclient.SignalEntry(ctx, testkit.FinishState)
	if err != nil {
		runenv.RecordFailure(err)
		return err
	}

	return nil
}
