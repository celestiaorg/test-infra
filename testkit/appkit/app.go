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
	"github.com/celestiaorg/nmt/namespace"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	tmjson "github.com/tendermint/tendermint/libs/json"
	"github.com/tendermint/tendermint/pkg/consts"
	"github.com/tendermint/tendermint/rpc/coretypes"
	"github.com/tendermint/tendermint/rpc/jsonrpc/types"

	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
)

type ValidatorNode struct {
	PubKey string
	IP     net.IP
}

type AppKit struct {
	m   sync.Mutex
	Cmd *cobra.Command
}

func New() *AppKit {
	return &AppKit{
		Cmd: appcmd.NewRootCmd(),
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

func (ak *AppKit) InitChain(moniker string, chainId string, home string) (string, error) {
	return ak.execCmd([]string{"init", moniker, "--chain-id", chainId, "--home", home})
}

func (ak *AppKit) CreateKey(name string, krbackend string, home string) (string, error) {
	_, err := ak.execCmd([]string{"keys", "add", name, "--keyring-backend", krbackend, "--home", home, "--keyring-dir", home})
	if err != nil {
		return "", err
	}
	return ak.execCmd([]string{"keys", "show", name, "-a", "--keyring-backend", krbackend, "--home", home, "--keyring-dir", home})
}

func (ak *AppKit) AddGenAccount(addr string, amount string, home string) (string, error) {
	return ak.execCmd([]string{"add-genesis-account", addr, amount, "--home", home})
}

func (ak *AppKit) SignGenTx(accName string, amount string, krbackend string, chainId string, home string) (string, error) {
	return ak.execCmd([]string{"gentx", accName, amount, "--keyring-backend", krbackend, "--chain-id", chainId, "--home", home, "--keyring-dir", home})
}

func (ak *AppKit) CollectGenTxs(home string) (string, error) {
	return ak.execCmd([]string{"collect-gentxs", "--home", home})
}

func (ak *AppKit) GetNodeId(home string) (string, error) {
	return ak.execCmd([]string{"tendermint", "show-node-id", "--home", home})
}

func (ak *AppKit) StartNode(home, loglvl string) error {
	ak.Cmd.ResetFlags()

	ak.Cmd.SetErr(os.Stdout)
	ak.Cmd.SetArgs([]string{"start", "--home", home, "--log_level", loglvl, "--log_format", "plain"})

	return svrcmd.Execute(ak.Cmd, appcmd.EnvPrefix, app.DefaultNodeHome)
}

func (ak *AppKit) PayForData(accAdr string, msg int, krbackend, chainId, home string) error {
	ak.Cmd.ResetFlags()
	ak.Cmd.SetArgs([]string{
		"tx", "payment", "payForData", fmt.Sprint(msg),
		"--from", accAdr, "-b", "block", "-y", "--gas", "1000000000",
		"--fees", "100000000000utia", //15
		"--keyring-backend", krbackend, "--chain-id", chainId, "--home", home, "--keyring-dir", home,
	})

	return svrcmd.Execute(ak.Cmd, appcmd.EnvPrefix, app.DefaultNodeHome)
}

func getResultBlockResponse(uri string) (coretypes.ResultBlock, error) {
	resp, err := http.Get(uri)
	if err != nil {
		return coretypes.ResultBlock{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return coretypes.ResultBlock{}, err
	}

	var rpcResponse types.RPCResponse
	if err := rpcResponse.UnmarshalJSON(body); err != nil {
		return coretypes.ResultBlock{}, err
	}

	var resBlock coretypes.ResultBlock
	if err := tmjson.Unmarshal(rpcResponse.Result, &resBlock); err != nil {
		return coretypes.ResultBlock{}, err
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

func updateConfig(path, key, value string) error {
	fh, err := os.OpenFile(path, os.O_RDWR, 0777)
	if err != nil {
		return err
	}

	viper.SetConfigType("toml")
	err = viper.ReadConfig(fh)
	if err != nil {
		return err
	}

	fmt.Println("--------------------------------------")
	fmt.Println(viper.Get("mempool.max-txs-bytes"))
	fmt.Println(viper.Get("mempool.max-tx-bytes"))
	fmt.Println(viper.Get("mempool.size"))
	fmt.Println(viper.Get("consensus.timeout-commit"))
	fmt.Println("--------------------------------------")

	viper.Set(key, value)
	err = viper.WriteConfigAs(path)
	if err != nil {
		return err
	}

	return nil
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
	err := updateConfig(path, "consensus.timeout-commit", "30s")
	if err != nil {
		return err
	}
	return updateConfig(path, "p2p.persistent-peers", peersStr.String())
}

// ChangeNodeMode changes the mode type in config.toml of the app to either be:
// a) Full - Downloads the block but not produces any new ones
// b) Validator
// c) Seed - Only crawls the p2p network to find and share peers with each other
func ChangeNodeMode(path string, mode string) error {
	return updateConfig(path, "mode", mode)
}

func ChangeRPCServerAddress(path string, ip net.IP) error {
	return updateConfig(path, "rpc.laddr", fmt.Sprintf("tcp://%s:26657", ip.To4().String()))
}

func GetRandomNamespace() namespace.ID {
	for {
		s := tmrand.Bytes(8)
		if bytes.Compare(s, consts.MaxReservedNamespace) > 0 {
			return s
		}
	}
}

func GetRandomMessageBySize(size int) []byte {
	return tmrand.Bytes(size)
}
