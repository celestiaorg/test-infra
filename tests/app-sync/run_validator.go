package appsync

import (
	"context"
	"net"
	"time"

	"github.com/celestiaorg/test-infra/testkit/appkit"
	"github.com/celestiaorg/test-infra/tests/common"

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
			Bandwidth: 5 << 26, // 320Mib
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

	appcmd, err := common.BuildValidator(ctx, runenv, initCtx)
	if err != nil {
		return err
	}

	runenv.RecordMessage("starting........")
	go appcmd.StartNode("info")

	// // wait for a new block to be produced
	time.Sleep(1 * time.Minute)

	// If all 3 validators submit pfd - it will take too long to produce a new block
	for i := 0; i < 10; i++ {
		runenv.RecordMessage("Submitting PFD with 90k bytes random data")
		err = appcmd.PayForData(
			appcmd.AccountAddress,
			50000,
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

	time.Sleep(30 * time.Second)
	runenv.RecordSuccess()

	return nil
}
