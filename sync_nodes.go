package main

import (
	"context"
	"time"

	"github.com/celestiaorg/test-infra/synctest"

	"github.com/testground/sdk-go/network"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
)

func syncNodes(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	ctx := context.Background()
	client := initCtx.SyncClient
	netclient := network.NewClient(client, runenv)

	netclient.MustWaitNetworkInitialized(ctx)

	config := network.Config{
		Network: "default",

		Enable: true,

		Default: network.LinkShape{
			Latency:   100 * time.Millisecond,
			Bandwidth: 1 << 20, // 1Mib
		},

		CallbackState: "network-configured",
		RoutingPolicy: network.AllowAll,
	}

	config.IPv4 = runenv.TestSubnet
	ipC := byte((initCtx.GlobalSeq >> 8) + 1)
	ipD := byte(initCtx.GlobalSeq)
	config.IPv4.IP = append(config.IPv4.IP[0:2:2], ipC, ipD)

	err := netclient.ConfigureNetwork(ctx, &config)
	if err != nil {
		runenv.RecordCrash(err)
		return err
	}

	switch in := runenv.TestGroupID; in {
	case "app":
		err = synctest.RunAppValidator(runenv, initCtx)
	case "bridge":
		err = synctest.RunBridgeNode(runenv, initCtx)
	case "full":
		err = synctest.RunFullNode(runenv, initCtx)
	case "light":
		err = synctest.RunLightNode(runenv, initCtx)
	}

	if err != nil {
		return err
	}
	return nil
}
