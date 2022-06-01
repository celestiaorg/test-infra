package nodekit

import (
	"github.com/celestiaorg/celestia-node/node"
	"github.com/celestiaorg/celestia-node/params"
)

// option withTrustedHash should be passed!
func NewNode(path string, tp node.Type, options ...node.Option) (*node.Node, error) {
	err := node.Init(path, tp)
	if err != nil {
		return nil, err
	}
	store, err := node.OpenStore(path)
	if err != nil {
		return nil, err
	}

	options = append(options,
		node.WithNetwork(params.Private),
	)

	nd, err := node.New(tp, store, options...)
	if err != nil {
		return nil, err
	}

	return nd, nil
}
