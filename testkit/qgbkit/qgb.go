package qgbkit

import (
	"bytes"
	"crypto/ecdsa"
	"fmt"
	appcmd "github.com/celestiaorg/celestia-app/cmd/celestia-appd/cmd"
	"github.com/celestiaorg/orchestrator-relayer/cmd/qgb/relayer"
	"github.com/libp2p/go-libp2p/core/crypto"
	"io"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/celestiaorg/celestia-app/app"
	qgbbase "github.com/celestiaorg/orchestrator-relayer/cmd/qgb/base"
	qgbdeploy "github.com/celestiaorg/orchestrator-relayer/cmd/qgb/deploy"
	qgborch "github.com/celestiaorg/orchestrator-relayer/cmd/qgb/orchestrator"
	qgbcmd "github.com/celestiaorg/orchestrator-relayer/cmd/qgb/root"
	"github.com/cosmos/cosmos-sdk/client/flags"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
	"github.com/spf13/cobra"
)

type BootstrapperNode struct {
	P2PID string
	IP    net.IP
}

type QGBKit struct {
	m             sync.Mutex
	Home          string
	Cmd           *cobra.Command
	P2PPrivateKey *crypto.PrivKey
	EVMPrivateKey *ecdsa.PrivateKey
}

func wrapFlag(str string) string {
	return fmt.Sprintf("--%s", str)
}

// New creates a new QGBKit for testground.
// Note: the provided private keys do not get added automatically to the store. Make sure to
// add them using the below import helpers before using them.
func New(qgbPath string, p2pPrivateKey *crypto.PrivKey, evmPrivateKey *ecdsa.PrivateKey) *QGBKit {
	return &QGBKit{
		Home:          qgbPath,
		Cmd:           qgbcmd.Cmd(),
		P2PPrivateKey: p2pPrivateKey,
		EVMPrivateKey: evmPrivateKey,
	}
}

func (ak *QGBKit) execCmd(args []string) (output string, err error) {
	ak.Cmd.ResetFlags()

	ak.m.Lock()
	scrapStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	out := new(bytes.Buffer)
	ak.Cmd.Println(out)
	ak.Cmd.SetArgs(args)
	if err := svrcmd.Execute(ak.Cmd, appcmd.EnvPrefix, app.DefaultNodeHome); err != nil {
		return "", err
	}

	w.Close()
	outStr, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	os.Stdout = scrapStdout
	ak.m.Unlock()

	output = string(outStr)
	output = strings.ReplaceAll(output, "\n", "")
	return output, nil
}

func (ak *QGBKit) GetHomePath() string {
	return ak.Home
}

// InitService initializes the storage of a service.
// A service is either an orchestrator, relayer or deployer.
func (ak *QGBKit) InitService(service string) (string, error) {
	return ak.execCmd(
		[]string{
			service,
			"init",
			wrapFlag(flags.FlagHome),
			ak.Home,
		},
	)
}

// ImportEVMKey Imports the specified private key to service
// keystore. A service is either orchestrator, relayer or deployer.
func (ak *QGBKit) ImportEVMKey(service, evmPrivateKey, passphrase string) (string, error) {
	return ak.execCmd(
		[]string{
			service,
			"keys",
			"evm",
			"import",
			"ecdsa",
			evmPrivateKey,
			wrapFlag(qgbbase.FlagEVMPassphrase),
			passphrase,
			wrapFlag(flags.FlagHome),
			ak.Home,
		},
	)
}

// ListEVMKeys Lists the EVM keys in store.
func (ak *QGBKit) ListEVMKeys(service string) (string, error) {
	return ak.execCmd(
		[]string{
			service,
			"keys",
			"evm",
			"list",
			wrapFlag(flags.FlagHome),
			ak.Home,
		},
	)
}

// ImportP2PKey imports a P2P private key to the service keystore.
// The nickname is the name given to the private key, and the service is the
// target service: orchestrator, relayer or deployer.
func (ak *QGBKit) ImportP2PKey(service, p2pPrivateKey, nickname string) (string, error) {
	return ak.execCmd(
		[]string{
			service,
			"keys",
			"p2p",
			"import",
			nickname,
			p2pPrivateKey,
			wrapFlag(flags.FlagHome),
			ak.Home,
		},
	)
}

