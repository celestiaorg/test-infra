package fundaccounts

import (
	"context"
	"time"

	"github.com/celestiaorg/test-infra/testkit"
	"github.com/celestiaorg/test-infra/plans/001-big-blocks/tests/common"

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
	go appcmd.StartNode("error")

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

	accsCh := make(chan string)
	runenv.RecordMessage("start funding celestia-node accounts")
	sub, err := syncclient.Subscribe(ctx, testkit.FundAccountTopic, accsCh)
	if err != nil {
		return err
	}

	total := runenv.TestInstanceCount - runenv.IntParam("validator")
	var fundAccs []string
	for i := 0; i < total; i++ {
		select {
		case err = <-sub.Done():
			if err != nil {
				return err
			}
		case account := <-accsCh:
			runenv.RecordMessage(account)
			fundAccs = append(fundAccs, account)
		}
	}

	err = appcmd.FundAccounts(
		appcmd.AccountAddress,
		"10000000utia",
		"test",
		appcmd.GetHomePath(),
		fundAccs...)
	if err != nil {
		return err
	}

	_, err = syncclient.SignalEntry(ctx, testkit.AccountsFundedState)
	if err != nil {
		return err
	}

	_, err = syncclient.SignalAndWait(ctx, testkit.FinishState, runenv.TestInstanceCount)
	if err != nil {
		return err
	}

	return nil
}
