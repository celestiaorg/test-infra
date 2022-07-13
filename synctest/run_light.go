package synctest

import (
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
)

func RunLightNode(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
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

	// 		//we receive bridgeIDs that contain the ID of bridge and the total amount of bridges
	// 		//we need to assign light nodes 30/30/30 per each bridge
	// 		if int(initCtx.GroupSeq)%bridge.Amount == bridge.ID%bridge.Amount {
	// 			ndhome := fmt.Sprintf("/.celestia-light-%d", int(initCtx.GroupSeq))
	// 			runenv.RecordMessage(ndhome)
	// 			ip, err := initCtx.NetClient.GetDataNetworkIP()
	// 			if err != nil {
	// 				return err
	// 			}

	// 			nd, err := nodekit.NewNode(ndhome, node.Light, ip, node.WithTrustedHash(bridge.TrustedHash), node.WithTrustedPeers(bridge.Maddr))
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
	// 			runenv.RecordSuccess()

	// 			err = nd.Stop(ctx)
	// 			if err != nil {
	// 				return err
	// 			}

	// 			return nil
	// 		}
	// 	}
	// }
}
