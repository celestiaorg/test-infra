package appsync

import (
	"context"
	"fmt"
	tmrand "github.com/tendermint/tendermint/libs/rand"
	"path/filepath"
	"time"

	"github.com/celestiaorg/test-infra/testkit"
	"github.com/celestiaorg/test-infra/testkit/appkit"
	"github.com/celestiaorg/test-infra/tests/helpers/common"

	"github.com/testground/sdk-go/network"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
)

// RunSeed configures a tendermint full node running with seed settings
func RunSeed(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*time.Duration(runenv.IntParam("execution-time")))
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

	var peers []appkit.ValidatorNode
	for i := 0; i < runenv.IntParam("validator"); i++ {
		select {
		case err = <-sub.Done():
			if err != nil {
				return err
			}
		case val := <-valCh:
			peers = append(peers, *val)
		}
	}
	runenv.RecordMessage("Validator Received is equal to: %d", len(peers))

	randomizer := tmrand.Intn(runenv.IntParam("validator"))
	peersRange := runenv.IntParam("validator") / runenv.IntParam("seed")
	randPeers := common.GetRandomisedPeers(randomizer, peersRange, peers)
	if randPeers == nil {
		return fmt.Errorf("no peers added for seed's addrbook, got %s", randPeers)
	}

	err = appkit.AddPeersToAddressBook(home, randPeers)
	if err != nil {
		return err
	}

	runenv.RecordMessage("Added %d to the address book", len(randPeers))

	ipCh := make(chan *string)
	sub, err = syncclient.Subscribe(ctx, testkit.CurlGenesisState, ipCh)
	if err != nil {
		return err
	}

	var uri string
	select {
	case err := <-sub.Done():
		if err != nil {
			return err
		}
	case ip := <-ipCh:
		runenv.RecordMessage("curling genesis state from this validator's ip - %s", *ip)
		time.Sleep(30 * time.Second)

		// We need to curl the instance 1 with RPC to get the genesis.json file
		// Only 1 validator must fire up to provide the RPC
		uri = fmt.Sprintf("http://%s:26657/genesis", *ip)
	}

	genState, err := appkit.GetGenesisState(uri)
	if err != nil {
		return err
	}

	err = genState.Genesis.SaveAs(fmt.Sprintf("%s/config/genesis.json", home))
	if err != nil {
		return err
	}

	nodeId, err := cmd.GetNodeId()
	if err != nil {
		return err
	}

	err = changeConfig(configPath)
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

	// wait and crawl
	_, err = syncclient.SignalAndWait(ctx, testkit.FinishState, runenv.TestInstanceCount)
	if err != nil {
		return err
	}

	return nil
}

func changeConfig(path string) error {
	cfg := map[string]map[string]interface{}{
		"p2p": {
			"seed_mode":                   true,
			"max_num_inbound_peers":       100,
			"max_num_outbound_peers":      100,
			"max_packet_msg_payload_size": 1024,
			"persistent_peers":            "",
		},
	}

	for i, j := range cfg {
		for k, v := range j {
			err := appkit.ChangeConfigParam(path, i, k, v)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
