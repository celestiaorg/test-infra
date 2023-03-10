package nodekit

import (
	"context"
	"fmt"
	"github.com/celestiaorg/celestia-app/app"
	"github.com/celestiaorg/celestia-app/app/encoding"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"net"
	"os"
	"path/filepath"

	"github.com/celestiaorg/celestia-node/logs"
	"github.com/celestiaorg/celestia-node/nodebuilder"
	"github.com/celestiaorg/celestia-node/nodebuilder/node"
	"github.com/celestiaorg/celestia-node/nodebuilder/p2p"
	logging "github.com/ipfs/go-log/v2"

	"github.com/celestiaorg/celestia-app/app"
	"github.com/celestiaorg/celestia-app/app/encoding"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
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

	keysPath := filepath.Join(path, "keys")
	encConf := encoding.MakeConfig(app.ModuleEncodingRegisters...)
	ring, err := keyring.New(app.Name, cfg.State.KeyringBackend, keysPath, os.Stdin, encConf.Codec)
	if err != nil {
		return nil, err
	}

	store, err := nodebuilder.OpenStore(path, ring)
	if err != nil {
		return nil, err
	}
	return nodebuilder.NewWithConfig(tp, p2p.Network(network), store, cfg, options...)
}

func IsSyncing(ctx context.Context, nd *nodebuilder.Node) bool {
	syncer, err := nd.HeaderServ.SyncState(ctx)
	if err != nil {
		return false
	}
	return !syncer.Finished()
}

func SetLoggersLevel(lvl string) error {
	level, err := logging.LevelFromString(lvl)
	if err != nil {
		return err
	}
	logs.SetAllLoggers(level)
	logging.SetAllLoggers(level)
	_ = logging.SetLogLevel("engine", "FATAL")
	_ = logging.SetLogLevel("blockservice", "WARN")
	_ = logging.SetLogLevel("bs:sess", "WARN")
	_ = logging.SetLogLevel("addrutil", "INFO")
	_ = logging.SetLogLevel("dht", "ERROR")
	_ = logging.SetLogLevel("swarm2", "WARN")
	_ = logging.SetLogLevel("bitswap", "WARN")
	_ = logging.SetLogLevel("connmgr", "WARN")
	_ = logging.SetLogLevel("nat", "INFO")
	_ = logging.SetLogLevel("dht/RtRefreshManager", "FATAL")
	_ = logging.SetLogLevel("bitswap_network", "ERROR")
	_ = logging.SetLogLevel("badger", "INFO")
	_ = logging.SetLogLevel("basichost", "INFO")
	_ = logging.SetLogLevel("bitswap-client", "INFO")
	_ = logging.SetLogLevel("share/light", "INFO")

	return nil
}

// func SetModuleLoggerLevel(module, lvl string)
