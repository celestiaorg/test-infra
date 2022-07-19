package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/celestiaorg/test-infra/testkit"
	"github.com/celestiaorg/test-infra/testkit/appkit"

	"github.com/testground/sdk-go/network"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
)

// This test-case is a 101 on how Celestia-App only should be started.
// In this test-case, we are testing the following scenario:
// 1. Every instance can create an account
// 2. The orchestrator(described more below) funds the account at genesis
//    and sends the initial genesis.json to the rest of the validators' set
// 3. After receiving the initial genesis.json, validators are signing the
//    genesis transaction(gentx)
// 4. Validators collects all genesis transactions
// 5. The chain is started
func initVal(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*4)
	defer cancel()

	client := initCtx.SyncClient
	netclient := network.NewClient(client, runenv)

	netclient.MustWaitNetworkInitialized(ctx)

	config := network.Config{
		Network: "default",
		Enable:  true,
		Default: network.LinkShape{
			Latency:   100 * time.Millisecond,
			Bandwidth: 1 << 20, // 1Mib
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

	cmd := appkit.New()

	keyringName := fmt.Sprintf("keyName-%d", initCtx.GlobalSeq)
	accAddr, err := cmd.CreateKey(keyringName, "test", home)
	if err != nil {
		return err
	}

	_, err = client.Publish(ctx, testkit.AccountAddressTopic, accAddr)
	if err != nil {
		return err
	}

	// Here we assign the first instance to be the orchestrator role
	//
	// Orchestrator is receiving all accounts by subscription, to then
	// execute the `add-genesis-account` command and send back to the rest
	// of the validators to set the initial genesis.json
	const chainId string = "tia-test"
	if initCtx.GlobalSeq == 1 {
		accAddrCh := make(chan string)
		_, err = client.Subscribe(ctx, testkit.AccountAddressTopic, accAddrCh)
		if err != nil {
			return err
		}

		var accounts []string
		for i := 0; i < runenv.TestInstanceCount; i++ {
			addr := <-accAddrCh
			runenv.RecordMessage("Received address: %s", addr)
			accounts = append(accounts, addr)
		}

		moniker := fmt.Sprintf("validator-%d", initCtx.GlobalSeq)

		_, err = cmd.InitChain(moniker, chainId, home)
		if err != nil {
			return err
		}

		for _, v := range accounts {
			_, err := cmd.AddGenAccount(v, "1000000000000000utia", home)
			if err != nil {
				return err
			}
		}

		gen, err := os.Open(fmt.Sprintf("%s/config/genesis.json", home))
		if err != nil {
			return err
		}

		bt, err := ioutil.ReadAll(gen)
		if err != nil {
			return err
		}

		_, err = client.Publish(ctx, testkit.InitialGenenesisTopic, string(bt))
		if err != nil {
			return err
		}

		runenv.RecordMessage("Orchestrator has sent initial genesis with accounts")
	} else {
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
			err = os.WriteFile(fmt.Sprintf("%s/config/genesis.json", home), []byte(initGen), 0777)
			if err != nil {
				return err
			}
		}
		runenv.RecordMessage("Validator has received the initial genesis")
	}

	_, err = cmd.SignGenTx(keyringName, "5000000000utia", "test", chainId, home)
	if err != nil {
		return err
	}

	fs, err := os.ReadDir(fmt.Sprintf("%s/config/gentx", home))
	if err != nil {
		return err
	}
	// slice is needed because of auto-gen gentx-name
	for _, f := range fs {
		gentx, err := os.Open(fmt.Sprintf("%s/config/gentx/%s", home, f.Name()))
		if err != nil {
			return err
		}

		bt, err := ioutil.ReadAll(gentx)
		if err != nil {
			return err
		}

		_, err = client.Publish(ctx, testkit.GenesisTxTopic, string(bt))
		if err != nil {
			return err
		}

	}

	genTxCh := make(chan string)
	sub, err := client.Subscribe(ctx, testkit.GenesisTxTopic, genTxCh)
	if err != nil {
		return err
	}

	for i := 0; i < runenv.TestInstanceCount; i++ {
		select {
		case err = <-sub.Done():
			if err != nil {
				return err
			}
		case genTx := <-genTxCh:
			if !strings.Contains(genTx, accAddr) {
				err := ioutil.WriteFile(fmt.Sprintf("%s/config/gentx/%d.json", home, i), []byte(genTx), 0777)
				if err != nil {
					return err
				}
			}
		}
	}

	_, err = cmd.CollectGenTxs(home)
	if err != nil {
		return err
	}

	configPath := filepath.Join(home, "config", "config.toml")
	err = appkit.ChangeNodeMode(configPath, "validator")
	if err != nil {
		return err
	}

	runenv.RecordMessage("starting........")

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

	valCh := make(chan *appkit.ValidatorNode)
	sub, err = client.Subscribe(ctx, testkit.ValidatorPeerTopic, valCh)
	if err != nil {
		return err
	}

	var persPeers []string
	for i := 0; i < runenv.TestGroupInstanceCount; i++ {
		select {
		case err = <-sub.Done():
			if err != nil {
				return err
			}
		case val := <-valCh:
			runenv.RecordMessage("Validator Received: %s, %s", val.IP, val.PubKey)
			if !val.IP.Equal(config.IPv4.IP) {
				persPeers = append(persPeers, fmt.Sprintf("%s@%s", val.PubKey, val.IP.To4().String()))
			}
		}
	}

	err = appkit.AddPersistentPeers(configPath, persPeers)
	if err != nil {
		return err
	}

	go cmd.StartNode(home)

	// wait for a new block to be produced
	time.Sleep(1 * time.Minute)

	blockHeight := 2
	bh, err := appkit.GetBlockHashByHeight(net.ParseIP("127.0.0.1"), blockHeight)

	if err != nil {
		return err
	}
	runenv.RecordMessage(bh)

	_, err = client.Publish(ctx, testkit.BlockHashTopic, bh)
	if err != nil {
		return err
	}

	blockHashCh := make(chan string)
	sub, err = client.Subscribe(ctx, testkit.BlockHashTopic, blockHashCh)
	if err != nil {
		return err
	}

	for i := 0; i < runenv.TestInstanceCount; i++ {
		select {
		case err := <-sub.Done():
			if err != nil {
				return err
			}
		case blockHash := <-blockHashCh:
			runenv.RecordMessage(blockHash)
			if bh != blockHash {
				return fmt.Errorf("hashes for block#%d differs", blockHeight)
			}
		}
	}
	// runenv.RecordSuccess()

	// we need to start randomizing pay for data namespace and messages
	// messages size should be defined in the params of the test-case
	// params should contain occurence of pfd too

	// we need a mechanism in app to check what is the block size in the header?
	// populate it here
	// check that we can produce at least 10-20 blocks constantly with max block size

	// change a flag for block timeout = 30-40 seconds?
	//

	for i := 0; i < 10; i++ {
		runenv.RecordMessage("iterating from -> %d", i)
		s, err := appkit.GetLatestsBlockSize(net.ParseIP("127.0.0.1"))
		if err != nil {
			return err
		}

		runenv.RecordMessage("prev size of the block is - %d", s)

		time.Sleep(15 * time.Second)
		out, err := cmd.PayForData(
			accAddr,
			appkit.GetRandomNamespace(),
			appkit.GetRandomMessageBySize(1000000),
			"test",
			chainId,
			home,
		)

		fmt.Println(err)
		fmt.Println(out)

		s, err := appkit.GetLatestsBlockSize(net.ParseIP("127.0.0.1"))
		if err != nil {
			return err
		}

		runenv.RecordMessage("latest size of the block is - %d", s)
	}

	return nil
}
