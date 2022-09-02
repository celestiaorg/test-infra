package appkit

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/celestiaorg/celestia-app/app"
	appcmd "github.com/celestiaorg/celestia-app/cmd/celestia-appd/cmd"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
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
	outStr, err := ioutil.ReadAll(r)
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
	return ak.execCmd([]string{"init", moniker, "--chain-id", ak.ChainId, "--home", ak.home})
}

func (ak *AppKit) CreateKey(name string, krbackend string, krpath string) (string, error) {
	_, err := ak.execCmd([]string{"keys", "add", name, "--keyring-backend", krbackend, "--home", ak.home, "--keyring-dir", krpath})
	if err != nil {
		return "", err
	}
	return ak.execCmd([]string{"keys", "show", name, "-a", "--keyring-backend", krbackend, "--home", ak.home, "--keyring-dir", krpath})
}

func (ak *AppKit) AddGenAccount(addr string, amount string) (string, error) {
	return ak.execCmd([]string{"add-genesis-account", addr, amount, "--home", ak.home})
}

func (ak *AppKit) SignGenTx(accName string, amount string, krbackend string, chainId string, krpath string) (string, error) {
	return ak.execCmd([]string{"gentx", accName, amount, "--keyring-backend", krbackend, "--chain-id", chainId, "--home", ak.home, "--keyring-dir", krpath})
}

func (ak *AppKit) CollectGenTxs() (string, error) {
	return ak.execCmd([]string{"collect-gentxs", "--home", ak.home})
}

func (ak *AppKit) GetNodeId() (string, error) {
	return ak.execCmd([]string{"tendermint", "show-node-id", "--home", ak.home})
}

func (ak *AppKit) StartNode(loglvl string) error {
	ak.Cmd.ResetFlags()

	ak.Cmd.SetErr(os.Stdout)
	ak.Cmd.SetArgs([]string{"start", "--home", ak.home, "--log_level", loglvl, "--log_format", "plain"})

	return svrcmd.Execute(ak.Cmd, appcmd.EnvPrefix, app.DefaultNodeHome)
}

func (ak *AppKit) PayForData(accAdr string, msg int, krbackend, krpath string) error {
	ak.Cmd.ResetFlags()
	ak.Cmd.SetArgs([]string{
		"tx", "payment", "payForData", fmt.Sprint(msg),
		"--from", accAdr, "-b", "block", "-y", "--gas", "1000000000",
		"--fees", "100000000000utia",
		"--keyring-backend", krbackend, "--chain-id", ak.ChainId, "--home", ak.home, "--keyring-dir", krpath,
	})

	return svrcmd.Execute(ak.Cmd, appcmd.EnvPrefix, app.DefaultNodeHome)
}

func getResultBlockResponse(uri string) (*coretypes.ResultBlock, error) {
	resp, err := http.Get(uri)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
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
