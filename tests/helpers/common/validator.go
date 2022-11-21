package common

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/celestiaorg/test-infra/testkit"
	"github.com/celestiaorg/test-infra/testkit/appkit"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
	"github.com/testground/sdk-go/sync"
)

func BuildValidator(ctx context.Context, runenv *runtime.RunEnv, initCtx *run.InitContext) (*appkit.AppKit, error) {
	syncclient := initCtx.SyncClient

	home := fmt.Sprintf("/.celestia-app-%d", initCtx.GroupSeq)
	runenv.RecordMessage(home)

	const chainId string = "private"
	cmd := appkit.New(home, chainId)

	keyringName := fmt.Sprintf("keyName-%d", initCtx.GroupSeq)
	accAddr, err := cmd.CreateKey(keyringName, "test", home)
	if err != nil {
		return nil, err
	}
	cmd.AccountAddress = accAddr

	// we need this dirty-hack to check the k8s cluster has
	// the time to ramp up all the instances
	time.Sleep(30 * time.Second)
	seq, err := syncclient.Publish(ctx, testkit.AccountAddressTopic, accAddr)
	if err != nil {
		return nil, err
	}

	accAddrCh := make(chan string)
	_, err = syncclient.Subscribe(ctx, testkit.AccountAddressTopic, accAddrCh)
	if err != nil {
		return nil, err
	}

	var accounts []string
	for i := 0; i < runenv.IntParam("validator"); i++ {
		addr := <-accAddrCh
		accounts = append(accounts, addr)
	}

	moniker := fmt.Sprintf("validator-%d", initCtx.GroupSeq)

	// Here we assign the first instance to be the orchestrator role
	//
	// Orchestrator is only initing the chain and sending the genesis.json
	// to others, so the genesis time is the same everywhere
	if seq == 1 {
		_, err = cmd.InitChain(moniker)
		if err != nil {
			return nil, err
		}
		runenv.RecordMessage("Chain initialised")

		gen, err := os.Open(fmt.Sprintf("%s/config/genesis.json", home))
		if err != nil {
			return nil, err
		}

		bt, err := io.ReadAll(gen)
		if err != nil {
			return nil, err
		}

		_, err = syncclient.Publish(ctx, testkit.InitialGenenesisTopic, string(bt))
		if err != nil {
			return nil, err
		}

		runenv.RecordMessage("Orchestrator has sent initial genesis with accounts")
	} else {
		initGenCh := make(chan string)
		sub, err := syncclient.Subscribe(ctx, testkit.InitialGenenesisTopic, initGenCh)
		if err != nil {
			return nil, err
		}
		select {
		case err = <-sub.Done():
			if err != nil {
				return nil, err
			}
		case initGen := <-initGenCh:
			err = os.WriteFile(fmt.Sprintf("%s/config/genesis.json", home), []byte(initGen), 0777)
			if err != nil {
				return nil, err
			}
		}
		runenv.RecordMessage("Validator has received the initial genesis")
	}

	for _, v := range accounts {
		_, err := cmd.AddGenAccount(v, "1000000000000000utia")
		if err != nil {
			return nil, err
		}
	}

	runenv.RecordMessage("Validator is signing its own GenTx")
	_, err = cmd.SignGenTx(keyringName, "5000000000utia", "test", home)
	if err != nil {
		return nil, err
	}

	fs, err := os.ReadDir(fmt.Sprintf("%s/config/gentx", home))
	if err != nil {
		return nil, err
	}
	// slice is needed because of auto-gen gentx-name
	for _, f := range fs {
		gentx, err := os.Open(fmt.Sprintf("%s/config/gentx/%s", home, f.Name()))
		if err != nil {
			return nil, err
		}

		bt, err := io.ReadAll(gentx)
		if err != nil {
			return nil, err
		}

		_, err = syncclient.Publish(ctx, testkit.GenesisTxTopic, string(bt))
		if err != nil {
			return nil, err
		}

	}

	genTxCh := make(chan string)
	sub, err := syncclient.Subscribe(ctx, testkit.GenesisTxTopic, genTxCh)
	if err != nil {
		return nil, err
	}

	for i := 0; i < runenv.IntParam("validator"); i++ {
		select {
		case err = <-sub.Done():
			if err != nil {
				return nil, err
			}
		case genTx := <-genTxCh:
			if !strings.Contains(genTx, accAddr) {
				err := os.WriteFile(fmt.Sprintf("%s/config/gentx/%d.json", home, i), []byte(genTx), 0777)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	_, err = cmd.CollectGenTxs()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(home, "config", "config.toml")
	err = appkit.ChangeRPCServerAddress(configPath, net.ParseIP("0.0.0.0"))
	if err != nil {
		return nil, err
	}

	err = changeConfig(configPath)
	if err != nil {
		return nil, err
	}

	ip, err := initCtx.NetClient.GetDataNetworkIP()
	if err != nil {
		return nil, err
	}

	if initCtx.GroupSeq <= int64(runenv.IntParam("persistent-peers")) {
		nodeId, err := cmd.GetNodeId()
		if err != nil {
			return nil, err
		}

		_, err = syncclient.Publish(
			ctx,
			testkit.ValidatorPeerTopic,
			&appkit.ValidatorNode{
				PubKey: nodeId,
				IP:     ip},
		)
		if err != nil {
			return nil, err
		}
	}

	valCh := make(chan *appkit.ValidatorNode)
	sub, err = syncclient.Subscribe(ctx, testkit.ValidatorPeerTopic, valCh)
	if err != nil {
		return nil, err
	}

	var persPeers []string
	for i := 0; i < runenv.IntParam("persistent-peers"); i++ {
		select {
		case err = <-sub.Done():
			if err != nil {
				return nil, err
			}
		case val := <-valCh:
			runenv.RecordMessage("Validator Received: %s, %s", val.IP, val.PubKey)
			if !val.IP.Equal(ip) {
				persPeers = append(persPeers, fmt.Sprintf("%s@%s", val.PubKey, val.IP.To4().String()))
			}

			err = appkit.AddPersistentPeers(configPath, persPeers)
			if err != nil {
				return nil, err
			}
		}
	}

	return cmd, nil
}

func changeConfig(path string) error {
	cfg := map[string]map[string]string{
		"consensus": {
			"timeout_propose":   "10s",
			"timeout_prevote":   "1s",
			"timeout_precommit": "1s",
			"timeout_commit":    "15s",
		},
		"rpc": {
			"timeout_broadcast_tx_commit": "30s",
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

func GetValidatorInfo(ctx context.Context, syncclient sync.Client, valAmount, id int) (*testkit.AppNodeInfo, error) {
	appInfoCh := make(chan *testkit.AppNodeInfo, valAmount)
	sub, err := syncclient.Subscribe(ctx, testkit.AppNodeTopic, appInfoCh)
	if err != nil {
		return nil, err
	}

	for {
		select {
		case err = <-sub.Done():
			if err != nil {
				return nil, fmt.Errorf("no app has been sent for this node to connect to remotely")
			}
		case appInfo := <-appInfoCh:
			if (appInfo.ID % valAmount) == (id % valAmount) {
				return appInfo, nil
			}
		}
	}
}
