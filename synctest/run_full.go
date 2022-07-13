package synctest

import (
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
)

func RunFullNode(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	return nil
	// ctx, cancel := context.WithTimeout(context.Background(), time.Minute*2)
	// defer cancel()

	// client := initCtx.SyncClient
	// os.Setenv("GOLOG_OUTPUT", "stdout")
	// level, err := logging.LevelFromString("INFO")
	// if err != nil {
	// 	return err
	// }
	// logs.SetAllLoggers(level)

	// client := initCtx.SyncClient
	// netclient := network.NewClient(client, runenv)

	// netclient.MustWaitNetworkInitialized(ctx)

	// config := network.Config{
	// 	Network: "default",
	// 	Enable:  true,
	// 	Default: network.LinkShape{
	// 		Latency:   100 * time.Millisecond,
	// 		Bandwidth: 1 << 20, // 1Mib
	// 	},
	// 	CallbackState: "network-configured",
	// 	RoutingPolicy: network.AllowAll,
	// }

	// config.IPv4 = runenv.TestSubnet

	// // using the assigned `GlobalSequencer` id per each of instance
	// // to fill in the last 2 octects of the new IP address for the instance
	// ipC := byte((initCtx.GlobalSeq >> 8) + 1)
	// ipD := byte(initCtx.GlobalSeq)
	// config.IPv4.IP = append(config.IPv4.IP[0:2:2], ipC, ipD)

	// err = netclient.ConfigureNetwork(ctx, &config)
	// if err != nil {
	// 	return err
	// }

	// err = <-client.MustBarrier(ctx, testkit.AppStartedState, int(initCtx.GroupSeq)).C
	// if err != nil {
	// 	return err
	// }

	// bridgeCh := make(chan *BridgeId)
	// sub, err := client.Subscribe(ctx, BridgeNodeTopic, bridgeCh)
	// if err != nil {
	// 	return err
	// }

	// for {
	// 	select {
	// 	case <-sub.Done():
	// 		return fmt.Errorf("nodeId hasn't received")
	// 	case bridge := <-bridgeCh:
	// 		if int(initCtx.GroupSeq) == bridge.ID {
	// 			ndhome := fmt.Sprintf("/.celestia-full-%d", initCtx.GroupSeq)
	// 			runenv.RecordMessage(ndhome)
	// 			ip, err := initCtx.NetClient.GetDataNetworkIP()
	// 			if err != nil {
	// 				return err
	// 			}
	// 			nd, err := nodekit.NewNode(ndhome, node.Full, ip, node.WithTrustedHash(bridge.TrustedHash), node.WithTrustedPeers(bridge.Maddr))
	// 			if err != nil {
	// 				return err
	// 			}

	// 			err = nd.Start(ctx)
	// 			if err != nil {
	// 				return err
	// 			}

	// 			eh, err := nd.HeaderServ.GetByHeight(ctx, uint64(9))
	// 			if err != nil {
	// 				return err
	// 			}
	// 			runenv.RecordMessage("Reached Block#9 contains Hash: %s", eh.Commit.BlockID.Hash.String())

	// 			err = nd.Stop(ctx)
	// 			if err != nil {
	// 				return err
	// 			}
	// 			return nil
	// 		}
	// 	}
	// }
}
