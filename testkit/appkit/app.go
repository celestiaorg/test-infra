package appkit

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/p2p/pex"

	"github.com/celestiaorg/celestia-app/app"
	appcmd "github.com/celestiaorg/celestia-app/cmd/celestia-appd/cmd"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/keys"
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
	Home           string
	AccountAddress string
	AccountName    string
	ValopAddress   string
	ChainId        string
	Cmd            *cobra.Command
}

func wrapFlag(str string) string {
	return fmt.Sprintf("--%s", str)
}

func New(path, chainId string) *AppKit {
	return &AppKit{
		Home:    path,
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
	return ak.Home
}

func (ak *AppKit) InitChain(moniker string) (string, error) {
	return ak.execCmd(
		[]string{
			"init",
			moniker,
			wrapFlag(flags.FlagChainID),
			ak.ChainId,
			wrapFlag(flags.FlagHome),
			ak.Home,
		},
	)
}

func (ak *AppKit) CreateKey(name, krbackend, krpath string) (string, string, error) {
	_, err := ak.execCmd(
		[]string{
			"keys",
			"add",
			name,
			wrapFlag(flags.FlagKeyringBackend),
			krbackend,
			wrapFlag(flags.FlagHome),
			ak.Home,
			wrapFlag(flags.FlagKeyringDir),
			krpath,
		},
	)
	if err != nil {
		return "", "", err
	}
	accAddr, err := ak.execCmd(
		[]string{
			"keys",
			"show",
			name,
			wrapFlag(keys.FlagAddress),
			wrapFlag(flags.FlagKeyringBackend),
			krbackend,
			wrapFlag(flags.FlagHome),
			ak.Home,
			wrapFlag(flags.FlagKeyringDir),
			krpath,
		},
	)
	if err != nil {
		return "", "", err
	}

	valopAddr, err := ak.execCmd(
		[]string{
			"keys",
			"show",
			name,
			wrapFlag(keys.FlagAddress),
			wrapFlag(flags.FlagKeyringBackend),
			krbackend,
			wrapFlag(flags.FlagHome),
			ak.Home,
			wrapFlag(flags.FlagKeyringDir),
			krpath,
			wrapFlag(keys.FlagBechPrefix),
			"val",
		},
	)
	if err != nil {
		return "", "", err
	}
	return accAddr, valopAddr, nil
}

func (ak *AppKit) AddGenAccount(addr, amount string) (string, error) {
	return ak.execCmd(
		[]string{"add-genesis-account", addr, amount,
			wrapFlag(flags.FlagHome), ak.Home,
		},
	)
}

func (ak *AppKit) SignGenTx(accName, amount, krbackend, krpath string) (string, error) {
	args := []string{
		"gentx",
		accName,
		amount,
		wrapFlag(flags.FlagKeyringBackend),
		krbackend,
		wrapFlag(flags.FlagChainID),
		ak.ChainId,
		wrapFlag(flags.FlagHome),
		ak.Home,
		wrapFlag(flags.FlagKeyringDir),
		krpath,
	}

	ak.Cmd.ResetFlags()

	ak.m.Lock()

	ak.Cmd.SetArgs(args)
	if err := svrcmd.Execute(ak.Cmd, appcmd.EnvPrefix, app.DefaultNodeHome); err != nil {
		return "", err
	}

	ak.m.Unlock()

	return "", nil
}

func (ak *AppKit) CollectGenTxs() (string, error) {
	args := []string{"collect-gentxs", wrapFlag(flags.FlagHome), ak.Home}
	ak.Cmd.ResetFlags()

	ak.m.Lock()

	ak.Cmd.SetArgs(args)
	if err := svrcmd.Execute(ak.Cmd, appcmd.EnvPrefix, app.DefaultNodeHome); err != nil {
		return "", err
	}

	ak.m.Unlock()

	return "", nil
}

func (ak *AppKit) GetNodeId() (string, error) {
	return ak.execCmd(
		[]string{"tendermint", "show-node-id", wrapFlag(flags.FlagHome), ak.Home},
	)
}

func (ak *AppKit) StartNode(loglvl string) error {
	ak.Cmd.ResetFlags()

	// SetErr: send the error logs to stderr stream.
	ak.Cmd.SetErr(os.Stderr)
	ak.Cmd.SetArgs(
		[]string{
			"start",
			wrapFlag(flags.FlagHome),
			ak.Home,
			wrapFlag(flags.FlagLogLevel),
			loglvl,
			wrapFlag(flags.FlagLogFormat),
			"json",
		},
	)
	log, err := os.Create(filepath.Join("/var/log", "node.log"))
	if err != nil {
		return err
	}

	ak.Cmd.SetErr(log)

	return svrcmd.Execute(ak.Cmd, appcmd.EnvPrefix, app.DefaultNodeHome)
}

func (ak *AppKit) FundAccounts(accAdr, amount, krbackend, krpath string, accAddrs ...string) error {
	args := []string{"tx", "bank", "multi-send", accAdr}
	args = append(args, accAddrs...)
	args = append(args, amount,
		wrapFlag(flags.FlagBroadcastMode), flags.BroadcastBlock,
		wrapFlag(flags.FlagSkipConfirmation),
		wrapFlag(flags.FlagGas), "2000000",
		wrapFlag(flags.FlagFees), "100000utia",
		wrapFlag(flags.FlagKeyringBackend),
		krbackend,
		wrapFlag(flags.FlagChainID),
		ak.ChainId,
		wrapFlag(flags.FlagHome),
		ak.Home,
		wrapFlag(flags.FlagKeyringDir),
		krpath,
	)

	ak.Cmd.ResetFlags()
	ak.Cmd.SetArgs(args)

	return svrcmd.Execute(ak.Cmd, appcmd.EnvPrefix, app.DefaultNodeHome)
}

func (ak *AppKit) RegisterEVMAddress(valoperAddr, evmAddr, krbackend, krpath, from string) error {
	args := []string{"tx", "qgb", "register", valoperAddr, evmAddr}
	args = append(args,
		wrapFlag(flags.FlagBroadcastMode), flags.BroadcastBlock,
		wrapFlag(flags.FlagSkipConfirmation),
		wrapFlag(flags.FlagFees), "100000utia",
		wrapFlag(flags.FlagKeyringBackend), krbackend,
		wrapFlag(flags.FlagChainID), ak.ChainId,
		wrapFlag(flags.FlagHome), ak.Home,
		wrapFlag(flags.FlagKeyringDir), krpath,
		wrapFlag(flags.FlagFrom), from,
	)
	fmt.Println(args)
	ak.m.Lock()
	defer ak.m.Unlock()
	ak.Cmd.ResetFlags()
	ak.Cmd.SetArgs(args)

	return svrcmd.Execute(ak.Cmd, appcmd.EnvPrefix, app.DefaultNodeHome)
}

func (ak *AppKit) PayForBlob(accAdr string, msg int, krbackend, krpath string) error {
	ak.Cmd.ResetFlags()
	ak.Cmd.SetArgs([]string{
		"tx", "blob", "TestRandBlob", fmt.Sprint(msg),
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
		ak.Home,
		wrapFlag(flags.FlagKeyringDir),
		krpath,
	})

	return svrcmd.Execute(ak.Cmd, appcmd.EnvPrefix, app.DefaultNodeHome)
}

func GetGenesisState(uri string) (*coretypes.ResultGenesis, error) {
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

	var genState *coretypes.ResultGenesis
	if err := tmjson.Unmarshal(rpcResponse.Result, &genState); err != nil {
		return nil, err
	}

	return genState, nil
}

// GetResponse returns the response from the given uri of the app node
func GetResponse(uri string) (*coretypes.ResultBlock, error) {
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

	resBlock, err := GetResponse(uri)
	if err != nil {
		return "", err
	}

	return resBlock.BlockID.Hash.String(), nil
}

func GetLatestsBlockSize(ip net.IP) (int, error) {
	uri := fmt.Sprintf("http://%s:26657/block", ip.To4().String())

	resBlock, err := GetResponse(uri)
	if err != nil {
		return 0, err
	}

	return resBlock.Block.Size(), nil
}

func GetLatestBlockSizeAndHeight(ip net.IP) (int, uint64, error) {
	uri := fmt.Sprintf("http://%s:26657/block", ip.To4().String())

	resBlock, err := GetResponse(uri)
	if err != nil {
		return 0, 0, err
	}

	return resBlock.Block.Size(), uint64(resBlock.Block.Height), nil
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

func AddPeersToAddressBook(path string, peers []ValidatorNode) error {
	var filePath string = fmt.Sprintf("%s/config/addrbook.json", path)

	_, err := os.Create(filePath)
	if err != nil {
		return err
	}
	addrBook := pex.NewAddrBook(filePath, false)

	for _, peer := range peers {
		if peer.IP != nil {
			netAddr := p2p.NetAddress{
				ID:   p2p.ID(peer.PubKey),
				IP:   peer.IP,
				Port: 26656,
			}
			err = addrBook.AddAddress(&netAddr, &netAddr)
			if err != nil {
				return err
			}
		}
	}

	addrBook.Save()
	return nil
}

func ChangeRPCServerAddress(path string, ip net.IP) error {
	return updateConfig(path, "rpc.laddr", fmt.Sprintf("tcp://%s:26657", ip.To4().String()))
}

func ChangePruningStrategy(path string, strategy string) error {
	return updateConfig(path, "pruning", strategy)
}

func ChangeConfigParam(path, section, mode string, value interface{}) error {
	field := fmt.Sprintf("%s.%s", section, mode)
	return updateConfig(path, field, value)
}
