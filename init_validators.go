package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/celestiaorg/test-infra/testkit/appkit"

	"github.com/testground/sdk-go/network"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
	"github.com/testground/sdk-go/sync"
)

func initVal(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*4)
	defer cancel()

	client := initCtx.SyncClient
	netclient := network.NewClient(client, runenv)

	netclient.MustWaitNetworkInitialized(ctx)

	config := network.Config{
		// Control the "default" network. At the moment, this is the only network.
		Network: "default",

		// Enable this network. Setting this to false will disconnect this test
		// instance from this network. You probably don't want to do that.
		Enable: true,

		// Set the traffic shaping characteristics.
		Default: network.LinkShape{
			Latency:   100 * time.Millisecond,
			Bandwidth: 1 << 20, // 1Mib
		},

		// Set what state the sidecar should signal back to you when it's done.
		CallbackState: "network-configured",
		RoutingPolicy: network.AllowAll,
	}

	config.IPv4 = runenv.TestSubnet

	ipC := byte((initCtx.GlobalSeq >> 8) + 1)
	ipD := byte(initCtx.GlobalSeq)
	config.IPv4.IP = append(config.IPv4.IP[0:2:2], ipC, ipD)

	err := netclient.ConfigureNetwork(ctx, &config)
	if err != nil {
		return err
	}

	// init the chain
	home := fmt.Sprintf("/.celestia-app-%d", initCtx.GroupSeq)
	fmt.Println(home)

	cmd := appkit.NewRootCmd()

	addrt := sync.NewTopic("account-address", "")

	accAddr, err := appkit.CreateKey(cmd, "xm1", "test", home)
	if err != nil {
		return err
	}

	_, err = client.Publish(ctx, addrt, accAddr)
	if err != nil {
		return err
	}

	jsont := sync.NewTopic("init-gen", "")

	if initCtx.GlobalSeq == 1 {
		addrch := make(chan string)
		_, err = client.Subscribe(ctx, addrt, addrch)
		if err != nil {
			return err
		}

		var accounts []string
		for i := 0; i < runenv.TestInstanceCount; i++ {
			addr := <-addrch
			runenv.RecordMessage("Received address: %s", addr)
			accounts = append(accounts, addr)
		}

		_, err = appkit.InitChain(cmd, "kek", "tia-test", home)
		if err != nil {
			return err
		}

		for _, v := range accounts {
			_, err := appkit.AddGenAccount(cmd, v, "1000000000000000utia", home)
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

		_, err = client.Publish(ctx, jsont, string(bt))
		if err != nil {
			return err
		}

		runenv.RecordMessage("Orchestrator has sent initial genesis with accounts")
	}

	if initCtx.GlobalSeq != 1 {
		ingench := make(chan string)
		_, err := client.Subscribe(ctx, jsont, ingench)
		if err != nil {
			return err
		}

		ingen := <-ingench

		err = os.WriteFile(fmt.Sprintf("%s/config/genesis.json", home), []byte(ingen), 0777)
		if err != nil {
			return err
		}
		runenv.RecordMessage("Validator has received the initial genesis")
	}

	// TODO(@Bidon15): Figure out why we need this workaround of new sync.clients
	// instead of using the existing one
	// issue: #30
	initCtx.SyncClient, err = sync.NewBoundClient(ctx, runenv)
	gent := sync.NewTopic("genesis", "")

	if err != nil {
		return err
	}

	_, err = appkit.SignGenTx(cmd, "xm1", "5000000000utia", "test", "tia-test", home)
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

		client.Publish(ctx, gent, string(bt))

	}

	gentch := make(chan string)
	client.Subscribe(ctx, gent, gentch)

	for i := 0; i < runenv.TestInstanceCount; i++ {
		gentx := <-gentch
		if !strings.Contains(gentx, accAddr) {
			err := ioutil.WriteFile(fmt.Sprintf("%s/config/gentx/%d.json", home, i), []byte(gentx), 0777)
			if err != nil {
				return err
			}
		}
	}

	_, err = appkit.CollectGenTxs(cmd, home)
	if err != nil {
		return err
	}

	configPath := filepath.Join(home, "config", "config.toml")
	err = appkit.ChangeNodeMode(configPath, "validator")
	if err != nil {
		return err
	}

	runenv.RecordMessage("starting........")

	appkit.StartNode(cmd, home)

	return nil
}
