package appsync

import (
	"context"
	"fmt"
	"github.com/celestiaorg/test-infra/testkit"
	"github.com/celestiaorg/test-infra/testkit/appkit"
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
	runenv.RecordMessage("running orch........")
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*40)
	defer cancel()

	syncclient := initCtx.SyncClient
	netclient := network.NewClient(syncclient, runenv)

	netclient.MustWaitNetworkInitialized(ctx)

	orchcmd, err := common.BuildOrchestrator(ctx, runenv, initCtx)
	if err != nil {
		return err
	}

	go RunValidatorWithEVMAddress(runenv, initCtx, common.ECDSAToAddress(orchcmd.EVMPrivateKey))

	runenv.RecordMessage("waiting for validator to start......")
	// wait for the validator to start
	time.Sleep(4 * time.Minute)

	if initCtx.GroupSeq == 1 {
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

		runenv.RecordMessage("starting bootstrapper........")
		go orchcmd.StartOrchestrator(common.ECDSAToAddress(orchcmd.EVMPrivateKey).Hex(), common.EVMPrivateKeyPassphrase, "", "")
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
				// to the format /ip4/127.0.0.1/tcp/30000/p2p/12D3KooWQKobCvC2jms83hGeer8iSSxcxSKa9x7RyWMTKdTKoNvH
				bootstrappers = append(bootstrappers, fmt.Sprintf(
					"/ip4/%s/tcp/30000/p2p/%s",
					bootstrapper.IP.To4().String(),
					bootstrapper.P2PID,
				))
			}
		}

		runenv.RecordMessage("waiting for bootstrapper node to be up")
		time.Sleep(time.Minute)
		go orchcmd.StartOrchestrator(common.ECDSAToAddress(orchcmd.EVMPrivateKey).Hex(), common.EVMPrivateKeyPassphrase, "", strings.Join(bootstrappers, ","))
	}

	_, err = syncclient.SignalAndWait(ctx, testkit.FinishState, runenv.TestInstanceCount)
	if err != nil {
		return err
	}

	return nil
}

func RunValidatorWithRelayer(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	runenv.RecordMessage("running relayer........")

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*40)
	defer cancel()

	syncclient := initCtx.SyncClient
	netclient := network.NewClient(syncclient, runenv)

	netclient.MustWaitNetworkInitialized(ctx)

	relCmd, err := common.BuildRelayer(ctx, runenv, initCtx)
	if err != nil {
		return err
	}

	go RunValidatorWithEVMAddress(runenv, initCtx, common.ECDSAToAddress(relCmd.EVMPrivateKey))

	runenv.RecordMessage("waiting for validator to start......")
	// wait for the validator to start
	time.Sleep(4 * time.Minute)

	runenv.RecordMessage("getting bootstrappers information........")
	bootstrapperCh := make(chan *qgbkit.BootstrapperNode)
	sub, err := initCtx.SyncClient.Subscribe(ctx, testkit.QGBBootstrapperTopic, bootstrapperCh)
	if err != nil {
		return err
	}

	bs := ""
	select {
	case err := <-sub.Done():
		if err != nil {
			return err
		}
	case bootstrapper := <-bootstrapperCh:
		// to the format /ip4/127.0.0.1/tcp/30000/p2p/12D3KooWQKobCvC2jms83hGeer8iSSxcxSKa9x7RyWMTKdTKoNvH
		bs = fmt.Sprintf(
			"/ip4/%s/tcp/30000/p2p/%s",
			bootstrapper.IP.To4().String(),
			bootstrapper.P2PID,
		)
	}

	chainID := runenv.StringParam("chain-id")
	if chainID == "" {
		return fmt.Errorf("invalid chain ID. please set it in configuration")
	}
	evmRPC := runenv.StringParam("evm-rpc")
	if chainID == "" {
		return fmt.Errorf("invalid EVM RPC. please set it in configuration")
	}

	// to give time for validators to register their EVM addresses
	time.Sleep(time.Minute)

	retries := 0
	var addr string
	for {
		addr, err = relCmd.DeployContract(
			common.ECDSAToAddress(relCmd.EVMPrivateKey).Hex(),
			common.EVMPrivateKeyPassphrase,
			chainID,
			evmRPC,
		)
		if err == nil {
			break
		}
		if retries > 5 {
			return err
		}
		runenv.RecordMessage(err.Error())
		retries++
		time.Sleep(10 * time.Second)
		fmt.Println("retrying deploying contract")
	}

	go func() {
		err := relCmd.StartRelayer(
			common.ECDSAToAddress(relCmd.EVMPrivateKey).Hex(),
			common.EVMPrivateKeyPassphrase,
			chainID,
			evmRPC,
			addr,
			"",
			bs,
		)
		if err != nil {
			runenv.RecordMessage(err.Error())
		}
	}()

	_, err = syncclient.SignalAndWait(ctx, testkit.FinishState, runenv.TestInstanceCount)
	if err != nil {
		return err
	}

	return nil
}

// RunValidatorWithEVMAddress runs a validator with the specified EVM address.
func RunValidatorWithEVMAddress(runenv *runtime.RunEnv, initCtx *run.InitContext, evmAddr *common2.Address) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*40)
	defer cancel()

	syncclient := initCtx.SyncClient
	netclient := network.NewClient(syncclient, runenv)

	netclient.MustWaitNetworkInitialized(ctx)

	config := appsync.CreateConfig(runenv, initCtx)

	err := netclient.ConfigureNetwork(ctx, &config)
	if err != nil {
		return err
	}

	appcmd, err := common.BuildValidator(ctx, runenv, initCtx)
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

	err = RegisterEVMAddress(runenv, appcmd, evmAddr)
	if err != nil {
		runenv.RecordFailure(err)
		return err
	}

	err = appsync.SubmitPFBs(runenv, appcmd)
	if err != nil {
		return err
	}

	// keep the validator running long enough for attestations to get signed
	time.Sleep(40 * time.Minute)

	return nil
}

func RegisterEVMAddress(runenv *runtime.RunEnv, appcmd *appkit.AppKit, evmAddr *common2.Address) error {
	runenv.RecordMessage("Registering EVM address for validator")
	return appcmd.RegisterEVMAddress(
		appcmd.ValopAddress,
		evmAddr.Hex(),
		"test",
		appcmd.GetHomePath(),
		appcmd.AccountName,
	)
}
