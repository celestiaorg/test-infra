package common

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/celestiaorg/test-infra/testkit"
	"github.com/celestiaorg/test-infra/testkit/appkit"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
)

func BuildValidator(ctx context.Context, runenv *runtime.RunEnv, initCtx *run.InitContext) (*appkit.AppKit, error) {
	syncclient := initCtx.SyncClient

	home := fmt.Sprintf("/.celestia-app-%d", initCtx.GlobalSeq)
	runenv.RecordMessage(home)

	cmd := appkit.New(home)

	keyringName := fmt.Sprintf("keyName-%d", initCtx.GlobalSeq)
	accAddr, err := cmd.CreateKey(keyringName, "test", home)
	if err != nil {
		return nil, err
	}
	cmd.AccountAddress = accAddr

	_, err = syncclient.Publish(ctx, testkit.AccountAddressTopic, accAddr)
	if err != nil {
		return nil, err
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
			return nil, err
		}

		var accounts []string
		for i := 0; i < runenv.IntParam("validator"); i++ {
			addr := <-accAddrCh
			runenv.RecordMessage("Received address: %s", addr)
			accounts = append(accounts, addr)
		}

		moniker := fmt.Sprintf("validator-%d", initCtx.GlobalSeq)

		_, err = cmd.InitChain(moniker, chainId)
		if err != nil {
			return nil, err
		}

		for _, v := range accounts {
			_, err := cmd.AddGenAccount(v, "1000000000000000utia")
			if err != nil {
				return nil, err
			}
		}

		gen, err := os.Open(fmt.Sprintf("%s/config/genesis.json", home))
		if err != nil {
			return nil, err
		}

		bt, err := ioutil.ReadAll(gen)
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

	_, err = cmd.SignGenTx(keyringName, "5000000000utia", "test", chainId, home)
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

		bt, err := ioutil.ReadAll(gentx)
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
				err := ioutil.WriteFile(fmt.Sprintf("%s/config/gentx/%d.json", home, i), []byte(genTx), 0777)
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

	if initCtx.GlobalSeq <= int64(runenv.IntParam("persistent-peers")) {
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
	for i := 0; i < runenv.IntParam("validator"); i++ {
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
			"timeout_propose":   "3s",
			"timeout_prevote":   "1s",
			"timeout_precommit": "1s",
			"timeout_commit":    "25s",
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
