package synctest

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

func RunAppValidator(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*4)
	defer cancel()

	syncclient := initCtx.SyncClient
	netclient := network.NewClient(syncclient, runenv)

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
	if initCtx.GroupSeq == 1 {
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

	for i := 0; i < runenv.TestGroupInstanceCount; i++ {
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

	err = appkit.ChangeRPCServerAddress(configPath, net.ParseIP("0.0.0.0"))
	if err != nil {
		return err
	}

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

	runenv.RecordMessage("starting........")
	go cmd.StartNode(home)

	// wait for a new block to be produced
	// RPC is also being initialized...
	time.Sleep(30 * time.Second)

	_, err = syncclient.SignalEntry(ctx, testkit.AppStartedState)
	if err != nil {
		return err
	}

	// testableInstances are full and light nodes. We are multiplying app's
	// by 2 as we have ratio on 1 app per 1 bridge node
	testableInstances := runenv.TestInstanceCount - (runenv.TestGroupInstanceCount * 2)
	err = <-syncclient.MustBarrier(ctx, testkit.FinishState, testableInstances).C
	return err
}
