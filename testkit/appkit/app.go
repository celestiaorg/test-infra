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

	"github.com/cosmos/cosmos-sdk/client/flags"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
)

type ValidatorNode struct {
	PubKey string
	IP     net.IP
}

func execCmd(cmd *cobra.Command, args []string) (output string, err error) {
	fmt.Println(args)
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

func GetNodeId(cmd *cobra.Command, home string) (id string, err error) {
	return execCmd(cmd, []string{"tendermint", "show-node-id", "--home", home})
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

	fh, err := os.OpenFile(path, os.O_RDWR, 0777)
	if err != nil {
		return err
	}

	viper.SetConfigType("toml")
	err = viper.ReadConfig(fh)
	if err != nil {
		return err
	}

	viper.Set("p2p.persistent-peers", peersStr.String())
	err = viper.WriteConfigAs(path)
	if err != nil {
		return err
	}

	return nil
}

func InitChain(cmd *cobra.Command, moniker string, chainId string, home string) (output string, err error) {
	return execCmd(cmd, []string{"init", moniker, "--chain-id", chainId, "--home", home})
}

func CreateKey(cmd *cobra.Command, name string, krbackend string, home string) (output string, err error) {
	_, err = execCmd(cmd, []string{"keys", "add", name, "--keyring-backend", krbackend, "--home", home, "--keyring-dir", home})
	if err != nil {
		return "", err
	}
	return execCmd(cmd, []string{"keys", "show", name, "-a", "--keyring-backend", krbackend, "--home", home, "--keyring-dir", home})
}

// celestia-appd add-genesis-account celestia1mld039ypx3wu82h9wua4vjygze7es3s6rl9xfl 1000000000000000utia --home ~/.celestia-app-1
func AddGenAccount(cmd *cobra.Command, addr string, amount string, home string) (output string, err error) {
	return execCmd(cmd, []string{"add-genesis-account", addr, amount, "--home", home})
}

func StartNode(cmd *cobra.Command, home string) error {
	cmd.ResetFlags()
	cmd.Flags().Set(flags.FlagHome, "")

	cmd.SetErr(os.Stdout)
	cmd.SetArgs([]string{"start", "--home", home, "--log_level", "info"})

	if err := svrcmd.Execute(cmd, EnvPrefix, app.DefaultNodeHome); err != nil {
		return err
	}

	return nil
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
