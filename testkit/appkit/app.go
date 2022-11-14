package appkit

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/celestiaorg/celestia-app/app"
	appcmd "github.com/celestiaorg/celestia-app/cmd/celestia-appd/cmd"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/keys"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
	"github.com/cosmos/cosmos-sdk/x/staking/teststaking"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	tmjson "github.com/tendermint/tendermint/libs/json"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
	"github.com/tendermint/tendermint/rpc/jsonrpc/types"
)

type ValidatorNode struct {
	PubKey string
	IP     net.IP
}

type AppKit struct {
	m              sync.Mutex
	home           string
	AccountAddress string
	ChainId        string
	Cmd            *cobra.Command
}

func wrapFlag(str string) string {
	return fmt.Sprintf("--%s", str)
}

func New(path, chainId string) *AppKit {
	return &AppKit{
		home:    path,
		ChainId: chainId,
		Cmd:     appcmd.NewRootCmd(),
	}
}

func (ak *AppKit) execCmd(args []string) (output string, err error) {
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

func (ak *AppKit) GetHomePath() string {
	return ak.home
}

func (ak *AppKit) InitChain(moniker string) (string, error) {
	return ak.execCmd(
		[]string{
			"init",
			moniker,
			wrapFlag(flags.FlagChainID),
			ak.ChainId,
			wrapFlag(flags.FlagHome),
			ak.home,
		},
	)
}

func (ak *AppKit) CreateKey(name, krbackend, krpath string) (string, error) {
	_, err := ak.execCmd(
		[]string{
			"keys",
			"add",
			name,
			wrapFlag(flags.FlagKeyringBackend),
			krbackend,
			wrapFlag(flags.FlagHome),
			ak.home,
			wrapFlag(flags.FlagKeyringDir),
			krpath,
		},
	)
	if err != nil {
		return "", err
	}
	return ak.execCmd(
		[]string{
			"keys",
			"show",
			name,
			wrapFlag(keys.FlagAddress),
			wrapFlag(flags.FlagKeyringBackend),
			krbackend,
			wrapFlag(flags.FlagHome),
			ak.home,
			wrapFlag(flags.FlagKeyringDir),
			krpath,
		},
	)
}

func (ak *AppKit) AddGenAccount(addr, amount string) (string, error) {
	return ak.execCmd(
		[]string{"add-genesis-account", addr, amount,
			wrapFlag(flags.FlagHome), ak.home,
		},
	)
}

func (ak *AppKit) SignGenTx(accName, amount, krbackend, krpath string) (string, error) {
	ethAddress, err := teststaking.RandomEVMAddress()
	if err != nil {
		return "", err
	}

	return ak.execCmd(
		[]string{
			"gentx",
			accName,
			amount,
			wrapFlag(flags.FlagOrchestratorAddress),
			ak.AccountAddress,
			wrapFlag(flags.FlagEVMAddress),
			ethAddress.String(),
			wrapFlag(flags.FlagKeyringBackend),
			krbackend,
			wrapFlag(flags.FlagChainID),
			ak.ChainId,
			wrapFlag(flags.FlagHome),
			ak.home,
			wrapFlag(flags.FlagKeyringDir),
			krpath,
		},
	)
}

func (ak *AppKit) CollectGenTxs() (string, error) {
	return ak.execCmd(
		[]string{"collect-gentxs", wrapFlag(flags.FlagHome), ak.home},
	)
}

func (ak *AppKit) GetNodeId() (string, error) {
	return ak.execCmd(
		[]string{"tendermint", "show-node-id", wrapFlag(flags.FlagHome), ak.home},
	)
}

func (ak *AppKit) StartNode(loglvl string) error {
	ak.Cmd.ResetFlags()

	ak.Cmd.SetErr(os.Stdout)
	ak.Cmd.SetArgs(
		[]string{
			"start",
			wrapFlag(flags.FlagHome),
			ak.home,
			wrapFlag(flags.FlagLogLevel),
			loglvl,
			wrapFlag(flags.FlagLogFormat),
			"plain",
		},
	)

	return svrcmd.Execute(ak.Cmd, appcmd.EnvPrefix, app.DefaultNodeHome)
}

func (ak *AppKit) FundAccounts(accAdr, amount, krbackend, krpath string, accAddrs ...string) error {
	args := []string{"tx", "bank", "multi-send", accAdr}
	args = append(args, accAddrs...)
	args = append(args, amount,
		wrapFlag(flags.FlagBroadcastMode), flags.BroadcastBlock,
		wrapFlag(flags.FlagSkipConfirmation),
		wrapFlag(flags.FlagGas), "1000000",
		wrapFlag(flags.FlagFees), "100000utia",
		wrapFlag(flags.FlagKeyringBackend),
		krbackend,
		wrapFlag(flags.FlagChainID),
		ak.ChainId,
		wrapFlag(flags.FlagHome),
		ak.home,
		wrapFlag(flags.FlagKeyringDir),
		krpath,
	)

	ak.Cmd.ResetFlags()
	ak.Cmd.SetArgs(args)

	return svrcmd.Execute(ak.Cmd, appcmd.EnvPrefix, app.DefaultNodeHome)
}

func (ak *AppKit) PayForData(accAdr string, msg int, krbackend, krpath string) error {
	ak.Cmd.ResetFlags()
	ak.Cmd.SetArgs([]string{
		"tx", "payment", "payForData", fmt.Sprint(msg),
		wrapFlag(flags.FlagFrom), accAdr,
		wrapFlag(flags.FlagBroadcastMode), flags.BroadcastBlock,
		wrapFlag(flags.FlagSkipConfirmation),
		wrapFlag(flags.FlagGas), "1000000000",
		wrapFlag(flags.FlagFees), "100000000000utia",
		wrapFlag(flags.FlagKeyringBackend),
		krbackend,
		wrapFlag(flags.FlagChainID),
		ak.ChainId,
		wrapFlag(flags.FlagHome),
		ak.home,
		wrapFlag(flags.FlagKeyringDir),
		krpath,
	})

	return svrcmd.Execute(ak.Cmd, appcmd.EnvPrefix, app.DefaultNodeHome)
}

func getResultBlockResponse(uri string) (*coretypes.ResultBlock, error) {
	resp, err := http.Get(uri)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var rpcResponse types.RPCResponse
	if err := rpcResponse.UnmarshalJSON(body); err != nil {
		return nil, err
	}

	var resBlock *coretypes.ResultBlock
	if err := tmjson.Unmarshal(rpcResponse.Result, &resBlock); err != nil {
		return nil, err
	}

	return resBlock, nil
}

func GetBlockHashByHeight(ip net.IP, height int) (string, error) {
	uri := fmt.Sprintf("http://%s:26657/block?height=%d", ip.To4().String(), height)

	resBlock, err := getResultBlockResponse(uri)
	if err != nil {
		return "", err
	}

	return resBlock.BlockID.Hash.String(), nil
}

func GetLatestsBlockSize(ip net.IP) (int, error) {
	uri := fmt.Sprintf("http://%s:26657/block", ip.To4().String())

	resBlock, err := getResultBlockResponse(uri)
	if err != nil {
		return 0, err
	}

	return resBlock.Block.Size(), nil
}

func updateConfig(path, key string, value interface{}) error {
	fh, err := os.OpenFile(path, os.O_RDWR, 0777)
	if err != nil {
		return err
	}

	viper.SetConfigType("toml")
	err = viper.ReadConfig(fh)
	if err != nil {
		return err
	}

	viper.Set(key, value)
	err = viper.WriteConfigAs(path)
	if err != nil {
		return err
	}

	return nil
}

func AddSeedPeers(path string, peers []string) error {
	var (
		peersStr  bytes.Buffer
		port      int    = 26656
		separator string = ","
	)

	for k, peer := range peers {
		if k == (len(peers) - 1) {
			separator = ""
		}
		peersStr.WriteString(fmt.Sprintf("%s:%d%s", peer, port, separator))
	}
	return updateConfig(path, "p2p.seeds", peersStr.String())
}

// AddPersistentPeers modifies the respective field in the config.toml
// to allow the peer to always connect to a set of defined peers
func AddPersistentPeers(path string, peers []string) error {
	var peersStr bytes.Buffer
	var port int = 26656
	var separator string = ","
	for k, peer := range peers {
		if k == (len(peers) - 1) {
			separator = ""
		}
		peersStr.WriteString(fmt.Sprintf("%s:%d%s", peer, port, separator))
	}
	return updateConfig(path, "p2p.persistent_peers", peersStr.String())
}

func ChangeRPCServerAddress(path string, ip net.IP) error {
	return updateConfig(path, "rpc.laddr", fmt.Sprintf("tcp://%s:26657", ip.To4().String()))
}

func ChangeConfigParam(path, section, mode string, value interface{}) error {
	field := fmt.Sprintf("%s.%s", section, mode)
	return updateConfig(path, field, value)
}
