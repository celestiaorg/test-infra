package synctest

import (
	"context"
	"fmt"
	"os"
	"time"

	logging "github.com/ipfs/go-log/v2"

	"github.com/celestiaorg/celestia-node/logs"
	"github.com/celestiaorg/celestia-node/node"
	"github.com/celestiaorg/test-infra/testkit/nodekit"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
)

func RunFullNode(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*2)
	defer cancel()

	client := initCtx.SyncClient
	os.Setenv("GOLOG_OUTPUT", "stdout")
	level, err := logging.LevelFromString("INFO")
	if err != nil {
		return err
	}
	logs.SetAllLoggers(level)

	bridgeCh := make(chan *BridgeId)
	sub, err := client.Subscribe(ctx, BridgeNodeTopic, bridgeCh)
	if err != nil {
		return err
	}

	for {
		select {
		case <-sub.Done():
			return fmt.Errorf("nodeId hasn't received")
		case bridge := <-bridgeCh:
			if int(initCtx.GroupSeq) == bridge.ID {
				ndhome := fmt.Sprintf("/.celestia-full-%d", initCtx.GroupSeq)
				runenv.RecordMessage(ndhome)
				ip, err := initCtx.NetClient.GetDataNetworkIP()
				if err != nil {
					return err
				}
				nd, err := nodekit.NewNode(ndhome, node.Full, ip, node.WithTrustedHash(bridge.TrustedHash), node.WithTrustedPeers(bridge.Maddr))
				if err != nil {
					return err
				}

				err = nd.Start(ctx)
				if err != nil {
					return err
				}

				eh, err := nd.HeaderServ.GetByHeight(ctx, uint64(9))
				if err != nil {
					return err
				}
				runenv.RecordMessage("Reached Block#9 contains Hash: %s", eh.Commit.BlockID.Hash.String())

				err = nd.Stop(ctx)
				if err != nil {
					return err
				}
				return nil
			}
		}
	}
}
