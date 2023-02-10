package common

import (
	"bytes"
	"context"
	"cosmossdk.io/math"
	"fmt"
	"github.com/celestiaorg/celestia-app/pkg/appconsts"
	"github.com/celestiaorg/celestia-node/header"
	"github.com/celestiaorg/celestia-node/nodebuilder"
	"github.com/celestiaorg/celestia-node/state"
	"github.com/celestiaorg/nmt/namespace"
	tmrand "github.com/tendermint/tendermint/libs/rand"
	"github.com/testground/sdk-go/runtime"
)

// DefaultNameId is used in cases where we only have 1 Namespace.ID used
// across all nodes that submit pfd and get shares by this ID
var DefaultNameId = namespace.ID{100, 100, 150, 150, 200, 200, 250, 255}

// TODO(@Bidon15): We need to start testing gas mechanism sooner than later
const gasLimit uint64 = 2000000

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

// GenerateNamespaceID returns a namespace ID based on runenv.StringParams defined in the composition file
// TODO(@Bidon15): We actually need to refactor this out using runenv.IntParam()
func GenerateNamespaceID(amount string) namespace.ID {
	if amount == "1" {
		return DefaultNameId
	}
	return GetRandomNamespace()
}

// GetRandomMessageBySize returns a random []byte per each call made by
// each instance of node type. The size is defined in the .toml file
func GetRandomMessageBySize(size int) []byte {
	return tmrand.Bytes(size)
}

// SubmitData calls a node.StateService SubmitPayForBlob() method with recording a txLog output.
func SubmitData(ctx context.Context, runenv *runtime.RunEnv, nd *nodebuilder.Node, nid namespace.ID, data []byte) error {
	fee := math.NewInt(30000)
	tx, err := nd.StateServ.SubmitPayForBlob(ctx, nid, data, fee, gasLimit)
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

// CheckSharesByNamespace accepts an expected namespace.ID and data that was submitted.
// Next, it verifies the data against the received shares of a block from a user-specified extended header
// by comparing length of each
func CheckSharesByNamespace(ctx context.Context, nd *nodebuilder.Node, nid namespace.ID, eh *header.ExtendedHeader, expectedData []byte) error {
	shares, err := nd.ShareServ.GetSharesByNamespace(ctx, eh.DAH, nid)
	if err != nil {
		return err
	}

	if len(shares) >= len(expectedData) {
		return nil
	}

	return fmt.Errorf("expected data is not in the shares slice")
}

// VerifyDataInNamespace encapsulates 3 steps to get the data verified against the next block's shares
// found in a user-specified namespace.ID
func VerifyDataInNamespace(ctx context.Context, nd *nodebuilder.Node, nid namespace.ID, data []byte) error {
	eh, err := nd.HeaderServ.Head(ctx)
	if err != nil {
		return err
	}

	waitH := eh.Height()
	waitH++
	eh, err = nd.HeaderServ.GetByHeight(ctx, uint64(waitH))
	if err != nil {
		return err
	}

	err = CheckSharesByNamespace(ctx, nd, nid, eh, data)
	if err != nil {
		return err
	}
	return nil
}

// CheckBalanceDeduction checks if the balance of a node has been deducted after a successful pfb
func CheckBalanceDeduction(ctx context.Context, nd *nodebuilder.Node, bal *state.Balance) error {
	latestbal, err := nd.StateServ.Balance(ctx)
	if err != nil {
		return err
	}

	if latestbal.IsGTE(*bal) {
		return fmt.Errorf("no balance deducted from the %s node", nd.Type.String())
	}
	return nil
}
