package synctest

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/celestiaorg/test-infra/testkit/appkit"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
	"github.com/testground/sdk-go/sync"
)

func RunAppValidator(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*2)
	defer cancel()
	client := initCtx.SyncClient

	home := fmt.Sprintf("/.celestia-app-%d", initCtx.GroupSeq)
	runenv.RecordMessage(home)

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

	initgent := sync.NewTopic("init-gen", "")

	// Here we assign the first instance to be the orchestrator role
	//
	// Orchestrator is receiving all accounts by subscription, to then
	// execute the `add-genesis-account` command and send back to the rest
	// of the validators' set the initial genesis.json
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

		_, err = client.Publish(ctx, initgent, string(bt))
		if err != nil {
			return err
		}

		runenv.RecordMessage("Orchestrator has sent initial genesis with accounts")
	}

	if initCtx.GlobalSeq != 1 {
		ingench := make(chan string)
		_, err := client.Subscribe(ctx, initgent, ingench)
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

	gent := sync.NewTopic("genesis", "")

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

	runenv.RecordMessage("publishing app-validator address")
	ip, err := initCtx.NetClient.GetDataNetworkIP()
	if err != nil {
		return err
	}

	client.Publish(ctx, AppNodeTopic, &AppId{int(initCtx.GroupSeq), ip})

	runenv.RecordMessage("starting........")
	go appkit.StartNode(cmd, home)

	time.Sleep(30 * time.Second)
	return nil
}

// if runenv.TestGroupID == "app" {
// 	home := runenv.StringParam(fmt.Sprintf("app%d", initCtx.GroupSeq))

// 	fmt.Println(home)
// 	cmd := appkit.NewRootCmd()

// 	nodeId, err := appkit.GetNodeId(cmd, home)
// 	if err != nil {
// 		runenv.RecordCrash(err)
// 		return err
// 	}

// 	valt := sync.NewTopic("validator-info", &appkit.ValidatorNode{})
// 	client.Publish(ctx, valt, &appkit.ValidatorNode{nodeId, config.IPv4.IP})

// 	rdySt := sync.State("appReady")
// 	appseq := client.MustSignalEntry(ctx, rdySt)

// 	<-client.MustBarrier(ctx, rdySt, runenv.TestGroupInstanceCount).C

// 	valCh := make(chan *appkit.ValidatorNode)
// 	client.Subscribe(ctx, valt, valCh)

// 	var persPeers []string
// 	for i := 0; i < runenv.TestGroupInstanceCount; i++ {
// 		val := <-valCh
// 		runenv.RecordMessage("Validator Received: %s, %s", val.IP, val.PubKey)
// 		if !val.IP.Equal(config.IPv4.IP) {
// 			persPeers = append(persPeers, fmt.Sprintf("%s@%s", val.PubKey, val.IP.To4().String()))
// 		}
// 	}

// 	configPath := filepath.Join(home, "config", "config.toml")
// 	err = appkit.AddPersistentPeers(configPath, persPeers)
// 	if err != nil {
// 		return err
// 	}

// 	go appkit.StartNode(cmd, home)
// 	client.MustSignalAndWait(ctx, stateDone, int(initCtx.GlobalSeq))
// 	err = syscall.Kill(syscall.Getpid(), syscall.SIGINT)
// 	if err != nil {
// 		return err
// 	}

// 	return nil
