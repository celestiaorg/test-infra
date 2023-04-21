package appsync

import (
	"context"
	"fmt"
	"github.com/celestiaorg/test-infra/testkit"
	"github.com/celestiaorg/test-infra/testkit/qgbkit"
	appsync "github.com/celestiaorg/test-infra/tests/helpers/app-sync"
	"github.com/celestiaorg/test-infra/tests/helpers/common"
	common2 "github.com/ethereum/go-ethereum/common"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/testground/sdk-go/network"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
	"strings"
	"time"
)

func RunValidatorWithOrchestrator(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*4)
	defer cancel()

	qgbcmd, err := common.BuildOrchestrator(ctx, runenv, initCtx)

	err = RunValidatorWithEVMAddress(runenv, initCtx, common.ECDSAToAddress(qgbcmd.EVMPrivateKey))
	if err != nil {
		return err
	}

	err = RunOrchestrator(runenv, initCtx)
	if err != nil {
		return err
	}
	return nil
}

func RunValidatorWithEVMAddress(runenv *runtime.RunEnv, initCtx *run.InitContext, evmAddr *common2.Address) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*4)
	defer cancel()

	syncclient := initCtx.SyncClient
	netclient := network.NewClient(syncclient, runenv)

	netclient.MustWaitNetworkInitialized(ctx)

	config := appsync.CreateConfig(runenv, initCtx)

	err := netclient.ConfigureNetwork(ctx, &config)
	if err != nil {
		return err
	}

	appcmd, err := common.BuildValidatorWithEVMAddress(ctx, runenv, initCtx, evmAddr)
	if err != nil {
		return err
	}

	if initCtx.GroupSeq == 1 {
		ip, err := netclient.GetDataNetworkIP()
		if err != nil {
			return err
		}

		_, err = syncclient.Publish(ctx, testkit.CurlGenesisState, ip.To4().String())
		if err != nil {
			return err
		}

		go appcmd.StartNode("info")
	}

	err = appsync.HandleSeedPeers(ctx, runenv, appcmd, initCtx)
	if err != nil {
		return err
	}

	if initCtx.GroupSeq != 1 {
		runenv.RecordMessage("starting........")
		go appcmd.StartNode("info")
	}

	// wait for a new block to be produced
	time.Sleep(2 * time.Minute)

	_, err = syncclient.SignalAndWait(ctx, testkit.FinishState, runenv.TestInstanceCount)
	if err != nil {
		return err
	}

	err = appsync.SubmitPFBs(runenv, appcmd)
	if err != nil {
		return err
	}

	return nil
}

func RunOrchestrator(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*4)
	defer cancel()

	syncclient := initCtx.SyncClient
	netclient := network.NewClient(syncclient, runenv)

	netclient.MustWaitNetworkInitialized(ctx)

	orchcmd, err := common.BuildOrchestrator(ctx, runenv, initCtx)
	if err != nil {
		return err
	}

	if initCtx.GroupSeq <= 2 {
		ip, err := netclient.GetDataNetworkIP()
		if err != nil {
			return err
		}

		id, err := peer.IDFromPrivateKey(*orchcmd.P2PPrivateKey)
		if err != nil {
			return err
		}

		bootstrapperNode := &qgbkit.BootstrapperNode{
			P2PID: id.String(),
			IP:    ip,
		}
		_, err = syncclient.Publish(ctx, testkit.QGBBootstrapperTopic, bootstrapperNode)
		if err != nil {
			return err
		}

		runenv.RecordMessage("starting seed........")
		go orchcmd.StartOrchestrator(common.ECDSAToAddress(orchcmd.EVMPrivateKey).Hex(), common.EVMPrivateKeyPassphrase, "", "")
	}

	if initCtx.GroupSeq > 2 {
		runenv.RecordMessage("getting bootstrappers information........")
		bootstrapperCh := make(chan *qgbkit.BootstrapperNode)
		sub, err := initCtx.SyncClient.Subscribe(ctx, testkit.QGBBootstrapperTopic, bootstrapperCh)
		if err != nil {
			return err
		}

		var bootstrappers []string
		for i := 0; i < 2; i++ {
			select {
			case err := <-sub.Done():
				if err != nil {
					return err
				}
			case bootstrapper := <-bootstrapperCh:
				// to the format /ip4/127.0.0.1/tcp/30000/p2p/12D3KooWQKobCvC2jms83hGeer8iSSxcxSKa9x7RyWMTKdTKoNvH
				bootstrappers = append(bootstrappers, fmt.Sprintf(
					"/ip4/%s/tcp/30000/p2p/%s",
					bootstrapper.IP.To4().String(),
					bootstrapper.P2PID,
				))
			}
		}

		runenv.RecordMessage("starting........")
		go orchcmd.StartOrchestrator(common.ECDSAToAddress(orchcmd.EVMPrivateKey).Hex(), common.EVMPrivateKeyPassphrase, "", strings.Join(bootstrappers, ","))
	}

	// wait for the orchestrator to start
	time.Sleep(2 * time.Minute)

	_, err = syncclient.SignalAndWait(ctx, testkit.FinishState, runenv.TestInstanceCount)
	if err != nil {
		return err
	}

	return nil
}
