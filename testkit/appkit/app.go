package appkit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/celestiaorg/celestia-app/app"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
)

type ValidatorNode struct {
	PubKey string
	IP     net.IP
}

func execCmd(cmd *cobra.Command, args []string) (output string, err error) {
	cmd.ResetFlags()

	scrapStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	out := new(bytes.Buffer)
	cmd.Println(out)
	cmd.SetArgs(args)
	if err := svrcmd.Execute(cmd, EnvPrefix, app.DefaultNodeHome); err != nil {
		return "", err
	}

	w.Close()
	outStr, err := ioutil.ReadAll(r)
	if err != nil {
		return "", err
	}
	os.Stdout = scrapStdout

	output = string(outStr)
	output = strings.ReplaceAll(output, "\n", "")
	return output, nil
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

	viper.Set(key, value)
	err = viper.WriteConfigAs(path)
	if err != nil {
		return err
	}

	return nil
}

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

	return updateConfig(path, "p2p.persistent-peers", peersStr.String())
}

func ChangeNodeMode(path string, mode string) error {
	return updateConfig(path, "mode", mode)
}

func InitChain(cmd *cobra.Command, moniker string, chainId string, home string) (string, error) {
	return execCmd(cmd, []string{"init", moniker, "--chain-id", chainId, "--home", home})
}

func CreateKey(cmd *cobra.Command, name string, krbackend string, home string) (string, error) {
	_, err := execCmd(cmd, []string{"keys", "add", name, "--keyring-backend", krbackend, "--home", home, "--keyring-dir", home})
	if err != nil {
		return "", err
	}
	return execCmd(cmd, []string{"keys", "show", name, "-a", "--keyring-backend", krbackend, "--home", home, "--keyring-dir", home})
}

func AddGenAccount(cmd *cobra.Command, addr string, amount string, home string) (string, error) {
	return execCmd(cmd, []string{"add-genesis-account", addr, amount, "--home", home})
}

func SignGenTx(cmd *cobra.Command, accName string, amount string, krbackend string, chainId string, home string) (string, error) {
	return execCmd(cmd, []string{"gentx", accName, amount, "--keyring-backend", krbackend, "--chain-id", chainId, "--home", home, "--keyring-dir", home})
}

func CollectGenTxs(cmd *cobra.Command, home string) (string, error) {
	return execCmd(cmd, []string{"collect-gentxs", "--home", home})
}

func GetNodeId(cmd *cobra.Command, home string) (string, error) {
	return execCmd(cmd, []string{"tendermint", "show-node-id", "--home", home})
}

func StartNode(cmd *cobra.Command, home string) (string, error) {
	cmd.SetErr(os.Stdout)
	return execCmd(cmd, []string{"start", "--home", home, "--log_level", "info"})
}

func GetBlockHashByHeight(ip net.IP, height int) (string, error) {
	uri := fmt.Sprintf("http://%s:26657/block?height=%d", ip.To4().String(), height)
	resp, err := http.Get(uri)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var res BlockResp
	if err := json.Unmarshal(body, &res); err != nil {
		return "", err
	}
	return res.Result.BlockID.Hash, nil
}
