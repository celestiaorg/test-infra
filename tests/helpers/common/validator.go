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

	tmrand "github.com/tendermint/tendermint/libs/rand"

	"github.com/celestiaorg/test-infra/testkit"
	"github.com/celestiaorg/test-infra/testkit/appkit"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
	"github.com/testground/sdk-go/sync"
)

func BuildValidator(ctx context.Context, runenv *runtime.RunEnv, initCtx *run.InitContext) (*appkit.AppKit, error) {
	home := "/.celestia-app"
	runenv.RecordMessage(home)

	cmd, keyringName, accAddr, err := InitChainAndMaybeBroadcastGenesis(ctx, runenv, initCtx, home)
	if err != nil {
		return nil, err
	}

	runenv.RecordMessage("Validator is signing its own GenTx")
	_, err = cmd.SignGenTx(keyringName, "5000000000utia", "test", home)
	if err != nil {
		return nil, err
	}

	err = BroadcastAndCollectGenTx(ctx, home, accAddr, initCtx, runenv, cmd)
	if err != nil {
		return nil, err
	}

	ip, err := UpdateAndPublishConfig(ctx, home, cmd, initCtx)
	if err != nil {
		return nil, err
	}

	if runenv.IntParam("validator") > 1 {
		err := DiscoverPeers(ctx, home, ip, initCtx, runenv)
		if err != nil {
			return nil, err
		}
	}

	return cmd, nil
}

func InitChainAndMaybeBroadcastGenesis(
	ctx context.Context,
	runenv *runtime.RunEnv,
	initCtx *run.InitContext,
	home string,
) (*appkit.AppKit, string, string, error) {
	syncclient := initCtx.SyncClient

	const chainId string = "private"
	cmd := appkit.New(home, chainId)

	keyringName := fmt.Sprintf("keyName-%d", initCtx.GroupSeq)
	accAddr, valopAddr, err := cmd.CreateKey(keyringName, "test", home)
	if err != nil {
		return nil, "", "", err
	}
	cmd.AccountAddress = accAddr
	cmd.ValopAddress = valopAddr
	cmd.AccountName = keyringName

	// we need this dirty-hack to check the k8s cluster has
	// the time to ramp up all the instances
	time.Sleep(30 * time.Second)

	seq, err := syncclient.Publish(ctx, testkit.AccountAddressTopic, accAddr)
	if err != nil {
		return nil, "", "", err
	}

	accAddrCh := make(chan string)
	_, err = syncclient.Subscribe(ctx, testkit.AccountAddressTopic, accAddrCh)
	if err != nil {
		return nil, "", "", err
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
			return nil, "", "", err
		}
		runenv.RecordMessage("Chain initialised")

		gen, err := os.Open(fmt.Sprintf("%s/config/genesis.json", home))
		if err != nil {
			return nil, "", "", err
		}

		bt, err := io.ReadAll(gen)
		if err != nil {
			return nil, "", "", err
		}

		_, err = syncclient.Publish(ctx, testkit.InitialGenesisTopic, string(bt))
		if err != nil {
			return nil, "", "", err
		}

		runenv.RecordMessage("Orchestrator has sent initial genesis")
	} else {
		initGenCh := make(chan string)
		sub, err := syncclient.Subscribe(ctx, testkit.InitialGenesisTopic, initGenCh)
		if err != nil {
			return nil, "", "", err
		}
		select {
		case err = <-sub.Done():
			if err != nil {
				return nil, "", "", err
			}
		case initGen := <-initGenCh:
			runenv.RecordMessage(initGen)
			err = os.WriteFile(fmt.Sprintf("%s/config/genesis.json", home), []byte(initGen), 0777)
			if err != nil {
				return nil, "", "", err
			}
		}
		runenv.RecordMessage("Validator has received the initial genesis")
	}

	for _, v := range accounts {
		_, err := cmd.AddGenAccount(v, "10000000000000000utia")
		if err != nil {
			return nil, "", "", err
		}
	}

	return cmd, keyringName, accAddr, nil
}

func BroadcastAndCollectGenTx(ctx context.Context, home string, accAddr string, initCtx *run.InitContext, runenv *runtime.RunEnv, cmd *appkit.AppKit) error {
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

		bt, err := io.ReadAll(gentx)
		if err != nil {
			return err
		}

		_, err = initCtx.SyncClient.Publish(ctx, testkit.GenesisTxTopic, string(bt))
		if err != nil {
			return err
		}
	}

	genTxCh := make(chan string)
	sub, err := initCtx.SyncClient.Subscribe(ctx, testkit.GenesisTxTopic, genTxCh)
	if err != nil {
		return err
	}

	for i := 0; i < runenv.IntParam("validator"); i++ {
		select {
		case err = <-sub.Done():
			if err != nil {
				return err
			}
		case genTx := <-genTxCh:
			if !strings.Contains(genTx, accAddr) {
				err := os.WriteFile(fmt.Sprintf("%s/config/gentx/%d.json", home, i), []byte(genTx), 0777)
				if err != nil {
					return err
				}
			}
		}
	}

	_, err = cmd.CollectGenTxs()
	if err != nil {
		return err
	}
	return nil
}

