package appkit

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
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

func GetNodeId(cmd *cobra.Command, home string) (id string, err error) {
	scrapStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetArgs([]string{"tendermint", "show-node-id", "--home", home})
	if err := svrcmd.Execute(cmd, EnvPrefix, app.DefaultNodeHome); err != nil {
		return "", err
	}

	w.Close()
	outStr, err := ioutil.ReadAll(r)
	if err != nil {
		return "", err
	}
	os.Stdout = scrapStdout

	valPubKey := string(outStr)
	valPubKey = strings.ReplaceAll(valPubKey, "\n", "")
	return valPubKey, nil
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
