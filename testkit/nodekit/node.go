package nodekit

import (
	"fmt"
	"net"

	"github.com/celestiaorg/celestia-node/logs"
	"github.com/celestiaorg/celestia-node/nodebuilder"
	"github.com/celestiaorg/celestia-node/nodebuilder/node"
	"github.com/celestiaorg/celestia-node/nodebuilder/p2p"
	logging "github.com/ipfs/go-log/v2"

	"github.com/celestiaorg/celestia-app/app"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/celestiaorg/celestia-app/app/encoding"
	"go.uber.org/fx"
	"os"
)

func NewConfig(
	tp node.Type,
	IP net.IP,
	trustedPeers []string,
	trustedHash string,
) *nodebuilder.Config {
	cfg := nodebuilder.DefaultConfig(tp)
	cfg.P2P.ListenAddresses = []string{
		fmt.Sprintf("/ip4/%s/udp/2121/quic-v1", IP),
		fmt.Sprintf("/ip4/%s/tcp/2121", IP),
	}
	cfg.Header.TrustedPeers = trustedPeers
	cfg.Header.TrustedHash = trustedHash

	return cfg
}

func NewNode(path string, tp node.Type, network string, cfg *nodebuilder.Config, options ...fx.Option) (*nodebuilder.Node, error) {
	err := nodebuilder.Init(*cfg, path, tp)
	if err != nil {
		return nil, err
	}
	encConf := encoding.MakeConfig(app.ModuleEncodingRegisters...)
	ring, err := keyring.New(app.Name, "test", "~/.store/keys", os.Stdin, encConf.Codec)
	if err != nil {
		return nil, err
	}
	store, err := nodebuilder.OpenStore(path, ring)
	if err != nil {
		return nil, err
	}
	return nodebuilder.NewWithConfig(tp, p2p.Network(network), store, cfg, options...)

}

func SetLoggersLevel(lvl string) error {
	level, err := logging.LevelFromString(lvl)
	if err != nil {
		return err
	}
	logs.SetAllLoggers(level)

	return nil
}