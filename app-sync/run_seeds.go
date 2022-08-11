package appsync

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/celestiaorg/test-infra/testkit"
	"github.com/celestiaorg/test-infra/testkit/appkit"

	"github.com/testground/sdk-go/network"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
)

func RunSeed(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*4)
	defer cancel()

	client := initCtx.SyncClient
	netclient := network.NewClient(client, runenv)

	netclient.MustWaitNetworkInitialized(ctx)

	config := network.Config{
		Network: "default",
		Enable:  true,
		Default: network.LinkShape{
			// Latency:   100 * time.Millisecond,
			Bandwidth: 4 << 26, // 256Mib
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
	err = os.Mkdir(home, 0777)
	if err != nil {
		return err
	}

	err = <-client.MustBarrier(ctx, testkit.FinalGenesisState, 1).C
	if err != nil {
		return err
	}

	cmd := appkit.New()

	initGenCh := make(chan string)
	sub, err := client.Subscribe(ctx, testkit.InitialGenenesisTopic, initGenCh)
	if err != nil {
		return err
	}
	select {
	case err = <-sub.Done():
		if err != nil {
			return err
		}
	case initGen := <-initGenCh:
		err = os.Mkdir(fmt.Sprintf("%s/config", home), 0777)
		if err != nil {
			return err
		}
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

	nodeId, err := cmd.GetNodeId(home)
	if err != nil {
		return err
	}
	_, err = client.Publish(
		ctx,
		testkit.ValidatorPeerTopic,
		&appkit.ValidatorNode{
			PubKey: nodeId,
			IP:     config.IPv4.IP},
	)
	if err != nil {
		return err
	}

	go cmd.StartNode(home, "info")

	// // wait for a new block to be produced
	time.Sleep(10 * time.Minute)

	runenv.RecordSuccess()
	// }

	return nil
}
