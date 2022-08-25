package appsync

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

	home := fmt.Sprintf("/.celestia-app-%d", initCtx.GlobalSeq)
	runenv.RecordMessage(home)

	cmd := appkit.New()

	keyringName := fmt.Sprintf("keyName-%d", initCtx.GlobalSeq)
	accAddr, err := cmd.CreateKey(keyringName, "test", home)
	if err != nil {
		return err
	}

	_, err = syncclient.Publish(ctx, testkit.AccountAddressTopic, accAddr)
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
		_, err = syncclient.Subscribe(ctx, testkit.AccountAddressTopic, accAddrCh)
		if err != nil {
			return err
		}

		var accounts []string
		for i := 0; i < runenv.TestGroupInstanceCount; i++ {
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
			_, err := cmd.AddGenAccount(v, "100000000000000000utia", home)
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

		_, err = syncclient.Publish(ctx, testkit.InitialGenenesisTopic, string(bt))
		if err != nil {
			return err
		}

		runenv.RecordMessage("Orchestrator has sent initial genesis with accounts")
	} else {
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

		_, err = syncclient.Publish(ctx, testkit.GenesisTxTopic, string(bt))
		if err != nil {
			return err
		}

	}

	genTxCh := make(chan string)
	sub, err := syncclient.Subscribe(ctx, testkit.GenesisTxTopic, genTxCh)
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

	if initCtx.GlobalSeq <= 10 {
		nodeId, err := cmd.GetNodeId(home)
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
	} else {
		valCh := make(chan *appkit.ValidatorNode)
		sub, err = syncclient.Subscribe(ctx, testkit.ValidatorPeerTopic, valCh)
		if err != nil {
			return err
		}

		var persPeers []string
		for i := 0; i < 10; i++ {
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

				err = appkit.AddPersistentPeers(configPath, persPeers)
				if err != nil {
					return err
				}
			}
		}
	}

	err = appkit.ChangeConfigParam(configPath, "p2p", "external_address", fmt.Sprintf("%s:26656", config.IPv4.IP.To4().String()))
	if err != nil {
		return err
	}
	runenv.RecordMessage("starting........")

	err = changeConfig(configPath)
	if err != nil {
		return err
	}
	go cmd.StartNode(home, "info")

	// // wait for a new block to be produced
	time.Sleep(1 * time.Minute)

	// If all 3 validators submit pfd - it will take too long to produce a new block
	for i := 0; i < 10; i++ {
		runenv.RecordMessage("Submitting PFD with 90k bytes random data")
		err = cmd.PayForData(
			accAddr,
			50000,
			"test",
			chainId,
			home,
		)

		if err != nil {
			runenv.RecordFailure(err)
			return err
		}
		go func() {
			s, err := appkit.GetLatestsBlockSize(net.ParseIP("127.0.0.1"))
			if err != nil {
				runenv.RecordMessage("err in last size call, %s", err.Error())
			}

			runenv.RecordMessage("latest size on iteration %d of the block is - %d", i, s)
		}()
	}

	time.Sleep(30 * time.Second)
	runenv.RecordSuccess()

	return nil
}

func changeConfig(path string) error {
	cfg := map[string]map[string]string{
		"consensus": {
			"timeout_propose":   "3s",
			"timeout_prevote":   "1s",
			"timeout_precommit": "1s",
			"timeout_commit":    "30s",
		},
		"rpc": {
			"timeout_broadcast_tx_commit": "90s",
			"max_body_bytes":              "1000000",
			"max_header_bytes":            "1048576",
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
