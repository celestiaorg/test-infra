package common

import (
	"bytes"
	"github.com/celestiaorg/celestia-app/pkg/appconsts"
	"github.com/celestiaorg/nmt/namespace"
	tmrand "github.com/tendermint/tendermint/libs/rand"
)

// DefaultNameId is used in cases where we only have 1 Namespace.ID used
// across all nodes that submit pfd and get shares by this ID
var DefaultNameId = namespace.ID{100, 100, 150, 150, 200, 200, 250, 255}

// GetRandomNamespace returns a random namespace.ID per each call made by
// each instance of node type
func GetRandomNamespace() namespace.ID {
	for {
		s := tmrand.Bytes(8)
		if bytes.Compare(s, appconsts.MaxReservedNamespace) > 0 {
			return s
		}
	}
}

// GetRandomMessageBySize returns a random []byte per each call made by
// each instance of node type. The size is defined in the .toml file
func GetRandomMessageBySize(size int) []byte {
	return tmrand.Bytes(size)
}
