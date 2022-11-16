package common

import (
	"bytes"
	"context"
	"fmt"
	"github.com/celestiaorg/celestia-app/pkg/appconsts"
	"github.com/celestiaorg/celestia-node/nodebuilder"
	"github.com/celestiaorg/nmt/namespace"
	tmrand "github.com/tendermint/tendermint/libs/rand"
	"github.com/testground/sdk-go/runtime"
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

func SubmitData(ctx context.Context, runenv *runtime.RunEnv, nd *nodebuilder.Node, nid namespace.ID, data []byte) error {
	tx, err := nd.StateServ.SubmitPayForData(ctx, nid, data, 2000000)
	if err != nil {
		return err
	}

	runenv.RecordMessage("code response is %d", tx.Code)
	runenv.RecordMessage(tx.RawLog)
	if tx.Code != 0 {
		return fmt.Errorf("failed pfd")
	}
	return nil
}

func CheckSharesByNamespace(ctx context.Context, nd *nodebuilder.Node, nid namespace.ID, data []byte) error {
	eh, err := nd.HeaderServ.Head(ctx)
	if err != nil {
		return err
	}

	shares, err := nd.ShareServ.GetSharesByNamespace(ctx, eh.DAH, nid)
	if err != nil {
		return err
	}
	fmt.Println(data)
	for _, share := range shares {
		fmt.Println(share)
		if bytes.Equal(share, data) {
			return nil
		}
	}
	return fmt.Errorf("expected data is not equal to actual one")
}
