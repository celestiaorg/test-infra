package appsync

import (
	"context"
	"fmt"
	"github.com/celestiaorg/test-infra/testkit"
	"net"
	"path/filepath"
	"time"

	"github.com/celestiaorg/test-infra/testkit/appkit"
	"github.com/celestiaorg/test-infra/tests/helpers/common"

	"github.com/testground/sdk-go/network"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
)

func RunValidator(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*4)
	defer cancel()

	syncclient := initCtx.SyncClient
	netclient := network.NewClient(syncclient, runenv)

	netclient.MustWaitNetworkInitialized(ctx)

	config := CreateConfig(runenv, initCtx)

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

	err = HandleSeedPeers(ctx, runenv, appcmd, initCtx)
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

	err = SubmitPFBs(runenv, appcmd)
	if err != nil {
		return err
	}

	return nil
}

func CreateConfig(runenv *runtime.RunEnv, initCtx *run.InitContext) network.Config {
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

	return config
}

func SubmitPFBs(runenv *runtime.RunEnv, appcmd *appkit.AppKit) error {
	for i := 0; i < runenv.IntParam("submit-times"); i++ {
		runenv.RecordMessage("Submitting PFD with %d bytes random data", runenv.IntParam("msg-size"))
		err := appcmd.PayForBlob(
			appcmd.AccountAddress,
			runenv.IntParam("msg-size"),
			"test",
			appcmd.GetHomePath(),
		)
		if err != nil {
			runenv.RecordFailure(err)
			return err
		}

		s, err := appkit.GetLatestsBlockSize(net.ParseIP("127.0.0.1"))
		if err != nil {
			runenv.RecordMessage("err in last size call, %s", err.Error())
		}

		runenv.RecordMessage("latest size on iteration %d of the block is - %d", i, s)
	}
	return nil
}

func HandleSeedPeers(ctx context.Context, runenv *runtime.RunEnv, appcmd *appkit.AppKit, initCtx *run.InitContext) error {
	seedCh := make(chan *appkit.ValidatorNode)
	sub, err := initCtx.SyncClient.Subscribe(ctx, testkit.SeedNodeTopic, seedCh)
	if err != nil {
		return err
	}

	var seedPeers []string
	for i := 0; i < runenv.IntParam("seed"); i++ {
		select {
		case err := <-sub.Done():
			if err != nil {
				return err
			}
		case seed := <-seedCh:
			seedPeers = append(seedPeers, fmt.Sprintf("%s@%s", seed.PubKey, seed.IP.To4().String()))
		}
	}

	configPath := filepath.Join(appcmd.Home, "config", "config.toml")
	err = appkit.AddSeedPeers(configPath, seedPeers)
	if err != nil {
		return err
	}
	return nil
}
