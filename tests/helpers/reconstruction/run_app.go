package reconstruction

import (
	"context"
	"net"
	"time"

	"github.com/celestiaorg/test-infra/testkit"
	"github.com/celestiaorg/test-infra/testkit/appkit"
	"github.com/celestiaorg/test-infra/tests/helpers/common"
	"github.com/testground/sdk-go/network"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
)

func RunAppValidator(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Minute*time.Duration(runenv.IntParam("execution-time")),
	)
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
	// to fill in the last 2 octets of the new IP address for the instance
	ipC := byte((initCtx.GlobalSeq >> 8) + 1)
	ipD := byte(initCtx.GlobalSeq)
	config.IPv4.IP = append(config.IPv4.IP[0:2:2], ipC, ipD)

	err := netclient.ConfigureNetwork(ctx, &config)
	if err != nil {
		return err
	}

	appcmd, err := common.BuildValidator(ctx, runenv, initCtx)
	if err != nil {
		return err
	}

	runenv.RecordMessage("starting........")
	go appcmd.StartNode("info")

	// wait for a new block to be produced
	time.Sleep(1 * time.Minute)

	_, err = syncclient.SignalEntry(ctx, "validator-ready")
	if err != nil {
		return err
	}

	runenv.RecordMessage("publishing app-validator address")
	ip, err := initCtx.NetClient.GetDataNetworkIP()
	if err != nil {
		return err
	}

	appId := int(initCtx.GroupSeq)
	_, err = syncclient.Publish(
		ctx,
		testkit.AppNodeTopic,
		&testkit.AppNodeInfo{
			ID: appId,
			IP: ip,
		},
	)
	if err != nil {
		return err
	}

	for i := 0; i < runenv.IntParam("submit-times"); i++ {
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

		s, err := appkit.GetLatestsBlockSize(net.ParseIP("127.0.0.1"))
		if err != nil {
			runenv.RecordMessage("err in last size call, %s", err.Error())
		}

		runenv.RecordMessage("latest size on iteration %d of the block is - %d", i, s)
	}

	_, err = syncclient.SignalAndWait(ctx, testkit.FinishState, runenv.TestInstanceCount)
	if err != nil {
		return err
	}

	return nil
}
