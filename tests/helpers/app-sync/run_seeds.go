package appsync

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/celestiaorg/test-infra/testkit"
	"github.com/celestiaorg/test-infra/testkit/appkit"
	"github.com/celestiaorg/test-infra/tests/helpers/common"

	"github.com/testground/sdk-go/network"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
)

// TODO(@Bidon15): seed nodes are not working as expected
// Now this code is not used anywhere in test-plan/cases
// More info to follow up: https://github.com/tendermint/tendermint/issues/9289
func RunSeed(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*10)
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

	home := fmt.Sprintf("/.celestia-app-%d", initCtx.GroupSeq)
	runenv.RecordMessage(home)

	cmd := appkit.New(home, "private")
	configPath := filepath.Join(home, "config", "config.toml")

	moniker := fmt.Sprintf("seed-%d", initCtx.GroupSeq)
	_, err = cmd.InitChain(moniker)
	if err != nil {
		return err
	}
	runenv.RecordMessage("Chain initialised")

	valCh := make(chan *appkit.ValidatorNode)
	sub, err := syncclient.Subscribe(ctx, testkit.ValidatorPeerTopic, valCh)

	if err != nil {
		return err
	}

	for i := 0; i < runenv.IntParam("persistent-peers"); i++ {
		select {
		case err = <-sub.Done():
			if err != nil {
				return err
			}
		case val := <-valCh:
			runenv.RecordMessage("Validator Received: %s, %s", val.IP, val.PubKey)
		}
	}

	ipCh := make(chan *string)
	sub, err = syncclient.Subscribe(ctx, testkit.CurlGenesisState, ipCh)
	if err != nil {
		return err
	}

	runenv.RecordMessage("before barrier for the ip we need")
	ip := <-ipCh
	runenv.RecordMessage("curling genesis state from this validator's ip - %s", *ip)
	time.Sleep(30 * time.Second)

	uri := fmt.Sprintf("http://%s:26657/genesis", *ip)
	genState, err := appkit.GetGenesisState(uri)
	if err != nil {
		return err
	}

	err = genState.Genesis.SaveAs(fmt.Sprintf("%s/config/genesis.json", home))
	if err != nil {
		return err
	}

	// We need to curl the instance 1 with RPC to get the genesis.json file
	// Only 1 validator must fire up to provide the RPC

	nodeId, err := cmd.GetNodeId()
	if err != nil {
		return err
	}

	err = appkit.ChangeConfigParam(configPath, "p2p", "seed_mode", true)
	if err != nil {
		return err
	}

	_, err = syncclient.Publish(
		ctx,
		testkit.SeedNodeTopic,
		&appkit.ValidatorNode{
			PubKey: nodeId,
			IP:     config.IPv4.IP},
	)
	if err != nil {
		return err
	}

	go cmd.StartNode("info")

	// // wait for a new block to be produced
	time.Sleep(4 * time.Minute)

	runenv.RecordSuccess()

	return nil
}

//func saveGenesisState(runenv *runtime.RunEnv, sub *sync.Subscription, ipCh chan *string, home string) error {
//
//}