func GetRandomisedPeers(randomizer int, peersRange int, peers []appkit.ValidatorNode) []appkit.ValidatorNode {
	// if the test-case wants only a single validator, then we return nil
	if peersRange == 0 || randomizer > len(peers) {
		return nil
	}

	// if peersRange is 1, then only one peer is added to the address book
	if peersRange == 1 {
		return []appkit.ValidatorNode{peers[randomizer]}
	}

	// if peersRange is greater or equal to the number of peers, then all peers are added to the address book
	if peersRange >= len(peers) {
		fmt.Println("peersRange is greater or equal than the number of peers, which is not ok")
		return peers
	}

	startIndex := randomizer
	endIndex := randomizer + peersRange
	// if the endIndex is greater than the number of peers, then we set the endIndex to the number of peers
	// this is to avoid out of bounds error when slicing the peers array
	if endIndex > len(peers) && ((endIndex - len(peers)) < peersRange) {
		startIndex = endIndex - len(peers)
		endIndex = len(peers)
	} else if (endIndex - startIndex) > peersRange {
		endIndex = startIndex + peersRange
	} else if (endIndex - startIndex) < peersRange {
		startIndex = endIndex - peersRange
	}

	return peers[startIndex:endIndex]
}

func UpdateAndPublishConfig(
	ctx context.Context,
	home string,
	cmd *appkit.AppKit,
	initCtx *run.InitContext,
) (net.IP, error) {
	configPath := filepath.Join(home, "config", "config.toml")
	err := appkit.ChangeRPCServerAddress(configPath, net.ParseIP("0.0.0.0"))
	if err != nil {
		return nil, err
	}

	err = changeConfig(configPath, "v1")
	if err != nil {
		return nil, err
	}

	ip, err := initCtx.NetClient.GetDataNetworkIP()
	if err != nil {
		return nil, err
	}

	nodeId, err := cmd.GetNodeId()
	if err != nil {
		return nil, err
	}

	_, err = initCtx.SyncClient.Publish(
		ctx,
		testkit.ValidatorPeerTopic,
		&appkit.ValidatorNode{
			PubKey: nodeId,
			IP:     ip,
		},
	)
	if err != nil {
		return nil, err
	}
	return ip, nil
}

func DiscoverPeers(ctx context.Context, home string, ip net.IP, initCtx *run.InitContext, runenv *runtime.RunEnv) error {
	syncclient := initCtx.SyncClient
	valCh := make(chan *appkit.ValidatorNode)
	sub, err := syncclient.Subscribe(ctx, testkit.ValidatorPeerTopic, valCh)
	if err != nil {
		return err
	}

	runenv.RecordMessage("Discovering peers")
	var peers []appkit.ValidatorNode
	for i := 0; i < runenv.IntParam("validator"); i++ {
		select {
		case err = <-sub.Done():
			if err != nil {
				return err
			}
		case val := <-valCh:
			if !val.IP.Equal(ip) {
				peers = append(peers, *val)
			}
		}
	}
	runenv.RecordMessage("Validator Received is equal to: %d", len(peers))
	randomizer := tmrand.Intn(runenv.IntParam("validator"))
	runenv.RecordMessage("Randomized number is equal to: %d", randomizer)
	peersRange := runenv.IntParam("persistent-peers")
	runenv.RecordMessage("Peers Range is equal to: %d", peersRange)
	randPeers := GetRandomisedPeers(randomizer, peersRange, peers)
	if randPeers == nil {
		runenv.RecordMessage("No peers added to the address book")
		return nil
	}

	err = appkit.AddPeersToAddressBook(home, randPeers)
	if err != nil {
		return err
	}

	runenv.RecordMessage("Added %d to the address book", len(randPeers))
	return nil
}

func changeConfig(path, mempool string) error {
	cfg := map[string]map[string]interface{}{
		"mempool": {
			"version": mempool,
		},
		"consensus": {
			"timeout_propose":   "200ms",
			"timeout_prevote":   "200ms",
			"timeout_precommit": "200ms",
			"timeout_commit":    "200ms",
		},
		"rpc": {
			"max_subscriptions_per_client": 150,
			"timeout_broadcast_tx_commit":  "40s",
			"max_body_bytes":               6000000,
			"max_header_bytes":             6048576,
		},
		"p2p": {
			"max_num_inbound_peers":       40,
			"max_num_outbound_peers":      30,
			"send_rate":                   10240000,
			"recv_rate":                   10240000,
			"max_packet_msg_payload_size": 1024,
			"persistent_peers":            "",
		},
		"instrumentation": {
			"prometheus":             true,
			"prometheus_listen_addr": ":26660",
			"max_open_connections":   100,
			"namespace":              "tendermint",
		},
		"tx_index": {
			"indexer": "kv",
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
