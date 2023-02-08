package dasbenchmarks

import (
	"context"
	"net"
	"time"

	"github.com/celestiaorg/test-infra/testkit/appkit"
	"github.com/celestiaorg/test-infra/tests/helpers/common"

	"github.com/celestiaorg/test-infra/testkit"
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

	err := netclient.ConfigureNetwork(ctx, &config)
	if err != nil {
		return err
	}

	// false to disable peer discovery since we are runnign a singular validator
	appcmd, err := common.BuildValidator(ctx, runenv, initCtx, false)
	if err != nil {
		return err
	}

	// signal startup
	_, err = syncclient.SignalEntry(ctx, testkit.ValidatorReadyTopic)
	if err != nil {
		return err
	}

	runenv.RecordMessage("starting........")
	go appcmd.StartNode("info")

	// wait for a new block to be produced
	// RPC is also being initialized...
	time.Sleep(1 * time.Minute)

	runenv.RecordMessage("publishing app-validator address")

	ip, err := initCtx.NetClient.GetDataNetworkIP()
	if err != nil {
		return err
	}

	_, err = syncclient.Publish(
		ctx,
		testkit.AppNodeTopic,
		&testkit.AppNodeInfo{
			ID: int(initCtx.GroupSeq),
			IP: ip,
		},
	)

	if err != nil {
		return err
	}

	// // wait for a new block to be produced
	time.Sleep(1 * time.Minute)

	l, err := syncclient.Barrier(ctx, testkit.LightNodesStartedState, runenv.IntParam("light"))
	if err != nil {
		runenv.RecordFailure(err)
	}
	lerr := <-l.C
	if lerr != nil {
		runenv.RecordFailure(lerr)
	}

	for j := 0; j < runenv.IntParam("submit-times"); j++ {
		runenv.RecordMessage("Submitting PFD with %d bytes random data", runenv.IntParam("msg-size"))
		err = appcmd.PayForBlob(
			appcmd.AccountAddress,
			runenv.IntParam("msg-size"),
			"test",
			appcmd.GetHomePath(),
		)
		if err != nil {
			runenv.RecordFailure(err)
			return err
		}

		_, _, err := appkit.GetLatestBlockSizeAndHeight(net.ParseIP("127.0.0.1"))
		if err != nil {
			runenv.RecordMessage("err in last size call, %s", err.Error())
		}
	}

	l, err = syncclient.Barrier(ctx, testkit.FinishState, runenv.IntParam("light")+runenv.IntParam("bridge"))
	if err != nil {
		runenv.RecordFailure(err)
	}
	lerr = <-l.C
	if lerr != nil {
		runenv.RecordFailure(lerr)
	}
	runenv.RecordSuccess()

	return nil
}
