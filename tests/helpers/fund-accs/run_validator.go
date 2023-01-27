package fundaccounts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/celestiaorg/test-infra/testkit/appkit"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/celestiaorg/test-infra/testkit"
	"github.com/celestiaorg/test-infra/tests/helpers/common"

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
	// to fill in the last 2 octets of the new IP address for the instance
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

	if initCtx.GroupSeq == 1 {
		ip, err := netclient.GetDataNetworkIP()
		if err != nil {
			return err
		}

		_, err = syncclient.Publish(ctx, testkit.CurlGenesisState, ip.To4().String())
		if err != nil {
			return err
		}

		go appcmd.StartNode("info")
	}

	seedCh := make(chan *appkit.ValidatorNode)
	sub, err := syncclient.Subscribe(ctx, testkit.SeedNodeTopic, seedCh)
	if err != nil {
		return err
	}

	var seedPeers []string
	for i := 0; i < runenv.IntParam("seed"); i++ {
		select {
		case err := <-sub.Done():
			if err != nil {
				return err
			}
		case seed := <-seedCh:
			seedPeers = append(seedPeers, fmt.Sprintf("%s@%s", seed.PubKey, seed.IP.To4().String()))
		}
	}

	configPath := filepath.Join(appcmd.Home, "config", "config.toml")
	err = appkit.AddSeedPeers(configPath, seedPeers)
	if err != nil {
		return err
	}

	if initCtx.GroupSeq != 1 {
		runenv.RecordMessage("starting........")
		go appcmd.StartNode("info")
	}

	// wait for a new block to be produced
	time.Sleep(1 * time.Minute)

	_, err = syncclient.SignalEntry(ctx, "validator-ready")
	if err != nil {
		return err
	}

	runenv.RecordMessage("publishing app-validator address")
	ip, err := initCtx.NetClient.GetDataNetworkIP()
	if err != nil {
		return err
	}

	appId := int(initCtx.GroupSeq)
	_, err = syncclient.Publish(
		ctx,
		testkit.AppNodeTopic,
		&testkit.AppNodeInfo{
			ID: appId,
			IP: ip,
		},
	)
	if err != nil {
		return err
	}

	accsCh := make(chan string)
	runenv.RecordMessage("start funding celestia-node accounts")
	sub, err = syncclient.Subscribe(ctx, testkit.FundAccountTopic, accsCh)
	if err != nil {
		return err
	}

	total := runenv.TestInstanceCount - runenv.IntParam("validator") - runenv.IntParam("seed")
	var fundAccs []string
	for i := 0; i < total; i++ {
		select {
		case err = <-sub.Done():
			if err != nil {
				return err
			}
		case account := <-accsCh:
			fundAccs = append(fundAccs, account)
		}
	}

	// var i is the range we take on funding accounts in 1 block
	for i := 0; i < len(fundAccs); i += 50 {
		accsRange := fundAccs[i : i+50]
		err = appcmd.FundAccounts(
			appcmd.AccountAddress,
			"10000000utia",
			"test",
			appcmd.GetHomePath(),
			accsRange...)
		if err != nil {
			return err
		}
	}

	_, err = syncclient.SignalEntry(ctx, testkit.AccountsFundedState)
	if err != nil {
		return err
	}

	go func() {
		timeout := time.After(4 * time.Minute)
		ticker := time.NewTicker(15 * time.Second)

		for {
			fs, err := os.ReadDir(fmt.Sprintf("%s/data", appcmd.Home))
			if err != nil {
				return
			}
			select {
			case <-timeout:
				return
			case <-ticker.C:
				// slice is needed because of auto-gen metrics.json file
				for _, f := range fs {
					met, err := os.Open(fmt.Sprintf("%s/data/%s", appcmd.Home, f.Name()))
					if err != nil {
						return
					}

					bt, err := io.ReadAll(met)
					if err != nil {
						return
					}

					var pjson bytes.Buffer
					if err := json.Indent(&pjson, bt, "", "    "); err != nil {
						return
					}
					fmt.Println(pjson.String())
				}
			}
		}

	}()

	_, err = syncclient.SignalAndWait(ctx, testkit.FinishState, runenv.TestInstanceCount)
	if err != nil {
		return err
	}

	return nil
}
