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
	"net"
	"strings"
	"time"
)

func RunValidatorWithOrchestrator(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	runenv.RecordMessage("running orch........")
	err := RunOrchestrator(runenv, initCtx)
	if err != nil {
		return err
	}
	return nil
}

func RunValidatorWithEVMAddress(runenv *runtime.RunEnv, initCtx *run.InitContext, evmAddr *common2.Address) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*10)
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

		go appcmd.StartNode("error")
	}

	err = appsync.HandleSeedPeers(ctx, runenv, appcmd, initCtx)
	if err != nil {
		return err
	}

	if initCtx.GroupSeq != 1 {
		runenv.RecordMessage("starting........")
		go appcmd.StartNode("error")
	}

	// wait for a new block to be produced
	time.Sleep(9 * time.Minute)

	//err = appsync.SubmitPFBs(runenv, appcmd)
	//if err != nil {
	//	return err
	//}

	return nil
}

func RunOrchestrator(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*10)
	defer cancel()

	syncclient := initCtx.SyncClient
	netclient := network.NewClient(syncclient, runenv)

	netclient.MustWaitNetworkInitialized(ctx)

	orchcmd, err := common.BuildOrchestrator(ctx, runenv, initCtx)
	if err != nil {
		return err
	}

	go RunValidatorWithEVMAddress(runenv, initCtx, common.ECDSAToAddress(orchcmd.EVMPrivateKey))

	// wait for the validator to start
	time.Sleep(2 * time.Minute)

	if initCtx.GroupSeq == 1 {
		ip, err := netclient.GetDataNetworkIP()
		if err != nil {
			return err
		}

		id, err := peer.IDFromPrivateKey(*orchcmd.P2PPrivateKey)
		if err != nil {
			return err
		}

		port := getFreePort()
		bootstrapperNode := &qgbkit.BootstrapperNode{
			P2PID: id.String(),
			IP:    ip,
			Port:  port,
		}
		_, err = syncclient.Publish(ctx, testkit.QGBBootstrapperTopic, bootstrapperNode)
		if err != nil {
			return err
		}

		runenv.RecordMessage("starting bootstrapper........")
		go orchcmd.StartOrchestrator(
			common.ECDSAToAddress(orchcmd.EVMPrivateKey).Hex(),
			common.EVMPrivateKeyPassphrase,
			"",
			"",
			fmt.Sprintf(
				"/ip4/%s/tcp/%d/p2p/%s",
				bootstrapperNode.IP.To4().String(),
				bootstrapperNode.Port,
				bootstrapperNode.P2PID,
			),
		)
	} else {
		//time.Sleep(30 * time.Second)
		runenv.RecordMessage("getting bootstrappers information........")
		bootstrapperCh := make(chan *qgbkit.BootstrapperNode)
		_, err := syncclient.Subscribe(ctx, testkit.QGBBootstrapperTopic, bootstrapperCh)
		if err != nil {
			return err
		}

		var bootstrappers []string
		for i := 0; i < 1; i++ {
			select {
			case bootstrapper := <-bootstrapperCh:
				runenv.RecordMessage("got bootstrapper")
				runenv.RecordMessage(bootstrapper.P2PID)
				runenv.RecordMessage(bootstrapper.IP.To4().String())
				// to the format /ip4/127.0.0.1/tcp/30000/p2p/12D3KooWQKobCvC2jms83hGeer8iSSxcxSKa9x7RyWMTKdTKoNvH
				bootstrappers = append(bootstrappers, fmt.Sprintf(
					"/ip4/%s/tcp/30000/p2p/%s",
					bootstrapper.IP.To4().String(),
					bootstrapper.P2PID,
				))
			}
		}

		runenv.RecordMessage("got bootstrappers........")
		runenv.RecordMessage(strings.Join(bootstrappers, ":::::::::"))
		ip, err := netclient.GetDataNetworkIP()
		if err != nil {
			return err
		}
		port := getFreePort()
		listenAddr := fmt.Sprintf("/ip4/%s/tcp/%d", ip.To4().String(), port)
		runenv.RecordMessage(listenAddr)
		runenv.RecordMessage("Wait for bootstrapper node to be up")
		time.Sleep(time.Minute)
		go orchcmd.StartOrchestrator(
			common.ECDSAToAddress(orchcmd.EVMPrivateKey).Hex(),
			common.EVMPrivateKeyPassphrase,
			"",
			strings.Join(bootstrappers, ","),
			listenAddr,
		)
	}

	_, err = syncclient.SignalAndWait(ctx, testkit.FinishState, runenv.TestInstanceCount)
	if err != nil {
		return err
	}

	return nil
}

func getFreePort() int {
	a, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err == nil {
		var l *net.TCPListener
		if l, err = net.ListenTCP("tcp", a); err == nil {
			defer l.Close()
			return l.Addr().(*net.TCPAddr).Port
		}
	}
	panic("while getting free port: " + err.Error())
}
