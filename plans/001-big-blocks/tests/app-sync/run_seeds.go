package appsync

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/celestiaorg/test-infra/testkit"
	"github.com/celestiaorg/test-infra/testkit/appkit"
	"github.com/celestiaorg/test-infra/plans/001-big-blocks/tests/common"

	"github.com/testground/sdk-go/network"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
)

// TODO(@Bidon15): seed nodes are not working as expected
// Now this code is not used anywhere in test-plan/cases
// More info to follow up: https://github.com/tendermint/tendermint/issues/9289
func RunSeed(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
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

	home := fmt.Sprintf("/.celestia-app-%d", initCtx.GroupSeq)
	runenv.RecordMessage(home)

	cmd := appkit.New(home, "tia-test")

	nodeId, err := cmd.GetNodeId()
	if err != nil {
		return err
	}

	initGenCh := make(chan string)
	sub, err := syncclient.Subscribe(ctx, testkit.InitialGenenesisTopic, initGenCh)
	if err != nil {
		return err
	}
	select {
	case err = <-sub.Done():
		if err != nil {
			return err
		}
	case initGen := <-initGenCh:
		err = os.WriteFile(fmt.Sprintf("%s/config/genesis.json", home), []byte(initGen), 0777)
		if err != nil {
			return err
		}
	}
	runenv.RecordMessage("Validator has received the initial genesis")

	configPath := filepath.Join(home, "config", "config.toml")
	err = appkit.ChangeConfigParam(configPath, "p2p", "seed_mode", true)
	if err != nil {
		return err
	}

	_, err = syncclient.Publish(
		ctx,
		testkit.ValidatorPeerTopic,
		&appkit.ValidatorNode{
			PubKey: nodeId,
			IP:     config.IPv4.IP},
	)
	if err != nil {
		return err
	}

	go cmd.StartNode("info")

	// // wait for a new block to be produced
	time.Sleep(1 * time.Minute)

	runenv.RecordSuccess()

	return nil
}
