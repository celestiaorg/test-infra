package nodekit

import (
	"fmt"
	"net"

	"github.com/celestiaorg/celestia-node/node"
	"github.com/celestiaorg/celestia-node/params"
)

// option withTrustedHash should be passed!
func NewNode(path string, tp node.Type, IP net.IP, options ...node.Option) (*node.Node, error) {
	err := node.Init(path, tp)
	if err != nil {
		return nil, err
	}
	store, err := node.OpenStore(path)
	if err != nil {
		return nil, err
	}

	cfg := node.DefaultConfig(tp)

	cfg.P2P.ListenAddresses = []string{fmt.Sprintf("/ip4/%s/tcp/2121", IP)}

	options = append([]node.Option{node.WithConfig(cfg), node.WithNetwork(params.Private)}, options...)
	return node.New(tp, store, options...)
}
