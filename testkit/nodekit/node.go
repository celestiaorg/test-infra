package nodekit

import (
	"fmt"
	"net"

	"github.com/celestiaorg/celestia-app/app"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/celestiaorg/celestia-node/logs"
	"github.com/celestiaorg/celestia-node/nodebuilder"
	"github.com/celestiaorg/celestia-node/nodebuilder/node"
	"github.com/celestiaorg/celestia-node/params"
	logging "github.com/ipfs/go-log/v2"

	"go.uber.org/fx"
)

func NewConfig(
	tp node.Type,
	IP net.IP,
	trustedPeers []string,
	trustedHash string,
) *nodebuilder.Config {
	cfg := nodebuilder.DefaultConfig(tp)
	cfg.P2P.ListenAddresses = []string{fmt.Sprintf("/ip4/%s/tcp/2121", IP)}
	cfg.Header.TrustedPeers = trustedPeers
	cfg.Header.TrustedHash = trustedHash

	return cfg
}

func NewNode(
	path string,
	tp node.Type,
	cfg *nodebuilder.Config,
	options ...fx.Option,
) (*nodebuilder.Node, error) {
	// This is necessary to ensure that the account addresses are correctly prefixed
	// as in the celestia application.
	sdkcfg := sdk.GetConfig()
	sdkcfg.SetBech32PrefixForAccount(app.Bech32PrefixAccAddr, app.Bech32PrefixAccPub)
	sdkcfg.SetBech32PrefixForValidator(app.Bech32PrefixValAddr, app.Bech32PrefixValPub)
	sdkcfg.Seal()

	err := nodebuilder.Init(*cfg, path, tp)
	if err != nil {
		return nil, err
	}
	store, err := nodebuilder.OpenStore(path)
	if err != nil {
		return nil, err
	}

	options = append([]fx.Option{nodebuilder.WithNetwork(params.Private)}, options...)
	return nodebuilder.New(tp, store, options...)
}

func SetLoggersLevel(lvl string) error {
	level, err := logging.LevelFromString(lvl)
	if err != nil {
		return err
	}
	logs.SetAllLoggers(level)

	return nil
}
