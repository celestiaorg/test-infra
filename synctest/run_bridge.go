package synctest

import (
	"context"
	"fmt"
	"os"
	"time"

	logging "github.com/ipfs/go-log/v2"

	"github.com/celestiaorg/celestia-node/logs"
	"github.com/celestiaorg/celestia-node/node"
	"github.com/celestiaorg/test-infra/testkit/appkit"
	"github.com/celestiaorg/test-infra/testkit/nodekit"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
)

func RunBridgeNode(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*2)
	defer cancel()

	client := initCtx.SyncClient
	os.Setenv("GOLOG_OUTPUT", "stdout")

	time.Sleep(8 * time.Second)
	level, err := logging.LevelFromString("INFO")
	if err != nil {
		return err
	}
	logs.SetAllLoggers(level)
	appIPCh := make(chan *AppId)
	_, err = client.Subscribe(ctx, AppNodeTopic, appIPCh)
	if err != nil {
		return err
	}

	for i := 1; i <= runenv.TestGroupInstanceCount; i++ {
		appIP := <-appIPCh
		if appIP.ID == int(initCtx.GroupSeq) {
			h, err := appkit.GetBlockHashByHeight(appIP.IP, 1)
			if err != nil {
				return err
			}
			runenv.RecordMessage("Block#1 Hash: %s", h)

			ndhome := fmt.Sprintf("/.celestia-bridge-%d", initCtx.GroupSeq)
			rc := fmt.Sprintf("%s:26657", appIP.IP.To4().String())
			runenv.RecordMessage(rc)

			ip, err := initCtx.NetClient.GetDataNetworkIP()
			if err != nil {
				return err
			}

			nd, err := nodekit.NewNode(ndhome, node.Bridge, ip, node.WithTrustedHash(h), node.WithRemoteCore("tcp", rc))
			if err != nil {
				return err
			}

			nd.Start(ctx)
			if err != nil {
				return err
			}

			eh, err := nd.HeaderServ.GetByHeight(ctx, uint64(4))
			if err != nil {
				return err
			}

			runenv.RecordMessage("Reached Block#4 contains Hash: %s", eh.Commit.BlockID.Hash.String())

			//create a new subscription to publish bridge's multiaddress to full/light nodes
			addrs, err := peer.AddrInfoToP2pAddrs(host.InfoFromHost(nd.Host))
			if err != nil {
				return err
			}

			runenv.RecordMessage("Publishing bridgeID %d", int(initCtx.GroupSeq))
			runenv.RecordMessage("Publishing bridgeID Addr %s", addrs[0].String())

			bseq, err := client.Publish(ctx, BridgeNodeTopic, &BridgeId{int(initCtx.GroupSeq), addrs[0].String(), h, runenv.TestGroupInstanceCount})
			if err != nil {
				return err
			}

			runenv.RecordMessage("%d published bridge id", int(bseq))

			err = nd.Stop(ctx)
			if err != nil {
				return err
			}
			return nil
		}
	}
	return fmt.Errorf("nothing has been done")
}