// StartOrchestrator starts the orchestrator
// Set the p2p nickname or the bootstrappers to an empty string not to pass them to the
// start command.
func (ak *QGBKit) StartOrchestrator(evmAddress, evmPassphrase, p2pNickname, bootstrappers string) error {
	ak.Cmd.ResetFlags()

	args := []string{
		"orchestrator",
		"start",
		wrapFlag(flags.FlagHome),
		ak.Home,
		wrapFlag(qgborch.FlagEVMAccAddress),
		evmAddress,
		wrapFlag(qgbbase.FlagEVMPassphrase),
		evmPassphrase,
	}

	if p2pNickname != "" {
		args = append(args, wrapFlag(qgbbase.FlagP2PNickname), p2pNickname)
	}
	if bootstrappers != "" {
		args = append(args, wrapFlag(qgbbase.FlagBootstrappers), bootstrappers)
	}

	ak.Cmd.SetArgs(args)

	log, err := os.Create(filepath.Join("/var/log", "orch.log"))
	if err != nil {
		return err
	}

	ak.Cmd.SetErr(log)
	ak.Cmd.SetOut(log)

	return svrcmd.Execute(ak.Cmd, appcmd.EnvPrefix, app.DefaultNodeHome)
}

// StartRelayer starts the relayer
// Set the p2p nickname to an empty string not to pass them to the
// start command.
func (ak *QGBKit) StartRelayer(evmAddress, evmPassphrase, evmChainID, evmRPC, contractAddr, p2pNickname, bootstrappers string) error {
	ak.Cmd.ResetFlags()

	args := []string{
		"relayer",
		"start",
		wrapFlag(flags.FlagHome),
		ak.Home,
		wrapFlag(relayer.FlagEVMAccAddress),
		evmAddress,
		wrapFlag(qgbbase.FlagEVMPassphrase),
		evmPassphrase,
		wrapFlag(qgbbase.FlagBootstrappers),
		bootstrappers,
		wrapFlag(relayer.FlagEVMChainID),
		evmChainID,
		wrapFlag(relayer.FlagEVMRPC),
		evmRPC,
		wrapFlag(relayer.FlagContractAddress),
		contractAddr,
	}

	if p2pNickname != "" {
		args = append(args, wrapFlag(qgbbase.FlagP2PNickname), p2pNickname)
	}
	ak.Cmd.SetArgs(args)

	log, err := os.Create(filepath.Join("/var/log", "rel.log"))
	if err != nil {
		return err
	}

	ak.Cmd.SetErr(log)

	return svrcmd.Execute(ak.Cmd, appcmd.EnvPrefix, app.DefaultNodeHome)
}

// DeployContract deploys the QGB contract and returns its address.
func (ak *QGBKit) DeployContract(evmAddress, evmPassphrase, evmChainID, evmRPC string) (string, error) {
	ak.Cmd.ResetFlags()
	fmt.Println("deploying contract")

	out, err := ak.execCmd(
		[]string{
			"deploy",
			wrapFlag(flags.FlagHome),
			ak.Home,
			wrapFlag(qgbdeploy.FlagEVMAccAddress),
			evmAddress,
			wrapFlag(qgbbase.FlagEVMPassphrase),
			evmPassphrase,
			wrapFlag(qgbdeploy.FlagEVMChainID),
			evmChainID,
			wrapFlag(qgbdeploy.FlagEVMRPC),
			evmRPC,
			wrapFlag(qgbdeploy.FlagStartingNonce),
			"latest",
		},
	)
	if err != nil {
		return "", err
	}

	fmt.Println(out)
	addr, err := parseContractAddress(out)
	if err != nil {
		return "", err
	}
	fmt.Printf("contract address: %s\n", addr)
	return addr, nil
}

func parseContractAddress(log string) (string, error) {
	lines := strings.Split(log, "[")
	for _, line := range lines {
		match := regexp.MustCompile("deployed QGB contract").MatchString(line)
		if match {
			return regexp.MustCompile("0x[a-fA-F0-9]{40}").FindString(line), nil
		}
	}
	return "", fmt.Errorf("deployed QGB contract address not found in provided log")
}
